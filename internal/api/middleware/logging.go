// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a middleware that logs requests using slog.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID if set
		requestID := c.GetString("request_id")

		// Build log attributes
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		// Log based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("request completed", attrs...)
		case status >= 400:
			logger.Warn("request completed", attrs...)
		default:
			logger.Info("request completed", attrs...)
		}
	}
}
