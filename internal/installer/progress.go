// Package installer provides deployment progress tracking.
package installer

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StepStatus represents the status of a deployment step.
type StepStatus string

const (
	// StepStatusPending indicates the step has not started.
	StepStatusPending StepStatus = "pending"
	// StepStatusInProgress indicates the step is currently running.
	StepStatusInProgress StepStatus = "in_progress"
	// StepStatusCompleted indicates the step completed successfully.
	StepStatusCompleted StepStatus = "completed"
	// StepStatusFailed indicates the step failed.
	StepStatusFailed StepStatus = "failed"
	// StepStatusSkipped indicates the step was skipped.
	StepStatusSkipped StepStatus = "skipped"
)

// SubStep represents a granular action within a deployment step.
type SubStep struct {
	// ID is the unique identifier for the sub-step.
	ID string `json:"id"`
	// Name is the display name of the sub-step.
	Name string `json:"name"`
	// Status is the current status of the sub-step.
	Status StepStatus `json:"status"`
	// Details provides additional context about the sub-step.
	Details string `json:"details,omitempty"`
	// Current is the current item being processed (e.g., server 1).
	Current int `json:"current,omitempty"`
	// Total is the total number of items to process (e.g., of 3 servers).
	Total int `json:"total,omitempty"`
}

// DeploymentStep represents a deployment phase with optional sub-steps.
type DeploymentStep struct {
	// ID is the unique identifier for the step.
	ID string `json:"id"`
	// Name is the display name of the step.
	Name string `json:"name"`
	// Description explains what the step does.
	Description string `json:"description"`
	// Status is the current status of the step.
	Status StepStatus `json:"status"`
	// EstimatedTimeMs is the estimated time for this step in milliseconds.
	EstimatedTimeMs int64 `json:"estimated_time_ms"`
	// ElapsedTimeMs is the actual elapsed time in milliseconds.
	ElapsedTimeMs int64 `json:"elapsed_time_ms,omitempty"`
	// StartedAt is when the step started.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// CompletedAt is when the step completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	// SubSteps are the granular actions within this step.
	SubSteps []SubStep `json:"sub_steps,omitempty"`
	// CurrentSubStep is the index of the current sub-step.
	CurrentSubStep int `json:"current_sub_step"`
	// Error contains error details if the step failed.
	Error *StepError `json:"error,omitempty"`
}

// CreatedResource tracks a resource provisioned during deployment.
type CreatedResource struct {
	// Type is the resource type (e.g., "server", "network", "volume").
	Type string `json:"type"`
	// Name is the resource name.
	Name string `json:"name"`
	// ID is the provider-specific resource ID.
	ID string `json:"id,omitempty"`
	// Region is where the resource was created.
	Region string `json:"region,omitempty"`
}

// DeploymentProgress tracks the overall deployment state.
type DeploymentProgress struct {
	// DeploymentID is the deployment being tracked.
	DeploymentID uuid.UUID `json:"deployment_id"`
	// OverallProgress is the completion percentage (0-100).
	OverallProgress int `json:"overall_progress"`
	// CurrentStepIndex is the index of the current step.
	CurrentStepIndex int `json:"current_step_index"`
	// Steps is the list of deployment steps.
	Steps []DeploymentStep `json:"steps"`
	// StartedAt is when the deployment started.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// EstimatedRemainingMs is the estimated time remaining in milliseconds.
	EstimatedRemainingMs int64 `json:"estimated_remaining_ms"`
	// CanRetry indicates if the deployment can be retried.
	CanRetry bool `json:"can_retry"`
	// ResourcesCreated tracks provisioned resources for cleanup.
	ResourcesCreated []CreatedResource `json:"resources_created,omitempty"`
}

