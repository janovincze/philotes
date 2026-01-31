// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
	"github.com/janovincze/philotes/internal/scaling/kubernetes"
	"github.com/janovincze/philotes/internal/scaling/nodepool"
)

// NodeExecutor implements the Executor interface for infrastructure node scaling.
type NodeExecutor struct {
	poolRepo     *nodepool.Repository
	providers    *cloudprovider.Registry
	k8sClient    *kubernetes.Client
	drainer      *kubernetes.Drainer
	monitor      *kubernetes.Monitor
	logger       *slog.Logger

	// Configuration
	nodeReadyTimeout time.Duration
	drainTimeout     time.Duration
	drainGracePeriod time.Duration

	// State tracking for concurrent operations
	pendingOps map[uuid.UUID]*PendingOperation
	mu         sync.Mutex
}

// PendingOperation tracks an in-flight scaling operation.
type PendingOperation struct {
	PoolID      uuid.UUID
	OperationID uuid.UUID
	Action      nodepool.OperationAction
	TargetCount int
	StartTime   time.Time
}

// NodeExecutorConfig holds configuration for the NodeExecutor.
type NodeExecutorConfig struct {
	NodeReadyTimeout time.Duration
	DrainTimeout     time.Duration
	DrainGracePeriod time.Duration
}

// DefaultNodeExecutorConfig returns default configuration.
func DefaultNodeExecutorConfig() NodeExecutorConfig {
	return NodeExecutorConfig{
		NodeReadyTimeout: 10 * time.Minute,
		DrainTimeout:     5 * time.Minute,
		DrainGracePeriod: 30 * time.Second,
	}
}

// NewNodeExecutor creates a new node executor.
func NewNodeExecutor(
	poolRepo *nodepool.Repository,
	providers *cloudprovider.Registry,
	k8sClient *kubernetes.Client,
	config NodeExecutorConfig,
	logger *slog.Logger,
) *NodeExecutor {
	if logger == nil {
		logger = slog.Default()
	}

	var drainer *kubernetes.Drainer
	var monitor *kubernetes.Monitor
	if k8sClient != nil {
		drainer = kubernetes.NewDrainer(k8sClient.Clientset(), logger)
		monitor = kubernetes.NewMonitor(k8sClient.Clientset(), logger)
	}

	return &NodeExecutor{
		poolRepo:         poolRepo,
		providers:        providers,
		k8sClient:        k8sClient,
		drainer:          drainer,
		monitor:          monitor,
		logger:           logger.With("component", "node-executor"),
		nodeReadyTimeout: config.NodeReadyTimeout,
		drainTimeout:     config.DrainTimeout,
		drainGracePeriod: config.DrainGracePeriod,
		pendingOps:       make(map[uuid.UUID]*PendingOperation),
	}
}

// Name returns the executor name.
func (e *NodeExecutor) Name() string {
	return "node"
}

// GetCurrentReplicas returns the current node count for a node pool.
// For node scaling, targetID should be the node pool ID.
func (e *NodeExecutor) GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error) {
	if targetType != TargetNodes {
		return 0, fmt.Errorf("node executor only handles TargetNodes, got %s", targetType)
	}

	if targetID == nil {
		// For infrastructure-level scaling without a specific pool,
		// return total active nodes across all pools
		return e.getTotalActiveNodes(ctx)
	}

	pool, err := e.poolRepo.GetPool(ctx, *targetID)
	if err != nil {
		return 0, fmt.Errorf("failed to get pool: %w", err)
	}

	// Get actual count from database
	count, err := e.poolRepo.CountActiveNodesForPool(ctx, *targetID)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	// Update pool's current_nodes if out of sync
	if count != pool.CurrentNodes {
		e.logger.Warn("node count mismatch, updating pool",
			"pool", pool.Name,
			"stored", pool.CurrentNodes,
			"actual", count,
		)
		_ = e.poolRepo.UpdatePoolNodeCount(ctx, *targetID, count) //nolint:errcheck // best-effort update
	}

	return count, nil
}

