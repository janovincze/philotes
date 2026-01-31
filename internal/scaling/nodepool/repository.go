// Package nodepool provides node pool management for infrastructure auto-scaling.
package nodepool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// ErrDuplicateName is returned when a node pool with the same name exists.
var ErrDuplicateName = errors.New("node pool with this name already exists")

// Repository provides database operations for node pool resources.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new node pool repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// Node Pool Operations
// ============================================================================

// CreatePool creates a new node pool.
func (r *Repository) CreatePool(ctx context.Context, pool *NodePool) (*NodePool, error) {
	labelsJSON, err := json.Marshal(pool.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}

	taintsJSON, err := json.Marshal(pool.Taints)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal taints: %w", err)
	}

	query := `
		INSERT INTO philotes.node_pools (
			name, provider, region, instance_type, image, min_nodes, max_nodes,
			current_nodes, labels, taints, user_data_template, ssh_key_id,
			network_id, firewall_id, enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at`

	err = r.db.QueryRow(ctx, query,
		pool.Name,
		pool.Provider,
		pool.Region,
		pool.InstanceType,
		pool.Image,
		pool.MinNodes,
		pool.MaxNodes,
		pool.CurrentNodes,
		labelsJSON,
		taintsJSON,
		pool.UserDataTemplate,
		pool.SSHKeyID,
		pool.NetworkID,
		pool.FirewallID,
		pool.Enabled,
	).Scan(&pool.ID, &pool.CreatedAt, &pool.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("failed to create node pool: %w", err)
	}

	return pool, nil
}

// GetPool retrieves a node pool by ID.
func (r *Repository) GetPool(ctx context.Context, id uuid.UUID) (*NodePool, error) {
	query := `
		SELECT id, name, provider, region, instance_type, image, min_nodes, max_nodes,
			   current_nodes, labels, taints, user_data_template, ssh_key_id,
			   network_id, firewall_id, enabled, created_at, updated_at
		FROM philotes.node_pools
		WHERE id = $1`

	return r.scanPool(r.db.QueryRow(ctx, query, id))
}

// GetPoolByName retrieves a node pool by name.
func (r *Repository) GetPoolByName(ctx context.Context, name string) (*NodePool, error) {
	query := `
		SELECT id, name, provider, region, instance_type, image, min_nodes, max_nodes,
			   current_nodes, labels, taints, user_data_template, ssh_key_id,
			   network_id, firewall_id, enabled, created_at, updated_at
		FROM philotes.node_pools
		WHERE name = $1`

	return r.scanPool(r.db.QueryRow(ctx, query, name))
}

