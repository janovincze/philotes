// Package installer provides deployment progress tracking.
package installer

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProgressTracker manages deployment progress state.
type ProgressTracker struct {
	// progress maps deployment IDs to their progress state.
	progress map[uuid.UUID]*DeploymentProgress
	// mu protects the progress map.
	mu sync.RWMutex
	// hub is the WebSocket hub for broadcasting updates.
	hub *LogHub
	// logger is the structured logger.
	logger *slog.Logger
}

// NewProgressTracker creates a new ProgressTracker.
func NewProgressTracker(hub *LogHub, logger *slog.Logger) *ProgressTracker {
	if logger == nil {
		logger = slog.Default()
	}

	return &ProgressTracker{
		progress: make(map[uuid.UUID]*DeploymentProgress),
		hub:      hub,
		logger:   logger.With("component", "progress-tracker"),
	}
}

// InitProgress initializes progress tracking for a deployment.
func (t *ProgressTracker) InitProgress(deploymentID uuid.UUID, provider string, workerCount int) *DeploymentProgress {
	t.mu.Lock()
	defer t.mu.Unlock()

	steps := GetDeploymentSteps(provider, workerCount)
	now := time.Now()

	progress := &DeploymentProgress{
		DeploymentID:         deploymentID,
		OverallProgress:      0,
		CurrentStepIndex:     0,
		Steps:                steps,
		StartedAt:            &now,
		EstimatedRemainingMs: CalculateTotalEstimate(steps),
		CanRetry:             false,
		ResourcesCreated:     make([]CreatedResource, 0),
	}

	t.progress[deploymentID] = progress

	t.logger.Debug("initialized progress tracking",
		"deployment_id", deploymentID,
		"provider", provider,
		"worker_count", workerCount,
		"total_steps", len(steps),
	)

	// Broadcast initial progress
	t.broadcastProgress(deploymentID, progress)

	return progress
}

// GetProgress returns the current progress for a deployment.
func (t *ProgressTracker) GetProgress(deploymentID uuid.UUID) *DeploymentProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.progress[deploymentID]
}

// StartStep marks a step as in-progress.
func (t *ProgressTracker) StartStep(deploymentID uuid.UUID, stepID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		t.logger.Warn("no progress found for deployment", "deployment_id", deploymentID)
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		t.logger.Warn("step not found", "step_id", stepID)
		return
	}

	now := time.Now()
	progress.Steps[stepIndex].Status = StepStatusInProgress
	progress.Steps[stepIndex].StartedAt = &now
	progress.CurrentStepIndex = stepIndex

	// Update overall progress
	progress.OverallProgress = CalculateOverallProgress(progress.Steps)
	progress.EstimatedRemainingMs = CalculateRemainingTime(progress.Steps, stepIndex)

	t.logger.Debug("started step",
		"deployment_id", deploymentID,
		"step_id", stepID,
		"step_index", stepIndex,
	)

	// Broadcast step update
	t.broadcastStepUpdate(deploymentID, &progress.Steps[stepIndex])
	t.broadcastProgress(deploymentID, progress)
}

// CompleteStep marks a step as completed.
func (t *ProgressTracker) CompleteStep(deploymentID uuid.UUID, stepID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		return
	}

	now := time.Now()
	step := &progress.Steps[stepIndex]
	step.Status = StepStatusCompleted
	step.CompletedAt = &now

	// Calculate elapsed time
	if step.StartedAt != nil {
		step.ElapsedTimeMs = now.Sub(*step.StartedAt).Milliseconds()
	}

	// Mark all sub-steps as completed
	for i := range step.SubSteps {
		step.SubSteps[i].Status = StepStatusCompleted
	}

	// Update overall progress
	progress.OverallProgress = CalculateOverallProgress(progress.Steps)
	progress.EstimatedRemainingMs = CalculateRemainingTime(progress.Steps, progress.CurrentStepIndex)

	t.logger.Debug("completed step",
		"deployment_id", deploymentID,
		"step_id", stepID,
		"elapsed_ms", step.ElapsedTimeMs,
	)

	// Broadcast updates
	t.broadcastStepUpdate(deploymentID, step)
	t.broadcastProgress(deploymentID, progress)
}

// FailStep marks a step as failed with error information.
func (t *ProgressTracker) FailStep(deploymentID uuid.UUID, stepID string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		return
	}

	now := time.Now()
	step := &progress.Steps[stepIndex]
	step.Status = StepStatusFailed
	step.CompletedAt = &now

	// Calculate elapsed time
	if step.StartedAt != nil {
		step.ElapsedTimeMs = now.Sub(*step.StartedAt).Milliseconds()
	}

	// Get error suggestions
	step.Error = GetErrorSuggestion(err, stepID)

	// Check if retryable
	progress.CanRetry = step.Error != nil && step.Error.Retryable

	t.logger.Debug("step failed",
		"deployment_id", deploymentID,
		"step_id", stepID,
		"error", err,
		"retryable", progress.CanRetry,
	)

	// Broadcast updates
	t.broadcastStepUpdate(deploymentID, step)
	if step.Error != nil {
		t.broadcastError(deploymentID, stepID, step.Error)
	}
}

// UpdateSubStep updates sub-step progress.
func (t *ProgressTracker) UpdateSubStep(deploymentID uuid.UUID, stepID string, subStepIndex int, current, total int, details string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		return
	}

	step := &progress.Steps[stepIndex]
	if subStepIndex < 0 || subStepIndex >= len(step.SubSteps) {
		return
	}

	// Update current sub-step
	step.CurrentSubStep = subStepIndex
	step.SubSteps[subStepIndex].Status = StepStatusInProgress
	step.SubSteps[subStepIndex].Current = current
	step.SubSteps[subStepIndex].Total = total
	step.SubSteps[subStepIndex].Details = details

	// Mark previous sub-steps as completed
	for i := 0; i < subStepIndex; i++ {
		step.SubSteps[i].Status = StepStatusCompleted
	}

	t.logger.Debug("updated sub-step",
		"deployment_id", deploymentID,
		"step_id", stepID,
		"sub_step_index", subStepIndex,
		"current", current,
		"total", total,
	)

	// Broadcast step update
	t.broadcastStepUpdate(deploymentID, step)
}

