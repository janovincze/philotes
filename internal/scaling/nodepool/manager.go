// Package nodepool provides node pool management for infrastructure auto-scaling.
package nodepool

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager provides high-level node pool management operations.
type Manager struct {
	repo   *Repository
	logger *slog.Logger

	// Pool-level locks to prevent concurrent operations on the same pool
	poolLocks map[uuid.UUID]*sync.Mutex
	locksMu   sync.Mutex
}

// NewManager creates a new node pool manager.
func NewManager(repo *Repository, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		repo:      repo,
		logger:    logger.With("component", "nodepool-manager"),
		poolLocks: make(map[uuid.UUID]*sync.Mutex),
	}
}

// getPoolLock returns a lock for the given pool ID.
func (m *Manager) getPoolLock(poolID uuid.UUID) *sync.Mutex {
	m.locksMu.Lock()
	defer m.locksMu.Unlock()

	if lock, ok := m.poolLocks[poolID]; ok {
		return lock
	}

	lock := &sync.Mutex{}
	m.poolLocks[poolID] = lock
	return lock
}

// CreatePool creates a new node pool.
func (m *Manager) CreatePool(ctx context.Context, pool *NodePool) (*NodePool, error) {
	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pool configuration: %w", err)
	}

	// Set defaults
	if pool.Image == "" {
		pool.Image = "ubuntu-24.04"
	}
	if pool.Labels == nil {
		pool.Labels = make(map[string]string)
	}
	if pool.Taints == nil {
		pool.Taints = []Taint{}
	}

	// Add standard labels
	pool.Labels["philotes.io/managed"] = "true"
	pool.Labels["philotes.io/pool"] = pool.Name

	created, err := m.repo.CreatePool(ctx, pool)
	if err != nil {
		return nil, err
	}

	m.logger.Info("created node pool",
		"pool_id", created.ID,
		"name", created.Name,
		"provider", created.Provider,
		"region", created.Region,
	)

	return created, nil
}

