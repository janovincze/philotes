// Package services provides business logic for API resources.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/config"
)

// QueryScalingService provides business logic for query scaling operations.
type QueryScalingService struct {
	db           *sql.DB
	queryService *QueryService
	cfg          config.QueryScalingConfig
	logger       *slog.Logger
}

// NewQueryScalingService creates a new QueryScalingService.
func NewQueryScalingService(
	db *sql.DB,
	queryService *QueryService,
	cfg config.QueryScalingConfig,
	logger *slog.Logger,
) *QueryScalingService {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueryScalingService{
		db:           db,
		queryService: queryService,
		cfg:          cfg,
		logger:       logger.With("component", "query-scaling-service"),
	}
}

// CreatePolicy creates a new query scaling policy.
func (s *QueryScalingService) CreatePolicy(ctx context.Context, req *models.CreateQueryScalingPolicyRequest) (*models.QueryScalingPolicy, error) {
	// Apply defaults
	policy := &models.QueryScalingPolicy{
		ID:                      uuid.New(),
		Name:                    req.Name,
		QueryEngine:             req.QueryEngine,
		Enabled:                 true,
		MinReplicas:             s.cfg.DefaultMinReplicas,
		MaxReplicas:             s.cfg.DefaultMaxReplicas,
		CooldownSeconds:         s.cfg.DefaultCooldownSeconds,
		ScaleToZero:             false,
		QueuedQueriesThreshold:  s.cfg.DefaultQueuedQueriesThreshold,
		RunningQueriesThreshold: s.cfg.DefaultRunningQueriesThreshold,
		LatencyThresholdSeconds: s.cfg.DefaultLatencyThreshold,
		ScheduleEnabled:         false,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	// Apply request overrides
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.MinReplicas != nil {
		policy.MinReplicas = *req.MinReplicas
	}
	if req.MaxReplicas != nil {
		policy.MaxReplicas = *req.MaxReplicas
	}
	if req.CooldownSeconds != nil {
		policy.CooldownSeconds = *req.CooldownSeconds
	}
	if req.ScaleToZero != nil {
		policy.ScaleToZero = *req.ScaleToZero
	}
	if req.QueuedQueriesThreshold != nil {
		policy.QueuedQueriesThreshold = *req.QueuedQueriesThreshold
	}
	if req.RunningQueriesThreshold != nil {
		policy.RunningQueriesThreshold = *req.RunningQueriesThreshold
	}
	if req.LatencyThresholdSeconds != nil {
		policy.LatencyThresholdSeconds = *req.LatencyThresholdSeconds
	}
	if req.ScheduleEnabled != nil {
		policy.ScheduleEnabled = *req.ScheduleEnabled
	}
	policy.BusinessHoursMinReplicas = req.BusinessHoursMinReplicas
	policy.BusinessHoursStart = req.BusinessHoursStart
	policy.BusinessHoursEnd = req.BusinessHoursEnd
	policy.BusinessHoursTimezone = req.BusinessHoursTimezone

	// Insert into database
	query := `
		INSERT INTO query_scaling_policies (
			id, name, query_engine, enabled, min_replicas, max_replicas,
			cooldown_seconds, scale_to_zero, queued_queries_threshold,
			running_queries_threshold, latency_threshold_seconds,
			schedule_enabled, business_hours_min_replicas,
			business_hours_start, business_hours_end, business_hours_timezone,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := s.db.ExecContext(ctx, query,
		policy.ID, policy.Name, policy.QueryEngine, policy.Enabled,
		policy.MinReplicas, policy.MaxReplicas, policy.CooldownSeconds,
		policy.ScaleToZero, policy.QueuedQueriesThreshold,
		policy.RunningQueriesThreshold, policy.LatencyThresholdSeconds,
		policy.ScheduleEnabled, policy.BusinessHoursMinReplicas,
		policy.BusinessHoursStart, policy.BusinessHoursEnd, policy.BusinessHoursTimezone,
		policy.CreatedAt, policy.UpdatedAt,
	)
	if err != nil {
		s.logger.Error("failed to create query scaling policy", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	s.logger.Info("query scaling policy created", "id", policy.ID, "name", policy.Name)
	return policy, nil
}

// GetPolicy retrieves a query scaling policy by ID.
func (s *QueryScalingService) GetPolicy(ctx context.Context, id uuid.UUID) (*models.QueryScalingPolicy, error) {
	query := `
		SELECT id, name, query_engine, enabled, min_replicas, max_replicas,
			cooldown_seconds, scale_to_zero, queued_queries_threshold,
			running_queries_threshold, latency_threshold_seconds,
			schedule_enabled, business_hours_min_replicas,
			business_hours_start, business_hours_end, business_hours_timezone,
			created_at, updated_at
		FROM query_scaling_policies
		WHERE id = $1
	`

	policy := &models.QueryScalingPolicy{}
	var bhStart, bhEnd, bhTimezone sql.NullString
	var bhMinReplicas sql.NullInt32

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID, &policy.Name, &policy.QueryEngine, &policy.Enabled,
		&policy.MinReplicas, &policy.MaxReplicas, &policy.CooldownSeconds,
		&policy.ScaleToZero, &policy.QueuedQueriesThreshold,
		&policy.RunningQueriesThreshold, &policy.LatencyThresholdSeconds,
		&policy.ScheduleEnabled, &bhMinReplicas,
		&bhStart, &bhEnd, &bhTimezone,
		&policy.CreatedAt, &policy.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &NotFoundError{Resource: "query scaling policy", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if bhMinReplicas.Valid {
		val := int(bhMinReplicas.Int32)
		policy.BusinessHoursMinReplicas = &val
	}
	if bhStart.Valid {
		policy.BusinessHoursStart = &bhStart.String
	}
	if bhEnd.Valid {
		policy.BusinessHoursEnd = &bhEnd.String
	}
	if bhTimezone.Valid {
		policy.BusinessHoursTimezone = &bhTimezone.String
	}

	return policy, nil
}

// ListPolicies lists all query scaling policies.
func (s *QueryScalingService) ListPolicies(ctx context.Context, queryEngine *models.QueryEngine) (*models.QueryScalingPolicyListResponse, error) {
	query := `
		SELECT id, name, query_engine, enabled, min_replicas, max_replicas,
			cooldown_seconds, scale_to_zero, queued_queries_threshold,
			running_queries_threshold, latency_threshold_seconds,
			schedule_enabled, business_hours_min_replicas,
			business_hours_start, business_hours_end, business_hours_timezone,
			created_at, updated_at
		FROM query_scaling_policies
	`
	args := []interface{}{}

	if queryEngine != nil {
		query += " WHERE query_engine = $1"
		args = append(args, *queryEngine)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer rows.Close()

	policies := []models.QueryScalingPolicy{}
	for rows.Next() {
		var policy models.QueryScalingPolicy
		var bhStart, bhEnd, bhTimezone sql.NullString
		var bhMinReplicas sql.NullInt32

		err := rows.Scan(
			&policy.ID, &policy.Name, &policy.QueryEngine, &policy.Enabled,
			&policy.MinReplicas, &policy.MaxReplicas, &policy.CooldownSeconds,
			&policy.ScaleToZero, &policy.QueuedQueriesThreshold,
			&policy.RunningQueriesThreshold, &policy.LatencyThresholdSeconds,
			&policy.ScheduleEnabled, &bhMinReplicas,
			&bhStart, &bhEnd, &bhTimezone,
			&policy.CreatedAt, &policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		if bhMinReplicas.Valid {
			val := int(bhMinReplicas.Int32)
			policy.BusinessHoursMinReplicas = &val
		}
		if bhStart.Valid {
			policy.BusinessHoursStart = &bhStart.String
		}
		if bhEnd.Valid {
			policy.BusinessHoursEnd = &bhEnd.String
		}
		if bhTimezone.Valid {
			policy.BusinessHoursTimezone = &bhTimezone.String
		}

		policies = append(policies, policy)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	return &models.QueryScalingPolicyListResponse{
		Policies: policies,
		Total:    len(policies),
	}, nil
}

// UpdatePolicy updates a query scaling policy.
func (s *QueryScalingService) UpdatePolicy(ctx context.Context, id uuid.UUID, req *models.UpdateQueryScalingPolicyRequest) (*models.QueryScalingPolicy, error) {
	// Get existing policy
	policy, err := s.GetPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}
	if req.MinReplicas != nil {
		policy.MinReplicas = *req.MinReplicas
	}
	if req.MaxReplicas != nil {
		policy.MaxReplicas = *req.MaxReplicas
	}
	if req.CooldownSeconds != nil {
		policy.CooldownSeconds = *req.CooldownSeconds
	}
	if req.ScaleToZero != nil {
		policy.ScaleToZero = *req.ScaleToZero
	}
	if req.QueuedQueriesThreshold != nil {
		policy.QueuedQueriesThreshold = *req.QueuedQueriesThreshold
	}
	if req.RunningQueriesThreshold != nil {
		policy.RunningQueriesThreshold = *req.RunningQueriesThreshold
	}
	if req.LatencyThresholdSeconds != nil {
		policy.LatencyThresholdSeconds = *req.LatencyThresholdSeconds
	}
	if req.ScheduleEnabled != nil {
		policy.ScheduleEnabled = *req.ScheduleEnabled
	}
	if req.BusinessHoursMinReplicas != nil {
		policy.BusinessHoursMinReplicas = req.BusinessHoursMinReplicas
	}
	if req.BusinessHoursStart != nil {
		policy.BusinessHoursStart = req.BusinessHoursStart
	}
	if req.BusinessHoursEnd != nil {
		policy.BusinessHoursEnd = req.BusinessHoursEnd
	}
	if req.BusinessHoursTimezone != nil {
		policy.BusinessHoursTimezone = req.BusinessHoursTimezone
	}
	policy.UpdatedAt = time.Now()

	// Update in database
	query := `
		UPDATE query_scaling_policies SET
			name = $2, enabled = $3, min_replicas = $4, max_replicas = $5,
			cooldown_seconds = $6, scale_to_zero = $7, queued_queries_threshold = $8,
			running_queries_threshold = $9, latency_threshold_seconds = $10,
			schedule_enabled = $11, business_hours_min_replicas = $12,
			business_hours_start = $13, business_hours_end = $14,
			business_hours_timezone = $15, updated_at = $16
		WHERE id = $1
	`

	_, err = s.db.ExecContext(ctx, query,
		id, policy.Name, policy.Enabled, policy.MinReplicas, policy.MaxReplicas,
		policy.CooldownSeconds, policy.ScaleToZero, policy.QueuedQueriesThreshold,
		policy.RunningQueriesThreshold, policy.LatencyThresholdSeconds,
		policy.ScheduleEnabled, policy.BusinessHoursMinReplicas,
		policy.BusinessHoursStart, policy.BusinessHoursEnd, policy.BusinessHoursTimezone,
		policy.UpdatedAt,
	)
	if err != nil {
		s.logger.Error("failed to update query scaling policy", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update policy: %w", err)
	}

	s.logger.Info("query scaling policy updated", "id", id, "name", policy.Name)
	return policy, nil
}

// DeletePolicy deletes a query scaling policy.
func (s *QueryScalingService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM query_scaling_policies WHERE id = $1", id)
	if err != nil {
		s.logger.Error("failed to delete query scaling policy", "error", err, "id", id)
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("failed to get rows affected", "error", err, "id", id)
		return fmt.Errorf("failed to check deletion result: %w", err)
	}
	if rowsAffected == 0 {
		return &NotFoundError{Resource: "query scaling policy", ID: id.String()}
	}

	s.logger.Info("query scaling policy deleted", "id", id)
	return nil
}

// GetMetrics retrieves current query engine metrics.
func (s *QueryScalingService) GetMetrics(ctx context.Context) (*models.QueryScalingMetricsResponse, error) {
	metrics := []models.QueryScalingMetrics{}

	// Get Trino metrics if query service is available
	if s.queryService != nil {
		status, err := s.queryService.GetStatus(ctx)
		if err == nil && status.Available {
			metrics = append(metrics, models.QueryScalingMetrics{
				QueryEngine:    models.QueryEngineTrino,
				QueuedQueries:  status.QueuedQueries,
				RunningQueries: status.RunningQueries,
				BlockedQueries: status.BlockedQueries,
				ActiveWorkers:  status.ActiveWorkers,
				CollectedAt:    time.Now(),
			})
		}
	}

	return &models.QueryScalingMetricsResponse{
		Metrics: metrics,
	}, nil
}

// GetHistory retrieves query scaling history.
func (s *QueryScalingService) GetHistory(ctx context.Context, policyID *uuid.UUID, queryEngine *models.QueryEngine, limit int) (*models.QueryScalingHistoryResponse, error) {
	query := `
		SELECT id, policy_id, query_engine, action, previous_replicas,
			new_replicas, trigger_reason, trigger_value, executed_at
		FROM query_scaling_history
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if policyID != nil {
		query += fmt.Sprintf(" AND policy_id = $%d", argIdx)
		args = append(args, *policyID)
		argIdx++
	}
	if queryEngine != nil {
		query += fmt.Sprintf(" AND query_engine = $%d", argIdx)
		args = append(args, *queryEngine)
		argIdx++
	}
	query += " ORDER BY executed_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	history := []models.QueryScalingHistoryEntry{}
	for rows.Next() {
		var entry models.QueryScalingHistoryEntry
		var policyID sql.NullString
		var triggerReason sql.NullString
		var triggerValue sql.NullFloat64

		err := rows.Scan(
			&entry.ID, &policyID, &entry.QueryEngine, &entry.Action,
			&entry.PreviousReplicas, &entry.NewReplicas,
			&triggerReason, &triggerValue, &entry.ExecutedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history entry: %w", err)
		}

		if policyID.Valid {
			if parsedID, parseErr := uuid.Parse(policyID.String); parseErr == nil {
				entry.PolicyID = &parsedID
			}
		}
		if triggerReason.Valid {
			entry.TriggerReason = triggerReason.String
		}
		if triggerValue.Valid {
			entry.TriggerValue = &triggerValue.Float64
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history: %w", err)
	}

	return &models.QueryScalingHistoryResponse{
		History: history,
		Total:   len(history),
	}, nil
}

// RecordScalingAction records a scaling action in history.
func (s *QueryScalingService) RecordScalingAction(ctx context.Context, policyID *uuid.UUID, queryEngine models.QueryEngine, action string, previousReplicas, newReplicas int, triggerReason string, triggerValue *float64) error {
	query := `
		INSERT INTO query_scaling_history (
			id, policy_id, query_engine, action, previous_replicas,
			new_replicas, trigger_reason, trigger_value, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.ExecContext(ctx, query,
		uuid.New(), policyID, queryEngine, action,
		previousReplicas, newReplicas, triggerReason, triggerValue,
		time.Now(),
	)
	if err != nil {
		s.logger.Error("failed to record scaling action", "error", err)
		return fmt.Errorf("failed to record scaling action: %w", err)
	}

	return nil
}
