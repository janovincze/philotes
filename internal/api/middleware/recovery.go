package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
)

// Recovery returns a middleware that recovers from panics and returns a structured error.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := c.GetString("request_id")

				// Log the panic with stack trace
				attrs := []any{
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"stack", string(debug.Stack()),
				}

				if requestID != "" {
					attrs = append(attrs, "request_id", requestID)
				}

				logger.Error("panic recovered", attrs...)

				// Return a structured error response
				models.RespondWithError(c, models.NewInternalError(
					c.Request.URL.Path,
					"An unexpected error occurred",
				))

				c.Abort()
			}
		}()

		c.Next()
	}
}
