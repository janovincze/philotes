// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// OnboardingHandler handles onboarding wizard HTTP requests.
type OnboardingHandler struct {
	service *services.OnboardingService
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(service *services.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{service: service}
}

// Register adds all onboarding routes to the router.
func (h *OnboardingHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/onboarding/cluster/health", h.GetClusterHealth)
	rg.GET("/onboarding/progress", h.GetProgress)
	rg.POST("/onboarding/progress", h.SaveProgress)
	rg.POST("/onboarding/data/verify", h.VerifyDataFlow)
	rg.GET("/onboarding/admin/exists", h.CheckAdminExists)
}

// GetClusterHealth returns extended cluster health for onboarding.
// GET /api/v1/onboarding/cluster/health
func (h *OnboardingHandler) GetClusterHealth(c *gin.Context) {
	response := h.service.GetClusterHealth(c.Request.Context())
	c.JSON(http.StatusOK, response)
}

// GetProgress retrieves onboarding progress.
// GET /api/v1/onboarding/progress
func (h *OnboardingHandler) GetProgress(c *gin.Context) {
	// Get session ID from query params
	sessionID := c.Query("session_id")

	// Get user ID from auth context if available
	var userID *uuid.UUID
	if authCtx, exists := c.Get("auth_context"); exists {
		if auth, ok := authCtx.(*models.AuthContext); ok && auth != nil && auth.User != nil {
			userID = &auth.User.ID
		}
	}

	// Try to get existing progress
	progress, err := h.service.GetProgress(c.Request.Context(), userID, sessionID)
	if err != nil {
		if errors.Is(err, services.ErrOnboardingNotFound) {
			// No existing progress, create a new one
			progress, err = h.service.CreateProgress(c.Request.Context(), userID, sessionID)
			if err != nil {
				respondWithServiceError(c, err)
				return
			}
		} else {
			respondWithServiceError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, models.OnboardingProgressResponse{Progress: progress})
}

// SaveProgress saves onboarding progress.
// POST /api/v1/onboarding/progress
func (h *OnboardingHandler) SaveProgress(c *gin.Context) {
	var req models.SaveOnboardingProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Get progress ID from session or user
	var userID *uuid.UUID
	if authCtx, exists := c.Get("auth_context"); exists {
		if auth, ok := authCtx.(*models.AuthContext); ok && auth != nil && auth.User != nil {
			userID = &auth.User.ID
		}
	}

	// Get or create progress
	progress, err := h.service.GetOrCreateProgress(c.Request.Context(), userID, req.SessionID)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	// Update progress
	updated, err := h.service.SaveProgress(c.Request.Context(), progress.ID, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	// Check if completed (all required steps done)
	if isOnboardingComplete(updated.CompletedSteps) {
		updated, err = h.service.CompleteOnboarding(c.Request.Context(), progress.ID)
		if err != nil {
			respondWithServiceError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, models.OnboardingProgressResponse{Progress: updated})
}

// VerifyDataFlow verifies that data is flowing to Iceberg.
// POST /api/v1/onboarding/data/verify
func (h *OnboardingHandler) VerifyDataFlow(c *gin.Context) {
	var req models.DataVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	response, err := h.service.VerifyDataFlow(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// CheckAdminExists checks if an admin user exists.
// GET /api/v1/onboarding/admin/exists
func (h *OnboardingHandler) CheckAdminExists(c *gin.Context) {
	exists, err := h.service.CheckAdminExists(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.AdminExistsResponse{Exists: exists})
}

// isOnboardingComplete checks if all required steps are completed.
func isOnboardingComplete(completedSteps []int) bool {
	// Required steps: 1 (health), 2 (admin), 4 (source), 5 (pipeline), 6 (verify)
	// Optional steps: 3 (SSO), 7 (alerts)
	required := map[int]bool{
		1: true,
		2: true,
		4: true,
		5: true,
		6: true,
	}

	for _, step := range completedSteps {
		delete(required, step)
	}

	return len(required) == 0
}
