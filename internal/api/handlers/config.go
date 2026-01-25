package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/config"
)

// ConfigHandler handles configuration endpoints.
type ConfigHandler struct {
	cfg *config.Config
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(cfg *config.Config) *ConfigHandler {
	return &ConfigHandler{cfg: cfg}
}

// GetConfig returns a safe subset of the system configuration.
// GET /api/v1/config
//
// SECURITY WARNING: This endpoint exposes configuration to unauthenticated users.
// Only include non-sensitive configuration values here. Never expose:
// - Database passwords or connection strings
// - API keys or secrets
// - Storage credentials
// - Internal hostnames or IPs that could aid attackers
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	// Return only safe, non-sensitive configuration
	response := models.ConfigResponse{
		Environment: h.cfg.Environment,
		API: models.APIConfig{
			ListenAddr: h.cfg.API.ListenAddr,
			BaseURL:    h.cfg.API.BaseURL,
		},
		CDC: models.CDCConfig{
			BufferSize:    h.cfg.CDC.BufferSize,
			BatchSize:     h.cfg.CDC.BatchSize,
			FlushInterval: h.cfg.CDC.FlushInterval.String(),
		},
		Metrics: models.MetricConfig{
			Enabled:    h.cfg.Metrics.Enabled,
			ListenAddr: h.cfg.Metrics.ListenAddr,
		},
	}

	c.JSON(http.StatusOK, response)
}
