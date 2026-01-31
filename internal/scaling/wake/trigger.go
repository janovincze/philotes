// Package wake provides wake trigger handling for scale-to-zero functionality.
package wake

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling"
	"github.com/janovincze/philotes/internal/scaling/idle"
)

// Executor defines the interface for executing scale-up operations.
type Executor interface {
	Scale(ctx context.Context, policy *scaling.Policy, targetReplicas int) error
}

// PolicyProvider defines the interface for retrieving policies.
type PolicyProvider interface {
	GetPolicy(ctx context.Context, id uuid.UUID) (*scaling.Policy, error)
	GetPolicyState(ctx context.Context, id uuid.UUID) (*scaling.State, error)
	UpdateState(ctx context.Context, state *scaling.State) error
	RecordHistory(ctx context.Context, history *scaling.History) error
}

// Trigger handles wake operations for scaled-to-zero policies.
type Trigger struct {
	idleDetector   *idle.Detector
	executor       Executor
	policyProvider PolicyProvider
	logger         *slog.Logger

	mu              sync.RWMutex
	pendingWakes    map[uuid.UUID]*Operation
	coldStartConfig ColdStartConfig
}

// ColdStartConfig holds configuration for cold starts.
type ColdStartConfig struct {
	// Timeout is the maximum time to wait for a cold start.
	Timeout time.Duration

	// PollInterval is how often to check if the policy is ready.
	PollInterval time.Duration

	// DefaultReplicas is the default number of replicas to scale to.
	DefaultReplicas int
}

// DefaultColdStartConfig returns the default cold start configuration.
func DefaultColdStartConfig() ColdStartConfig {
	return ColdStartConfig{
		Timeout:         2 * time.Minute,
		PollInterval:    5 * time.Second,
		DefaultReplicas: 1,
	}
}

// Operation represents a pending wake operation.
type Operation struct {
	PolicyID       uuid.UUID
	Reason         scaling.WakeReason
	RequestedAt    time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	TargetReplicas int
	Status         Status
	Error          string
}

// Status represents the status of a wake operation.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Result represents the result of a wake operation.
type Result struct {
	PolicyID              uuid.UUID          `json:"policy_id"`
	PreviousReplicas      int                `json:"previous_replicas"`
	TargetReplicas        int                `json:"target_replicas"`
	Reason                scaling.WakeReason `json:"reason"`
	Status                Status             `json:"status"`
	EstimatedReadySeconds int                `json:"estimated_ready_seconds,omitempty"`
	Message               string             `json:"message"`
	Error                 string             `json:"error,omitempty"`
}

// NewTrigger creates a new wake trigger.
func NewTrigger(
	idleDetector *idle.Detector,
	executor Executor,
	policyProvider PolicyProvider,
	cfg ColdStartConfig,
	logger *slog.Logger,
) *Trigger {
	if logger == nil {
		logger = slog.Default()
	}

	return &Trigger{
		idleDetector:    idleDetector,
		executor:        executor,
		policyProvider:  policyProvider,
		logger:          logger.With("component", "wake-trigger"),
		pendingWakes:    make(map[uuid.UUID]*Operation),
		coldStartConfig: cfg,
	}
}

