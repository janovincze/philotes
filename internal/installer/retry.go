// Package installer provides deployment retry functionality.
package installer

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrNoProgressFound is returned when no progress exists for a deployment.
var ErrNoProgressFound = errors.New("no progress found for deployment")

// ErrNotRetryable is returned when a deployment cannot be retried.
var ErrNotRetryable = errors.New("deployment cannot be retried from current state")

// ErrDeploymentActive is returned when trying to retry an active deployment.
var ErrDeploymentActive = errors.New("deployment is still active")

// RetryableSteps defines which steps can be retried.
var RetryableSteps = map[string]bool{
	"auth":     true,
	"network":  true,
	"compute":  true,
	"k3s":      true,
	"storage":  true,
	"catalog":  true,
	"philotes": true,
	"health":   true,
	"ssl":      true,
	"ready":    false, // Final step, nothing to retry
}

// CanRetryStep returns whether a specific step can be retried.
func CanRetryStep(stepID string) bool {
	retryable, exists := RetryableSteps[stepID]
	return exists && retryable
}

// FindFailedStep returns the first failed step in a deployment progress.
func FindFailedStep(progress *DeploymentProgress) *DeploymentStep {
	if progress == nil {
		return nil
	}

	for i := range progress.Steps {
		if progress.Steps[i].Status == StepStatusFailed {
			return &progress.Steps[i]
		}
	}

	return nil
}

// CanRetryDeployment checks if a deployment can be retried.
func CanRetryDeployment(progress *DeploymentProgress) bool {
	if progress == nil {
		return false
	}

	// Check if any step is currently in progress
	for _, step := range progress.Steps {
		if step.Status == StepStatusInProgress {
			return false
		}
	}

	// Find the failed step
	failedStep := FindFailedStep(progress)
	if failedStep == nil {
		return false
	}

	// Check if the failed step is retryable
	return CanRetryStep(failedStep.ID)
}

// RetryDeployment attempts to retry a failed deployment.
func (o *DeploymentOrchestrator) RetryDeployment(ctx context.Context, deploymentID uuid.UUID, cfg *DeploymentConfig, statusCallback func(status string, err error)) error {
	progress := o.tracker.GetProgress(deploymentID)
	if progress == nil {
		return ErrNoProgressFound
	}

	// Check if any step is currently in progress
	for _, step := range progress.Steps {
		if step.Status == StepStatusInProgress {
			return ErrDeploymentActive
		}
	}

	// Find the failed step
	failedStep := FindFailedStep(progress)
	if failedStep == nil {
		return ErrNotRetryable
	}

	// Check if the step can be retried
	if !CanRetryStep(failedStep.ID) {
		return ErrNotRetryable
	}

	// Reset the failed step and subsequent steps
	o.tracker.ResetStep(deploymentID, failedStep.ID)

	// Log the retry
	o.logger.Info("retrying deployment",
		"deployment_id", deploymentID,
		"from_step", failedStep.ID,
	)

	// Broadcast retry status
	o.hub.BroadcastLog(deploymentID, "info", failedStep.ID, "Retrying deployment from step: "+failedStep.Name)

	// Create log callback
	logCallback := o.hub.CreateLogCallback(deploymentID)

	// Resume deployment from the failed step
	go func() {
		o.tracker.StartStep(deploymentID, failedStep.ID)
		o.hub.BroadcastStatus(deploymentID, "provisioning")
		statusCallback("provisioning", nil)

		// Run deployment from the failed step
		result, err := o.runner.DeployFromStep(ctx, cfg, failedStep.ID, logCallback, o.tracker)
		if err != nil {
			o.hub.BroadcastStatus(deploymentID, "failed")
			o.hub.BroadcastLog(deploymentID, "error", "failed", err.Error())
			statusCallback("failed", err)
			return
		}

		// Mark deployment as complete
		o.tracker.MarkComplete(deploymentID)

		// Broadcast completion
		o.hub.BroadcastStatus(deploymentID, "completed")
		o.hub.BroadcastLog(deploymentID, "info", "completed",
			"Deployment completed. Control plane IP: "+result.ControlPlaneIP)
		statusCallback("completed", nil)
	}()

	return nil
}

// GetRetryInfo returns information about retry capability for a deployment.
type RetryInfo struct {
	// CanRetry indicates if the deployment can be retried.
	CanRetry bool `json:"can_retry"`
	// FailedStep is the step that failed (if any).
	FailedStep *DeploymentStep `json:"failed_step,omitempty"`
	// FailedStepID is the ID of the failed step.
	FailedStepID string `json:"failed_step_id,omitempty"`
	// Reason explains why retry is or isn't possible.
	Reason string `json:"reason"`
}

// GetRetryInfo returns retry information for a deployment.
func (o *DeploymentOrchestrator) GetRetryInfo(deploymentID uuid.UUID) *RetryInfo {
	progress := o.tracker.GetProgress(deploymentID)
	if progress == nil {
		return &RetryInfo{
			CanRetry: false,
			Reason:   "No progress information found for deployment",
		}
	}

	// Check if any step is in progress
	for _, step := range progress.Steps {
		if step.Status == StepStatusInProgress {
			return &RetryInfo{
				CanRetry: false,
				Reason:   "Deployment is still in progress",
			}
		}
	}

	// Find failed step
	failedStep := FindFailedStep(progress)
	if failedStep == nil {
		// Check if completed
		lastStep := &progress.Steps[len(progress.Steps)-1]
		if lastStep.Status == StepStatusCompleted {
			return &RetryInfo{
				CanRetry: false,
				Reason:   "Deployment completed successfully",
			}
		}
		return &RetryInfo{
			CanRetry: false,
			Reason:   "No failed step found",
		}
	}

	// Check if retryable
	if !CanRetryStep(failedStep.ID) {
		return &RetryInfo{
			CanRetry:     false,
			FailedStep:   failedStep,
			FailedStepID: failedStep.ID,
			Reason:       "The failed step cannot be retried",
		}
	}

	// Check error retryability
	if failedStep.Error != nil && !failedStep.Error.Retryable {
		return &RetryInfo{
			CanRetry:     false,
			FailedStep:   failedStep,
			FailedStepID: failedStep.ID,
			Reason:       failedStep.Error.Suggestions[0],
		}
	}

	return &RetryInfo{
		CanRetry:     true,
		FailedStep:   failedStep,
		FailedStepID: failedStep.ID,
		Reason:       "Deployment can be retried from: " + failedStep.Name,
	}
}
