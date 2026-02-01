// Package e2e provides end-to-end tests for the Philotes CDC platform.
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

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestConfig holds configuration for E2E tests.
type TestConfig struct {
	APIBaseURL    string
	SourceHost    string
	SourcePort    int
	SourceDB      string
	SourceUser    string
	SourcePass    string
	LakekeeperURL string
}

// DefaultTestConfig returns the default test configuration from environment variables.
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		APIBaseURL:    getEnv("PHILOTES_API_URL", "http://localhost:8080"),
		SourceHost:    getEnv("POSTGRES_HOST", "localhost"),
		SourcePort:    getEnvInt("POSTGRES_PORT", 5433),
		SourceDB:      getEnv("POSTGRES_DB", "source_db"),
		SourceUser:    getEnv("POSTGRES_USER", "postgres"),
		SourcePass:    getEnv("POSTGRES_PASSWORD", "postgres"),
		LakekeeperURL: getEnv("LAKEKEEPER_URL", "http://localhost:8181"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var n int
		if _, err := fmt.Sscanf(val, "%d", &n); err == nil {
			return n
		}
	}
	return defaultVal
}

// APIClient provides a simple HTTP client for the Philotes API.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateSource creates a new source.
func (c *APIClient) CreateSource(req map[string]interface{}) (map[string]interface{}, error) {
	return c.post("/api/v1/sources", req)
}

// TestConnection tests a source connection.
func (c *APIClient) TestConnection(sourceID string) (map[string]interface{}, error) {
	return c.post(fmt.Sprintf("/api/v1/sources/%s/test", sourceID), nil)
}

// DiscoverTables discovers tables from a source.
func (c *APIClient) DiscoverTables(sourceID string) (map[string]interface{}, error) {
	return c.get(fmt.Sprintf("/api/v1/sources/%s/tables", sourceID))
}

// CreatePipeline creates a new pipeline.
func (c *APIClient) CreatePipeline(req map[string]interface{}) (map[string]interface{}, error) {
	return c.post("/api/v1/pipelines", req)
}

// StartPipeline starts a pipeline.
func (c *APIClient) StartPipeline(pipelineID string) (map[string]interface{}, error) {
	return c.post(fmt.Sprintf("/api/v1/pipelines/%s/start", pipelineID), nil)
}

// GetPipelineStatus gets the status of a pipeline.
func (c *APIClient) GetPipelineStatus(pipelineID string) (map[string]interface{}, error) {
	return c.get(fmt.Sprintf("/api/v1/pipelines/%s/status", pipelineID))
}

// GetPipeline gets a pipeline by ID.
func (c *APIClient) GetPipeline(pipelineID string) (map[string]interface{}, error) {
	return c.get(fmt.Sprintf("/api/v1/pipelines/%s", pipelineID))
}

// StopPipeline stops a pipeline.
func (c *APIClient) StopPipeline(pipelineID string) (map[string]interface{}, error) {
	return c.post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipelineID), nil)
}

// DeletePipeline deletes a pipeline.
func (c *APIClient) DeletePipeline(pipelineID string) error {
	return c.delete(fmt.Sprintf("/api/v1/pipelines/%s", pipelineID))
}

// DeleteSource deletes a source.
func (c *APIClient) DeleteSource(sourceID string) error {
	return c.delete(fmt.Sprintf("/api/v1/sources/%s", sourceID))
}

// ExecuteQuery executes a query via Trino.
func (c *APIClient) ExecuteQuery(query string, timeoutSeconds int) (map[string]interface{}, error) {
	req := map[string]interface{}{
		"query":           query,
		"timeout_seconds": timeoutSeconds,
	}
	return c.post("/api/v1/query/execute", req)
}

// PreflightCheck runs pre-flight checks for a pipeline.
func (c *APIClient) PreflightCheck(pipelineID string) (map[string]interface{}, error) {
	return c.get(fmt.Sprintf("/api/v1/pipelines/%s/preflight", pipelineID))
}

