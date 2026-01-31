// Package idle provides idle detection for scale-to-zero functionality.
package idle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling"
)

// Repository defines the interface for idle state persistence.
type Repository interface {
	// CreateIdleState creates a new idle state record.
	CreateIdleState(ctx context.Context, state *scaling.IdleState) error

	// GetIdleState retrieves an idle state by policy ID.
	GetIdleState(ctx context.Context, policyID uuid.UUID) (*scaling.IdleState, error)

	// UpdateIdleState updates an existing idle state.
	UpdateIdleState(ctx context.Context, state *scaling.IdleState) error

	// DeleteIdleState deletes an idle state by policy ID.
	DeleteIdleState(ctx context.Context, policyID uuid.UUID) error

	// ListIdleStates returns all idle states.
	ListIdleStates(ctx context.Context) ([]scaling.IdleState, error)

	// ListScaledToZero returns all policies currently scaled to zero.
	ListScaledToZero(ctx context.Context) ([]scaling.IdleState, error)

	// RecordCostSavings records cost savings for a policy.
	RecordCostSavings(ctx context.Context, savings *scaling.CostSavings) error

	// GetCostSavings retrieves cost savings for a policy within a date range.
	GetCostSavings(ctx context.Context, policyID uuid.UUID, startDate, endDate time.Time) ([]scaling.CostSavings, error)

	// GetTotalSavings retrieves total cost savings for a policy.
	GetTotalSavings(ctx context.Context, policyID uuid.UUID) (*SavingsSummary, error)
}

// SavingsSummary represents aggregated cost savings.
type SavingsSummary struct {
	TotalIdleSeconds         int64
	TotalScaledToZeroSeconds int64
	TotalSavingsCents        int
	DayCount                 int
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateIdleState creates a new idle state record.
func (r *PostgresRepository) CreateIdleState(ctx context.Context, state *scaling.IdleState) error {
	if state.ID == uuid.Nil {
		state.ID = uuid.New()
	}

	query := `
		INSERT INTO scaling_idle_state (
			id, policy_id, last_activity_at, idle_since, scaled_to_zero_at,
			last_wake_at, wake_reason, is_scaled_to_zero, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (policy_id) DO UPDATE SET
			last_activity_at = EXCLUDED.last_activity_at,
			idle_since = EXCLUDED.idle_since,
			scaled_to_zero_at = EXCLUDED.scaled_to_zero_at,
			last_wake_at = EXCLUDED.last_wake_at,
			wake_reason = EXCLUDED.wake_reason,
			is_scaled_to_zero = EXCLUDED.is_scaled_to_zero,
			updated_at = EXCLUDED.updated_at
	`

	var wakeReason *string
	if state.WakeReason != nil {
		s := state.WakeReason.String()
		wakeReason = &s
	}

	_, err := r.db.ExecContext(ctx, query,
		state.ID,
		state.PolicyID,
		state.LastActivityAt,
		state.IdleSince,
		state.ScaledToZeroAt,
		state.LastWakeAt,
		wakeReason,
		state.IsScaledToZero,
		state.CreatedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create idle state: %w", err)
	}

	return nil
}

// GetIdleState retrieves an idle state by policy ID.
func (r *PostgresRepository) GetIdleState(ctx context.Context, policyID uuid.UUID) (*scaling.IdleState, error) {
	query := `
		SELECT id, policy_id, last_activity_at, idle_since, scaled_to_zero_at,
			   last_wake_at, wake_reason, is_scaled_to_zero, created_at, updated_at
		FROM scaling_idle_state
		WHERE policy_id = $1
	`

	var state scaling.IdleState
	var wakeReason *string

	err := r.db.QueryRowContext(ctx, query, policyID).Scan(
		&state.ID,
		&state.PolicyID,
		&state.LastActivityAt,
		&state.IdleSince,
		&state.ScaledToZeroAt,
		&state.LastWakeAt,
		&wakeReason,
		&state.IsScaledToZero,
		&state.CreatedAt,
		&state.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idle state: %w", err)
	}

	if wakeReason != nil {
		wr := scaling.WakeReason(*wakeReason)
		state.WakeReason = &wr
	}

	return &state, nil
}

// UpdateIdleState updates an existing idle state.
func (r *PostgresRepository) UpdateIdleState(ctx context.Context, state *scaling.IdleState) error {
	query := `
		UPDATE scaling_idle_state SET
			last_activity_at = $2,
			idle_since = $3,
			scaled_to_zero_at = $4,
			last_wake_at = $5,
			wake_reason = $6,
			is_scaled_to_zero = $7,
			updated_at = $8
		WHERE policy_id = $1
	`

	var wakeReason *string
	if state.WakeReason != nil {
		s := state.WakeReason.String()
		wakeReason = &s
	}

	result, err := r.db.ExecContext(ctx, query,
		state.PolicyID,
		state.LastActivityAt,
		state.IdleSince,
		state.ScaledToZeroAt,
		state.LastWakeAt,
		wakeReason,
		state.IsScaledToZero,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update idle state: %w", err)
	}

	rowsAffected, _ := result.RowsAffected() //nolint:errcheck // error is not critical for update check
	if rowsAffected == 0 {
		// Create if doesn't exist
		return r.CreateIdleState(ctx, state)
	}

	return nil
}

// DeleteIdleState deletes an idle state by policy ID.
func (r *PostgresRepository) DeleteIdleState(ctx context.Context, policyID uuid.UUID) error {
	query := `DELETE FROM scaling_idle_state WHERE policy_id = $1`

	_, err := r.db.ExecContext(ctx, query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete idle state: %w", err)
	}

	return nil
}

// ListIdleStates returns all idle states.
func (r *PostgresRepository) ListIdleStates(ctx context.Context) ([]scaling.IdleState, error) {
	query := `
		SELECT id, policy_id, last_activity_at, idle_since, scaled_to_zero_at,
			   last_wake_at, wake_reason, is_scaled_to_zero, created_at, updated_at
		FROM scaling_idle_state
		ORDER BY updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list idle states: %w", err)
	}
	defer rows.Close()

	var states []scaling.IdleState
	for rows.Next() {
		var state scaling.IdleState
		var wakeReason *string

		if err := rows.Scan(
			&state.ID,
			&state.PolicyID,
			&state.LastActivityAt,
			&state.IdleSince,
			&state.ScaledToZeroAt,
			&state.LastWakeAt,
			&wakeReason,
			&state.IsScaledToZero,
			&state.CreatedAt,
			&state.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan idle state: %w", err)
		}

		if wakeReason != nil {
			wr := scaling.WakeReason(*wakeReason)
			state.WakeReason = &wr
		}

		states = append(states, state)
	}

	return states, rows.Err()
}

// ListScaledToZero returns all policies currently scaled to zero.
func (r *PostgresRepository) ListScaledToZero(ctx context.Context) ([]scaling.IdleState, error) {
	query := `
		SELECT id, policy_id, last_activity_at, idle_since, scaled_to_zero_at,
			   last_wake_at, wake_reason, is_scaled_to_zero, created_at, updated_at
		FROM scaling_idle_state
		WHERE is_scaled_to_zero = TRUE
		ORDER BY scaled_to_zero_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scaled-to-zero policies: %w", err)
	}
	defer rows.Close()

	var states []scaling.IdleState
	for rows.Next() {
		var state scaling.IdleState
		var wakeReason *string

		if err := rows.Scan(
			&state.ID,
			&state.PolicyID,
			&state.LastActivityAt,
			&state.IdleSince,
			&state.ScaledToZeroAt,
			&state.LastWakeAt,
			&wakeReason,
			&state.IsScaledToZero,
			&state.CreatedAt,
			&state.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan idle state: %w", err)
		}

		if wakeReason != nil {
			wr := scaling.WakeReason(*wakeReason)
			state.WakeReason = &wr
		}

		states = append(states, state)
	}

	return states, rows.Err()
}

