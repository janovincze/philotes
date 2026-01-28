// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Executor defines the interface for executing scaling actions.
type Executor interface {
	// GetCurrentReplicas returns the current replica count for a target.
	GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error)

	// Scale scales the target to the desired number of replicas.
	Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error

	// Name returns the executor name for logging purposes.
	Name() string
}

// KEDAExecutor is a stub implementation of the Executor interface for KEDA.
// In a real implementation, this would connect to the Kubernetes API to modify ScaledObject resources.
type KEDAExecutor struct {
	logger *slog.Logger

	// Simulated replica counts for testing
	replicas map[string]int
	mu       sync.RWMutex
}

// NewKEDAExecutor creates a new KEDA executor stub.
func NewKEDAExecutor(logger *slog.Logger) *KEDAExecutor {
	if logger == nil {
		logger = slog.Default()
	}

	return &KEDAExecutor{
		logger:   logger.With("component", "keda-executor"),
		replicas: make(map[string]int),
	}
}

// Name returns the executor name.
func (e *KEDAExecutor) Name() string {
	return "keda"
}

// GetCurrentReplicas returns the current replica count for a target.
// This is a stub implementation that returns simulated values.
func (e *KEDAExecutor) GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error) {
	key := e.targetKey(targetType, targetID)

	e.mu.RLock()
	defer e.mu.RUnlock()

	if replicas, ok := e.replicas[key]; ok {
		return replicas, nil
	}

	// Default to 1 replica if not set
	e.logger.Debug("no replica count found, defaulting to 1",
		"target_type", targetType,
		"target_id", targetID,
	)
	return 1, nil
}

// Scale scales the target to the desired number of replicas.
// This is a stub implementation that logs the action but doesn't actually scale.
func (e *KEDAExecutor) Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error {
	key := e.targetKey(targetType, targetID)

	e.mu.Lock()
	defer e.mu.Unlock()

	oldReplicas := e.replicas[key]
	if oldReplicas == 0 {
		oldReplicas = 1
	}

	if dryRun {
		e.logger.Info("[DRY-RUN] would scale target",
			"target_type", targetType,
			"target_id", targetID,
			"from_replicas", oldReplicas,
			"to_replicas", replicas,
		)
		return nil
	}

	// TODO: In a real implementation, this would:
	// 1. Connect to the Kubernetes API
	// 2. Find the ScaledObject for this target
	// 3. Update the minReplicaCount/maxReplicaCount or the target deployment's replicas

	e.logger.Info("scaling target (stub)",
		"target_type", targetType,
		"target_id", targetID,
		"from_replicas", oldReplicas,
		"to_replicas", replicas,
	)

	// Update simulated state
	e.replicas[key] = replicas

	return nil
}

// SetReplicas sets the simulated replica count for testing.
func (e *KEDAExecutor) SetReplicas(targetType TargetType, targetID *uuid.UUID, replicas int) {
	key := e.targetKey(targetType, targetID)

	e.mu.Lock()
	defer e.mu.Unlock()

	e.replicas[key] = replicas
}

// targetKey generates a unique key for a target.
func (e *KEDAExecutor) targetKey(targetType TargetType, targetID *uuid.UUID) string {
	if targetID != nil {
		return fmt.Sprintf("%s:%s", targetType, targetID.String())
	}
	return string(targetType)
}

// LoggingExecutor is an executor that only logs actions.
// Useful for dry-run mode and testing.
type LoggingExecutor struct {
	logger   *slog.Logger
	replicas map[string]int
	mu       sync.RWMutex
}

// NewLoggingExecutor creates a new logging executor.
func NewLoggingExecutor(logger *slog.Logger) *LoggingExecutor {
	if logger == nil {
		logger = slog.Default()
	}

	return &LoggingExecutor{
		logger:   logger.With("component", "logging-executor"),
		replicas: make(map[string]int),
	}
}

// Name returns the executor name.
func (e *LoggingExecutor) Name() string {
	return "logging"
}

// GetCurrentReplicas returns the current replica count for a target.
func (e *LoggingExecutor) GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error) {
	key := e.targetKey(targetType, targetID)

	e.mu.RLock()
	defer e.mu.RUnlock()

	if replicas, ok := e.replicas[key]; ok {
		return replicas, nil
	}

	return 1, nil
}

// Scale logs the scaling action without actually scaling.
func (e *LoggingExecutor) Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error {
	key := e.targetKey(targetType, targetID)

	e.mu.Lock()
	defer e.mu.Unlock()

	oldReplicas := e.replicas[key]
	if oldReplicas == 0 {
		oldReplicas = 1
	}

	prefix := ""
	if dryRun {
		prefix = "[DRY-RUN] "
	}

	e.logger.Info(prefix+"scaling action",
		"target_type", targetType,
		"target_id", targetID,
		"from_replicas", oldReplicas,
		"to_replicas", replicas,
		"dry_run", dryRun,
	)

	// Update state even for logging executor to track changes
	e.replicas[key] = replicas

	return nil
}

// SetReplicas sets the simulated replica count for testing.
func (e *LoggingExecutor) SetReplicas(targetType TargetType, targetID *uuid.UUID, replicas int) {
	key := e.targetKey(targetType, targetID)

	e.mu.Lock()
	defer e.mu.Unlock()

	e.replicas[key] = replicas
}

// targetKey generates a unique key for a target.
func (e *LoggingExecutor) targetKey(targetType TargetType, targetID *uuid.UUID) string {
	if targetID != nil {
		return fmt.Sprintf("%s:%s", targetType, targetID.String())
	}
	return string(targetType)
}

// CompositeExecutor combines multiple executors and runs them in sequence.
// Useful for combining a real executor with a logging executor.
type CompositeExecutor struct {
	executors []Executor
	logger    *slog.Logger
}

// NewCompositeExecutor creates a new composite executor.
func NewCompositeExecutor(executors []Executor, logger *slog.Logger) *CompositeExecutor {
	if logger == nil {
		logger = slog.Default()
	}

	return &CompositeExecutor{
		executors: executors,
		logger:    logger.With("component", "composite-executor"),
	}
}

// Name returns the executor name.
func (e *CompositeExecutor) Name() string {
	return "composite"
}

// GetCurrentReplicas returns the current replica count from the first executor.
func (e *CompositeExecutor) GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error) {
	if len(e.executors) == 0 {
		return 0, fmt.Errorf("no executors configured")
	}

	return e.executors[0].GetCurrentReplicas(ctx, targetType, targetID)
}

// Scale runs scaling action on all executors.
func (e *CompositeExecutor) Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error {
	var lastErr error

	for _, executor := range e.executors {
		if err := executor.Scale(ctx, targetType, targetID, replicas, dryRun); err != nil {
			e.logger.Error("executor failed",
				"executor", executor.Name(),
				"error", err,
			)
			lastErr = err
		}
	}

	return lastErr
}
