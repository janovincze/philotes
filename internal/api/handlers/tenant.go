// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// TenantHandler handles tenant-related HTTP requests.
type TenantHandler struct {
	tenantService *services.TenantService
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(tenantService *services.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// --- Tenant Operations ---

// Create creates a new tenant.
// POST /api/v1/tenants
func (h *TenantHandler) Create(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	tenant, err := h.tenantService.Create(c.Request.Context(), &req, authContext.User.ID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.TenantResponse{Tenant: tenant})
}

// List lists tenants the current user has access to.
// GET /api/v1/tenants
func (h *TenantHandler) List(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Global admins can see all tenants
	var tenants []models.Tenant
	var err error

	if authContext.User.Role == models.RoleAdmin {
		tenants, err = h.tenantService.List(c.Request.Context())
	} else {
		tenants, err = h.tenantService.ListByUser(c.Request.Context(), authContext.User.ID)
	}

	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TenantListResponse{
		Tenants:    tenants,
		TotalCount: len(tenants),
	})
}

// Get retrieves a tenant by ID.
// GET /api/v1/tenants/:id
func (h *TenantHandler) Get(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check membership unless global admin
	if authContext.User.Role != models.RoleAdmin {
		isMember, memberErr := h.tenantService.IsMember(c.Request.Context(), id, authContext.User.ID)
		if memberErr != nil {
			h.respondWithError(c, memberErr)
			return
		}
		if !isMember {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			return
		}
	}

	tenant, err := h.tenantService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.TenantResponse{Tenant: tenant})
}

// Update updates a tenant.
// PUT /api/v1/tenants/:id
func (h *TenantHandler) Update(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	var req models.UpdateTenantRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	tenant, updateErr := h.tenantService.Update(c.Request.Context(), id, &req)
	if updateErr != nil {
		h.respondWithError(c, updateErr)
		return
	}

	c.JSON(http.StatusOK, models.TenantResponse{Tenant: tenant})
}

// Delete deletes a tenant.
// DELETE /api/v1/tenants/:id
func (h *TenantHandler) Delete(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Only the owner or global admin can delete
	tenant, err := h.tenantService.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	isOwner := tenant.OwnerUserID != nil && *tenant.OwnerUserID == authContext.User.ID
	isGlobalAdmin := authContext.User.Role == models.RoleAdmin

	if !isOwner && !isGlobalAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Only the tenant owner can delete this tenant",
		))
		return
	}

	if err := h.tenantService.Delete(c.Request.Context(), id); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Member Operations ---

// ListMembers lists all members of a tenant.
// GET /api/v1/tenants/:id/members
func (h *TenantHandler) ListMembers(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check membership unless global admin
	if authContext.User.Role != models.RoleAdmin {
		isMember, memberErr := h.tenantService.IsMember(c.Request.Context(), id, authContext.User.ID)
		if memberErr != nil {
			h.respondWithError(c, memberErr)
			return
		}
		if !isMember {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			return
		}
	}

	members, err := h.tenantService.ListMembers(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.MemberListResponse{
		Members:    members,
		TotalCount: len(members),
	})
}

// AddMember adds a member to a tenant.
// POST /api/v1/tenants/:id/members
func (h *TenantHandler) AddMember(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	var req models.AddMemberRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	member, addErr := h.tenantService.AddMember(c.Request.Context(), id, &req)
	if addErr != nil {
		h.respondWithError(c, addErr)
		return
	}

	c.JSON(http.StatusCreated, models.MemberResponse{Member: member})
}

// UpdateMember updates a member's role.
// PUT /api/v1/tenants/:id/members/:user_id
func (h *TenantHandler) UpdateMember(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid user ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	var req models.UpdateMemberRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	member, updateErr := h.tenantService.UpdateMember(c.Request.Context(), id, userID, &req)
	if updateErr != nil {
		h.respondWithError(c, updateErr)
		return
	}

	c.JSON(http.StatusOK, models.MemberResponse{Member: member})
}

// RemoveMember removes a member from a tenant.
// DELETE /api/v1/tenants/:id/members/:user_id
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid user ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	if err := h.tenantService.RemoveMember(c.Request.Context(), id, userID); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Custom Role Operations ---

// ListRoles lists all custom roles in a tenant.
// GET /api/v1/tenants/:id/roles
func (h *TenantHandler) ListRoles(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check membership unless global admin
	if authContext.User.Role != models.RoleAdmin {
		isMember, memberErr := h.tenantService.IsMember(c.Request.Context(), id, authContext.User.ID)
		if memberErr != nil {
			h.respondWithError(c, memberErr)
			return
		}
		if !isMember {
			models.RespondWithError(c, models.NewNotTenantMemberError(c.Request.URL.Path))
			return
		}
	}

	roles, err := h.tenantService.ListCustomRoles(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.CustomRoleListResponse{
		Roles:      roles,
		TotalCount: len(roles),
	})
}

