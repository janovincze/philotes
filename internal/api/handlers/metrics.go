// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// MetricsHandler handles metrics-related HTTP requests.
type MetricsHandler struct {
	service *services.MetricsService
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(service *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{service: service}
}

// GetPipelineMetrics returns current metrics for a pipeline.
// GET /api/v1/pipelines/:id/metrics
func (h *MetricsHandler) GetPipelineMetrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	metrics, err := h.service.GetPipelineMetrics(c.Request.Context(), id)
	if err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.PipelineMetricsResponse{Metrics: metrics})
}

// GetPipelineMetricsHistory returns historical metrics for a pipeline.
// GET /api/v1/pipelines/:id/metrics/history?range=1h
func (h *MetricsHandler) GetPipelineMetricsHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	// Get time range from query parameter, default to 1h
	rangeStr := c.DefaultQuery("range", "1h")

	history, err := h.service.GetPipelineMetricsHistory(c.Request.Context(), id, rangeStr)
	if err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.MetricsHistoryResponse{History: history})
}

// respondWithServiceError converts service errors to HTTP responses.
func (h *MetricsHandler) respondWithServiceError(c *gin.Context, err error) {
	var validationErr *services.ValidationError
	var notFoundErr *services.NotFoundError

	switch {
	case errors.As(err, &validationErr):
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			validationErr.Errors,
		))
	case errors.As(err, &notFoundErr):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			notFoundErr.Error(),
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an internal error occurred",
		))
	}
}
