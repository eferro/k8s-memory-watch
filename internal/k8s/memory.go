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
	return c.GetPodsMemoryInfo(ctx, "", true)
}

// GetPodsMemoryInfo retrieves memory information for pods
// If namespace is empty and allNamespaces is true, gets all pods from all namespaces
// If namespace is specified, gets pods only from that namespace
func (c *Client) GetPodsMemoryInfo(ctx context.Context, namespace string, allNamespaces bool) (
	[]PodMemoryInfo, *MemorySummary, error) {
	if namespace != "" && allNamespaces {
		return nil, nil, fmt.Errorf("cannot specify both namespace and allNamespaces")
	}

	if namespace != "" {
		// Monitor specific namespace
		slog.Info("Starting to collect memory information for specific namespace", "namespace", namespace)
		return c.getSingleNamespacePodsMemoryInfo(ctx, namespace)
	}

	if allNamespaces {
		// Monitor all namespaces
		slog.Info("Starting to collect memory information for all namespaces")
		return c.getAllNamespacesPodsMemoryInfo(ctx)
	}

	// Default behavior (should not reach here with current config logic)
	return c.getAllNamespacesPodsMemoryInfo(ctx)
}

// getSingleNamespacePodsMemoryInfo gets memory info for pods in a single namespace
func (c *Client) getSingleNamespacePodsMemoryInfo(ctx context.Context, namespace string) (
	[]PodMemoryInfo, *MemorySummary, error) {
	pods, nsUsage, err := c.getNamespacePodsMemoryInfo(ctx, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pods for namespace %s: %w", namespace, err)
	}

	// Create summary for single namespace
	summary := &MemorySummary{
		Timestamp:          time.Now(),
		NamespaceCount:     1,
		TotalPods:          len(pods),
		TotalMemoryUsage:   nsUsage.TotalMemoryUsage,
		TotalMemoryLimit:   nsUsage.TotalMemoryLimit,
		TotalMemoryRequest: nsUsage.TotalMemoryRequest,
		RunningPods:        nsUsage.RunningPods,
		PodsWithMetrics:    nsUsage.PodsWithMetrics,
		PodsWithLimits:     nsUsage.PodsWithLimits,
		PodsWithRequests:   nsUsage.PodsWithRequests,
	}

	slog.Info("Memory collection completed for namespace",
		"namespace", namespace,
		"total_pods", summary.TotalPods,
		"running_pods", summary.RunningPods,
		"pods_with_metrics", summary.PodsWithMetrics,
		"total_usage", FormatMemory(&summary.TotalMemoryUsage))

	return pods, summary, nil
}

// getAllNamespacesPodsMemoryInfo gets memory info for all namespaces
func (c *Client) getAllNamespacesPodsMemoryInfo(ctx context.Context) ([]PodMemoryInfo, *MemorySummary, error) {
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
	for i := range namespaces.Items {
		nsName := namespaces.Items[i].Name
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
	for i := range pods.Items {
		pod := &pods.Items[i]
		podInfo := c.processPodMemoryInfo(pod, metricsMap[pod.Name])
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

func (c *Client) processContainerMemoryInfo(container *corev1.Container, usage corev1.ResourceList) (ContainerMemoryInfo, int64, int64, bool, bool) {
	info := ContainerMemoryInfo{ContainerName: container.Name}
	var req, lim int64
	if r, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
		req = r.Value()
		v := r
		info.MemoryRequest = &v
	}
	if l, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
		lim = l.Value()
		v := l
		info.MemoryLimit = &v
	}
	if u, ok := usage[corev1.ResourceMemory]; ok {
		v := u
		info.CurrentUsage = &v
	}
	return info, req, lim, info.MemoryRequest != nil, info.MemoryLimit != nil
}

// processPodMemoryInfo creates PodMemoryInfo from pod spec and metrics
func (c *Client) processPodMemoryInfo(pod *corev1.Pod, metrics *metricsv1beta1.PodMetrics) PodMemoryInfo {
	podInfo := PodMemoryInfo{
		Namespace:   pod.Namespace,
		PodName:     pod.Name,
		Timestamp:   time.Now(),
		Phase:       string(pod.Status.Phase),
		Ready:       c.isPodReady(pod),
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	// Copy pod labels and annotations
	for k, v := range pod.Labels {
		podInfo.Labels[k] = v
	}
	for k, v := range pod.Annotations {
		podInfo.Annotations[k] = v
	}

	// Extract memory limits and requests from all containers
	var totalRequest, totalLimit int64
	hasRequest, hasLimit := true, true

	// Build a map of metrics by container name
	metricsByName := make(map[string]corev1.ResourceList)
	if metrics != nil {
		for _, m := range metrics.Containers {
			metricsByName[m.Name] = m.Usage
		}
	}

	podInfo.Containers = make([]ContainerMemoryInfo, 0, len(pod.Spec.Containers))
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		usage := metricsByName[container.Name]
		cm, req, lim, hasReq, hasLim := c.processContainerMemoryInfo(container, usage)
		totalRequest += req
		totalLimit += lim
		hasRequest = hasRequest && hasReq
		hasLimit = hasLimit && hasLim
		podInfo.Containers = append(podInfo.Containers, cm)
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
