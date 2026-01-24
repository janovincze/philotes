// Package cdc provides Change Data Capture functionality for PostgreSQL databases.
package cdc

import (
	"time"
)

// Operation represents the type of database operation captured by CDC.
type Operation string

const (
	// OperationInsert represents an INSERT operation.
	OperationInsert Operation = "INSERT"
	// OperationUpdate represents an UPDATE operation.
	OperationUpdate Operation = "UPDATE"
	// OperationDelete represents a DELETE operation.
	OperationDelete Operation = "DELETE"
	// OperationTruncate represents a TRUNCATE operation.
	OperationTruncate Operation = "TRUNCATE"
)

// Event represents a single CDC event captured from the source database.
type Event struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`

	// LSN is the Log Sequence Number from PostgreSQL WAL.
	LSN string `json:"lsn"`

	// TransactionID is the PostgreSQL transaction ID.
	TransactionID int64 `json:"transaction_id"`

	// Timestamp is when the event occurred in the source database.
	Timestamp time.Time `json:"timestamp"`

	// Schema is the database schema name (e.g., "public").
	Schema string `json:"schema"`

	// Table is the table name.
	Table string `json:"table"`

	// Operation is the type of operation (INSERT, UPDATE, DELETE, TRUNCATE).
	Operation Operation `json:"operation"`

	// Before contains the row data before the operation (for UPDATE and DELETE).
	Before map[string]any `json:"before,omitempty"`

	// After contains the row data after the operation (for INSERT and UPDATE).
	After map[string]any `json:"after,omitempty"`

	// KeyColumns contains the names of the primary key columns.
	KeyColumns []string `json:"key_columns,omitempty"`

	// Metadata contains additional event metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Column represents a column in a database table.
type Column struct {
	// Name is the column name.
	Name string `json:"name"`

	// Type is the PostgreSQL data type.
	Type string `json:"type"`

	// Nullable indicates if the column allows NULL values.
	Nullable bool `json:"nullable"`

	// PrimaryKey indicates if this column is part of the primary key.
	PrimaryKey bool `json:"primary_key"`

	// DefaultValue is the default value expression, if any.
	DefaultValue *string `json:"default_value,omitempty"`
}

// TableSchema represents the schema of a database table.
type TableSchema struct {
	// Schema is the database schema name.
	Schema string `json:"schema"`

	// Table is the table name.
	Table string `json:"table"`

	// Columns is the list of columns in the table.
	Columns []Column `json:"columns"`

	// Version is the schema version number.
	Version int `json:"version"`

	// CapturedAt is when this schema was captured.
	CapturedAt time.Time `json:"captured_at"`

	// LSN is the LSN when this schema was captured.
	LSN string `json:"lsn"`
}

// Checkpoint represents a CDC checkpoint for recovery.
type Checkpoint struct {
	// SourceID identifies the source being checkpointed.
	SourceID string `json:"source_id"`

	// LSN is the last processed Log Sequence Number.
	LSN string `json:"lsn"`

	// TransactionID is the last processed transaction ID.
	TransactionID int64 `json:"transaction_id,omitempty"`

	// CommittedAt is when this checkpoint was committed.
	CommittedAt time.Time `json:"committed_at"`

	// Metadata contains additional checkpoint metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// FullyQualifiedTable returns the fully qualified table name (schema.table).
func (e *Event) FullyQualifiedTable() string {
	return e.Schema + "." + e.Table
}

// HasBefore returns true if the event has before data.
func (e *Event) HasBefore() bool {
	return len(e.Before) > 0
}

// HasAfter returns true if the event has after data.
func (e *Event) HasAfter() bool {
	return len(e.After) > 0
}

// FullyQualifiedName returns the fully qualified table name (schema.table).
func (t *TableSchema) FullyQualifiedName() string {
	return t.Schema + "." + t.Table
}

// PrimaryKeyColumns returns the columns that are part of the primary key.
func (t *TableSchema) PrimaryKeyColumns() []Column {
	var pkColumns []Column
	for _, col := range t.Columns {
		if col.PrimaryKey {
			pkColumns = append(pkColumns, col)
		}
	}
	return pkColumns
}
