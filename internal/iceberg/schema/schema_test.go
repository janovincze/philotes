package schema

import (
	"testing"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
	"github.com/janovincze/philotes/internal/iceberg"
)

func TestMapPostgresToIceberg(t *testing.T) {
	tests := []struct {
		pgType   string
		expected iceberg.Type
	}{
		// Integer types
		{"integer", iceberg.TypeInt},
		{"int4", iceberg.TypeInt},
		{"bigint", iceberg.TypeLong},
		{"int8", iceberg.TypeLong},
		{"smallint", iceberg.TypeInt},

		// Floating point
		{"real", iceberg.TypeFloat},
		{"float4", iceberg.TypeFloat},
		{"double precision", iceberg.TypeDouble},
		{"float8", iceberg.TypeDouble},
		{"numeric", iceberg.TypeDouble},

		// Boolean
		{"boolean", iceberg.TypeBoolean},
		{"bool", iceberg.TypeBoolean},

		// String types
		{"text", iceberg.TypeString},
		{"varchar", iceberg.TypeString},
		{"varchar(255)", iceberg.TypeString},
		{"char(10)", iceberg.TypeString},

		// Date/time
		{"date", iceberg.TypeDate},
		{"timestamp", iceberg.TypeTimestamp},
		{"timestamptz", iceberg.TypeTimestamp},
		{"timestamp with time zone", iceberg.TypeTimestamp},

		// Binary
		{"bytea", iceberg.TypeBinary},

		// UUID
		{"uuid", iceberg.TypeUUID},

		// JSON
		{"json", iceberg.TypeString},
		{"jsonb", iceberg.TypeString},

		// Arrays (stored as JSON string)
		{"integer[]", iceberg.TypeString},
		{"text[]", iceberg.TypeString},

		// Unknown types default to string
		{"unknown_type", iceberg.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			result := MapPostgresToIceberg(tt.pgType)
			if result != tt.expected {
				t.Errorf("MapPostgresToIceberg(%q) = %q, want %q", tt.pgType, result, tt.expected)
			}
		})
	}
}

func TestInferTypeFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected iceberg.Type
	}{
		{"nil", nil, iceberg.TypeString},
		{"bool", true, iceberg.TypeBoolean},
		{"int", 42, iceberg.TypeInt},
		{"int32", int32(42), iceberg.TypeInt},
		{"int64", int64(42), iceberg.TypeLong},
		{"float32", float32(3.14), iceberg.TypeFloat},
		{"float64", 3.14, iceberg.TypeDouble},
		{"string", "hello", iceberg.TypeString},
		{"bytes", []byte{1, 2, 3}, iceberg.TypeBinary},
		{"map", map[string]any{}, iceberg.TypeString},
		{"slice", []string{}, iceberg.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InferTypeFromValue(tt.value)
			if result != tt.expected {
				t.Errorf("InferTypeFromValue(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestBuilderBuildFromEvents(t *testing.T) {
	builder := NewBuilder()

	events := []cdc.Event{
		{
			Schema:    "public",
			Table:     "users",
			Operation: cdc.OperationInsert,
			Timestamp: time.Now(),
			After: map[string]any{
				"id":    int64(1),
				"name":  "Alice",
				"email": "alice@example.com",
			},
		},
		{
			Schema:    "public",
			Table:     "users",
			Operation: cdc.OperationInsert,
			Timestamp: time.Now(),
			After: map[string]any{
				"id":    int64(2),
				"name":  "Bob",
				"email": "bob@example.com",
				"age":   int64(30), // New column
			},
		},
	}

	schema := builder.BuildFromEvents(events)

	// Should have user columns + CDC system columns
	// User columns: age, email, id, name (sorted alphabetically)
	// CDC columns: _cdc_operation, _cdc_timestamp, _cdc_lsn
	expectedColumns := 7

	if len(schema.Fields) != expectedColumns {
		t.Errorf("Expected %d fields, got %d", expectedColumns, len(schema.Fields))
	}

	// Verify user columns are present
	userColumns := []string{"age", "email", "id", "name"}
	for _, colName := range userColumns {
		found := false
		for _, field := range schema.Fields {
			if field.Name == colName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected column %q not found in schema", colName)
		}
	}

	// Verify CDC system columns are present
	cdcColumns := []string{"_cdc_operation", "_cdc_timestamp", "_cdc_lsn"}
	for _, colName := range cdcColumns {
		found := false
		for _, field := range schema.Fields {
			if field.Name == colName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CDC column %q not found in schema", colName)
		}
	}
}

func TestDefaultPartitionSpec(t *testing.T) {
	schema := iceberg.Schema{
		SchemaID: 0,
		Fields: []iceberg.Field{
			{ID: 1, Name: "id", Type: iceberg.TypeLong},
			{ID: 2, Name: "_cdc_timestamp", Type: iceberg.TypeTimestamp},
		},
	}

	spec := DefaultPartitionSpec(schema)

	if spec.SpecID != 0 {
		t.Errorf("Expected spec ID 0, got %d", spec.SpecID)
	}

	if len(spec.Fields) != 1 {
		t.Errorf("Expected 1 partition field, got %d", len(spec.Fields))
	}

	if spec.Fields[0].SourceID != 2 {
		t.Errorf("Expected source ID 2, got %d", spec.Fields[0].SourceID)
	}

	if spec.Fields[0].Transform != "day" {
		t.Errorf("Expected transform 'day', got %q", spec.Fields[0].Transform)
	}
}

func TestGetFieldByName(t *testing.T) {
	schema := iceberg.Schema{
		Fields: []iceberg.Field{
			{ID: 1, Name: "id", Type: iceberg.TypeLong},
			{ID: 2, Name: "name", Type: iceberg.TypeString},
		},
	}

	// Found case
	field := GetFieldByName(schema, "name")
	if field == nil {
		t.Error("Expected to find field 'name'")
	} else if field.ID != 2 {
		t.Errorf("Expected field ID 2, got %d", field.ID)
	}

	// Not found case
	field = GetFieldByName(schema, "nonexistent")
	if field != nil {
		t.Error("Expected nil for nonexistent field")
	}
}
