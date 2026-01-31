// Package services provides business logic for API resources.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/scaling/kubernetes"
	"github.com/janovincze/philotes/internal/scaling/nodepool"
)

// NodePoolService provides business logic for node pool operations.
type NodePoolService struct {
	manager *nodepool.Manager
	monitor *kubernetes.Monitor
	drainer *kubernetes.Drainer
	logger  *slog.Logger
}

// NewNodePoolService creates a new NodePoolService.
func NewNodePoolService(
	manager *nodepool.Manager,
	monitor *kubernetes.Monitor,
	drainer *kubernetes.Drainer,
	logger *slog.Logger,
) *NodePoolService {
	if logger == nil {
		logger = slog.Default()
	}
	return &NodePoolService{
		manager: manager,
		monitor: monitor,
		drainer: drainer,
		logger:  logger.With("component", "nodepool-service"),
	}
}

// CreatePool creates a new node pool.
func (s *NodePoolService) CreatePool(ctx context.Context, req *models.CreateNodePoolRequest) (*nodepool.NodePool, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Convert to domain model
	pool := req.ToNodePool()

	// Create pool via manager
	created, err := s.manager.CreatePool(ctx, pool)
	if err != nil {
		s.logger.Error("failed to create node pool", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to create node pool: %w", err)
	}

	s.logger.Info("node pool created", "id", created.ID, "name", created.Name)
	return created, nil
}

// GetPool retrieves a node pool by ID with its nodes.
func (s *NodePoolService) GetPool(ctx context.Context, id uuid.UUID) (*nodepool.NodePool, []nodepool.Node, error) {
	pool, nodes, err := s.manager.GetPool(ctx, id)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, nil, &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		return nil, nil, fmt.Errorf("failed to get node pool: %w", err)
	}
	return pool, nodes, nil
}

// ListPools lists all node pools.
func (s *NodePoolService) ListPools(ctx context.Context, enabledOnly bool) ([]nodepool.NodePool, error) {
	pools, err := s.manager.ListPools(ctx, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list node pools: %w", err)
	}
	if pools == nil {
		pools = []nodepool.NodePool{}
	}
	return pools, nil
}

// UpdatePool updates a node pool.
func (s *NodePoolService) UpdatePool(ctx context.Context, id uuid.UUID, req *models.UpdateNodePoolRequest) (*nodepool.NodePool, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Get existing pool
	pool, _, err := s.manager.GetPool(ctx, id)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get node pool: %w", err)
	}

	// Apply updates
	req.ApplyToPool(pool)

	// Update via manager
	if err := s.manager.UpdatePool(ctx, pool); err != nil {
		s.logger.Error("failed to update node pool", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update node pool: %w", err)
	}

	s.logger.Info("node pool updated", "id", id, "name", pool.Name)
	return pool, nil
}

