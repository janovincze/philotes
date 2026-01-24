// Package iceberg provides Apache Iceberg table management for CDC events.
package iceberg

import (
	"time"
)

// Type represents an Iceberg data type.
type Type string

// Iceberg primitive types.
const (
	TypeBoolean   Type = "boolean"
	TypeInt       Type = "int"
	TypeLong      Type = "long"
	TypeFloat     Type = "float"
	TypeDouble    Type = "double"
	TypeDate      Type = "date"
	TypeTime      Type = "time"
	TypeTimestamp Type = "timestamp"
	TypeString    Type = "string"
	TypeUUID      Type = "uuid"
	TypeBinary    Type = "binary"
)

// Field represents a field in an Iceberg schema.
type Field struct {
	// ID is the unique field identifier.
	ID int `json:"id"`

	// Name is the field name.
	Name string `json:"name"`

	// Type is the field data type.
	Type Type `json:"type"`

	// Required indicates if the field is required (not nullable).
	Required bool `json:"required"`

	// Doc is an optional documentation string.
	Doc string `json:"doc,omitempty"`
}

// Schema represents an Iceberg table schema.
type Schema struct {
	// SchemaID is the schema identifier.
	SchemaID int `json:"schema-id"`

	// Fields is the list of fields in the schema.
	Fields []Field `json:"fields"`
}

// PartitionField represents a partition field specification.
type PartitionField struct {
	// SourceID is the ID of the source field.
	SourceID int `json:"source-id"`

	// FieldID is the partition field ID.
	FieldID int `json:"field-id"`

	// Name is the partition field name.
	Name string `json:"name"`

	// Transform is the partition transform (identity, year, month, day, hour, etc.).
	Transform string `json:"transform"`
}

// PartitionSpec represents an Iceberg partition specification.
type PartitionSpec struct {
	// SpecID is the partition spec identifier.
	SpecID int `json:"spec-id"`

	// Fields is the list of partition fields.
	Fields []PartitionField `json:"fields"`
}

// DataFile represents metadata about a data file in Iceberg.
type DataFile struct {
	// FilePath is the path to the data file.
	FilePath string `json:"file-path"`

	// FileFormat is the file format (parquet, avro, orc).
	FileFormat string `json:"file-format"`

	// RecordCount is the number of records in the file.
	RecordCount int64 `json:"record-count"`

	// FileSizeInBytes is the file size.
	FileSizeInBytes int64 `json:"file-size-in-bytes"`

	// PartitionData contains the partition values for this file.
	PartitionData map[string]any `json:"partition,omitempty"`
}

// Snapshot represents an Iceberg table snapshot.
type Snapshot struct {
	// SnapshotID is the unique snapshot identifier.
	SnapshotID int64 `json:"snapshot-id"`

	// ParentSnapshotID is the parent snapshot ID (0 for first snapshot).
	ParentSnapshotID int64 `json:"parent-snapshot-id,omitempty"`

	// TimestampMs is the snapshot creation timestamp.
	TimestampMs int64 `json:"timestamp-ms"`

	// ManifestList is the path to the manifest list file.
	ManifestList string `json:"manifest-list"`

	// Summary contains snapshot summary metadata.
	Summary map[string]string `json:"summary,omitempty"`
}

// TableMetadata represents Iceberg table metadata.
type TableMetadata struct {
	// FormatVersion is the Iceberg format version (1 or 2).
	FormatVersion int `json:"format-version"`

	// TableUUID is the table's unique identifier.
	TableUUID string `json:"table-uuid"`

	// Location is the table's base location in storage.
	Location string `json:"location"`

	// LastUpdatedMs is the last update timestamp.
	LastUpdatedMs int64 `json:"last-updated-ms"`

	// LastColumnID is the highest assigned column ID.
	LastColumnID int `json:"last-column-id"`

	// Schemas is the list of schemas.
	Schemas []Schema `json:"schemas"`

	// CurrentSchemaID is the current schema ID.
	CurrentSchemaID int `json:"current-schema-id"`

	// PartitionSpecs is the list of partition specifications.
	PartitionSpecs []PartitionSpec `json:"partition-specs"`

	// DefaultSpecID is the default partition spec ID.
	DefaultSpecID int `json:"default-spec-id"`

	// LastPartitionID is the highest assigned partition field ID.
	LastPartitionID int `json:"last-partition-id"`

	// Properties contains table properties.
	Properties map[string]string `json:"properties,omitempty"`

	// CurrentSnapshotID is the current snapshot ID.
	CurrentSnapshotID int64 `json:"current-snapshot-id,omitempty"`

	// Snapshots is the list of snapshots.
	Snapshots []Snapshot `json:"snapshots,omitempty"`
}

// Namespace represents an Iceberg namespace (database).
type Namespace struct {
	// Name is the namespace name.
	Name string `json:"name"`

	// Properties contains namespace properties.
	Properties map[string]string `json:"properties,omitempty"`
}

// TableIdentifier identifies a table within a namespace.
type TableIdentifier struct {
	// Namespace is the namespace name.
	Namespace string

	// Name is the table name.
	Name string
}

// String returns the fully qualified table name.
func (t TableIdentifier) String() string {
	return t.Namespace + "." + t.Name
}

// CDCSystemColumns defines the CDC system columns added to every table.
var CDCSystemColumns = []Field{
	{
		ID:       -1, // Will be assigned during schema creation
		Name:     "_cdc_operation",
		Type:     TypeString,
		Required: true,
		Doc:      "CDC operation type (INSERT, UPDATE, DELETE)",
	},
	{
		ID:       -2,
		Name:     "_cdc_timestamp",
		Type:     TypeTimestamp,
		Required: true,
		Doc:      "Timestamp when the CDC event occurred",
	},
	{
		ID:       -3,
		Name:     "_cdc_lsn",
		Type:     TypeString,
		Required: true,
		Doc:      "PostgreSQL Log Sequence Number",
	},
}

// NewTableMetadata creates a new TableMetadata with defaults.
func NewTableMetadata(location string, schema Schema, partitionSpec PartitionSpec) *TableMetadata {
	now := time.Now().UnixMilli()
	return &TableMetadata{
		FormatVersion:   2,
		Location:        location,
		LastUpdatedMs:   now,
		LastColumnID:    len(schema.Fields),
		Schemas:         []Schema{schema},
		CurrentSchemaID: schema.SchemaID,
		PartitionSpecs:  []PartitionSpec{partitionSpec},
		DefaultSpecID:   partitionSpec.SpecID,
		Properties:      make(map[string]string),
	}
}
