// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
	"github.com/janovincze/philotes/internal/config"
)

// Context keys for tenant data.
const (
	TenantContextKey     = "tenant_context"
	TenantIDContextKey   = "tenant_id"
	TenantRoleContextKey = "tenant_role"
)

// TenantConfig holds tenant middleware configuration.
type TenantConfig struct {
	// Enabled enables multi-tenancy
	Enabled bool

	// TenantService is the tenant service for membership validation
	TenantService *services.TenantService

	// TenantHeader is the HTTP header to extract tenant ID from
	TenantHeader string

	// DefaultTenantID is used when multi-tenancy is disabled
	DefaultTenantID string

	// AllowCrossTenantAccess allows global admins to access any tenant
	AllowCrossTenantAccess bool
}

// TenantContext holds tenant context for a request.
type TenantContext struct {
	TenantID    uuid.UUID
	Tenant      *models.Tenant
	Role        models.TenantRole
	Permissions []string
}

// ExtractTenant returns a middleware that extracts tenant context from request.
// It reads tenant ID from header or JWT claims and sets the tenant context.
func ExtractTenant(cfg TenantConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			// Multi-tenancy disabled, use default tenant
			defaultID, err := uuid.Parse(cfg.DefaultTenantID)
			if err != nil {
				// Use hardcoded default if config is invalid
				defaultID = models.GetDefaultTenantUUID()
			}
			c.Set(TenantIDContextKey, defaultID)
			c.Next()
			return
		}

		// Try to get tenant ID from header
		tenantIDStr := c.GetHeader(cfg.TenantHeader)
		if tenantIDStr == "" {
			// Try X-Tenant-ID as fallback if different header configured
			if cfg.TenantHeader != "X-Tenant-ID" {
				tenantIDStr = c.GetHeader("X-Tenant-ID")
			}
		}

		if tenantIDStr != "" {
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				models.RespondWithError(c, &models.ProblemDetails{
					Type:     "https://philotes.io/errors/invalid-tenant-id",
					Title:    "Invalid Tenant ID",
					Status:   400,
					Detail:   "The provided tenant ID is not a valid UUID",
					Instance: c.Request.URL.Path,
				})
				c.Abort()
				return
			}
			c.Set(TenantIDContextKey, tenantID)
		}

		c.Next()
	}
}

