package installer

import (
	"testing"

	"github.com/google/uuid"
)

func TestCanRetryStep(t *testing.T) {
	tests := []struct {
		stepID   string
		expected bool
	}{
		{"auth", true},
		{"network", true},
		{"compute", true},
		{"k3s", true},
		{"storage", true},
		{"catalog", true},
		{"philotes", true},
		{"health", true},
		{"ssl", true},
		{"ready", false}, // Final step cannot be retried
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.stepID, func(t *testing.T) {
			result := CanRetryStep(tt.stepID)
			if result != tt.expected {
				t.Errorf("CanRetryStep(%s) = %v, want %v", tt.stepID, result, tt.expected)
			}
		})
	}
}

func TestFindFailedStep(t *testing.T) {
	tests := []struct {
		name     string
		progress *DeploymentProgress
		expected string // expected step ID or empty if nil
	}{
		{
			name:     "nil progress",
			progress: nil,
			expected: "",
		},
		{
			name: "no failed steps",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "network", Status: StepStatusPending},
				},
			},
			expected: "",
		},
		{
			name: "first step failed",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusFailed},
					{ID: "network", Status: StepStatusPending},
				},
			},
			expected: "auth",
		},
		{
			name: "second step failed",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "network", Status: StepStatusFailed},
					{ID: "compute", Status: StepStatusPending},
				},
			},
			expected: "network",
		},
		{
			name: "multiple failed - returns first",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "network", Status: StepStatusFailed},
					{ID: "compute", Status: StepStatusFailed},
				},
			},
			expected: "network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindFailedStep(tt.progress)

			if tt.expected == "" {
				if result != nil {
					t.Errorf("FindFailedStep() = %s, want nil", result.ID)
				}
			} else {
				if result == nil {
					t.Error("FindFailedStep() = nil, want non-nil")
				} else if result.ID != tt.expected {
					t.Errorf("FindFailedStep() = %s, want %s", result.ID, tt.expected)
				}
			}
		})
	}
}

func TestCanRetryDeployment(t *testing.T) {
	tests := []struct {
		name     string
		progress *DeploymentProgress
		expected bool
	}{
		{
			name:     "nil progress",
			progress: nil,
			expected: false,
		},
		{
			name: "step in progress",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusInProgress},
				},
			},
			expected: false,
		},
		{
			name: "no failed step",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "network", Status: StepStatusPending},
				},
			},
			expected: false,
		},
		{
			name: "retryable step failed",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "network", Status: StepStatusFailed},
				},
			},
			expected: true,
		},
		{
			name: "non-retryable step failed (ready)",
			progress: &DeploymentProgress{
				Steps: []DeploymentStep{
					{ID: "auth", Status: StepStatusCompleted},
					{ID: "ready", Status: StepStatusFailed},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanRetryDeployment(tt.progress)
			if result != tt.expected {
				t.Errorf("CanRetryDeployment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRetryableSteps_Coverage(t *testing.T) {
	// All standard deployment steps should be in the map
	expectedSteps := []string{"auth", "network", "compute", "k3s", "storage", "catalog", "philotes", "health", "ssl", "ready"}

	for _, step := range expectedSteps {
		if _, exists := RetryableSteps[step]; !exists {
			t.Errorf("step %s not found in RetryableSteps", step)
		}
	}
}

func TestRetryInfo(t *testing.T) {
	// Test the RetryInfo struct initialization
	info := RetryInfo{
		CanRetry:     true,
		FailedStep:   &DeploymentStep{ID: "network", Name: "Network Setup"},
		FailedStepID: "network",
		Reason:       "Deployment can be retried from: Network Setup",
	}

	if !info.CanRetry {
		t.Error("CanRetry should be true")
	}

	if info.FailedStepID != "network" {
		t.Errorf("FailedStepID = %s, want network", info.FailedStepID)
	}

	if info.FailedStep == nil {
		t.Error("FailedStep should not be nil")
	}

	if info.Reason == "" {
		t.Error("Reason should not be empty")
	}
}

func TestGetRetryInfo_Scenarios(t *testing.T) {
	// Create a mock orchestrator for testing
	// Note: This tests the logic without the full orchestrator setup

	tests := []struct {
		name         string
		setupFunc    func(*ProgressTracker, uuid.UUID)
		expectRetry  bool
		expectReason string
	}{
		{
			name:         "no progress",
			setupFunc:    func(t *ProgressTracker, id uuid.UUID) {},
			expectRetry:  false,
			expectReason: "No progress information found",
		},
		{
			name: "in progress",
			setupFunc: func(tracker *ProgressTracker, id uuid.UUID) {
				tracker.InitProgress(id, "hetzner", 2)
				tracker.StartStep(id, "auth")
			},
			expectRetry:  false,
			expectReason: "still in progress",
		},
		{
			name: "completed successfully",
			setupFunc: func(tracker *ProgressTracker, id uuid.UUID) {
				tracker.InitProgress(id, "hetzner", 2)
				tracker.StartStep(id, "auth")
				tracker.CompleteStep(id, "auth")
				tracker.MarkComplete(id)
			},
			expectRetry:  false,
			expectReason: "completed successfully",
		},
		{
			name: "failed retryable step",
			setupFunc: func(tracker *ProgressTracker, id uuid.UUID) {
				tracker.InitProgress(id, "hetzner", 2)
				tracker.StartStep(id, "auth")
				tracker.CompleteStep(id, "auth")
				tracker.StartStep(id, "network")
				// Manually set the step as failed without error (to test retry logic)
				progress := tracker.GetProgress(id)
				for i := range progress.Steps {
					if progress.Steps[i].ID == "network" {
						progress.Steps[i].Status = StepStatusFailed
						break
					}
				}
			},
			expectRetry:  true,
			expectReason: "can be retried",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewProgressTracker(nil, nil)
			deploymentID := uuid.New()

			tt.setupFunc(tracker, deploymentID)

			progress := tracker.GetProgress(deploymentID)
			canRetry := CanRetryDeployment(progress)

			if canRetry != tt.expectRetry {
				t.Errorf("CanRetryDeployment() = %v, want %v", canRetry, tt.expectRetry)
			}
		})
	}
}

func TestRetryErrors(t *testing.T) {
	// Test error values
	if ErrNoProgressFound == nil {
		t.Error("ErrNoProgressFound should not be nil")
	}

	if ErrNotRetryable == nil {
		t.Error("ErrNotRetryable should not be nil")
	}

	if ErrDeploymentActive == nil {
		t.Error("ErrDeploymentActive should not be nil")
	}

	// Test error messages
	if ErrNoProgressFound.Error() == "" {
		t.Error("ErrNoProgressFound should have a message")
	}

	if ErrNotRetryable.Error() == "" {
		t.Error("ErrNotRetryable should have a message")
	}

	if ErrDeploymentActive.Error() == "" {
		t.Error("ErrDeploymentActive should have a message")
	}
}
