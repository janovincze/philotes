// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// Repository provides database operations for scaling resources.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new scaling repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// Policy Operations
// ============================================================================

// CreatePolicy creates a new scaling policy.
func (r *Repository) CreatePolicy(ctx context.Context, policy *ScalingPolicy) (*ScalingPolicy, error) {
	query := `
		INSERT INTO scaling_policies (
			name, target_type, target_id, min_replicas, max_replicas,
			cooldown_seconds, max_hourly_cost, scale_to_zero, enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		policy.Name,
		policy.TargetType,
		policy.TargetID,
		policy.MinReplicas,
		policy.MaxReplicas,
		policy.CooldownSeconds,
		policy.MaxHourlyCost,
		policy.ScaleToZero,
		policy.Enabled,
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	return policy, nil
}

// GetPolicy retrieves a scaling policy by ID.
func (r *Repository) GetPolicy(ctx context.Context, id uuid.UUID) (*ScalingPolicy, error) {
	query := `
		SELECT id, name, target_type, target_id, min_replicas, max_replicas,
			   cooldown_seconds, max_hourly_cost, scale_to_zero, enabled,
			   created_at, updated_at
		FROM scaling_policies
		WHERE id = $1`

	policy := &ScalingPolicy{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&policy.ID,
		&policy.Name,
		&policy.TargetType,
		&policy.TargetID,
		&policy.MinReplicas,
		&policy.MaxReplicas,
		&policy.CooldownSeconds,
		&policy.MaxHourlyCost,
		&policy.ScaleToZero,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return policy, nil
}

// GetPolicyByName retrieves a scaling policy by name.
func (r *Repository) GetPolicyByName(ctx context.Context, name string) (*ScalingPolicy, error) {
	query := `
		SELECT id, name, target_type, target_id, min_replicas, max_replicas,
			   cooldown_seconds, max_hourly_cost, scale_to_zero, enabled,
			   created_at, updated_at
		FROM scaling_policies
		WHERE name = $1`

	policy := &ScalingPolicy{}
	err := r.db.QueryRow(ctx, query, name).Scan(
		&policy.ID,
		&policy.Name,
		&policy.TargetType,
		&policy.TargetID,
		&policy.MinReplicas,
		&policy.MaxReplicas,
		&policy.CooldownSeconds,
		&policy.MaxHourlyCost,
		&policy.ScaleToZero,
		&policy.Enabled,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy by name: %w", err)
	}

	return policy, nil
}

// ListPolicies lists all scaling policies, optionally filtered by enabled status.
func (r *Repository) ListPolicies(ctx context.Context, enabledOnly bool) ([]ScalingPolicy, error) {
	query := `
		SELECT id, name, target_type, target_id, min_replicas, max_replicas,
			   cooldown_seconds, max_hourly_cost, scale_to_zero, enabled,
			   created_at, updated_at
		FROM scaling_policies`

	if enabledOnly {
		query += ` WHERE enabled = true`
	}
	query += ` ORDER BY name`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer rows.Close()

	var policies []ScalingPolicy
	for rows.Next() {
		var policy ScalingPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.TargetType,
			&policy.TargetID,
			&policy.MinReplicas,
			&policy.MaxReplicas,
			&policy.CooldownSeconds,
			&policy.MaxHourlyCost,
			&policy.ScaleToZero,
			&policy.Enabled,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, policy)
	}

	return policies, rows.Err()
}

// UpdatePolicy updates a scaling policy.
func (r *Repository) UpdatePolicy(ctx context.Context, policy *ScalingPolicy) error {
	query := `
		UPDATE scaling_policies
		SET name = $2, target_type = $3, target_id = $4, min_replicas = $5,
			max_replicas = $6, cooldown_seconds = $7, max_hourly_cost = $8,
			scale_to_zero = $9, enabled = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRow(ctx, query,
		policy.ID,
		policy.Name,
		policy.TargetType,
		policy.TargetID,
		policy.MinReplicas,
		policy.MaxReplicas,
		policy.CooldownSeconds,
		policy.MaxHourlyCost,
		policy.ScaleToZero,
		policy.Enabled,
	).Scan(&policy.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	return nil
}

// DeletePolicy deletes a scaling policy.
func (r *Repository) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM scaling_policies WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ============================================================================
// Rule Operations
// ============================================================================

// CreateRule creates a new scaling rule.
func (r *Repository) CreateRule(ctx context.Context, rule *ScalingRule) (*ScalingRule, error) {
	query := `
		INSERT INTO scaling_rules (
			policy_id, rule_type, metric, operator, threshold, duration_seconds, scale_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		rule.PolicyID,
		rule.RuleType,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		rule.DurationSeconds,
		rule.ScaleBy,
	).Scan(&rule.ID, &rule.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	return rule, nil
}

// GetRulesForPolicy retrieves all rules for a policy.
func (r *Repository) GetRulesForPolicy(ctx context.Context, policyID uuid.UUID) ([]ScalingRule, error) {
	query := `
		SELECT id, policy_id, rule_type, metric, operator, threshold,
			   duration_seconds, scale_by, created_at
		FROM scaling_rules
		WHERE policy_id = $1
		ORDER BY rule_type, created_at`

	rows, err := r.db.Query(ctx, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}
	defer rows.Close()

	var rules []ScalingRule
	for rows.Next() {
		var rule ScalingRule
		err := rows.Scan(
			&rule.ID,
			&rule.PolicyID,
			&rule.RuleType,
			&rule.Metric,
			&rule.Operator,
			&rule.Threshold,
			&rule.DurationSeconds,
			&rule.ScaleBy,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// DeleteRule deletes a scaling rule.
func (r *Repository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM scaling_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteRulesForPolicy deletes all rules for a policy.
func (r *Repository) DeleteRulesForPolicy(ctx context.Context, policyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM scaling_rules WHERE policy_id = $1", policyID)
	if err != nil {
		return fmt.Errorf("failed to delete rules: %w", err)
	}
	return nil
}

// ============================================================================
// Schedule Operations
// ============================================================================

// CreateSchedule creates a new scaling schedule.
func (r *Repository) CreateSchedule(ctx context.Context, schedule *ScalingSchedule) (*ScalingSchedule, error) {
	query := `
		INSERT INTO scaling_schedules (
			policy_id, cron_expression, desired_replicas, timezone, enabled
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query,
		schedule.PolicyID,
		schedule.CronExpression,
		schedule.DesiredReplicas,
		schedule.Timezone,
		schedule.Enabled,
	).Scan(&schedule.ID, &schedule.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return schedule, nil
}

// GetSchedulesForPolicy retrieves all schedules for a policy.
func (r *Repository) GetSchedulesForPolicy(ctx context.Context, policyID uuid.UUID) ([]ScalingSchedule, error) {
	query := `
		SELECT id, policy_id, cron_expression, desired_replicas, timezone, enabled, created_at
		FROM scaling_schedules
		WHERE policy_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	defer rows.Close()

	var schedules []ScalingSchedule
	for rows.Next() {
		var schedule ScalingSchedule
		err := rows.Scan(
			&schedule.ID,
			&schedule.PolicyID,
			&schedule.CronExpression,
			&schedule.DesiredReplicas,
			&schedule.Timezone,
			&schedule.Enabled,
			&schedule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	return schedules, rows.Err()
}

// DeleteSchedule deletes a scaling schedule.
func (r *Repository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, "DELETE FROM scaling_schedules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteSchedulesForPolicy deletes all schedules for a policy.
func (r *Repository) DeleteSchedulesForPolicy(ctx context.Context, policyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM scaling_schedules WHERE policy_id = $1", policyID)
	if err != nil {
		return fmt.Errorf("failed to delete schedules: %w", err)
	}
	return nil
}

// ============================================================================
// History Operations
// ============================================================================

// CreateHistory creates a new scaling history entry.
func (r *Repository) CreateHistory(ctx context.Context, history *ScalingHistory) (*ScalingHistory, error) {
	query := `
		INSERT INTO scaling_history (
			policy_id, policy_name, action, target_type, target_id,
			previous_replicas, new_replicas, reason, triggered_by, dry_run
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, executed_at`

	err := r.db.QueryRow(ctx, query,
		history.PolicyID,
		history.PolicyName,
		history.Action,
		history.TargetType,
		history.TargetID,
		history.PreviousReplicas,
		history.NewReplicas,
		history.Reason,
		history.TriggeredBy,
		history.DryRun,
	).Scan(&history.ID, &history.ExecutedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create history: %w", err)
	}

	return history, nil
}

// ListHistory lists scaling history with optional filters.
func (r *Repository) ListHistory(ctx context.Context, policyID *uuid.UUID, limit int) ([]ScalingHistory, error) {
	query := `
		SELECT id, policy_id, policy_name, action, target_type, target_id,
			   previous_replicas, new_replicas, reason, triggered_by, dry_run, executed_at
		FROM scaling_history`

	args := []any{}
	if policyID != nil {
		query += ` WHERE policy_id = $1`
		args = append(args, *policyID)
	}

	query += ` ORDER BY executed_at DESC`

	if limit > 0 {
		if len(args) > 0 {
			query += fmt.Sprintf(` LIMIT $%d`, len(args)+1)
		} else {
			query += ` LIMIT $1`
		}
		args = append(args, limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var history []ScalingHistory
	for rows.Next() {
		var h ScalingHistory
		err := rows.Scan(
			&h.ID,
			&h.PolicyID,
			&h.PolicyName,
			&h.Action,
			&h.TargetType,
			&h.TargetID,
			&h.PreviousReplicas,
			&h.NewReplicas,
			&h.Reason,
			&h.TriggeredBy,
			&h.DryRun,
			&h.ExecutedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// ============================================================================
// State Operations
// ============================================================================

// GetState retrieves the scaling state for a policy.
func (r *Repository) GetState(ctx context.Context, policyID uuid.UUID) (*ScalingState, error) {
	query := `
		SELECT id, policy_id, current_replicas, last_scale_time, last_scale_action,
			   pending_conditions, updated_at
		FROM scaling_state
		WHERE policy_id = $1`

	state := &ScalingState{}
	var pendingConditionsJSON sql.NullString

	err := r.db.QueryRow(ctx, query, policyID).Scan(
		&state.ID,
		&state.PolicyID,
		&state.CurrentReplicas,
		&state.LastScaleTime,
		&state.LastScaleAction,
		&pendingConditionsJSON,
		&state.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	if pendingConditionsJSON.Valid {
		if err := json.Unmarshal([]byte(pendingConditionsJSON.String), &state.PendingConditions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pending_conditions: %w", err)
		}
	}

	return state, nil
}

// UpsertState creates or updates the scaling state for a policy.
func (r *Repository) UpsertState(ctx context.Context, state *ScalingState) error {
	var pendingConditionsJSON []byte
	var err error
	if state.PendingConditions != nil {
		pendingConditionsJSON, err = json.Marshal(state.PendingConditions)
		if err != nil {
			return fmt.Errorf("failed to marshal pending_conditions: %w", err)
		}
	}

	query := `
		INSERT INTO scaling_state (
			policy_id, current_replicas, last_scale_time, last_scale_action, pending_conditions
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (policy_id) DO UPDATE SET
			current_replicas = EXCLUDED.current_replicas,
			last_scale_time = EXCLUDED.last_scale_time,
			last_scale_action = EXCLUDED.last_scale_action,
			pending_conditions = EXCLUDED.pending_conditions
		RETURNING id, updated_at`

	err = r.db.QueryRow(ctx, query,
		state.PolicyID,
		state.CurrentReplicas,
		state.LastScaleTime,
		state.LastScaleAction,
		pendingConditionsJSON,
	).Scan(&state.ID, &state.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert state: %w", err)
	}

	return nil
}

// CreateState creates a new scaling state (alias for UpsertState).
func (r *Repository) CreateState(ctx context.Context, state *ScalingState) (*ScalingState, error) {
	if err := r.UpsertState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// UpdateState updates an existing scaling state (alias for UpsertState).
func (r *Repository) UpdateState(ctx context.Context, state *ScalingState) error {
	return r.UpsertState(ctx, state)
}

// GetHistoryForPolicy retrieves scaling history for a specific policy.
func (r *Repository) GetHistoryForPolicy(ctx context.Context, policyID uuid.UUID, limit int) ([]ScalingHistory, error) {
	return r.ListHistory(ctx, &policyID, limit)
}

// ListRecentHistory retrieves recent scaling history across all policies.
func (r *Repository) ListRecentHistory(ctx context.Context, limit int) ([]ScalingHistory, error) {
	return r.ListHistory(ctx, nil, limit)
}

// ============================================================================
// Aggregate Operations
// ============================================================================

// GetPolicyWithDetails retrieves a policy with all its rules and schedules.
func (r *Repository) GetPolicyWithDetails(ctx context.Context, id uuid.UUID) (*ScalingPolicy, error) {
	policy, err := r.GetPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	rules, err := r.GetRulesForPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	for i := range rules {
		if rules[i].RuleType == RuleTypeScaleUp {
			policy.ScaleUpRules = append(policy.ScaleUpRules, rules[i])
		} else {
			policy.ScaleDownRules = append(policy.ScaleDownRules, rules[i])
		}
	}

	schedules, err := r.GetSchedulesForPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	policy.Schedules = schedules

	return policy, nil
}

// ListPoliciesWithDetails lists all policies with their rules and schedules.
func (r *Repository) ListPoliciesWithDetails(ctx context.Context, enabledOnly bool) ([]ScalingPolicy, error) {
	policies, err := r.ListPolicies(ctx, enabledOnly)
	if err != nil {
		return nil, err
	}

	for i := range policies {
		rules, err := r.GetRulesForPolicy(ctx, policies[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rules for policy %s: %w", policies[i].ID, err)
		}

		for j := range rules {
			if rules[j].RuleType == RuleTypeScaleUp {
				policies[i].ScaleUpRules = append(policies[i].ScaleUpRules, rules[j])
			} else {
				policies[i].ScaleDownRules = append(policies[i].ScaleDownRules, rules[j])
			}
		}

		schedules, err := r.GetSchedulesForPolicy(ctx, policies[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get schedules for policy %s: %w", policies[i].ID, err)
		}
		policies[i].Schedules = schedules
	}

	return policies, nil
}
