// Package models provides API request and response types.
package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling/nodepool"
)

// TaintRequest represents a Kubernetes taint in API requests.
type TaintRequest struct {
	Key    string `json:"key" binding:"required"`
	Value  string `json:"value"`
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"`
}

// CreateNodePoolRequest represents a request to create a node pool.
type CreateNodePoolRequest struct {
	Name             string            `json:"name" binding:"required,min=1,max=100"`
	Provider         string            `json:"provider" binding:"required,oneof=hetzner scaleway ovh exoscale contabo"`
	Region           string            `json:"region" binding:"required"`
	InstanceType     string            `json:"instance_type" binding:"required"`
	Image            string            `json:"image,omitempty"`
	MinNodes         int               `json:"min_nodes" binding:"gte=0"`
	MaxNodes         int               `json:"max_nodes" binding:"gte=1"`
	Labels           map[string]string `json:"labels,omitempty"`
	Taints           []TaintRequest    `json:"taints,omitempty"`
	UserDataTemplate string            `json:"user_data_template,omitempty"`
	SSHKeyID         string            `json:"ssh_key_id,omitempty"`
	NetworkID        string            `json:"network_id,omitempty"`
	FirewallID       string            `json:"firewall_id,omitempty"`
	Enabled          *bool             `json:"enabled,omitempty"`
}

// Validate validates the create node pool request.
func (r *CreateNodePoolRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}

	if r.Provider == "" {
		errors = append(errors, FieldError{Field: "provider", Message: "provider is required"})
	} else if !nodepool.Provider(r.Provider).IsValid() {
		errors = append(errors, FieldError{Field: "provider", Message: "provider must be one of: hetzner, scaleway, ovh, exoscale, contabo"})
	}

	if r.Region == "" {
		errors = append(errors, FieldError{Field: "region", Message: "region is required"})
	}

	if r.InstanceType == "" {
		errors = append(errors, FieldError{Field: "instance_type", Message: "instance_type is required"})
	}

	if r.MinNodes < 0 {
		errors = append(errors, FieldError{Field: "min_nodes", Message: "min_nodes must be >= 0"})
	}

	if r.MaxNodes < 1 {
		errors = append(errors, FieldError{Field: "max_nodes", Message: "max_nodes must be >= 1"})
	}

	if r.MinNodes > r.MaxNodes {
		errors = append(errors, FieldError{Field: "min_nodes", Message: "min_nodes cannot be greater than max_nodes"})
	}

	for i, taint := range r.Taints {
		if taint.Key == "" {
			errors = append(errors, FieldError{Field: "taints[" + itoa(i) + "].key", Message: "key is required"})
		}
		if taint.Effect != "NoSchedule" && taint.Effect != "PreferNoSchedule" && taint.Effect != "NoExecute" {
			errors = append(errors, FieldError{Field: "taints[" + itoa(i) + "].effect", Message: "effect must be NoSchedule, PreferNoSchedule, or NoExecute"})
		}
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateNodePoolRequest) ApplyDefaults() {
	if r.Image == "" {
		r.Image = "ubuntu-24.04"
	}
	if r.MinNodes == 0 && r.MaxNodes == 0 {
		r.MinNodes = 1
		r.MaxNodes = 10
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
}

// ToNodePool converts the request to a NodePool.
func (r *CreateNodePoolRequest) ToNodePool() *nodepool.NodePool {
	pool := &nodepool.NodePool{
		Name:             r.Name,
		Provider:         nodepool.Provider(r.Provider),
		Region:           r.Region,
		InstanceType:     r.InstanceType,
		Image:            r.Image,
		MinNodes:         r.MinNodes,
		MaxNodes:         r.MaxNodes,
		Labels:           r.Labels,
		UserDataTemplate: r.UserDataTemplate,
		SSHKeyID:         r.SSHKeyID,
		NetworkID:        r.NetworkID,
		FirewallID:       r.FirewallID,
		Enabled:          *r.Enabled,
	}

	for _, taint := range r.Taints {
		pool.Taints = append(pool.Taints, nodepool.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: taint.Effect,
		})
	}

	return pool
}

