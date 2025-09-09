package monitor

import (
	"testing"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
	"k8s.io/apimachinery/pkg/api/resource"
)

func qty(v int64) *resource.Quantity {
	q := resource.NewQuantity(v, resource.BinarySI)
	return q
}

func pct(v float64) *float64 {
	return &v
}

func TestGetMemoryStatus_NoData(t *testing.T) {
	pod := &k8s.PodMemoryInfo{}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "no_data" {
		t.Errorf("expected no_data, got %s", status)
	}
}

func TestGetMemoryStatus_NoConfig(t *testing.T) {
	pod := &k8s.PodMemoryInfo{CurrentUsage: qty(1)}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "no_config" {
		t.Errorf("expected no_config, got %s", status)
	}
}

func TestGetMemoryStatus_NoRequest(t *testing.T) {
	pod := &k8s.PodMemoryInfo{CurrentUsage: qty(1), MemoryLimit: qty(1)}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "no_request" {
		t.Errorf("expected no_request, got %s", status)
	}
}

func TestGetMemoryStatus_NoLimit(t *testing.T) {
	pod := &k8s.PodMemoryInfo{CurrentUsage: qty(1), MemoryRequest: qty(1)}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "no_limit" {
		t.Errorf("expected no_limit, got %s", status)
	}
}

func TestGetMemoryStatus_CriticalByUsage(t *testing.T) {
	pod := &k8s.PodMemoryInfo{
		CurrentUsage:  qty(1),
		MemoryRequest: qty(1),
		MemoryLimit:   qty(1),
		UsagePercent:  pct(95),
	}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "critical" {
		t.Errorf("expected critical, got %s", status)
	}
}

func TestGetMemoryStatus_CriticalByLimitUsage(t *testing.T) {
	pod := &k8s.PodMemoryInfo{
		CurrentUsage:      qty(1),
		MemoryRequest:     qty(1),
		MemoryLimit:       qty(1),
		LimitUsagePercent: pct(90),
	}
	status := getMemoryStatus(pod, &config.Config{})
	if status != "critical" {
		t.Errorf("expected critical, got %s", status)
	}
}

func TestGetMemoryStatus_Warning(t *testing.T) {
	pod := &k8s.PodMemoryInfo{
		CurrentUsage:  qty(1),
		MemoryRequest: qty(1),
		MemoryLimit:   qty(1),
		UsagePercent:  pct(80),
	}
	cfg := &config.Config{MemoryWarningPercent: 70}
	status := getMemoryStatus(pod, cfg)
	if status != "warning" {
		t.Errorf("expected warning, got %s", status)
	}
}

func TestGetMemoryStatus_NotReady(t *testing.T) {
	pod := &k8s.PodMemoryInfo{
		CurrentUsage:  qty(1),
		MemoryRequest: qty(1),
		MemoryLimit:   qty(1),
		UsagePercent:  pct(10),
		Ready:         false,
		Phase:         "Pending",
	}
	cfg := &config.Config{MemoryWarningPercent: 80}
	status := getMemoryStatus(pod, cfg)
	if status != "not_ready" {
		t.Errorf("expected not_ready, got %s", status)
	}
}

func TestGetMemoryStatus_Ok(t *testing.T) {
	pod := &k8s.PodMemoryInfo{
		CurrentUsage:  qty(1),
		MemoryRequest: qty(2),
		MemoryLimit:   qty(3),
		UsagePercent:  pct(50),
		Ready:         true,
		Phase:         "Running",
	}
	cfg := &config.Config{MemoryWarningPercent: 80}
	status := getMemoryStatus(pod, cfg)
	if status != "ok" {
		t.Errorf("expected ok, got %s", status)
	}
}