// RecordCostSavings records cost savings for a policy.
func (r *PostgresRepository) RecordCostSavings(ctx context.Context, savings *scaling.CostSavings) error {
	if savings.ID == uuid.Nil {
		savings.ID = uuid.New()
	}

	query := `
		INSERT INTO scaling_cost_savings (
			id, policy_id, date, idle_seconds, scaled_to_zero_seconds,
			estimated_savings_cents, hourly_cost_cents, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (policy_id, date) DO UPDATE SET
			idle_seconds = scaling_cost_savings.idle_seconds + EXCLUDED.idle_seconds,
			scaled_to_zero_seconds = scaling_cost_savings.scaled_to_zero_seconds + EXCLUDED.scaled_to_zero_seconds,
			estimated_savings_cents = scaling_cost_savings.estimated_savings_cents + EXCLUDED.estimated_savings_cents,
			hourly_cost_cents = COALESCE(EXCLUDED.hourly_cost_cents, scaling_cost_savings.hourly_cost_cents),
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		savings.ID,
		savings.PolicyID,
		savings.Date,
		savings.IdleSeconds,
		savings.ScaledToZeroSeconds,
		savings.EstimatedSavingsCents,
		savings.HourlyCostCents,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to record cost savings: %w", err)
	}

	return nil
}

// GetCostSavings retrieves cost savings for a policy within a date range.
func (r *PostgresRepository) GetCostSavings(ctx context.Context, policyID uuid.UUID, startDate, endDate time.Time) ([]scaling.CostSavings, error) {
	query := `
		SELECT id, policy_id, date, idle_seconds, scaled_to_zero_seconds,
			   estimated_savings_cents, hourly_cost_cents, created_at, updated_at
		FROM scaling_cost_savings
		WHERE policy_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, policyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost savings: %w", err)
	}
	defer rows.Close()

	var savings []scaling.CostSavings
	for rows.Next() {
		var s scaling.CostSavings
		if err := rows.Scan(
			&s.ID,
			&s.PolicyID,
			&s.Date,
			&s.IdleSeconds,
			&s.ScaledToZeroSeconds,
			&s.EstimatedSavingsCents,
			&s.HourlyCostCents,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cost savings: %w", err)
		}
		savings = append(savings, s)
	}

	return savings, rows.Err()
}

// GetTotalSavings retrieves total cost savings for a policy.
func (r *PostgresRepository) GetTotalSavings(ctx context.Context, policyID uuid.UUID) (*SavingsSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(idle_seconds), 0) as total_idle_seconds,
			COALESCE(SUM(scaled_to_zero_seconds), 0) as total_scaled_to_zero_seconds,
			COALESCE(SUM(estimated_savings_cents), 0) as total_savings_cents,
			COUNT(*) as day_count
		FROM scaling_cost_savings
		WHERE policy_id = $1
	`

	var summary SavingsSummary
	err := r.db.QueryRowContext(ctx, query, policyID).Scan(
		&summary.TotalIdleSeconds,
		&summary.TotalScaledToZeroSeconds,
		&summary.TotalSavingsCents,
		&summary.DayCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get total savings: %w", err)
	}

	return &summary, nil
}
