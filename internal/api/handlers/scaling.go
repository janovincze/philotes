// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// ScalingHandler handles scaling-related HTTP requests.
type ScalingHandler struct {
	service *services.ScalingService
}

// NewScalingHandler creates a new ScalingHandler.
func NewScalingHandler(service *services.ScalingService) *ScalingHandler {
	return &ScalingHandler{service: service}
}

// Register adds all scaling routes to the router.
func (h *ScalingHandler) Register(rg *gin.RouterGroup) {
	// Scaling Policies
	rg.POST("/scaling/policies", h.CreatePolicy)
	rg.GET("/scaling/policies", h.ListPolicies)
	rg.GET("/scaling/policies/:id", h.GetPolicy)
	rg.PUT("/scaling/policies/:id", h.UpdatePolicy)
	rg.DELETE("/scaling/policies/:id", h.DeletePolicy)
	rg.POST("/scaling/policies/:id/enable", h.EnablePolicy)
	rg.POST("/scaling/policies/:id/disable", h.DisablePolicy)
	rg.POST("/scaling/policies/:id/evaluate", h.EvaluatePolicy)
	rg.GET("/scaling/policies/:id/state", h.GetPolicyState)

	// Scaling History
	rg.GET("/scaling/history", h.ListHistory)
	rg.GET("/scaling/policies/:id/history", h.GetPolicyHistory)
}

// CreatePolicy creates a new scaling policy.
// POST /api/v1/scaling/policies
func (h *ScalingHandler) CreatePolicy(c *gin.Context) {
	var req models.CreateScalingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	policy, err := h.service.CreatePolicy(c.Request.Context(), &req)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.ScalingPolicyResponse{Policy: policy})
}

// GetPolicy retrieves a scaling policy by ID.
// GET /api/v1/scaling/policies/:id
func (h *ScalingHandler) GetPolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	policy, err := h.service.GetPolicy(c.Request.Context(), id)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ScalingPolicyResponse{Policy: policy})
}

// ListPolicies lists all scaling policies.
// GET /api/v1/scaling/policies
func (h *ScalingHandler) ListPolicies(c *gin.Context) {
	enabledOnly := c.Query("enabled") == "true"

	response, err := h.service.ListPolicies(c.Request.Context(), enabledOnly)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePolicy updates a scaling policy.
// PUT /api/v1/scaling/policies/:id
func (h *ScalingHandler) UpdatePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	var req models.UpdateScalingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	policy, err := h.service.UpdatePolicy(c.Request.Context(), id, &req)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ScalingPolicyResponse{Policy: policy})
}

// DeletePolicy deletes a scaling policy.
// DELETE /api/v1/scaling/policies/:id
func (h *ScalingHandler) DeletePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	if err := h.service.DeletePolicy(c.Request.Context(), id); err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// EnablePolicy enables a scaling policy.
// POST /api/v1/scaling/policies/:id/enable
func (h *ScalingHandler) EnablePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	if err := h.service.EnablePolicy(c.Request.Context(), id); err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "policy enabled"})
}

// DisablePolicy disables a scaling policy.
// POST /api/v1/scaling/policies/:id/disable
func (h *ScalingHandler) DisablePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	if err := h.service.DisablePolicy(c.Request.Context(), id); err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "policy disabled"})
}

// EvaluatePolicy evaluates a scaling policy and optionally executes scaling.
// POST /api/v1/scaling/policies/:id/evaluate
func (h *ScalingHandler) EvaluatePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	var req models.EvaluatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to dry-run if no body provided
		req.DryRun = true
	}

	response, err := h.service.EvaluatePolicy(c.Request.Context(), id, req.DryRun)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPolicyState retrieves the current scaling state for a policy.
// GET /api/v1/scaling/policies/:id/state
func (h *ScalingHandler) GetPolicyState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	response, err := h.service.GetPolicyState(c.Request.Context(), id)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListHistory lists scaling history for all policies.
// GET /api/v1/scaling/history
func (h *ScalingHandler) ListHistory(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	response, err := h.service.ListHistory(c.Request.Context(), nil, limit)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPolicyHistory retrieves scaling history for a specific policy.
// GET /api/v1/scaling/policies/:id/history
func (h *ScalingHandler) GetPolicyHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	response, err := h.service.ListHistory(c.Request.Context(), &id, limit)
	if err != nil {
		respondWithScalingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// respondWithScalingServiceError handles service errors and returns appropriate HTTP responses.
func respondWithScalingServiceError(c *gin.Context, err error) {
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
		models.RespondWithError(c, newConflictError(
			c.Request.URL.Path,
			conflictErr.Message,
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an internal error occurred",
		))
	}
}

// Compile-time check to ensure ScalingHandler implements the required methods
var _ interface {
	Register(rg *gin.RouterGroup)
} = (*ScalingHandler)(nil)
