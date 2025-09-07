package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// GetAllPodsMemoryInfo retrieves memory information for all pods across all namespaces
func (c *Client) GetAllPodsMemoryInfo(ctx context.Context) ([]PodMemoryInfo, *MemorySummary, error) {
	slog.Info("Starting to collect memory information for all pods")

	// Get all namespaces
	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	slog.Info("Found namespaces", "count", len(namespaces.Items))

	var allPods []PodMemoryInfo
	summary := &MemorySummary{
		Timestamp:          time.Now(),
		NamespaceCount:     len(namespaces.Items),
		TotalMemoryUsage:   *resource.NewQuantity(0, resource.BinarySI),
		TotalMemoryLimit:   *resource.NewQuantity(0, resource.BinarySI),
		TotalMemoryRequest: *resource.NewQuantity(0, resource.BinarySI),
	}

	// Process each namespace
	for _, ns := range namespaces.Items {
		nsName := ns.Name
		slog.Debug("Processing namespace", "namespace", nsName)

		pods, nsUsage, err := c.getNamespacePodsMemoryInfo(ctx, nsName)
		if err != nil {
			slog.Warn("Failed to get pods for namespace", "namespace", nsName, "error", err)
			continue
		}

		allPods = append(allPods, pods...)

		// Update summary
		summary.TotalPods += len(pods)
		summary.TotalMemoryUsage.Add(nsUsage.TotalMemoryUsage)
		summary.TotalMemoryLimit.Add(nsUsage.TotalMemoryLimit)
		summary.TotalMemoryRequest.Add(nsUsage.TotalMemoryRequest)
		summary.RunningPods += nsUsage.RunningPods
		summary.PodsWithMetrics += nsUsage.PodsWithMetrics
		summary.PodsWithLimits += nsUsage.PodsWithLimits
		summary.PodsWithRequests += nsUsage.PodsWithRequests
	}

	slog.Info("Memory collection completed",
		"total_pods", summary.TotalPods,
		"running_pods", summary.RunningPods,
		"pods_with_metrics", summary.PodsWithMetrics,
		"total_usage", FormatMemory(&summary.TotalMemoryUsage))

	return allPods, summary, nil
}

// getNamespacePodsMemoryInfo gets memory info for pods in a specific namespace
func (c *Client) getNamespacePodsMemoryInfo(ctx context.Context, namespace string) (
	[]PodMemoryInfo, *MemorySummary, error) {
	// Get all pods in the namespace
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	// Get metrics for the namespace (this might fail if metrics-server is not available)
	var podMetrics *metricsv1beta1.PodMetricsList
	podMetrics, err = c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("Failed to get pod metrics for namespace", "namespace", namespace, "error", err)
		// Continue without metrics - we can still show limits/requests
	}

	// Create a map of pod metrics for quick lookup
	metricsMap := make(map[string]*metricsv1beta1.PodMetrics)
	if podMetrics != nil {
		for i := range podMetrics.Items {
			pm := &podMetrics.Items[i]
			metricsMap[pm.Name] = pm
		}
	}

	podInfos := make([]PodMemoryInfo, 0, len(pods.Items))
	summary := &MemorySummary{
		TotalMemoryUsage:   *resource.NewQuantity(0, resource.BinarySI),
		TotalMemoryLimit:   *resource.NewQuantity(0, resource.BinarySI),
		TotalMemoryRequest: *resource.NewQuantity(0, resource.BinarySI),
	}

	// Process each pod
	for _, pod := range pods.Items {
		podInfo := c.processPodMemoryInfo(&pod, metricsMap[pod.Name])
		podInfos = append(podInfos, podInfo)

		// Update summary
		if pod.Status.Phase == corev1.PodRunning {
			summary.RunningPods++
		}
		if podInfo.CurrentUsage != nil {
			summary.PodsWithMetrics++
			summary.TotalMemoryUsage.Add(*podInfo.CurrentUsage)
		}
		if podInfo.MemoryRequest != nil {
			summary.PodsWithRequests++
			summary.TotalMemoryRequest.Add(*podInfo.MemoryRequest)
		}
		if podInfo.MemoryLimit != nil {
			summary.PodsWithLimits++
			summary.TotalMemoryLimit.Add(*podInfo.MemoryLimit)
		}
	}

	return podInfos, summary, nil
}

// processPodMemoryInfo creates PodMemoryInfo from pod spec and metrics
func (c *Client) processPodMemoryInfo(pod *corev1.Pod, metrics *metricsv1beta1.PodMetrics) PodMemoryInfo {
	podInfo := PodMemoryInfo{
		Namespace: pod.Namespace,
		PodName:   pod.Name,
		Timestamp: time.Now(),
		Phase:     string(pod.Status.Phase),
		Ready:     c.isPodReady(pod),
	}

	// Extract memory limits and requests from all containers
	var totalRequest, totalLimit int64
	hasRequest, hasLimit := false, false

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if memReq, exists := container.Resources.Requests[corev1.ResourceMemory]; exists {
				totalRequest += memReq.Value()
				hasRequest = true
			}
		}

		if container.Resources.Limits != nil {
			if memLimit, exists := container.Resources.Limits[corev1.ResourceMemory]; exists {
				totalLimit += memLimit.Value()
				hasLimit = true
			}
		}
	}

	if hasRequest {
		podInfo.MemoryRequest = resource.NewQuantity(totalRequest, resource.BinarySI)
	}
	if hasLimit {
		podInfo.MemoryLimit = resource.NewQuantity(totalLimit, resource.BinarySI)
	}

	// Extract current usage from metrics
	if metrics != nil {
		var totalUsage int64
		for _, container := range metrics.Containers {
			if container.Usage != nil {
				if memUsage, exists := container.Usage[corev1.ResourceMemory]; exists {
					totalUsage += memUsage.Value()
				}
			}
		}
		if totalUsage > 0 {
			podInfo.CurrentUsage = resource.NewQuantity(totalUsage, resource.BinarySI)
		}
	}

	return podInfo
}

// isPodReady checks if a pod is ready
func (c *Client) isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