// RequireTenant returns a middleware that requires a valid tenant context.
// It verifies the user is a member of the specified tenant.
// Must be used after Authenticate and ExtractTenant middleware.
func RequireTenant(cfg TenantConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			// Multi-tenancy disabled, allow all requests
			c.Next()
			return
		}

		// Get tenant ID from context
		tenantIDValue, exists := c.Get(TenantIDContextKey)
		if !exists {
			models.RespondWithError(c, &models.ProblemDetails{
				Type:     "https://philotes.io/errors/tenant-required",
				Title:    "Tenant Required",
				Status:   400,
				Detail:   "A tenant ID must be specified via the " + cfg.TenantHeader + " header",
				Instance: c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			models.RespondWithError(c, &models.ProblemDetails{
				Type:     "https://philotes.io/errors/invalid-tenant-id",
				Title:    "Invalid Tenant ID",
				Status:   400,
				Detail:   "The tenant ID is not valid",
				Instance: c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		// Get auth context
		authContext := GetAuthContext(c)
		if authContext == nil || authContext.User == nil {
			models.RespondWithError(c, models.NewUnauthorizedError(
				c.Request.URL.Path,
				"Authentication required to access tenant resources",
			))
			c.Abort()
			return
		}

		// Check if user is global admin with cross-tenant access
		if cfg.AllowCrossTenantAccess && authContext.User.Role == models.RoleAdmin {
			// Global admin can access any tenant
			// Set tenant context without membership check
			c.Set(TenantContextKey, &TenantContext{
				TenantID:    tenantID,
				Role:        models.TenantRoleAdmin,
				Permissions: models.TenantRolePermissions[models.TenantRoleAdmin],
			})
			c.Next()
			return
		}

		// Verify user is a member of the tenant
		if cfg.TenantService == nil {
			// TenantService not configured - this is a configuration error
			models.RespondWithError(c, &models.ProblemDetails{
				Type:     "https://philotes.io/errors/configuration-error",
				Title:    "Configuration Error",
				Status:   500,
				Detail:   "Tenant service not configured",
				Instance: c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		isMember, err := cfg.TenantService.IsMember(c.Request.Context(), tenantID, authContext.User.ID)
		if err != nil {
			models.RespondWithError(c, &models.ProblemDetails{
				Type:     "https://philotes.io/errors/internal-error",
				Title:    "Internal Error",
				Status:   500,
				Detail:   "Failed to verify tenant membership",
				Instance: c.Request.URL.Path,
			})
			c.Abort()
			return
		}

		if !isMember {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			c.Abort()
			return
		}

		// Get user's role and permissions in this tenant
		role, err := cfg.TenantService.GetMemberRole(c.Request.Context(), tenantID, authContext.User.ID)
		if err != nil {
			// This shouldn't happen if isMember is true, but handle gracefully
			role = models.TenantRoleViewer
		}

		permissions, err := cfg.TenantService.GetMemberPermissions(c.Request.Context(), tenantID, authContext.User.ID)
		if err != nil {
			permissions = models.TenantRolePermissions[models.TenantRoleViewer]
		}

		// Set tenant context
		c.Set(TenantContextKey, &TenantContext{
			TenantID:    tenantID,
			Role:        role,
			Permissions: permissions,
		})
		c.Set(TenantRoleContextKey, role)

		c.Next()
	}
}

// RequireTenantRole returns a middleware that requires a minimum tenant role.
// Must be used after RequireTenant middleware.
// For custom roles, it checks if the user has all permissions that the minimum role would have.
func RequireTenantRole(minRole models.TenantRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantContext := GetTenantContext(c)
		if tenantContext == nil {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			c.Abort()
			return
		}

		if !hasMinimumRole(tenantContext, minRole) {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTenantPermission returns a middleware that requires a specific permission within the tenant.
// Must be used after RequireTenant middleware.
func RequireTenantPermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantContext := GetTenantContext(c)
		if tenantContext == nil {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			c.Abort()
			return
		}

		if !hasTenantPermission(tenantContext, permission) {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetTenantContext retrieves the tenant context from a Gin context.
func GetTenantContext(c *gin.Context) *TenantContext {
	value, exists := c.Get(TenantContextKey)
	if !exists {
		return nil
	}
	tenantContext, ok := value.(*TenantContext)
	if !ok {
		return nil
	}
	return tenantContext
}

// GetTenantID retrieves the tenant ID from a Gin context.
func GetTenantID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(TenantIDContextKey)
	if !exists {
		return uuid.Nil, false
	}
	tenantID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return tenantID, true
}

// GetTenantIDOrDefault returns the tenant ID from context or the default tenant ID.
func GetTenantIDOrDefault(c *gin.Context, cfg *config.MultiTenancyConfig) uuid.UUID {
	tenantID, ok := GetTenantID(c)
	if ok {
		return tenantID
	}

	// Use default tenant ID
	if cfg != nil && cfg.DefaultTenantID != "" {
		if id, err := uuid.Parse(cfg.DefaultTenantID); err == nil {
			return id
		}
	}

	return models.GetDefaultTenantUUID()
}

// hasMinimumRole checks if the user's role meets or exceeds the minimum required role.
// For custom roles, it checks if the user has all permissions that the minimum role would have.
func hasMinimumRole(tc *TenantContext, minRole models.TenantRole) bool {
	roleRanks := map[models.TenantRole]int{
		models.TenantRoleViewer:   1,
		models.TenantRoleOperator: 2,
		models.TenantRoleAdmin:    3,
	}

	userRank, ok := roleRanks[tc.Role]
	if !ok {
		// Custom role - check if user has all permissions of the minimum role
		minRolePerms := models.TenantRolePermissions[minRole]
		if hasAllPermissions(tc.Permissions, minRolePerms) {
			return true
		}
		// Fall back to viewer level if permissions don't match
		userRank = 1
	}

	minRank, ok := roleRanks[minRole]
	if !ok {
		minRank = 1
	}

	return userRank >= minRank
}

// hasAllPermissions checks if the user's permissions include all required permissions.
func hasAllPermissions(userPerms, requiredPerms []string) bool {
	permSet := make(map[string]bool, len(userPerms))
	for _, p := range userPerms {
		permSet[p] = true
	}
	for _, required := range requiredPerms {
		if !permSet[required] {
			return false
		}
	}
	return true
}

// hasTenantPermission checks if the tenant context has a specific permission.
func hasTenantPermission(tc *TenantContext, permission string) bool {
	for _, p := range tc.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
