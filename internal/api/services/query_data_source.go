// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
)

// QueryDataSourceService provides business logic for query data source operations.
type QueryDataSourceService struct {
	repo         *repositories.QueryDataSourceRepository
	queryService *QueryService
	logger       *slog.Logger
}

// NewQueryDataSourceService creates a new QueryDataSourceService.
func NewQueryDataSourceService(
	repo *repositories.QueryDataSourceRepository,
	queryService *QueryService,
	logger *slog.Logger,
) *QueryDataSourceService {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueryDataSourceService{
		repo:         repo,
		queryService: queryService,
		logger:       logger.With("component", "query-data-source-service"),
	}
}

// Create creates a new query data source.
func (s *QueryDataSourceService) Create(ctx context.Context, req *models.CreateQueryDataSourceRequest) (*models.QueryDataSource, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Create in database
	ds, err := s.repo.Create(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNameExists) {
			return nil, &ConflictError{Message: "query data source with this name already exists"}
		}
		if errors.Is(err, repositories.ErrQueryDataSourceCatalogExists) {
			return nil, &ConflictError{Message: "query data source with this catalog name already exists"}
		}
		s.logger.Error("failed to create query data source", "error", err)
		return nil, fmt.Errorf("failed to create query data source: %w", err)
	}

	// Create Trino catalog
	if err := s.createTrinoCatalog(ctx, ds, req.Password); err != nil {
		s.logger.Error("failed to create Trino catalog", "id", ds.ID, "catalog", ds.CatalogName, "error", err)
		// Update status to error
		if statusErr := s.repo.UpdateStatus(ctx, ds.ID, string(models.QueryDataSourceStatusError), err.Error()); statusErr != nil {
			s.logger.Error("failed to update status to error", "id", ds.ID, "error", statusErr)
		}
		ds.Status = models.QueryDataSourceStatusError
		ds.ErrorMessage = err.Error()
	} else {
		// Update status to active
		if statusErr := s.repo.UpdateStatus(ctx, ds.ID, string(models.QueryDataSourceStatusActive), ""); statusErr != nil {
			s.logger.Error("failed to update status to active", "id", ds.ID, "error", statusErr)
		}
		ds.Status = models.QueryDataSourceStatusActive
	}

	s.logger.Info("query data source created", "id", ds.ID, "name", ds.Name, "catalog", ds.CatalogName)
	return ds, nil
}

// List retrieves all query data sources.
func (s *QueryDataSourceService) List(ctx context.Context) ([]models.QueryDataSource, error) {
	dataSources, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list query data sources: %w", err)
	}
	if dataSources == nil {
		dataSources = []models.QueryDataSource{}
	}
	return dataSources, nil
}

// Get retrieves a query data source by ID.
func (s *QueryDataSourceService) Get(ctx context.Context, id uuid.UUID) (*models.QueryDataSource, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return nil, &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get query data source: %w", err)
	}
	return ds, nil
}

// Update updates a query data source.
func (s *QueryDataSourceService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateQueryDataSourceRequest) (*models.QueryDataSource, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Get existing data source to check it exists and get catalog name
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return nil, &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get query data source: %w", err)
	}

	// Update in database
	ds, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return nil, &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrQueryDataSourceNameExists) {
			return nil, &ConflictError{Message: "query data source with this name already exists"}
		}
		return nil, fmt.Errorf("failed to update query data source: %w", err)
	}

	// If connection-related fields changed, recreate the Trino catalog
	if req.HasConnectionChanges() {
		s.logger.Info("connection changes detected, recreating Trino catalog", "id", id, "catalog", existing.CatalogName)

		// Drop existing catalog (ignore errors)
		if err := s.dropTrinoCatalog(ctx, existing.CatalogName); err != nil {
			s.logger.Warn("failed to drop existing Trino catalog during update", "catalog", existing.CatalogName, "error", err)
		}

		// Get updated data source with password
		updatedDS, password, err := s.repo.GetByIDWithPassword(ctx, id)
		if err != nil {
			s.logger.Error("failed to get updated data source with password", "id", id, "error", err)
			return nil, fmt.Errorf("failed to get updated data source: %w", err)
		}

		// Recreate catalog
		if err := s.createTrinoCatalog(ctx, updatedDS, password); err != nil {
			s.logger.Error("failed to recreate Trino catalog", "id", id, "catalog", updatedDS.CatalogName, "error", err)
			if statusErr := s.repo.UpdateStatus(ctx, id, string(models.QueryDataSourceStatusError), err.Error()); statusErr != nil {
				s.logger.Error("failed to update status to error", "id", id, "error", statusErr)
			}
			ds.Status = models.QueryDataSourceStatusError
			ds.ErrorMessage = err.Error()
		} else {
			if statusErr := s.repo.UpdateStatus(ctx, id, string(models.QueryDataSourceStatusActive), ""); statusErr != nil {
				s.logger.Error("failed to update status to active", "id", id, "error", statusErr)
			}
			ds.Status = models.QueryDataSourceStatusActive
			ds.ErrorMessage = ""
		}
	}

	s.logger.Info("query data source updated", "id", ds.ID, "name", ds.Name)
	return ds, nil
}

