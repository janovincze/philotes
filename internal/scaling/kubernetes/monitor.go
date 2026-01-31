// Package kubernetes provides Kubernetes client wrapper for node operations.
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Monitor provides cluster monitoring capabilities.
type Monitor struct {
	client *kubernetes.Clientset
	logger *slog.Logger
}

// NewMonitor creates a new cluster monitor.
func NewMonitor(client *kubernetes.Clientset, logger *slog.Logger) *Monitor {
	return &Monitor{
		client: client,
		logger: logger.With("component", "monitor"),
	}
}

// PendingPodsSummary summarizes pending pods in the cluster.
type PendingPodsSummary struct {
	TotalPending     int                     `json:"total_pending"`
	Unschedulable    int                     `json:"unschedulable"`
	WaitingForResources int                  `json:"waiting_for_resources"`
	ResourceRequests ResourceRequests        `json:"resource_requests"`
	OldestPending    *time.Time              `json:"oldest_pending,omitempty"`
	ByReason         map[string]int          `json:"by_reason"`
	TopPods          []PendingPodInfo        `json:"top_pods,omitempty"`
}

// PendingPodInfo contains information about a pending pod.
type PendingPodInfo struct {
	Name        string           `json:"name"`
	Namespace   string           `json:"namespace"`
	Age         time.Duration    `json:"age"`
	Reason      string           `json:"reason,omitempty"`
	Message     string           `json:"message,omitempty"`
	CPURequest  resource.Quantity `json:"cpu_request"`
	MemRequest  resource.Quantity `json:"memory_request"`
}

// ResourceRequests represents total resource requests from pending pods.
type ResourceRequests struct {
	CPUMillicores int64 `json:"cpu_millicores"`
	MemoryMB      int64 `json:"memory_mb"`
	Pods          int   `json:"pods"`
}

// GetPendingPodsSummary returns a summary of pending pods.
func (m *Monitor) GetPendingPodsSummary(ctx context.Context) (*PendingPodsSummary, error) {
	pods, err := m.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending pods: %w", err)
	}

	summary := &PendingPodsSummary{
		ByReason: make(map[string]int),
	}

	var pendingPods []PendingPodInfo

	for _, pod := range pods.Items {
		summary.TotalPending++

		// Calculate resource requests
		for _, container := range pod.Spec.Containers {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				summary.ResourceRequests.CPUMillicores += cpu.MilliValue()
			}
			if mem := container.Resources.Requests.Memory(); mem != nil {
				summary.ResourceRequests.MemoryMB += mem.Value() / (1024 * 1024)
			}
		}
		summary.ResourceRequests.Pods++

		// Check for unschedulable reason
		reason := "Unknown"
		message := ""
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled {
				if condition.Status == corev1.ConditionFalse {
					reason = condition.Reason
					message = condition.Message
					if condition.Reason == corev1.PodReasonUnschedulable {
						summary.Unschedulable++
						if isResourceRelated(condition.Message) {
							summary.WaitingForResources++
						}
					}
				}
				break
			}
		}

		summary.ByReason[reason]++

		// Track oldest pending
		creationTime := pod.CreationTimestamp.Time
		if summary.OldestPending == nil || creationTime.Before(*summary.OldestPending) {
			summary.OldestPending = &creationTime
		}

		// Collect pod info
		var cpuReq, memReq resource.Quantity
		for _, container := range pod.Spec.Containers {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				cpuReq.Add(*cpu)
			}
			if mem := container.Resources.Requests.Memory(); mem != nil {
				memReq.Add(*mem)
			}
		}

		pendingPods = append(pendingPods, PendingPodInfo{
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			Age:        time.Since(creationTime),
			Reason:     reason,
			Message:    message,
			CPURequest: cpuReq,
			MemRequest: memReq,
		})
	}

	// Sort by age (oldest first) and take top 10
	sort.Slice(pendingPods, func(i, j int) bool {
		return pendingPods[i].Age > pendingPods[j].Age
	})
	if len(pendingPods) > 10 {
		pendingPods = pendingPods[:10]
	}
	summary.TopPods = pendingPods

	return summary, nil
}

// NodeUtilization represents resource utilization for a node.
type NodeUtilization struct {
	Name              string  `json:"name"`
	CPURequested      int64   `json:"cpu_requested_millicores"`
	CPUAllocatable    int64   `json:"cpu_allocatable_millicores"`
	CPUUtilization    float64 `json:"cpu_utilization_percent"`
	MemoryRequested   int64   `json:"memory_requested_mb"`
	MemoryAllocatable int64   `json:"memory_allocatable_mb"`
	MemoryUtilization float64 `json:"memory_utilization_percent"`
	PodCount          int     `json:"pod_count"`
	PodCapacity       int64   `json:"pod_capacity"`
	IsReady           bool    `json:"is_ready"`
	IsSchedulable     bool    `json:"is_schedulable"`
}

