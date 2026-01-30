package installer

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewProgressTracker(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)

	if tracker == nil {
		t.Fatal("NewProgressTracker returned nil")
	}

	if tracker.progress == nil {
		t.Error("progress map not initialized")
	}

	if tracker.logger == nil {
		t.Error("logger not initialized")
	}
}

func TestProgressTracker_InitProgress(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()

	progress := tracker.InitProgress(deploymentID, "hetzner", 2)

	if progress == nil {
		t.Fatal("InitProgress returned nil")
	}

	if progress.DeploymentID != deploymentID {
		t.Error("deployment ID mismatch")
	}

	if progress.OverallProgress != 0 {
		t.Errorf("initial progress = %d, want 0", progress.OverallProgress)
	}

	if progress.CurrentStepIndex != 0 {
		t.Errorf("initial step index = %d, want 0", progress.CurrentStepIndex)
	}

	if len(progress.Steps) != 10 {
		t.Errorf("steps count = %d, want 10", len(progress.Steps))
	}

	if progress.StartedAt == nil {
		t.Error("StartedAt not set")
	}

	if progress.EstimatedRemainingMs <= 0 {
		t.Error("EstimatedRemainingMs should be positive")
	}

	// Verify progress is stored
	stored := tracker.GetProgress(deploymentID)
	if stored == nil {
		t.Error("progress not stored in tracker")
	}
}

func TestProgressTracker_GetProgress_NotFound(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	progress := tracker.GetProgress(uuid.New())

	if progress != nil {
		t.Error("GetProgress should return nil for unknown deployment")
	}
}

func TestProgressTracker_StartStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "auth")

	progress := tracker.GetProgress(deploymentID)
	if progress == nil {
		t.Fatal("progress not found")
	}

	// Find auth step
	authStep := findStepByID(progress.Steps, "auth")
	if authStep == nil {
		t.Fatal("auth step not found")
	}

	if authStep.Status != StepStatusInProgress {
		t.Errorf("step status = %s, want in_progress", authStep.Status)
	}

	if authStep.StartedAt == nil {
		t.Error("StartedAt not set")
	}

	if progress.CurrentStepIndex != 0 {
		t.Errorf("CurrentStepIndex = %d, want 0", progress.CurrentStepIndex)
	}
}

func TestProgressTracker_StartStep_Unknown(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// This should not panic
	tracker.StartStep(deploymentID, "nonexistent")

	// Verify nothing changed
	progress := tracker.GetProgress(deploymentID)
	for _, step := range progress.Steps {
		if step.Status != StepStatusPending {
			t.Errorf("step %s has unexpected status %s", step.ID, step.Status)
		}
	}
}

func TestProgressTracker_CompleteStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// Start then complete
	tracker.StartStep(deploymentID, "auth")
	time.Sleep(10 * time.Millisecond) // Small delay to get elapsed time
	tracker.CompleteStep(deploymentID, "auth")

	progress := tracker.GetProgress(deploymentID)
	authStep := findStepByID(progress.Steps, "auth")

	if authStep.Status != StepStatusCompleted {
		t.Errorf("step status = %s, want completed", authStep.Status)
	}

	if authStep.CompletedAt == nil {
		t.Error("CompletedAt not set")
	}

	if authStep.ElapsedTimeMs <= 0 {
		t.Error("ElapsedTimeMs should be positive")
	}

	// Progress should have increased
	if progress.OverallProgress == 0 {
		t.Error("OverallProgress should have increased")
	}
}

func TestProgressTracker_CompleteStep_WithSubSteps(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// Network step has sub-steps
	tracker.StartStep(deploymentID, "network")
	tracker.CompleteStep(deploymentID, "network")

	progress := tracker.GetProgress(deploymentID)
	networkStep := findStepByID(progress.Steps, "network")

	// All sub-steps should be marked complete
	for _, subStep := range networkStep.SubSteps {
		if subStep.Status != StepStatusCompleted {
			t.Errorf("sub-step %s has status %s, want completed", subStep.ID, subStep.Status)
		}
	}
}

