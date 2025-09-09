package monitor

import (
	"fmt"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
)

// AnalysisReporter handles analysis output formatting
type AnalysisReporter struct{}

// NewAnalysisReporter creates a new analysis reporter
func NewAnalysisReporter() *AnalysisReporter {
	return &AnalysisReporter{}
}

// PrintAnalysis prints the analysis results with warnings and recommendations
func (r *AnalysisReporter) PrintAnalysis(analysis *AnalysisResult, cfg *config.Config) {
	fmt.Printf("\n")
	fmt.Printf("=== Memory Usage Analysis ===\n")

	r.printProblems(analysis)
	r.printHighUsagePods(analysis, cfg)
	r.printWarningPods(analysis, cfg)

	fmt.Printf("\n")
	printRecommendations(analysis)
}

// printProblems prints the detected problems
func (r *AnalysisReporter) printProblems(analysis *AnalysisResult) {
	if len(analysis.ProblemsFound) == 0 {
		fmt.Printf("✅ No memory issues detected.\n")
		return
	}

	fmt.Printf("🚨 Found %d potential issues:\n\n", len(analysis.ProblemsFound))
	for i, problem := range analysis.ProblemsFound {
		fmt.Printf("%d. %s\n", i+1, problem)
	}
}

// printHighUsagePods prints pods with high memory usage
func (r *AnalysisReporter) printHighUsagePods(analysis *AnalysisResult, cfg *config.Config) {
	filteredHigh := r.filterAllLimited(analysis.HighUsagePods)
	if len(filteredHigh) == 0 {
		return
	}

	fmt.Printf("\n🔥 High Memory Usage Pods (%d):\n", len(filteredHigh))
	for i := range filteredHigh {
		pod := &filteredHigh[i]
		fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
	}
}

// printWarningPods prints pods with warning-level memory usage
func (r *AnalysisReporter) printWarningPods(analysis *AnalysisResult, cfg *config.Config) {
	filteredHigh := r.filterAllLimited(analysis.HighUsagePods)
	filteredWarn := r.filterAllLimited(analysis.WarningPods)

	if len(filteredWarn) == 0 {
		return
	}

	fmt.Printf("\n⚠️  Warning Level Pods (%d):\n", len(filteredWarn))
	for i := range filteredWarn {
		pod := &filteredWarn[i]
		if !contains(filteredHigh, pod) {
			fmt.Printf("  %s\n", formatPodInfo(pod, cfg))
		}
	}
}

// filterAllLimited filters pods to only those with All limits for pod-level sections
func (r *AnalysisReporter) filterAllLimited(pods []k8s.PodMemoryInfo) []k8s.PodMemoryInfo {
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