// TimeEstimates holds time estimates for each step per provider.
type TimeEstimates struct {
	Auth     int64 // Authentication
	Network  int64 // Network provisioning
	Compute  int64 // Per-server compute provisioning
	K3s      int64 // K3s installation
	Storage  int64 // MinIO deployment
	Catalog  int64 // Lakekeeper deployment
	Philotes int64 // Philotes deployment
	Health   int64 // Health checks
	SSL      int64 // SSL/TLS configuration
}

// Provider time estimates in milliseconds.
var providerTimeEstimates = map[string]TimeEstimates{
	"hetzner": {
		Auth:     5000,   // 5s
		Network:  30000,  // 30s
		Compute:  60000,  // 60s per server
		K3s:      120000, // 2min
		Storage:  60000,  // 1min
		Catalog:  45000,  // 45s
		Philotes: 90000,  // 1.5min
		Health:   30000,  // 30s
		SSL:      30000,  // 30s
	},
	"scaleway": {
		Auth:     5000,
		Network:  45000,  // 45s
		Compute:  90000,  // 90s per server
		K3s:      150000, // 2.5min
		Storage:  60000,
		Catalog:  45000,
		Philotes: 90000,
		Health:   30000,
		SSL:      30000,
	},
	"ovh": {
		Auth:     5000,
		Network:  60000,  // 1min
		Compute:  120000, // 2min per server
		K3s:      180000, // 3min
		Storage:  60000,
		Catalog:  45000,
		Philotes: 90000,
		Health:   30000,
		SSL:      30000,
	},
	"exoscale": {
		Auth:     5000,
		Network:  30000,
		Compute:  45000,  // 45s per server
		K3s:      120000, // 2min
		Storage:  60000,
		Catalog:  45000,
		Philotes: 90000,
		Health:   30000,
		SSL:      30000,
	},
	"contabo": {
		Auth:     5000,
		Network:  30000,
		Compute:  180000, // 3min per server (slower provisioning)
		K3s:      180000, // 3min
		Storage:  60000,
		Catalog:  45000,
		Philotes: 90000,
		Health:   30000,
		SSL:      30000,
	},
}

// getTimeEstimates returns time estimates for a provider.
func getTimeEstimates(provider string) TimeEstimates {
	if estimates, ok := providerTimeEstimates[provider]; ok {
		return estimates
	}
	// Default to Hetzner estimates
	return providerTimeEstimates["hetzner"]
}

// generateComputeSubSteps creates sub-steps for compute node creation.
func generateComputeSubSteps(workerCount int) []SubStep {
	subSteps := make([]SubStep, 0, workerCount+1)

	// Control plane
	subSteps = append(subSteps, SubStep{
		ID:      "control-plane",
		Name:    "Control Plane",
		Status:  StepStatusPending,
		Current: 1,
		Total:   1,
	})

	// Workers
	for i := 1; i <= workerCount; i++ {
		subSteps = append(subSteps, SubStep{
			ID:      fmt.Sprintf("worker-%d", i),
			Name:    "Worker Node",
			Status:  StepStatusPending,
			Current: i,
			Total:   workerCount,
		})
	}

	return subSteps
}

