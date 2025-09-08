package monitor

import (
	"strings"
	"testing"

	"github.com/eduardoferro/k8s-memory-watch/internal/config"
	"github.com/eduardoferro/k8s-memory-watch/internal/k8s"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestAnalyzeReport_PerContainerMessages(t *testing.T) {
	cfg := &config.Config{MemoryWarningPercent: 80.0}

	report := &MemoryReport{
		Summary: k8s.MemorySummary{},
		Pods: []k8s.PodMemoryInfo{
			{
				Namespace: "ns",
				PodName:   "p",
				Containers: []k8s.ContainerMemoryInfo{
					{
						ContainerName: "a",
						CurrentUsage:  resource.NewQuantity(1024*1024*500, resource.BinarySI), // 500Mi of 512Mi -> ~97.7% limit
						MemoryRequest: resource.NewQuantity(1024*1024*400, resource.BinarySI), // 125% of request
						MemoryLimit:   resource.NewQuantity(1024*1024*512, resource.BinarySI),
					},
					{
						ContainerName: "b",
						CurrentUsage:  resource.NewQuantity(1024*1024*100, resource.BinarySI),
						MemoryRequest: resource.NewQuantity(1024*1024*300, resource.BinarySI),
						MemoryLimit:   nil,
					},
				},
			},
		},
	}

	analysis := analyzeReport(report, cfg)
	joined := strings.Join(analysis.ProblemsFound, "\n")
	if !strings.Contains(joined, "Pod ns/p container a is using") {
		t.Fatalf("expected over-limit message for container a, got: %s", joined)
	}
	if !strings.Contains(joined, "Pod ns/p container b has no memory limit defined") {
		t.Fatalf("expected missing limit message for container b, got: %s", joined)
	}
}
