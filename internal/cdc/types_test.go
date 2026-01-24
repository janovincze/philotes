package cdc

import (
	"testing"
	"time"
)

func TestOperation_String(t *testing.T) {
	tests := []struct {
		op   Operation
		want string
	}{
		{OperationInsert, "INSERT"},
		{OperationUpdate, "UPDATE"},
		{OperationDelete, "DELETE"},
		{OperationTruncate, "TRUNCATE"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := string(tt.op); got != tt.want {
				t.Errorf("Operation = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvent_FullyQualifiedTable(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		table  string
		want   string
	}{
		{"public schema", "public", "users", "public.users"},
		{"custom schema", "myschema", "orders", "myschema.orders"},
		{"empty schema", "", "test", ".test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Schema: tt.schema, Table: tt.table}
			if got := e.FullyQualifiedTable(); got != tt.want {
				t.Errorf("FullyQualifiedTable() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvent_HasBefore(t *testing.T) {
	tests := []struct {
		name   string
		before map[string]any
		want   bool
	}{
		{"nil before", nil, false},
		{"empty before", map[string]any{}, false},
		{"with before", map[string]any{"id": 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{Before: tt.before}
			if got := e.HasBefore(); got != tt.want {
				t.Errorf("HasBefore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_HasAfter(t *testing.T) {
	tests := []struct {
		name  string
		after map[string]any
		want  bool
	}{
		{"nil after", nil, false},
		{"empty after", map[string]any{}, false},
		{"with after", map[string]any{"id": 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{After: tt.after}
			if got := e.HasAfter(); got != tt.want {
				t.Errorf("HasAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableSchema_FullyQualifiedName(t *testing.T) {
	ts := TableSchema{Schema: "public", Table: "users"}
	want := "public.users"
	if got := ts.FullyQualifiedName(); got != want {
		t.Errorf("FullyQualifiedName() = %q, want %q", got, want)
	}
}

func TestTableSchema_PrimaryKeyColumns(t *testing.T) {
	ts := TableSchema{
		Columns: []Column{
			{Name: "id", PrimaryKey: true},
			{Name: "name", PrimaryKey: false},
			{Name: "tenant_id", PrimaryKey: true},
			{Name: "email", PrimaryKey: false},
		},
	}

	pkCols := ts.PrimaryKeyColumns()
	if len(pkCols) != 2 {
		t.Errorf("PrimaryKeyColumns() returned %d columns, want 2", len(pkCols))
	}

	names := make(map[string]bool)
	for _, col := range pkCols {
		names[col.Name] = true
	}

	if !names["id"] || !names["tenant_id"] {
		t.Errorf("PrimaryKeyColumns() = %v, want id and tenant_id", pkCols)
	}
}

func TestCheckpoint(t *testing.T) {
	now := time.Now()
	cp := Checkpoint{
		SourceID:      "test-source",
		LSN:           "0/1234567",
		TransactionID: 12345,
		CommittedAt:   now,
		Metadata: map[string]any{
			"version": "1.0",
		},
	}

	if cp.SourceID != "test-source" {
		t.Errorf("SourceID = %q, want %q", cp.SourceID, "test-source")
	}
	if cp.LSN != "0/1234567" {
		t.Errorf("LSN = %q, want %q", cp.LSN, "0/1234567")
	}
	if cp.TransactionID != 12345 {
		t.Errorf("TransactionID = %d, want %d", cp.TransactionID, 12345)
	}
	if !cp.CommittedAt.Equal(now) {
		t.Errorf("CommittedAt = %v, want %v", cp.CommittedAt, now)
	}
	if cp.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v, want %q", cp.Metadata["version"], "1.0")
	}
}
