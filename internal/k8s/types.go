package k8s

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

// PodMemoryInfo contains memory information for a single pod
type PodMemoryInfo struct {
	Namespace string    `json:"namespace"`
	PodName   string    `json:"pod_name"`
	Timestamp time.Time `json:"timestamp"`

	// Current usage (from metrics API)
	CurrentUsage *resource.Quantity `json:"current_usage,omitempty"`

	// Limits and requests (from pod spec)
	MemoryRequest *resource.Quantity `json:"memory_request,omitempty"`
	MemoryLimit   *resource.Quantity `json:"memory_limit,omitempty"`

	// Calculated fields
	UsagePercent      *float64 `json:"usage_percent,omitempty"`       // Usage vs Request
	LimitUsagePercent *float64 `json:"limit_usage_percent,omitempty"` // Usage vs Limit

	// Pod status
	Phase string `json:"phase"`
	Ready bool   `json:"ready"`
}

// MemorySummary provides cluster-wide memory statistics
type MemorySummary struct {
	Timestamp          time.Time         `json:"timestamp"`
	TotalPods          int               `json:"total_pods"`
	RunningPods        int               `json:"running_pods"`
	PodsWithMetrics    int               `json:"pods_with_metrics"`
	PodsWithLimits     int               `json:"pods_with_limits"`
	PodsWithRequests   int               `json:"pods_with_requests"`
	TotalMemoryUsage   resource.Quantity `json:"total_memory_usage"`
	TotalMemoryLimit   resource.Quantity `json:"total_memory_limit"`
	TotalMemoryRequest resource.Quantity `json:"total_memory_request"`
	NamespaceCount     int               `json:"namespace_count"`
}

// FormatMemory formats a memory quantity in human-readable format
func FormatMemory(q *resource.Quantity) string {
	if q == nil {
		return "N/A"
	}

	value := q.Value()

	// Convert to appropriate unit
	if value >= 1024*1024*1024 { // GB
		return fmt.Sprintf("%.2f GB", float64(value)/(1024*1024*1024))
	} else if value >= 1024*1024 { // MB
		return fmt.Sprintf("%.1f MB", float64(value)/(1024*1024))
	} else if value >= 1024 { // KB
		return fmt.Sprintf("%.1f KB", float64(value)/1024)
	}

	return fmt.Sprintf("%d B", value)
}

// FormatPercent formats a percentage value
func FormatPercent(percent *float64) string {
	if percent == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%%", *percent)
}

// CalculateUsagePercent calculates usage percentage against request or limit
func (p *PodMemoryInfo) CalculateUsagePercent() {
	if p.CurrentUsage == nil {
		return
	}

	currentValue := float64(p.CurrentUsage.Value())

	// Calculate usage vs request
	if p.MemoryRequest != nil && p.MemoryRequest.Value() > 0 {
		requestValue := float64(p.MemoryRequest.Value())
		percent := (currentValue / requestValue) * 100
		p.UsagePercent = &percent
	}

	// Calculate usage vs limit
	if p.MemoryLimit != nil && p.MemoryLimit.Value() > 0 {
		limitValue := float64(p.MemoryLimit.Value())
		percent := (currentValue / limitValue) * 100
		p.LimitUsagePercent = &percent
	}
}

// String provides a human-readable representation of pod memory info
func (p *PodMemoryInfo) String() string {
	p.CalculateUsagePercent()

	return fmt.Sprintf(
		"Pod: %s/%s | Phase: %s | Ready: %t | Usage: %s | Request: %s (%s) | Limit: %s (%s)",
		p.Namespace,
		p.PodName,
		p.Phase,
		p.Ready,
		FormatMemory(p.CurrentUsage),
		FormatMemory(p.MemoryRequest),
		FormatPercent(p.UsagePercent),
		FormatMemory(p.MemoryLimit),
		FormatPercent(p.LimitUsagePercent),
	)
}