// Wake initiates a wake operation for a policy.
func (t *Trigger) Wake(ctx context.Context, policyID uuid.UUID, reason scaling.WakeReason) (*Result, error) {
	t.logger.Info("initiating wake operation",
		"policy_id", policyID,
		"reason", reason,
	)

	// Get policy
	policy, err := t.policyProvider.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	if policy == nil {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	// Get current state
	state, err := t.policyProvider.GetPolicyState(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy state: %w", err)
	}

	// Check if already running
	if state != nil && state.CurrentReplicas > 0 {
		return &Result{
			PolicyID:         policyID,
			PreviousReplicas: state.CurrentReplicas,
			TargetReplicas:   state.CurrentReplicas,
			Reason:           reason,
			Status:           StatusCompleted,
			Message:          "Policy is already running",
		}, nil
	}

	// Determine target replicas
	targetReplicas := t.coldStartConfig.DefaultReplicas
	if policy.MinReplicas > 0 {
		targetReplicas = policy.MinReplicas
	}

	// Create wake operation
	now := time.Now()
	op := &Operation{
		PolicyID:       policyID,
		Reason:         reason,
		RequestedAt:    now,
		StartedAt:      &now,
		TargetReplicas: targetReplicas,
		Status:         StatusInProgress,
	}

	t.mu.Lock()
	t.pendingWakes[policyID] = op
	t.mu.Unlock()

	// Execute scaling
	if err := t.executor.Scale(ctx, policy, targetReplicas); err != nil {
		op.Status = StatusFailed
		op.Error = err.Error()
		completed := time.Now()
		op.CompletedAt = &completed

		return &Result{
			PolicyID:         policyID,
			PreviousReplicas: 0,
			TargetReplicas:   targetReplicas,
			Reason:           reason,
			Status:           StatusFailed,
			Message:          "Failed to wake policy",
			Error:            err.Error(),
		}, err
	}

	// Mark as woken in idle detector
	if err := t.idleDetector.MarkWoken(ctx, policyID, reason); err != nil {
		t.logger.Warn("failed to mark policy as woken in idle detector",
			"policy_id", policyID,
			"error", err,
		)
	}

	// Record history
	history := &scaling.History{
		ID:               uuid.New(),
		PolicyID:         &policyID,
		PolicyName:       policy.Name,
		Action:           scaling.ActionScaleUp,
		TargetType:       policy.TargetType,
		TargetID:         policy.TargetID,
		PreviousReplicas: 0,
		NewReplicas:      targetReplicas,
		Reason:           fmt.Sprintf("wake: %s", reason),
		TriggeredBy:      string(reason),
		DryRun:           false,
		ExecutedAt:       now,
	}
	if err := t.policyProvider.RecordHistory(ctx, history); err != nil {
		t.logger.Warn("failed to record wake history",
			"policy_id", policyID,
			"error", err,
		)
	}

	// Update operation status
	completed := time.Now()
	op.Status = StatusCompleted
	op.CompletedAt = &completed

	t.mu.Lock()
	delete(t.pendingWakes, policyID)
	t.mu.Unlock()

	t.logger.Info("wake operation completed",
		"policy_id", policyID,
		"reason", reason,
		"target_replicas", targetReplicas,
	)

	return &Result{
		PolicyID:              policyID,
		PreviousReplicas:      0,
		TargetReplicas:        targetReplicas,
		Reason:                reason,
		Status:                StatusCompleted,
		EstimatedReadySeconds: int(t.coldStartConfig.Timeout.Seconds()),
		Message:               "Wake initiated successfully",
	}, nil
}

// WakeAll wakes all scaled-to-zero policies.
func (t *Trigger) WakeAll(ctx context.Context, reason scaling.WakeReason) ([]Result, error) {
	policies, err := t.idleDetector.ListScaledToZeroPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list scaled-to-zero policies: %w", err)
	}

	results := make([]Result, 0, len(policies))
	for _, policyID := range policies {
		result, err := t.Wake(ctx, policyID, reason)
		if err != nil {
			results = append(results, Result{
				PolicyID: policyID,
				Reason:   reason,
				Status:   StatusFailed,
				Error:    err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}

	return results, nil
}

// WakeMultiple wakes specific policies.
func (t *Trigger) WakeMultiple(ctx context.Context, policyIDs []uuid.UUID, reason scaling.WakeReason) ([]Result, error) {
	results := make([]Result, 0, len(policyIDs))

	for _, policyID := range policyIDs {
		result, err := t.Wake(ctx, policyID, reason)
		if err != nil {
			results = append(results, Result{
				PolicyID: policyID,
				Reason:   reason,
				Status:   StatusFailed,
				Error:    err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}

	return results, nil
}

// GetPendingWake returns a pending wake operation if one exists.
func (t *Trigger) GetPendingWake(policyID uuid.UUID) *Operation {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.pendingWakes[policyID]
}

// IsWaking checks if a policy is currently waking.
func (t *Trigger) IsWaking(policyID uuid.UUID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, exists := t.pendingWakes[policyID]
	return exists
}
