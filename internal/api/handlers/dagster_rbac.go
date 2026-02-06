// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
)

// resourcePatternRegex validates resource patterns for Dagster RBAC.
// Allows alphanumeric, underscores, hyphens, slashes, and wildcards.
// Wildcards (*) are only allowed at the start or end of a segment (e.g., "etl_*", "*/warehouse").
// Path traversal sequences like ".." are explicitly rejected.
var resourcePatternRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-/*]+$`)

// resourcePatternDangerousRegex detects potentially dangerous patterns like path traversal.
var resourcePatternDangerousRegex = regexp.MustCompile(`\.\.`)

// DagsterRBACHandler handles Dagster RBAC-related HTTP requests.
type DagsterRBACHandler struct {
	repo     *repositories.DagsterRBACRepository
	userRepo *repositories.UserRepository
}

// NewDagsterRBACHandler creates a new DagsterRBACHandler.
func NewDagsterRBACHandler(repo *repositories.DagsterRBACRepository, userRepo *repositories.UserRepository) *DagsterRBACHandler {
	return &DagsterRBACHandler{repo: repo, userRepo: userRepo}
}

// GetUserPermissions returns all Dagster permissions for a user.
// GET /api/v1/users/:id/dagster-permissions
func (h *DagsterRBACHandler) GetUserPermissions(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid user ID",
		))
		return
	}

	// Check if user is requesting their own permissions or is an admin
	if authContext.User.ID != userID && authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Cannot view other user's permissions",
		))
		return
	}

	// Get roles
	roles, err := h.repo.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	// Get custom permissions
	customPerms, err := h.repo.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	// Get effective permissions
	effectivePerms, err := h.repo.GetAllEffectivePermissions(c.Request.Context(), userID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	response := models.DagsterUserPermissions{
		UserID:               userID,
		Roles:                roles,
		CustomPermissions:    customPerms,
		EffectivePermissions: effectivePerms,
	}

	c.JSON(http.StatusOK, response)
}

// AssignRole assigns a Dagster role to a user.
// POST /api/v1/users/:id/dagster-roles
func (h *DagsterRBACHandler) AssignRole(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Only admins can assign roles
	if authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Admin access required",
		))
		return
	}

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid user ID",
		))
		return
	}

	var req models.AssignDagsterRoleRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			errs,
		))
		return
	}

	// Validate resource pattern if provided
	if req.ResourcePattern != "" {
		if len(req.ResourcePattern) > 255 {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"resource pattern too long (max 255 characters)",
			))
			return
		}
		if !resourcePatternRegex.MatchString(req.ResourcePattern) {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"invalid resource pattern: only alphanumeric, underscore, hyphen, slash, and wildcard (*) allowed",
			))
			return
		}
		if resourcePatternDangerousRegex.MatchString(req.ResourcePattern) {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"invalid resource pattern: path traversal sequences are not allowed",
			))
			return
		}
	}

	// Verify user exists
	_, err = h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"user not found",
			))
			return
		}
		h.respondWithError(c, err)
		return
	}

	assignment, err := h.repo.AssignRole(c.Request.Context(), userID, req.Role, req.ResourcePattern)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, assignment)
}

// RemoveRole removes a Dagster role assignment.
// DELETE /api/v1/dagster-roles/:id
func (h *DagsterRBACHandler) RemoveRole(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Only admins can remove roles
	if authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Admin access required",
		))
		return
	}

	// Parse role assignment ID from URL
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid role assignment ID",
		))
		return
	}

	if err := h.repo.RemoveRole(c.Request.Context(), id); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AddPermission adds a custom Dagster permission for a user.
// POST /api/v1/users/:id/dagster-permissions
func (h *DagsterRBACHandler) AddPermission(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Only admins can add permissions
	if authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Admin access required",
		))
		return
	}

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid user ID",
		))
		return
	}

	var req models.AddDagsterPermissionRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			errs,
		))
		return
	}

	// Verify user exists
	_, err = h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"user not found",
			))
			return
		}
		h.respondWithError(c, err)
		return
	}

	permission, err := h.repo.AddPermission(c.Request.Context(), userID, req.Permission)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, permission)
}

// RemovePermission removes a custom Dagster permission.
// DELETE /api/v1/dagster-permissions/:id
func (h *DagsterRBACHandler) RemovePermission(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Only admins can remove permissions
	if authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Admin access required",
		))
		return
	}

	// Parse permission ID from URL
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid permission ID",
		))
		return
	}

	if err := h.repo.RemovePermission(c.Request.Context(), id); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAllRoleAssignments lists all Dagster role assignments (admin only).
// GET /api/v1/dagster-roles
func (h *DagsterRBACHandler) ListAllRoleAssignments(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	// Only admins can list all assignments
	if authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewForbiddenError(
			c.Request.URL.Path,
			"Admin access required",
		))
		return
	}

	limit := 50
	offset := 0

	assignments, total, err := h.repo.ListAllRoleAssignments(c.Request.Context(), limit, offset)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role_assignments": assignments,
		"total_count":      total,
	})
}

// GetAvailableRoles returns information about available Dagster roles.
// GET /api/v1/dagster-roles/available
func (h *DagsterRBACHandler) GetAvailableRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"roles": models.AvailableDagsterRoles,
	})
}

// respondWithError converts repository errors to HTTP responses.
func (h *DagsterRBACHandler) respondWithError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrDagsterRoleNotFound):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"Dagster role assignment not found",
		))
	case errors.Is(err, repositories.ErrDagsterPermissionNotFound):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"Dagster permission not found",
		))
	case errors.Is(err, repositories.ErrDagsterRoleExists):
		models.RespondWithError(c, models.NewConflictError(
			c.Request.URL.Path,
			"Role assignment already exists",
		))
	case errors.Is(err, repositories.ErrDagsterPermissionExists):
		models.RespondWithError(c, models.NewConflictError(
			c.Request.URL.Path,
			"Permission already exists",
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"Internal server error",
		))
	}
}
