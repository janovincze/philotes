// Package kubernetes provides Kubernetes client wrapper for node operations.
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// DrainOptions configures node drain behavior.
type DrainOptions struct {
	// GracePeriodSeconds is the grace period for pod termination
	GracePeriodSeconds int64

	// Timeout is the maximum time to wait for drain to complete
	Timeout time.Duration

	// DeleteEmptyDirData allows deletion of pods using emptyDir volumes
	DeleteEmptyDirData bool

	// IgnoreDaemonSets allows draining nodes with DaemonSet pods
	IgnoreDaemonSets bool

	// Force continues even if there are pods not managed by a controller
	Force bool

	// SkipWaitForDeleteTimeoutSeconds is the time to skip waiting for pods
	SkipWaitForDeleteTimeoutSeconds int
}

// DefaultDrainOptions returns sensible defaults for draining.
func DefaultDrainOptions() DrainOptions {
	return DrainOptions{
		GracePeriodSeconds:              30,
		Timeout:                         5 * time.Minute,
		DeleteEmptyDirData:              true,
		IgnoreDaemonSets:                true,
		Force:                           false,
		SkipWaitForDeleteTimeoutSeconds: 10,
	}
}

// Drainer handles node draining operations.
type Drainer struct {
	client *kubernetes.Clientset
	logger *slog.Logger
}

// NewDrainer creates a new node drainer.
func NewDrainer(client *kubernetes.Clientset, logger *slog.Logger) *Drainer {
	return &Drainer{
		client: client,
		logger: logger.With("component", "drainer"),
	}
}

// CordonNode marks a node as unschedulable.
func (d *Drainer) CordonNode(ctx context.Context, nodeName string) error {
	d.logger.Info("cordoning node", "node", nodeName)

	node, err := d.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if node.Spec.Unschedulable {
		d.logger.Debug("node already cordoned", "node", nodeName)
		return nil
	}

	node.Spec.Unschedulable = true
	_, err = d.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	d.logger.Info("node cordoned", "node", nodeName)
	return nil
}

// UncordonNode marks a node as schedulable.
func (d *Drainer) UncordonNode(ctx context.Context, nodeName string) error {
	d.logger.Info("uncordoning node", "node", nodeName)

	node, err := d.client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if !node.Spec.Unschedulable {
		d.logger.Debug("node already uncordoned", "node", nodeName)
		return nil
	}

	node.Spec.Unschedulable = false
	_, err = d.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to uncordon node: %w", err)
	}

	d.logger.Info("node uncordoned", "node", nodeName)
	return nil
}

// DrainNode drains all pods from a node.
func (d *Drainer) DrainNode(ctx context.Context, nodeName string, opts DrainOptions) error {
	d.logger.Info("draining node", "node", nodeName, "timeout", opts.Timeout)

	// First, cordon the node
	if err := d.CordonNode(ctx, nodeName); err != nil {
		return err
	}

	// Get pods on the node
	pods, err := d.getPodsForDrain(ctx, nodeName, opts)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		d.logger.Info("no pods to drain", "node", nodeName)
		return nil
	}

	d.logger.Info("draining pods", "node", nodeName, "count", len(pods))

	// Create a context with timeout
	drainCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Evict pods
	for i := range pods {
		if err := d.evictPod(drainCtx, &pods[i], opts); err != nil {
			if !opts.Force {
				return fmt.Errorf("failed to evict pod %s/%s: %w", pods[i].Namespace, pods[i].Name, err)
			}
			d.logger.Warn("failed to evict pod, continuing due to force flag",
				"pod", pods[i].Name, "namespace", pods[i].Namespace, "error", err)
		}
	}

	// Wait for pods to be deleted
	if err := d.waitForPodsDeleted(drainCtx, nodeName, pods, opts); err != nil {
		if !opts.Force {
			return fmt.Errorf("timed out waiting for pods to be deleted: %w", err)
		}
		d.logger.Warn("timed out waiting for pods, continuing due to force flag", "error", err)
	}

	d.logger.Info("node drained successfully", "node", nodeName)
	return nil
}