// Delete deletes a query data source.
func (s *QueryDataSourceService) Delete(ctx context.Context, id uuid.UUID) error {
	// Get existing to find catalog name
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		return fmt.Errorf("failed to get query data source: %w", err)
	}

	// Drop Trino catalog (ignore errors — catalog might not exist)
	if err := s.dropTrinoCatalog(ctx, existing.CatalogName); err != nil {
		s.logger.Warn("failed to drop Trino catalog during delete", "catalog", existing.CatalogName, "error", err)
	}

	// Delete from database
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		return fmt.Errorf("failed to delete query data source: %w", err)
	}

	s.logger.Info("query data source deleted", "id", id, "catalog", existing.CatalogName)
	return nil
}

// TestConnection tests the connection to a query data source via Trino.
func (s *QueryDataSourceService) TestConnection(ctx context.Context, id uuid.UUID) (*models.QueryDataSourceTestResult, error) {
	ds, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrQueryDataSourceNotFound) {
			return nil, &NotFoundError{Resource: "query data source", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get query data source: %w", err)
	}

	if ds.Status != models.QueryDataSourceStatusActive {
		return &models.QueryDataSourceTestResult{
			Success:     false,
			Message:     "Data source is not active",
			ErrorDetail: fmt.Sprintf("Current status: %s. %s", ds.Status, ds.ErrorMessage),
		}, nil
	}

	// Test connectivity by querying via Trino
	testQuery := fmt.Sprintf("SELECT 1 FROM %s.information_schema.tables LIMIT 1", ds.CatalogName)

	start := time.Now()
	err = s.queryService.ExecuteAdminQuery(ctx, testQuery)
	latency := time.Since(start)

	if err != nil {
		s.logger.Error("connection test failed", "id", id, "catalog", ds.CatalogName, "error", err)
		return &models.QueryDataSourceTestResult{
			Success:     false,
			Message:     "Connection test failed",
			LatencyMs:   latency.Milliseconds(),
			ErrorDetail: err.Error(),
		}, nil
	}

	s.logger.Info("connection test successful", "id", id, "catalog", ds.CatalogName, "latency_ms", latency.Milliseconds())
	return &models.QueryDataSourceTestResult{
		Success:   true,
		Message:   "Connection successful",
		LatencyMs: latency.Milliseconds(),
	}, nil
}

// createTrinoCatalog creates a dynamic Trino catalog for the data source.
func (s *QueryDataSourceService) createTrinoCatalog(ctx context.Context, ds *models.QueryDataSource, password string) error {
	var connectorName, jdbcURL string

	switch ds.Type {
	case "postgresql":
		connectorName = "postgresql"
		jdbcURL = fmt.Sprintf("jdbc:postgresql://%s:%d/%s", ds.Host, ds.Port, ds.DatabaseName)
		if ds.SSLMode != "" {
			jdbcURL += "?sslmode=" + ds.SSLMode
		}
	case "mysql":
		connectorName = "mysql"
		jdbcURL = fmt.Sprintf("jdbc:mysql://%s:%d/%s", ds.Host, ds.Port, ds.DatabaseName)
		if ds.SSLMode != "" && ds.SSLMode != "prefer" {
			// MySQL JDBC uses useSSL/requireSSL parameters
			switch ds.SSLMode {
			case "disable":
				jdbcURL += "?useSSL=false"
			case "require", "verify-ca", "verify-full":
				jdbcURL += "?useSSL=true&requireSSL=true"
			}
		}
	default:
		return fmt.Errorf("unsupported data source type: %s", ds.Type)
	}

	// Escape single quotes in connection string values
	escapedURL := strings.ReplaceAll(jdbcURL, "'", "''")
	escapedUser := strings.ReplaceAll(ds.Username, "'", "''")
	escapedPassword := strings.ReplaceAll(password, "'", "''")

	query := fmt.Sprintf(
		`CREATE CATALOG %s USING %s WITH ("connection-url" = '%s', "connection-user" = '%s', "connection-password" = '%s')`,
		ds.CatalogName, connectorName, escapedURL, escapedUser, escapedPassword,
	)

	return s.queryService.ExecuteAdminQuery(ctx, query)
}

// dropTrinoCatalog drops a dynamic Trino catalog.
func (s *QueryDataSourceService) dropTrinoCatalog(ctx context.Context, catalogName string) error {
	query := fmt.Sprintf("DROP CATALOG IF EXISTS %s", catalogName)
	return s.queryService.ExecuteAdminQuery(ctx, query)
}
