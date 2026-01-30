package installer

import (
	"strconv"
	"testing"
	"time"
)

func TestGetDeploymentSteps(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		workerCount int
		wantSteps   int
		wantSubStep int // expected sub-steps in compute step
	}{
		{
			name:        "hetzner with 2 workers",
			provider:    "hetzner",
			workerCount: 2,
			wantSteps:   10,
			wantSubStep: 3, // 1 control plane + 2 workers
		},
		{
			name:        "scaleway with 3 workers",
			provider:    "scaleway",
			workerCount: 3,
			wantSteps:   10,
			wantSubStep: 4, // 1 control plane + 3 workers
		},
		{
			name:        "unknown provider defaults to hetzner",
			provider:    "unknown",
			workerCount: 2,
			wantSteps:   10,
			wantSubStep: 3,
		},
		{
			name:        "zero workers defaults to 2",
			provider:    "hetzner",
			workerCount: 0,
			wantSteps:   10,
			wantSubStep: 3, // 1 control plane + 2 default workers
		},
		{
			name:        "negative workers defaults to 2",
			provider:    "hetzner",
			workerCount: -1,
			wantSteps:   10,
			wantSubStep: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := GetDeploymentSteps(tt.provider, tt.workerCount)

			if len(steps) != tt.wantSteps {
				t.Errorf("GetDeploymentSteps() returned %d steps, want %d", len(steps), tt.wantSteps)
			}

			// Check compute step sub-steps
			computeStep := findStepByID(steps, "compute")
			if computeStep == nil {
				t.Fatal("compute step not found")
			}

			if len(computeStep.SubSteps) != tt.wantSubStep {
				t.Errorf("compute step has %d sub-steps, want %d", len(computeStep.SubSteps), tt.wantSubStep)
			}

			// Verify all steps have required fields
			for _, step := range steps {
				if step.ID == "" {
					t.Error("step has empty ID")
				}
				if step.Name == "" {
					t.Error("step has empty Name")
				}
				if step.Description == "" {
					t.Error("step has empty Description")
				}
				if step.Status != StepStatusPending {
					t.Errorf("step %s has status %s, want pending", step.ID, step.Status)
				}
			}
		})
	}
}

func TestGetDeploymentSteps_StepOrder(t *testing.T) {
	steps := GetDeploymentSteps("hetzner", 2)

	expectedOrder := []string{"auth", "network", "compute", "k3s", "storage", "catalog", "philotes", "health", "ssl", "ready"}

	if len(steps) != len(expectedOrder) {
		t.Fatalf("expected %d steps, got %d", len(expectedOrder), len(steps))
	}

	for i, expected := range expectedOrder {
		if steps[i].ID != expected {
			t.Errorf("step %d: expected ID %s, got %s", i, expected, steps[i].ID)
		}
	}
}

func TestGetDeploymentSteps_ProviderTimeEstimates(t *testing.T) {
	providers := []string{"hetzner", "scaleway", "ovh", "exoscale", "contabo"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			steps := GetDeploymentSteps(provider, 2)

			// Auth step should be 5 seconds for all providers
			authStep := findStepByID(steps, "auth")
			if authStep == nil {
				t.Fatal("auth step not found")
			}
			if authStep.EstimatedTimeMs != 5000 {
				t.Errorf("auth step estimated time: got %d, want 5000", authStep.EstimatedTimeMs)
			}

			// All steps should have non-negative time estimates (except ready which is 0)
			for _, step := range steps {
				if step.EstimatedTimeMs < 0 {
					t.Errorf("step %s has negative estimated time: %d", step.ID, step.EstimatedTimeMs)
				}
				if step.ID == "ready" && step.EstimatedTimeMs != 0 {
					t.Errorf("ready step should have 0 estimated time, got %d", step.EstimatedTimeMs)
				}
			}
		})
	}
}

func TestCalculateTotalEstimate(t *testing.T) {
	steps := []DeploymentStep{
		{ID: "step1", EstimatedTimeMs: 1000},
		{ID: "step2", EstimatedTimeMs: 2000},
		{ID: "step3", EstimatedTimeMs: 3000},
	}

	total := CalculateTotalEstimate(steps)
	expected := int64(6000)

	if total != expected {
		t.Errorf("CalculateTotalEstimate() = %d, want %d", total, expected)
	}
}

func TestCalculateTotalEstimate_Empty(t *testing.T) {
	total := CalculateTotalEstimate([]DeploymentStep{})
	if total != 0 {
		t.Errorf("CalculateTotalEstimate(empty) = %d, want 0", total)
	}
}

