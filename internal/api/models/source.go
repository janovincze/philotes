// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"
)

// SourceStatus represents the status of a source.
type SourceStatus string

const (
	// SourceStatusInactive indicates the source is registered but not active.
	SourceStatusInactive SourceStatus = "inactive"
	// SourceStatusActive indicates the source is active and can be used.
	SourceStatusActive SourceStatus = "active"
	// SourceStatusError indicates the source has an error.
	SourceStatusError SourceStatus = "error"
)

// Source represents a CDC source database in the system.
type Source struct {
	ID              uuid.UUID    `json:"id"`
	TenantID        *uuid.UUID   `json:"tenant_id,omitempty"`
	Name            string       `json:"name"`
	Type            string       `json:"type"`
	Host            string       `json:"host"`
	Port            int          `json:"port"`
	DatabaseName    string       `json:"database_name"`
	Username        string       `json:"username"`
	SSLMode         string       `json:"ssl_mode"`
	SlotName        string       `json:"slot_name,omitempty"`
	PublicationName string       `json:"publication_name,omitempty"`
	Status          SourceStatus `json:"status"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// CreateSourceRequest represents a request to create a new source.
type CreateSourceRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=255"`
	Type            string `json:"type,omitempty"`
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port,omitempty"`
	DatabaseName    string `json:"database_name" binding:"required"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	SSLMode         string `json:"ssl_mode,omitempty"`
	SlotName        string `json:"slot_name,omitempty"`
	PublicationName string `json:"publication_name,omitempty"`
}

// Validate validates the create source request.
func (r *CreateSourceRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if r.Host == "" {
		errors = append(errors, FieldError{Field: "host", Message: "host is required"})
	}
	if r.DatabaseName == "" {
		errors = append(errors, FieldError{Field: "database_name", Message: "database_name is required"})
	}
	if r.Username == "" {
		errors = append(errors, FieldError{Field: "username", Message: "username is required"})
	}
	if r.Password == "" {
		errors = append(errors, FieldError{Field: "password", Message: "password is required"})
	}
	if r.Port != 0 && (r.Port < 1 || r.Port > 65535) {
		errors = append(errors, FieldError{Field: "port", Message: "port must be between 1 and 65535"})
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateSourceRequest) ApplyDefaults() {
	if r.Type == "" {
		r.Type = "postgresql"
	}
	if r.Port == 0 {
		r.Port = 5432
	}
	if r.SSLMode == "" {
		r.SSLMode = "prefer"
	}
}

// UpdateSourceRequest represents a request to update a source.
type UpdateSourceRequest struct {
	Name            *string `json:"name,omitempty"`
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	DatabaseName    *string `json:"database_name,omitempty"`
	Username        *string `json:"username,omitempty"`
	Password        *string `json:"password,omitempty"`
	SSLMode         *string `json:"ssl_mode,omitempty"`
	SlotName        *string `json:"slot_name,omitempty"`
	PublicationName *string `json:"publication_name,omitempty"`
}

// Validate validates the update source request.
func (r *UpdateSourceRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}
	if r.Host != nil && *r.Host == "" {
		errors = append(errors, FieldError{Field: "host", Message: "host cannot be empty"})
	}
	if r.Port != nil && (*r.Port < 1 || *r.Port > 65535) {
		errors = append(errors, FieldError{Field: "port", Message: "port must be between 1 and 65535"})
	}

	return errors
}

// SourceResponse wraps a source for API responses.
type SourceResponse struct {
	Source *Source `json:"source"`
}

// SourceListResponse wraps a list of sources for API responses.
type SourceListResponse struct {
	Sources    []Source `json:"sources"`
	TotalCount int      `json:"total_count"`
}

// ConnectionTestResult represents the result of a connection test.
type ConnectionTestResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	ServerInfo  string `json:"server_info,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

// TableInfo represents information about a table in a source database.
type TableInfo struct {
	Schema  string       `json:"schema"`
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns,omitempty"`
}

// ColumnInfo represents information about a column in a table.
type ColumnInfo struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Nullable   bool    `json:"nullable"`
	PrimaryKey bool    `json:"primary_key"`
	Default    *string `json:"default,omitempty"`
}

// TableDiscoveryResponse wraps table discovery results.
type TableDiscoveryResponse struct {
	Tables []TableInfo `json:"tables"`
	Count  int         `json:"count"`
}
