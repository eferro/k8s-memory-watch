package monitor

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	limitStateAll     = "All"
	limitStatePartial = "Partial"
	limitStateNone    = "None"
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
	for i := range r.Pods {
		pod := &r.Pods[i]
		if pod.Namespace != currentNamespace {
			currentNamespace = pod.Namespace
			fmt.Printf("\nNamespace: %s\n", currentNamespace)
			fmt.Printf("%s\n", strings.Repeat("-", 80))
		}

		fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
	}
	fmt.Printf("\n")
}

// PrintCSV prints pod memory information in CSV format
func (r *MemoryReport) PrintCSV(cfg *config.Config, showHeader bool) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header only if requested (first time)
	if showHeader {
		// Create dynamic header based on requested labels and annotations
		header := []string{
			"timestamp",
			"memory_status",
			"namespace",
			"pod_name",
			"phase",
			"ready",
			"usage_bytes",
			"request_bytes",
			"limit_bytes",
			"usage_percent",
			"limit_usage_percent",
			"container_name",
		}

		// Add label columns
		for _, label := range cfg.Labels {
			header = append(header, "label_"+strings.ReplaceAll(label, ".", "_"))
		}

		// Add annotation columns
		for _, annotation := range cfg.Annotations {
			header = append(header, "annotation_"+strings.ReplaceAll(annotation, ".", "_"))
		}

		// Write header
		if err := writer.Write(header); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV header: %v\n", err)
			return
		}
	}

	// Write pod data
	for i := range r.Pods {
		pod := &r.Pods[i]
		pod.CalculateUsagePercent()

		// If we have container breakdown, emit one row per container
		if len(pod.Containers) > 0 {
			for _, c := range pod.Containers {
				c.CalculateUsagePercent()
				record := buildCSVRecord(pod, &c, cfg, r.Summary.Timestamp)
				if err := writer.Write(record); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing CSV record: %v\n", err)
					continue
				}
			}
			continue
		}

		// Fallback: emit one row for the pod without specific container
		record := buildCSVRecordForPod(pod, cfg, r.Summary.Timestamp)
		if err := writer.Write(record); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing CSV record: %v\n", err)
			continue
		}
	}
}

// buildCSVRecord creates a CSV record for a container within a pod
func buildCSVRecord(pod *k8s.PodMemoryInfo, container *k8s.ContainerMemoryInfo, cfg *config.Config, timestamp time.Time) []string {
	record := []string{
		timestamp.Format(time.RFC3339),
		getMemoryStatus(pod, cfg),
		pod.Namespace,
		pod.PodName,
		pod.Phase,
		strconv.FormatBool(pod.Ready),
		formatBytesForCSV(container.CurrentUsage),
		formatBytesForCSV(container.MemoryRequest),
		formatBytesForCSV(container.MemoryLimit),
		formatPercentForCSV(container.UsagePercent),
		formatPercentForCSV(container.LimitUsagePercent),
		container.ContainerName,
	}

	// Add label values
	for _, label := range cfg.Labels {
		if value, exists := pod.Labels[label]; exists {
			record = append(record, value)
		} else {
			record = append(record, "")
		}
	}

	// Add annotation values
	for _, annotation := range cfg.Annotations {
		if value, exists := pod.Annotations[annotation]; exists {
			// Clean annotation values for CSV (remove newlines and quotes)
			cleanValue := strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "\r", " ")
			record = append(record, cleanValue)
		} else {
			record = append(record, "")
		}
	}

	return record
}

