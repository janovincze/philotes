// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/janovincze/philotes/internal/api/models"
)

// Onboarding repository errors.
var (
	ErrOnboardingNotFound = errors.New("onboarding progress not found")
)

// OnboardingRepository handles database operations for onboarding progress.
type OnboardingRepository struct {
	db *sql.DB
}

// NewOnboardingRepository creates a new OnboardingRepository.
func NewOnboardingRepository(db *sql.DB) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

// onboardingRow represents a database row for onboarding progress.
type onboardingRow struct {
	ID             uuid.UUID
	UserID         uuid.NullUUID
	SessionID      sql.NullString
	CurrentStep    int
	CompletedSteps []int64
	StepData       json.RawMessage
	Metrics        json.RawMessage
	StartedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    sql.NullTime
}

// toModel converts a database row to an API model.
func (r *onboardingRow) toModel() (*models.OnboardingProgress, error) {
	progress := &models.OnboardingProgress{
		ID:          r.ID,
		SessionID:   r.SessionID.String,
		CurrentStep: r.CurrentStep,
		StartedAt:   r.StartedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.UserID.Valid {
		progress.UserID = &r.UserID.UUID
	}

	if r.CompletedAt.Valid {
		progress.CompletedAt = &r.CompletedAt.Time
	}

	// Convert []int64 to []int
	progress.CompletedSteps = make([]int, len(r.CompletedSteps))
	for i, v := range r.CompletedSteps {
		progress.CompletedSteps[i] = int(v)
	}

	// Parse step data JSON
	if len(r.StepData) > 0 {
		if err := json.Unmarshal(r.StepData, &progress.StepData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step_data: %w", err)
		}
	} else {
		progress.StepData = make(map[string]interface{})
	}

	// Parse metrics JSON
	if len(r.Metrics) > 0 {
		var metrics models.OnboardingMetrics
		if err := json.Unmarshal(r.Metrics, &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}
		progress.Metrics = &metrics
	}

	return progress, nil
}

// Create creates a new onboarding progress record.
func (r *OnboardingRepository) Create(ctx context.Context, userID *uuid.UUID, sessionID string) (*models.OnboardingProgress, error) {
	query := `
		INSERT INTO philotes.onboarding_progress (user_id, session_id, current_step, completed_steps, step_data, metrics)
		VALUES ($1, $2, 1, '{}', '{}', '{}')
		RETURNING id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
	`

	var row onboardingRow
	err := r.db.QueryRowContext(ctx, query, nullUUID(userID), nullString(sessionID)).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create onboarding progress: %w", err)
	}

	return row.toModel()
}

// GetByID retrieves onboarding progress by ID.
func (r *OnboardingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OnboardingProgress, error) {
	query := `
		SELECT id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
		FROM philotes.onboarding_progress
		WHERE id = $1
	`

	var row onboardingRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOnboardingNotFound
		}
		return nil, fmt.Errorf("failed to get onboarding progress: %w", err)
	}

	return row.toModel()
}

// GetByUserID retrieves the latest onboarding progress for a user.
func (r *OnboardingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.OnboardingProgress, error) {
	query := `
		SELECT id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
		FROM philotes.onboarding_progress
		WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`

	var row onboardingRow
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOnboardingNotFound
		}
		return nil, fmt.Errorf("failed to get onboarding progress by user: %w", err)
	}

	return row.toModel()
}

// GetBySessionID retrieves onboarding progress by session ID.
func (r *OnboardingRepository) GetBySessionID(ctx context.Context, sessionID string) (*models.OnboardingProgress, error) {
	query := `
		SELECT id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
		FROM philotes.onboarding_progress
		WHERE session_id = $1 AND completed_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`

	var row onboardingRow
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOnboardingNotFound
		}
		return nil, fmt.Errorf("failed to get onboarding progress by session: %w", err)
	}

	return row.toModel()
}

// Update updates onboarding progress.
func (r *OnboardingRepository) Update(ctx context.Context, id uuid.UUID, req *models.SaveOnboardingProgressRequest) (*models.OnboardingProgress, error) {
	// First get the existing record to merge step data
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Merge step data
	mergedStepData := existing.StepData
	if mergedStepData == nil {
		mergedStepData = make(map[string]interface{})
	}
	for k, v := range req.StepData {
		mergedStepData[k] = v
	}

	stepDataJSON, err := json.Marshal(mergedStepData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal step_data: %w", err)
	}

	// Update metrics
	metrics := existing.Metrics
	if metrics == nil {
		metrics = &models.OnboardingMetrics{
			TimePerStep: make(map[int]int64),
		}
	}
	if req.StepTimeMs != nil && req.CurrentStep > 0 {
		if metrics.TimePerStep == nil {
			metrics.TimePerStep = make(map[int]int64)
		}
		metrics.TimePerStep[req.CurrentStep] = *req.StepTimeMs
	}
	if req.StepSkipped != nil {
		metrics.StepsSkipped = append(metrics.StepsSkipped, *req.StepSkipped)
	}

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	// Convert completed steps to int64 for pq.Array
	completedSteps := make([]int64, len(req.CompletedSteps))
	for i, v := range req.CompletedSteps {
		completedSteps[i] = int64(v)
	}

	query := `
		UPDATE philotes.onboarding_progress
		SET current_step = $1, completed_steps = $2, step_data = $3, metrics = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
	`

	var row onboardingRow
	err = r.db.QueryRowContext(ctx, query,
		req.CurrentStep,
		pq.Array(completedSteps),
		stepDataJSON,
		metricsJSON,
		id,
	).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update onboarding progress: %w", err)
	}

	return row.toModel()
}

// Complete marks the onboarding as complete.
func (r *OnboardingRepository) Complete(ctx context.Context, id uuid.UUID) (*models.OnboardingProgress, error) {
	// Get current record to calculate total time
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update total time in metrics
	metrics := existing.Metrics
	if metrics == nil {
		metrics = &models.OnboardingMetrics{
			TimePerStep: make(map[int]int64),
		}
	}
	metrics.TotalTimeMs = time.Since(existing.StartedAt).Milliseconds()

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	query := `
		UPDATE philotes.onboarding_progress
		SET completed_at = NOW(), metrics = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, user_id, session_id, current_step, completed_steps, step_data, metrics, started_at, updated_at, completed_at
	`

	var row onboardingRow
	err = r.db.QueryRowContext(ctx, query, metricsJSON, id).Scan(
		&row.ID,
		&row.UserID,
		&row.SessionID,
		&row.CurrentStep,
		pq.Array(&row.CompletedSteps),
		&row.StepData,
		&row.Metrics,
		&row.StartedAt,
		&row.UpdatedAt,
		&row.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to complete onboarding progress: %w", err)
	}

	return row.toModel()
}

// AssociateUser associates a user with an onboarding session.
func (r *OnboardingRepository) AssociateUser(ctx context.Context, id, userID uuid.UUID) error {
	query := `
		UPDATE philotes.onboarding_progress
		SET user_id = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, id)
	if err != nil {
		return fmt.Errorf("failed to associate user with onboarding: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOnboardingNotFound
	}

	return nil
}

// nullUUID converts a UUID pointer to sql.NullUUID.
func nullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}
