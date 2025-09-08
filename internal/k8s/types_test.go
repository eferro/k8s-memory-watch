package k8s

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestFormatMemory(t *testing.T) {
	testCases := []struct {
		name     string
		quantity *resource.Quantity
		expected string
	}{
		{
			name:     "nil quantity",
			quantity: nil,
			expected: "N/A",
		},
		{
			name:     "bytes",
			quantity: resource.NewQuantity(512, resource.BinarySI),
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			quantity: resource.NewQuantity(1024*5, resource.BinarySI),
			expected: "5.0 KB",
		},
		{
			name:     "megabytes",
			quantity: resource.NewQuantity(1024*1024*100, resource.BinarySI),
			expected: "100.0 MB",
		},
		{
			name:     "gigabytes",
			quantity: resource.NewQuantity(1024*1024*1024*2, resource.BinarySI),
			expected: "2.00 GB",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatMemory(tc.quantity)
			if result != tc.expected {
				t.Errorf("FormatMemory() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	testCases := []struct {
		name     string
		percent  *float64
		expected string
	}{
		{
			name:     "nil percent",
			percent:  nil,
			expected: "N/A",
		},
		{
			name:     "zero percent",
			percent:  floatPtr(0.0),
			expected: "0.0%",
		},
		{
			name:     "normal percent",
			percent:  floatPtr(75.5),
			expected: "75.5%",
		},
		{
			name:     "over 100 percent",
			percent:  floatPtr(150.2),
			expected: "150.2%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatPercent(tc.percent)
			if result != tc.expected {
				t.Errorf("FormatPercent() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestPodMemoryInfo_CalculateUsagePercent(t *testing.T) {
	testCases := []struct {
		name                      string
		podInfo                   PodMemoryInfo
		expectedUsagePercent      *float64
		expectedLimitUsagePercent *float64
	}{
		{
			name: "no current usage",
			podInfo: PodMemoryInfo{
				CurrentUsage:  nil,
				MemoryRequest: resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100MB
				MemoryLimit:   resource.NewQuantity(1024*1024*200, resource.BinarySI), // 200MB
			},
			expectedUsagePercent:      nil,
			expectedLimitUsagePercent: nil,
		},
		{
			name: "with usage and request",
			podInfo: PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(1024*1024*50, resource.BinarySI),  // 50MB
				MemoryRequest: resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100MB
				MemoryLimit:   nil,
			},
			expectedUsagePercent:      floatPtr(50.0),
			expectedLimitUsagePercent: nil,
		},
		{
			name: "with usage, request, and limit",
			podInfo: PodMemoryInfo{
				CurrentUsage:  resource.NewQuantity(1024*1024*75, resource.BinarySI),  // 75MB
				MemoryRequest: resource.NewQuantity(1024*1024*50, resource.BinarySI),  // 50MB
				MemoryLimit:   resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100MB
			},
			expectedUsagePercent:      floatPtr(150.0),
			expectedLimitUsagePercent: floatPtr(75.0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.podInfo.CalculateUsagePercent()

			if !floatEqual(tc.podInfo.UsagePercent, tc.expectedUsagePercent) {
				t.Errorf("UsagePercent = %v, want %v",
					formatFloatPtr(tc.podInfo.UsagePercent),
					formatFloatPtr(tc.expectedUsagePercent))
			}

			if !floatEqual(tc.podInfo.LimitUsagePercent, tc.expectedLimitUsagePercent) {
				t.Errorf("LimitUsagePercent = %v, want %v",
					formatFloatPtr(tc.podInfo.LimitUsagePercent),
					formatFloatPtr(tc.expectedLimitUsagePercent))
			}
		})
	}
}

func TestPodMemoryInfo_String(t *testing.T) {
	podInfo := PodMemoryInfo{
		Namespace:     "default",
		PodName:       "test-pod",
		Phase:         "Running",
		Ready:         true,
		CurrentUsage:  resource.NewQuantity(1024*1024*75, resource.BinarySI),  // 75MB
		MemoryRequest: resource.NewQuantity(1024*1024*50, resource.BinarySI),  // 50MB
		MemoryLimit:   resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100MB
		Timestamp:     time.Now(),
	}

	result := podInfo.String()

	// Check that the string contains expected components
	expectedSubstrings := []string{
		"default/test-pod",
		"Phase: Running",
		"Ready: true",
		"75.0 MB",  // current usage
		"50.0 MB",  // request
		"100.0 MB", // limit
		"150.0%",   // usage vs request
		"75.0%",    // usage vs limit
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("String() result should contain %q, but got: %s", substr, result)
		}
	}
}

func TestContainerMemoryInfo_CalculateUsagePercent(t *testing.T) {
	container := ContainerMemoryInfo{
		CurrentUsage:  resource.NewQuantity(1024*1024*75, resource.BinarySI),  // 75MB
		MemoryRequest: resource.NewQuantity(1024*1024*50, resource.BinarySI),  // 50MB
		MemoryLimit:   resource.NewQuantity(1024*1024*100, resource.BinarySI), // 100MB
	}

	container.CalculateUsagePercent()

	if !floatEqual(container.UsagePercent, floatPtr(150.0)) {
		t.Errorf("UsagePercent = %v, want %v", formatFloatPtr(container.UsagePercent), formatFloatPtr(floatPtr(150.0)))
	}

	if !floatEqual(container.LimitUsagePercent, floatPtr(75.0)) {
		t.Errorf("LimitUsagePercent = %v, want %v", formatFloatPtr(container.LimitUsagePercent), formatFloatPtr(floatPtr(75.0)))
	}
}

// Helper functions

func floatPtr(f float64) *float64 {
	return &f
}

func floatEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func formatFloatPtr(f *float64) string {
	if f == nil {
		return "nil"
	}
	return FormatPercent(f)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
