package monitor

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFormatPodInfo(t *testing.T) {
	// Create empty config for testing (no labels/annotations to display)
	cfg := &config.Config{
		Labels:      []string{},
		Annotations: []string{},
	}

	testCases := []struct {
		name             string
		pod              k8s.PodMemoryInfo
		expectedSymbol   string
		expectedContains []string
		description      string
	}{
		{
			name: "pod_with_no_memory_metrics",
			pod: k8s.PodMemoryInfo{
				PodName:      "test-pod",
				Namespace:    "default",
				Phase:        "Running",
				Ready:        true,
				CurrentUsage: nil, // No metrics available
				MemoryRequest: func() *resource.Quantity {
					q := resource.MustParse("100Mi")
					return &q
				}(),
				MemoryLimit: func() *resource.Quantity {
					q := resource.MustParse("200Mi")
					return &q
				}(),
			},
			expectedSymbol:   "⚪",
			expectedContains: []string{"⚪", "default/test-pod", "[Running/Ready]", "Usage: N/A"},
			description:      "Pod without memory metrics should show grey symbol",
		},
		{
			name: "running_pod_with_metrics",
			pod: k8s.PodMemoryInfo{
				PodName:   "healthy-pod",
				Namespace: "production",
				Phase:     "Running",
				Ready:     true,
				CurrentUsage: func() *resource.Quantity {
					q := resource.MustParse("50Mi")
					return &q
				}(),
				MemoryRequest: func() *resource.Quantity {
					q := resource.MustParse("100Mi")
					return &q
				}(),
				MemoryLimit: func() *resource.Quantity {
					q := resource.MustParse("200Mi")
					return &q
				}(),
			},
			expectedSymbol:   "🟢",
			expectedContains: []string{"🟢", "production/healthy-pod", "[Running/Ready]"},
			description:      "Running pod with metrics should show green symbol",
		},
		{
			name: "pending_pod_with_no_metrics",
			pod: k8s.PodMemoryInfo{
				PodName:      "pending-pod",
				Namespace:    "default",
				Phase:        "Pending",
				Ready:        false,
				CurrentUsage: nil, // No metrics available
				MemoryRequest: func() *resource.Quantity {
					q := resource.MustParse("100Mi")
					return &q
				}(),
			},
			expectedSymbol:   "⚪",
			expectedContains: []string{"⚪", "default/pending-pod", "[Pending/NotReady]", "Usage: N/A"},
			description:      "Pending pod without metrics should show grey symbol (not yellow)",
		},
		{
			name: "failing_pod_with_no_metrics",
			pod: k8s.PodMemoryInfo{
				PodName:      "failing-pod",
				Namespace:    "default",
				Phase:        "Running",
				Ready:        false,
				CurrentUsage: nil, // No metrics available
			},
			expectedSymbol:   "⚪",
			expectedContains: []string{"⚪", "default/failing-pod", "[Running/NotReady]", "Usage: N/A"},
			description:      "Non-ready pod without metrics should show grey symbol (not red)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPodInfo(&tc.pod, cfg)

			// Check if the expected symbol is present
			if !strings.Contains(result, tc.expectedSymbol) {
				t.Errorf("%s: Expected symbol '%s' not found in result: %s",
					tc.description, tc.expectedSymbol, result)
			}

			// Check if all expected strings are present
			for _, expected := range tc.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("%s: Expected string '%s' not found in result: %s",
						tc.description, expected, result)
				}
			}

			// Log the result for debugging
			t.Logf("%s: %s", tc.description, result)
		})
	}
}

func TestFormatPodInfo_NoMetricsOverridesStatus(t *testing.T) {
	// Create empty config for testing (no labels/annotations to display)
	cfg := &config.Config{
		Labels:      []string{},
		Annotations: []string{},
	}

	// Test that grey symbol (no metrics) takes precedence over pod status
	testCases := []struct {
		phase      string
		ready      bool
		shouldShow string
	}{
		{"Running", true, "⚪"},  // Would normally be green
		{"Running", false, "⚪"}, // Would normally be red
		{"Pending", false, "⚪"}, // Would normally be yellow
		{"Failed", false, "⚪"},  // Would normally be red
	}

	for _, tc := range testCases {
		t.Run("phase_"+tc.phase, func(t *testing.T) {
			pod := k8s.PodMemoryInfo{
				PodName:      "test-pod",
				Namespace:    "default",
				Phase:        tc.phase,
				Ready:        tc.ready,
				CurrentUsage: nil, // No metrics - this should override status
			}

			result := formatPodInfo(&pod, cfg)

			if !strings.Contains(result, tc.shouldShow) {
				t.Errorf("Expected grey symbol ⚪ for phase=%s ready=%t, but got: %s",
					tc.phase, tc.ready, result)
			}
		})
	}
}

