// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/cdc/health"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	healthManager *health.Manager
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(healthManager *health.Manager) *HealthHandler {
	return &HealthHandler{
		healthManager: healthManager,
	}
}

// GetHealth returns the overall health status.
// GET /health
func (h *HealthHandler) GetHealth(c *gin.Context) {
	if h.healthManager == nil {
		// No health manager configured, return basic healthy response
		c.JSON(http.StatusOK, models.HealthResponse{
			Status:    string(health.StatusHealthy),
			Timestamp: time.Now(),
		})
		return
	}

	status := h.healthManager.GetOverallStatus(c.Request.Context())

	// Convert health.OverallStatus to models.HealthResponse
	response := models.HealthResponse{
		Status:     string(status.Status),
		Components: make(map[string]models.ComponentHealth),
		Timestamp:  status.Timestamp,
	}

	for name, result := range status.Components {
		response.Components[name] = models.ComponentHealth{
			Name:       result.Name,
			Status:     string(result.Status),
			Message:    result.Message,
			DurationMs: result.Duration.Milliseconds(),
			LastCheck:  result.LastCheck,
			Error:      result.Error,
		}
	}

	// Set status code based on health
	statusCode := http.StatusOK
	if status.Status == health.StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// GetLiveness returns the liveness status.
// GET /health/live
func (h *HealthHandler) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, models.LivenessResponse{
		Status:    "alive",
		Timestamp: time.Now(),
	})
}

// GetReadiness returns the readiness status.
// GET /health/ready
func (h *HealthHandler) GetReadiness(c *gin.Context) {
	if h.healthManager == nil {
		// No health manager configured, assume ready
		c.JSON(http.StatusOK, models.ReadinessResponse{
			Status:    "ready",
			Timestamp: time.Now(),
		})
		return
	}

	if h.healthManager.IsReady(c.Request.Context()) {
		c.JSON(http.StatusOK, models.ReadinessResponse{
			Status:    "ready",
			Timestamp: time.Now(),
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, models.ReadinessResponse{
			Status:    "not_ready",
			Timestamp: time.Now(),
		})
	}
}
