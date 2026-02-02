//go:build e2e

// Package e2e provides end-to-end tests for the Philotes CDC platform.
// These tests require the Docker Compose environment to be running.
//
// Run with: go test -tags=e2e -v ./tests/e2e/...
package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// Test configuration - can be overridden via environment variables
var (
	apiBaseURL       = getEnv("PHILOTES_API_URL", "http://localhost:8080")
	sourceDBHost     = getEnv("PHILOTES_SOURCE_HOST", "localhost")
	sourceDBPort     = getEnv("PHILOTES_SOURCE_PORT", "5433")
	sourceDBUser     = getEnv("PHILOTES_SOURCE_USER", "source")
	sourceDBPassword = getEnv("PHILOTES_SOURCE_PASSWORD", "source")
	sourceDBName     = getEnv("PHILOTES_SOURCE_DB", "source")
	trinoHost        = getEnv("PHILOTES_TRINO_HOST", "localhost")
	trinoPort        = getEnv("PHILOTES_TRINO_PORT", "8085")
)

// Timeouts for async operations
const (
	defaultTimeout     = 30 * time.Second
	pipelineStartWait  = 60 * time.Second
	cdcPropagationWait = 90 * time.Second
	pollInterval       = 2 * time.Second
)

// APIClient provides methods to interact with the Philotes API
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Request sends an HTTP request and returns the response body
func (c *APIClient) Request(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// Get performs a GET request
func (c *APIClient) Get(path string) ([]byte, int, error) {
	return c.Request(http.MethodGet, path, nil)
}

// Post performs a POST request with JSON body
func (c *APIClient) Post(path string, body interface{}) ([]byte, int, error) {
	return c.Request(http.MethodPost, path, body)
}

// Put performs a PUT request with JSON body
func (c *APIClient) Put(path string, body interface{}) ([]byte, int, error) {
	return c.Request(http.MethodPut, path, body)
}

// Delete performs a DELETE request
func (c *APIClient) Delete(path string) ([]byte, int, error) {
	return c.Request(http.MethodDelete, path, nil)
}

// SourceDB provides methods to interact with the source PostgreSQL database
type SourceDB struct {
	db *sql.DB
}

// NewSourceDB creates a new source database connection
func NewSourceDB() (*SourceDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		sourceDBHost, sourceDBPort, sourceDBUser, sourceDBPassword, sourceDBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SourceDB{db: db}, nil
}

// Close closes the database connection
func (s *SourceDB) Close() error {
	return s.db.Close()
}

// Exec executes a SQL statement
func (s *SourceDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// Query executes a SQL query and returns rows
func (s *SourceDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// QueryRow executes a SQL query and returns a single row
func (s *SourceDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// PollUntil repeatedly calls the check function until it returns true or timeout
func PollUntil(ctx context.Context, interval time.Duration, check func() (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			done, err := check()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

// WaitForPipelineStatus polls until the pipeline reaches the expected status
func WaitForPipelineStatus(client *APIClient, pipelineID, expectedStatus string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return PollUntil(ctx, pollInterval, func() (bool, error) {
		body, status, err := client.Get(fmt.Sprintf("/api/v1/pipelines/%s/status", pipelineID))
		if err != nil {
			return false, err
		}
		if status != http.StatusOK {
			return false, nil // Keep polling
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return false, err
		}

		currentStatus, ok := result["status"].(string)
		if !ok {
			return false, nil
		}

		return currentStatus == expectedStatus, nil
	})
}

// API Request/Response structures

// CreateSourceRequest represents a request to create a source
type CreateSourceRequest struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	DatabaseName    string `json:"database_name"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	SSLMode         string `json:"ssl_mode"`
	SlotName        string `json:"slot_name"`
	PublicationName string `json:"publication_name"`
}

// Source represents a source in API responses
type Source struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	DatabaseName    string    `json:"database_name"`
	Status          string    `json:"status"`
	SlotName        string    `json:"slot_name"`
	PublicationName string    `json:"publication_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SourceResponse wraps a source for API responses
type SourceResponse struct {
	Source Source `json:"source"`
}

// SourceListResponse wraps a list of sources for API responses
type SourceListResponse struct {
	Sources    []Source `json:"sources"`
	TotalCount int      `json:"total_count"`
}

// CreatePipelineRequest represents a request to create a pipeline
type CreatePipelineRequest struct {
	Name            string                 `json:"name"`
	SourceID        string                 `json:"source_id"`
	DestinationType string                 `json:"destination_type"`
	Config          map[string]interface{} `json:"config,omitempty"`
}

// Pipeline represents a pipeline in API responses
type Pipeline struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	SourceID        string                 `json:"source_id"`
	DestinationType string                 `json:"destination_type"`
	Status          string                 `json:"status"`
	Config          map[string]interface{} `json:"config"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PipelineResponse wraps a pipeline for API responses
type PipelineResponse struct {
	Pipeline Pipeline `json:"pipeline"`
}

// PipelineListResponse wraps a list of pipelines for API responses
type PipelineListResponse struct {
	Pipelines  []Pipeline `json:"pipelines"`
	TotalCount int        `json:"total_count"`
}

// PipelineStatusResponse represents pipeline status
type PipelineStatusResponse struct {
	Status          string     `json:"status"`
	LSNPosition     string     `json:"lsn_position,omitempty"`
	EventsProcessed int64      `json:"events_processed"`
	LastEventAt     *time.Time `json:"last_event_at,omitempty"`
}

// AddTableMappingRequest represents a request to add a table mapping
type AddTableMappingRequest struct {
	Schema  string         `json:"schema,omitempty"`
	Table   string         `json:"table"`
	Enabled *bool          `json:"enabled,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

// TableMapping represents a table mapping in API responses
type TableMapping struct {
	ID           string         `json:"id"`
	PipelineID   string         `json:"pipeline_id"`
	SourceSchema string         `json:"source_schema"`
	SourceTable  string         `json:"source_table"`
	Enabled      bool           `json:"enabled"`
	Config       map[string]any `json:"config,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// TestConnectionResponse represents the response from testing a source connection
type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DiscoverTablesResponse represents the response from discovering tables
type DiscoverTablesResponse struct {
	Tables []TableInfo `json:"tables"`
}

// TableInfo represents information about a table
type TableInfo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ParseJSON parses JSON response body into the target struct
func ParseJSON(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}

// RequireStatusCode fails the test if the status code doesn't match
func RequireStatusCode(t *testing.T, expected, actual int, body []byte) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected status %d, got %d. Body: %s", expected, actual, string(body))
	}
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Verify API is reachable
	client := NewAPIClient(apiBaseURL)
	_, status, err := client.Get("/health")
	if err != nil || status != http.StatusOK {
		fmt.Printf("API not reachable at %s. Ensure Docker Compose is running and API server is started.\n", apiBaseURL)
		fmt.Printf("Run: docker compose -f deployments/docker/docker-compose.yml up -d\n")
		fmt.Printf("Run: go run cmd/philotes-api/main.go\n")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}
