package services

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

func TestSourceService_Create_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.CreateSourceRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.CreateSourceRequest{
				Name:         "test-source",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "user",
				Password:     "pass",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &models.CreateSourceRequest{
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "user",
				Password:     "pass",
			},
			wantErr: true,
		},
		{
			name: "missing host",
			req: &models.CreateSourceRequest{
				Name:         "test-source",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "user",
				Password:     "pass",
			},
			wantErr: true,
		},
		{
			name: "missing database",
			req: &models.CreateSourceRequest{
				Name:     "test-source",
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			req: &models.CreateSourceRequest{
				Name:         "test-source",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Password:     "pass",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: &models.CreateSourceRequest{
				Name:         "test-source",
				Host:         "localhost",
				Port:         5432,
				DatabaseName: "testdb",
				Username:     "user",
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

func TestSourceService_Create_ApplyDefaults(t *testing.T) {
	req := &models.CreateSourceRequest{
		Name:         "test-source",
		Host:         "localhost",
		DatabaseName: "testdb",
		Username:     "user",
		Password:     "pass",
	}

	req.ApplyDefaults()

	if req.Type != "postgresql" {
		t.Errorf("expected type 'postgresql', got '%s'", req.Type)
	}
	if req.Port != 5432 {
		t.Errorf("expected port 5432, got %d", req.Port)
	}
	if req.SSLMode != "prefer" {
		t.Errorf("expected ssl_mode 'prefer', got '%s'", req.SSLMode)
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Resource: "source", ID: "123"}
	expected := "source not found: 123"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Errors: []models.FieldError{{Field: "name", Message: "required"}},
	}
	if err.Error() != "validation error" {
		t.Errorf("expected 'validation error', got '%s'", err.Error())
	}
}

func TestConflictError(t *testing.T) {
	err := &ConflictError{Message: "already exists"}
	if err.Error() != "already exists" {
		t.Errorf("expected 'already exists', got '%s'", err.Error())
	}
}

func TestUpdateSourceRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.UpdateSourceRequest
		wantErr bool
	}{
		{
			name:    "empty request is valid",
			req:     &models.UpdateSourceRequest{},
			wantErr: false,
		},
		{
			name: "valid name",
			req: &models.UpdateSourceRequest{
				Name: stringPtr("new-name"),
			},
			wantErr: false,
		},
		{
			name: "empty name not allowed",
			req: &models.UpdateSourceRequest{
				Name: stringPtr(""),
			},
			wantErr: true,
		},
		{
			name: "empty host not allowed",
			req: &models.UpdateSourceRequest{
				Host: stringPtr(""),
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			req: &models.UpdateSourceRequest{
				Port: intPtr(70000),
			},
			wantErr: true,
		},
		{
			name: "invalid port zero",
			req: &models.UpdateSourceRequest{
				Port: intPtr(0),
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


// Integration tests would require a real database connection
// and are typically run separately with docker-compose

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }

// MockSourceRepository would be used for unit testing the service
// with a mock repository, but for now we test validation only.

func TestSourceStatus(t *testing.T) {
	tests := []struct {
		status models.SourceStatus
		want   string
	}{
		{models.SourceStatusInactive, "inactive"},
		{models.SourceStatusActive, "active"},
		{models.SourceStatusError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("expected %s, got %s", tt.want, tt.status)
		}
	}
}

func TestSource_UUID(t *testing.T) {
	// Verify UUID generation works
	id := uuid.New()
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
}

// Test that validation catches bad port values
func TestCreateSourceRequest_Validate_Port(t *testing.T) {
	ctx := context.Background()
	_ = ctx // Would be used with actual service

	req := &models.CreateSourceRequest{
		Name:         "test",
		Host:         "localhost",
		Port:         -1,
		DatabaseName: "db",
		Username:     "user",
		Password:     "pass",
	}

	errors := req.Validate()
	if len(errors) == 0 {
		t.Error("expected validation error for negative port")
	}

	// Check that port field is flagged
	found := false
	for _, e := range errors {
		if e.Field == "port" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected port field error")
	}
}
