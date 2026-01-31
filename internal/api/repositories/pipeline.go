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

// Pipeline repository errors.
var (
	ErrPipelineNotFound     = errors.New("pipeline not found")
	ErrPipelineNameExists   = errors.New("pipeline with this name already exists")
	ErrTableMappingExists   = errors.New("table mapping already exists for this pipeline")
	ErrTableMappingNotFound = errors.New("table mapping not found")
)

// PipelineRepository handles database operations for pipelines.
type PipelineRepository struct {
	db *sql.DB
}

// NewPipelineRepository creates a new PipelineRepository.
func NewPipelineRepository(db *sql.DB) *PipelineRepository {
	return &PipelineRepository{db: db}
}

// pipelineRow represents a database row for a pipeline.
type pipelineRow struct {
	ID           uuid.UUID
	TenantID     sql.NullString
	Name         string
	SourceID     uuid.UUID
	Status       string
	Config       []byte
	ErrorMessage sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    sql.NullTime
	StoppedAt    sql.NullTime
}

// toModel converts a database row to an API model.
func (r *pipelineRow) toModel() *models.Pipeline {
	pipeline := &models.Pipeline{
		ID:        r.ID,
		Name:      r.Name,
		SourceID:  r.SourceID,
		Status:    models.PipelineStatus(r.Status),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	if r.TenantID.Valid {
		if tenantID, err := uuid.Parse(r.TenantID.String); err == nil {
			pipeline.TenantID = &tenantID
		}
	}
	if r.Config != nil {
		if err := json.Unmarshal(r.Config, &pipeline.Config); err != nil {
			slog.Warn("failed to unmarshal pipeline config", "pipeline_id", r.ID, "error", err)
		}
	}
	if r.ErrorMessage.Valid {
		pipeline.ErrorMessage = r.ErrorMessage.String
	}
	if r.StartedAt.Valid {
		pipeline.StartedAt = &r.StartedAt.Time
	}
	if r.StoppedAt.Valid {
		pipeline.StoppedAt = &r.StoppedAt.Time
	}

	return pipeline
}

// tableMappingRow represents a database row for a table mapping.
type tableMappingRow struct {
	ID           uuid.UUID
	PipelineID   uuid.UUID
	SourceSchema string
	SourceTable  string
	Enabled      bool
	Config       []byte
	CreatedAt    time.Time
}

// toModel converts a database row to an API model.
func (r *tableMappingRow) toModel() *models.TableMapping {
	mapping := &models.TableMapping{
		ID:           r.ID,
		PipelineID:   r.PipelineID,
		SourceSchema: r.SourceSchema,
		SourceTable:  r.SourceTable,
		Enabled:      r.Enabled,
		CreatedAt:    r.CreatedAt,
	}

	if r.Config != nil {
		if err := json.Unmarshal(r.Config, &mapping.Config); err != nil {
			slog.Warn("failed to unmarshal table mapping config", "mapping_id", r.ID, "error", err)
		}
	}

	return mapping
}

// Create creates a new pipeline in the database.
func (r *PipelineRepository) Create(ctx context.Context, req *models.CreatePipelineRequest) (*models.Pipeline, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert pipeline
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO philotes.pipelines (name, source_id, status, config)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
	`

	var row pipelineRow
	err = tx.QueryRowContext(ctx, query,
		req.Name,
		req.SourceID,
		models.PipelineStatusStopped,
		configJSON,
	).Scan(
		&row.ID,
		&row.Name,
		&row.SourceID,
		&row.Status,
		&row.Config,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.StartedAt,
		&row.StoppedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPipelineNameExists
		}
		if isForeignKeyViolation(err) {
			return nil, ErrSourceNotFound
		}
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	pipeline := row.toModel()

	// Insert table mappings
	for _, tableReq := range req.Tables {
		mapping, err := r.createTableMappingTx(ctx, tx, row.ID, &tableReq)
		if err != nil {
			return nil, err
		}
		pipeline.Tables = append(pipeline.Tables, *mapping)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return pipeline, nil
}

// createTableMappingTx creates a table mapping within a transaction.
func (r *PipelineRepository) createTableMappingTx(ctx context.Context, tx *sql.Tx, pipelineID uuid.UUID, req *models.CreateTableMappingRequest) (*models.TableMapping, error) {
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal table config: %w", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	query := `
		INSERT INTO philotes.table_mappings (pipeline_id, source_schema, source_table, enabled, config)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, pipeline_id, source_schema, source_table, enabled, config, created_at
	`

	var row tableMappingRow
	err = tx.QueryRowContext(ctx, query,
		pipelineID,
		req.Schema,
		req.Table,
		enabled,
		configJSON,
	).Scan(
		&row.ID,
		&row.PipelineID,
		&row.SourceSchema,
		&row.SourceTable,
		&row.Enabled,
		&row.Config,
		&row.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTableMappingExists
		}
		return nil, fmt.Errorf("failed to create table mapping: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a pipeline by its ID.
func (r *PipelineRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Pipeline, error) {
	query := `
		SELECT id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
		FROM philotes.pipelines
		WHERE id = $1
	`

	var row pipelineRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.SourceID,
		&row.Status,
		&row.Config,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.StartedAt,
		&row.StoppedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	pipeline := row.toModel()

	// Load table mappings
	tables, err := r.GetTableMappings(ctx, id)
	if err != nil {
		return nil, err
	}
	pipeline.Tables = tables

	return pipeline, nil
}

// GetTableMappings retrieves table mappings for a pipeline.
func (r *PipelineRepository) GetTableMappings(ctx context.Context, pipelineID uuid.UUID) ([]models.TableMapping, error) {
	query := `
		SELECT id, pipeline_id, source_schema, source_table, enabled, config, created_at
		FROM philotes.table_mappings
		WHERE pipeline_id = $1
		ORDER BY source_schema, source_table
	`

	rows, err := r.db.QueryContext(ctx, query, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get table mappings: %w", err)
	}
	defer rows.Close()

	var mappings []models.TableMapping
	for rows.Next() {
		var row tableMappingRow
		err := rows.Scan(
			&row.ID,
			&row.PipelineID,
			&row.SourceSchema,
			&row.SourceTable,
			&row.Enabled,
			&row.Config,
			&row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table mapping: %w", err)
		}
		mappings = append(mappings, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate table mappings: %w", err)
	}

	return mappings, nil
}

// List retrieves all pipelines.
func (r *PipelineRepository) List(ctx context.Context) ([]models.Pipeline, error) {
	query := `
		SELECT id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
		FROM philotes.pipelines
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	var pipelines []models.Pipeline
	for rows.Next() {
		var row pipelineRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.SourceID,
			&row.Status,
			&row.Config,
			&row.ErrorMessage,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.StartedAt,
			&row.StoppedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pipeline row: %w", err)
		}
		pipelines = append(pipelines, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pipelines: %w", err)
	}

	return pipelines, nil
}

// Update updates a pipeline in the database.
func (r *PipelineRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdatePipelineRequest) (*models.Pipeline, error) {
	// First check if pipeline exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.pipelines SET updated_at = NOW()`
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

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPipelineNameExists
		}
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	return r.GetByID(ctx, id)
}