func TestProgressTracker_FailStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "auth")
	tracker.FailStep(deploymentID, "auth", errors.New("authentication failed"))

	progress := tracker.GetProgress(deploymentID)
	authStep := findStepByID(progress.Steps, "auth")

	if authStep.Status != StepStatusFailed {
		t.Errorf("step status = %s, want failed", authStep.Status)
	}

	if authStep.Error == nil {
		t.Error("Error not set")
	}

	if authStep.Error.Code != "AUTH_FAILED" {
		t.Errorf("Error code = %s, want AUTH_FAILED", authStep.Error.Code)
	}

	// Should be retryable
	if !progress.CanRetry {
		t.Error("CanRetry should be true for auth failures")
	}
}

func TestProgressTracker_FailStep_NonRetryable(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "compute")
	tracker.FailStep(deploymentID, "compute", errors.New("quota exceeded"))

	progress := tracker.GetProgress(deploymentID)

	// Quota exceeded is not retryable
	if progress.CanRetry {
		t.Error("CanRetry should be false for quota errors")
	}
}

func TestProgressTracker_UpdateSubStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "compute")
	tracker.UpdateSubStep(deploymentID, "compute", 0, 1, 1, "Creating control plane")

	progress := tracker.GetProgress(deploymentID)
	computeStep := findStepByID(progress.Steps, "compute")

	if computeStep.CurrentSubStep != 0 {
		t.Errorf("CurrentSubStep = %d, want 0", computeStep.CurrentSubStep)
	}

	if computeStep.SubSteps[0].Status != StepStatusInProgress {
		t.Errorf("sub-step status = %s, want in_progress", computeStep.SubSteps[0].Status)
	}

	if computeStep.SubSteps[0].Details != "Creating control plane" {
		t.Errorf("sub-step details = %s, want 'Creating control plane'", computeStep.SubSteps[0].Details)
	}
}

func TestProgressTracker_UpdateSubStep_MarksPreviousComplete(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "compute")
	tracker.UpdateSubStep(deploymentID, "compute", 0, 1, 1, "Control plane")
	tracker.UpdateSubStep(deploymentID, "compute", 1, 1, 2, "Worker 1")

	progress := tracker.GetProgress(deploymentID)
	computeStep := findStepByID(progress.Steps, "compute")

	// First sub-step should be marked complete
	if computeStep.SubSteps[0].Status != StepStatusCompleted {
		t.Errorf("first sub-step status = %s, want completed", computeStep.SubSteps[0].Status)
	}

	// Current sub-step should be in progress
	if computeStep.SubSteps[1].Status != StepStatusInProgress {
		t.Errorf("second sub-step status = %s, want in_progress", computeStep.SubSteps[1].Status)
	}
}

func TestProgressTracker_CompleteSubStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.StartStep(deploymentID, "compute")
	tracker.UpdateSubStep(deploymentID, "compute", 0, 1, 1, "Control plane")
	tracker.CompleteSubStep(deploymentID, "compute", 0)

	progress := tracker.GetProgress(deploymentID)
	computeStep := findStepByID(progress.Steps, "compute")

	if computeStep.SubSteps[0].Status != StepStatusCompleted {
		t.Errorf("sub-step status = %s, want completed", computeStep.SubSteps[0].Status)
	}
}

func TestProgressTracker_AddResource(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	resource := CreatedResource{
		Type:   "server",
		Name:   "control-plane-1",
		ID:     "srv-12345",
		Region: "nbg1",
	}

	tracker.AddResource(deploymentID, resource)

	progress := tracker.GetProgress(deploymentID)
	if len(progress.ResourcesCreated) != 1 {
		t.Fatalf("ResourcesCreated count = %d, want 1", len(progress.ResourcesCreated))
	}

	if progress.ResourcesCreated[0].Name != "control-plane-1" {
		t.Error("resource name mismatch")
	}
}

