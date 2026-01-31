// Package models provides API request and response types.
package models

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TenantRole represents the role of a user within a tenant.
type TenantRole string

const (
	// TenantRoleAdmin has full access within the tenant.
	TenantRoleAdmin TenantRole = "admin"
	// TenantRoleOperator can manage pipelines and sources within the tenant.
	TenantRoleOperator TenantRole = "operator"
	// TenantRoleViewer has read-only access within the tenant.
	TenantRoleViewer TenantRole = "viewer"
	// TenantRoleCustom uses a custom role with specific permissions.
	TenantRoleCustom TenantRole = "custom"
)

// Tenant permission constants.
const (
	PermissionTenantsRead  = "tenants:read"
	PermissionTenantsWrite = "tenants:write"
	PermissionMembersRead  = "members:read"
	PermissionMembersWrite = "members:write"
	PermissionRolesRead    = "roles:read"
	PermissionRolesWrite   = "roles:write"
)

// ValidTenantPermissions is the set of all valid tenant permission strings.
var ValidTenantPermissions = map[string]bool{
	PermissionTenantsRead:    true,
	PermissionTenantsWrite:   true,
	PermissionMembersRead:    true,
	PermissionMembersWrite:   true,
	PermissionRolesRead:      true,
	PermissionRolesWrite:     true,
	PermissionSourcesRead:    true,
	PermissionSourcesWrite:   true,
	PermissionPipelinesRead:  true,
	PermissionPipelinesWrite: true,
	PermissionPipelinesCtrl:  true,
	PermissionAPIKeysRead:    true,
	PermissionAPIKeysWrite:   true,
	PermissionScalingRead:    true,
	PermissionScalingWrite:   true,
	PermissionAlertsRead:     true,
	PermissionAlertsWrite:    true,
}

// IsValidPermission checks if a permission string is valid.
func IsValidPermission(permission string) bool {
	return ValidTenantPermissions[permission]
}

// ValidatePermissions checks if all permissions in the slice are valid.
func ValidatePermissions(permissions []string) []string {
	var invalid []string
	for _, p := range permissions {
		if !IsValidPermission(p) {
			invalid = append(invalid, p)
		}
	}
	return invalid
}

// TenantRolePermissions maps tenant roles to their default permissions.
var TenantRolePermissions = map[TenantRole][]string{
	TenantRoleAdmin: {
		PermissionTenantsRead, PermissionTenantsWrite,
		PermissionMembersRead, PermissionMembersWrite,
		PermissionRolesRead, PermissionRolesWrite,
		PermissionSourcesRead, PermissionSourcesWrite,
		PermissionPipelinesRead, PermissionPipelinesWrite, PermissionPipelinesCtrl,
		PermissionAPIKeysRead, PermissionAPIKeysWrite,
		PermissionScalingRead, PermissionScalingWrite,
		PermissionAlertsRead, PermissionAlertsWrite,
	},
	TenantRoleOperator: {
		PermissionTenantsRead,
		PermissionMembersRead,
		PermissionRolesRead,
		PermissionSourcesRead, PermissionSourcesWrite,
		PermissionPipelinesRead, PermissionPipelinesWrite, PermissionPipelinesCtrl,
		PermissionAPIKeysRead, PermissionAPIKeysWrite,
		PermissionScalingRead, PermissionScalingWrite,
		PermissionAlertsRead, PermissionAlertsWrite,
	},
	TenantRoleViewer: {
		PermissionTenantsRead,
		PermissionMembersRead,
		PermissionRolesRead,
		PermissionSourcesRead,
		PermissionPipelinesRead,
		PermissionAPIKeysRead,
		PermissionScalingRead,
		PermissionAlertsRead,
	},
}

// Tenant represents a tenant (organization) in the system.
type Tenant struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	OwnerUserID *uuid.UUID             `json:"owner_user_id,omitempty"`
	IsActive    bool                   `json:"is_active"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TenantMember represents a user's membership in a tenant.
type TenantMember struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	UserID            uuid.UUID  `json:"user_id"`
	Role              TenantRole `json:"role"`
	CustomPermissions []string   `json:"custom_permissions,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Populated on read
	User *User `json:"user,omitempty"`
}

// TenantCustomRole represents a custom role defined for a tenant.
type TenantCustomRole struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// slugRegex validates tenant slugs.
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,98}[a-z0-9])?$`)

// CreateTenantRequest represents a request to create a tenant.
type CreateTenantRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Slug     string                 `json:"slug" binding:"required"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// Validate validates the create tenant request.
func (r *CreateTenantRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if r.Slug == "" {
		errors = append(errors, FieldError{Field: "slug", Message: "slug is required"})
	} else if !slugRegex.MatchString(r.Slug) {
		errors = append(errors, FieldError{
			Field:   "slug",
			Message: "slug must be lowercase alphanumeric with hyphens, 1-100 characters",
		})
	}
	return errors
}