// buildCSVRecordForPod creates a CSV record for a pod without container breakdown
func buildCSVRecordForPod(pod *k8s.PodMemoryInfo, cfg *config.Config, timestamp time.Time) []string {
	record := []string{
		timestamp.Format(time.RFC3339),
		getMemoryStatus(pod, cfg),
		pod.Namespace,
		pod.PodName,
		pod.Phase,
		strconv.FormatBool(pod.Ready),
		formatBytesForCSV(pod.CurrentUsage),
		formatBytesForCSV(pod.MemoryRequest),
		formatBytesForCSV(pod.MemoryLimit),
		formatPercentForCSV(pod.UsagePercent),
		formatPercentForCSV(pod.LimitUsagePercent),
		"", // empty container_name for pod-level record
	}

	// Add label values
	for _, label := range cfg.Labels {
		if value, exists := pod.Labels[label]; exists {
			record = append(record, value)
		} else {
			record = append(record, "")
		}
	}

	// Add annotation values
	for _, annotation := range cfg.Annotations {
		if value, exists := pod.Annotations[annotation]; exists {
			// Clean annotation values for CSV (remove newlines and quotes)
			cleanValue := strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "\r", " ")
			record = append(record, cleanValue)
		} else {
			record = append(record, "")
		}
	}

	return record
}

// Helper functions for CSV formatting
func formatBytesForCSV(q *resource.Quantity) string {
	if q == nil {
		return ""
	}
	return strconv.FormatInt(q.Value(), 10)
}

func formatPercentForCSV(percent *float64) string {
	if percent == nil {
		return ""
	}
	return strconv.FormatFloat(*percent, 'f', 2, 64)
}

// getMemoryStatus determines the memory status of a pod for CSV output
func getMemoryStatus(pod *k8s.PodMemoryInfo, cfg *config.Config) string {
	// No metrics available
	if pod.CurrentUsage == nil {
		return "no_data"
	}

	// Check for missing configurations first
	if pod.MemoryRequest == nil && pod.MemoryLimit == nil {
		return "no_config"
	}

	if pod.MemoryRequest == nil {
		return "no_request"
	}

	if pod.MemoryLimit == nil {
		return "no_limit"
	}

	// Critical level checks (highest priority)
	if pod.UsagePercent != nil && *pod.UsagePercent >= 95.0 {
		return "critical"
	}

	if pod.LimitUsagePercent != nil && *pod.LimitUsagePercent >= 90.0 {
		return "critical"
	}

	// Warning level check
	if pod.UsagePercent != nil && *pod.UsagePercent >= cfg.MemoryWarningPercent {
		return "warning"
	}

	// Pod not running properly
	if !pod.Ready || pod.Phase != "Running" {
		return "not_ready"
	}

	// Everything looks good
	return "ok"
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

	// Filter pods to only those with All limits for pod-level sections
	filterAllLimited := func(pods []k8s.PodMemoryInfo) []k8s.PodMemoryInfo {
		if len(pods) == 0 {
			return pods
		}
		result := make([]k8s.PodMemoryInfo, 0, len(pods))
		for i := range pods {
			p := pods[i]
			lim, _ := limitState(&p)
			if lim == limitStateAll {
				result = append(result, p)
			}
		}
		return result
	}

	filteredHigh := filterAllLimited(a.HighUsagePods)
	filteredWarn := filterAllLimited(a.WarningPods)

	if len(filteredHigh) > 0 {
		fmt.Printf("\n🔥 High Memory Usage Pods (%d):\n", len(filteredHigh))
		for i := range filteredHigh {
			pod := &filteredHigh[i]
			fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
		}
	}

	if len(filteredWarn) > 0 {
		fmt.Printf("\n⚠️  Warning Level Pods (%d):\n", len(filteredWarn))
		for i := range filteredWarn {
			pod := &filteredWarn[i]
			if !contains(filteredHigh, pod) {
				fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
			}
		}
	}

	fmt.Printf("\n")
	printRecommendations(a)
}

// formatPodInfo formats a single pod's memory information
func formatPodInfo(pod *k8s.PodMemoryInfo, cfg *config.Config) string {
	var parts []string
	parts = append(parts, formatPodBaseInfo(pod))
	if c := formatContainerSection(pod.Containers); c != "" {
		parts = append(parts, c)
	}
	if m := formatMetadataSection(pod, cfg); m != "" {
		parts = append(parts, m)
	}
	return strings.Join(parts, "\n")
}