func TestProgressTracker_ResetStep(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// Complete first two steps, fail the third
	tracker.StartStep(deploymentID, "auth")
	tracker.CompleteStep(deploymentID, "auth")
	tracker.StartStep(deploymentID, "network")
	tracker.CompleteStep(deploymentID, "network")
	tracker.StartStep(deploymentID, "compute")
	tracker.FailStep(deploymentID, "compute", errors.New("timeout"))

	// Reset from compute
	tracker.ResetStep(deploymentID, "compute")

	progress := tracker.GetProgress(deploymentID)

	// First two should still be complete
	authStep := findStepByID(progress.Steps, "auth")
	if authStep.Status != StepStatusCompleted {
		t.Errorf("auth status = %s, want completed", authStep.Status)
	}

	networkStep := findStepByID(progress.Steps, "network")
	if networkStep.Status != StepStatusCompleted {
		t.Errorf("network status = %s, want completed", networkStep.Status)
	}

	// Compute and later should be pending
	computeStep := findStepByID(progress.Steps, "compute")
	if computeStep.Status != StepStatusPending {
		t.Errorf("compute status = %s, want pending", computeStep.Status)
	}

	if computeStep.Error != nil {
		t.Error("compute error should be cleared")
	}

	// CanRetry should be false after reset
	if progress.CanRetry {
		t.Error("CanRetry should be false after reset")
	}
}

func TestProgressTracker_MarkComplete(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// Complete all steps
	steps := []string{"auth", "network", "compute", "k3s", "storage", "catalog", "philotes", "health", "ssl"}
	for _, stepID := range steps {
		tracker.StartStep(deploymentID, stepID)
		tracker.CompleteStep(deploymentID, stepID)
	}

	tracker.MarkComplete(deploymentID)

	progress := tracker.GetProgress(deploymentID)

	if progress.OverallProgress != 100 {
		t.Errorf("OverallProgress = %d, want 100", progress.OverallProgress)
	}

	if progress.EstimatedRemainingMs != 0 {
		t.Errorf("EstimatedRemainingMs = %d, want 0", progress.EstimatedRemainingMs)
	}

	// Ready step should be complete
	readyStep := findStepByID(progress.Steps, "ready")
	if readyStep.Status != StepStatusCompleted {
		t.Errorf("ready status = %s, want completed", readyStep.Status)
	}
}

func TestProgressTracker_Cleanup(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	tracker.Cleanup(deploymentID)

	progress := tracker.GetProgress(deploymentID)
	if progress != nil {
		t.Error("progress should be nil after cleanup")
	}
}

func TestProgressTracker_GetResourcesForCleanup(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	// Add some resources
	tracker.AddResource(deploymentID, CreatedResource{Type: "server", Name: "server-1"})
	tracker.AddResource(deploymentID, CreatedResource{Type: "network", Name: "network-1"})

	resources := tracker.GetResourcesForCleanup(deploymentID)

	if len(resources) != 2 {
		t.Fatalf("resources count = %d, want 2", len(resources))
	}

	// Should be a copy, not the original
	resources[0].Name = "modified"
	original := tracker.GetProgress(deploymentID).ResourcesCreated
	if original[0].Name == "modified" {
		t.Error("GetResourcesForCleanup should return a copy")
	}
}

func TestProgressTracker_GetResourcesForCleanup_NotFound(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	resources := tracker.GetResourcesForCleanup(uuid.New())

	if resources != nil {
		t.Error("should return nil for unknown deployment")
	}
}

func TestProgressTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	deploymentID := uuid.New()
	tracker.InitProgress(deploymentID, "hetzner", 2)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			tracker.GetProgress(deploymentID)
		}()
	}

	// Concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(n int) {
			defer wg.Done()
			tracker.AddResource(deploymentID, CreatedResource{
				Type: "server",
				Name: "server-" + string(rune(n)),
			})
		}(i)
	}

	wg.Wait()

	// Verify no panics occurred and state is consistent
	progress := tracker.GetProgress(deploymentID)
	if progress == nil {
		t.Error("progress should not be nil")
	}
}

func TestProgressTracker_OperationsOnUnknownDeployment(t *testing.T) {
	tracker := NewProgressTracker(nil, nil)
	unknownID := uuid.New()

	// These should not panic
	tracker.StartStep(unknownID, "auth")
	tracker.CompleteStep(unknownID, "auth")
	tracker.FailStep(unknownID, "auth", errors.New("test"))
	tracker.UpdateSubStep(unknownID, "auth", 0, 1, 1, "test")
	tracker.CompleteSubStep(unknownID, "auth", 0)
	tracker.AddResource(unknownID, CreatedResource{})
	tracker.ResetStep(unknownID, "auth")
	tracker.MarkComplete(unknownID)
	tracker.Cleanup(unknownID)
}