// Scale scales the node pool to the desired number of nodes.
func (e *NodeExecutor) Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error {
	if targetType != TargetNodes {
		return fmt.Errorf("node executor only handles TargetNodes, got %s", targetType)
	}

	if targetID == nil {
		return fmt.Errorf("targetID (node pool ID) is required for node scaling")
	}

	pool, err := e.poolRepo.GetPool(ctx, *targetID)
	if err != nil {
		return fmt.Errorf("failed to get pool: %w", err)
	}

	if !pool.Enabled {
		return fmt.Errorf("node pool %s is disabled", pool.Name)
	}

	// Clamp to pool limits
	targetNodes := pool.ClampNodeCount(replicas)

	currentNodes, err := e.GetCurrentReplicas(ctx, targetType, targetID)
	if err != nil {
		return fmt.Errorf("failed to get current node count: %w", err)
	}

	if currentNodes == targetNodes {
		e.logger.Debug("already at target node count", "pool", pool.Name, "count", currentNodes)
		return nil
	}

	// Determine action
	var action nodepool.OperationAction
	if targetNodes > currentNodes {
		action = nodepool.OperationActionScaleUp
	} else {
		action = nodepool.OperationActionScaleDown
	}

	e.logger.Info("scaling node pool",
		"pool", pool.Name,
		"provider", pool.Provider,
		"from", currentNodes,
		"to", targetNodes,
		"action", action,
		"dry_run", dryRun,
	)

	if dryRun {
		e.logger.Info("[DRY-RUN] would scale node pool",
			"pool", pool.Name,
			"from", currentNodes,
			"to", targetNodes,
		)
		return nil
	}

	// Create operation record
	op := &nodepool.ScalingOperation{
		PoolID:        pool.ID,
		Action:        action,
		PreviousCount: currentNodes,
		TargetCount:   targetNodes,
		Status:        nodepool.OperationStatusInProgress,
		Reason:        fmt.Sprintf("Scaling from %d to %d nodes", currentNodes, targetNodes),
		TriggeredBy:   "policy",
	}

	op, err = e.poolRepo.CreateOperation(ctx, op)
	if err != nil {
		return fmt.Errorf("failed to create operation record: %w", err)
	}

	// Track pending operation
	e.mu.Lock()
	e.pendingOps[op.ID] = &PendingOperation{
		PoolID:      pool.ID,
		OperationID: op.ID,
		Action:      action,
		TargetCount: targetNodes,
		StartTime:   time.Now(),
	}
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.pendingOps, op.ID)
		e.mu.Unlock()
	}()

	// Execute scaling
	var scalingErr error
	if action == nodepool.OperationActionScaleUp {
		scalingErr = e.scaleUp(ctx, pool, currentNodes, targetNodes, op)
	} else {
		scalingErr = e.scaleDown(ctx, pool, currentNodes, targetNodes, op)
	}

	// Update operation status
	finalCount, _ := e.GetCurrentReplicas(ctx, targetType, targetID)
	if scalingErr != nil {
		updateErr := e.poolRepo.UpdateOperationStatus(ctx, op.ID, nodepool.OperationStatusFailed, &finalCount, scalingErr.Error())
		if updateErr != nil {
			e.logger.Error("failed to update operation status", "error", updateErr)
		}
		return scalingErr
	}

	updateErr := e.poolRepo.UpdateOperationStatus(ctx, op.ID, nodepool.OperationStatusCompleted, &finalCount, "")
	if updateErr != nil {
		e.logger.Error("failed to update operation status", "error", updateErr)
	}

	return nil
}

// scaleUp adds nodes to the pool.
func (e *NodeExecutor) scaleUp(ctx context.Context, pool *nodepool.NodePool, currentCount, targetCount int, op *nodepool.ScalingOperation) error {
	provider, ok := e.providers.Get(pool.Provider.String())
	if !ok {
		return fmt.Errorf("provider %s not registered", pool.Provider)
	}

	nodesToAdd := targetCount - currentCount
	e.logger.Info("adding nodes to pool", "pool", pool.Name, "count", nodesToAdd)

	var createdNodeIDs []uuid.UUID

	for i := 0; i < nodesToAdd; i++ {
		nodeName := fmt.Sprintf("%s-node-%d", pool.Name, time.Now().UnixNano())

		// Create server via cloud provider
		server, err := provider.CreateServer(ctx, cloudprovider.CreateServerOptions{
			Name:         nodeName,
			Region:       pool.Region,
			InstanceType: pool.InstanceType,
			Image:        pool.Image,
			SSHKeyIDs:    []string{pool.SSHKeyID},
			UserData:     pool.UserDataTemplate,
			Labels:       pool.Labels,
			NetworkID:    pool.NetworkID,
			FirewallID:   pool.FirewallID,
		})
		if err != nil {
			e.logger.Error("failed to create server",
				"pool", pool.Name,
				"error", err,
			)
			continue
		}

		// Create node record
		node := &nodepool.Node{
			PoolID:       pool.ID,
			ProviderID:   server.ID,
			NodeName:     "", // Will be set when node joins cluster
			Status:       nodepool.NodeStatusCreating,
			PublicIP:     server.PublicIP,
			PrivateIP:    server.PrivateIP,
			InstanceType: pool.InstanceType,
		}

		node, err = e.poolRepo.CreateNode(ctx, node)
		if err != nil {
			e.logger.Error("failed to create node record",
				"server_id", server.ID,
				"error", err,
			)
			continue
		}

		createdNodeIDs = append(createdNodeIDs, node.ID)
		e.logger.Info("created node",
			"node_id", node.ID,
			"server_id", server.ID,
			"pool", pool.Name,
		)

		// Update node status to joining
		err = e.poolRepo.UpdateNodeStatus(ctx, node.ID, nodepool.NodeStatusJoining, "")
		if err != nil {
			e.logger.Warn("failed to update node status", "error", err)
		}
	}

	// Update operation with affected nodes
	err := e.poolRepo.UpdateOperationNodesAffected(ctx, op.ID, createdNodeIDs)
	if err != nil {
		e.logger.Warn("failed to update operation nodes", "error", err)
	}

	// Update pool node count
	newCount := currentCount + len(createdNodeIDs)
	err = e.poolRepo.UpdatePoolNodeCount(ctx, pool.ID, newCount)
	if err != nil {
		e.logger.Warn("failed to update pool node count", "error", err)
	}

	// Wait for nodes to be ready (async)
	go e.waitForNodesToBeReady(context.Background(), pool, createdNodeIDs)

	if len(createdNodeIDs) < nodesToAdd {
		return fmt.Errorf("only created %d of %d nodes", len(createdNodeIDs), nodesToAdd)
	}

	return nil
}