// CreateRole creates a new custom role in a tenant.
// POST /api/v1/tenants/:id/roles
func (h *TenantHandler) CreateRole(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	var req models.CreateCustomRoleRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	customRole, createErr := h.tenantService.CreateCustomRole(c.Request.Context(), id, &req)
	if createErr != nil {
		h.respondWithError(c, createErr)
		return
	}

	c.JSON(http.StatusCreated, models.CustomRoleResponse{Role: customRole})
}

// UpdateRole updates a custom role.
// PUT /api/v1/tenants/:id/roles/:role_id
func (h *TenantHandler) UpdateRole(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid role ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	// Verify the role belongs to this tenant
	existingRole, getErr := h.tenantService.GetCustomRole(c.Request.Context(), roleID)
	if getErr != nil {
		h.respondWithError(c, getErr)
		return
	}
	if existingRole.TenantID != id {
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"Role not found in this tenant",
		))
		return
	}

	var req models.UpdateCustomRoleRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	customRole, updateErr := h.tenantService.UpdateCustomRole(c.Request.Context(), roleID, &req)
	if updateErr != nil {
		h.respondWithError(c, updateErr)
		return
	}

	c.JSON(http.StatusOK, models.CustomRoleResponse{Role: customRole})
}

// DeleteRole deletes a custom role.
// DELETE /api/v1/tenants/:id/roles/:role_id
func (h *TenantHandler) DeleteRole(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil || authContext.User == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid tenant ID format",
		))
		return
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid role ID format",
		))
		return
	}

	// Check if user is tenant admin or global admin
	if authContext.User.Role != models.RoleAdmin {
		role, roleErr := h.tenantService.GetMemberRole(c.Request.Context(), id, authContext.User.ID)
		if roleErr != nil {
			h.respondWithError(c, roleErr)
			return
		}
		if role != models.TenantRoleAdmin {
			models.RespondWithError(c, models.NewInsufficientRoleError(c.Request.URL.Path))
			return
		}
	}

	// Verify the role belongs to this tenant
	existingRole, err := h.tenantService.GetCustomRole(c.Request.Context(), roleID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}
	if existingRole.TenantID != id {
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"Role not found in this tenant",
		))
		return
	}

	if err := h.tenantService.DeleteCustomRole(c.Request.Context(), roleID); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Register registers routes for the tenant handler.
func (h *TenantHandler) Register(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	tenants := rg.Group("/tenants")
	tenants.Use(authMiddleware)

	// Tenant CRUD
	tenants.POST("", h.Create)
	tenants.GET("", h.List)
	tenants.GET("/:id", h.Get)
	tenants.PUT("/:id", h.Update)
	tenants.DELETE("/:id", h.Delete)

	// Member management
	tenants.GET("/:id/members", h.ListMembers)
	tenants.POST("/:id/members", h.AddMember)
	tenants.PUT("/:id/members/:user_id", h.UpdateMember)
	tenants.DELETE("/:id/members/:user_id", h.RemoveMember)

	// Custom role management
	tenants.GET("/:id/roles", h.ListRoles)
	tenants.POST("/:id/roles", h.CreateRole)
	tenants.PUT("/:id/roles/:role_id", h.UpdateRole)
	tenants.DELETE("/:id/roles/:role_id", h.DeleteRole)
}

// respondWithError converts service errors to HTTP responses.
func (h *TenantHandler) respondWithError(c *gin.Context, err error) {
	var validationErr *services.ValidationError
	var notFoundErr *services.NotFoundError
	var conflictErr *services.ConflictError

	switch {
	case errors.As(err, &validationErr):
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			validationErr.Errors,
		))
	case errors.As(err, &notFoundErr):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			notFoundErr.Error(),
		))
	case errors.As(err, &conflictErr):
		models.RespondWithError(c, &models.ProblemDetails{
			Type:     "https://philotes.io/errors/conflict",
			Title:    "Conflict",
			Status:   http.StatusConflict,
			Detail:   conflictErr.Error(),
			Instance: c.Request.URL.Path,
		})
	case errors.Is(err, services.ErrCannotRemoveOwner):
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Cannot remove the tenant owner",
		))
	case errors.Is(err, services.ErrCannotDemoteLastAdmin):
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Cannot demote the last admin in the tenant",
		))
	case errors.Is(err, services.ErrCannotRemoveLastAdmin):
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Cannot remove the last admin from the tenant",
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an unexpected error occurred",
		))
	}
}
