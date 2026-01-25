// Package models provides API request and response types.
package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

// PipelineStatus represents the status of a pipeline.
type PipelineStatus string

const (
	// PipelineStatusStopped indicates the pipeline is not running.
	PipelineStatusStopped PipelineStatus = "stopped"
	// PipelineStatusStarting indicates the pipeline is starting.
	PipelineStatusStarting PipelineStatus = "starting"
	// PipelineStatusRunning indicates the pipeline is running.
	PipelineStatusRunning PipelineStatus = "running"
	// PipelineStatusStopping indicates the pipeline is stopping.
	PipelineStatusStopping PipelineStatus = "stopping"
	// PipelineStatusError indicates the pipeline has an error.
	PipelineStatusError PipelineStatus = "error"
)

// Pipeline represents a CDC pipeline in the system.
type Pipeline struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	SourceID     uuid.UUID         `json:"source_id"`
	Status       PipelineStatus    `json:"status"`
	Config       map[string]any    `json:"config,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Tables       []TableMapping    `json:"tables,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	StoppedAt    *time.Time        `json:"stopped_at,omitempty"`
}

// TableMapping represents a table configuration for a pipeline.
type TableMapping struct {
	ID           uuid.UUID      `json:"id"`
	PipelineID   uuid.UUID      `json:"pipeline_id"`
	SourceSchema string         `json:"source_schema"`
	SourceTable  string         `json:"source_table"`
	Enabled      bool           `json:"enabled"`
	Config       map[string]any `json:"config,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// CreatePipelineRequest represents a request to create a new pipeline.
type CreatePipelineRequest struct {
	Name     string                      `json:"name" binding:"required,min=1,max=255"`
	SourceID uuid.UUID                   `json:"source_id" binding:"required"`
	Tables   []CreateTableMappingRequest `json:"tables,omitempty"`
	Config   map[string]any              `json:"config,omitempty"`
}

// CreateTableMappingRequest represents a table mapping in a create request.
type CreateTableMappingRequest struct {
	Schema  string         `json:"schema,omitempty"`
	Table   string         `json:"table" binding:"required"`
	Enabled *bool          `json:"enabled,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

// Validate validates the create pipeline request.
func (r *CreatePipelineRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if r.SourceID == uuid.Nil {
		errors = append(errors, FieldError{Field: "source_id", Message: "source_id is required"})
	}

	for i, table := range r.Tables {
		if table.Table == "" {
			errors = append(errors, FieldError{
				Field:   "tables[" + strconv.Itoa(i) + "].table",
				Message: "table name is required",
			})
		}
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreatePipelineRequest) ApplyDefaults() {
	for i := range r.Tables {
		if r.Tables[i].Schema == "" {
			r.Tables[i].Schema = "public"
		}
		if r.Tables[i].Enabled == nil {
			enabled := true
			r.Tables[i].Enabled = &enabled
		}
	}
}

// UpdatePipelineRequest represents a request to update a pipeline.
type UpdatePipelineRequest struct {
	Name   *string        `json:"name,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

// Validate validates the update pipeline request.
func (r *UpdatePipelineRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}

	return errors
}

// PipelineResponse wraps a pipeline for API responses.
type PipelineResponse struct {
	Pipeline *Pipeline `json:"pipeline"`
}

// PipelineListResponse wraps a list of pipelines for API responses.
type PipelineListResponse struct {
	Pipelines  []Pipeline `json:"pipelines"`
	TotalCount int        `json:"total_count"`
}

// PipelineStatusResponse represents detailed pipeline status.
type PipelineStatusResponse struct {
	ID              uuid.UUID      `json:"id"`
	Name            string         `json:"name"`
	Status          PipelineStatus `json:"status"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	EventsProcessed int64          `json:"events_processed"`
	LastEventAt     *time.Time     `json:"last_event_at,omitempty"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	Uptime          string         `json:"uptime,omitempty"`
}

// AddTableMappingRequest represents a request to add a table mapping to a pipeline.
type AddTableMappingRequest struct {
	Schema  string         `json:"schema,omitempty"`
	Table   string         `json:"table" binding:"required"`
	Enabled *bool          `json:"enabled,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

// Validate validates the add table mapping request.
func (r *AddTableMappingRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Table == "" {
		errors = append(errors, FieldError{Field: "table", Message: "table is required"})
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *AddTableMappingRequest) ApplyDefaults() {
	if r.Schema == "" {
		r.Schema = "public"
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
}