// scaleDown removes nodes from the pool.
func (e *NodeExecutor) scaleDown(ctx context.Context, pool *nodepool.NodePool, currentCount, targetCount int, op *nodepool.ScalingOperation) error {
	provider, ok := e.providers.Get(pool.Provider.String())
	if !ok {
		return fmt.Errorf("provider %s not registered", pool.Provider)
	}

	nodesToRemove := currentCount - targetCount
	e.logger.Info("removing nodes from pool", "pool", pool.Name, "count", nodesToRemove)

	// Select nodes for removal
	var nodeNames []string
	if e.monitor != nil {
		var selectErr error
		nodeNames, selectErr = e.monitor.SelectNodesForScaleDown(ctx, pool.Labels, kubernetes.DefaultSelectionCriteria(), nodesToRemove)
		if selectErr != nil {
			e.logger.Warn("failed to select nodes for scale-down, using database", "error", selectErr)
		}
	}

	// If we couldn't select nodes via K8s, fall back to database
	if len(nodeNames) == 0 {
		nodes, err := e.poolRepo.ListNodesForPool(ctx, pool.ID, true)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		// Select newest nodes first (they have less workload typically)
		for i := len(nodes) - 1; i >= 0 && len(nodeNames) < nodesToRemove; i-- {
			if nodes[i].Status == nodepool.NodeStatusReady {
				nodeNames = append(nodeNames, nodes[i].NodeName)
			}
		}
	}

	var removedNodeIDs []uuid.UUID

	for _, nodeName := range nodeNames {
		// Get node from database
		node, err := e.poolRepo.GetNodeByName(ctx, nodeName)
		if err != nil {
			e.logger.Warn("node not found in database", "name", nodeName, "error", err)
			continue
		}

		// Update status to draining
		err = e.poolRepo.UpdateNodeStatus(ctx, node.ID, nodepool.NodeStatusDraining, "")
		if err != nil {
			e.logger.Warn("failed to update node status", "error", err)
		}

		// Drain the node
		if e.drainer != nil && nodeName != "" {
			drainOpts := kubernetes.DefaultDrainOptions()
			drainOpts.Timeout = e.drainTimeout
			drainOpts.GracePeriodSeconds = int64(e.drainGracePeriod.Seconds())

			drainErr := e.drainer.DrainNode(ctx, nodeName, drainOpts)
			if drainErr != nil {
				e.logger.Warn("failed to drain node",
					"node", nodeName,
					"error", drainErr,
				)
				// Continue with deletion anyway after marking as failed
			}
		}

		// Update status to deleting
		err = e.poolRepo.UpdateNodeStatus(ctx, node.ID, nodepool.NodeStatusDeleting, "")
		if err != nil {
			e.logger.Warn("failed to update node status", "error", err)
		}

		// Delete server from cloud provider
		err = provider.DeleteServer(ctx, node.ProviderID)
		if err != nil {
			e.logger.Error("failed to delete server",
				"provider_id", node.ProviderID,
				"error", err,
			)
			errMsg := err.Error()
			updateErr := e.poolRepo.UpdateNodeStatus(ctx, node.ID, nodepool.NodeStatusFailed, errMsg)
			if updateErr != nil {
				e.logger.Warn("failed to update node status", "error", updateErr)
			}
			continue
		}

		// Delete node from Kubernetes
		if e.k8sClient != nil && nodeName != "" {
			deleteErr := e.k8sClient.DeleteNode(ctx, nodeName)
			if deleteErr != nil {
				e.logger.Warn("failed to delete K8s node", "node", nodeName, "error", deleteErr)
			}
		}

		// Mark node as deleted in database
		err = e.poolRepo.SoftDeleteNode(ctx, node.ID)
		if err != nil {
			e.logger.Warn("failed to soft delete node", "error", err)
		}

		removedNodeIDs = append(removedNodeIDs, node.ID)
		e.logger.Info("removed node",
			"node_id", node.ID,
			"provider_id", node.ProviderID,
			"pool", pool.Name,
		)
	}

	// Update operation with affected nodes
	err := e.poolRepo.UpdateOperationNodesAffected(ctx, op.ID, removedNodeIDs)
	if err != nil {
		e.logger.Warn("failed to update operation nodes", "error", err)
	}

	// Update pool node count
	newCount := currentCount - len(removedNodeIDs)
	if newCount < 0 {
		newCount = 0
	}
	err = e.poolRepo.UpdatePoolNodeCount(ctx, pool.ID, newCount)
	if err != nil {
		e.logger.Warn("failed to update pool node count", "error", err)
	}

	if len(removedNodeIDs) < nodesToRemove {
		return fmt.Errorf("only removed %d of %d nodes", len(removedNodeIDs), nodesToRemove)
	}

	return nil
}