func TestCalculateOverallProgress(t *testing.T) {
	tests := []struct {
		name     string
		steps    []DeploymentStep
		expected int
	}{
		{
			name:     "empty steps",
			steps:    []DeploymentStep{},
			expected: 0,
		},
		{
			name: "no completed steps",
			steps: []DeploymentStep{
				{Status: StepStatusPending},
				{Status: StepStatusPending},
			},
			expected: 0,
		},
		{
			name: "one of two completed",
			steps: []DeploymentStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusPending},
			},
			expected: 50,
		},
		{
			name: "all completed",
			steps: []DeploymentStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
			},
			expected: 100,
		},
		{
			name: "skipped counts as completed",
			steps: []DeploymentStep{
				{Status: StepStatusSkipped},
				{Status: StepStatusPending},
			},
			expected: 50,
		},
		{
			name: "in_progress does not count",
			steps: []DeploymentStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusInProgress},
				{Status: StepStatusPending},
			},
			expected: 33, // 1 of 3
		},
		{
			name: "three of four completed",
			steps: []DeploymentStep{
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
				{Status: StepStatusCompleted},
				{Status: StepStatusPending},
			},
			expected: 75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateOverallProgress(tt.steps)
			if result != tt.expected {
				t.Errorf("CalculateOverallProgress() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCalculateRemainingTime(t *testing.T) {
	now := time.Now()
	started := now.Add(-5 * time.Second) // started 5 seconds ago

	tests := []struct {
		name             string
		steps            []DeploymentStep
		currentStepIndex int
		expected         int64
	}{
		{
			name: "all pending",
			steps: []DeploymentStep{
				{Status: StepStatusPending, EstimatedTimeMs: 10000},
				{Status: StepStatusPending, EstimatedTimeMs: 20000},
			},
			currentStepIndex: 0,
			expected:         30000,
		},
		{
			name: "first completed",
			steps: []DeploymentStep{
				{Status: StepStatusCompleted, EstimatedTimeMs: 10000},
				{Status: StepStatusPending, EstimatedTimeMs: 20000},
			},
			currentStepIndex: 1,
			expected:         20000,
		},
		{
			name: "in progress with elapsed time",
			steps: []DeploymentStep{
				{Status: StepStatusInProgress, EstimatedTimeMs: 10000, StartedAt: &started},
				{Status: StepStatusPending, EstimatedTimeMs: 20000},
			},
			currentStepIndex: 0,
			expected:         25000, // ~5000 remaining from first + 20000 from second
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRemainingTime(tt.steps, tt.currentStepIndex)
			// Allow some tolerance for timing
			tolerance := int64(1000) // 1 second
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				t.Errorf("CalculateRemainingTime() = %d, want approximately %d", result, tt.expected)
			}
		})
	}
}

func TestStepIDToIndex(t *testing.T) {
	steps := []DeploymentStep{
		{ID: "auth"},
		{ID: "network"},
		{ID: "compute"},
	}

	tests := []struct {
		stepID   string
		expected int
	}{
		{"auth", 0},
		{"network", 1},
		{"compute", 2},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		t.Run(tt.stepID, func(t *testing.T) {
			result := StepIDToIndex(steps, tt.stepID)
			if result != tt.expected {
				t.Errorf("StepIDToIndex(%s) = %d, want %d", tt.stepID, result, tt.expected)
			}
		})
	}
}

func TestGenerateComputeSubSteps(t *testing.T) {
	tests := []struct {
		workerCount int
		expected    int // total sub-steps (1 control plane + workers)
	}{
		{1, 2},
		{2, 3},
		{5, 6},
	}

	for _, tt := range tests {
		t.Run("workers="+strconv.Itoa(tt.workerCount), func(t *testing.T) {
			subSteps := generateComputeSubSteps(tt.workerCount)
			if len(subSteps) != tt.expected {
				t.Errorf("generateComputeSubSteps(%d) returned %d sub-steps, want %d",
					tt.workerCount, len(subSteps), tt.expected)
			}

			// First sub-step should be control plane
			if subSteps[0].ID != "control-plane" {
				t.Errorf("first sub-step ID = %s, want control-plane", subSteps[0].ID)
			}

			// Check worker sub-steps
			for i := 1; i < len(subSteps); i++ {
				if subSteps[i].Current != i {
					t.Errorf("worker %d has Current = %d, want %d", i, subSteps[i].Current, i)
				}
				if subSteps[i].Total != tt.workerCount {
					t.Errorf("worker %d has Total = %d, want %d", i, subSteps[i].Total, tt.workerCount)
				}
			}
		})
	}
}

// Helper function
func findStepByID(steps []DeploymentStep, id string) *DeploymentStep {
	for i := range steps {
		if steps[i].ID == id {
			return &steps[i]
		}
	}
	return nil
}