// DeletePool deletes a node pool.
func (s *NodePoolService) DeletePool(ctx context.Context, id uuid.UUID) error {
	if err := s.manager.DeletePool(ctx, id); err != nil {
		if err == nodepool.ErrNotFound {
			return &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		s.logger.Error("failed to delete node pool", "error", err, "id", id)
		return fmt.Errorf("failed to delete node pool: %w", err)
	}

	s.logger.Info("node pool deleted", "id", id)
	return nil
}

// EnablePool enables a node pool.
func (s *NodePoolService) EnablePool(ctx context.Context, id uuid.UUID) error {
	if err := s.manager.EnablePool(ctx, id); err != nil {
		if err == nodepool.ErrNotFound {
			return &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		return fmt.Errorf("failed to enable node pool: %w", err)
	}
	return nil
}

// DisablePool disables a node pool.
func (s *NodePoolService) DisablePool(ctx context.Context, id uuid.UUID) error {
	if err := s.manager.DisablePool(ctx, id); err != nil {
		if err == nodepool.ErrNotFound {
			return &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		return fmt.Errorf("failed to disable node pool: %w", err)
	}
	return nil
}

// GetPoolStatus gets the status of a node pool.
func (s *NodePoolService) GetPoolStatus(ctx context.Context, id uuid.UUID) (*nodepool.NodePoolStatus, error) {
	status, err := s.manager.GetPoolStatus(ctx, id)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, &NotFoundError{Resource: "node pool", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get node pool status: %w", err)
	}
	return status, nil
}

// GetAllPoolStatuses gets status for all node pools.
func (s *NodePoolService) GetAllPoolStatuses(ctx context.Context) ([]nodepool.NodePoolStatus, error) {
	statuses, err := s.manager.GetAllPoolStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool statuses: %w", err)
	}
	if statuses == nil {
		statuses = []nodepool.NodePoolStatus{}
	}
	return statuses, nil
}

// ListNodes lists nodes in a pool.
func (s *NodePoolService) ListNodes(ctx context.Context, poolID uuid.UUID, activeOnly bool) ([]nodepool.Node, error) {
	nodes, err := s.manager.ListNodes(ctx, poolID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	if nodes == nil {
		nodes = []nodepool.Node{}
	}
	return nodes, nil
}

// GetNode retrieves a node by ID.
func (s *NodePoolService) GetNode(ctx context.Context, nodeID uuid.UUID) (*nodepool.Node, error) {
	node, err := s.manager.GetNode(ctx, nodeID)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, &NotFoundError{Resource: "node", ID: nodeID.String()}
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return node, nil
}

// DrainNode drains a node.
func (s *NodePoolService) DrainNode(ctx context.Context, nodeID uuid.UUID, req *models.DrainNodeRequest) error {
	// Get node to find its K8s name
	node, err := s.manager.GetNode(ctx, nodeID)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return &NotFoundError{Resource: "node", ID: nodeID.String()}
		}
		return fmt.Errorf("failed to get node: %w", err)
	}

	if node.NodeName == "" {
		return fmt.Errorf("node %s has not joined the cluster yet", nodeID)
	}

	if s.drainer == nil {
		return fmt.Errorf("node drainer not configured")
	}

	// Build drain options
	opts := kubernetes.DrainOptions{
		Force:              req.Force,
		DeleteEmptyDirData: true,
		IgnoreDaemonSets:   true,
	}
	if req.GracePeriodSeconds != nil {
		opts.GracePeriodSeconds = int64(*req.GracePeriodSeconds)
	}
	if req.TimeoutSeconds != nil {
		opts.Timeout = time.Duration(*req.TimeoutSeconds) * time.Second
	}

	// Cordon first
	if err := s.drainer.CordonNode(ctx, node.NodeName); err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// Then drain
	if err := s.drainer.DrainNode(ctx, node.NodeName, opts); err != nil {
		return fmt.Errorf("failed to drain node: %w", err)
	}

	s.logger.Info("node drained", "node_id", nodeID, "node_name", node.NodeName)
	return nil
}

// ListOperations lists scaling operations for a pool.
func (s *NodePoolService) ListOperations(ctx context.Context, poolID uuid.UUID, limit int) ([]nodepool.ScalingOperation, error) {
	ops, err := s.manager.ListOperations(ctx, poolID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	if ops == nil {
		ops = []nodepool.ScalingOperation{}
	}
	return ops, nil
}

// GetOperation retrieves a scaling operation.
func (s *NodePoolService) GetOperation(ctx context.Context, operationID uuid.UUID) (*nodepool.ScalingOperation, error) {
	op, err := s.manager.GetOperation(ctx, operationID)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, &NotFoundError{Resource: "operation", ID: operationID.String()}
		}
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}
	return op, nil
}

// CancelOperation cancels a pending scaling operation.
func (s *NodePoolService) CancelOperation(ctx context.Context, operationID uuid.UUID) error {
	if err := s.manager.CancelOperation(ctx, operationID); err != nil {
		if err == nodepool.ErrNotFound {
			return &NotFoundError{Resource: "operation", ID: operationID.String()}
		}
		return fmt.Errorf("failed to cancel operation: %w", err)
	}
	return nil
}