// GetNodeUtilization returns resource utilization for a specific node.
func (m *Monitor) GetNodeUtilization(ctx context.Context, nodeName string) (*NodeUtilization, error) {
	node, err := m.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	pods, err := m.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node: %w", err)
	}

	var cpuRequested, memRequested int64
	podCount := 0

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodPending {
			continue
		}
		podCount++

		for _, container := range pod.Spec.Containers {
			if cpu := container.Resources.Requests.Cpu(); cpu != nil {
				cpuRequested += cpu.MilliValue()
			}
			if mem := container.Resources.Requests.Memory(); mem != nil {
				memRequested += mem.Value() / (1024 * 1024)
			}
		}
	}

	cpuAllocatable := node.Status.Allocatable.Cpu().MilliValue()
	memAllocatable := node.Status.Allocatable.Memory().Value() / (1024 * 1024)
	podCapacity := node.Status.Allocatable.Pods().Value()

	isReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			isReady = condition.Status == corev1.ConditionTrue
			break
		}
	}

	util := &NodeUtilization{
		Name:              nodeName,
		CPURequested:      cpuRequested,
		CPUAllocatable:    cpuAllocatable,
		MemoryRequested:   memRequested,
		MemoryAllocatable: memAllocatable,
		PodCount:          podCount,
		PodCapacity:       podCapacity,
		IsReady:           isReady,
		IsSchedulable:     !node.Spec.Unschedulable,
	}

	if cpuAllocatable > 0 {
		util.CPUUtilization = float64(cpuRequested) / float64(cpuAllocatable) * 100
	}
	if memAllocatable > 0 {
		util.MemoryUtilization = float64(memRequested) / float64(memAllocatable) * 100
	}

	return util, nil
}

// GetAllNodeUtilization returns utilization for all nodes.
func (m *Monitor) GetAllNodeUtilization(ctx context.Context) ([]NodeUtilization, error) {
	nodes, err := m.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var result []NodeUtilization
	for _, node := range nodes.Items {
		util, utilErr := m.GetNodeUtilization(ctx, node.Name)
		if utilErr != nil {
			m.logger.Warn("failed to get utilization for node", "node", node.Name, "error", utilErr)
			continue
		}
		result = append(result, *util)
	}

	return result, nil
}

// NodeSelectionCriteria defines criteria for selecting nodes for scale-down.
type NodeSelectionCriteria struct {
	// PreferEmpty prioritizes nodes with no workload pods
	PreferEmpty bool

	// PreferLowUtilization prioritizes nodes with lower utilization
	PreferLowUtilization bool

	// PreferNewest prioritizes newer nodes
	PreferNewest bool

	// MaxUtilization is the maximum utilization threshold for scale-down candidates
	MaxUtilization float64

	// MinAge is the minimum age for a node to be considered
	MinAge time.Duration

	// ExcludeLabels are labels that exclude a node from selection
	ExcludeLabels map[string]string

	// RequireLabels are labels that must be present for selection
	RequireLabels map[string]string
}

// DefaultSelectionCriteria returns default node selection criteria.
func DefaultSelectionCriteria() NodeSelectionCriteria {
	return NodeSelectionCriteria{
		PreferEmpty:          true,
		PreferLowUtilization: true,
		PreferNewest:         true,
		MaxUtilization:       50.0,
		MinAge:               10 * time.Minute,
		ExcludeLabels:        make(map[string]string),
		RequireLabels:        make(map[string]string),
	}
}

// SelectNodesForScaleDown selects nodes that are candidates for removal.
func (m *Monitor) SelectNodesForScaleDown(ctx context.Context, poolLabels map[string]string, criteria NodeSelectionCriteria, count int) ([]string, error) {
	// Get nodes matching pool labels
	selector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: poolLabels,
	})

	nodes, err := m.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	type nodeScore struct {
		name  string
		score float64
		util  *NodeUtilization
	}

	var candidates []nodeScore

	for _, node := range nodes.Items {
		// Check exclusion labels
		excluded := false
		for k, v := range criteria.ExcludeLabels {
			if node.Labels[k] == v {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check required labels
		hasRequired := true
		for k, v := range criteria.RequireLabels {
			if node.Labels[k] != v {
				hasRequired = false
				break
			}
		}
		if !hasRequired {
			continue
		}

		// Check minimum age
		if time.Since(node.CreationTimestamp.Time) < criteria.MinAge {
			continue
		}

		// Get utilization
		util, utilErr := m.GetNodeUtilization(ctx, node.Name)
		if utilErr != nil {
			continue
		}

		// Check max utilization threshold
		if util.CPUUtilization > criteria.MaxUtilization || util.MemoryUtilization > criteria.MaxUtilization {
			continue
		}

		// Skip nodes that aren't ready or schedulable
		if !util.IsReady || !util.IsSchedulable {
			// Actually, unschedulable nodes are good candidates
			if !util.IsReady {
				continue
			}
		}

		// Calculate score (lower is better for removal)
		score := 0.0

		if criteria.PreferEmpty && util.PodCount == 0 {
			score -= 1000 // Strong preference for empty nodes
		}

		if criteria.PreferLowUtilization {
			score += util.CPUUtilization + util.MemoryUtilization
		}

		if criteria.PreferNewest {
			age := time.Since(node.CreationTimestamp.Time)
			score += age.Hours() // Newer nodes have lower score
		}

		candidates = append(candidates, nodeScore{
			name:  node.Name,
			score: score,
			util:  util,
		})
	}

	// Sort by score (ascending, lower is better candidate)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})

	// Return top N candidates
	result := make([]string, 0, count)
	for i := 0; i < count && i < len(candidates); i++ {
		result = append(result, candidates[i].name)
		m.logger.Debug("selected node for scale-down",
			"node", candidates[i].name,
			"score", candidates[i].score,
			"cpu_util", candidates[i].util.CPUUtilization,
			"mem_util", candidates[i].util.MemoryUtilization,
			"pods", candidates[i].util.PodCount,
		)
	}

	return result, nil
}

// isResourceRelated checks if an unschedulable message is resource-related.
func isResourceRelated(message string) bool {
	resourceKeywords := []string{
		"Insufficient cpu",
		"Insufficient memory",
		"Insufficient pods",
		"didn't fit",
		"insufficient resources",
		"nodes are available",
	}

	for _, kw := range resourceKeywords {
		if containsIgnoreCase(message, kw) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