// UpdateTenantRequest represents a request to update a tenant.
type UpdateTenantRequest struct {
	Name     *string                 `json:"name,omitempty"`
	Slug     *string                 `json:"slug,omitempty"`
	IsActive *bool                   `json:"is_active,omitempty"`
	Settings *map[string]interface{} `json:"settings,omitempty"`
}

// Validate validates the update tenant request.
func (r *UpdateTenantRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Slug != nil && !slugRegex.MatchString(*r.Slug) {
		errors = append(errors, FieldError{
			Field:   "slug",
			Message: "slug must be lowercase alphanumeric with hyphens, 1-100 characters",
		})
	}
	return errors
}

// AddMemberRequest represents a request to add a member to a tenant.
type AddMemberRequest struct {
	UserID            uuid.UUID  `json:"user_id" binding:"required"`
	Role              TenantRole `json:"role" binding:"required"`
	CustomPermissions []string   `json:"custom_permissions,omitempty"`
}

// Validate validates the add member request.
func (r *AddMemberRequest) Validate() []FieldError {
	var errors []FieldError
	if r.UserID == uuid.Nil {
		errors = append(errors, FieldError{Field: "user_id", Message: "user_id is required"})
	}
	if r.Role == "" {
		errors = append(errors, FieldError{Field: "role", Message: "role is required"})
	} else if !isValidTenantRole(r.Role) {
		errors = append(errors, FieldError{Field: "role", Message: "invalid role"})
	}
	if len(r.CustomPermissions) > 0 {
		if invalid := ValidatePermissions(r.CustomPermissions); len(invalid) > 0 {
			errors = append(errors, FieldError{
				Field:   "custom_permissions",
				Message: "invalid permissions: " + strings.Join(invalid, ", "),
			})
		}
	}
	return errors
}

// ApplyDefaults applies default values to the request.
func (r *AddMemberRequest) ApplyDefaults() {
	if r.Role == "" {
		r.Role = TenantRoleViewer
	}
}

// UpdateMemberRequest represents a request to update a member's role.
type UpdateMemberRequest struct {
	Role              *TenantRole `json:"role,omitempty"`
	CustomPermissions *[]string   `json:"custom_permissions,omitempty"`
}

// Validate validates the update member request.
func (r *UpdateMemberRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Role != nil && !isValidTenantRole(*r.Role) {
		errors = append(errors, FieldError{Field: "role", Message: "invalid role"})
	}
	if r.CustomPermissions != nil && len(*r.CustomPermissions) > 0 {
		if invalid := ValidatePermissions(*r.CustomPermissions); len(invalid) > 0 {
			errors = append(errors, FieldError{
				Field:   "custom_permissions",
				Message: "invalid permissions: " + strings.Join(invalid, ", "),
			})
		}
	}
	return errors
}

// CreateCustomRoleRequest represents a request to create a custom role.
type CreateCustomRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions" binding:"required"`
}

// Validate validates the create custom role request.
func (r *CreateCustomRoleRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if len(r.Permissions) == 0 {
		errors = append(errors, FieldError{Field: "permissions", Message: "at least one permission is required"})
	} else if invalid := ValidatePermissions(r.Permissions); len(invalid) > 0 {
		errors = append(errors, FieldError{
			Field:   "permissions",
			Message: "invalid permissions: " + strings.Join(invalid, ", "),
		})
	}
	return errors
}

// UpdateCustomRoleRequest represents a request to update a custom role.
type UpdateCustomRoleRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Permissions *[]string `json:"permissions,omitempty"`
}

// Validate validates the update custom role request.
func (r *UpdateCustomRoleRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Permissions != nil {
		if len(*r.Permissions) == 0 {
			errors = append(errors, FieldError{Field: "permissions", Message: "at least one permission is required"})
		} else if invalid := ValidatePermissions(*r.Permissions); len(invalid) > 0 {
			errors = append(errors, FieldError{
				Field:   "permissions",
				Message: "invalid permissions: " + strings.Join(invalid, ", "),
			})
		}
	}
	return errors
}

// TenantResponse wraps a tenant for API responses.
type TenantResponse struct {
	Tenant *Tenant `json:"tenant"`
}

// TenantListResponse wraps a list of tenants for API responses.
type TenantListResponse struct {
	Tenants    []Tenant `json:"tenants"`
	TotalCount int      `json:"total_count"`
}

