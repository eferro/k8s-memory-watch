package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/eduardoferro/mgmt-monitoring/internal/config"
	"github.com/eduardoferro/mgmt-monitoring/internal/k8s"
)

// MemoryMonitor orchestrates memory monitoring operations
type MemoryMonitor struct {
	k8sClient *k8s.Client
	config    *config.Config
}

// New creates a new memory monitor
func New(cfg *config.Config) (*MemoryMonitor, error) {
	// Create Kubernetes client
	client, err := k8s.NewClient(cfg.KubeConfig, cfg.InCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &MemoryMonitor{
		k8sClient: client,
		config:    cfg,
	}, nil
}

// HealthCheck verifies the monitor can connect to Kubernetes
func (m *MemoryMonitor) HealthCheck(ctx context.Context) error {
	slog.Info("Performing health check...")

	err := m.k8sClient.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}

	slog.Info("Health check passed - Kubernetes cluster is accessible")
	return nil
}

// CollectMemoryInfo collects memory information from all pods
func (m *MemoryMonitor) CollectMemoryInfo(ctx context.Context) (*MemoryReport, error) {
	slog.Info("Starting memory information collection...")

	pods, summary, err := m.k8sClient.GetAllPodsMemoryInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory info: %w", err)
	}

	// Sort pods by namespace and name for consistent output
	sort.Slice(pods, func(i, j int) bool {
		if pods[i].Namespace != pods[j].Namespace {
			return pods[i].Namespace < pods[j].Namespace
		}
		return pods[i].PodName < pods[j].PodName
	})

	report := &MemoryReport{
		Summary: *summary,
		Pods:    pods,
	}

	slog.Info("Memory collection completed successfully",
		"total_pods", summary.TotalPods,
		"running_pods", summary.RunningPods,
		"namespaces", summary.NamespaceCount)

	return report, nil
}

// AnalyzeMemoryUsage performs analysis on memory usage and identifies potential issues
func (m *MemoryMonitor) AnalyzeMemoryUsage(ctx context.Context) (*AnalysisResult, error) {
	report, err := m.CollectMemoryInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect memory info for analysis: %w", err)
	}

	analysis := &AnalysisResult{
		Report:        *report,
		HighUsagePods: []k8s.PodMemoryInfo{},
		WarningPods:   []k8s.PodMemoryInfo{},
		ProblemsFound: []string{},
	}

	// Analyze each pod
	for _, pod := range report.Pods {
		// Skip pods without current usage data
		if pod.CurrentUsage == nil {
			continue
		}

		// Calculate percentages
		pod.CalculateUsagePercent()

		// Check for high usage against requests
		if pod.UsagePercent != nil && *pod.UsagePercent >= m.config.MemoryWarningPercent {
			analysis.WarningPods = append(analysis.WarningPods, pod)

			if *pod.UsagePercent >= 95.0 {
				analysis.HighUsagePods = append(analysis.HighUsagePods, pod)
				analysis.ProblemsFound = append(analysis.ProblemsFound,
					fmt.Sprintf("Pod %s/%s is using %.1f%% of its memory request",
						pod.Namespace, pod.PodName, *pod.UsagePercent))
			}
		}

		// Check for high usage against limits
		if pod.LimitUsagePercent != nil && *pod.LimitUsagePercent >= 90.0 {
			analysis.HighUsagePods = append(analysis.HighUsagePods, pod)
			analysis.ProblemsFound = append(analysis.ProblemsFound,
				fmt.Sprintf("Pod %s/%s is using %.1f%% of its memory limit",
					pod.Namespace, pod.PodName, *pod.LimitUsagePercent))
		}

		// Check for pods without memory limits
		if pod.MemoryLimit == nil {
			analysis.ProblemsFound = append(analysis.ProblemsFound,
				fmt.Sprintf("Pod %s/%s has no memory limit defined",
					pod.Namespace, pod.PodName))
		}

		// Check for pods without memory requests
		if pod.MemoryRequest == nil {
			analysis.ProblemsFound = append(analysis.ProblemsFound,
				fmt.Sprintf("Pod %s/%s has no memory request defined",
					pod.Namespace, pod.PodName))
		}
	}

	slog.Info("Memory analysis completed",
		"warning_pods", len(analysis.WarningPods),
		"high_usage_pods", len(analysis.HighUsagePods),
		"problems_found", len(analysis.ProblemsFound))

	return analysis, nil
}
