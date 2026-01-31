// Package services provides business logic for the API.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/config"
)

// identifierRegex validates SQL identifiers (catalog, schema, table names).
// Only allows alphanumeric characters and underscores to prevent SQL injection.
var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// QueryService provides operations for the Trino query layer.
type QueryService struct {
	cfg        config.TrinoConfig
	httpClient *http.Client
	logger     *slog.Logger
}

// NewQueryService creates a new QueryService.
func NewQueryService(cfg config.TrinoConfig, logger *slog.Logger) *QueryService {
	if logger == nil {
		logger = slog.Default()
	}

	return &QueryService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.QueryTimeout,
		},
		logger: logger.With("component", "query-service"),
	}
}

// validateIdentifier validates a SQL identifier to prevent SQL injection.
func validateIdentifier(name, identifierType string) error {
	if name == "" {
		return fmt.Errorf("%s name cannot be empty", identifierType)
	}
	if !identifierRegex.MatchString(name) {
		return fmt.Errorf("invalid %s name: must contain only alphanumeric characters and underscores, and start with a letter or underscore", identifierType)
	}
	return nil
}

// GetStatus returns the current status of the query layer.
func (s *QueryService) GetStatus(ctx context.Context) (*models.QueryLayerStatus, error) {
	status := &models.QueryLayerStatus{
		CoordinatorURL: s.cfg.URL,
		CheckedAt:      models.TimeNow(),
	}

	if !s.cfg.Enabled {
		status.Available = false
		status.Error = "Trino query layer is not enabled"
		return status, nil
	}

	// Get cluster info
	info, err := s.getClusterInfo(ctx)
	if err != nil {
		status.Available = false
		status.Error = fmt.Sprintf("Failed to connect to Trino: %v", err)
		return status, nil
	}

	status.Available = !info.Starting
	status.TrinoVersion = info.NodeVersion.Version
	status.Uptime = info.Uptime

	// Get cluster stats
	stats, err := s.getClusterStats(ctx)
	if err != nil {
		s.logger.Warn("failed to get cluster stats", "error", err)
	} else {
		status.RunningQueries = stats.RunningQueries
		status.QueuedQueries = stats.QueuedQueries
		status.BlockedQueries = stats.BlockedQueries
		status.ActiveWorkers = stats.ActiveWorkers
		status.NodeCount = stats.ActiveWorkers + 1 // workers + coordinator
	}

	return status, nil
}

// GetHealth returns health status of the query layer.
func (s *QueryService) GetHealth(ctx context.Context) (*models.QueryHealthResponse, error) {
	status, err := s.GetStatus(ctx)
	if err != nil {
		return &models.QueryHealthResponse{
			Status:  "unknown",
			Message: fmt.Sprintf("Failed to check health: %v", err),
		}, nil
	}

	if !status.Available {
		return &models.QueryHealthResponse{
			Status:  "unhealthy",
			Message: status.Error,
			Details: status,
		}, nil
	}

	return &models.QueryHealthResponse{
		Status:  "healthy",
		Message: "Trino query layer is operational",
		Details: status,
	}, nil
}

// ListCatalogs returns all available Trino catalogs.
func (s *QueryService) ListCatalogs(ctx context.Context) (*models.CatalogListResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("Trino query layer is not enabled")
	}

	// Query catalogs using Trino REST API
	rows, err := s.executeQuery(ctx, "SHOW CATALOGS")
	if err != nil {
		return nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	catalogs := make([]models.TrinoCatalog, 0, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			catalogs = append(catalogs, models.TrinoCatalog{
				Name: fmt.Sprintf("%v", row[0]),
			})
		}
	}

	return &models.CatalogListResponse{
		Catalogs: catalogs,
		Total:    len(catalogs),
	}, nil
}

// ListSchemas returns all schemas in a catalog.
func (s *QueryService) ListSchemas(ctx context.Context, catalog string) (*models.SchemaListResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("Trino query layer is not enabled")
	}

	// Validate catalog name to prevent SQL injection
	if err := validateIdentifier(catalog, "catalog"); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SHOW SCHEMAS FROM %s", catalog)
	rows, err := s.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	schemas := make([]models.TrinoSchema, 0, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			schemas = append(schemas, models.TrinoSchema{
				Name:    fmt.Sprintf("%v", row[0]),
				Catalog: catalog,
			})
		}
	}

	return &models.SchemaListResponse{
		Schemas: schemas,
		Catalog: catalog,
		Total:   len(schemas),
	}, nil
}