// GetPool retrieves a node pool by ID with its nodes.
func (m *Manager) GetPool(ctx context.Context, poolID uuid.UUID) (*NodePool, []Node, error) {
	pool, err := m.repo.GetPool(ctx, poolID)
	if err != nil {
		return nil, nil, err
	}

	nodes, err := m.repo.ListNodesForPool(ctx, poolID, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return pool, nodes, nil
}

// GetPoolByName retrieves a node pool by name.
func (m *Manager) GetPoolByName(ctx context.Context, name string) (*NodePool, error) {
	return m.repo.GetPoolByName(ctx, name)
}

// ListPools lists all node pools.
func (m *Manager) ListPools(ctx context.Context, enabledOnly bool) ([]NodePool, error) {
	return m.repo.ListPools(ctx, enabledOnly)
}

// UpdatePool updates a node pool configuration.
func (m *Manager) UpdatePool(ctx context.Context, pool *NodePool) error {
	if err := pool.Validate(); err != nil {
		return fmt.Errorf("invalid pool configuration: %w", err)
	}

	lock := m.getPoolLock(pool.ID)
	lock.Lock()
	defer lock.Unlock()

	// Get current pool to check constraints
	current, err := m.repo.GetPool(ctx, pool.ID)
	if err != nil {
		return err
	}

	// Validate that we're not reducing max below current
	if pool.MaxNodes < current.CurrentNodes {
		return fmt.Errorf("cannot set max_nodes (%d) below current node count (%d)", pool.MaxNodes, current.CurrentNodes)
	}

	// Ensure labels contain managed flag
	if pool.Labels == nil {
		pool.Labels = make(map[string]string)
	}
	pool.Labels["philotes.io/managed"] = "true"
	pool.Labels["philotes.io/pool"] = pool.Name

	if err := m.repo.UpdatePool(ctx, pool); err != nil {
		return err
	}

	m.logger.Info("updated node pool",
		"pool_id", pool.ID,
		"name", pool.Name,
	)

	return nil
}

// DeletePool deletes a node pool and all its nodes.
func (m *Manager) DeletePool(ctx context.Context, poolID uuid.UUID) error {
	lock := m.getPoolLock(poolID)
	lock.Lock()
	defer lock.Unlock()

	pool, err := m.repo.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	// Check if pool has nodes
	nodes, err := m.repo.ListNodesForPool(ctx, poolID, true)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	if len(nodes) > 0 {
		return fmt.Errorf("cannot delete pool with %d active nodes; scale down first", len(nodes))
	}

	if err := m.repo.DeletePool(ctx, poolID); err != nil {
		return err
	}

	m.logger.Info("deleted node pool",
		"pool_id", poolID,
		"name", pool.Name,
	)

	return nil
}

// EnablePool enables a node pool.
func (m *Manager) EnablePool(ctx context.Context, poolID uuid.UUID) error {
	pool, err := m.repo.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	pool.Enabled = true
	if err := m.repo.UpdatePool(ctx, pool); err != nil {
		return err
	}

	m.logger.Info("enabled node pool", "pool_id", poolID, "name", pool.Name)
	return nil
}

// DisablePool disables a node pool.
func (m *Manager) DisablePool(ctx context.Context, poolID uuid.UUID) error {
	pool, err := m.repo.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	pool.Enabled = false
	if err := m.repo.UpdatePool(ctx, pool); err != nil {
		return err
	}

	m.logger.Info("disabled node pool", "pool_id", poolID, "name", pool.Name)
	return nil
}

// ListNodes lists nodes in a pool.
func (m *Manager) ListNodes(ctx context.Context, poolID uuid.UUID, activeOnly bool) ([]Node, error) {
	return m.repo.ListNodesForPool(ctx, poolID, activeOnly)
}

// GetNode retrieves a node by ID.
func (m *Manager) GetNode(ctx context.Context, nodeID uuid.UUID) (*Node, error) {
	return m.repo.GetNode(ctx, nodeID)
}

// ListOperations lists scaling operations for a pool.
func (m *Manager) ListOperations(ctx context.Context, poolID uuid.UUID, limit int) ([]ScalingOperation, error) {
	return m.repo.ListOperationsForPool(ctx, poolID, limit)
}

// GetOperation retrieves a scaling operation.
func (m *Manager) GetOperation(ctx context.Context, operationID uuid.UUID) (*ScalingOperation, error) {
	return m.repo.GetOperation(ctx, operationID)
}

// CancelOperation cancels a pending scaling operation.
func (m *Manager) CancelOperation(ctx context.Context, operationID uuid.UUID) error {
	op, err := m.repo.GetOperation(ctx, operationID)
	if err != nil {
		return err
	}

	if op.Status != OperationStatusPending && op.Status != OperationStatusInProgress {
		return fmt.Errorf("cannot cancel operation in %s status", op.Status)
	}

	if err := m.repo.UpdateOperationStatus(ctx, operationID, OperationStatusCancelled, nil, "cancelled by user"); err != nil {
		return err
	}

	m.logger.Info("cancelled operation", "operation_id", operationID)
	return nil
}

// GetPoolStatus returns the status of a node pool.
func (m *Manager) GetPoolStatus(ctx context.Context, poolID uuid.UUID) (*NodePoolStatus, error) {
	pool, err := m.repo.GetPool(ctx, poolID)
	if err != nil {
		return nil, err
	}

	nodes, err := m.repo.ListNodesForPool(ctx, poolID, true)
	if err != nil {
		return nil, err
	}

	readyCount := 0
	var totalCost float64
	for _, node := range nodes {
		if node.Status == NodeStatusReady {
			readyCount++
		}
		if node.HourlyCost != nil {
			totalCost += *node.HourlyCost
		}
	}

	return &NodePoolStatus{
		ID:           pool.ID,
		Name:         pool.Name,
		Provider:     pool.Provider,
		Region:       pool.Region,
		InstanceType: pool.InstanceType,
		MinNodes:     pool.MinNodes,
		MaxNodes:     pool.MaxNodes,
		CurrentNodes: len(nodes),
		ReadyNodes:   readyCount,
		Enabled:      pool.Enabled,
		HourlyCost:   totalCost,
	}, nil
}

// GetAllPoolStatuses returns status for all node pools.
func (m *Manager) GetAllPoolStatuses(ctx context.Context) ([]NodePoolStatus, error) {
	pools, err := m.repo.ListPools(ctx, false)
	if err != nil {
		return nil, err
	}

	var statuses []NodePoolStatus
	for _, pool := range pools {
		status, statusErr := m.GetPoolStatus(ctx, pool.ID)
		if statusErr != nil {
			m.logger.Warn("failed to get pool status", "pool", pool.Name, "error", statusErr)
			continue
		}
		statuses = append(statuses, *status)
	}

	return statuses, nil
}

// GetPricing retrieves pricing for an instance type.
func (m *Manager) GetPricing(ctx context.Context, provider Provider, instanceType, region string) (*InstanceTypePricing, error) {
	return m.repo.GetPricing(ctx, provider, instanceType, region)
}

// EstimatePoolCost estimates the hourly cost for a node pool configuration.
func (m *Manager) EstimatePoolCost(ctx context.Context, provider Provider, instanceType, region string, nodeCount int) (float64, error) {
	pricing, err := m.repo.GetPricing(ctx, provider, instanceType, region)
	if err != nil {
		// If pricing not found, return 0
		if err == ErrNotFound {
			return 0, nil
		}
		return 0, err
	}

	return pricing.HourlyCost * float64(nodeCount), nil
}

// ReconcilePoolNodeCount updates the pool's current_nodes to match actual count.
func (m *Manager) ReconcilePoolNodeCount(ctx context.Context, poolID uuid.UUID) error {
	lock := m.getPoolLock(poolID)
	lock.Lock()
	defer lock.Unlock()

	actualCount, err := m.repo.CountActiveNodesForPool(ctx, poolID)
	if err != nil {
		return err
	}

	return m.repo.UpdatePoolNodeCount(ctx, poolID, actualCount)
}

// CleanupStaleOperations marks old in-progress operations as failed.
func (m *Manager) CleanupStaleOperations(ctx context.Context, maxAge time.Duration) (int, error) {
	pools, err := m.repo.ListPools(ctx, false)
	if err != nil {
		return 0, err
	}

	cleaned := 0
	cutoff := time.Now().Add(-maxAge)

	for _, pool := range pools {
		ops, opsErr := m.repo.ListOperationsForPool(ctx, pool.ID, 100)
		if opsErr != nil {
			continue
		}

		for _, op := range ops {
			if op.Status == OperationStatusInProgress && op.StartedAt.Before(cutoff) {
				updateErr := m.repo.UpdateOperationStatus(ctx, op.ID, OperationStatusFailed, nil, "operation timed out")
				if updateErr == nil {
					cleaned++
					m.logger.Info("cleaned up stale operation",
						"operation_id", op.ID,
						"pool", pool.Name,
						"started_at", op.StartedAt,
					)
				}
			}
		}
	}

	return cleaned, nil
}
