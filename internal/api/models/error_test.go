package models

import (
	"net/http"
	"testing"
)

func TestProblemDetails_Error(t *testing.T) {
	pd := &ProblemDetails{
		Type:   ErrorTypeValidation,
		Title:  "Validation Error",
		Status: http.StatusBadRequest,
		Detail: "Invalid input",
	}

	expected := "Validation Error: Invalid input"
	if pd.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, pd.Error())
	}
}

func TestNewValidationError(t *testing.T) {
	errors := []FieldError{
		{Field: "name", Message: "name is required"},
		{Field: "email", Message: "invalid email format"},
	}

	pd := NewValidationError("/api/v1/test", errors)

	if pd.Type != ErrorTypeValidation {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeValidation, pd.Type)
	}

	if pd.Status != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, pd.Status)
	}

	if pd.Instance != "/api/v1/test" {
		t.Errorf("expected instance '/api/v1/test', got '%s'", pd.Instance)
	}

	if len(pd.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(pd.Errors))
	}
}

func TestNewNotFoundError(t *testing.T) {
	pd := NewNotFoundError("/api/v1/sources/123", "Source not found")

	if pd.Type != ErrorTypeNotFound {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeNotFound, pd.Type)
	}

	if pd.Status != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, pd.Status)
	}

	if pd.Detail != "Source not found" {
		t.Errorf("expected detail 'Source not found', got '%s'", pd.Detail)
	}
}

func TestNewInternalError(t *testing.T) {
	pd := NewInternalError("/api/v1/test", "Database connection failed")

	if pd.Type != ErrorTypeInternal {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeInternal, pd.Type)
	}

	if pd.Status != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, pd.Status)
	}
}

func TestNewNotImplementedError(t *testing.T) {
	pd := NewNotImplementedError("/api/v1/sources")

	if pd.Type != ErrorTypeNotImplemented {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeNotImplemented, pd.Type)
	}

	if pd.Status != http.StatusNotImplemented {
		t.Errorf("expected status %d, got %d", http.StatusNotImplemented, pd.Status)
	}
}

func TestNewBadRequestError(t *testing.T) {
	pd := NewBadRequestError("/api/v1/test", "Invalid JSON")

	if pd.Type != ErrorTypeBadRequest {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeBadRequest, pd.Type)
	}

	if pd.Status != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, pd.Status)
	}
}

func TestNewRateLimitedError(t *testing.T) {
	pd := NewRateLimitedError("/api/v1/test")

	if pd.Type != ErrorTypeRateLimited {
		t.Errorf("expected type '%s', got '%s'", ErrorTypeRateLimited, pd.Type)
	}

	if pd.Status != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, pd.Status)
	}
}