// ListTables returns all tables in a schema.
func (s *QueryService) ListTables(ctx context.Context, catalog, schema string) (*models.TableListResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("Trino query layer is not enabled")
	}

	// Validate identifiers to prevent SQL injection
	if err := validateIdentifier(catalog, "catalog"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(schema, "schema"); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SHOW TABLES FROM %s.%s", catalog, schema)
	rows, err := s.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	tables := make([]models.TrinoTable, 0, len(rows))
	for _, row := range rows {
		if len(row) > 0 {
			tables = append(tables, models.TrinoTable{
				Name:    fmt.Sprintf("%v", row[0]),
				Schema:  schema,
				Catalog: catalog,
				Type:    "TABLE",
			})
		}
	}

	return &models.TableListResponse{
		Tables:  tables,
		Catalog: catalog,
		Schema:  schema,
		Total:   len(tables),
	}, nil
}

// GetTableInfo returns detailed information about a table.
func (s *QueryService) GetTableInfo(ctx context.Context, catalog, schema, table string) (*models.TableInfoResponse, error) {
	if !s.cfg.Enabled {
		return nil, fmt.Errorf("Trino query layer is not enabled")
	}

	// Validate identifiers to prevent SQL injection
	if err := validateIdentifier(catalog, "catalog"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(schema, "schema"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(table, "table"); err != nil {
		return nil, err
	}

	query := fmt.Sprintf("DESCRIBE %s.%s.%s", catalog, schema, table)
	rows, err := s.executeQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}

	columns := make([]models.TrinoColumn, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 2 {
			col := models.TrinoColumn{
				Name: fmt.Sprintf("%v", row[0]),
				Type: fmt.Sprintf("%v", row[1]),
			}
			if len(row) > 2 && row[2] != nil {
				col.Comment = fmt.Sprintf("%v", row[2])
			}
			columns = append(columns, col)
		}
	}

	return &models.TableInfoResponse{
		Name:    table,
		Schema:  schema,
		Catalog: catalog,
		Type:    "TABLE",
		Columns: columns,
	}, nil
}

// getClusterInfo fetches Trino cluster info from /v1/info.
func (s *QueryService) getClusterInfo(ctx context.Context) (*models.TrinoClusterInfo, error) {
	url := strings.TrimSuffix(s.cfg.URL, "/") + "/v1/info"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	if s.cfg.Username != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("Trino returned status %d (failed to read body: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("Trino returned status %d: %s", resp.StatusCode, string(body))
	}

	var info models.TrinoClusterInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode cluster info: %w", err)
	}

	return &info, nil
}

// getClusterStats fetches Trino cluster stats from /v1/cluster.
func (s *QueryService) getClusterStats(ctx context.Context) (*models.TrinoClusterStats, error) {
	url := strings.TrimSuffix(s.cfg.URL, "/") + "/v1/cluster"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	if s.cfg.Username != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Trino cluster endpoint returned status %d", resp.StatusCode)
	}

	var stats models.TrinoClusterStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode cluster stats: %w", err)
	}

	return &stats, nil
}

// executeQuery executes a Trino SQL query and returns the results.
// This is a simplified implementation using the Trino REST API.
func (s *QueryService) executeQuery(ctx context.Context, query string) ([][]interface{}, error) {
	url := strings.TrimSuffix(s.cfg.URL, "/") + "/v1/statement"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Trino-User", s.cfg.Username)
	if s.cfg.Username == "" {
		req.Header.Set("X-Trino-User", "philotes")
	}
	req.Header.Set("X-Trino-Catalog", s.cfg.Catalog)
	req.Header.Set("X-Trino-Schema", s.cfg.Schema)

	if s.cfg.Username != "" && s.cfg.Password != "" {
		req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("query failed with status %d (failed to read body: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse initial response
	var result struct {
		ID      string          `json:"id"`
		NextURI string          `json:"nextUri"`
		Data    [][]interface{} `json:"data"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("query error: %s", result.Error.Message)
	}

	allData := result.Data

	// Follow nextUri to get all results
	for result.NextURI != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.NextURI, http.NoBody)
		if err != nil {
			return nil, err
		}

		if s.cfg.Username != "" && s.cfg.Password != "" {
			req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("query continuation failed with status %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode query continuation: %w", err)
		}
		resp.Body.Close()

		if result.Error != nil {
			return nil, fmt.Errorf("query error: %s", result.Error.Message)
		}

		if result.Data != nil {
			allData = append(allData, result.Data...)
		}
	}

	return allData, nil
}
