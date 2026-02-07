// Package models provides API request and response types.
package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

// catalogNameRegex validates Trino catalog names.
var catalogNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// QueryDataSourceStatus represents the status of a query data source.
type QueryDataSourceStatus string

const (
	// QueryDataSourceStatusInactive indicates the data source is registered but not active.
	QueryDataSourceStatusInactive QueryDataSourceStatus = "inactive"
	// QueryDataSourceStatusActive indicates the data source is active and available for queries.
	QueryDataSourceStatusActive QueryDataSourceStatus = "active"
	// QueryDataSourceStatusError indicates the data source has an error.
	QueryDataSourceStatusError QueryDataSourceStatus = "error"
)

// QueryDataSource represents an external data source for query federation.
type QueryDataSource struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     *uuid.UUID             `json:"tenant_id,omitempty"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	CatalogName  string                 `json:"catalog_name"`
	Host         string                 `json:"host"`
	Port         int                    `json:"port"`
	DatabaseName string                 `json:"database_name"`
	Username     string                 `json:"username"`
	SSLMode      string                 `json:"ssl_mode"`
	ExtraConfig  map[string]interface{} `json:"extra_config,omitempty"`
	Status       QueryDataSourceStatus  `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CreateQueryDataSourceRequest represents a request to create a new query data source.
type CreateQueryDataSourceRequest struct {
	Name         string                 `json:"name" binding:"required,min=1,max=255"`
	Type         string                 `json:"type" binding:"required"`
	CatalogName  string                 `json:"catalog_name" binding:"required"`
	Host         string                 `json:"host" binding:"required"`
	Port         int                    `json:"port,omitempty"`
	DatabaseName string                 `json:"database_name" binding:"required"`
	Username     string                 `json:"username" binding:"required"`
	Password     string                 `json:"password" binding:"required"`
	SSLMode      string                 `json:"ssl_mode,omitempty"`
	ExtraConfig  map[string]interface{} `json:"extra_config,omitempty"`
}

// Validate validates the create request.
func (r *CreateQueryDataSourceRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if r.Type == "" {
		errors = append(errors, FieldError{Field: "type", Message: "type is required"})
	} else if r.Type != "postgresql" && r.Type != "mysql" {
		errors = append(errors, FieldError{Field: "type", Message: "type must be 'postgresql' or 'mysql'"})
	}
	if r.CatalogName == "" {
		errors = append(errors, FieldError{Field: "catalog_name", Message: "catalog_name is required"})
	} else if !catalogNameRegex.MatchString(r.CatalogName) {
		errors = append(errors, FieldError{Field: "catalog_name", Message: "catalog_name must contain only alphanumeric characters and underscores, starting with a letter or underscore"})
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

// ApplyDefaults applies default values.
func (r *CreateQueryDataSourceRequest) ApplyDefaults() {
	if r.Port == 0 {
		switch r.Type {
		case "mysql":
			r.Port = 3306
		default:
			r.Port = 5432
		}
	}
	if r.SSLMode == "" {
		r.SSLMode = "prefer"
	}
}

// UpdateQueryDataSourceRequest represents a request to update a query data source.
type UpdateQueryDataSourceRequest struct {
	Name         *string                `json:"name,omitempty"`
	Host         *string                `json:"host,omitempty"`
	Port         *int                   `json:"port,omitempty"`
	DatabaseName *string                `json:"database_name,omitempty"`
	Username     *string                `json:"username,omitempty"`
	Password     *string                `json:"password,omitempty"`
	SSLMode      *string                `json:"ssl_mode,omitempty"`
	ExtraConfig  map[string]interface{} `json:"extra_config,omitempty"`
}

// Validate validates the update request.
func (r *UpdateQueryDataSourceRequest) Validate() []FieldError {
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

// HasConnectionChanges returns true if any connection-related fields are being updated.
func (r *UpdateQueryDataSourceRequest) HasConnectionChanges() bool {
	return r.Host != nil || r.Port != nil || r.DatabaseName != nil ||
		r.Username != nil || r.Password != nil || r.SSLMode != nil
}

// QueryDataSourceResponse wraps a single data source for API responses.
type QueryDataSourceResponse struct {
	DataSource *QueryDataSource `json:"data_source"`
}

// QueryDataSourceListResponse wraps a list of data sources for API responses.
type QueryDataSourceListResponse struct {
	DataSources []QueryDataSource `json:"data_sources"`
	TotalCount  int               `json:"total_count"`
}

// QueryDataSourceTestResult represents the result of a connection test.
type QueryDataSourceTestResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	ErrorDetail string `json:"error_detail,omitempty"`
}
