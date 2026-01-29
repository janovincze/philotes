// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
	"github.com/janovincze/philotes/internal/installer"
)

// InstallerHandler handles installer-related HTTP requests.
type InstallerHandler struct {
	service *services.InstallerService
	logHub  *installer.LogHub
}

// NewInstallerHandler creates a new InstallerHandler.
func NewInstallerHandler(service *services.InstallerService, logHub *installer.LogHub) *InstallerHandler {
	return &InstallerHandler{
		service: service,
		logHub:  logHub,
	}
}

// ListProviders lists all supported cloud providers.
// GET /api/v1/installer/providers
func (h *InstallerHandler) ListProviders(c *gin.Context) {
	providers := h.service.GetProviders(c.Request.Context())

	c.JSON(http.StatusOK, models.ProviderListResponse{
		Providers: providers,
	})
}

// GetProvider retrieves a provider by ID.
// GET /api/v1/installer/providers/:id
func (h *InstallerHandler) GetProvider(c *gin.Context) {
	providerID := c.Param("id")

	provider, err := h.service.GetProvider(c.Request.Context(), providerID)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ProviderResponse{Provider: provider})
}

// GetCostEstimate calculates the cost estimate for a deployment configuration.
// GET /api/v1/installer/providers/:id/estimate?size=small|medium|large
func (h *InstallerHandler) GetCostEstimate(c *gin.Context) {
	providerID := c.Param("id")
	sizeStr := c.DefaultQuery("size", "small")

	size := models.DeploymentSize(sizeStr)
	if size != models.DeploymentSizeSmall &&
		size != models.DeploymentSizeMedium &&
		size != models.DeploymentSizeLarge {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"size must be one of: small, medium, large",
		))
		return
	}

	estimate, err := h.service.GetCostEstimate(c.Request.Context(), providerID, size)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.CostEstimateResponse{Estimate: estimate})
}

// CreateDeployment creates a new deployment.
// POST /api/v1/installer/deployments
func (h *InstallerHandler) CreateDeployment(c *gin.Context) {
	var req models.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Get user ID from context if authenticated
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uuid.UUID); ok {
			userID = &uid
		}
	}

	deployment, err := h.service.CreateDeployment(c.Request.Context(), &req, userID)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.DeploymentResponse{Deployment: deployment})
}

// ListDeployments lists all deployments.
// GET /api/v1/installer/deployments
func (h *InstallerHandler) ListDeployments(c *gin.Context) {
	// Get user ID from context if authenticated (optional filtering)
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uuid.UUID); ok {
			userID = &uid
		}
	}

	deployments, err := h.service.ListDeployments(c.Request.Context(), userID)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.DeploymentListResponse{
		Deployments: deployments,
		TotalCount:  len(deployments),
	})
}

// GetDeployment retrieves a deployment by ID.
// GET /api/v1/installer/deployments/:id
func (h *InstallerHandler) GetDeployment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid deployment ID format",
		))
		return
	}

	deployment, err := h.service.GetDeployment(c.Request.Context(), id)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.DeploymentResponse{Deployment: deployment})
}

// CancelDeployment cancels a deployment.
// POST /api/v1/installer/deployments/:id/cancel
func (h *InstallerHandler) CancelDeployment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid deployment ID format",
		))
		return
	}

	if err := h.service.CancelDeployment(c.Request.Context(), id); err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deployment cancelled"})
}

// DeleteDeployment deletes a deployment.
// DELETE /api/v1/installer/deployments/:id
func (h *InstallerHandler) DeleteDeployment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid deployment ID format",
		))
		return
	}

	if err := h.service.DeleteDeployment(c.Request.Context(), id); err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDeploymentLogs retrieves logs for a deployment.
// GET /api/v1/installer/deployments/:id/logs
func (h *InstallerHandler) GetDeploymentLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid deployment ID format",
		))
		return
	}

	// Parse optional limit parameter
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		// Try to parse as positive integer
		if n, parseErr := parsePositiveInt(limitStr); parseErr == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	logs, err := h.service.GetDeploymentLogs(c.Request.Context(), id, limit)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.DeploymentLogsResponse{
		Logs:       logs,
		TotalCount: len(logs),
	})
}

// respondWithInstallerError handles service errors for installer endpoints.
func respondWithInstallerError(c *gin.Context, err error) {
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
			"an unexpected error occurred",
		))
	}
}

// parsePositiveInt parses a string as a positive integer.
func parsePositiveInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid integer")
		}
		n = n*10 + int(c-'0')
		if n > 1000000 {
			return 0, errors.New("integer too large")
		}
	}
	return n, nil
}

// StreamDeploymentLogs streams deployment logs via WebSocket.
// GET /api/v1/installer/deployments/:id/logs/stream (WebSocket)
func (h *InstallerHandler) StreamDeploymentLogs(c *gin.Context) {
	if h.logHub == nil {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"WebSocket support not configured",
		))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid deployment ID format",
		))
		return
	}

	// Verify deployment exists
	_, err = h.service.GetDeployment(c.Request.Context(), id)
	if err != nil {
		respondWithInstallerError(c, err)
		return
	}

	// Upgrade to WebSocket
	if err := h.logHub.HandleWebSocket(c.Writer, c.Request, id); err != nil {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to upgrade to WebSocket: "+err.Error(),
		))
		return
	}
}