// getPodsForDrain returns pods that need to be drained from a node.
func (d *Drainer) getPodsForDrain(ctx context.Context, nodeName string, opts DrainOptions) ([]corev1.Pod, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{
		"spec.nodeName": nodeName,
	})

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node: %w", err)
	}

	filteredPods := make([]corev1.Pod, 0, len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		// Skip pods that are already terminating
		if pod.DeletionTimestamp != nil {
			continue
		}

		// Skip mirror pods (created by kubelet)
		if _, ok := pod.Annotations["kubernetes.io/config.mirror"]; ok {
			continue
		}

		// Skip DaemonSet pods if configured
		if opts.IgnoreDaemonSets && isDaemonSetPod(pod) {
			d.logger.Debug("skipping DaemonSet pod", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}

		// Check for pods using emptyDir
		if !opts.DeleteEmptyDirData && hasEmptyDir(pod) {
			if !opts.Force {
				return nil, fmt.Errorf("pod %s/%s uses emptyDir volume and DeleteEmptyDirData is false",
					pod.Namespace, pod.Name)
			}
			d.logger.Warn("pod uses emptyDir, data will be lost",
				"pod", pod.Name, "namespace", pod.Namespace)
		}

		// Check for unmanaged pods
		if !opts.Force && !isManagedPod(pod) {
			return nil, fmt.Errorf("pod %s/%s is not managed by a controller",
				pod.Namespace, pod.Name)
		}

		filteredPods = append(filteredPods, pods.Items[i])
	}

	return filteredPods, nil
}

// evictPod evicts a single pod using the Eviction API.
func (d *Drainer) evictPod(ctx context.Context, pod *corev1.Pod, opts DrainOptions) error {
	d.logger.Debug("evicting pod", "pod", pod.Name, "namespace", pod.Namespace)

	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: &opts.GracePeriodSeconds,
		},
	}

	err := d.client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction)
	if err != nil {
		if apierrors.IsTooManyRequests(err) {
			// PDB prevents eviction, wait and retry
			d.logger.Debug("PDB preventing eviction, will retry", "pod", pod.Name)
			return err
		}
		if apierrors.IsNotFound(err) {
			// Pod already gone
			return nil
		}
		return err
	}

	return nil
}

// waitForPodsDeleted waits for all drained pods to be deleted.
func (d *Drainer) waitForPodsDeleted(ctx context.Context, nodeName string, pods []corev1.Pod, opts DrainOptions) error {
	podNames := make(map[string]string) // namespace/name -> ""
	for i := range pods {
		podNames[fmt.Sprintf("%s/%s", pods[i].Namespace, pods[i].Name)] = ""
	}

	return wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		// Check if pods still exist
		remaining := 0
		for key := range podNames {
			namespace, name, found := strings.Cut(key, "/")
			if !found {
				delete(podNames, key)
				continue
			}

			_, err := d.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					delete(podNames, key)
					continue
				}
				d.logger.Warn("error checking pod status", "pod", key, "error", err)
			}
			remaining++
		}

		if remaining == 0 {
			return true, nil
		}

		d.logger.Debug("waiting for pods to be deleted", "remaining", remaining)
		return false, nil
	})
}

// isDaemonSetPod checks if a pod is owned by a DaemonSet.
func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// isManagedPod checks if a pod is managed by a controller.
func isManagedPod(pod *corev1.Pod) bool {
	return len(pod.OwnerReferences) > 0
}

// hasEmptyDir checks if a pod uses emptyDir volumes.
func hasEmptyDir(pod *corev1.Pod) bool {
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].EmptyDir != nil {
			return true
		}
	}
	return false
}

// CountPodsOnNode returns the number of non-DaemonSet pods on a node.
func (d *Drainer) CountPodsOnNode(ctx context.Context, nodeName string) (int, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{
		"spec.nodeName": nodeName,
	})

	pods, err := d.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list pods on node: %w", err)
	}

	count := 0
	for i := range pods.Items {
		// Skip DaemonSet pods
		if isDaemonSetPod(&pods.Items[i]) {
			continue
		}
		// Skip terminated pods
		if pods.Items[i].DeletionTimestamp != nil {
			continue
		}
		count++
	}

	return count, nil
}

// IsNodeEmpty returns true if the node has no workload pods.
func (d *Drainer) IsNodeEmpty(ctx context.Context, nodeName string) (bool, error) {
	count, err := d.CountPodsOnNode(ctx, nodeName)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}