func podStatusSymbol(pod *k8s.PodMemoryInfo) string {
	if pod.CurrentUsage == nil {
		return "⚪"
	}
	if pod.Ready && pod.Phase == "Running" {
		return "🟢"
	}
	if pod.Phase == "Pending" {
		return "🟡"
	}
	return "🔴"
}

func formatPodBaseInfo(pod *k8s.PodMemoryInfo) string {
	pod.CalculateUsagePercent()
	readyStatus := "Ready"
	if !pod.Ready {
		readyStatus = "NotReady"
	}
	stateInfo := fmt.Sprintf("[%s/%s]", pod.Phase, readyStatus)
	limState, reqState := limitState(pod)
	return fmt.Sprintf("%s %s %s | Usage: %s | Request: %s (%s) | Limit: %s (%s) | Limits: %s | Requests: %s",
		podStatusSymbol(pod),
		fmt.Sprintf("%s/%s", pod.Namespace, pod.PodName),
		stateInfo,
		k8s.FormatMemory(pod.CurrentUsage),
		k8s.FormatMemory(pod.MemoryRequest),
		k8s.FormatPercent(pod.UsagePercent),
		k8s.FormatMemory(pod.MemoryLimit),
		k8s.FormatPercent(pod.LimitUsagePercent),
		limState,
		reqState,
	)
}

func formatContainerSection(containers []k8s.ContainerMemoryInfo) string {
	if len(containers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("      🧩 Containers:")
	for i := range containers {
		c := containers[i]
		c.CalculateUsagePercent()
		b.WriteString("\n        - " + c.ContainerName)
		b.WriteString(" | Usage: " + k8s.FormatMemory(c.CurrentUsage))
		b.WriteString(" | Request: " + k8s.FormatMemory(c.MemoryRequest))
		b.WriteString(" (" + k8s.FormatPercent(c.UsagePercent) + ") | Limit: ")
		b.WriteString(k8s.FormatMemory(c.MemoryLimit))
		b.WriteString(" (" + k8s.FormatPercent(c.LimitUsagePercent) + ")")
	}
	return b.String()
}

// printRecommendations prints actionable recommendations based on the analysis
func printRecommendations(a *AnalysisResult) {
	fmt.Printf("📋 Recommendations:\n")

	podsWithoutLimits := 0
	podsWithoutRequests := 0

	for i := range a.Report.Pods {
		pod := &a.Report.Pods[i]
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

// formatMetadataSection formats labels and annotations for display based on configuration
func formatMetadataSection(pod *k8s.PodMemoryInfo, cfg *config.Config) string {
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

func contains(pods []k8s.PodMemoryInfo, target *k8s.PodMemoryInfo) bool {
	for i := range pods {
		pod := &pods[i]
		if pod.Namespace == target.Namespace && pod.PodName == target.PodName {
			return true
		}
	}
	return false
}

func limitState(pod *k8s.PodMemoryInfo) (limits, requests string) {
	summarize := func(all bool, anyPresent bool) string {
		switch {
		case all:
			return limitStateAll
		case anyPresent:
			return limitStatePartial
		default:
			return limitStateNone
		}
	}

	if len(pod.Containers) == 0 {
		limits = summarize(pod.MemoryLimit != nil, pod.MemoryLimit != nil)
		requests = summarize(pod.MemoryRequest != nil, pod.MemoryRequest != nil)
		return limits, requests
	}

	allLimits, anyLimits := true, false
	allRequests, anyRequests := true, false
	for i := range pod.Containers {
		c := pod.Containers[i]
		if c.MemoryLimit != nil {
			anyLimits = true
		} else {
			allLimits = false
		}
		if c.MemoryRequest != nil {
			anyRequests = true
		} else {
			allRequests = false
		}
	}

	limits = summarize(allLimits, anyLimits)
	requests = summarize(allRequests, anyRequests)
	return limits, requests
}
