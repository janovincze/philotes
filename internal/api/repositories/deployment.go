// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

// Deployment repository errors.
var (
	ErrDeploymentNotFound  = errors.New("deployment not found")
	ErrDeploymentNameExists = errors.New("deployment with this name already exists")
)

// DeploymentRepository handles database operations for deployments.
type DeploymentRepository struct {
	db *sql.DB
}

// NewDeploymentRepository creates a new DeploymentRepository.
func NewDeploymentRepository(db *sql.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

// deploymentRow represents a database row for a deployment.
type deploymentRow struct {
	ID              uuid.UUID
	UserID          uuid.NullUUID
	Name            string
	Provider        string
	Region          string
	Size            string
	Status          string
	Environment     string
	Config          []byte
	Outputs         []byte
	PulumiStackName sql.NullString
	ErrorMessage    sql.NullString
	StartedAt       sql.NullTime
	CompletedAt     sql.NullTime
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// toModel converts a database row to an API model.
func (r *deploymentRow) toModel() *models.Deployment {
	deployment := &models.Deployment{
		ID:          r.ID,
		Name:        r.Name,
		Provider:    r.Provider,
		Region:      r.Region,
		Size:        models.DeploymentSize(r.Size),
		Status:      models.DeploymentStatus(r.Status),
		Environment: r.Environment,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}

	if r.UserID.Valid {
		deployment.UserID = &r.UserID.UUID
	}
	if r.Config != nil {
		var config models.DeploymentConfig
		if err := json.Unmarshal(r.Config, &config); err != nil {
			slog.Warn("failed to unmarshal deployment config", "deployment_id", r.ID, "error", err)
		} else {
			deployment.Config = &config
		}
	}
	if r.Outputs != nil {
		var outputs models.DeploymentOutput
		if err := json.Unmarshal(r.Outputs, &outputs); err != nil {
			slog.Warn("failed to unmarshal deployment outputs", "deployment_id", r.ID, "error", err)
		} else {
			deployment.Outputs = &outputs
		}
	}
	if r.PulumiStackName.Valid {
		deployment.PulumiStackName = r.PulumiStackName.String
	}
	if r.ErrorMessage.Valid {
		deployment.ErrorMessage = r.ErrorMessage.String
	}
	if r.StartedAt.Valid {
		deployment.StartedAt = &r.StartedAt.Time
	}
	if r.CompletedAt.Valid {
		deployment.CompletedAt = &r.CompletedAt.Time
	}

	return deployment
}

// Create creates a new deployment in the database.
func (r *DeploymentRepository) Create(ctx context.Context, deployment *models.Deployment) (*models.Deployment, error) {
	configJSON, err := json.Marshal(deployment.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var userID interface{}
	if deployment.UserID != nil {
		userID = *deployment.UserID
	}

	query := `
		INSERT INTO philotes.deployments (
			name, user_id, provider, region, size, status, environment, config, pulumi_stack_name
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, name, provider, region, size, status, environment,
			config, outputs, pulumi_stack_name, error_message, started_at, completed_at,
			created_at, updated_at
	`

	var row deploymentRow
	err = r.db.QueryRowContext(ctx, query,
		deployment.Name,
		userID,
		deployment.Provider,
		deployment.Region,
		deployment.Size,
		deployment.Status,
		deployment.Environment,
		configJSON,
		sql.NullString{String: deployment.PulumiStackName, Valid: deployment.PulumiStackName != ""},
	).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.Provider,
		&row.Region,
		&row.Size,
		&row.Status,
		&row.Environment,
		&row.Config,
		&row.Outputs,
		&row.PulumiStackName,
		&row.ErrorMessage,
		&row.StartedAt,
		&row.CompletedAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isDeploymentUniqueViolation(err) {
			return nil, ErrDeploymentNameExists
		}
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a deployment by its ID.
func (r *DeploymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Deployment, error) {
	query := `
		SELECT id, user_id, name, provider, region, size, status, environment,
			config, outputs, pulumi_stack_name, error_message, started_at, completed_at,
			created_at, updated_at
		FROM philotes.deployments
		WHERE id = $1
	`

	var row deploymentRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.Provider,
		&row.Region,
		&row.Size,
		&row.Status,
		&row.Environment,
		&row.Config,
		&row.Outputs,
		&row.PulumiStackName,
		&row.ErrorMessage,
		&row.StartedAt,
		&row.CompletedAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeploymentNotFound
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return row.toModel(), nil
}

// List retrieves all deployments, optionally filtered by user ID.
func (r *DeploymentRepository) List(ctx context.Context, userID *uuid.UUID) ([]models.Deployment, error) {
	var query string
	var args []interface{}

	if userID != nil {
		query = `
			SELECT id, user_id, name, provider, region, size, status, environment,
				config, outputs, pulumi_stack_name, error_message, started_at, completed_at,
				created_at, updated_at
			FROM philotes.deployments
			WHERE user_id = $1
			ORDER BY created_at DESC
		`
		args = []interface{}{*userID}
	} else {
		query = `
			SELECT id, user_id, name, provider, region, size, status, environment,
				config, outputs, pulumi_stack_name, error_message, started_at, completed_at,
				created_at, updated_at
			FROM philotes.deployments
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []models.Deployment
	for rows.Next() {
		var row deploymentRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.Name,
			&row.Provider,
			&row.Region,
			&row.Size,
			&row.Status,
			&row.Environment,
			&row.Config,
			&row.Outputs,
			&row.PulumiStackName,
			&row.ErrorMessage,
			&row.StartedAt,
			&row.CompletedAt,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan deployment row: %w", err)
		}
		deployments = append(deployments, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate deployments: %w", err)
	}

	return deployments, nil
}

// UpdateStatus updates the status of a deployment.
func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.DeploymentStatus, errorMessage string) error {
	var query string
	var args []interface{}

	switch status {
	case models.DeploymentStatusProvisioning, models.DeploymentStatusConfiguring, models.DeploymentStatusDeploying, models.DeploymentStatusVerifying:
		query = `
			UPDATE philotes.deployments
			SET status = $1, started_at = COALESCE(started_at, NOW()), updated_at = NOW()
			WHERE id = $2
		`
		args = []interface{}{status, id}
	case models.DeploymentStatusCompleted:
		query = `
			UPDATE philotes.deployments
			SET status = $1, completed_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`
		args = []interface{}{status, id}
	case models.DeploymentStatusFailed:
		query = `
			UPDATE philotes.deployments
			SET status = $1, error_message = $2, completed_at = NOW(), updated_at = NOW()
			WHERE id = $3
		`
		args = []interface{}{status, errorMessage, id}
	case models.DeploymentStatusCancelled:
		query = `
			UPDATE philotes.deployments
			SET status = $1, completed_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`
		args = []interface{}{status, id}
	default:
		query = `
			UPDATE philotes.deployments
			SET status = $1, updated_at = NOW()
			WHERE id = $2
		`
		args = []interface{}{status, id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeploymentNotFound
	}

	return nil
}

// UpdateOutputs updates the outputs of a deployment.
func (r *DeploymentRepository) UpdateOutputs(ctx context.Context, id uuid.UUID, outputs *models.DeploymentOutput) error {
	outputsJSON, err := json.Marshal(outputs)
	if err != nil {
		return fmt.Errorf("failed to marshal outputs: %w", err)
	}

	query := `
		UPDATE philotes.deployments
		SET outputs = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, outputsJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update deployment outputs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeploymentNotFound
	}

	return nil
}

// UpdatePulumiStackName updates the Pulumi stack name.
func (r *DeploymentRepository) UpdatePulumiStackName(ctx context.Context, id uuid.UUID, stackName string) error {
	query := `
		UPDATE philotes.deployments
		SET pulumi_stack_name = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, stackName, id)
	if err != nil {
		return fmt.Errorf("failed to update pulumi stack name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeploymentNotFound
	}

	return nil
}

// Delete deletes a deployment from the database.
func (r *DeploymentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Logs and credentials are deleted by CASCADE
	query := `DELETE FROM philotes.deployments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrDeploymentNotFound
	}

	return nil
}

// AddLog adds a log entry for a deployment.
func (r *DeploymentRepository) AddLog(ctx context.Context, deploymentID uuid.UUID, level, step, message string) error {
	query := `
		INSERT INTO philotes.deployment_logs (deployment_id, level, step, message)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query, deploymentID, level, step, message)
	if err != nil {
		return fmt.Errorf("failed to add deployment log: %w", err)
	}

	return nil
}

// GetLogs retrieves logs for a deployment.
func (r *DeploymentRepository) GetLogs(ctx context.Context, deploymentID uuid.UUID, limit int) ([]models.DeploymentLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, deployment_id, level, step, message, timestamp
		FROM philotes.deployment_logs
		WHERE deployment_id = $1
		ORDER BY timestamp ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, deploymentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}
	defer rows.Close()

	var logs []models.DeploymentLog
	for rows.Next() {
		var log models.DeploymentLog
		var step sql.NullString
		err := rows.Scan(
			&log.ID,
			&log.DeploymentID,
			&log.Level,
			&step,
			&log.Message,
			&log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		if step.Valid {
			log.Step = step.String
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate logs: %w", err)
	}

	return logs, nil
}

// isDeploymentUniqueViolation checks if an error is a unique violation error.
func isDeploymentUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL unique violation error code is 23505
	return strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "duplicate key value violates unique constraint")
}
