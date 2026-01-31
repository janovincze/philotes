// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// WakeHandler handles wake-related HTTP requests.
type WakeHandler struct {
	service *services.WakeService
}

// NewWakeHandler creates a new WakeHandler.
func NewWakeHandler(service *services.WakeService) *WakeHandler {
	return &WakeHandler{service: service}
}

// Register adds all wake routes to the router.
func (h *WakeHandler) Register(rg *gin.RouterGroup) {
	// Wake endpoints
	rg.POST("/scaling/policies/:id/wake", h.WakePolicy)
	rg.POST("/scaling/wake", h.WakeAll)

	// Idle state endpoints
	rg.GET("/scaling/policies/:id/idle", h.GetIdleState)
	rg.GET("/scaling/scaled-to-zero", h.ListScaledToZero)

	// Cost savings endpoints
	rg.GET("/scaling/policies/:id/savings", h.GetCostSavings)
	rg.GET("/scaling/savings/summary", h.GetSavingsSummary)
}

// WakePolicy wakes a specific policy from scaled-to-zero state.
// POST /api/v1/scaling/policies/:id/wake
func (h *WakeHandler) WakePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	var req models.WakePolicyRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		// Default values if no body
		req = models.WakePolicyRequest{}
	}

	response, err := h.service.WakePolicy(c.Request.Context(), id, req.GetReason())
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// WakeAll wakes all scaled-to-zero policies or specific policies.
// POST /api/v1/scaling/wake
func (h *WakeHandler) WakeAll(c *gin.Context) {
	var req models.WakeAllRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		// Default values if no body
		req = models.WakeAllRequest{}
	}

	response, err := h.service.WakeAll(c.Request.Context(), req.PolicyIDs, req.GetReason())
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetIdleState retrieves the idle state for a policy.
// GET /api/v1/scaling/policies/:id/idle
func (h *WakeHandler) GetIdleState(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	response, err := h.service.GetIdleState(c.Request.Context(), id)
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListScaledToZero returns all policies currently scaled to zero.
// GET /api/v1/scaling/scaled-to-zero
func (h *WakeHandler) ListScaledToZero(c *gin.Context) {
	response, err := h.service.ListScaledToZero(c.Request.Context())
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetCostSavings retrieves cost savings for a policy.
// GET /api/v1/scaling/policies/:id/savings
func (h *WakeHandler) GetCostSavings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, parseErr := strconv.Atoi(daysStr); parseErr == nil && d > 0 {
			days = d
		}
	}

	response, err := h.service.GetCostSavings(c.Request.Context(), id, days)
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSavingsSummary retrieves overall cost savings summary.
// GET /api/v1/scaling/savings/summary
func (h *WakeHandler) GetSavingsSummary(c *gin.Context) {
	response, err := h.service.GetSavingsSummary(c.Request.Context())
	if err != nil {
		respondWithWakeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// respondWithWakeServiceError handles wake service errors.
func respondWithWakeServiceError(c *gin.Context, err error) {
	// Reuse the scaling service error handler
	respondWithScalingServiceError(c, err)
}

// Compile-time check to ensure WakeHandler implements the required methods.
var _ interface {
	Register(rg *gin.RouterGroup)
} = (*WakeHandler)(nil)
