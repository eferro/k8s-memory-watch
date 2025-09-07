package monitor

import (
	"strings"
	"testing"

	"github.com/eduardoferro/mgmt-monitoring/internal/k8s"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFormatPodInfo(t *testing.T) {
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
			expectedContains: []string{"⚪", "default/test-pod", "Usage: N/A"},
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
			expectedContains: []string{"🟢", "production/healthy-pod"},
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
			expectedContains: []string{"⚪", "default/pending-pod", "Usage: N/A"},
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
			expectedContains: []string{"⚪", "default/failing-pod", "Usage: N/A"},
			description:      "Non-ready pod without metrics should show grey symbol (not red)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPodInfo(tc.pod)

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
	// Test that grey symbol (no metrics) takes precedence over pod status
	testCases := []struct {
		phase      string
		ready      bool
		shouldShow string
	}{
		{"Running", true, "⚪"},   // Would normally be green
		{"Running", false, "⚪"},  // Would normally be red
		{"Pending", false, "⚪"},  // Would normally be yellow
		{"Failed", false, "⚪"},   // Would normally be red
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

			result := formatPodInfo(pod)

			if !strings.Contains(result, tc.shouldShow) {
				t.Errorf("Expected grey symbol ⚪ for phase=%s ready=%t, but got: %s", 
					tc.phase, tc.ready, result)
			}
		})
	}
}
