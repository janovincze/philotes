//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestSourceCRUD tests the full CRUD lifecycle of sources
func TestSourceCRUD(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Generate unique name to avoid conflicts
	sourceName := fmt.Sprintf("e2e-test-source-%d", time.Now().UnixNano())

	// Create source
	t.Run("Create", func(t *testing.T) {
		req := CreateSourceRequest{
			Name:            sourceName,
			Host:            sourceDBHost,
			Port:            5433,
			DatabaseName:    sourceDBName,
			Username:        sourceDBUser,
			Password:        sourceDBPassword,
			SSLMode:         "disable",
			SlotName:        fmt.Sprintf("e2e_slot_%d", time.Now().UnixNano()),
			PublicationName: "philotes_pub",
		}

		body, status, err := client.Post("/api/v1/sources", req)
		if err != nil {
			t.Fatalf("Failed to create source: %v", err)
		}
		RequireStatusCode(t, http.StatusCreated, status, body)

		var source SourceResponse
		if err := ParseJSON(body, &source); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if source.ID == "" {
			t.Fatal("Expected source ID to be set")
		}
		if source.Name != sourceName {
			t.Errorf("Expected name %s, got %s", sourceName, source.Name)
		}

		// Store for later tests
		t.Logf("Created source with ID: %s", source.ID)

		// Test Get
		t.Run("Get", func(t *testing.T) {
			body, status, err := client.Get("/api/v1/sources/" + source.ID)
			if err != nil {
				t.Fatalf("Failed to get source: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var retrieved SourceResponse
			if err := ParseJSON(body, &retrieved); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if retrieved.ID != source.ID {
				t.Errorf("Expected ID %s, got %s", source.ID, retrieved.ID)
			}
		})

		// Test List
		t.Run("List", func(t *testing.T) {
			body, status, err := client.Get("/api/v1/sources")
			if err != nil {
				t.Fatalf("Failed to list sources: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var sources []SourceResponse
			if err := ParseJSON(body, &sources); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Should have at least our created source
			found := false
			for _, s := range sources {
				if s.ID == source.ID {
					found = true
					break
				}
			}
			if !found {
				t.Error("Created source not found in list")
			}
		})

		// Test Update
		t.Run("Update", func(t *testing.T) {
			updateReq := map[string]interface{}{
				"name": sourceName + "-updated",
			}

			body, status, err := client.Put("/api/v1/sources/"+source.ID, updateReq)
			if err != nil {
				t.Fatalf("Failed to update source: %v", err)
			}
			RequireStatusCode(t, http.StatusOK, status, body)

			var updated SourceResponse
			if err := ParseJSON(body, &updated); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if updated.Name != sourceName+"-updated" {
				t.Errorf("Expected name %s, got %s", sourceName+"-updated", updated.Name)
			}
		})

		// Test Delete
		t.Run("Delete", func(t *testing.T) {
			_, status, err := client.Delete("/api/v1/sources/" + source.ID)
			if err != nil {
				t.Fatalf("Failed to delete source: %v", err)
			}
			RequireStatusCode(t, http.StatusNoContent, status, nil)

			// Verify deletion
			_, status, _ = client.Get("/api/v1/sources/" + source.ID)
			if status != http.StatusNotFound {
				t.Errorf("Expected 404 after deletion, got %d", status)
			}
		})
	})
}

// TestSourceConnection tests the connection testing functionality
func TestSourceConnection(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create a source for testing
	sourceName := fmt.Sprintf("e2e-connection-test-%d", time.Now().UnixNano())
	req := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_conn_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", req)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var source SourceResponse
	if err := ParseJSON(body, &source); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Cleanup
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	t.Run("TestConnection", func(t *testing.T) {
		body, status, err := client.Post("/api/v1/sources/"+source.ID+"/test", nil)
		if err != nil {
			t.Fatalf("Failed to test connection: %v", err)
		}
		RequireStatusCode(t, http.StatusOK, status, body)

		var result TestConnectionResponse
		if err := ParseJSON(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !result.Success {
			t.Errorf("Expected connection test to succeed, got: %s", result.Message)
		}
	})
}

// TestSourceDiscoverTables tests the table discovery functionality
func TestSourceDiscoverTables(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create a source for testing
	sourceName := fmt.Sprintf("e2e-discover-test-%d", time.Now().UnixNano())
	req := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        sourceDBUser,
		Password:        sourceDBPassword,
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_disc_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", req)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var source SourceResponse
	if err := ParseJSON(body, &source); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Cleanup
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	t.Run("DiscoverTables", func(t *testing.T) {
		body, status, err := client.Get("/api/v1/sources/" + source.ID + "/tables")
		if err != nil {
			t.Fatalf("Failed to discover tables: %v", err)
		}
		RequireStatusCode(t, http.StatusOK, status, body)

		var result DiscoverTablesResponse
		if err := ParseJSON(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should find the e-commerce tables
		expectedTables := map[string]bool{
			"customers":   false,
			"products":    false,
			"orders":      false,
			"order_items": false,
		}

		for _, table := range result.Tables {
			if table.Schema == "public" {
				if _, ok := expectedTables[table.Name]; ok {
					expectedTables[table.Name] = true
				}
			}
		}

		for table, found := range expectedTables {
			if !found {
				t.Errorf("Expected to find table %s in discovery results", table)
			}
		}
	})
}

// TestSourceInvalidCredentials tests error handling for invalid credentials
func TestSourceInvalidCredentials(t *testing.T) {
	client := NewAPIClient(apiBaseURL)

	// Create source with invalid credentials
	sourceName := fmt.Sprintf("e2e-invalid-creds-%d", time.Now().UnixNano())
	req := CreateSourceRequest{
		Name:            sourceName,
		Host:            sourceDBHost,
		Port:            5433,
		DatabaseName:    sourceDBName,
		Username:        "invalid_user",
		Password:        "invalid_password",
		SSLMode:         "disable",
		SlotName:        fmt.Sprintf("e2e_inv_%d", time.Now().UnixNano()),
		PublicationName: "philotes_pub",
	}

	body, status, err := client.Post("/api/v1/sources", req)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	RequireStatusCode(t, http.StatusCreated, status, body)

	var source SourceResponse
	if err := ParseJSON(body, &source); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Cleanup
	defer func() {
		client.Delete("/api/v1/sources/" + source.ID)
	}()

	t.Run("TestConnectionFails", func(t *testing.T) {
		body, status, err := client.Post("/api/v1/sources/"+source.ID+"/test", nil)
		if err != nil {
			t.Fatalf("Failed to test connection: %v", err)
		}

		// Should either return a non-2xx status or success=false
		if status == http.StatusOK {
			var result TestConnectionResponse
			if err := ParseJSON(body, &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}
			if result.Success {
				t.Error("Expected connection test to fail with invalid credentials")
			}
		}
		// Status 4xx or 5xx is also acceptable for failed connection
	})
}
