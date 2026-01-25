// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// PipelineHandler handles pipeline-related HTTP requests.
type PipelineHandler struct {
	service *services.PipelineService
}

// NewPipelineHandler creates a new PipelineHandler.
func NewPipelineHandler(service *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{service: service}
}

// Create creates a new pipeline.
// POST /api/v1/pipelines
func (h *PipelineHandler) Create(c *gin.Context) {
	var req models.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	pipeline, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.PipelineResponse{Pipeline: pipeline})
}

// List lists all pipelines.
// GET /api/v1/pipelines
func (h *PipelineHandler) List(c *gin.Context) {
	pipelines, err := h.service.List(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.PipelineListResponse{
		Pipelines:  pipelines,
		TotalCount: len(pipelines),
	})
}

// Get retrieves a pipeline by ID.
// GET /api/v1/pipelines/:id
func (h *PipelineHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	pipeline, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.PipelineResponse{Pipeline: pipeline})
}

// Update updates a pipeline.
// PUT /api/v1/pipelines/:id
func (h *PipelineHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	var req models.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	pipeline, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.PipelineResponse{Pipeline: pipeline})
}

// Delete deletes a pipeline.
// DELETE /api/v1/pipelines/:id
func (h *PipelineHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Start starts a pipeline.
// POST /api/v1/pipelines/:id/start
func (h *PipelineHandler) Start(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	if err := h.service.Start(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pipeline started"})
}

// Stop stops a pipeline.
// POST /api/v1/pipelines/:id/stop
func (h *PipelineHandler) Stop(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	if err := h.service.Stop(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pipeline stopped"})
}

// GetStatus gets the status of a pipeline.
// GET /api/v1/pipelines/:id/status
func (h *PipelineHandler) GetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	status, err := h.service.GetStatus(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// AddTableMapping adds a table mapping to a pipeline.
// POST /api/v1/pipelines/:id/tables
func (h *PipelineHandler) AddTableMapping(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	var req models.AddTableMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	mapping, err := h.service.AddTableMapping(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapping)
}

// RemoveTableMapping removes a table mapping from a pipeline.
// DELETE /api/v1/pipelines/:id/tables/:mappingId
func (h *PipelineHandler) RemoveTableMapping(c *gin.Context) {
	pipelineID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid pipeline ID format",
		))
		return
	}

	mappingID, err := uuid.Parse(c.Param("mappingId"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid mapping ID format",
		))
		return
	}

	if err := h.service.RemoveTableMapping(c.Request.Context(), pipelineID, mappingID); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
