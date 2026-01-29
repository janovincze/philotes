// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/api/services"
)

// OAuthHandler handles OAuth-related HTTP requests.
type OAuthHandler struct {
	service *services.OAuthService
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(service *services.OAuthService) *OAuthHandler {
	return &OAuthHandler{
		service: service,
	}
}

// Authorize starts the OAuth flow for a provider.
// POST /api/v1/installer/oauth/:provider/authorize
func (h *OAuthHandler) Authorize(c *gin.Context) {
	providerID := c.Param("provider")

	var req models.OAuthAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		models.RespondWithError(c, models.NewValidationError(c.Request.URL.Path, errs))
		return
	}

	// Get user ID from context if authenticated
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uuid.UUID); ok {
			userID = &uid
		}
	}

	resp, err := h.service.StartAuthorization(
		c.Request.Context(),
		providerID,
		req.RedirectURI,
		userID,
		req.SessionID,
	)
	if err != nil {
		respondWithOAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Callback handles the OAuth callback from the provider.
// GET /api/v1/installer/oauth/:provider/callback
func (h *OAuthHandler) Callback(c *gin.Context) {
	providerID := c.Param("provider")

	// Check for OAuth error from provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		if errDesc == "" {
			errDesc = errParam
		}
		// Redirect to frontend with error
		state := c.Query("state")
		redirectWithError(c, state, errDesc)
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"missing code or state parameter",
		))
		return
	}

	resp, err := h.service.HandleCallback(c.Request.Context(), providerID, code, state)
	if err != nil {
		respondWithOAuthError(c, err)
		return
	}

	// Redirect to frontend
	if resp.RedirectURI != "" {
		redirectURL := resp.RedirectURI
		if resp.Success {
			redirectURL += "?success=true&provider=" + url.QueryEscape(resp.Provider) + "&credential_id=" + url.QueryEscape(resp.CredentialID.String())
		} else {
			redirectURL += "?error=" + url.QueryEscape(resp.Error)
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StoreCredential stores manual API credentials.
// POST /api/v1/installer/credentials/:provider
func (h *OAuthHandler) StoreCredential(c *gin.Context) {
	providerID := c.Param("provider")

	var req models.StoreCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Override provider from URL
	req.Provider = providerID

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		models.RespondWithError(c, models.NewValidationError(c.Request.URL.Path, errs))
		return
	}

	// Get user ID from context if authenticated
	var userID *uuid.UUID
	if id, exists := c.Get("user_id"); exists {
		if uid, ok := id.(uuid.UUID); ok {
			userID = &uid
		}
	}

	resp, err := h.service.StoreManualCredential(c.Request.Context(), &req, userID)
	if err != nil {
		respondWithOAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// ListCredentials lists all credentials for the current user.
// GET /api/v1/installer/credentials
func (h *OAuthHandler) ListCredentials(c *gin.Context) {
	// Get user ID from context (required for this endpoint)
	userID, exists := c.Get("user_id")
	if !exists {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"authentication required",
		))
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"invalid user ID format",
		))
		return
	}

	resp, err := h.service.ListCredentials(c.Request.Context(), uid)
	if err != nil {
		respondWithOAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteCredential deletes a stored credential.
// DELETE /api/v1/installer/credentials/:provider
func (h *OAuthHandler) DeleteCredential(c *gin.Context) {
	providerID := c.Param("provider")

	// Get user ID from context (required for this endpoint)
	userID, exists := c.Get("user_id")
	if !exists {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"authentication required",
		))
		return
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"invalid user ID format",
		))
		return
	}

	if err := h.service.DeleteCredentialByProvider(c.Request.Context(), uid, providerID); err != nil {
		respondWithOAuthError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetOAuthProviders returns information about available OAuth providers.
// GET /api/v1/installer/oauth/providers
func (h *OAuthHandler) GetOAuthProviders(c *gin.Context) {
	resp := h.service.GetOAuthProviders()
	c.JSON(http.StatusOK, resp)
}

// respondWithOAuthError handles service errors for OAuth endpoints.
func respondWithOAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrOAuthStateNotFound):
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid or expired OAuth state",
		))
	case errors.Is(err, repositories.ErrOAuthStateExpired):
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"OAuth state has expired",
		))
	case errors.Is(err, repositories.ErrCredentialNotFound):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			"credential not found",
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"OAuth operation failed: "+err.Error(),
		))
	}
}

// redirectWithError redirects to the frontend with an error.
func redirectWithError(c *gin.Context, state, errMsg string) {
	// Try to get redirect URI from state (would need to look up in DB)
	// For now, just return an error response
	models.RespondWithError(c, models.NewBadRequestError(
		c.Request.URL.Path,
		"OAuth error: "+errMsg,
	))
}
