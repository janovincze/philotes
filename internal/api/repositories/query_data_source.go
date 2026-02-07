// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

// Common repository errors for query data sources.
var (
	ErrQueryDataSourceNotFound      = errors.New("query data source not found")
	ErrQueryDataSourceNameExists    = errors.New("query data source with this name already exists")
	ErrQueryDataSourceCatalogExists = errors.New("query data source with this catalog name already exists")
)

// QueryDataSourceRepository handles database operations for query data sources.
type QueryDataSourceRepository struct {
	db *sql.DB
}

// NewQueryDataSourceRepository creates a new QueryDataSourceRepository.
func NewQueryDataSourceRepository(db *sql.DB) *QueryDataSourceRepository {
	return &QueryDataSourceRepository{db: db}
}

// queryDataSourceRow represents a database row for a query data source.
type queryDataSourceRow struct {
	ID           uuid.UUID
	TenantID     sql.NullString
	Name         string
	Type         string
	CatalogName  string
	Host         string
	Port         int
	DatabaseName string
	Username     string
	Password     string
	SSLMode      string
	ExtraConfig  sql.NullString
	Status       string
	ErrorMessage sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// toModel converts a database row to an API model.
// Note: password is intentionally NOT copied to the model.
func (r *queryDataSourceRow) toModel() *models.QueryDataSource {
	ds := &models.QueryDataSource{
		ID:           r.ID,
		Name:         r.Name,
		Type:         r.Type,
		CatalogName:  r.CatalogName,
		Host:         r.Host,
		Port:         r.Port,
		DatabaseName: r.DatabaseName,
		Username:     r.Username,
		SSLMode:      r.SSLMode,
		Status:       models.QueryDataSourceStatus(r.Status),
		ErrorMessage: r.ErrorMessage.String,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if r.TenantID.Valid {
		if tenantID, err := uuid.Parse(r.TenantID.String); err == nil {
			ds.TenantID = &tenantID
		}
	}
	if r.ExtraConfig.Valid && r.ExtraConfig.String != "" {
		var extraConfig map[string]interface{}
		if err := json.Unmarshal([]byte(r.ExtraConfig.String), &extraConfig); err == nil {
			ds.ExtraConfig = extraConfig
		}
	}
	return ds
}

// Create creates a new query data source in the database.
func (r *QueryDataSourceRepository) Create(ctx context.Context, req *models.CreateQueryDataSourceRequest) (*models.QueryDataSource, error) {
	var extraConfigJSON sql.NullString
	if req.ExtraConfig != nil {
		data, err := json.Marshal(req.ExtraConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal extra_config: %w", err)
		}
		extraConfigJSON = sql.NullString{String: string(data), Valid: true}
	}

	query := `
		INSERT INTO philotes.query_data_sources (
			name, type, catalog_name, host, port, database_name, username, password,
			ssl_mode, extra_config, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, name, type, catalog_name, host, port, database_name, username, password,
			ssl_mode, extra_config, status, error_message, created_at, updated_at
	`

	var row queryDataSourceRow
	err := r.db.QueryRowContext(ctx, query,
		req.Name,
		req.Type,
		req.CatalogName,
		req.Host,
		req.Port,
		req.DatabaseName,
		req.Username,
		req.Password,
		req.SSLMode,
		extraConfigJSON,
		models.QueryDataSourceStatusInactive,
	).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.CatalogName,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.ExtraConfig,
		&row.Status,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			errStr := err.Error()
			if strings.Contains(errStr, "catalog_name") {
				return nil, ErrQueryDataSourceCatalogExists
			}
			return nil, ErrQueryDataSourceNameExists
		}
		return nil, fmt.Errorf("failed to create query data source: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a query data source by its ID (without password).
func (r *QueryDataSourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.QueryDataSource, error) {
	query := `
		SELECT id, name, type, catalog_name, host, port, database_name, username, password,
			ssl_mode, extra_config, status, error_message, created_at, updated_at
		FROM philotes.query_data_sources
		WHERE id = $1
	`

	var row queryDataSourceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.CatalogName,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.ExtraConfig,
		&row.Status,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQueryDataSourceNotFound
		}
		return nil, fmt.Errorf("failed to get query data source: %w", err)
	}

	return row.toModel(), nil
}

// GetByIDWithPassword retrieves a query data source by ID including the password.
func (r *QueryDataSourceRepository) GetByIDWithPassword(ctx context.Context, id uuid.UUID) (*models.QueryDataSource, string, error) {
	query := `
		SELECT id, name, type, catalog_name, host, port, database_name, username, password,
			ssl_mode, extra_config, status, error_message, created_at, updated_at
		FROM philotes.query_data_sources
		WHERE id = $1
	`

	var row queryDataSourceRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.CatalogName,
		&row.Host,
		&row.Port,
		&row.DatabaseName,
		&row.Username,
		&row.Password,
		&row.SSLMode,
		&row.ExtraConfig,
		&row.Status,
		&row.ErrorMessage,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrQueryDataSourceNotFound
		}
		return nil, "", fmt.Errorf("failed to get query data source: %w", err)
	}

	return row.toModel(), row.Password, nil
}

// List retrieves all query data sources (without passwords).
func (r *QueryDataSourceRepository) List(ctx context.Context) ([]models.QueryDataSource, error) {
	query := `
		SELECT id, name, type, catalog_name, host, port, database_name, username, password,
			ssl_mode, extra_config, status, error_message, created_at, updated_at
		FROM philotes.query_data_sources
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list query data sources: %w", err)
	}
	defer rows.Close()

	var dataSources []models.QueryDataSource
	for rows.Next() {
		var row queryDataSourceRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Type,
			&row.CatalogName,
			&row.Host,
			&row.Port,
			&row.DatabaseName,
			&row.Username,
			&row.Password,
			&row.SSLMode,
			&row.ExtraConfig,
			&row.Status,
			&row.ErrorMessage,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query data source row: %w", err)
		}
		dataSources = append(dataSources, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate query data sources: %w", err)
	}

	return dataSources, nil
}

// Update updates a query data source in the database.
func (r *QueryDataSourceRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateQueryDataSourceRequest) (*models.QueryDataSource, error) {
	// First check if data source exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.query_data_sources SET updated_at = NOW()`
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
	if req.ExtraConfig != nil {
		data, err := json.Marshal(req.ExtraConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal extra_config: %w", err)
		}
		query += fmt.Sprintf(", extra_config = $%d", argIdx)
		args = append(args, string(data))
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrQueryDataSourceNameExists
		}
		return nil, fmt.Errorf("failed to update query data source: %w", err)
	}

	return r.GetByID(ctx, id)
}

// UpdateStatus updates the status and error message of a query data source.
func (r *QueryDataSourceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage string) error {
	query := `
		UPDATE philotes.query_data_sources
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, status, nullString(errorMessage), id)
	if err != nil {
		return fmt.Errorf("failed to update query data source status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrQueryDataSourceNotFound
	}

	return nil
}

// Delete deletes a query data source from the database.
func (r *QueryDataSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.query_data_sources WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete query data source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrQueryDataSourceNotFound
	}

	return nil
}