// MemberResponse wraps a tenant member for API responses.
type MemberResponse struct {
	Member *TenantMember `json:"member"`
}

// MemberListResponse wraps a list of tenant members for API responses.
type MemberListResponse struct {
	Members    []TenantMember `json:"members"`
	TotalCount int            `json:"total_count"`
}

// CustomRoleResponse wraps a custom role for API responses.
type CustomRoleResponse struct {
	Role *TenantCustomRole `json:"role"`
}

// CustomRoleListResponse wraps a list of custom roles for API responses.
type CustomRoleListResponse struct {
	Roles      []TenantCustomRole `json:"roles"`
	TotalCount int                `json:"total_count"`
}

// Tenant error types.
const (
	ErrorTypeTenantNotFound      = "https://philotes.io/errors/tenant-not-found"
	ErrorTypeTenantSlugConflict  = "https://philotes.io/errors/tenant-slug-conflict"
	ErrorTypeMemberNotFound      = "https://philotes.io/errors/member-not-found"
	ErrorTypeMemberAlreadyExists = "https://philotes.io/errors/member-already-exists"
	ErrorTypeRoleNotFound        = "https://philotes.io/errors/role-not-found"
	ErrorTypeRoleNameConflict    = "https://philotes.io/errors/role-name-conflict"
	ErrorTypeNotTenantMember     = "https://philotes.io/errors/not-tenant-member"
	ErrorTypeInsufficientRole    = "https://philotes.io/errors/insufficient-role"
)

// NewTenantNotFoundError creates a tenant not found error.
func NewTenantNotFoundError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeTenantNotFound,
		Title:    "Tenant Not Found",
		Status:   http.StatusNotFound,
		Detail:   "The requested tenant was not found",
		Instance: instance,
	}
}

// NewTenantSlugConflictError creates a tenant slug conflict error.
func NewTenantSlugConflictError(instance, slug string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeTenantSlugConflict,
		Title:    "Tenant Slug Conflict",
		Status:   http.StatusConflict,
		Detail:   "A tenant with slug '" + slug + "' already exists",
		Instance: instance,
	}
}

// NewMemberNotFoundError creates a member not found error.
func NewMemberNotFoundError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeMemberNotFound,
		Title:    "Member Not Found",
		Status:   http.StatusNotFound,
		Detail:   "The requested member was not found",
		Instance: instance,
	}
}

// NewMemberAlreadyExistsError creates a member already exists error.
func NewMemberAlreadyExistsError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeMemberAlreadyExists,
		Title:    "Member Already Exists",
		Status:   http.StatusConflict,
		Detail:   "The user is already a member of this tenant",
		Instance: instance,
	}
}

// NewRoleNotFoundError creates a role not found error.
func NewRoleNotFoundError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeRoleNotFound,
		Title:    "Role Not Found",
		Status:   http.StatusNotFound,
		Detail:   "The requested role was not found",
		Instance: instance,
	}
}

// NewRoleNameConflictError creates a role name conflict error.
func NewRoleNameConflictError(instance, name string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeRoleNameConflict,
		Title:    "Role Name Conflict",
		Status:   http.StatusConflict,
		Detail:   "A role with name '" + name + "' already exists in this tenant",
		Instance: instance,
	}
}

// NewNotTenantMemberError creates a not tenant member error.
func NewNotTenantMemberError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeNotTenantMember,
		Title:    "Not a Tenant Member",
		Status:   http.StatusForbidden,
		Detail:   "You are not a member of this tenant",
		Instance: instance,
	}
}

// NewInsufficientRoleError creates an insufficient role error.
func NewInsufficientRoleError(instance string) *ProblemDetails {
	return &ProblemDetails{
		Type:     ErrorTypeInsufficientRole,
		Title:    "Insufficient Role",
		Status:   http.StatusForbidden,
		Detail:   "You do not have sufficient permissions for this operation",
		Instance: instance,
	}
}

// isValidTenantRole checks if a role is valid.
func isValidTenantRole(role TenantRole) bool {
	switch role {
	case TenantRoleAdmin, TenantRoleOperator, TenantRoleViewer, TenantRoleCustom:
		return true
	}
	return false
}

// DefaultTenantID is the UUID for the default system tenant.
const DefaultTenantID = "00000000-0000-0000-0000-000000000001"

// GetDefaultTenantUUID returns the default tenant UUID.
func GetDefaultTenantUUID() uuid.UUID {
	// DefaultTenantID is a constant; MustParse will panic if it is ever invalid.
	return uuid.MustParse(DefaultTenantID)
}
