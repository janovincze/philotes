//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestPipelineCRUD tests the full CRUD lifecycle of pipelines
func TestPipelineCRUD(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// First, create a source for the pipeline
	sourceName := fmt.Sprintf("e2e-pipeline-source-%d", time.Now().UnixNano())
	sourceReq := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_pipe_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", sourceReq)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var sourceResp SourceResponse
	if err := ParseJSON(body, &sourceResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	source := sourceResp.Source

	// Cleanup source at the end
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	// Generate unique pipeline name
	pipelineName := fmt.Sprintf("e2e-test-pipeline-%d", time.Now().UnixNano())

	// Create pipeline
	t.Run("Create", func(t *testing.T) {
		req := CreatePipelineRequest{
			Name:     pipelineName,
			SourceID: source.ID,
			Config: map[string]interface{}{
				"batch_size":             1000,
				"flush_interval_seconds": 10,
			},
		}

		body, status, err := client.Post("/api/v1/pipelines", req)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}
		RequireStatusCode(t, http.StatusCreated, status, body)

		var pipelineResp PipelineResponse
		if err := ParseJSON(body, &pipelineResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		pipeline := pipelineResp.Pipeline

		if pipeline.ID == "" {
			t.Fatal("Expected pipeline ID to be set")
		}
		if pipeline.Name != pipelineName {
			t.Errorf("Expected name %s, got %s", pipelineName, pipeline.Name)
		}
		if pipeline.SourceID != source.ID {
			t.Errorf("Expected source_id %s, got %s", source.ID, pipeline.SourceID)
		}

		t.Logf("Created pipeline with ID: %s", pipeline.ID)

		// Test Get
		t.Run("Get", func(t *testing.T) {
			body, status, err := client.Get("/api/v1/pipelines/" + pipeline.ID)
			if err != nil {
				t.Fatalf("Failed to get pipeline: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var retrieved PipelineResponse
			if err := ParseJSON(body, &retrieved); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if retrieved.Pipeline.ID != pipeline.ID {
				t.Errorf("Expected ID %s, got %s", pipeline.ID, retrieved.Pipeline.ID)
			}
		})

		// Test List
		t.Run("List", func(t *testing.T) {
			body, status, err := client.Get("/api/v1/pipelines")
			if err != nil {
				t.Fatalf("Failed to list pipelines: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var listResp PipelineListResponse
			if err := ParseJSON(body, &listResp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			found := false
			for _, p := range listResp.Pipelines {
				if p.ID == pipeline.ID {
					found = true
					break
				}
			}
			if !found {
				t.Error("Created pipeline not found in list")
			}
		})

		// Test Update
		t.Run("Update", func(t *testing.T) {
			updateReq := map[string]interface{}{
				"name": pipelineName + "-updated",
			}

			body, status, err := client.Put("/api/v1/pipelines/"+pipeline.ID, updateReq)
			if err != nil {
				t.Fatalf("Failed to update pipeline: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var updated PipelineResponse
			if err := ParseJSON(body, &updated); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if updated.Pipeline.Name != pipelineName+"-updated" {
				t.Errorf("Expected name %s, got %s", pipelineName+"-updated", updated.Pipeline.Name)
			}
		})

		// Test Delete
		t.Run("Delete", func(t *testing.T) {
			_, status, err := client.Delete("/api/v1/pipelines/" + pipeline.ID)
			if err != nil {
				t.Fatalf("Failed to delete pipeline: %v", err)
			}
			RequireStatusCode(t, http.StatusNoContent, status, nil)

			// Verify deletion
			_, status, _ = client.Get("/api/v1/pipelines/" + pipeline.ID)
			if status != http.StatusNotFound {
				t.Errorf("Expected 404 after deletion, got %d", status)
			}
		})
	})
}

// TestPipelineTableMappings tests adding and removing table mappings
func TestPipelineTableMappings(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create source
	sourceName := fmt.Sprintf("e2e-mapping-source-%d", time.Now().UnixNano())
	sourceReq := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_map_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", sourceReq)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var sourceResp SourceResponse
	if err := ParseJSON(body, &sourceResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	source := sourceResp.Source
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	// Create pipeline
	pipelineName := fmt.Sprintf("e2e-mapping-pipeline-%d", time.Now().UnixNano())
	pipelineReq := CreatePipelineRequest{
		Name:     pipelineName,
		SourceID: source.ID,
	}

	body, status, err = client.Post("/api/v1/pipelines", pipelineReq)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var pipelineResp PipelineResponse
	if err := ParseJSON(body, &pipelineResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	pipeline := pipelineResp.Pipeline
	defer func() {
		client.Delete("/api/v1/pipelines/" + pipeline.ID)
	}()

	t.Run("AddTableMapping", func(t *testing.T) {
		enabled := true
		mappingReq := AddTableMappingRequest{
			Schema:  "public",
			Table:   "customers",
			Enabled: &enabled,
		}

		body, status, err := client.Post(
			fmt.Sprintf("/api/v1/pipelines/%s/tables", pipeline.ID),
			mappingReq,
		)
		if err != nil {
			t.Fatalf("Failed to add table mapping: %v", err)
		}
		RequireStatusCode(t, http.StatusCreated, status, body)

		var mapping TableMapping
		if err := ParseJSON(body, &mapping); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if mapping.ID == "" {
			t.Fatal("Expected mapping ID to be set")
		}
		if mapping.SourceTable != "customers" {
			t.Errorf("Expected source_table customers, got %s", mapping.SourceTable)
		}

		t.Run("RemoveTableMapping", func(t *testing.T) {
			_, status, err := client.Delete(
				fmt.Sprintf("/api/v1/pipelines/%s/tables/%s", pipeline.ID, mapping.ID),
			)
			if err != nil {
				t.Fatalf("Failed to remove table mapping: %v", err)
			}
			RequireStatusCode(t, http.StatusNoContent, status, nil)
		})
	})
}

// TestPipelineStatus tests the pipeline status endpoint
func TestPipelineStatus(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create source
	sourceName := fmt.Sprintf("e2e-status-source-%d", time.Now().UnixNano())
	sourceReq := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_stat_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", sourceReq)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var sourceResp SourceResponse
	if err := ParseJSON(body, &sourceResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	source := sourceResp.Source
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	// Create pipeline
	pipelineName := fmt.Sprintf("e2e-status-pipeline-%d", time.Now().UnixNano())
	pipelineReq := CreatePipelineRequest{
		Name:     pipelineName,
		SourceID: source.ID,
	}

	body, status, err = client.Post("/api/v1/pipelines", pipelineReq)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var pipelineResp PipelineResponse
	if err := ParseJSON(body, &pipelineResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	pipeline := pipelineResp.Pipeline
	defer func() {
		// Stop pipeline if running before deleting
		client.Post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipeline.ID), nil)
		client.Delete("/api/v1/pipelines/" + pipeline.ID)
	}()

	t.Run("GetStatus", func(t *testing.T) {
		body, status, err := client.Get(fmt.Sprintf("/api/v1/pipelines/%s/status", pipeline.ID))
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		RequireStatusCode(t, http.StatusOK, status, body)

		var statusResp PipelineStatusResponse
		if err := ParseJSON(body, &statusResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// New pipeline should be in stopped or pending state
		validStatuses := map[string]bool{"stopped": true, "pending": true, "created": true}
		if !validStatuses[statusResp.Status] {
			t.Errorf("Expected status to be stopped/pending/created, got %s", statusResp.Status)
		}
	})
}

// TestPipelineStartStop tests starting and stopping a pipeline
// Note: This test requires the CDC worker to be running for full functionality
func TestPipelineStartStop(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create source
	sourceName := fmt.Sprintf("e2e-startstop-source-%d", time.Now().UnixNano())
	sourceReq := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_ss_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", sourceReq)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var sourceResp SourceResponse
	if err := ParseJSON(body, &sourceResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	source := sourceResp.Source
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	// Create pipeline with table mapping
	pipelineName := fmt.Sprintf("e2e-startstop-pipeline-%d", time.Now().UnixNano())
	pipelineReq := CreatePipelineRequest{
		Name:     pipelineName,
		SourceID: source.ID,
	}

	body, status, err = client.Post("/api/v1/pipelines", pipelineReq)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var pipelineResp PipelineResponse
	if err := ParseJSON(body, &pipelineResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	pipeline := pipelineResp.Pipeline
	defer func() {
		client.Post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipeline.ID), nil)
		client.Delete("/api/v1/pipelines/" + pipeline.ID)
	}()

	// Add a table mapping
	enabled := true
	mappingReq := AddTableMappingRequest{
		Schema:  "public",
		Table:   "customers",
		Enabled: &enabled,
	}

	_, status, err = client.Post(
		fmt.Sprintf("/api/v1/pipelines/%s/tables", pipeline.ID),
		mappingReq,
	)
	if err != nil {
		t.Fatalf("Failed to add table mapping: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	t.Run("Start", func(t *testing.T) {
		body, status, err := client.Post(fmt.Sprintf("/api/v1/pipelines/%s/start", pipeline.ID), nil)
		if err != nil {
			t.Fatalf("Failed to start pipeline: %v", err)
		}

		// Accept 200 (success) or 202 (accepted/async)
		if status != http.StatusOK && status != http.StatusAccepted {
			t.Fatalf("Expected status 200 or 202, got %d. Body: %s", status, string(body))
		}

		t.Logf("Pipeline start response: %s", string(body))
	})

	t.Run("Stop", func(t *testing.T) {
		// Give the pipeline a moment to start
		time.Sleep(2 * time.Second)

		body, status, err := client.Post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipeline.ID), nil)
		if err != nil {
			t.Fatalf("Failed to stop pipeline: %v", err)
		}

		// Accept 200 (success) or 202 (accepted/async)
		if status != http.StatusOK && status != http.StatusAccepted {
			t.Fatalf("Expected status 200 or 202, got %d. Body: %s", status, string(body))
		}

		t.Logf("Pipeline stop response: %s", string(body))
	})
}
