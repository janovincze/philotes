// Package models provides API request and response types.
package models

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserRole represents the role of a user.
type UserRole string

const (
	// RoleAdmin has full access to all resources.
	RoleAdmin UserRole = "admin"
	// RoleOperator can manage pipelines and sources.
	RoleOperator UserRole = "operator"
	// RoleViewer has read-only access.
	RoleViewer UserRole = "viewer"
)

// Permission constants for fine-grained access control.
const (
	PermissionSourcesRead    = "sources:read"
	PermissionSourcesWrite   = "sources:write"
	PermissionPipelinesRead  = "pipelines:read"
	PermissionPipelinesWrite = "pipelines:write"
	PermissionPipelinesCtrl  = "pipelines:control"
	PermissionAPIKeysRead    = "api-keys:read"
	PermissionAPIKeysWrite   = "api-keys:write"
	PermissionUsersRead      = "users:read"
	PermissionUsersWrite     = "users:write"
	PermissionScalingRead    = "scaling:read"
	PermissionScalingWrite   = "scaling:write"
	PermissionAlertsRead     = "alerts:read"
	PermissionAlertsWrite    = "alerts:write"
)

// RolePermissions maps roles to their default permissions.
var RolePermissions = map[UserRole][]string{
	RoleAdmin: {
		PermissionSourcesRead, PermissionSourcesWrite,
		PermissionPipelinesRead, PermissionPipelinesWrite, PermissionPipelinesCtrl,
		PermissionAPIKeysRead, PermissionAPIKeysWrite,
		PermissionUsersRead, PermissionUsersWrite,
		PermissionScalingRead, PermissionScalingWrite,
		PermissionAlertsRead, PermissionAlertsWrite,
	},
	RoleOperator: {
		PermissionSourcesRead, PermissionSourcesWrite,
		PermissionPipelinesRead, PermissionPipelinesWrite, PermissionPipelinesCtrl,
		PermissionAPIKeysRead, PermissionAPIKeysWrite,
		PermissionScalingRead, PermissionScalingWrite,
		PermissionAlertsRead, PermissionAlertsWrite,
	},
	RoleViewer: {
		PermissionSourcesRead,
		PermissionPipelinesRead,
		PermissionAPIKeysRead,
		PermissionScalingRead,
		PermissionAlertsRead,
	},
}

// User represents a user in the system.
type User struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name,omitempty"`
	Role        UserRole   `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	UserID       *uuid.UUID             `json:"user_id,omitempty"`
	APIKeyID     *uuid.UUID             `json:"api_key_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Audit action constants.
const (
	AuditActionLogin         = "login"
	AuditActionLoginFailed   = "login_failed"
	AuditActionLogout        = "logout"
	AuditActionAPIKeyCreated = "api_key_created"
	AuditActionAPIKeyRevoked = "api_key_revoked"
	AuditActionAPIKeyDeleted = "api_key_deleted"
	AuditActionAPIKeyUsed    = "api_key_used"
	AuditActionUserCreated   = "user_created"
	AuditActionUserUpdated   = "user_updated"
	AuditActionUserDeleted   = "user_deleted"
	AuditActionUnauthorized  = "unauthorized"
	AuditActionForbidden     = "forbidden"
)

// JWTClaims represents the claims in a JWT token.
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Role        UserRole  `json:"role"`
	Permissions []string  `json:"permissions,omitempty"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Validate validates the login request.
func (r *LoginRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Email == "" {
		errors = append(errors, FieldError{Field: "email", Message: "email is required"})
	}
	if r.Password == "" {
		errors = append(errors, FieldError{Field: "password", Message: "password is required"})
	}
	return errors
}

// LoginResponse represents a login response.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Name     string   `json:"name,omitempty"`
	Role     UserRole `json:"role,omitempty"`
}

// Validate validates the create user request.
func (r *CreateUserRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Email == "" {
		errors = append(errors, FieldError{Field: "email", Message: "email is required"})
	}
	if r.Password == "" {
		errors = append(errors, FieldError{Field: "password", Message: "password is required"})
	} else if len(r.Password) < 8 {
		errors = append(errors, FieldError{Field: "password", Message: "password must be at least 8 characters"})
	}
	if r.Role != "" && r.Role != RoleAdmin && r.Role != RoleOperator && r.Role != RoleViewer {
		errors = append(errors, FieldError{Field: "role", Message: "invalid role"})
	}
	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateUserRequest) ApplyDefaults() {
	if r.Role == "" {
		r.Role = RoleViewer
	}
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Name     *string   `json:"name,omitempty"`
	Role     *UserRole `json:"role,omitempty"`
	IsActive *bool     `json:"is_active,omitempty"`
}

// CreateAPIKeyRequest represents a request to create an API key.
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Permissions []string   `json:"permissions,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Validate validates the create API key request.
func (r *CreateAPIKeyRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	return errors
}

// CreateAPIKeyResponse represents the response when creating an API key.
type CreateAPIKeyResponse struct {
	APIKey *APIKey `json:"api_key"`
	Key    string  `json:"key"` // Plaintext key, shown only once
}

// UserResponse wraps a user for API responses.
type UserResponse struct {
	User *User `json:"user"`
}

// UserListResponse wraps a list of users for API responses.
type UserListResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"total_count"`
}

// APIKeyResponse wraps an API key for API responses.
type APIKeyResponse struct {
	APIKey *APIKey `json:"api_key"`
}

// APIKeyListResponse wraps a list of API keys for API responses.
type APIKeyListResponse struct {
	APIKeys    []APIKey `json:"api_keys"`
	TotalCount int      `json:"total_count"`
}

// Auth error types.
const (
	ErrorTypeUnauthorized = "https://philotes.io/errors/unauthorized"
	ErrorTypeForbidden    = "https://philotes.io/errors/forbidden"
)

// NewUnauthorizedError creates an unauthorized error.
func NewUnauthorizedError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeUnauthorized,
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	}
}

// NewForbiddenError creates a forbidden error.
func NewForbiddenError(instance, detail string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeForbidden,
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	}
}

// AuthContext holds authentication context for a request.
type AuthContext struct {
	User        *User
	APIKey      *APIKey
	Permissions []string
	IsAPIKey    bool
}

// HasPermission checks if the auth context has a specific permission.
func (a *AuthContext) HasPermission(permission string) bool {
	for _, p := range a.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