func TestGetMemoryStatus(t *testing.T) {
	cfg := &config.Config{
		MemoryWarningPercent: 80.0,
	}

	tests := []struct {
		name     string
		pod      k8s.PodMemoryInfo
		expected string
	}{
		{
			name: "no data - no current usage",
			pod: k8s.PodMemoryInfo{
				CurrentUsage: nil,
			},
			expected: "no_data",
		},
		{
			name: "no config - no request or limit",
			pod: k8s.PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(100*1024*1024, resource.BinarySI), // 100MB
				MemoryRequest: nil,
				MemoryLimit:   nil,
			},
			expected: "no_config",
		},
		{
			name: "critical - high usage vs request",
			pod: k8s.PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(950*1024*1024, resource.BinarySI),  // 950MB
				MemoryRequest: resource.NewQuantity(1000*1024*1024, resource.BinarySI), // 1000MB
				MemoryLimit:   resource.NewQuantity(2000*1024*1024, resource.BinarySI), // 2000MB
				Phase:         "Running",
				Ready:         true,
			},
			expected: "critical",
		},
		{
			name: "warning - above warning threshold",
			pod: k8s.PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(850*1024*1024, resource.BinarySI),  // 850MB
				MemoryRequest: resource.NewQuantity(1000*1024*1024, resource.BinarySI), // 1000MB
				MemoryLimit:   resource.NewQuantity(2000*1024*1024, resource.BinarySI), // 2000MB
				Phase:         "Running",
				Ready:         true,
			},
			expected: "warning",
		},
		{
			name: "not ready - pod not ready",
			pod: k8s.PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(100*1024*1024, resource.BinarySI),  // 100MB
				MemoryRequest: resource.NewQuantity(1000*1024*1024, resource.BinarySI), // 1000MB
				MemoryLimit:   resource.NewQuantity(2000*1024*1024, resource.BinarySI), // 2000MB
				Phase:         "Running",
				Ready:         false,
			},
			expected: "not_ready",
		},
		{
			name: "ok - normal operation",
			pod: k8s.PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(500*1024*1024, resource.BinarySI),  // 500MB
				MemoryRequest: resource.NewQuantity(1000*1024*1024, resource.BinarySI), // 1000MB
				MemoryLimit:   resource.NewQuantity(2000*1024*1024, resource.BinarySI), // 2000MB
				Phase:         "Running",
				Ready:         true,
			},
			expected: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate usage percentages
			tt.pod.CalculateUsagePercent()
			result := getMemoryStatus(&tt.pod, cfg)
			if result != tt.expected {
				t.Errorf("getMemoryStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatPodInfo_IncludesContainerIdentifiers(t *testing.T) {
	cfg := &config.Config{Labels: []string{}, Annotations: []string{}}

	pod := k8s.PodMemoryInfo{
		Namespace:     "default",
		PodName:       "multi",
		Phase:         "Running",
		Ready:         true,
		CurrentUsage:  func() *resource.Quantity { q := resource.MustParse("300Mi"); return &q }(),
		MemoryRequest: func() *resource.Quantity { q := resource.MustParse("600Mi"); return &q }(),
		MemoryLimit:   func() *resource.Quantity { q := resource.MustParse("1Gi"); return &q }(),
	}

	pod.Containers = []k8s.ContainerMemoryInfo{
		{
			ContainerName: "app",
			CurrentUsage:  resource.NewQuantity(1024*1024*200, resource.BinarySI), // 200Mi approx
			MemoryRequest: resource.NewQuantity(1024*1024*300, resource.BinarySI),
			MemoryLimit:   resource.NewQuantity(1024*1024*512, resource.BinarySI),
		},
		{
			ContainerName: "sidecar",
			CurrentUsage:  resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100Mi approx
			MemoryRequest: nil,
			MemoryLimit:   nil,
		},
	}

	out := formatPodInfo(&pod, cfg)
	if !strings.Contains(out, "default/multi") {
		t.Fatalf("base pod info missing")
	}
	if !strings.Contains(out, "app") || !strings.Contains(out, "sidecar") {
		t.Fatalf("expected container names in output, got: %s", out)
	}
}

func TestFormatPodInfo_ShowsLimitState(t *testing.T) {
	cfg := &config.Config{}
	pod := k8s.PodMemoryInfo{
		Namespace: "ns",
		PodName:   "p",
		Phase:     "Running",
		Ready:     true,
		Containers: []k8s.ContainerMemoryInfo{
			{ContainerName: "a", MemoryLimit: resource.NewQuantity(1024*1024*256, resource.BinarySI)},
			{ContainerName: "b"},
		},
	}
	out := formatPodInfo(&pod, cfg)
	if !strings.Contains(out, "Limit state: Partial") && !strings.Contains(out, "Limits: Partial") {
		t.Fatalf("expected Partial limit state in output, got: %s", out)
	}
}

func TestPrintCSV_PerContainerRows(t *testing.T) {
	cfg := &config.Config{Output: config.OutputFormatCSV}

	report := MemoryReport{
		Summary: k8s.MemorySummary{Timestamp: time.Now()},
		Pods: []k8s.PodMemoryInfo{
			{
				Namespace: "ns",
				PodName:   "p1",
				Phase:     "Running",
				Ready:     true,
				Containers: []k8s.ContainerMemoryInfo{
					{ContainerName: "a"},
					{ContainerName: "b"},
				},
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	report.PrintCSV(cfg, true)

	_ = w.Close()
	os.Stdout = oldStdout
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, r)

	out := buf.String()
	if !strings.Contains(out, "container_name") {
		t.Fatalf("expected container_name header, got: %s", out)
	}
	if !strings.Contains(out, ",ns,p1,Running,true,,,,,,a") || !strings.Contains(out, ",ns,p1,Running,true,,,,,,b") {
		t.Fatalf("expected two rows for containers a and b, got: %s", out)
	}
}

func TestPrintAnalysis_FiltersPartialLimitPods(t *testing.T) {
	cfg := &config.Config{MemoryWarningPercent: 80.0}

	podAll := k8s.PodMemoryInfo{
		Namespace: "ns", PodName: "all", Phase: "Running", Ready: true,
		CurrentUsage:  resource.NewQuantity(950*1024*1024, resource.BinarySI),
		MemoryRequest: resource.NewQuantity(1000*1024*1024, resource.BinarySI),
		MemoryLimit:   resource.NewQuantity(1000*1024*1024, resource.BinarySI),
		Containers:    []k8s.ContainerMemoryInfo{{ContainerName: "c", MemoryLimit: resource.NewQuantity(1, resource.BinarySI), MemoryRequest: resource.NewQuantity(1, resource.BinarySI)}},
	}
	podAll.CalculateUsagePercent()

	podPartial := k8s.PodMemoryInfo{
		Namespace: "ns", PodName: "partial", Phase: "Running", Ready: true,
		CurrentUsage:  resource.NewQuantity(900*1024*1024, resource.BinarySI),
		MemoryRequest: nil, // pod-level absent due to not all containers requesting
		MemoryLimit:   nil, // absent due to partial limits
		Containers: []k8s.ContainerMemoryInfo{
			{ContainerName: "a", MemoryLimit: resource.NewQuantity(1, resource.BinarySI)},
			{ContainerName: "b"},
		},
	}
	podPartial.CalculateUsagePercent()

	analysis := &AnalysisResult{
		Report:        MemoryReport{Pods: []k8s.PodMemoryInfo{podAll, podPartial}},
		HighUsagePods: []k8s.PodMemoryInfo{podAll, podPartial},
		WarningPods:   []k8s.PodMemoryInfo{podAll, podPartial},
		ProblemsFound: []string{"dummy"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	analysis.PrintAnalysis(cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, r)
	out := buf.String()

	if strings.Contains(out, "partial") {
		t.Fatalf("expected pod with Partial limits to be omitted from High/Warning sections, got: %s", out)
	}
	if !strings.Contains(out, "all") {
		t.Fatalf("expected pod with All limits to appear, got: %s", out)
	}
}
