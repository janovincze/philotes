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

// OIDCHandler handles OIDC authentication HTTP requests.
type OIDCHandler struct {
	oidcService *services.OIDCService
}

// NewOIDCHandler creates a new OIDCHandler.
func NewOIDCHandler(oidcService *services.OIDCService) *OIDCHandler {
	return &OIDCHandler{
		oidcService: oidcService,
	}
}

// --- Public Endpoints ---

// ListEnabledProviders lists all enabled OIDC providers.
// GET /api/v1/auth/oidc/providers
func (h *OIDCHandler) ListEnabledProviders(c *gin.Context) {
	response, err := h.oidcService.ListEnabledProviders(c.Request.Context())
	if err != nil {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to list providers",
		))
		return
	}

	c.JSON(http.StatusOK, response)
}

// Authorize initiates the OIDC authorization flow.
// POST /api/v1/auth/oidc/:provider/authorize
func (h *OIDCHandler) Authorize(c *gin.Context) {
	providerName := c.Param("provider")
	if providerName == "" {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"provider is required",
		))
		return
	}

	var req models.OIDCAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			fieldErrors,
		))
		return
	}

	response, err := h.oidcService.StartAuthorization(c.Request.Context(), providerName, req.RedirectURI)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOIDCProviderNotFound):
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"provider not found",
			))
		case errors.Is(err, services.ErrOIDCProviderDisabled):
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"provider is disabled",
			))
		default:
			models.RespondWithError(c, models.NewInternalError(
				c.Request.URL.Path,
				"failed to start authorization",
			))
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

// Callback handles the OIDC callback from the identity provider.
// POST /api/v1/auth/oidc/callback
func (h *OIDCHandler) Callback(c *gin.Context) {
	var req models.OIDCCallbackRequest

	// Try to bind from query params first, then from JSON body
	if err := c.ShouldBindQuery(&req); err != nil || (req.Code == "" && req.State == "") {
		if err := c.ShouldBindJSON(&req); err != nil {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"invalid request: code and state are required",
			))
			return
		}
	}

	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			fieldErrors,
		))
		return
	}

	// Check for error from IdP
	if req.Error != "" {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"authentication failed: "+req.Error,
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	response, err := h.oidcService.HandleCallback(c.Request.Context(), req.Code, req.State, ipAddress, userAgent)
	if err != nil {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"callback processing failed",
		))
		return
	}

	if !response.Success {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			response.Error,
		))
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- Admin Endpoints ---

// ListProviders lists all OIDC providers (admin).
// GET /api/v1/settings/oidc/providers
func (h *OIDCHandler) ListProviders(c *gin.Context) {
	response, err := h.oidcService.ListProviders(c.Request.Context())
	if err != nil {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to list providers",
		))
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateProvider creates a new OIDC provider.
// POST /api/v1/settings/oidc/providers
func (h *OIDCHandler) CreateProvider(c *gin.Context) {
	var req models.CreateOIDCProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	provider, err := h.oidcService.CreateProvider(c.Request.Context(), &req)
	if err != nil {
		var validationErr *services.ValidationError
		if errors.As(err, &validationErr) {
			models.RespondWithError(c, models.NewValidationError(
				c.Request.URL.Path,
				validationErr.Errors,
			))
			return
		}
		var conflictErr *services.ConflictError
		if errors.As(err, &conflictErr) {
			models.RespondWithError(c, models.NewConflictError(
				c.Request.URL.Path,
				conflictErr.Error(),
			))
			return
		}
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to create provider",
		))
		return
	}

	c.JSON(http.StatusCreated, models.OIDCProviderResponse{Provider: provider.ToSummary()})
}

// GetProvider retrieves an OIDC provider by ID.
// GET /api/v1/settings/oidc/providers/:id
func (h *OIDCHandler) GetProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid provider ID",
		))
		return
	}

	provider, err := h.oidcService.GetProvider(c.Request.Context(), id)
	if err != nil {
		var notFoundErr *services.NotFoundError
		if errors.As(err, &notFoundErr) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"provider not found",
			))
			return
		}
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to get provider",
		))
		return
	}

	c.JSON(http.StatusOK, models.OIDCProviderResponse{Provider: provider.ToSummary()})
}

// UpdateProvider updates an OIDC provider.
// PUT /api/v1/settings/oidc/providers/:id
func (h *OIDCHandler) UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid provider ID",
		))
		return
	}

	var req models.UpdateOIDCProviderRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+bindErr.Error(),
		))
		return
	}

	provider, err := h.oidcService.UpdateProvider(c.Request.Context(), id, &req)
	if err != nil {
		var notFoundErr *services.NotFoundError
		if errors.As(err, &notFoundErr) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"provider not found",
			))
			return
		}
		var validationErr *services.ValidationError
		if errors.As(err, &validationErr) {
			models.RespondWithError(c, models.NewValidationError(
				c.Request.URL.Path,
				validationErr.Errors,
			))
			return
		}
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to update provider",
		))
		return
	}

	c.JSON(http.StatusOK, models.OIDCProviderResponse{Provider: provider.ToSummary()})
}

// DeleteProvider deletes an OIDC provider.
// DELETE /api/v1/settings/oidc/providers/:id
func (h *OIDCHandler) DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid provider ID",
		))
		return
	}

	err = h.oidcService.DeleteProvider(c.Request.Context(), id)
	if err != nil {
		var notFoundErr *services.NotFoundError
		if errors.As(err, &notFoundErr) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"provider not found",
			))
			return
		}
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to delete provider",
		))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// TestProvider tests an OIDC provider configuration.
// POST /api/v1/settings/oidc/providers/:id/test
func (h *OIDCHandler) TestProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid provider ID",
		))
		return
	}

	err = h.oidcService.TestProvider(c.Request.Context(), id)
	if err != nil {
		var notFoundErr *services.NotFoundError
		if errors.As(err, &notFoundErr) {
			models.RespondWithError(c, models.NewNotFoundError(
				c.Request.URL.Path,
				"provider not found",
			))
			return
		}
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "provider connection successful"})
}

// Register registers OIDC routes.
func (h *OIDCHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	// Public OIDC auth endpoints
	auth := rg.Group("/auth/oidc")
	auth.GET("/providers", h.ListEnabledProviders)
	auth.POST("/:provider/authorize", h.Authorize)
	auth.POST("/callback", h.Callback)
	auth.GET("/callback", h.Callback) // Also support GET for IdP redirects

	// Admin settings endpoints (protected)
	settings := rg.Group("/settings/oidc")
	settings.Use(requireAuth)
	settings.GET("/providers", h.ListProviders)
	settings.POST("/providers", h.CreateProvider)
	settings.GET("/providers/:id", h.GetProvider)
	settings.PUT("/providers/:id", h.UpdateProvider)
	settings.DELETE("/providers/:id", h.DeleteProvider)
	settings.POST("/providers/:id/test", h.TestProvider)
}
