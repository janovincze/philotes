// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService   *services.AuthService
	apiKeyService *services.APIKeyService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *services.AuthService, apiKeyService *services.APIKeyService) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		apiKeyService: apiKeyService,
	}
}

// Login authenticates a user and returns a JWT token.
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	response, err := h.authService.Login(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			models.RespondWithError(c, models.NewUnauthorizedError(
				c.Request.URL.Path,
				"Invalid email or password",
			))
		case errors.Is(err, services.ErrUserInactive):
			models.RespondWithError(c, models.NewForbiddenError(
				c.Request.URL.Path,
				"User account is inactive",
			))
		default:
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
				"an unexpected error occurred",
			))
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMe returns the current authenticated user.
// GET /api/v1/auth/me
func (h *AuthHandler) GetMe(c *gin.Context) {
	authContext := middleware.GetAuthContext(c)
	if authContext == nil {
		models.RespondWithError(c, models.NewUnauthorizedError(
			c.Request.URL.Path,
			"Authentication required",
		))
		return
	}

	c.JSON(http.StatusOK, models.UserResponse{User: authContext.User})
}

// RegisterAdmin creates the first admin user during onboarding.
// POST /api/v1/auth/register
func (h *AuthHandler) RegisterAdmin(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	ipAddress := middleware.GetClientIP(c)
	userAgent := middleware.GetUserAgent(c)

	// Register the first admin
	response, err := h.authService.RegisterFirstAdmin(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAdminAlreadyExists):
			models.RespondWithError(c, models.NewConflictError(
				c.Request.URL.Path,
				"Admin user already exists. Use login instead.",
			))
		default:
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
				"an unexpected error occurred",
			))
		}
		return
	}

	// If API key generation is requested (default is true)
	generateAPIKey := req.GenerateAPIKey == nil || *req.GenerateAPIKey
	if generateAPIKey && h.apiKeyService != nil {
		apiKeyReq := &models.CreateAPIKeyRequest{
			Name: "Initial Admin API Key",
		}
		apiKeyResp, err := h.apiKeyService.Create(c.Request.Context(), response.User.ID, apiKeyReq, ipAddress, userAgent)
		if err == nil {
			response.APIKey = &apiKeyResp.Key
		}
		// If API key creation fails, we still return the user (just without the API key)
	}

	c.JSON(http.StatusCreated, response)
}

// Register registers routes for the auth handler.
func (h *AuthHandler) Register(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := rg.Group("/auth")
	// Public routes
	auth.POST("/login", h.Login)
	auth.POST("/register", h.RegisterAdmin)
	// Protected routes
	auth.GET("/me", authMiddleware, h.GetMe)
}
