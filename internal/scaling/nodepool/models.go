// Package nodepool provides node pool management for infrastructure auto-scaling.
package nodepool

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Provider represents a supported cloud provider.
type Provider string

const (
	ProviderHetzner  Provider = "hetzner"
	ProviderScaleway Provider = "scaleway"
	ProviderOVH      Provider = "ovh"
	ProviderExoscale Provider = "exoscale"
	ProviderContabo  Provider = "contabo"
)

// IsValid checks if the provider is valid.
func (p Provider) IsValid() bool {
	switch p {
	case ProviderHetzner, ProviderScaleway, ProviderOVH, ProviderExoscale, ProviderContabo:
		return true
	}
	return false
}

// String returns the string representation of the provider.
func (p Provider) String() string {
	return string(p)
}

// NodeStatus represents the status of a node in a pool.
type NodeStatus string

const (
	NodeStatusCreating NodeStatus = "creating"
	NodeStatusJoining  NodeStatus = "joining"
	NodeStatusReady    NodeStatus = "ready"
	NodeStatusDraining NodeStatus = "draining"
	NodeStatusDeleting NodeStatus = "deleting"
	NodeStatusDeleted  NodeStatus = "deleted"
	NodeStatusFailed   NodeStatus = "failed"
)

// IsValid checks if the node status is valid.
func (s NodeStatus) IsValid() bool {
	switch s {
	case NodeStatusCreating, NodeStatusJoining, NodeStatusReady,
		NodeStatusDraining, NodeStatusDeleting, NodeStatusDeleted, NodeStatusFailed:
		return true
	}
	return false
}

// IsActive returns true if the node is in an active state (not deleted or failed).
func (s NodeStatus) IsActive() bool {
	switch s {
	case NodeStatusCreating, NodeStatusJoining, NodeStatusReady, NodeStatusDraining, NodeStatusDeleting:
		return true
	}
	return false
}

// OperationStatus represents the status of a scaling operation.
type OperationStatus string

const (
	OperationStatusPending    OperationStatus = "pending"
	OperationStatusInProgress OperationStatus = "in_progress"
	OperationStatusCompleted  OperationStatus = "completed"
	OperationStatusFailed     OperationStatus = "failed"
	OperationStatusCanceled   OperationStatus = "canceled"
)

// IsValid checks if the operation status is valid.
func (s OperationStatus) IsValid() bool {
	switch s {
	case OperationStatusPending, OperationStatusInProgress,
		OperationStatusCompleted, OperationStatusFailed, OperationStatusCanceled:
		return true
	}
	return false
}

// IsTerminal returns true if the operation is in a terminal state.
func (s OperationStatus) IsTerminal() bool {
	switch s {
	case OperationStatusCompleted, OperationStatusFailed, OperationStatusCanceled:
		return true
	}
	return false
}

// OperationAction represents the type of scaling operation.
type OperationAction string

const (
	OperationActionScaleUp   OperationAction = "scale_up"
	OperationActionScaleDown OperationAction = "scale_down"
)

// Taint represents a Kubernetes taint to apply to nodes.
type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"` // NoSchedule, PreferNoSchedule, NoExecute
}

// NodePool represents a group of nodes with similar characteristics.
type NodePool struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	Provider         Provider          `json:"provider"`
	Region           string            `json:"region"`
	InstanceType     string            `json:"instance_type"`
	Image            string            `json:"image"`
	MinNodes         int               `json:"min_nodes"`
	MaxNodes         int               `json:"max_nodes"`
	CurrentNodes     int               `json:"current_nodes"`
	Labels           map[string]string `json:"labels,omitempty"`
	Taints           []Taint           `json:"taints,omitempty"`
	UserDataTemplate string            `json:"user_data_template,omitempty"`
	SSHKeyID         string            `json:"ssh_key_id,omitempty"`
	NetworkID        string            `json:"network_id,omitempty"`
	FirewallID       string            `json:"firewall_id,omitempty"`
	Enabled          bool              `json:"enabled"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// Validate validates the node pool configuration.
func (p *NodePool) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !p.Provider.IsValid() {
		return fmt.Errorf("invalid provider: %s", p.Provider)
	}
	if p.Region == "" {
		return fmt.Errorf("region is required")
	}
	if p.InstanceType == "" {
		return fmt.Errorf("instance_type is required")
	}
	if p.MinNodes < 0 {
		return fmt.Errorf("min_nodes must be >= 0")
	}
	if p.MaxNodes < 1 {
		return fmt.Errorf("max_nodes must be >= 1")
	}
	if p.MinNodes > p.MaxNodes {
		return fmt.Errorf("min_nodes (%d) cannot be greater than max_nodes (%d)", p.MinNodes, p.MaxNodes)
	}
	return nil
}

// CanScaleUp returns true if the pool can scale up.
func (p *NodePool) CanScaleUp() bool {
	return p.Enabled && p.CurrentNodes < p.MaxNodes
}

// CanScaleDown returns true if the pool can scale down.
func (p *NodePool) CanScaleDown() bool {
	return p.Enabled && p.CurrentNodes > p.MinNodes
}

// ClampNodeCount clamps the given node count to pool limits.
func (p *NodePool) ClampNodeCount(count int) int {
	if count < p.MinNodes {
		return p.MinNodes
	}
	if count > p.MaxNodes {
		return p.MaxNodes
	}
	return count
}

// Node represents a single node in a node pool.
type Node struct {
	ID            uuid.UUID  `json:"id"`
	PoolID        uuid.UUID  `json:"pool_id"`
	ProviderID    string     `json:"provider_id"`
	NodeName      string     `json:"node_name,omitempty"`
	Status        NodeStatus `json:"status"`
	PublicIP      string     `json:"public_ip,omitempty"`
	PrivateIP     string     `json:"private_ip,omitempty"`
	InstanceType  string     `json:"instance_type"`
	HourlyCost    *float64   `json:"hourly_cost,omitempty"`
	IsSpot        bool       `json:"is_spot"`
	FailureReason string     `json:"failure_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// IsActive returns true if the node is in an active state.
