package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the context key for request ID.
	RequestIDKey = "request_id"
)

// RequestID returns a middleware that ensures each request has a unique ID.
// If the request already has an X-Request-ID header, it is used.
// Otherwise, a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing request ID
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new ID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context for logging and other middleware
		c.Set(RequestIDKey, requestID)

		// Set response header
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
