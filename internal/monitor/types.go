package monitor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
)

// MemoryReport contains the complete memory report for the cluster
type MemoryReport struct {
	Summary k8s.MemorySummary   `json:"summary"`
	Pods    []k8s.PodMemoryInfo `json:"pods"`
}

// AnalysisResult contains the analysis of memory usage patterns and issues
type AnalysisResult struct {
	Report        MemoryReport        `json:"report"`
	HighUsagePods []k8s.PodMemoryInfo `json:"high_usage_pods"`
	WarningPods   []k8s.PodMemoryInfo `json:"warning_pods"`
	ProblemsFound []string            `json:"problems_found"`
}

// PrintSummary prints a human-readable summary of the memory report
func (r *MemoryReport) PrintSummary() {
	fmt.Printf("\n")
	fmt.Printf("=== Kubernetes Memory Report ===\n")
	fmt.Printf("Generated at: %s\n", r.Summary.Timestamp.Format(time.RFC3339))
	fmt.Printf("\n")

	fmt.Printf("Cluster Overview:\n")
	fmt.Printf("  Namespaces: %d\n", r.Summary.NamespaceCount)
	fmt.Printf("  Total Pods: %d\n", r.Summary.TotalPods)
	fmt.Printf("  Running Pods: %d\n", r.Summary.RunningPods)
	fmt.Printf("  Pods with Metrics: %d\n", r.Summary.PodsWithMetrics)
	fmt.Printf("  Pods with Limits: %d\n", r.Summary.PodsWithLimits)
	fmt.Printf("  Pods with Requests: %d\n", r.Summary.PodsWithRequests)
	fmt.Printf("\n")

	fmt.Printf("Memory Totals:\n")
	fmt.Printf("  Total Usage: %s\n", k8s.FormatMemory(&r.Summary.TotalMemoryUsage))
	fmt.Printf("  Total Requests: %s\n", k8s.FormatMemory(&r.Summary.TotalMemoryRequest))
	fmt.Printf("  Total Limits: %s\n", k8s.FormatMemory(&r.Summary.TotalMemoryLimit))
	fmt.Printf("\n")
}

// PrintDetailedReport prints detailed pod-by-pod memory information
func (r *MemoryReport) PrintDetailedReport(cfg *config.Config) {
	r.PrintSummary()

	if len(r.Pods) == 0 {
		fmt.Printf("No pods found.\n")
		return
	}

	fmt.Printf("=== Detailed Pod Memory Information ===\n")

	currentNamespace := ""
	for _, pod := range r.Pods {
		if pod.Namespace != currentNamespace {
			currentNamespace = pod.Namespace
			fmt.Printf("\nNamespace: %s\n", currentNamespace)
			fmt.Printf("%s\n", strings.Repeat("-", 80))
		}

		fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
	}
	fmt.Printf("\n")
}

// PrintAnalysis prints the analysis results with warnings and recommendations
func (a *AnalysisResult) PrintAnalysis(cfg *config.Config) {
	fmt.Printf("\n")
	fmt.Printf("=== Memory Usage Analysis ===\n")

	if len(a.ProblemsFound) == 0 {
		fmt.Printf("✅ No memory issues detected.\n")
	} else {
		fmt.Printf("🚨 Found %d potential issues:\n\n", len(a.ProblemsFound))

		for i, problem := range a.ProblemsFound {
			fmt.Printf("%d. %s\n", i+1, problem)
		}
	}

	if len(a.HighUsagePods) > 0 {
		fmt.Printf("\n🔥 High Memory Usage Pods (%d):\n", len(a.HighUsagePods))
		for _, pod := range a.HighUsagePods {
			fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
		}
	}

	if len(a.WarningPods) > 0 {
		fmt.Printf("\n⚠️  Warning Level Pods (%d):\n", len(a.WarningPods))
		for _, pod := range a.WarningPods {
			if !contains(a.HighUsagePods, pod) {
				fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
			}
		}
	}

	fmt.Printf("\n")
	printRecommendations(a)
}

// formatPodInfo formats a single pod's memory information
func formatPodInfo(pod k8s.PodMemoryInfo, cfg *config.Config) string {
	pod.CalculateUsagePercent()

	// Format pod state info for diagnostics
	readyStatus := "Ready"
	if !pod.Ready {
		readyStatus = "NotReady"
	}
	stateInfo := fmt.Sprintf("[%s/%s]", pod.Phase, readyStatus)

	// If no memory metrics are available, show grey status (no info available)
	if pod.CurrentUsage == nil {
		status := "⚪"
		baseInfo := fmt.Sprintf("%s %s %s | Usage: %s | Request: %s (%s) | Limit: %s (%s)",
			status,
			fmt.Sprintf("%s/%s", pod.Namespace, pod.PodName),
			stateInfo,
			k8s.FormatMemory(pod.CurrentUsage),
			k8s.FormatMemory(pod.MemoryRequest),
			k8s.FormatPercent(pod.UsagePercent),
			k8s.FormatMemory(pod.MemoryLimit),
			k8s.FormatPercent(pod.LimitUsagePercent),
		)
		// Add labels and annotations if requested
		metadata := formatPodMetadata(pod, cfg)
		if metadata != "" {
			return fmt.Sprintf("%s\n%s", baseInfo, metadata)
		}
		return baseInfo
	}

	// Normal status logic when we have memory metrics
	status := "🔴"
	if pod.Ready && pod.Phase == "Running" {
		status = "🟢"
	} else if pod.Phase == "Pending" {
		status = "🟡"
	}

	baseInfo := fmt.Sprintf("%s %s %s | Usage: %s | Request: %s (%s) | Limit: %s (%s)",
		status,
		fmt.Sprintf("%s/%s", pod.Namespace, pod.PodName),
		stateInfo,
		k8s.FormatMemory(pod.CurrentUsage),
		k8s.FormatMemory(pod.MemoryRequest),
		k8s.FormatPercent(pod.UsagePercent),
		k8s.FormatMemory(pod.MemoryLimit),
		k8s.FormatPercent(pod.LimitUsagePercent),
	)

	// Add labels and annotations if requested
	metadata := formatPodMetadata(pod, cfg)
	if metadata != "" {
		return fmt.Sprintf("%s\n%s", baseInfo, metadata)
	}
	return baseInfo
}

