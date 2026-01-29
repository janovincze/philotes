// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// Context keys for auth data.
const (
	AuthContextKey = "auth_context"
)

// AuthConfig holds authentication middleware configuration.
type AuthConfig struct {
	// Enabled enables authentication
	Enabled bool

	// AuthService is the auth service for JWT validation
	AuthService *services.AuthService

	// APIKeyService is the API key service for API key validation
	APIKeyService *services.APIKeyService

	// APIKeyPrefix is the prefix used for API keys (e.g., "pk_")
	APIKeyPrefix string
}

// Authenticate returns a middleware that extracts authentication credentials
// and sets the auth context. It does not reject unauthenticated requests.
func Authenticate(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			// Auth disabled, continue without setting auth context
			c.Next()
			return
		}

		// Try to extract credentials from headers
		authContext := extractAuthContext(c, cfg)
		if authContext != nil {
			c.Set(AuthContextKey, authContext)
		}

		c.Next()
	}
}

// RequireAuth returns a middleware that requires authentication.
// Must be used after Authenticate middleware.
func RequireAuth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			// Auth disabled, allow all requests
			c.Next()
			return
		}

		authContext := GetAuthContext(c)
		if authContext == nil {
			models.RespondWithError(c, models.NewUnauthorizedError(
				c.Request.URL.Path,
				"Authentication required",
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission returns a middleware that requires a specific permission.
// Must be used after Authenticate and RequireAuth middleware.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authContext := GetAuthContext(c)
		if authContext == nil {
			models.RespondWithError(c, models.NewUnauthorizedError(
				c.Request.URL.Path,
				"Authentication required",
			))
			c.Abort()
			return
		}

		if !authContext.HasPermission(permission) {
			models.RespondWithError(c, models.NewForbiddenError(
				c.Request.URL.Path,
				"Insufficient permissions",
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAuthContext retrieves the auth context from a Gin context.
func GetAuthContext(c *gin.Context) *models.AuthContext {
	value, exists := c.Get(AuthContextKey)
	if !exists {
		return nil
	}
	authContext, ok := value.(*models.AuthContext)
	if !ok {
		return nil
	}
	return authContext
}

// extractAuthContext attempts to extract auth credentials from the request.
func extractAuthContext(c *gin.Context, cfg AuthConfig) *models.AuthContext {
	// Check for API key in X-API-Key header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return validateAPIKey(c, cfg, apiKey)
	}

	// Check for Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil
	}

	// Parse Authorization header
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil
	}

	scheme := strings.ToLower(parts[0])
	credential := parts[1]

	switch scheme {
	case "bearer":
		// Could be JWT or API key
		// API keys start with configured prefix (e.g., "pk_")
		prefix := cfg.APIKeyPrefix
		if prefix == "" {
			prefix = "pk_" // Default prefix
		}
		if strings.HasPrefix(credential, prefix) {
			return validateAPIKey(c, cfg, credential)
		}
		// Otherwise, treat as JWT
		return validateJWT(c, cfg, credential)
	default:
		return nil
	}
}

// validateAPIKey validates an API key and returns the auth context.
func validateAPIKey(c *gin.Context, cfg AuthConfig, key string) *models.AuthContext {
	if cfg.APIKeyService == nil {
		return nil
	}

	user, apiKey, err := cfg.APIKeyService.Validate(c.Request.Context(), key)
	if err != nil {
		return nil
	}

	return &models.AuthContext{
		User:        user,
		APIKey:      apiKey,
		Permissions: apiKey.Permissions,
		IsAPIKey:    true,
	}
}

// validateJWT validates a JWT token and returns the auth context.
func validateJWT(c *gin.Context, cfg AuthConfig, token string) *models.AuthContext {
	if cfg.AuthService == nil {
		return nil
	}

	claims, err := cfg.AuthService.ValidateJWT(token)
	if err != nil {
		return nil
	}

	// Get user from claims
	user, err := cfg.AuthService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil
	}

	// Check if user is still active
	if !user.IsActive {
		return nil
	}

	return &models.AuthContext{
		User:        user,
		Permissions: claims.Permissions,
		IsAPIKey:    false,
	}
}

// GetClientIP returns the client IP address from the request.
func GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetUserAgent returns the user agent from the request.
func GetUserAgent(c *gin.Context) string {
	return c.Request.UserAgent()
}
