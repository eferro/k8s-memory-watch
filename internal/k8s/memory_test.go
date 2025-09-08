package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestProcessPodMemoryInfo_PopulatesContainers(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "p",
			Namespace: "ns",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("300Mi")},
						Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("512Mi")},
					},
				},
				{
					Name: "sidecar",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{},
						Limits:   corev1.ResourceList{},
					},
				},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	metrics := &metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: "p"},
		Containers: []metricsv1beta1.ContainerMetrics{
			{
				Name:  "app",
				Usage: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("200Mi")},
			},
			{
				Name:  "sidecar",
				Usage: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("100Mi")},
			},
		},
	}

	c := &Client{}
	info := c.processPodMemoryInfo(pod, metrics)

	if len(info.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(info.Containers))
	}

	// Verify app container
	var app *ContainerMemoryInfo
	var sidecar *ContainerMemoryInfo
	for i := range info.Containers {
		cc := &info.Containers[i]
		if cc.ContainerName == "app" {
			app = cc
		}
		if cc.ContainerName == "sidecar" {
			sidecar = cc
		}
	}
	if app == nil || sidecar == nil {
		t.Fatalf("missing expected container entries: app=%v sidecar=%v", app != nil, sidecar != nil)
	}

	if app.CurrentUsage == nil || app.CurrentUsage.Value() == 0 {
		t.Fatalf("app usage not set")
	}
	if app.MemoryRequest == nil || app.MemoryLimit == nil {
		t.Fatalf("app requests/limits not set")
	}
	if sidecar.MemoryRequest != nil || sidecar.MemoryLimit != nil {
		t.Fatalf("sidecar should not have requests/limits set")
	}
}

func TestProcessContainerMemoryInfo_PopulatesFields(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("100Mi")},
			Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("200Mi")},
		},
	}
	usage := corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("150Mi")}

	c := &Client{}
	info, req, lim, hasReq, hasLim := c.processContainerMemoryInfo(container, usage)

	if info.ContainerName != "app" || !hasReq || !hasLim {
		t.Fatalf("missing data")
	}
	if req != int64(100*1024*1024) || lim != int64(200*1024*1024) {
		t.Fatalf("wrong totals")
	}
	if info.CurrentUsage == nil || info.CurrentUsage.Value() == 0 {
		t.Fatalf("usage not set")
	}
}

func TestAggregatePodResources_SumsValues(t *testing.T) {
	r1 := resource.MustParse("100Mi")
	l1 := resource.MustParse("200Mi")
	r2 := resource.MustParse("50Mi")
	containers := []ContainerMemoryInfo{{MemoryRequest: &r1, MemoryLimit: &l1}, {MemoryRequest: &r2}}
	c := &Client{}
	req, lim, hasReq, hasLim := c.aggregatePodResources(containers)
	if !hasReq || hasLim {
		t.Fatalf("wrong flags")
	}
	if req == nil || req.Value() != int64(150*1024*1024) {
		t.Fatalf("wrong request")
	}
	if lim != nil {
		t.Fatalf("limit should be nil")
	}
}
