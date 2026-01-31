// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

// Common repository errors.
var (
	ErrSourceNotFound     = errors.New("source not found")
	ErrSourceNameExists   = errors.New("source with this name already exists")
	ErrSourceHasPipelines = errors.New("source has associated pipelines")
)

// SourceRepository handles database operations for sources.
type SourceRepository struct {
	db *sql.DB
}

// NewSourceRepository creates a new SourceRepository.
func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

// sourceRow represents a database row for a source.
type sourceRow struct {
	ID              uuid.UUID
	Name            string
	Type            string
	Host            string
	Port            int
	DatabaseName    string
	Username        string
	Password        string
	SSLMode         string
	SlotName        sql.NullString
	PublicationName sql.NullString
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// toModel converts a database row to an API model.
func (r *sourceRow) toModel() *models.Source {
	return &models.Source{
		ID:              r.ID,
		Name:            r.Name,
		Type:            r.Type,
		Host:            r.Host,
		Port:            r.Port,
		DatabaseName:    r.DatabaseName,
		Username:        r.Username,
		SSLMode:         r.SSLMode,
		SlotName:        r.SlotName.String,
		PublicationName: r.PublicationName.String,
		Status:          models.SourceStatus(r.Status),
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// Create creates a new source in the database.
func (r *SourceRepository) Create(ctx context.Context, req *models.CreateSourceRequest) (*models.Source, error) {
	query := `
		INSERT INTO philotes.sources (
			name, type, host, port, database_name, username, password,
			ssl_mode, slot_name, publication_name, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, name, type, host, port, database_name, username, password,
			ssl_mode, slot_name, publication_name, status, created_at, updated_at
	`

	var row sourceRow
	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Type,
		req.Host,
		req.Port,
		req.DatabaseName,
		req.Username,
		req.Password,
		req.SSLMode,
		nullString(req.SlotName),
		nullString(req.PublicationName),
		models.SourceStatusInactive,
	).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.SlotName,
		&row.PublicationName,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSourceNameExists
		}
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a source by its ID.
func (r *SourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	query := `
		SELECT id, name, type, host, port, database_name, username, password,
			ssl_mode, slot_name, publication_name, status, created_at, updated_at
		FROM philotes.sources
		WHERE id = $1
	`

	var row sourceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.SlotName,
		&row.PublicationName,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSourceNotFound
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	return row.toModel(), nil
}

// GetByIDWithPassword retrieves a source by its ID including the password.
func (r *SourceRepository) GetByIDWithPassword(ctx context.Context, id uuid.UUID) (*models.Source, string, error) {
	query := `
		SELECT id, name, type, host, port, database_name, username, password,
			ssl_mode, slot_name, publication_name, status, created_at, updated_at
		FROM philotes.sources
		WHERE id = $1
	`

	var row sourceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.SlotName,
		&row.PublicationName,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrSourceNotFound
		}
		return nil, "", fmt.Errorf("failed to get source: %w", err)
	}

	return row.toModel(), row.Password, nil
}

// List retrieves all sources.
func (r *SourceRepository) List(ctx context.Context) ([]models.Source, error) {
	query := `
		SELECT id, name, type, host, port, database_name, username, password,
			ssl_mode, slot_name, publication_name, status, created_at, updated_at
		FROM philotes.sources
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var row sourceRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Type,
			&row.Host,
			&row.Port,
			&row.DatabaseName,
			&row.Username,
			&row.Password,
			&row.SSLMode,
			&row.SlotName,
			&row.PublicationName,
			&row.Status,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source row: %w", err)
		}
		sources = append(sources, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sources: %w", err)
	}

	return sources, nil
}

// Update updates a source in the database.
func (r *SourceRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateSourceRequest) (*models.Source, error) {
	// First check if source exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.sources SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Host != nil {
		query += fmt.Sprintf(", host = $%d", argIdx)
		args = append(args, *req.Host)
		argIdx++
	}
	if req.Port != nil {
		query += fmt.Sprintf(", port = $%d", argIdx)
		args = append(args, *req.Port)
		argIdx++
	}
	if req.DatabaseName != nil {
		query += fmt.Sprintf(", database_name = $%d", argIdx)
		args = append(args, *req.DatabaseName)
		argIdx++
	}
	if req.Username != nil {
		query += fmt.Sprintf(", username = $%d", argIdx)
		args = append(args, *req.Username)
		argIdx++
	}
	if req.Password != nil {
		query += fmt.Sprintf(", password = $%d", argIdx)
		args = append(args, *req.Password)
		argIdx++
	}
	if req.SSLMode != nil {
		query += fmt.Sprintf(", ssl_mode = $%d", argIdx)
		args = append(args, *req.SSLMode)
		argIdx++
	}
	if req.SlotName != nil {
		query += fmt.Sprintf(", slot_name = $%d", argIdx)
		args = append(args, nullString(*req.SlotName))
		argIdx++
	}
	if req.PublicationName != nil {
		query += fmt.Sprintf(", publication_name = $%d", argIdx)
		args = append(args, nullString(*req.PublicationName))
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSourceNameExists
		}
		return nil, fmt.Errorf("failed to update source: %w", err)
	}

	return r.GetByID(ctx, id)
}

// UpdateStatus updates the status of a source.
func (r *SourceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.SourceStatus) error {
	query := `
		UPDATE philotes.sources
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update source status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSourceNotFound
	}

	return nil
}

// Delete deletes a source from the database.
func (r *SourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check for associated pipelines
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM philotes.pipelines WHERE source_id = $1", id,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for pipelines: %w", err)
	}
	if count > 0 {
		return ErrSourceHasPipelines
	}

	query := `DELETE FROM philotes.sources WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSourceNotFound
	}

	return nil
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// isUniqueViolation checks if an error is a unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// PostgreSQL unique violation error code is 23505
	return strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "duplicate key value violates unique constraint")
}
