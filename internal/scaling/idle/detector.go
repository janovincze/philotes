// Package idle provides idle detection for scale-to-zero functionality.
package idle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling"
)

// Detector tracks idle state for scaling policies and determines when they should scale to zero.
type Detector struct {
	repo          Repository
	logger        *slog.Logger
	checkInterval time.Duration

	mu     sync.RWMutex
	states map[uuid.UUID]*scaling.IdleState

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Config holds configuration for the idle detector.
type Config struct {
	// CheckInterval is how often to check and update idle states
	CheckInterval time.Duration

	// DefaultIdleThreshold is the default idle duration before scaling to zero
	DefaultIdleThreshold time.Duration

	// DefaultKeepAliveWindow is the grace period to prevent flapping
	DefaultKeepAliveWindow time.Duration
}

// DefaultConfig returns the default detector configuration.
func DefaultConfig() Config {
	return Config{
		CheckInterval:          1 * time.Minute,
		DefaultIdleThreshold:   30 * time.Minute,
		DefaultKeepAliveWindow: 5 * time.Minute,
	}
}

// NewDetector creates a new idle detector.
func NewDetector(repo Repository, cfg Config, logger *slog.Logger) *Detector {
	if logger == nil {
		logger = slog.Default()
	}

	return &Detector{
		repo:          repo,
		logger:        logger.With("component", "idle-detector"),
		checkInterval: cfg.CheckInterval,
		states:        make(map[uuid.UUID]*scaling.IdleState),
		stopCh:        make(chan struct{}),
	}
}

// Start begins the idle detection loop.
func (d *Detector) Start(ctx context.Context) error {
	d.logger.Info("starting idle detector", "check_interval", d.checkInterval)

	// Load initial states from database
	if err := d.loadStates(ctx); err != nil {
		d.logger.Warn("failed to load initial idle states", "error", err)
	}

	d.wg.Add(1)
	go d.runLoop(ctx)

	return nil
}

// Stop stops the idle detector.
func (d *Detector) Stop() {
	d.logger.Info("stopping idle detector")
	close(d.stopCh)
	d.wg.Wait()
}

// runLoop is the main detection loop.
func (d *Detector) runLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			if err := d.updateIdleStates(ctx); err != nil {
				d.logger.Error("failed to update idle states", "error", err)
			}
		}
	}
}

// loadStates loads all idle states from the database.
func (d *Detector) loadStates(ctx context.Context) error {
	states, err := d.repo.ListIdleStates(ctx)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for i := range states {
		d.states[states[i].PolicyID] = &states[i]
	}

	d.logger.Info("loaded idle states", "count", len(states))
	return nil
}

// updateIdleStates updates all idle states.
func (d *Detector) updateIdleStates(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	for policyID, state := range d.states {
		// Calculate idle duration
		idleDuration := now.Sub(state.LastActivityAt)

		// Update idle_since if becoming idle
		if state.IdleSince == nil && idleDuration > 0 {
			state.IdleSince = &state.LastActivityAt
		}

		// Persist state
		if err := d.repo.UpdateIdleState(ctx, state); err != nil {
			d.logger.Error("failed to update idle state",
				"policy_id", policyID,
				"error", err,
			)
		}
	}

	return nil
}

// RecordActivity records activity for a policy, resetting its idle timer.
func (d *Detector) RecordActivity(ctx context.Context, policyID uuid.UUID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	state, exists := d.states[policyID]
	if !exists {
		// Create new state
		state = &scaling.IdleState{
			PolicyID:       policyID,
			LastActivityAt: now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		d.states[policyID] = state

		// Create in database
		if err := d.repo.CreateIdleState(ctx, state); err != nil {
			return err
		}
	} else {
		// Update existing state
		state.LastActivityAt = now
		state.IdleSince = nil // Reset idle since
		state.UpdatedAt = now

		if err := d.repo.UpdateIdleState(ctx, state); err != nil {
			return err
		}
	}

	d.logger.Debug("recorded activity", "policy_id", policyID)
	return nil
}

// GetIdleState returns the current idle state for a policy.
func (d *Detector) GetIdleState(ctx context.Context, policyID uuid.UUID) (*scaling.IdleState, error) {
	d.mu.RLock()
	state, exists := d.states[policyID]
	d.mu.RUnlock()

	if exists {
		return state, nil
	}

	// Try to load from database
	return d.repo.GetIdleState(ctx, policyID)
}

// IsIdle checks if a policy is considered idle based on its threshold.
func (d *Detector) IsIdle(ctx context.Context, policyID uuid.UUID, threshold time.Duration) (bool, time.Duration, error) {
	state, err := d.GetIdleState(ctx, policyID)
	if err != nil {
		return false, 0, err
	}
	if state == nil {
		return false, 0, nil
	}

	idleDuration := state.IdleDuration()
	return idleDuration >= threshold, idleDuration, nil
}

// MarkScaledToZero marks a policy as scaled to zero.
func (d *Detector) MarkScaledToZero(ctx context.Context, policyID uuid.UUID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	state, exists := d.states[policyID]
	if !exists {
		state = &scaling.IdleState{
			PolicyID:       policyID,
			LastActivityAt: now,
			CreatedAt:      now,
		}
		d.states[policyID] = state
	}

	state.ScaledToZeroAt = &now
	state.IsScaledToZero = true
	state.UpdatedAt = now

	if err := d.repo.UpdateIdleState(ctx, state); err != nil {
		return err
	}

	d.logger.Info("marked policy as scaled to zero", "policy_id", policyID)
	return nil
}

// MarkWoken marks a policy as woken from scaled-to-zero state.
func (d *Detector) MarkWoken(ctx context.Context, policyID uuid.UUID, reason scaling.WakeReason) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	state, exists := d.states[policyID]
	if !exists {
		state = &scaling.IdleState{
			PolicyID:       policyID,
			LastActivityAt: now,
			CreatedAt:      now,
		}
		d.states[policyID] = state
	}

	state.LastWakeAt = &now
	state.WakeReason = &reason
	state.IsScaledToZero = false
	state.ScaledToZeroAt = nil
	state.IdleSince = nil
	state.LastActivityAt = now
	state.UpdatedAt = now

	if err := d.repo.UpdateIdleState(ctx, state); err != nil {
		return err
	}

	d.logger.Info("marked policy as woken",
		"policy_id", policyID,
		"reason", reason,
	)
	return nil
}

// ListScaledToZeroPolicies returns all policies currently scaled to zero.
func (d *Detector) ListScaledToZeroPolicies(ctx context.Context) ([]uuid.UUID, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []uuid.UUID
	for policyID, state := range d.states {
		if state.IsScaledToZero {
			result = append(result, policyID)
		}
	}
	return result, nil
}

// RemovePolicy removes a policy from tracking.
func (d *Detector) RemovePolicy(ctx context.Context, policyID uuid.UUID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.states, policyID)

	if err := d.repo.DeleteIdleState(ctx, policyID); err != nil {
		d.logger.Warn("failed to delete idle state from database",
			"policy_id", policyID,
			"error", err,
		)
	}

	return nil
}
