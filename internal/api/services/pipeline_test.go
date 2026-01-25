package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

func TestPipelineService_Create_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreatePipelineRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.CreatePipelineRequest{
				Name:     "test-pipeline",
				SourceID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &models.CreatePipelineRequest{
				SourceID: uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "missing source_id",
			req: &models.CreatePipelineRequest{
				Name: "test-pipeline",
			},
			wantErr: true,
		},
		{
			name: "with tables",
			req: &models.CreatePipelineRequest{
				Name:     "test-pipeline",
				SourceID: uuid.New(),
				Tables: []models.CreateTableMappingRequest{
					{Schema: "public", Table: "users"},
					{Schema: "public", Table: "orders"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
		})
	}
}

func TestPipelineService_Create_ApplyDefaults(t *testing.T) {
	req := &models.CreatePipelineRequest{
		Name:     "test-pipeline",
		SourceID: uuid.New(),
		Tables: []models.CreateTableMappingRequest{
			{Table: "users"},
			{Schema: "custom", Table: "orders"},
		},
	}

	req.ApplyDefaults()

	// First table should have default schema
	if req.Tables[0].Schema != "public" {
		t.Errorf("expected schema 'public', got '%s'", req.Tables[0].Schema)
	}

	// First table should have default enabled
	if req.Tables[0].Enabled == nil || !*req.Tables[0].Enabled {
		t.Error("expected enabled to be true")
	}

	// Second table should keep its schema
	if req.Tables[1].Schema != "custom" {
		t.Errorf("expected schema 'custom', got '%s'", req.Tables[1].Schema)
	}
}

func TestUpdatePipelineRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.UpdatePipelineRequest
		wantErr bool
	}{
		{
			name:    "empty request is valid",
			req:     &models.UpdatePipelineRequest{},
			wantErr: false,
		},
		{
			name: "valid name update",
			req: &models.UpdatePipelineRequest{
				Name: stringPtr("new-name"),
			},
			wantErr: false,
		},
		{
			name: "empty name not allowed",
			req: &models.UpdatePipelineRequest{
				Name: stringPtr(""),
			},
			wantErr: true,
		},
		{
			name: "config update",
			req: &models.UpdatePipelineRequest{
				Config: map[string]any{"key": "value"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
		})
	}
}

func TestAddTableMappingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.AddTableMappingRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.AddTableMappingRequest{
				Table: "users",
			},
			wantErr: false,
		},
		{
			name: "with schema",
			req: &models.AddTableMappingRequest{
				Schema: "custom",
				Table:  "users",
			},
			wantErr: false,
		},
		{
			name: "missing table",
			req: &models.AddTableMappingRequest{
				Schema: "public",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.req.Validate()
			if tt.wantErr && len(errors) == 0 {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("unexpected validation error: %v", errors)
			}
		})
	}
}

func TestAddTableMappingRequest_ApplyDefaults(t *testing.T) {
	req := &models.AddTableMappingRequest{
		Table: "users",
	}

	req.ApplyDefaults()

	if req.Schema != "public" {
		t.Errorf("expected schema 'public', got '%s'", req.Schema)
	}

	if req.Enabled == nil || !*req.Enabled {
		t.Error("expected enabled to be true")
	}
}

func TestPipelineStatus(t *testing.T) {
	tests := []struct {
		status models.PipelineStatus
		want   string
	}{
		{models.PipelineStatusStopped, "stopped"},
		{models.PipelineStatusStarting, "starting"},
		{models.PipelineStatusRunning, "running"},
		{models.PipelineStatusStopping, "stopping"},
		{models.PipelineStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("expected %s, got %s", tt.want, tt.status)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3665 * time.Second, "1h1m5s"},
		{0, "0s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, got, tt.want)
		}
	}
}

func TestPipeline_Model(t *testing.T) {
	now := time.Now()
	pipeline := &models.Pipeline{
		ID:        uuid.New(),
		Name:      "test-pipeline",
		SourceID:  uuid.New(),
		Status:    models.PipelineStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
		StartedAt: &now,
	}

	if pipeline.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}

	if pipeline.Status != models.PipelineStatusRunning {
		t.Errorf("expected running status, got %s", pipeline.Status)
	}

	if pipeline.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
}

func TestTableMapping_Model(t *testing.T) {
	mapping := &models.TableMapping{
		ID:           uuid.New(),
		PipelineID:   uuid.New(),
		SourceSchema: "public",
		SourceTable:  "users",
		Enabled:      true,
		CreatedAt:    time.Now(),
	}

	if mapping.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}

	if !mapping.Enabled {
		t.Error("expected enabled to be true")
	}
}
