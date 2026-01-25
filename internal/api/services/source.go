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
	"github.com/janovincze/philotes/internal/api/repositories"
)

// SourceService provides business logic for source operations.
type SourceService struct {
	repo   *repositories.SourceRepository
	logger *slog.Logger
}

// NewSourceService creates a new SourceService.
func NewSourceService(repo *repositories.SourceRepository, logger *slog.Logger) *SourceService {
	return &SourceService{
		repo:   repo,
		logger: logger.With("component", "source-service"),
	}
}

// Create creates a new source.
func (s *SourceService) Create(ctx context.Context, req *models.CreateSourceRequest) (*models.Source, error) {
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		return nil, &ValidationError{Errors: errors}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Create source
	source, err := s.repo.Create(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNameExists) {
			return nil, &ConflictError{Message: "source with this name already exists"}
		}
		s.logger.Error("failed to create source", "error", err)
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	s.logger.Info("source created", "id", source.ID, "name", source.Name)
	return source, nil
}

// Get retrieves a source by ID.
func (s *SourceService) Get(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return nil, &NotFoundError{Resource: "source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}
	return source, nil
}

// List retrieves all sources.
func (s *SourceService) List(ctx context.Context) ([]models.Source, error) {
	sources, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	if sources == nil {
		sources = []models.Source{}
	}
	return sources, nil
}

// Update updates a source.
func (s *SourceService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateSourceRequest) (*models.Source, error) {
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		return nil, &ValidationError{Errors: errors}
	}

	// Update source
	source, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return nil, &NotFoundError{Resource: "source", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrSourceNameExists) {
			return nil, &ConflictError{Message: "source with this name already exists"}
		}
		return nil, fmt.Errorf("failed to update source: %w", err)
	}

	s.logger.Info("source updated", "id", source.ID, "name", source.Name)
	return source, nil
}

// Delete deletes a source.
func (s *SourceService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return &NotFoundError{Resource: "source", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrSourceHasPipelines) {
			return &ConflictError{Message: "cannot delete source with associated pipelines"}
		}
		return fmt.Errorf("failed to delete source: %w", err)
	}

	s.logger.Info("source deleted", "id", id)
	return nil
}

// TestConnection tests the connection to a source database.
func (s *SourceService) TestConnection(ctx context.Context, id uuid.UUID) (*models.ConnectionTestResult, error) {
	// Get source with password
	source, password, err := s.repo.GetByIDWithPassword(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return nil, &NotFoundError{Resource: "source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	// Build connection string (password is not logged in errors from sql.Open/Ping)
	dsn := buildDSN(source.Host, source.Port, source.DatabaseName, source.Username, password, source.SSLMode)

	// Test connection with timeout
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		// Don't expose internal error details that might contain connection info
		s.logger.Error("failed to open database connection", "source_id", id, "error", err)
		return &models.ConnectionTestResult{
			Success:     false,
			Message:     "Failed to open connection",
			ErrorDetail: "Could not initialize database driver",
		}, nil
	}
	defer db.Close()

	// Ping database
	if err := db.PingContext(testCtx); err != nil {
		// Sanitize error message to avoid leaking sensitive info
		s.logger.Error("failed to ping database", "source_id", id, "error", err)
		return &models.ConnectionTestResult{
			Success:     false,
			Message:     "Failed to connect to database",
			ErrorDetail: sanitizeConnectionError(err),
		}, nil
	}

	latency := time.Since(start)

	// Get server version
	var version string
	err = db.QueryRowContext(testCtx, "SELECT version()").Scan(&version)
	if err != nil {
		version = "unknown"
	}

	// Update source status
	var statusWarning string
	if err := s.repo.UpdateStatus(ctx, id, models.SourceStatusActive); err != nil {
		s.logger.Warn("failed to update source status", "id", id, "error", err)
		statusWarning = " (warning: status update failed)"
	}

	s.logger.Info("connection test successful", "id", id, "latency_ms", latency.Milliseconds())

	return &models.ConnectionTestResult{
		Success:    true,
		Message:    "Connection successful" + statusWarning,
		LatencyMs:  latency.Milliseconds(),
		ServerInfo: version,
	}, nil
}

// buildDSN constructs a PostgreSQL connection string.
func buildDSN(host string, port int, dbname, user, password, sslmode string) string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		host, port, dbname, user, password, sslmode,
	)
}

// sanitizeConnectionError removes potentially sensitive information from connection errors.
func sanitizeConnectionError(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	// Common safe error patterns to pass through
	safePatterns := []string{
		"connection refused",
		"no such host",
		"timeout",
		"network is unreachable",
		"connection reset",
		"authentication failed",
		"password authentication failed",
		"database",
		"does not exist",
		"SSL",
	}
	for _, pattern := range safePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return errStr
		}
	}
	// Generic message for unknown errors
	return "connection failed"
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && findSubstringIgnoreCase(s, substr))
}

func findSubstringIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// DiscoverTables discovers tables in a source database.
func (s *SourceService) DiscoverTables(ctx context.Context, id uuid.UUID, schema string) (*models.TableDiscoveryResponse, error) {
	// Get source with password
	source, password, err := s.repo.GetByIDWithPassword(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return nil, &NotFoundError{Resource: "source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	// Default schema
	if schema == "" {
		schema = "public"
	}

	// Build connection string
	dsn := buildDSN(source.Host, source.Port, source.DatabaseName, source.Username, password, source.SSLMode)

	// Connect with timeout
	discoverCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		s.logger.Error("failed to open connection for table discovery", "source_id", id, "error", err)
		return nil, fmt.Errorf("failed to open connection to source database")
	}
	defer db.Close()

	// Query tables
	tableQuery := `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.QueryContext(discoverCtx, tableQuery, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []models.TableInfo
	for rows.Next() {
		var tableSchema, tableName string
		if err := rows.Scan(&tableSchema, &tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		// Get columns for this table
		columns, err := s.discoverColumns(discoverCtx, db, tableSchema, tableName)
		if err != nil {
			s.logger.Warn("failed to discover columns", "table", tableName, "error", err)
			columns = nil
		}

		tables = append(tables, models.TableInfo{
			Schema:  tableSchema,
			Name:    tableName,
			Columns: columns,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tables: %w", err)
	}

	s.logger.Info("table discovery completed", "id", id, "schema", schema, "count", len(tables))

	return &models.TableDiscoveryResponse{
		Tables: tables,
		Count:  len(tables),
	}, nil
}

// discoverColumns discovers columns for a table.
func (s *SourceService) discoverColumns(ctx context.Context, db *sql.DB, schema, table string) ([]models.ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable,
			c.column_default,
			COALESCE(
				(SELECT true FROM information_schema.key_column_usage kcu
				 JOIN information_schema.table_constraints tc
				   ON kcu.constraint_name = tc.constraint_name
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				   AND kcu.table_schema = c.table_schema
				   AND kcu.table_name = c.table_name
				   AND kcu.column_name = c.column_name
				 LIMIT 1), false
			) as is_primary_key
		FROM information_schema.columns c
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var columnDefault sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &columnDefault, &col.PrimaryKey); err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}

		if columnDefault.Valid {
			col.Default = &columnDefault.String
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate columns: %w", err)
	}

	return columns, nil
}

// Service errors.

// ValidationError represents a validation error.
type ValidationError struct {
	Errors []models.FieldError
}

func (e *ValidationError) Error() string {
	return "validation error"
}

// NotFoundError represents a not found error.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ConflictError represents a conflict error.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}