// UpdateStatus updates the status of a pipeline.
func (r *PipelineRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PipelineStatus, errorMessage string) error {
	var query string
	var args []any

	switch status {
	case models.PipelineStatusRunning, models.PipelineStatusStarting:
		query = `
			UPDATE philotes.pipelines
			SET status = $1, error_message = NULL, started_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`
		args = []any{status, id}
	case models.PipelineStatusStopped, models.PipelineStatusStopping:
		query = `
			UPDATE philotes.pipelines
			SET status = $1, stopped_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`
		args = []any{status, id}
	case models.PipelineStatusError:
		query = `
			UPDATE philotes.pipelines
			SET status = $1, error_message = $2, stopped_at = NOW(), updated_at = NOW()
			WHERE id = $3
		`
		args = []any{status, errorMessage, id}
	default:
		query = `
			UPDATE philotes.pipelines
			SET status = $1, updated_at = NOW()
			WHERE id = $2
		`
		args = []any{status, id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPipelineNotFound
	}

	return nil
}

// Delete deletes a pipeline from the database.
func (r *PipelineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Table mappings are deleted by CASCADE
	query := `DELETE FROM philotes.pipelines WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrPipelineNotFound
	}

	return nil
}

// AddTableMapping adds a table mapping to a pipeline.
func (r *PipelineRepository) AddTableMapping(ctx context.Context, pipelineID uuid.UUID, req *models.AddTableMappingRequest) (*models.TableMapping, error) {
	// Check if pipeline exists
	_, err := r.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal table config: %w", err)
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	query := `
		INSERT INTO philotes.table_mappings (pipeline_id, source_schema, source_table, enabled, config)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, pipeline_id, source_schema, source_table, enabled, config, created_at
	`

	var row tableMappingRow
	err = r.db.QueryRowContext(ctx, query,
		pipelineID,
		req.Schema,
		req.Table,
		enabled,
		configJSON,
	).Scan(
		&row.ID,
		&row.PipelineID,
		&row.SourceSchema,
		&row.SourceTable,
		&row.Enabled,
		&row.Config,
		&row.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTableMappingExists
		}
		return nil, fmt.Errorf("failed to add table mapping: %w", err)
	}

	return row.toModel(), nil
}

// RemoveTableMapping removes a table mapping from a pipeline.
func (r *PipelineRepository) RemoveTableMapping(ctx context.Context, pipelineID, mappingID uuid.UUID) error {
	query := `DELETE FROM philotes.table_mappings WHERE id = $1 AND pipeline_id = $2`

	result, err := r.db.ExecContext(ctx, query, mappingID, pipelineID)
	if err != nil {
		return fmt.Errorf("failed to remove table mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTableMappingNotFound
	}

	return nil
}

// isForeignKeyViolation checks if an error is a foreign key violation.
func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL foreign key violation error code is 23503
	return strings.Contains(err.Error(), "23503")
}

// --- Tenant-scoped operations ---

// CreateWithTenant creates a new pipeline with a tenant ID.
func (r *PipelineRepository) CreateWithTenant(ctx context.Context, req *models.CreatePipelineRequest, tenantID uuid.UUID) (*models.Pipeline, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Insert pipeline
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO philotes.pipelines (tenant_id, name, source_id, status, config)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
	`

	var row pipelineRow
	err = tx.QueryRowContext(ctx, query,
		tenantID,
		req.Name,
		req.SourceID,
		models.PipelineStatusStopped,
		configJSON,
	).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.SourceID,
		&row.Status,
		&row.Config,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.StartedAt,
		&row.StoppedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPipelineNameExists
		}
		if isForeignKeyViolation(err) {
			return nil, ErrSourceNotFound
		}
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	pipeline := row.toModel()

	// Insert table mappings
	for _, tableReq := range req.Tables {
		mapping, err := r.createTableMappingTx(ctx, tx, row.ID, &tableReq)
		if err != nil {
			return nil, err
		}
		pipeline.Tables = append(pipeline.Tables, *mapping)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return pipeline, nil
}

// ListByTenant retrieves all pipelines for a specific tenant.
func (r *PipelineRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.Pipeline, error) {
	query := `
		SELECT id, tenant_id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
		FROM philotes.pipelines
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines by tenant: %w", err)
	}
	defer rows.Close()

	var pipelines []models.Pipeline
	for rows.Next() {
		var row pipelineRow
		err := rows.Scan(
			&row.ID,
			&row.TenantID,
			&row.Name,
			&row.SourceID,
			&row.Status,
			&row.Config,
			&row.ErrorMessage,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.StartedAt,
			&row.StoppedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pipeline row: %w", err)
		}
		pipelines = append(pipelines, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pipelines: %w", err)
	}

	return pipelines, nil
}

// GetByIDAndTenant retrieves a pipeline by ID within a specific tenant.
func (r *PipelineRepository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*models.Pipeline, error) {
	query := `
		SELECT id, tenant_id, name, source_id, status, config, error_message,
			created_at, updated_at, started_at, stopped_at
		FROM philotes.pipelines
		WHERE id = $1 AND tenant_id = $2
	`

	var row pipelineRow
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.SourceID,
		&row.Status,
		&row.Config,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.StartedAt,
		&row.StoppedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	pipeline := row.toModel()

	// Load table mappings
	tables, err := r.GetTableMappings(ctx, id)
	if err != nil {
		return nil, err
	}
	pipeline.Tables = tables

	return pipeline, nil
}