// printRecommendations prints actionable recommendations based on the analysis
func printRecommendations(a *AnalysisResult) {
	fmt.Printf("📋 Recommendations:\n")

	podsWithoutLimits := 0
	podsWithoutRequests := 0

	for _, pod := range a.Report.Pods {
		if pod.MemoryLimit == nil {
			podsWithoutLimits++
		}
		if pod.MemoryRequest == nil {
			podsWithoutRequests++
		}
	}

	if podsWithoutLimits > 0 {
		fmt.Printf("• Set memory limits for %d pods to prevent OOM kills and resource contention\n", podsWithoutLimits)
	}

	if podsWithoutRequests > 0 {
		fmt.Printf("• Set memory requests for %d pods to enable proper scheduling\n", podsWithoutRequests)
	}

	if len(a.HighUsagePods) > 0 {
		fmt.Printf("• Monitor %d high-usage pods closely - consider scaling or optimization\n", len(a.HighUsagePods))
	}

	if a.Report.Summary.PodsWithMetrics < a.Report.Summary.RunningPods {
		fmt.Printf("• Consider installing/checking metrics-server for complete memory monitoring\n")
	}

	fmt.Printf("• Regular monitoring recommended with current threshold: %.1f%%\n", 80.0)
}

// formatPodMetadata formats labels and annotations for display based on configuration
func formatPodMetadata(pod k8s.PodMemoryInfo, cfg *config.Config) string {
	// Only show metadata if specifically requested
	if len(cfg.Labels) == 0 && len(cfg.Annotations) == 0 {
		return ""
	}

	var result strings.Builder

	// Format requested labels
	if requestedLabels := formatRequestedLabels(pod.Labels, cfg.Labels); len(requestedLabels) > 0 {
		result.WriteString("      📏 Labels:")
		for _, labelPair := range requestedLabels {
			result.WriteString(fmt.Sprintf("\n        - %s", labelPair))
		}
	}

	// Format requested annotations
	requestedAnnotations := formatRequestedAnnotations(pod.Annotations, cfg.Annotations)
	if len(requestedAnnotations) > 0 {
		if result.Len() > 0 {
			result.WriteString("\n") // Add separator if we already have labels
		}
		result.WriteString("      📝 Annotations:")
		for _, annotationPair := range requestedAnnotations {
			result.WriteString(fmt.Sprintf("\n        - %s", annotationPair))
		}
	}

	return result.String()
}

// formatRequestedLabels extracts and formats only the requested labels from a pod
func formatRequestedLabels(podLabels map[string]string, requestedLabels []string) []string {
	if len(requestedLabels) == 0 || len(podLabels) == 0 {
		return nil
	}

	result := make([]string, 0, len(requestedLabels))
	for _, requestedLabel := range requestedLabels {
		if value, exists := podLabels[requestedLabel]; exists {
			result = append(result, fmt.Sprintf("%s: %s", requestedLabel, value))
		}
	}

	sort.Strings(result) // Sort for consistent output
	return result
}

// formatRequestedAnnotations extracts and formats only the requested annotations from a pod
func formatRequestedAnnotations(podAnnotations map[string]string, requestedAnnotations []string) []string {
	if len(requestedAnnotations) == 0 || len(podAnnotations) == 0 {
		return nil
	}

	result := make([]string, 0, len(requestedAnnotations))
	for _, requestedAnnotation := range requestedAnnotations {
		if value, exists := podAnnotations[requestedAnnotation]; exists {
			// Limit annotation values to prevent extremely long output
			if len(value) > 80 {
				value = value[:77] + "..."
			}
			result = append(result, fmt.Sprintf("%s: %s", requestedAnnotation, value))
		}
	}

	sort.Strings(result) // Sort for consistent output
	return result
}

// Helper functions

func contains(pods []k8s.PodMemoryInfo, target k8s.PodMemoryInfo) bool {
	for _, pod := range pods {
		if pod.Namespace == target.Namespace && pod.PodName == target.PodName {
			return true
		}
	}
	return false
}
