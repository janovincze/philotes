// Package models provides API request and response types.
package models

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProblemDetails represents an RFC 7807 problem details response.
type ProblemDetails struct {
	// Type is a URI reference identifying the problem type.
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`

	// Status is the HTTP status code.
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference identifying the specific occurrence.
	Instance string `json:"instance,omitempty"`

	// Errors contains field-level validation errors.
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (p *ProblemDetails) Error() string {
	return fmt.Sprintf("%s: %s", p.Title, p.Detail)
}

// Common error types.
const (
	ErrorTypeValidation     = "https://philotes.io/errors/validation-error"
	ErrorTypeNotFound       = "https://philotes.io/errors/not-found"
	ErrorTypeInternal       = "https://philotes.io/errors/internal-error"
	ErrorTypeNotImplemented = "https://philotes.io/errors/not-implemented"
	ErrorTypeBadRequest     = "https://philotes.io/errors/bad-request"
	ErrorTypeRateLimited    = "https://philotes.io/errors/rate-limited"
	ErrorTypeConflict       = "https://philotes.io/errors/conflict"
)

// NewValidationError creates a validation error with field errors.
func NewValidationError(instance string, errors []FieldError) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeValidation,
		Title:    "Validation Error",
		Status:   http.StatusBadRequest,
		Detail:   "The request contains invalid fields",
		Instance: instance,
		Errors:   errors,
	}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeNotFound,
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	}
}

// NewInternalError creates an internal server error.
func NewInternalError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeInternal,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: instance,
	}
}

// NewNotImplementedError creates a not implemented error.
func NewNotImplementedError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeNotImplemented,
		Title:    "Not Implemented",
		Status:   http.StatusNotImplemented,
		Detail:   "This endpoint is not yet implemented",
		Instance: instance,
	}
}

// NewBadRequestError creates a bad request error.
func NewBadRequestError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeBadRequest,
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
	}
}

// NewRateLimitedError creates a rate limited error.
func NewRateLimitedError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeRateLimited,
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   "Rate limit exceeded. Please try again later.",
		Instance: instance,
	}
}

// NewConflictError creates a conflict error.
func NewConflictError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeConflict,
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   detail,
		Instance: instance,
	}
}

// RespondWithError sends a ProblemDetails error response.
func RespondWithError(c *gin.Context, err *ProblemDetails) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(err.Status, err)
}