func (n *Node) IsActive() bool {
	return n.Status.IsActive() && n.DeletedAt == nil
}

// ScalingOperation represents an audit log entry for node scaling operations.
type ScalingOperation struct {
	ID                  uuid.UUID       `json:"id"`
	PoolID              uuid.UUID       `json:"pool_id"`
	PolicyID            *uuid.UUID      `json:"policy_id,omitempty"`
	Action              OperationAction `json:"action"`
	PreviousCount       int             `json:"previous_count"`
	TargetCount         int             `json:"target_count"`
	ActualCount         *int            `json:"actual_count,omitempty"`
	Status              OperationStatus `json:"status"`
	Reason              string          `json:"reason,omitempty"`
	TriggeredBy         string          `json:"triggered_by,omitempty"`
	NodesAffected       []uuid.UUID     `json:"nodes_affected,omitempty"`
	StartedAt           time.Time       `json:"started_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	ErrorMessage        string          `json:"error_message,omitempty"`
	EstimatedCostChange *float64        `json:"estimated_cost_change,omitempty"`
	DryRun              bool            `json:"dry_run"`
}

// Delta returns the change in node count.
func (o *ScalingOperation) Delta() int {
	return o.TargetCount - o.PreviousCount
}

// InstanceTypePricing represents cached pricing information for an instance type.
type InstanceTypePricing struct {
	ID             uuid.UUID `json:"id"`
	Provider       Provider  `json:"provider"`
	InstanceType   string    `json:"instance_type"`
	Region         string    `json:"region"`
	HourlyCost     float64   `json:"hourly_cost"`
	CPUCores       int       `json:"cpu_cores"`
	MemoryMB       int       `json:"memory_mb"`
	DiskGB         *int      `json:"disk_gb,omitempty"`
	SupportsSpot   bool      `json:"supports_spot"`
	SpotHourlyCost *float64  `json:"spot_hourly_cost,omitempty"`
	LastUpdated    time.Time `json:"last_updated"`
}

// ClusterCapacity represents the current capacity of the cluster.
type ClusterCapacity struct {
	TotalNodes          int          `json:"total_nodes"`
	ReadyNodes          int          `json:"ready_nodes"`
	TotalCPUCores       int          `json:"total_cpu_cores"`
	TotalMemoryMB       int          `json:"total_memory_mb"`
	AllocatableCPU      int          `json:"allocatable_cpu_cores"`
	AllocatableMemory   int          `json:"allocatable_memory_mb"`
	UsedCPUPercent      float64      `json:"used_cpu_percent"`
	UsedMemoryPercent   float64      `json:"used_memory_percent"`
	PendingPods         int          `json:"pending_pods"`
	UnschedulablePods   int          `json:"unschedulable_pods"`
	EstimatedHourlyCost float64      `json:"estimated_hourly_cost"`
	NodePools           []PoolStatus `json:"node_pools"`
}

// PoolStatus represents the status summary of a node pool.
type PoolStatus struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Provider     Provider  `json:"provider"`
	Region       string    `json:"region"`
	InstanceType string    `json:"instance_type"`
	MinNodes     int       `json:"min_nodes"`
	MaxNodes     int       `json:"max_nodes"`
	CurrentNodes int       `json:"current_nodes"`
	ReadyNodes   int       `json:"ready_nodes"`
	Enabled      bool      `json:"enabled"`
	HourlyCost   float64   `json:"hourly_cost"`
}