// ListPools lists all node pools.
func (r *Repository) ListPools(ctx context.Context, enabledOnly bool) ([]NodePool, error) {
	query := `
		SELECT id, name, provider, region, instance_type, image, min_nodes, max_nodes,
			   current_nodes, labels, taints, user_data_template, ssh_key_id,
			   network_id, firewall_id, enabled, created_at, updated_at
		FROM philotes.node_pools`

	if enabledOnly {
		query += ` WHERE enabled = true`
	}
	query += ` ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list node pools: %w", err)
	}
	defer rows.Close()

	var pools []NodePool
	for rows.Next() {
		pool, scanErr := r.scanPoolFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		pools = append(pools, *pool)
	}

	return pools, rows.Err()
}

// UpdatePool updates a node pool.
func (r *Repository) UpdatePool(ctx context.Context, pool *NodePool) error {
	labelsJSON, err := json.Marshal(pool.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	taintsJSON, err := json.Marshal(pool.Taints)
	if err != nil {
		return fmt.Errorf("failed to marshal taints: %w", err)
	}

	query := `
		UPDATE philotes.node_pools
		SET name = $2, provider = $3, region = $4, instance_type = $5, image = $6,
			min_nodes = $7, max_nodes = $8, current_nodes = $9, labels = $10,
			taints = $11, user_data_template = $12, ssh_key_id = $13,
			network_id = $14, firewall_id = $15, enabled = $16
		WHERE id = $1
		RETURNING updated_at`

	err = r.db.QueryRow(ctx, query,
		pool.ID,
		pool.Name,
		pool.Provider,
		pool.Region,
		pool.InstanceType,
		pool.Image,
		pool.MinNodes,
		pool.MaxNodes,
		pool.CurrentNodes,
		labelsJSON,
		taintsJSON,
		pool.UserDataTemplate,
		pool.SSHKeyID,
		pool.NetworkID,
		pool.FirewallID,
		pool.Enabled,
	).Scan(&pool.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateName
		}
		return fmt.Errorf("failed to update node pool: %w", err)
	}

	return nil
}

// UpdatePoolNodeCount updates only the current_nodes field of a pool.
func (r *Repository) UpdatePoolNodeCount(ctx context.Context, poolID uuid.UUID, count int) error {
	query := `UPDATE philotes.node_pools SET current_nodes = $2 WHERE id = $1`
	result, err := r.db.Exec(ctx, query, poolID, count)
	if err != nil {
		return fmt.Errorf("failed to update node count: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeletePool deletes a node pool.
func (r *Repository) DeletePool(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM philotes.node_pools WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete node pool: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// Node Operations
// ============================================================================

// CreateNode creates a new node in a pool.
func (r *Repository) CreateNode(ctx context.Context, node *Node) (*Node, error) {
	query := `
		INSERT INTO philotes.node_pool_nodes (
			pool_id, provider_id, node_name, status, public_ip, private_ip,
			instance_type, hourly_cost, is_spot, failure_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		node.PoolID,
		node.ProviderID,
		node.NodeName,
		node.Status,
		node.PublicIP,
		node.PrivateIP,
		node.InstanceType,
		node.HourlyCost,
		node.IsSpot,
		node.FailureReason,
	).Scan(&node.ID, &node.CreatedAt, &node.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return node, nil
}

// GetNode retrieves a node by ID.
func (r *Repository) GetNode(ctx context.Context, id uuid.UUID) (*Node, error) {
	query := `
		SELECT id, pool_id, provider_id, node_name, status, public_ip, private_ip,
			   instance_type, hourly_cost, is_spot, failure_reason,
			   created_at, updated_at, deleted_at
		FROM philotes.node_pool_nodes
		WHERE id = $1`

	return r.scanNode(r.db.QueryRow(ctx, query, id))
}

// GetNodeByProviderID retrieves a node by cloud provider ID.
func (r *Repository) GetNodeByProviderID(ctx context.Context, providerID string) (*Node, error) {
	query := `
		SELECT id, pool_id, provider_id, node_name, status, public_ip, private_ip,
			   instance_type, hourly_cost, is_spot, failure_reason,
			   created_at, updated_at, deleted_at
		FROM philotes.node_pool_nodes
		WHERE provider_id = $1`

	return r.scanNode(r.db.QueryRow(ctx, query, providerID))
}

// GetNodeByName retrieves a node by Kubernetes node name.
func (r *Repository) GetNodeByName(ctx context.Context, nodeName string) (*Node, error) {
	query := `
		SELECT id, pool_id, provider_id, node_name, status, public_ip, private_ip,
			   instance_type, hourly_cost, is_spot, failure_reason,
			   created_at, updated_at, deleted_at
		FROM philotes.node_pool_nodes
		WHERE node_name = $1`

	return r.scanNode(r.db.QueryRow(ctx, query, nodeName))
}

// ListNodesForPool lists all nodes in a pool.
func (r *Repository) ListNodesForPool(ctx context.Context, poolID uuid.UUID, activeOnly bool) ([]Node, error) {
	query := `
		SELECT id, pool_id, provider_id, node_name, status, public_ip, private_ip,
			   instance_type, hourly_cost, is_spot, failure_reason,
			   created_at, updated_at, deleted_at
		FROM philotes.node_pool_nodes
		WHERE pool_id = $1`

	if activeOnly {
		query += ` AND deleted_at IS NULL AND status NOT IN ('deleted', 'failed')`
	}
	query += ` ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		node, scanErr := r.scanNodeFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		nodes = append(nodes, *node)
	}

	return nodes, rows.Err()
}

// UpdateNodeStatus updates the status of a node.
func (r *Repository) UpdateNodeStatus(ctx context.Context, id uuid.UUID, status NodeStatus, failureReason string) error {
	query := `
		UPDATE philotes.node_pool_nodes
		SET status = $2, failure_reason = $3
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, status, failureReason)
	if err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateNode updates a node's details.
func (r *Repository) UpdateNode(ctx context.Context, node *Node) error {
	query := `
		UPDATE philotes.node_pool_nodes
		SET provider_id = $2, node_name = $3, status = $4, public_ip = $5, private_ip = $6,
			instance_type = $7, hourly_cost = $8, is_spot = $9, failure_reason = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, query,
		node.ID,
		node.ProviderID,
		node.NodeName,
		node.Status,
		node.PublicIP,
		node.PrivateIP,
		node.InstanceType,
		node.HourlyCost,
		node.IsSpot,
		node.FailureReason,
	).Scan(&node.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// SoftDeleteNode marks a node as deleted.
func (r *Repository) SoftDeleteNode(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE philotes.node_pool_nodes
		SET status = 'deleted', deleted_at = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountActiveNodesForPool counts active nodes in a pool.
func (r *Repository) CountActiveNodesForPool(ctx context.Context, poolID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM philotes.node_pool_nodes
		WHERE pool_id = $1 AND deleted_at IS NULL AND status NOT IN ('deleted', 'failed')`

	var count int
	err := r.db.QueryRow(ctx, query, poolID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count nodes: %w", err)
	}
	return count, nil
}

// ============================================================================
// Scaling Operation Operations
// ============================================================================

// CreateOperation creates a new scaling operation.
func (r *Repository) CreateOperation(ctx context.Context, op *ScalingOperation) (*ScalingOperation, error) {
	nodesAffectedJSON, err := json.Marshal(op.NodesAffected)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal nodes_affected: %w", err)
	}

	query := `
		INSERT INTO philotes.node_scaling_operations (
			pool_id, policy_id, action, previous_count, target_count, actual_count,
			status, reason, triggered_by, nodes_affected, error_message,
			estimated_cost_change, dry_run
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, started_at`

	err = r.db.QueryRow(ctx, query,
		op.PoolID,
		op.PolicyID,
		op.Action,
		op.PreviousCount,
		op.TargetCount,
		op.ActualCount,
		op.Status,
		op.Reason,
		op.TriggeredBy,
		nodesAffectedJSON,
		op.ErrorMessage,
		op.EstimatedCostChange,
		op.DryRun,
	).Scan(&op.ID, &op.StartedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create operation: %w", err)
	}

	return op, nil
}