// GetClusterCapacity returns cluster capacity information.
func (s *NodePoolService) GetClusterCapacity(ctx context.Context) (*models.ClusterCapacityResponse, error) {
	if s.monitor == nil {
		return nil, fmt.Errorf("kubernetes monitor not configured")
	}

	// Get pool statuses for node pool breakdown
	statuses, err := s.manager.GetAllPoolStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool statuses: %w", err)
	}

	// Calculate totals from pool statuses
	var totalNodes, readyNodes int
	var totalHourlyCost float64
	for _, status := range statuses {
		totalNodes += status.CurrentNodes
		readyNodes += status.ReadyNodes
		totalHourlyCost += status.HourlyCost
	}

	// Get utilization data from monitor
	utilization, err := s.monitor.GetAllNodeUtilization(ctx)
	if err != nil {
		s.logger.Warn("failed to get node utilization", "error", err)
		// Continue without utilization data
	}

	var totalCPU, totalMem, allocCPU, allocMem int64
	for _, u := range utilization {
		totalCPU += u.CPUAllocatable
		totalMem += u.MemoryAllocatable
		allocCPU += u.CPURequested
		allocMem += u.MemoryRequested
	}

	// Get pending pods
	pendingSummary, err := s.monitor.GetPendingPodsSummary(ctx)
	if err != nil {
		s.logger.Warn("failed to get pending pods", "error", err)
	}

	var pendingPods, unschedulable int
	if pendingSummary != nil {
		pendingPods = pendingSummary.TotalPending
		unschedulable = pendingSummary.Unschedulable
	}

	// Calculate utilization percentages
	var cpuPercent, memPercent float64
	if totalCPU > 0 {
		cpuPercent = float64(allocCPU) / float64(totalCPU) * 100
	}
	if totalMem > 0 {
		memPercent = float64(allocMem) / float64(totalMem) * 100
	}

	return &models.ClusterCapacityResponse{
		TotalNodes:          totalNodes,
		ReadyNodes:          readyNodes,
		TotalCPUCores:       totalCPU / 1000, // Convert millicores to cores
		TotalMemoryMB:       totalMem,
		AllocatableCPU:      allocCPU / 1000,
		AllocatableMemory:   allocMem,
		UsedCPUPercent:      cpuPercent,
		UsedMemoryPercent:   memPercent,
		PendingPods:         pendingPods,
		UnschedulablePods:   unschedulable,
		EstimatedHourlyCost: totalHourlyCost,
		NodePools:           statuses,
	}, nil
}

// GetPendingPods returns pending pods summary.
func (s *NodePoolService) GetPendingPods(ctx context.Context) (*models.PendingPodsResponse, error) {
	if s.monitor == nil {
		return nil, fmt.Errorf("kubernetes monitor not configured")
	}

	summary, err := s.monitor.GetPendingPodsSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending pods: %w", err)
	}

	return &models.PendingPodsResponse{
		TotalPending:        summary.TotalPending,
		Unschedulable:       summary.Unschedulable,
		WaitingForResources: summary.WaitingForResources,
		OldestPending:       summary.OldestPending,
		ByReason:            summary.ByReason,
		CPURequested:        summary.ResourceRequests.CPUMillicores,
		MemoryRequested:     summary.ResourceRequests.MemoryMB,
	}, nil
}

// GetPricing retrieves pricing for an instance type.
func (s *NodePoolService) GetPricing(ctx context.Context, provider nodepool.Provider, instanceType, region string) (*nodepool.InstanceTypePricing, error) {
	pricing, err := s.manager.GetPricing(ctx, provider, instanceType, region)
	if err != nil {
		if err == nodepool.ErrNotFound {
			return nil, &NotFoundError{Resource: "pricing", ID: fmt.Sprintf("%s/%s/%s", provider, region, instanceType)}
		}
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}
	return pricing, nil
}
