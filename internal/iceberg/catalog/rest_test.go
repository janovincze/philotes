package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/janovincze/philotes/internal/iceberg"
)

func TestNewRESTCatalog(t *testing.T) {
	cfg := Config{
		CatalogURL: "http://localhost:8181",
		Warehouse:  "test",
	}

	client := NewRESTCatalog(cfg, nil)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.config.CatalogURL != cfg.CatalogURL {
		t.Errorf("Expected CatalogURL %q, got %q", cfg.CatalogURL, client.config.CatalogURL)
	}

	if client.config.Warehouse != cfg.Warehouse {
		t.Errorf("Expected Warehouse %q, got %q", cfg.Warehouse, client.config.Warehouse)
	}
}

func TestNamespaceExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"exists", http.StatusOK, true},
		{"not_found", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := Config{
				CatalogURL: server.URL,
				Warehouse:  "test",
			}

			client := NewRESTCatalog(cfg, nil)
			ctx := context.Background()

			exists, err := client.NamespaceExists(ctx, "myns")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if exists != tt.expected {
				t.Errorf("Expected exists=%v, got %v", tt.expected, exists)
			}
		})
	}
}

func TestCreateNamespace(t *testing.T) {
	// Test creating a new namespace
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog/v1/test/namespaces/myns":
			// Namespace doesn't exist
			w.WriteHeader(http.StatusNotFound)
		case "/catalog/v1/test/namespaces":
			// Create namespace
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		CatalogURL: server.URL,
		Warehouse:  "test",
	}

	client := NewRESTCatalog(cfg, nil)
	ctx := context.Background()

	err := client.CreateNamespace(ctx, "myns", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestTableExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"exists", http.StatusOK, true},
		{"not_found", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			cfg := Config{
				CatalogURL: server.URL,
				Warehouse:  "test",
			}

			client := NewRESTCatalog(cfg, nil)
			ctx := context.Background()

			exists, err := client.TableExists(ctx, "myns", "mytable")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if exists != tt.expected {
				t.Errorf("Expected exists=%v, got %v", tt.expected, exists)
			}
		})
	}
}

func TestLoadTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := loadTableResponse{
			MetadataLocation: "s3://bucket/metadata.json",
		}
		response.Metadata.FormatVersion = 2
		response.Metadata.TableUUID = "test-uuid"
		response.Metadata.Location = "s3://bucket/table"
		response.Metadata.Schemas = []restSchema{
			{
				Type:     "struct",
				SchemaID: 0,
				Fields: []restField{
					{ID: 1, Name: "id", Type: "long", Required: true},
					{ID: 2, Name: "name", Type: "string", Required: false},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response) //nolint:errcheck
	}))
	defer server.Close()

	cfg := Config{
		CatalogURL: server.URL,
		Warehouse:  "test",
	}

	client := NewRESTCatalog(cfg, nil)
	ctx := context.Background()

	meta, err := client.LoadTable(ctx, "myns", "mytable")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if meta.FormatVersion != 2 {
		t.Errorf("Expected format version 2, got %d", meta.FormatVersion)
	}

	if meta.TableUUID != "test-uuid" {
		t.Errorf("Expected table UUID 'test-uuid', got %q", meta.TableUUID)
	}

	if len(meta.Schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(meta.Schemas))
	}

	if len(meta.Schemas[0].Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(meta.Schemas[0].Fields))
	}
}

func TestConvertSchemaToREST(t *testing.T) {
	schema := iceberg.Schema{
		SchemaID: 1,
		Fields: []iceberg.Field{
			{ID: 1, Name: "id", Type: iceberg.TypeLong, Required: true},
			{ID: 2, Name: "name", Type: iceberg.TypeString, Required: false, Doc: "User name"},
		},
	}

	rest := convertSchemaToREST(schema)

	if rest.Type != "struct" {
		t.Errorf("Expected type 'struct', got %q", rest.Type)
	}

	if rest.SchemaID != 1 {
		t.Errorf("Expected schema ID 1, got %d", rest.SchemaID)
	}

	if len(rest.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(rest.Fields))
	}

	if rest.Fields[0].Name != "id" {
		t.Errorf("Expected first field name 'id', got %q", rest.Fields[0].Name)
	}

	if rest.Fields[1].Doc != "User name" {
		t.Errorf("Expected doc 'User name', got %q", rest.Fields[1].Doc)
	}
}

func TestConvertPartitionSpecToREST(t *testing.T) {
	spec := iceberg.PartitionSpec{
		SpecID: 0,
		Fields: []iceberg.PartitionField{
			{SourceID: 1, FieldID: 1000, Name: "date", Transform: "day"},
		},
	}

	rest := convertPartitionSpecToREST(spec)

	if rest.SpecID != 0 {
		t.Errorf("Expected spec ID 0, got %d", rest.SpecID)
	}

	if len(rest.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(rest.Fields))
	}

	if rest.Fields[0].Transform != "day" {
		t.Errorf("Expected transform 'day', got %q", rest.Fields[0].Transform)
	}
}
