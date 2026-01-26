// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/alerting"
	"github.com/janovincze/philotes/internal/api/models"
)

// Alert repository errors.
var (
	ErrAlertRuleNotFound       = errors.New("alert rule not found")
	ErrAlertRuleNameExists     = errors.New("alert rule with this name already exists")
	ErrAlertInstanceNotFound   = errors.New("alert instance not found")
	ErrSilenceNotFound         = errors.New("silence not found")
	ErrChannelNotFound         = errors.New("notification channel not found")
	ErrChannelNameExists       = errors.New("notification channel with this name already exists")
	ErrRouteNotFound           = errors.New("alert route not found")
	ErrRouteExists             = errors.New("alert route already exists for this rule and channel")
)

// AlertRepository handles database operations for alerting.
type AlertRepository struct {
	db *sql.DB
}

// NewAlertRepository creates a new AlertRepository.
func NewAlertRepository(db *sql.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// alertRuleRow represents a database row for an alert rule.
type alertRuleRow struct {
	ID              uuid.UUID
	Name            string
	Description     sql.NullString
	MetricName      string
	Operator        string
	Threshold       float64
	DurationSeconds int
	Severity        string
	Labels          []byte
	Annotations     []byte
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// toModel converts a database row to an alerting model.
func (r *alertRuleRow) toModel() *alerting.AlertRule {
	rule := &alerting.AlertRule{
		ID:              r.ID,
		Name:            r.Name,
		MetricName:      r.MetricName,
		Operator:        alerting.Operator(r.Operator),
		Threshold:       r.Threshold,
		DurationSeconds: r.DurationSeconds,
		Severity:        alerting.AlertSeverity(r.Severity),
		Enabled:         r.Enabled,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}

	if r.Description.Valid {
		rule.Description = r.Description.String
	}
	if r.Labels != nil {
		if err := json.Unmarshal(r.Labels, &rule.Labels); err != nil {
			slog.Warn("failed to unmarshal alert rule labels", "rule_id", r.ID, "error", err)
		}
	}
	if r.Annotations != nil {
		if err := json.Unmarshal(r.Annotations, &rule.Annotations); err != nil {
			slog.Warn("failed to unmarshal alert rule annotations", "rule_id", r.ID, "error", err)
		}
	}

	return rule
}

// CreateRule creates a new alert rule in the database.
func (r *AlertRepository) CreateRule(ctx context.Context, req *models.CreateAlertRuleRequest) (*alerting.AlertRule, error) {
	labelsJSON, err := json.Marshal(req.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}
	annotationsJSON, err := json.Marshal(req.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal annotations: %w", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	query := `
		INSERT INTO philotes.alert_rules (
			name, description, metric_name, operator, threshold,
			duration_seconds, severity, labels, annotations, enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, description, metric_name, operator, threshold,
			duration_seconds, severity, labels, annotations, enabled, created_at, updated_at
	`

	var row alertRuleRow
	err = r.db.QueryRowContext(ctx, query,
		req.Name,
		nullString(req.Description),
		req.MetricName,
		req.Operator,
		req.Threshold,
		req.DurationSeconds,
		req.Severity,
		labelsJSON,
		annotationsJSON,
		enabled,
	).Scan(
		&row.ID,
		&row.Name,
		&row.Description,
		&row.MetricName,
		&row.Operator,
		&row.Threshold,
		&row.DurationSeconds,
		&row.Severity,
		&row.Labels,
		&row.Annotations,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlertRuleNameExists
		}
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}

	return row.toModel(), nil
}

// GetRule retrieves an alert rule by its ID.
func (r *AlertRepository) GetRule(ctx context.Context, id uuid.UUID) (*alerting.AlertRule, error) {
	query := `
		SELECT id, name, description, metric_name, operator, threshold,
			duration_seconds, severity, labels, annotations, enabled, created_at, updated_at
		FROM philotes.alert_rules
		WHERE id = $1
	`

	var row alertRuleRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Description,
		&row.MetricName,
		&row.Operator,
		&row.Threshold,
		&row.DurationSeconds,
		&row.Severity,
		&row.Labels,
		&row.Annotations,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertRuleNotFound
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return row.toModel(), nil
}

// ListRules retrieves all alert rules.
// Note: For internal use by the alerting manager (enabledOnly=true), this returns all enabled rules.
// For API pagination, use ListRulesPaginated instead.
func (r *AlertRepository) ListRules(ctx context.Context, enabledOnly bool) ([]alerting.AlertRule, error) {
	return r.ListRulesPaginated(ctx, enabledOnly, 0, 0)
}

// ListRulesPaginated retrieves alert rules with optional pagination.
// If limit is 0, all matching rules are returned.
func (r *AlertRepository) ListRulesPaginated(ctx context.Context, enabledOnly bool, limit, offset int) ([]alerting.AlertRule, error) {
	query := `
		SELECT id, name, description, metric_name, operator, threshold,
			duration_seconds, severity, labels, annotations, enabled, created_at, updated_at
		FROM philotes.alert_rules
	`
	args := []any{}
	argIdx := 1

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, limit, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert rules: %w", err)
	}
	defer rows.Close()

	var rules []alerting.AlertRule
	for rows.Next() {
		var row alertRuleRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Description,
			&row.MetricName,
			&row.Operator,
			&row.Threshold,
			&row.DurationSeconds,
			&row.Severity,
			&row.Labels,
			&row.Annotations,
			&row.Enabled,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule row: %w", err)
		}
		rules = append(rules, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate alert rules: %w", err)
	}

	return rules, nil
}

// UpdateRule updates an alert rule in the database.
func (r *AlertRepository) UpdateRule(ctx context.Context, id uuid.UUID, req *models.UpdateAlertRuleRequest) (*alerting.AlertRule, error) {
	// First check if rule exists
	_, err := r.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.alert_rules SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, nullString(*req.Description))
		argIdx++
	}
	if req.MetricName != nil {
		query += fmt.Sprintf(", metric_name = $%d", argIdx)
		args = append(args, *req.MetricName)
		argIdx++
	}
	if req.Operator != nil {
		query += fmt.Sprintf(", operator = $%d", argIdx)
		args = append(args, *req.Operator)
		argIdx++
	}
	if req.Threshold != nil {
		query += fmt.Sprintf(", threshold = $%d", argIdx)
		args = append(args, *req.Threshold)
		argIdx++
	}
	if req.DurationSeconds != nil {
		query += fmt.Sprintf(", duration_seconds = $%d", argIdx)
		args = append(args, *req.DurationSeconds)
		argIdx++
	}
	if req.Severity != nil {
		query += fmt.Sprintf(", severity = $%d", argIdx)
		args = append(args, *req.Severity)
		argIdx++
	}
	if req.Labels != nil {
		labelsJSON, err := json.Marshal(req.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal labels: %w", err)
		}
		query += fmt.Sprintf(", labels = $%d", argIdx)
		args = append(args, labelsJSON)
		argIdx++
	}
	if req.Annotations != nil {
		annotationsJSON, err := json.Marshal(req.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations: %w", err)
		}
		query += fmt.Sprintf(", annotations = $%d", argIdx)
		args = append(args, annotationsJSON)
		argIdx++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argIdx)
		args = append(args, *req.Enabled)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlertRuleNameExists
		}
		return nil, fmt.Errorf("failed to update alert rule: %w", err)
	}

	return r.GetRule(ctx, id)
}

// DeleteRule deletes an alert rule from the database.
func (r *AlertRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.alert_rules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertRuleNotFound
	}

	return nil
}

// alertInstanceRow represents a database row for an alert instance.
type alertInstanceRow struct {
	ID             uuid.UUID
	RuleID         uuid.UUID
	Fingerprint    string
	Status         string
	Labels         []byte
	Annotations    []byte
	CurrentValue   sql.NullFloat64
	FiredAt        time.Time
	ResolvedAt     sql.NullTime
	AcknowledgedAt sql.NullTime
	AcknowledgedBy sql.NullString
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// toModel converts a database row to an alerting model.
func (r *alertInstanceRow) toModel() *alerting.AlertInstance {
	instance := &alerting.AlertInstance{
		ID:          r.ID,
		RuleID:      r.RuleID,
		Fingerprint: r.Fingerprint,
		Status:      alerting.AlertStatus(r.Status),
		FiredAt:     r.FiredAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.Labels != nil {
		if err := json.Unmarshal(r.Labels, &instance.Labels); err != nil {
			slog.Warn("failed to unmarshal alert instance labels", "instance_id", r.ID, "error", err)
		}
	}
	if r.Annotations != nil {
		if err := json.Unmarshal(r.Annotations, &instance.Annotations); err != nil {
			slog.Warn("failed to unmarshal alert instance annotations", "instance_id", r.ID, "error", err)
		}
	}
	if r.CurrentValue.Valid {
		instance.CurrentValue = &r.CurrentValue.Float64
	}
	if r.ResolvedAt.Valid {
		instance.ResolvedAt = &r.ResolvedAt.Time
	}
	if r.AcknowledgedAt.Valid {
		instance.AcknowledgedAt = &r.AcknowledgedAt.Time
	}
	if r.AcknowledgedBy.Valid {
		instance.AcknowledgedBy = r.AcknowledgedBy.String
	}

	return instance
}

// CreateInstance creates a new alert instance in the database.
func (r *AlertRepository) CreateInstance(ctx context.Context, instance *alerting.AlertInstance) (*alerting.AlertInstance, error) {
	labelsJSON, err := json.Marshal(instance.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}
	annotationsJSON, err := json.Marshal(instance.Annotations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal annotations: %w", err)
	}

	var currentValue sql.NullFloat64
	if instance.CurrentValue != nil {
		currentValue = sql.NullFloat64{Float64: *instance.CurrentValue, Valid: true}
	}

	query := `
		INSERT INTO philotes.alert_instances (
			rule_id, fingerprint, status, labels, annotations, current_value, fired_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, rule_id, fingerprint, status, labels, annotations, current_value,
			fired_at, resolved_at, acknowledged_at, acknowledged_by, created_at, updated_at
	`

	var row alertInstanceRow
	err = r.db.QueryRowContext(ctx, query,
		instance.RuleID,
		instance.Fingerprint,
		instance.Status,
		labelsJSON,
		annotationsJSON,
		currentValue,
		instance.FiredAt,
	).Scan(
		&row.ID,
		&row.RuleID,
		&row.Fingerprint,
		&row.Status,
		&row.Labels,
		&row.Annotations,
		&row.CurrentValue,
		&row.FiredAt,
		&row.ResolvedAt,
		&row.AcknowledgedAt,
		&row.AcknowledgedBy,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert instance: %w", err)
	}

	return row.toModel(), nil
}

// GetInstance retrieves an alert instance by its ID.
func (r *AlertRepository) GetInstance(ctx context.Context, id uuid.UUID) (*alerting.AlertInstance, error) {
	query := `
		SELECT id, rule_id, fingerprint, status, labels, annotations, current_value,
			fired_at, resolved_at, acknowledged_at, acknowledged_by, created_at, updated_at
		FROM philotes.alert_instances
		WHERE id = $1
	`

	var row alertInstanceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.RuleID,
		&row.Fingerprint,
		&row.Status,
		&row.Labels,
		&row.Annotations,
		&row.CurrentValue,
		&row.FiredAt,
		&row.ResolvedAt,
		&row.AcknowledgedAt,
		&row.AcknowledgedBy,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get alert instance: %w", err)
	}

	return row.toModel(), nil
}

// GetInstanceByFingerprint retrieves an alert instance by rule ID and fingerprint.
func (r *AlertRepository) GetInstanceByFingerprint(ctx context.Context, ruleID uuid.UUID, fingerprint string) (*alerting.AlertInstance, error) {
	query := `
		SELECT id, rule_id, fingerprint, status, labels, annotations, current_value,
			fired_at, resolved_at, acknowledged_at, acknowledged_by, created_at, updated_at
		FROM philotes.alert_instances
		WHERE rule_id = $1 AND fingerprint = $2
	`

	var row alertInstanceRow
	err := r.db.QueryRowContext(ctx, query, ruleID, fingerprint).Scan(
		&row.ID,
		&row.RuleID,
		&row.Fingerprint,
		&row.Status,
		&row.Labels,
		&row.Annotations,
		&row.CurrentValue,
		&row.FiredAt,
		&row.ResolvedAt,
		&row.AcknowledgedAt,
		&row.AcknowledgedBy,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get alert instance by fingerprint: %w", err)
	}

	return row.toModel(), nil
}

// ListInstances retrieves alert instances with optional filtering.
func (r *AlertRepository) ListInstances(ctx context.Context, status *alerting.AlertStatus, ruleID *uuid.UUID) ([]alerting.AlertInstance, error) {
	query := `
		SELECT id, rule_id, fingerprint, status, labels, annotations, current_value,
			fired_at, resolved_at, acknowledged_at, acknowledged_by, created_at, updated_at
		FROM philotes.alert_instances
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if ruleID != nil {
		query += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, *ruleID)
		argIdx++
	}

	query += " ORDER BY fired_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert instances: %w", err)
	}
	defer rows.Close()

	var instances []alerting.AlertInstance
	for rows.Next() {
		var row alertInstanceRow
		err := rows.Scan(
			&row.ID,
			&row.RuleID,
			&row.Fingerprint,
			&row.Status,
			&row.Labels,
			&row.Annotations,
			&row.CurrentValue,
			&row.FiredAt,
			&row.ResolvedAt,
			&row.AcknowledgedAt,
			&row.AcknowledgedBy,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert instance row: %w", err)
		}
		instances = append(instances, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate alert instances: %w", err)
	}

	return instances, nil
}

// UpdateInstance updates an alert instance in the database.
func (r *AlertRepository) UpdateInstance(ctx context.Context, id uuid.UUID, status alerting.AlertStatus, currentValue *float64, resolvedAt *time.Time) error {
	query := `
		UPDATE philotes.alert_instances
		SET status = $1, current_value = $2, resolved_at = $3, updated_at = NOW()
		WHERE id = $4
	`

	var cv sql.NullFloat64
	if currentValue != nil {
		cv = sql.NullFloat64{Float64: *currentValue, Valid: true}
	}

	var ra sql.NullTime
	if resolvedAt != nil {
		ra = sql.NullTime{Time: *resolvedAt, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, status, cv, ra, id)
	if err != nil {
		return fmt.Errorf("failed to update alert instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertInstanceNotFound
	}

	return nil
}

// AcknowledgeInstance acknowledges an alert instance.
func (r *AlertRepository) AcknowledgeInstance(ctx context.Context, id uuid.UUID, acknowledgedBy string) error {
	query := `
		UPDATE philotes.alert_instances
		SET acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, acknowledgedBy, id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAlertInstanceNotFound
	}

	return nil
}

// CreateHistory creates an alert history entry.
func (r *AlertRepository) CreateHistory(ctx context.Context, history *alerting.AlertHistory) (*alerting.AlertHistory, error) {
	metadataJSON, err := json.Marshal(history.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var value sql.NullFloat64
	if history.Value != nil {
		value = sql.NullFloat64{Float64: *history.Value, Valid: true}
	}

	query := `
		INSERT INTO philotes.alert_history (alert_id, rule_id, event_type, message, value, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, alert_id, rule_id, event_type, message, value, metadata, created_at
	`

	var id uuid.UUID
	var alertID, ruleID uuid.UUID
	var eventType, message string
	var returnedValue sql.NullFloat64
	var metadata []byte
	var createdAt time.Time

	err = r.db.QueryRowContext(ctx, query,
		history.AlertID,
		history.RuleID,
		history.EventType,
		history.Message,
		value,
		metadataJSON,
	).Scan(&id, &alertID, &ruleID, &eventType, &message, &returnedValue, &metadata, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert history: %w", err)
	}

	result := &alerting.AlertHistory{
		ID:        id,
		AlertID:   alertID,
		RuleID:    ruleID,
		EventType: alerting.EventType(eventType),
		Message:   message,
		CreatedAt: createdAt,
	}
	if returnedValue.Valid {
		result.Value = &returnedValue.Float64
	}
	if metadata != nil {
		if err := json.Unmarshal(metadata, &result.Metadata); err != nil {
			slog.Warn("failed to unmarshal alert history metadata", "history_id", id, "error", err)
		}
	}

	return result, nil
}

// ListHistory retrieves alert history for an alert instance.
func (r *AlertRepository) ListHistory(ctx context.Context, alertID *uuid.UUID, ruleID *uuid.UUID, limit int) ([]alerting.AlertHistory, error) {
	query := `
		SELECT id, alert_id, rule_id, event_type, message, value, metadata, created_at
		FROM philotes.alert_history
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if alertID != nil {
		query += fmt.Sprintf(" AND alert_id = $%d", argIdx)
		args = append(args, *alertID)
		argIdx++
	}
	if ruleID != nil {
		query += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, *ruleID)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert history: %w", err)
	}
	defer rows.Close()

	var history []alerting.AlertHistory
	for rows.Next() {
		var h alerting.AlertHistory
		var value sql.NullFloat64
		var metadata []byte

		err := rows.Scan(
			&h.ID,
			&h.AlertID,
			&h.RuleID,
			&h.EventType,
			&h.Message,
			&value,
			&metadata,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert history row: %w", err)
		}

		if value.Valid {
			h.Value = &value.Float64
		}
		if metadata != nil {
			if err := json.Unmarshal(metadata, &h.Metadata); err != nil {
				slog.Warn("failed to unmarshal alert history metadata", "history_id", h.ID, "error", err)
			}
		}

		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate alert history: %w", err)
	}

	return history, nil
}

// silenceRow represents a database row for an alert silence.
type silenceRow struct {
	ID        uuid.UUID
	Matchers  []byte
	StartsAt  time.Time
	EndsAt    time.Time
	CreatedBy string
	Comment   sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

// toModel converts a database row to an alerting model.
func (r *silenceRow) toModel() *alerting.AlertSilence {
	silence := &alerting.AlertSilence{
		ID:        r.ID,
		StartsAt:  r.StartsAt,
		EndsAt:    r.EndsAt,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	if r.Matchers != nil {
		if err := json.Unmarshal(r.Matchers, &silence.Matchers); err != nil {
			slog.Warn("failed to unmarshal silence matchers", "silence_id", r.ID, "error", err)
		}
	}
	if r.Comment.Valid {
		silence.Comment = r.Comment.String
	}

	return silence
}

// CreateSilence creates a new alert silence in the database.
func (r *AlertRepository) CreateSilence(ctx context.Context, req *models.CreateSilenceRequest) (*alerting.AlertSilence, error) {
	matchersJSON, err := json.Marshal(req.Matchers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal matchers: %w", err)
	}

	query := `
		INSERT INTO philotes.alert_silences (matchers, starts_at, ends_at, created_by, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, matchers, starts_at, ends_at, created_by, comment, created_at, updated_at
	`

	var row silenceRow
	err = r.db.QueryRowContext(ctx, query,
		matchersJSON,
		req.StartsAt,
		req.EndsAt,
		req.CreatedBy,
		nullString(req.Comment),
	).Scan(
		&row.ID,
		&row.Matchers,
		&row.StartsAt,
		&row.EndsAt,
		&row.CreatedBy,
		&row.Comment,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create silence: %w", err)
	}

	return row.toModel(), nil
}

// GetSilence retrieves a silence by its ID.
func (r *AlertRepository) GetSilence(ctx context.Context, id uuid.UUID) (*alerting.AlertSilence, error) {
	query := `
		SELECT id, matchers, starts_at, ends_at, created_by, comment, created_at, updated_at
		FROM philotes.alert_silences
		WHERE id = $1
	`

	var row silenceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Matchers,
		&row.StartsAt,
		&row.EndsAt,
		&row.CreatedBy,
		&row.Comment,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSilenceNotFound
		}
		return nil, fmt.Errorf("failed to get silence: %w", err)
	}

	return row.toModel(), nil
}

// ListSilences retrieves all silences, optionally filtering to only active ones.
func (r *AlertRepository) ListSilences(ctx context.Context, activeOnly bool) ([]alerting.AlertSilence, error) {
	query := `
		SELECT id, matchers, starts_at, ends_at, created_by, comment, created_at, updated_at
		FROM philotes.alert_silences
	`

	if activeOnly {
		query += " WHERE starts_at <= NOW() AND ends_at > NOW()"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list silences: %w", err)
	}
	defer rows.Close()

	var silences []alerting.AlertSilence
	for rows.Next() {
		var row silenceRow
		err := rows.Scan(
			&row.ID,
			&row.Matchers,
			&row.StartsAt,
			&row.EndsAt,
			&row.CreatedBy,
			&row.Comment,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan silence row: %w", err)
		}
		silences = append(silences, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate silences: %w", err)
	}

	return silences, nil
}

// DeleteSilence deletes a silence from the database.
func (r *AlertRepository) DeleteSilence(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.alert_silences WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete silence: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSilenceNotFound
	}

	return nil
}

// channelRow represents a database row for a notification channel.
type channelRow struct {
	ID        uuid.UUID
	Name      string
	Type      string
	Config    []byte
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// toModel converts a database row to an alerting model.
func (r *channelRow) toModel() *alerting.NotificationChannel {
	channel := &alerting.NotificationChannel{
		ID:        r.ID,
		Name:      r.Name,
		Type:      alerting.ChannelType(r.Type),
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	if r.Config != nil {
		if err := json.Unmarshal(r.Config, &channel.Config); err != nil {
			slog.Warn("failed to unmarshal channel config", "channel_id", r.ID, "error", err)
		}
	}

	return channel
}

// CreateChannel creates a new notification channel in the database.
func (r *AlertRepository) CreateChannel(ctx context.Context, req *models.CreateChannelRequest) (*alerting.NotificationChannel, error) {
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	query := `
		INSERT INTO philotes.notification_channels (name, type, config, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, type, config, enabled, created_at, updated_at
	`

	var row channelRow
	err = r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Type,
		configJSON,
		enabled,
	).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Config,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrChannelNameExists
		}
		return nil, fmt.Errorf("failed to create notification channel: %w", err)
	}

	return row.toModel(), nil
}

// GetChannel retrieves a notification channel by its ID.
func (r *AlertRepository) GetChannel(ctx context.Context, id uuid.UUID) (*alerting.NotificationChannel, error) {
	query := `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM philotes.notification_channels
		WHERE id = $1
	`

	var row channelRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Config,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrChannelNotFound
		}
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}

	return row.toModel(), nil
}

// ListChannels retrieves all notification channels.
func (r *AlertRepository) ListChannels(ctx context.Context, enabledOnly bool) ([]alerting.NotificationChannel, error) {
	query := `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM philotes.notification_channels
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []alerting.NotificationChannel
	for rows.Next() {
		var row channelRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Type,
			&row.Config,
			&row.Enabled,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification channel row: %w", err)
		}
		channels = append(channels, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate notification channels: %w", err)
	}

	return channels, nil
}

// UpdateChannel updates a notification channel in the database.
func (r *AlertRepository) UpdateChannel(ctx context.Context, id uuid.UUID, req *models.UpdateChannelRequest) (*alerting.NotificationChannel, error) {
	// First check if channel exists
	_, err := r.GetChannel(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.notification_channels SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		query += fmt.Sprintf(", config = $%d", argIdx)
		args = append(args, configJSON)
		argIdx++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argIdx)
		args = append(args, *req.Enabled)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrChannelNameExists
		}
		return nil, fmt.Errorf("failed to update notification channel: %w", err)
	}

	return r.GetChannel(ctx, id)
}

// DeleteChannel deletes a notification channel from the database.
func (r *AlertRepository) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.notification_channels WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrChannelNotFound
	}

	return nil
}

// routeRow represents a database row for an alert route.
type routeRow struct {
	ID                    uuid.UUID
	RuleID                uuid.UUID
	ChannelID             uuid.UUID
	RepeatIntervalSeconds int
	GroupWaitSeconds      int
	GroupIntervalSeconds  int
	Enabled               bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// toModel converts a database row to an alerting model.
func (r *routeRow) toModel() *alerting.AlertRoute {
	return &alerting.AlertRoute{
		ID:                    r.ID,
		RuleID:                r.RuleID,
		ChannelID:             r.ChannelID,
		RepeatIntervalSeconds: r.RepeatIntervalSeconds,
		GroupWaitSeconds:      r.GroupWaitSeconds,
		GroupIntervalSeconds:  r.GroupIntervalSeconds,
		Enabled:               r.Enabled,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}

// CreateRoute creates a new alert route in the database.
func (r *AlertRepository) CreateRoute(ctx context.Context, req *models.CreateRouteRequest) (*alerting.AlertRoute, error) {
	repeatInterval := 3600
	if req.RepeatIntervalSeconds != nil {
		repeatInterval = *req.RepeatIntervalSeconds
	}
	groupWait := 30
	if req.GroupWaitSeconds != nil {
		groupWait = *req.GroupWaitSeconds
	}
	groupInterval := 300
	if req.GroupIntervalSeconds != nil {
		groupInterval = *req.GroupIntervalSeconds
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	query := `
		INSERT INTO philotes.alert_routes (
			rule_id, channel_id, repeat_interval_seconds, group_wait_seconds,
			group_interval_seconds, enabled
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, rule_id, channel_id, repeat_interval_seconds, group_wait_seconds,
			group_interval_seconds, enabled, created_at, updated_at
	`

	var row routeRow
	err := r.db.QueryRowContext(ctx, query,
		req.RuleID,
		req.ChannelID,
		repeatInterval,
		groupWait,
		groupInterval,
		enabled,
	).Scan(
		&row.ID,
		&row.RuleID,
		&row.ChannelID,
		&row.RepeatIntervalSeconds,
		&row.GroupWaitSeconds,
		&row.GroupIntervalSeconds,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrRouteExists
		}
		if isForeignKeyViolation(err) {
			return nil, fmt.Errorf("rule or channel not found: %w", err)
		}
		return nil, fmt.Errorf("failed to create alert route: %w", err)
	}

	return row.toModel(), nil
}

// GetRoute retrieves an alert route by its ID.
func (r *AlertRepository) GetRoute(ctx context.Context, id uuid.UUID) (*alerting.AlertRoute, error) {
	query := `
		SELECT id, rule_id, channel_id, repeat_interval_seconds, group_wait_seconds,
			group_interval_seconds, enabled, created_at, updated_at
		FROM philotes.alert_routes
		WHERE id = $1
	`

	var row routeRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.RuleID,
		&row.ChannelID,
		&row.RepeatIntervalSeconds,
		&row.GroupWaitSeconds,
		&row.GroupIntervalSeconds,
		&row.Enabled,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRouteNotFound
		}
		return nil, fmt.Errorf("failed to get alert route: %w", err)
	}

	return row.toModel(), nil
}

// ListRoutes retrieves alert routes, optionally filtered by rule ID.
func (r *AlertRepository) ListRoutes(ctx context.Context, ruleID *uuid.UUID, enabledOnly bool) ([]alerting.AlertRoute, error) {
	query := `
		SELECT id, rule_id, channel_id, repeat_interval_seconds, group_wait_seconds,
			group_interval_seconds, enabled, created_at, updated_at
		FROM philotes.alert_routes
		WHERE 1=1
	`
	args := []any{}
	argIdx := 1

	if ruleID != nil {
		query += fmt.Sprintf(" AND rule_id = $%d", argIdx)
		args = append(args, *ruleID)
		argIdx++
	}
	if enabledOnly {
		query += " AND enabled = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert routes: %w", err)
	}
	defer rows.Close()

	var routes []alerting.AlertRoute
	for rows.Next() {
		var row routeRow
		err := rows.Scan(
			&row.ID,
			&row.RuleID,
			&row.ChannelID,
			&row.RepeatIntervalSeconds,
			&row.GroupWaitSeconds,
			&row.GroupIntervalSeconds,
			&row.Enabled,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert route row: %w", err)
		}
		routes = append(routes, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate alert routes: %w", err)
	}

	return routes, nil
}

// UpdateRoute updates an alert route in the database.
func (r *AlertRepository) UpdateRoute(ctx context.Context, id uuid.UUID, req *models.UpdateRouteRequest) (*alerting.AlertRoute, error) {
	// First check if route exists
	_, err := r.GetRoute(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.alert_routes SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.RepeatIntervalSeconds != nil {
		query += fmt.Sprintf(", repeat_interval_seconds = $%d", argIdx)
		args = append(args, *req.RepeatIntervalSeconds)
		argIdx++
	}
	if req.GroupWaitSeconds != nil {
		query += fmt.Sprintf(", group_wait_seconds = $%d", argIdx)
		args = append(args, *req.GroupWaitSeconds)
		argIdx++
	}
	if req.GroupIntervalSeconds != nil {
		query += fmt.Sprintf(", group_interval_seconds = $%d", argIdx)
		args = append(args, *req.GroupIntervalSeconds)
		argIdx++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argIdx)
		args = append(args, *req.Enabled)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update alert route: %w", err)
	}

	return r.GetRoute(ctx, id)
}

// DeleteRoute deletes an alert route from the database.
func (r *AlertRepository) DeleteRoute(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.alert_routes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert route: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrRouteNotFound
	}

	return nil
}

// GetAlertSummary returns summary statistics for alerts.
func (r *AlertRepository) GetAlertSummary(ctx context.Context) (*models.AlertSummaryResponse, error) {
	summary := &models.AlertSummaryResponse{}

	// Count total and enabled rules
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled
		FROM philotes.alert_rules
	`).Scan(&summary.TotalRules, &summary.EnabledRules)
	if err != nil {
		return nil, fmt.Errorf("failed to count rules: %w", err)
	}

	// Count firing and resolved alerts
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'firing') as firing,
			COUNT(*) FILTER (WHERE status = 'resolved') as resolved
		FROM philotes.alert_instances
	`).Scan(&summary.FiringAlerts, &summary.ResolvedAlerts)
	if err != nil {
		return nil, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Count active silences
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM philotes.alert_silences
		WHERE starts_at <= NOW() AND ends_at > NOW()
	`).Scan(&summary.ActiveSilences)
	if err != nil {
		return nil, fmt.Errorf("failed to count silences: %w", err)
	}

	// Count total and enabled channels
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled
		FROM philotes.notification_channels
	`).Scan(&summary.TotalChannels, &summary.EnabledChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to count channels: %w", err)
	}

	return summary, nil
}