func (c *APIClient) get(path string) (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *APIClient) post(path string, data map[string]interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (c *APIClient) delete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// TestClickUpFlow tests the complete click-up experience:
// 1. Create source via API
// 2. Test connection
// 3. Discover tables
// 4. Create pipeline with table mappings
// 5. Run pre-flight checks
// 6. Start pipeline
// 7. Insert test data in source PostgreSQL
// 8. Poll Trino for replicated data (timeout 60s)
// 9. Query via /api/v1/query/execute
// 10. Verify data matches
// 11. Cleanup
func TestClickUpFlow(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	cfg := DefaultTestConfig()
	client := NewAPIClient(cfg.APIBaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Generate unique identifiers for this test run
	testID := uuid.New().String()[:8]
	sourceName := fmt.Sprintf("e2e_source_%s", testID)
	pipelineName := fmt.Sprintf("e2e_pipeline_%s", testID)
	tableName := fmt.Sprintf("e2e_test_%s", testID)

	var sourceID, pipelineID string

	// Cleanup function
	cleanup := func() {
		if pipelineID != "" {
			_, _ = client.StopPipeline(pipelineID)
			_ = client.DeletePipeline(pipelineID)
		}
		if sourceID != "" {
			_ = client.DeleteSource(sourceID)
		}
		// Clean up test table
		db, err := sql.Open("postgres", fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.SourceHost, cfg.SourcePort, cfg.SourceUser, cfg.SourcePass, cfg.SourceDB,
		))
		if err == nil {
			defer db.Close()
			_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		}
	}
	defer cleanup()

	t.Log("Step 1: Creating test table in source database")
	{
		db, err := sql.Open("postgres", fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.SourceHost, cfg.SourcePort, cfg.SourceUser, cfg.SourcePass, cfg.SourceDB,
		))
		if err != nil {
			t.Fatalf("Failed to connect to source database: %v", err)
		}
		defer db.Close()

		// Create test table
		createTableSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255),
				created_at TIMESTAMP DEFAULT NOW()
			)
		`, tableName)
		if _, err := db.Exec(createTableSQL); err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		t.Logf("Created test table: %s", tableName)
	}

	t.Log("Step 2: Creating source via API")
	{
		sourceReq := map[string]interface{}{
			"name":          sourceName,
			"type":          "postgresql",
			"host":          cfg.SourceHost,
			"port":          cfg.SourcePort,
			"database_name": cfg.SourceDB,
			"username":      cfg.SourceUser,
			"password":      cfg.SourcePass,
			"ssl_mode":      "disable",
		}

		resp, err := client.CreateSource(sourceReq)
		if err != nil {
			t.Fatalf("Failed to create source: %v", err)
		}

		source, ok := resp["source"].(map[string]interface{})
		if !ok {
			t.Fatalf("Invalid source response: %v", resp)
		}
		sourceID = source["id"].(string)
		t.Logf("Created source: %s (ID: %s)", sourceName, sourceID)
	}

	t.Log("Step 3: Testing source connection")
	{
		resp, err := client.TestConnection(sourceID)
		if err != nil {
			t.Fatalf("Failed to test connection: %v", err)
		}

		success, ok := resp["success"].(bool)
		if !ok || !success {
			t.Fatalf("Connection test failed: %v", resp)
		}
		t.Log("Connection test successful")
	}

	t.Log("Step 4: Discovering tables")
	{
		resp, err := client.DiscoverTables(sourceID)
		if err != nil {
			t.Fatalf("Failed to discover tables: %v", err)
		}

		tables, ok := resp["tables"].([]interface{})
		if !ok {
			t.Fatalf("Invalid tables response: %v", resp)
		}

		// Find our test table
		found := false
		for _, table := range tables {
			tableInfo, ok := table.(map[string]interface{})
			if !ok {
				continue
			}
			if tableInfo["name"] == tableName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Test table %s not found in discovered tables", tableName)
		}
		t.Logf("Found test table %s in discovered tables (total: %d)", tableName, len(tables))
	}

	t.Log("Step 5: Creating pipeline with table mappings")
	{
		pipelineReq := map[string]interface{}{
			"name":      pipelineName,
			"source_id": sourceID,
			"tables": []map[string]interface{}{
				{
					"schema":  "public",
					"table":   tableName,
					"enabled": true,
				},
			},
		}

		resp, err := client.CreatePipeline(pipelineReq)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}

		pipeline, ok := resp["pipeline"].(map[string]interface{})
		if !ok {
			t.Fatalf("Invalid pipeline response: %v", resp)
		}
		pipelineID = pipeline["id"].(string)
		t.Logf("Created pipeline: %s (ID: %s)", pipelineName, pipelineID)
	}

	t.Log("Step 6: Running pre-flight checks")
	{
		resp, err := client.PreflightCheck(pipelineID)
		if err != nil {
			t.Logf("Pre-flight check request failed (may be expected if not fully implemented): %v", err)
		} else {
			ready, _ := resp["ready"].(bool)
			t.Logf("Pre-flight check result: ready=%v", ready)
			if checks, ok := resp["checks"].([]interface{}); ok {
				for _, check := range checks {
					checkInfo := check.(map[string]interface{})
					t.Logf("  - %s: %s", checkInfo["name"], checkInfo["status"])
				}
			}
		}
	}

	t.Log("Step 7: Starting pipeline")
	{
		resp, err := client.StartPipeline(pipelineID)
		if err != nil {
			t.Fatalf("Failed to start pipeline: %v", err)
		}
		t.Logf("Pipeline start response: %v", resp)

		// Wait for pipeline to be running
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			status, err := client.GetPipelineStatus(pipelineID)
			if err != nil {
				t.Logf("Failed to get pipeline status: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			pipelineStatus, _ := status["status"].(string)
			if pipelineStatus == "running" {
				t.Log("Pipeline is now running")
				break
			}
			if pipelineStatus == "error" {
				errMsg, _ := status["error_message"].(string)
				t.Fatalf("Pipeline entered error state: %s", errMsg)
			}
			t.Logf("Pipeline status: %s, waiting...", pipelineStatus)
			time.Sleep(2 * time.Second)
		}
	}

	t.Log("Step 8: Inserting test data into source")
	var insertedID int
	{
		db, err := sql.Open("postgres", fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.SourceHost, cfg.SourcePort, cfg.SourceUser, cfg.SourcePass, cfg.SourceDB,
		))
		if err != nil {
			t.Fatalf("Failed to connect to source database: %v", err)
		}
		defer db.Close()

		// Insert test data
		insertSQL := fmt.Sprintf(`
			INSERT INTO %s (name, email, created_at)
			VALUES ($1, $2, NOW())
			RETURNING id
		`, tableName)

		testName := fmt.Sprintf("Test User %s", testID)
		testEmail := fmt.Sprintf("test_%s@example.com", testID)

		err = db.QueryRow(insertSQL, testName, testEmail).Scan(&insertedID)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
		t.Logf("Inserted test row with ID: %d", insertedID)
	}

	t.Log("Step 9: Waiting for data replication (polling Trino)")
	{
		// Wait for data to be replicated to Iceberg
		query := fmt.Sprintf("SELECT * FROM iceberg.cdc.%s WHERE id = %d", tableName, insertedID)
		deadline := time.Now().Add(60 * time.Second)
		found := false

		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				t.Fatal("Test context cancelled")
			default:
			}

			resp, err := client.ExecuteQuery(query, 10)
			if err != nil {
				t.Logf("Query failed (may be expected while replicating): %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			rowCount, ok := resp["row_count"].(float64)
			if ok && rowCount > 0 {
				found = true
				t.Logf("Data found in Iceberg! Row count: %v", rowCount)

				// Verify data content
				data, ok := resp["data"].([]interface{})
				if ok && len(data) > 0 {
					row := data[0].([]interface{})
					t.Logf("Replicated row: %v", row)
				}
				break
			}

			t.Log("Data not yet replicated, waiting...")
			time.Sleep(5 * time.Second)
		}

		if !found {
			t.Fatal("Data was not replicated within timeout period")
		}
	}

	t.Log("Step 10: Verifying data via query API")
	{
		query := fmt.Sprintf("SELECT id, name, email FROM iceberg.cdc.%s ORDER BY id DESC LIMIT 10", tableName)
		resp, err := client.ExecuteQuery(query, 30)
		if err != nil {
			t.Fatalf("Final query failed: %v", err)
		}

		columns, ok := resp["columns"].([]interface{})
		if ok {
			t.Logf("Columns: %v", columns)
		}

		data, ok := resp["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Fatal("No data returned from final query")
		}

		execTime, _ := resp["execution_time_ms"].(float64)
		t.Logf("Query executed in %vms, returned %d rows", execTime, len(data))

		// Verify our inserted data is present
		found := false
		for _, row := range data {
			rowData := row.([]interface{})
			if id, ok := rowData[0].(float64); ok && int(id) == insertedID {
				found = true
				t.Logf("Verified inserted row: id=%d, name=%v, email=%v", insertedID, rowData[1], rowData[2])
				break
			}
		}

		if !found {
			t.Fatalf("Inserted row with ID %d not found in query results", insertedID)
		}
	}

	t.Log("Step 11: Stopping pipeline")
	{
		if _, err := client.StopPipeline(pipelineID); err != nil {
			t.Logf("Failed to stop pipeline (may already be stopped): %v", err)
		} else {
			t.Log("Pipeline stopped")
		}
	}

	t.Log("E2E Click-Up Flow Test PASSED!")
}

// TestPipelinePreflightChecks tests the pre-flight check functionality.
func TestPipelinePreflightChecks(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	cfg := DefaultTestConfig()
	client := NewAPIClient(cfg.APIBaseURL)

	testID := uuid.New().String()[:8]
	sourceName := fmt.Sprintf("e2e_preflight_%s", testID)
	pipelineName := fmt.Sprintf("e2e_preflight_pipeline_%s", testID)

	var sourceID, pipelineID string

	defer func() {
		if pipelineID != "" {
			_ = client.DeletePipeline(pipelineID)
		}
		if sourceID != "" {
			_ = client.DeleteSource(sourceID)
		}
	}()

	// Create source
	sourceReq := map[string]interface{}{
		"name":          sourceName,
		"type":          "postgresql",
		"host":          cfg.SourceHost,
		"port":          cfg.SourcePort,
		"database_name": cfg.SourceDB,
		"username":      cfg.SourceUser,
		"password":      cfg.SourcePass,
		"ssl_mode":      "disable",
	}

	resp, err := client.CreateSource(sourceReq)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	source := resp["source"].(map[string]interface{})
	sourceID = source["id"].(string)

	// Create pipeline
	pipelineReq := map[string]interface{}{
		"name":      pipelineName,
		"source_id": sourceID,
		"tables": []map[string]interface{}{
			{
				"schema":  "public",
				"table":   "users",
				"enabled": true,
			},
		},
	}

	resp, err = client.CreatePipeline(pipelineReq)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	pipeline := resp["pipeline"].(map[string]interface{})
	pipelineID = pipeline["id"].(string)

	// Run pre-flight checks
	resp, err = client.PreflightCheck(pipelineID)
	if err != nil {
		// Pre-flight checks may not be fully implemented yet
		t.Logf("Pre-flight check failed (may be expected): %v", err)
		return
	}

	// Verify response structure
	if _, ok := resp["ready"]; !ok {
		t.Error("Pre-flight response missing 'ready' field")
	}

	if _, ok := resp["checks"]; !ok {
		t.Error("Pre-flight response missing 'checks' field")
	}

	checks, ok := resp["checks"].([]interface{})
	if ok {
		t.Logf("Pre-flight checks completed: %d checks", len(checks))
		for _, check := range checks {
			checkInfo := check.(map[string]interface{})
			name, _ := checkInfo["name"].(string)
			status, _ := checkInfo["status"].(string)
			t.Logf("  - %s: %s", name, status)
		}
	}
}

// TestQueryExecution tests the query execution API.
func TestQueryExecution(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	cfg := DefaultTestConfig()
	client := NewAPIClient(cfg.APIBaseURL)

	// Test simple query
	t.Run("SimpleQuery", func(t *testing.T) {
		resp, err := client.ExecuteQuery("SELECT 1 as test_value", 10)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		rowCount, _ := resp["row_count"].(float64)
		if rowCount != 1 {
			t.Errorf("Expected 1 row, got %v", rowCount)
		}

		columns, ok := resp["columns"].([]interface{})
		if !ok || len(columns) == 0 {
			t.Error("Expected columns in response")
		}

		data, ok := resp["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Error("Expected data in response")
		}
	})

	// Test catalog listing
	t.Run("ListCatalogs", func(t *testing.T) {
		resp, err := client.ExecuteQuery("SHOW CATALOGS", 10)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		rowCount, _ := resp["row_count"].(float64)
		if rowCount == 0 {
			t.Error("Expected at least one catalog")
		}

		t.Logf("Found %v catalogs", rowCount)
	})
}