// GetDeploymentSteps returns the step definitions for a deployment.
func GetDeploymentSteps(provider string, workerCount int) []DeploymentStep {
	estimates := getTimeEstimates(provider)

	// Default worker count if not specified
	if workerCount <= 0 {
		workerCount = 2
	}

	return []DeploymentStep{
		{
			ID:              "auth",
			Name:            "Authenticating",
			Description:     "Verifying cloud provider credentials",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Auth,
		},
		{
			ID:              "network",
			Name:            "Provisioning Network",
			Description:     "Creating VPC, subnets, and security groups",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Network,
			SubSteps: []SubStep{
				{ID: "vpc", Name: "Creating VPC", Status: StepStatusPending},
				{ID: "subnet", Name: "Creating Subnet", Status: StepStatusPending},
				{ID: "firewall", Name: "Configuring Firewall", Status: StepStatusPending},
			},
		},
		{
			ID:              "compute",
			Name:            "Creating Compute Nodes",
			Description:     "Provisioning servers",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Compute * int64(workerCount+1),
			SubSteps:        generateComputeSubSteps(workerCount),
		},
		{
			ID:              "k3s",
			Name:            "Installing Kubernetes",
			Description:     "Setting up K3s cluster",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.K3s,
			SubSteps: []SubStep{
				{ID: "k3s-server", Name: "Installing K3s Server", Status: StepStatusPending},
				{ID: "k3s-agents", Name: "Joining Worker Nodes", Status: StepStatusPending},
				{ID: "kubeconfig", Name: "Retrieving Kubeconfig", Status: StepStatusPending},
			},
		},
		{
			ID:              "storage",
			Name:            "Deploying Storage",
			Description:     "Installing MinIO for object storage",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Storage,
		},
		{
			ID:              "catalog",
			Name:            "Deploying Catalog",
			Description:     "Installing Lakekeeper for Iceberg catalog",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Catalog,
		},
		{
			ID:              "philotes",
			Name:            "Deploying Philotes",
			Description:     "Installing Philotes services",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Philotes,
			SubSteps: []SubStep{
				{ID: "api", Name: "Deploying API Server", Status: StepStatusPending},
				{ID: "worker", Name: "Deploying CDC Worker", Status: StepStatusPending},
				{ID: "dashboard", Name: "Deploying Dashboard", Status: StepStatusPending},
			},
		},
		{
			ID:              "health",
			Name:            "Health Verification",
			Description:     "Running health checks",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.Health,
			SubSteps: []SubStep{
				{ID: "pods", Name: "Checking Pod Status", Status: StepStatusPending},
				{ID: "services", Name: "Verifying Services", Status: StepStatusPending},
				{ID: "endpoints", Name: "Testing Endpoints", Status: StepStatusPending},
			},
		},
		{
			ID:              "ssl",
			Name:            "Configuring SSL/TLS",
			Description:     "Setting up certificates",
			Status:          StepStatusPending,
			EstimatedTimeMs: estimates.SSL,
		},
		{
			ID:              "ready",
			Name:            "Ready!",
			Description:     "Deployment complete",
			Status:          StepStatusPending,
			EstimatedTimeMs: 0,
		},
	}
}

// CalculateTotalEstimate returns the total estimated time for all steps.
func CalculateTotalEstimate(steps []DeploymentStep) int64 {
	var total int64
	for i := range steps {
		total += steps[i].EstimatedTimeMs
	}
	return total
}

// CalculateOverallProgress returns the completion percentage based on step status.
func CalculateOverallProgress(steps []DeploymentStep) int {
	if len(steps) == 0 {
		return 0
	}

	completed := 0
	for i := range steps {
		if steps[i].Status == StepStatusCompleted || steps[i].Status == StepStatusSkipped {
			completed++
		}
	}

	// The last step "ready" counts as 100%
	if completed >= len(steps) {
		return 100
	}

	// Each step is worth an equal percentage
	return (completed * 100) / len(steps)
}

// CalculateRemainingTime estimates the remaining time based on current progress.
func CalculateRemainingTime(steps []DeploymentStep, currentStepIndex int) int64 {
	var remaining int64

	for i := currentStepIndex; i < len(steps); i++ {
		step := steps[i]
		if step.Status == StepStatusInProgress {
			// For in-progress step, estimate partial remaining time
			if step.StartedAt != nil {
				elapsed := time.Since(*step.StartedAt).Milliseconds()
				if elapsed < step.EstimatedTimeMs {
					remaining += step.EstimatedTimeMs - elapsed
				}
			} else {
				remaining += step.EstimatedTimeMs
			}
		} else if step.Status == StepStatusPending {
			remaining += step.EstimatedTimeMs
		}
	}

	return remaining
}

// StepIDToIndex returns the index of a step by ID.
func StepIDToIndex(steps []DeploymentStep, stepID string) int {
	for i := range steps {
		if steps[i].ID == stepID {
			return i
		}
	}
	return -1
}
