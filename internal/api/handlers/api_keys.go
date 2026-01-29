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

// APIKeyHandler handles API key-related HTTP requests.
type APIKeyHandler struct {
	apiKeyService *services.APIKeyService
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(apiKeyService *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService: apiKeyService}
}

// Create creates a new API key.
// POST /api/v1/api-keys
func (h *APIKeyHandler) Create(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	response, err := h.apiKeyService.Create(
		c.Request.Context(),
		authContext.User.ID,
		&req,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// List lists all API keys for the current user.
// GET /api/v1/api-keys
func (h *APIKeyHandler) List(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	keys, err := h.apiKeyService.List(c.Request.Context(), authContext.User.ID)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.APIKeyListResponse{
		APIKeys:    keys,
		TotalCount: len(keys),
	})
}

// Get retrieves an API key by ID.
// GET /api/v1/api-keys/:id
func (h *APIKeyHandler) Get(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
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
			"invalid API key ID format",
		))
		return
	}

	apiKey, err := h.apiKeyService.Get(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	// Check ownership
	if apiKey.UserID != authContext.User.ID && authContext.User.Role != models.RoleAdmin {
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"API key not found",
		))
		return
	}

	c.JSON(http.StatusOK, models.APIKeyResponse{APIKey: apiKey})
}

// Delete deletes an API key.
// DELETE /api/v1/api-keys/:id
func (h *APIKeyHandler) Delete(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
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
			"invalid API key ID format",
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	if err := h.apiKeyService.Delete(c.Request.Context(), id, authContext.User.ID, authContext.User.Role, ipAddress, userAgent); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Revoke revokes (deactivates) an API key.
// POST /api/v1/api-keys/:id/revoke
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
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
			"invalid API key ID format",
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	if err := h.apiKeyService.Revoke(c.Request.Context(), id, authContext.User.ID, authContext.User.Role, ipAddress, userAgent); err != nil {
		h.respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Register registers routes for the API key handler.
func (h *APIKeyHandler) Register(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	apiKeys := rg.Group("/api-keys")
	apiKeys.Use(authMiddleware)
	{
		apiKeys.POST("", h.Create)
		apiKeys.GET("", h.List)
		apiKeys.GET("/:id", h.Get)
		apiKeys.DELETE("/:id", h.Delete)
		apiKeys.POST("/:id/revoke", h.Revoke)
	}
}

// respondWithError converts service errors to HTTP responses.
func (h *APIKeyHandler) respondWithError(c *gin.Context, err error) {
	var validationErr *services.ValidationError
	var notFoundErr *services.NotFoundError

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
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an unexpected error occurred",
		))
	}
}
