package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS middleware configuration.
type CORSConfig struct {
	// AllowedOrigins is a list of origins allowed to make cross-origin requests.
	// Use "*" to allow all origins.
	AllowedOrigins []string

	// AllowCredentials indicates whether the request can include credentials.
	AllowCredentials bool

	// MaxAge indicates how long the results of a preflight request can be cached.
	MaxAge time.Duration
}

// DefaultCORSConfig returns a CORSConfig with sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// CORS returns a middleware that handles CORS.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}