// waitForNodesToBeReady waits for created nodes to join the cluster.
func (e *NodeExecutor) waitForNodesToBeReady(ctx context.Context, pool *nodepool.NodePool, nodeIDs []uuid.UUID) {
	if e.k8sClient == nil {
		return
	}

	deadline := time.Now().Add(e.nodeReadyTimeout)

	for _, nodeID := range nodeIDs {
		node, err := e.poolRepo.GetNode(ctx, nodeID)
		if err != nil {
			e.logger.Warn("failed to get node for ready check", "id", nodeID, "error", err)
			continue
		}

		nodeReady := false

		// Poll until node is ready or timeout
		for time.Now().Before(deadline) {
			// Try to find node in Kubernetes by IP
			nodes, err := e.k8sClient.ListNodes(ctx)
			if err != nil {
				e.logger.Debug("failed to list K8s nodes", "error", err)
				time.Sleep(10 * time.Second)
				continue
			}

			for _, k8sNode := range nodes {
				// Match by IP
				for _, addr := range k8sNode.Status.Addresses {
					if (addr.Type == "InternalIP" && addr.Address == node.PrivateIP) ||
						(addr.Type == "ExternalIP" && addr.Address == node.PublicIP) {

						// Found the node, check if ready
						isReady, readyErr := e.k8sClient.IsNodeReady(ctx, k8sNode.Name)
						if readyErr != nil {
							continue
						}

						// Update node name
						node.NodeName = k8sNode.Name
						updateErr := e.poolRepo.UpdateNode(ctx, node)
						if updateErr != nil {
							e.logger.Warn("failed to update node name", "error", updateErr)
						}

						if isReady {
							statusErr := e.poolRepo.UpdateNodeStatus(ctx, nodeID, nodepool.NodeStatusReady, "")
							if statusErr != nil {
								e.logger.Warn("failed to update node status to ready", "error", statusErr)
							}
							e.logger.Info("node is ready",
								"node_id", nodeID,
								"k8s_name", k8sNode.Name,
							)
							nodeReady = true
							break
						}
					}
				}
			}

			if nodeReady {
				break
			}
			time.Sleep(10 * time.Second)
		}

		// Timeout reached (only if not ready)
		if !nodeReady {
			e.logger.Warn("node did not become ready within timeout",
				"node_id", nodeID,
				"timeout", e.nodeReadyTimeout,
			)
			if failureErr := e.poolRepo.UpdateNodeStatus(ctx, nodeID, nodepool.NodeStatusFailed, "timeout waiting for node to become ready"); failureErr != nil {
				e.logger.Warn("failed to update node status", "error", failureErr)
			}
		}
	}
}

// getTotalActiveNodes returns the total active nodes across all pools.
func (e *NodeExecutor) getTotalActiveNodes(ctx context.Context) (int, error) {
	pools, err := e.poolRepo.ListPools(ctx, true)
	if err != nil {
		return 0, fmt.Errorf("failed to list pools: %w", err)
	}

	total := 0
	for _, pool := range pools {
		count, countErr := e.poolRepo.CountActiveNodesForPool(ctx, pool.ID)
		if countErr != nil {
			e.logger.Warn("failed to count nodes for pool", "pool", pool.Name, "error", countErr)
			continue
		}
		total += count
	}

	return total, nil
}
