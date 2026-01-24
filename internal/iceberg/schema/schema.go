package schema

import (
	"sort"

	"github.com/janovincze/philotes/internal/cdc"
	"github.com/janovincze/philotes/internal/iceberg"
)

// Builder builds Iceberg schemas from CDC events.
type Builder struct {
	// NextFieldID is the next available field ID.
	NextFieldID int
}

// NewBuilder creates a new schema builder.
func NewBuilder() *Builder {
	return &Builder{
		NextFieldID: 1,
	}
}

// BuildFromEvents builds an Iceberg schema from a set of CDC events.
// It analyzes the event data to determine column names and types.
func (b *Builder) BuildFromEvents(events []cdc.Event) iceberg.Schema {
	// Collect all column names and their inferred types
	columns := make(map[string]iceberg.Type)

	for _, event := range events {
		// Analyze after data (for INSERT/UPDATE)
		for name, value := range event.After {
			existingType, exists := columns[name]
			inferredType := InferTypeFromValue(value)

			if !exists {
				columns[name] = inferredType
			} else if existingType != inferredType {
				// Type conflict - use string as fallback
				columns[name] = iceberg.TypeString
			}
		}

		// Also check before data (for UPDATE/DELETE)
		for name, value := range event.Before {
			if _, exists := columns[name]; !exists {
				columns[name] = InferTypeFromValue(value)
			}
		}
	}

	return b.buildSchema(columns)
}

// BuildFromData builds an Iceberg schema from a single data map.
func (b *Builder) BuildFromData(data map[string]any) iceberg.Schema {
	columns := make(map[string]iceberg.Type)
	for name, value := range data {
		columns[name] = InferTypeFromValue(value)
	}
	return b.buildSchema(columns)
}

// buildSchema creates a schema from column definitions.
func (b *Builder) buildSchema(columns map[string]iceberg.Type) iceberg.Schema {
	// Get sorted column names for consistent ordering
	names := make([]string, 0, len(columns))
	for name := range columns {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build fields
	fields := make([]iceberg.Field, 0, len(columns)+len(iceberg.CDCSystemColumns))

	// Add user columns
	for _, name := range names {
		fields = append(fields, iceberg.Field{
			ID:       b.NextFieldID,
			Name:     name,
			Type:     columns[name],
			Required: false, // CDC columns are generally nullable
		})
		b.NextFieldID++
	}

	// Add CDC system columns
	for _, sysCol := range iceberg.CDCSystemColumns {
		fields = append(fields, iceberg.Field{
			ID:       b.NextFieldID,
			Name:     sysCol.Name,
			Type:     sysCol.Type,
			Required: sysCol.Required,
			Doc:      sysCol.Doc,
		})
		b.NextFieldID++
	}

	return iceberg.Schema{
		SchemaID: 0,
		Fields:   fields,
	}
}

// DefaultPartitionSpec returns a default partition specification.
// For CDC tables, we typically partition by the CDC timestamp date.
func DefaultPartitionSpec(schema iceberg.Schema) iceberg.PartitionSpec {
	// Find the _cdc_timestamp field
	var timestampFieldID int
	for _, field := range schema.Fields {
		if field.Name == "_cdc_timestamp" {
			timestampFieldID = field.ID
			break
		}
	}

	if timestampFieldID == 0 {
		// No timestamp field, return unpartitioned spec
		return iceberg.PartitionSpec{
			SpecID: 0,
			Fields: nil,
		}
	}

	return iceberg.PartitionSpec{
		SpecID: 0,
		Fields: []iceberg.PartitionField{
			{
				SourceID:  timestampFieldID,
				FieldID:   1000, // Partition field IDs start at 1000
				Name:      "_cdc_date",
				Transform: "day",
			},
		},
	}
}

// GetFieldByName returns a field by name from a schema.
func GetFieldByName(schema iceberg.Schema, name string) *iceberg.Field {
	for i := range schema.Fields {
		if schema.Fields[i].Name == name {
			return &schema.Fields[i]
		}
	}
	return nil
}

// MergeSchemas merges two schemas, adding any new fields from the second to the first.
// This is useful for schema evolution.
func MergeSchemas(existing, new iceberg.Schema, nextFieldID int) (iceberg.Schema, int) {
	// Create a map of existing fields
	existingFields := make(map[string]iceberg.Field)
	for _, field := range existing.Fields {
		existingFields[field.Name] = field
	}

	// Start with existing fields
	merged := iceberg.Schema{
		SchemaID: existing.SchemaID + 1,
		Fields:   make([]iceberg.Field, len(existing.Fields)),
	}
	copy(merged.Fields, existing.Fields)

	// Add new fields that don't exist
	for _, field := range new.Fields {
		if _, exists := existingFields[field.Name]; !exists {
			newField := field
			newField.ID = nextFieldID
			nextFieldID++
			merged.Fields = append(merged.Fields, newField)
		}
	}

	return merged, nextFieldID
}
