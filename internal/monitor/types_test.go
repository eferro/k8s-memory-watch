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

func TestFormatContainerSection_FormatsContainers(t *testing.T) {
	c := k8s.ContainerMemoryInfo{
		ContainerName: "app",
		CurrentUsage:  resource.NewQuantity(100*1024*1024, resource.BinarySI),
		MemoryRequest: resource.NewQuantity(200*1024*1024, resource.BinarySI),
		MemoryLimit:   resource.NewQuantity(400*1024*1024, resource.BinarySI),
	}
	result := formatContainerSection([]k8s.ContainerMemoryInfo{c})
	expected := "- app | Usage: 100.0 MB | Request: 200.0 MB (50.0%) | Limit: 400.0 MB (25.0%)"
	if !strings.Contains(result, expected) {
		t.Fatalf("expected %q in %q", expected, result)
	}
}

func TestFormatPodBaseInfo_FormatsBasicInfo(t *testing.T) {
	pod := k8s.PodMemoryInfo{
		PodName:       "app",
		Namespace:     "default",
		Phase:         "Running",
		Ready:         true,
		CurrentUsage:  resource.NewQuantity(50*1024*1024, resource.BinarySI),
		MemoryRequest: resource.NewQuantity(100*1024*1024, resource.BinarySI),
		MemoryLimit:   resource.NewQuantity(200*1024*1024, resource.BinarySI),
	}
	result := formatPodBaseInfo(&pod)
	expected := "🟢 default/app [Running/Ready] | Usage: 50.0 MB | Request: 100.0 MB (50.0%) | Limit: 200.0 MB (25.0%) | Limits: All | Requests: All"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
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

func TestBuildCSVRecord(t *testing.T) {
	cfg := &config.Config{
		Labels:               []string{"env", "team"},
		Annotations:          []string{"deployment.kubernetes.io/revision"},
		MemoryWarningPercent: 80.0,
	}

	timestamp := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)

	pod := &k8s.PodMemoryInfo{
		PodName:   "test-pod",
		Namespace: "default",
		Phase:     "Running",
		Ready:     true,
		Labels: map[string]string{
			"env":  "production",
			"team": "backend",
		},
		Annotations: map[string]string{
			"deployment.kubernetes.io/revision": "5",
		},
	}

	container := &k8s.ContainerMemoryInfo{
		ContainerName:     "app-container",
		CurrentUsage:      resource.NewQuantity(100*1024*1024, resource.BinarySI), // 100MB
		MemoryRequest:     resource.NewQuantity(200*1024*1024, resource.BinarySI), // 200MB
		MemoryLimit:       resource.NewQuantity(400*1024*1024, resource.BinarySI), // 400MB
		UsagePercent:      func() *float64 { v := 50.0; return &v }(),
		LimitUsagePercent: func() *float64 { v := 25.0; return &v }(),
	}

	// Calculate the actual values that will be returned
	expectedStatus := getContainerMemoryStatus(pod, container, cfg)
	expectedUsageBytes := formatBytesForCSV(container.CurrentUsage)
	expectedRequestBytes := formatBytesForCSV(container.MemoryRequest)
	expectedLimitBytes := formatBytesForCSV(container.MemoryLimit)
	expectedUsagePercent := formatPercentForCSV(container.UsagePercent)
	expectedLimitUsagePercent := formatPercentForCSV(container.LimitUsagePercent)

	expected := []string{
		"2023-12-01T10:00:00Z",
		expectedStatus,
		"default",
		"test-pod",
		"Running",
		"true",
		expectedUsageBytes,
		expectedRequestBytes,
		expectedLimitBytes,
		expectedUsagePercent,
		expectedLimitUsagePercent,
		"app-container",
		"production", // env label
		"backend",    // team label
		"5",          // revision annotation
	}

	result := buildCSVRecord(pod, container, cfg, timestamp)

	if len(result) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Field %d: expected '%s', got '%s'", i, exp, result[i])
		}
	}
}

func TestBuildCSVRecordForPod(t *testing.T) {
	cfg := &config.Config{
		Labels:               []string{"app", "version"},
		Annotations:          []string{"kubernetes.io/managed-by"},
		MemoryWarningPercent: 80.0,
	}

	timestamp := time.Date(2023, 12, 1, 15, 30, 0, 0, time.UTC)

	pod := &k8s.PodMemoryInfo{
		PodName:           "standalone-pod",
		Namespace:         "production",
		Phase:             "Running",
		Ready:             true,
		CurrentUsage:      resource.NewQuantity(300*1024*1024, resource.BinarySI),  // 300MB
		MemoryRequest:     resource.NewQuantity(500*1024*1024, resource.BinarySI),  // 500MB
		MemoryLimit:       resource.NewQuantity(1000*1024*1024, resource.BinarySI), // 1000MB
		UsagePercent:      func() *float64 { v := 60.0; return &v }(),
		LimitUsagePercent: func() *float64 { v := 30.0; return &v }(),
		Labels: map[string]string{
			"app":     "web-server",
			"version": "v1.2.3",
		},
		Annotations: map[string]string{
			"kubernetes.io/managed-by": "Deployment",
		},
	}

	// Calculate the actual values that will be returned
	expectedPodStatus := getMemoryStatus(pod, cfg)
	expectedPodUsageBytes := formatBytesForCSV(pod.CurrentUsage)
	expectedPodRequestBytes := formatBytesForCSV(pod.MemoryRequest)
	expectedPodLimitBytes := formatBytesForCSV(pod.MemoryLimit)
	expectedPodUsagePercent := formatPercentForCSV(pod.UsagePercent)
	expectedPodLimitUsagePercent := formatPercentForCSV(pod.LimitUsagePercent)

	expected := []string{
		"2023-12-01T15:30:00Z",
		expectedPodStatus,
		"production",
		"standalone-pod",
		"Running",
		"true",
		expectedPodUsageBytes,
		expectedPodRequestBytes,
		expectedPodLimitBytes,
		expectedPodUsagePercent,
		expectedPodLimitUsagePercent,
		"",           // empty container_name for pod-level record
		"web-server", // app label
		"v1.2.3",     // version label
		"Deployment", // managed-by annotation
	}

	result := buildCSVRecordForPod(pod, cfg, timestamp)

	if len(result) != len(expected) {
		t.Fatalf("Expected %d fields, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Field %d: expected '%s', got '%s'", i, exp, result[i])
		}
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
func TestFormatRequestedAnnotations_TruncatesLongValues(t *testing.T) {
	annotations := map[string]string{"key": strings.Repeat("a", 100)}
	result := formatRequestedAnnotations(annotations, []string{"key"})
	expected := "key: " + strings.Repeat("a", 77) + "..."
	if len(result) != 1 {
		t.Fatalf("expected one annotation, got %d", len(result))
	}
	if result[0] != expected {
		t.Errorf("expected %q, got %q", expected, result[0])
	}
}