// CompleteSubStep marks a sub-step as completed.
func (t *ProgressTracker) CompleteSubStep(deploymentID uuid.UUID, stepID string, subStepIndex int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		return
	}

	step := &progress.Steps[stepIndex]
	if subStepIndex < 0 || subStepIndex >= len(step.SubSteps) {
		return
	}

	step.SubSteps[subStepIndex].Status = StepStatusCompleted

	// Broadcast step update
	t.broadcastStepUpdate(deploymentID, step)
}

// AddResource records a created resource.
func (t *ProgressTracker) AddResource(deploymentID uuid.UUID, resource CreatedResource) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	progress.ResourcesCreated = append(progress.ResourcesCreated, resource)

	t.logger.Debug("added resource",
		"deployment_id", deploymentID,
		"resource_type", resource.Type,
		"resource_name", resource.Name,
	)
}

// ResetStep resets a step and all following steps to pending status.
func (t *ProgressTracker) ResetStep(deploymentID uuid.UUID, stepID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	stepIndex := StepIDToIndex(progress.Steps, stepID)
	if stepIndex < 0 {
		return
	}

	// Reset this step and all following steps
	for i := stepIndex; i < len(progress.Steps); i++ {
		step := &progress.Steps[i]
		step.Status = StepStatusPending
		step.StartedAt = nil
		step.CompletedAt = nil
		step.ElapsedTimeMs = 0
		step.Error = nil
		step.CurrentSubStep = 0

		// Reset sub-steps
		for j := range step.SubSteps {
			step.SubSteps[j].Status = StepStatusPending
			step.SubSteps[j].Details = ""
		}
	}

	// Update current step index
	progress.CurrentStepIndex = stepIndex
	progress.CanRetry = false

	// Recalculate progress
	progress.OverallProgress = CalculateOverallProgress(progress.Steps)
	progress.EstimatedRemainingMs = CalculateRemainingTime(progress.Steps, stepIndex)

	t.logger.Debug("reset step",
		"deployment_id", deploymentID,
		"step_id", stepID,
		"from_index", stepIndex,
	)

	// Broadcast progress
	t.broadcastProgress(deploymentID, progress)
}

// MarkComplete marks the deployment as complete.
func (t *ProgressTracker) MarkComplete(deploymentID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return
	}

	now := time.Now()

	// Mark the "ready" step as completed
	readyIndex := StepIDToIndex(progress.Steps, "ready")
	if readyIndex >= 0 {
		progress.Steps[readyIndex].Status = StepStatusCompleted
		progress.Steps[readyIndex].CompletedAt = &now
	}

	progress.OverallProgress = 100
	progress.EstimatedRemainingMs = 0
	progress.CurrentStepIndex = len(progress.Steps) - 1

	t.logger.Info("deployment completed",
		"deployment_id", deploymentID,
		"resources_created", len(progress.ResourcesCreated),
	)

	// Broadcast final progress
	t.broadcastProgress(deploymentID, progress)
}

// Cleanup removes progress tracking for a deployment.
func (t *ProgressTracker) Cleanup(deploymentID uuid.UUID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.progress, deploymentID)

	t.logger.Debug("cleaned up progress tracking", "deployment_id", deploymentID)
}

// GetResourcesForCleanup returns the resources that would be cleaned up.
func (t *ProgressTracker) GetResourcesForCleanup(deploymentID uuid.UUID) []CreatedResource {
	t.mu.RLock()
	defer t.mu.RUnlock()

	progress := t.progress[deploymentID]
	if progress == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	resources := make([]CreatedResource, len(progress.ResourcesCreated))
	copy(resources, progress.ResourcesCreated)
	return resources
}

// broadcastProgress sends a progress update via WebSocket.
func (t *ProgressTracker) broadcastProgress(deploymentID uuid.UUID, progress *DeploymentProgress) {
	if t.hub == nil {
		return
	}

	t.hub.BroadcastProgress(deploymentID, &ProgressUpdate{
		OverallPercent:       progress.OverallProgress,
		CurrentStepIndex:     progress.CurrentStepIndex,
		EstimatedRemainingMs: progress.EstimatedRemainingMs,
	})
}

// broadcastStepUpdate sends a step update via WebSocket.
func (t *ProgressTracker) broadcastStepUpdate(deploymentID uuid.UUID, step *DeploymentStep) {
	if t.hub == nil {
		return
	}

	update := &StepUpdate{
		StepID:        step.ID,
		Status:        step.Status,
		ElapsedTimeMs: step.ElapsedTimeMs,
	}

	if len(step.SubSteps) > 0 && step.CurrentSubStep < len(step.SubSteps) {
		subStep := step.SubSteps[step.CurrentSubStep]
		update.SubStepIndex = step.CurrentSubStep
		update.SubStepCurrent = subStep.Current
		update.SubStepTotal = subStep.Total
	}

	t.hub.BroadcastStepUpdate(deploymentID, update)
}

// broadcastError sends an error with suggestions via WebSocket.
func (t *ProgressTracker) broadcastError(deploymentID uuid.UUID, stepID string, err *StepError) {
	if t.hub == nil {
		return
	}

	t.hub.BroadcastErrorWithSuggestions(deploymentID, stepID, err)
}