// UpdateNodePoolRequest represents a request to update a node pool.
type UpdateNodePoolRequest struct {
	Name             *string           `json:"name,omitempty"`
	MinNodes         *int              `json:"min_nodes,omitempty"`
	MaxNodes         *int              `json:"max_nodes,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	Taints           []TaintRequest    `json:"taints,omitempty"`
	UserDataTemplate *string           `json:"user_data_template,omitempty"`
	SSHKeyID         *string           `json:"ssh_key_id,omitempty"`
	NetworkID        *string           `json:"network_id,omitempty"`
	FirewallID       *string           `json:"firewall_id,omitempty"`
	Enabled          *bool             `json:"enabled,omitempty"`
}

// Validate validates the update node pool request.
func (r *UpdateNodePoolRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}

	if r.MinNodes != nil && *r.MinNodes < 0 {
		errors = append(errors, FieldError{Field: "min_nodes", Message: "min_nodes must be >= 0"})
	}

	if r.MaxNodes != nil && *r.MaxNodes < 1 {
		errors = append(errors, FieldError{Field: "max_nodes", Message: "max_nodes must be >= 1"})
	}

	return errors
}

// ApplyToPool applies the update to an existing pool.
func (r *UpdateNodePoolRequest) ApplyToPool(pool *nodepool.NodePool) {
	if r.Name != nil {
		pool.Name = *r.Name
	}
	if r.MinNodes != nil {
		pool.MinNodes = *r.MinNodes
	}
	if r.MaxNodes != nil {
		pool.MaxNodes = *r.MaxNodes
	}
	if r.Labels != nil {
		pool.Labels = r.Labels
	}
	if r.Taints != nil {
		pool.Taints = make([]nodepool.Taint, 0, len(r.Taints))
		for _, taint := range r.Taints {
			pool.Taints = append(pool.Taints, nodepool.Taint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: taint.Effect,
			})
		}
	}
	if r.UserDataTemplate != nil {
		pool.UserDataTemplate = *r.UserDataTemplate
	}
	if r.SSHKeyID != nil {
		pool.SSHKeyID = *r.SSHKeyID
	}
	if r.NetworkID != nil {
		pool.NetworkID = *r.NetworkID
	}
	if r.FirewallID != nil {
		pool.FirewallID = *r.FirewallID
	}
	if r.Enabled != nil {
		pool.Enabled = *r.Enabled
	}
}

// ScaleNodePoolRequest represents a request to scale a node pool.
type ScaleNodePoolRequest struct {
	TargetNodes int    `json:"target_nodes" binding:"gte=0"`
	DryRun      bool   `json:"dry_run,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// Validate validates the scale request.
func (r *ScaleNodePoolRequest) Validate() []FieldError {
	var errors []FieldError

	if r.TargetNodes < 0 {
		errors = append(errors, FieldError{Field: "target_nodes", Message: "target_nodes must be >= 0"})
	}

	return errors
}

// NodePoolResponse wraps a node pool for API responses.
type NodePoolResponse struct {
	Pool  *nodepool.NodePool `json:"pool"`
	Nodes []nodepool.Node    `json:"nodes,omitempty"`
}

// NodePoolListResponse wraps a list of node pools for API responses.
type NodePoolListResponse struct {
	Pools      []nodepool.NodePool `json:"pools"`
	TotalCount int                 `json:"total_count"`
}

// NodePoolStatusResponse wraps node pool status for API responses.
type NodePoolStatusResponse struct {
	Status *nodepool.NodePoolStatus `json:"status"`
}

// NodePoolStatusListResponse wraps a list of node pool statuses.
type NodePoolStatusListResponse struct {
	Statuses   []nodepool.NodePoolStatus `json:"statuses"`
	TotalCount int                       `json:"total_count"`
}

// NodeResponse wraps a node for API responses.
type NodeResponse struct {
	Node *nodepool.Node `json:"node"`
}

// NodeListResponse wraps a list of nodes for API responses.
type NodeListResponse struct {
	Nodes      []nodepool.Node `json:"nodes"`
	TotalCount int             `json:"total_count"`
}

// ScalingOperationResponse wraps a scaling operation for API responses.
type ScalingOperationResponse struct {
	Operation *nodepool.ScalingOperation `json:"operation"`
}

// ScalingOperationListResponse wraps a list of scaling operations.
type ScalingOperationListResponse struct {
	Operations []nodepool.ScalingOperation `json:"operations"`
	TotalCount int                         `json:"total_count"`
}

// ClusterCapacityResponse represents cluster capacity for API responses.
type ClusterCapacityResponse struct {
	TotalNodes          int                       `json:"total_nodes"`
	ReadyNodes          int                       `json:"ready_nodes"`
	TotalCPUCores       int64                     `json:"total_cpu_cores"`
	TotalMemoryMB       int64                     `json:"total_memory_mb"`
	AllocatableCPU      int64                     `json:"allocatable_cpu_cores"`
	AllocatableMemory   int64                     `json:"allocatable_memory_mb"`
	UsedCPUPercent      float64                   `json:"used_cpu_percent"`
	UsedMemoryPercent   float64                   `json:"used_memory_percent"`
	PendingPods         int                       `json:"pending_pods"`
	UnschedulablePods   int                       `json:"unschedulable_pods"`
	EstimatedHourlyCost float64                   `json:"estimated_hourly_cost"`
	NodePools           []nodepool.NodePoolStatus `json:"node_pools"`
}

// PendingPodsResponse represents pending pods summary for API responses.
type PendingPodsResponse struct {
	TotalPending        int               `json:"total_pending"`
	Unschedulable       int               `json:"unschedulable"`
	WaitingForResources int               `json:"waiting_for_resources"`
	OldestPending       *time.Time        `json:"oldest_pending,omitempty"`
	ByReason            map[string]int    `json:"by_reason"`
	CPURequested        int64             `json:"cpu_requested_millicores"`
	MemoryRequested     int64             `json:"memory_requested_mb"`
}

// InstanceTypePricingResponse represents pricing for an instance type.
type InstanceTypePricingResponse struct {
	Provider       string   `json:"provider"`
	InstanceType   string   `json:"instance_type"`
	Region         string   `json:"region"`
	HourlyCost     float64  `json:"hourly_cost"`
	CPUCores       int      `json:"cpu_cores"`
	MemoryMB       int      `json:"memory_mb"`
	DiskGB         *int     `json:"disk_gb,omitempty"`
	SupportsSpot   bool     `json:"supports_spot"`
	SpotHourlyCost *float64 `json:"spot_hourly_cost,omitempty"`
}

// DrainNodeRequest represents a request to drain a node.
type DrainNodeRequest struct {
	Force              bool `json:"force,omitempty"`
	GracePeriodSeconds *int `json:"grace_period_seconds,omitempty"`
	TimeoutSeconds     *int `json:"timeout_seconds,omitempty"`
}

// ScaleResponse represents the result of a scale operation.
type ScaleResponse struct {
	OperationID     uuid.UUID `json:"operation_id"`
	Pool            string    `json:"pool"`
	PreviousCount   int       `json:"previous_count"`
	TargetCount     int       `json:"target_count"`
	Action          string    `json:"action"`
	DryRun          bool      `json:"dry_run"`
	EstimatedCostChange *float64 `json:"estimated_cost_change,omitempty"`
}

// helper function
func itoa(i int) string {
	return strconv.Itoa(i)
}
