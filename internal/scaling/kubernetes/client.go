// Package kubernetes provides Kubernetes client wrapper for node operations.
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes client for node operations.
type Client struct {
	clientset *kubernetes.Clientset
	logger    *slog.Logger
}

// Config holds configuration for the Kubernetes client.
type Config struct {
	// Kubeconfig path (optional, uses in-cluster config if empty)
	Kubeconfig string

	// InCluster forces in-cluster configuration
	InCluster bool
}

// NewClient creates a new Kubernetes client.
func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	var restConfig *rest.Config
	var err error

	switch {
	case cfg.InCluster:
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	case cfg.Kubeconfig != "":
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	default:
		// Try default kubeconfig location
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if _, statErr := os.Stat(kubeconfig); statErr == nil {
			restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		} else {
			// Fall back to in-cluster config
			restConfig, err = rest.InClusterConfig()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		logger:    logger.With("component", "k8s-client"),
	}, nil
}

// GetNode retrieves a Kubernetes node by name.
func (c *Client) GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}
	return node, nil
}

// ListNodes returns all nodes in the cluster.
func (c *Client) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	return nodes.Items, nil
}

// ListNodesByLabels returns nodes matching the given labels.
func (c *Client) ListNodesByLabels(ctx context.Context, labels map[string]string) ([]corev1.Node, error) {
	selector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes with labels: %w", err)
	}
	return nodes.Items, nil
}

// IsNodeReady checks if a node is in Ready condition.
func (c *Client) IsNodeReady(ctx context.Context, nodeName string) (bool, error) {
	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return false, err
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue, nil
		}
	}

	return false, nil
}

// GetNodeConditions returns all conditions for a node.
func (c *Client) GetNodeConditions(ctx context.Context, nodeName string) ([]corev1.NodeCondition, error) {
	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return nil, err
	}
	return node.Status.Conditions, nil
}

// SetNodeLabels updates the labels on a node.
func (c *Client) SetNodeLabels(ctx context.Context, nodeName string, labels map[string]string) error {
	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	for k, v := range labels {
		node.Labels[k] = v
	}

	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node labels: %w", err)
	}

	c.logger.Info("updated node labels", "node", nodeName, "labels", labels)
	return nil
}

// SetNodeTaints updates the taints on a node.
func (c *Client) SetNodeTaints(ctx context.Context, nodeName string, taints []corev1.Taint) error {
	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return err
	}

	node.Spec.Taints = taints

	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node taints: %w", err)
	}

	c.logger.Info("updated node taints", "node", nodeName, "taints", len(taints))
	return nil
}

// DeleteNode deletes a Kubernetes node.
func (c *Client) DeleteNode(ctx context.Context, nodeName string) error {
	err := c.clientset.CoreV1().Nodes().Delete(ctx, nodeName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete node %s: %w", nodeName, err)
	}

	c.logger.Info("deleted node", "node", nodeName)
	return nil
}

// GetPodsByNode returns all pods running on a specific node.
func (c *Client) GetPodsByNode(ctx context.Context, nodeName string) ([]corev1.Pod, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}
	return pods.Items, nil
}

// GetPendingPods returns all pods in Pending state.
func (c *Client) GetPendingPods(ctx context.Context) ([]corev1.Pod, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending pods: %w", err)
	}
	return pods.Items, nil
}

// GetUnschedulablePods returns pending pods that cannot be scheduled.
func (c *Client) GetUnschedulablePods(ctx context.Context) ([]corev1.Pod, error) {
	pendingPods, err := c.GetPendingPods(ctx)
	if err != nil {
		return nil, err
	}

	var unschedulable []corev1.Pod
	for i := range pendingPods {
		for j := range pendingPods[i].Status.Conditions {
			if pendingPods[i].Status.Conditions[j].Type == corev1.PodScheduled &&
				pendingPods[i].Status.Conditions[j].Status == corev1.ConditionFalse &&
				pendingPods[i].Status.Conditions[j].Reason == corev1.PodReasonUnschedulable {
				unschedulable = append(unschedulable, pendingPods[i])
				break
			}
		}
	}

	return unschedulable, nil
}

// NodeCapacity represents the capacity and allocatable resources of a node.
type NodeCapacity struct {
	Name              string
	CPUCores          int64
	MemoryMB          int64
	AllocatableCPU    int64
	AllocatableMemory int64
	Pods              int64
	IsReady           bool
	IsSchedulable     bool
}

// GetNodeCapacity returns the capacity of a node.
func (c *Client) GetNodeCapacity(ctx context.Context, nodeName string) (*NodeCapacity, error) {
	node, err := c.GetNode(ctx, nodeName)
	if err != nil {
		return nil, err
	}

	isReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			isReady = condition.Status == corev1.ConditionTrue
			break
		}
	}

	return &NodeCapacity{
		Name:              nodeName,
		CPUCores:          node.Status.Capacity.Cpu().MilliValue() / 1000,
		MemoryMB:          node.Status.Capacity.Memory().Value() / (1024 * 1024),
		AllocatableCPU:    node.Status.Allocatable.Cpu().MilliValue() / 1000,
		AllocatableMemory: node.Status.Allocatable.Memory().Value() / (1024 * 1024),
		Pods:              node.Status.Allocatable.Pods().Value(),
		IsReady:           isReady,
		IsSchedulable:     !node.Spec.Unschedulable,
	}, nil
}

// ClusterCapacity represents the total capacity of the cluster.
type ClusterCapacity struct {
	TotalNodes        int
	ReadyNodes        int
	SchedulableNodes  int
	TotalCPUCores     int64
	TotalMemoryMB     int64
	AllocatableCPU    int64
	AllocatableMemory int64
	TotalPods         int64
}

// GetClusterCapacity returns the total capacity of the cluster.
func (c *Client) GetClusterCapacity(ctx context.Context) (*ClusterCapacity, error) {
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	capacity := &ClusterCapacity{}

	for i := range nodes {
		capacity.TotalNodes++

		isReady := false
		for j := range nodes[i].Status.Conditions {
			if nodes[i].Status.Conditions[j].Type == corev1.NodeReady {
				isReady = nodes[i].Status.Conditions[j].Status == corev1.ConditionTrue
				break
			}
		}

		if isReady {
			capacity.ReadyNodes++
		}

		if !nodes[i].Spec.Unschedulable {
			capacity.SchedulableNodes++
		}

		capacity.TotalCPUCores += nodes[i].Status.Capacity.Cpu().MilliValue() / 1000
		capacity.TotalMemoryMB += nodes[i].Status.Capacity.Memory().Value() / (1024 * 1024)
		capacity.AllocatableCPU += nodes[i].Status.Allocatable.Cpu().MilliValue() / 1000
		capacity.AllocatableMemory += nodes[i].Status.Allocatable.Memory().Value() / (1024 * 1024)
		capacity.TotalPods += nodes[i].Status.Allocatable.Pods().Value()
	}

	return capacity, nil
}

// Clientset returns the underlying Kubernetes clientset.
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}
