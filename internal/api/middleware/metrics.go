// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/metrics"
)

// Metrics returns a middleware that records Prometheus metrics for HTTP requests.
// It tracks request count, latency, request size, and response size.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get the path template (e.g., "/api/v1/sources/:id" instead of "/api/v1/sources/123")
		// This prevents cardinality explosion from unique IDs
		path := c.FullPath()
		if path == "" {
			// Use a constant for unmatched routes (404s) to prevent cardinality explosion
			path = "/not_found"
		}

		method := c.Request.Method

		// Record request size
		if c.Request.ContentLength > 0 {
			metrics.APIRequestSize.WithLabelValues(path, method).Observe(float64(c.Request.ContentLength))
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		status := strconv.Itoa(c.Writer.Status())

		// Record metrics
		metrics.APIRequestsTotal.WithLabelValues(path, method, status).Inc()
		metrics.APIRequestDuration.WithLabelValues(path, method).Observe(duration)

		// Record response size
		responseSize := c.Writer.Size()
		if responseSize > 0 {
			metrics.APIResponseSize.WithLabelValues(path, method).Observe(float64(responseSize))
		}
	}
}