// GetOperation retrieves a scaling operation by ID.
func (r *Repository) GetOperation(ctx context.Context, id uuid.UUID) (*ScalingOperation, error) {
	query := `
		SELECT id, pool_id, policy_id, action, previous_count, target_count, actual_count,
			   status, reason, triggered_by, nodes_affected, started_at, completed_at,
			   error_message, estimated_cost_change, dry_run
		FROM philotes.node_scaling_operations
		WHERE id = $1`

	return r.scanOperation(r.db.QueryRow(ctx, query, id))
}

// ListOperationsForPool lists scaling operations for a pool.
func (r *Repository) ListOperationsForPool(ctx context.Context, poolID uuid.UUID, limit int) ([]ScalingOperation, error) {
	query := `
		SELECT id, pool_id, policy_id, action, previous_count, target_count, actual_count,
			   status, reason, triggered_by, nodes_affected, started_at, completed_at,
			   error_message, estimated_cost_change, dry_run
		FROM philotes.node_scaling_operations
		WHERE pool_id = $1
		ORDER BY started_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(ctx, query, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	defer rows.Close()

	var ops []ScalingOperation
	for rows.Next() {
		op, scanErr := r.scanOperationFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		ops = append(ops, *op)
	}

	return ops, rows.Err()
}

// UpdateOperationStatus updates a scaling operation's status.
func (r *Repository) UpdateOperationStatus(ctx context.Context, id uuid.UUID, status OperationStatus, actualCount *int, errorMessage string) error {
	var completedAt *time.Time
	if status.IsTerminal() {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE philotes.node_scaling_operations
		SET status = $2, actual_count = $3, error_message = $4, completed_at = $5
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id, status, actualCount, errorMessage, completedAt)
	if err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateOperationNodesAffected updates the nodes_affected field of an operation.
func (r *Repository) UpdateOperationNodesAffected(ctx context.Context, id uuid.UUID, nodeIDs []uuid.UUID) error {
	nodesJSON, err := json.Marshal(nodeIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal node IDs: %w", err)
	}

	query := `UPDATE philotes.node_scaling_operations SET nodes_affected = $2 WHERE id = $1`
	result, execErr := r.db.Exec(ctx, query, id, nodesJSON)
	if execErr != nil {
		return fmt.Errorf("failed to update nodes_affected: %w", execErr)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// Pricing Operations
// ============================================================================

// GetPricing retrieves pricing for an instance type.
func (r *Repository) GetPricing(ctx context.Context, provider Provider, instanceType, region string) (*InstanceTypePricing, error) {
	query := `
		SELECT id, provider, instance_type, region, hourly_cost, cpu_cores, memory_mb,
			   disk_gb, supports_spot, spot_hourly_cost, last_updated
		FROM philotes.instance_type_pricing
		WHERE provider = $1 AND instance_type = $2 AND region = $3`

	pricing := &InstanceTypePricing{}
	err := r.db.QueryRow(ctx, query, provider, instanceType, region).Scan(
		&pricing.ID,
		&pricing.Provider,
		&pricing.InstanceType,
		&pricing.Region,
		&pricing.HourlyCost,
		&pricing.CPUCores,
		&pricing.MemoryMB,
		&pricing.DiskGB,
		&pricing.SupportsSpot,
		&pricing.SpotHourlyCost,
		&pricing.LastUpdated,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	return pricing, nil
}

// ListPricingForProvider lists all pricing for a provider.
func (r *Repository) ListPricingForProvider(ctx context.Context, provider Provider, region string) ([]InstanceTypePricing, error) {
	query := `
		SELECT id, provider, instance_type, region, hourly_cost, cpu_cores, memory_mb,
			   disk_gb, supports_spot, spot_hourly_cost, last_updated
		FROM philotes.instance_type_pricing
		WHERE provider = $1`

	args := []any{provider}
	if region != "" {
		query += ` AND region = $2`
		args = append(args, region)
	}
	query += ` ORDER BY hourly_cost`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pricing: %w", err)
	}
	defer rows.Close()

	var pricing []InstanceTypePricing
	for rows.Next() {
		var p InstanceTypePricing
		scanErr := rows.Scan(
			&p.ID,
			&p.Provider,
			&p.InstanceType,
			&p.Region,
			&p.HourlyCost,
			&p.CPUCores,
			&p.MemoryMB,
			&p.DiskGB,
			&p.SupportsSpot,
			&p.SpotHourlyCost,
			&p.LastUpdated,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan pricing: %w", scanErr)
		}
		pricing = append(pricing, p)
	}

	return pricing, rows.Err()
}

// ============================================================================
// Helper Functions
// ============================================================================

func (r *Repository) scanPool(row pgx.Row) (*NodePool, error) {
	pool := &NodePool{}
	var labelsJSON, taintsJSON []byte

	err := row.Scan(
		&pool.ID,
		&pool.Name,
		&pool.Provider,
		&pool.Region,
		&pool.InstanceType,
		&pool.Image,
		&pool.MinNodes,
		&pool.MaxNodes,
		&pool.CurrentNodes,
		&labelsJSON,
		&taintsJSON,
		&pool.UserDataTemplate,
		&pool.SSHKeyID,
		&pool.NetworkID,
		&pool.FirewallID,
		&pool.Enabled,
		&pool.CreatedAt,
		&pool.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan node pool: %w", err)
	}

	if err := json.Unmarshal(labelsJSON, &pool.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	if err := json.Unmarshal(taintsJSON, &pool.Taints); err != nil {
		return nil, fmt.Errorf("failed to unmarshal taints: %w", err)
	}

	return pool, nil
}

func (r *Repository) scanPoolFromRows(rows pgx.Rows) (*NodePool, error) {
	pool := &NodePool{}
	var labelsJSON, taintsJSON []byte

	err := rows.Scan(
		&pool.ID,
		&pool.Name,
		&pool.Provider,
		&pool.Region,
		&pool.InstanceType,
		&pool.Image,
		&pool.MinNodes,
		&pool.MaxNodes,
		&pool.CurrentNodes,
		&labelsJSON,
		&taintsJSON,
		&pool.UserDataTemplate,
		&pool.SSHKeyID,
		&pool.NetworkID,
		&pool.FirewallID,
		&pool.Enabled,
		&pool.CreatedAt,
		&pool.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan node pool: %w", err)
	}

	if err := json.Unmarshal(labelsJSON, &pool.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	if err := json.Unmarshal(taintsJSON, &pool.Taints); err != nil {
		return nil, fmt.Errorf("failed to unmarshal taints: %w", err)
	}

	return pool, nil
}

func (r *Repository) scanNode(row pgx.Row) (*Node, error) {
	node := &Node{}
	err := row.Scan(
		&node.ID,
		&node.PoolID,
		&node.ProviderID,
		&node.NodeName,
		&node.Status,
		&node.PublicIP,
		&node.PrivateIP,
		&node.InstanceType,
		&node.HourlyCost,
		&node.IsSpot,
		&node.FailureReason,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.DeletedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan node: %w", err)
	}

	return node, nil
}

func (r *Repository) scanNodeFromRows(rows pgx.Rows) (*Node, error) {
	node := &Node{}
	err := rows.Scan(
		&node.ID,
		&node.PoolID,
		&node.ProviderID,
		&node.NodeName,
		&node.Status,
		&node.PublicIP,
		&node.PrivateIP,
		&node.InstanceType,
		&node.HourlyCost,
		&node.IsSpot,
		&node.FailureReason,
		&node.CreatedAt,
		&node.UpdatedAt,
		&node.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan node: %w", err)
	}

	return node, nil
}

func (r *Repository) scanOperation(row pgx.Row) (*ScalingOperation, error) {
	op := &ScalingOperation{}
	var nodesAffectedJSON []byte

	err := row.Scan(
		&op.ID,
		&op.PoolID,
		&op.PolicyID,
		&op.Action,
		&op.PreviousCount,
		&op.TargetCount,
		&op.ActualCount,
		&op.Status,
		&op.Reason,
		&op.TriggeredBy,
		&nodesAffectedJSON,
		&op.StartedAt,
		&op.CompletedAt,
		&op.ErrorMessage,
		&op.EstimatedCostChange,
		&op.DryRun,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan operation: %w", err)
	}

	if len(nodesAffectedJSON) > 0 {
		if jsonErr := json.Unmarshal(nodesAffectedJSON, &op.NodesAffected); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes_affected: %w", jsonErr)
		}
	}

	return op, nil
}

func (r *Repository) scanOperationFromRows(rows pgx.Rows) (*ScalingOperation, error) {
	op := &ScalingOperation{}
	var nodesAffectedJSON []byte

	err := rows.Scan(
		&op.ID,
		&op.PoolID,
		&op.PolicyID,
		&op.Action,
		&op.PreviousCount,
		&op.TargetCount,
		&op.ActualCount,
		&op.Status,
		&op.Reason,
		&op.TriggeredBy,
		&nodesAffectedJSON,
		&op.StartedAt,
		&op.CompletedAt,
		&op.ErrorMessage,
		&op.EstimatedCostChange,
		&op.DryRun,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan operation: %w", err)
	}

	if len(nodesAffectedJSON) > 0 {
		if jsonErr := json.Unmarshal(nodesAffectedJSON, &op.NodesAffected); jsonErr != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes_affected: %w", jsonErr)
		}
	}

	return op, nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Check for PostgreSQL unique constraint violation
	errStr := err.Error()
	return !errors.Is(err, pgx.ErrNoRows) &&
		(strings.Contains(errStr, "unique constraint") || strings.Contains(errStr, "duplicate key"))
}
