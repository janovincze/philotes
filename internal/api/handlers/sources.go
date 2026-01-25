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

// SourceHandler handles source-related HTTP requests.
type SourceHandler struct {
	service *services.SourceService
}

// NewSourceHandler creates a new SourceHandler.
func NewSourceHandler(service *services.SourceService) *SourceHandler {
	return &SourceHandler{service: service}
}

// Create creates a new source.
// POST /api/v1/sources
func (h *SourceHandler) Create(c *gin.Context) {
	var req models.CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	source, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.SourceResponse{Source: source})
}

// List lists all sources.
// GET /api/v1/sources
func (h *SourceHandler) List(c *gin.Context) {
	sources, err := h.service.List(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SourceListResponse{
		Sources:    sources,
		TotalCount: len(sources),
	})
}

// Get retrieves a source by ID.
// GET /api/v1/sources/:id
func (h *SourceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid source ID format",
		))
		return
	}

	source, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SourceResponse{Source: source})
}

// Update updates a source.
// PUT /api/v1/sources/:id
func (h *SourceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid source ID format",
		))
		return
	}

	var req models.UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	source, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SourceResponse{Source: source})
}

// Delete deletes a source.
// DELETE /api/v1/sources/:id
func (h *SourceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid source ID format",
		))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConnection tests the connection to a source.
// POST /api/v1/sources/:id/test
func (h *SourceHandler) TestConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid source ID format",
		))
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// DiscoverTables discovers tables in a source database.
// GET /api/v1/sources/:id/tables
func (h *SourceHandler) DiscoverTables(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid source ID format",
		))
		return
	}

	schema := c.DefaultQuery("schema", "public")

	result, err := h.service.DiscoverTables(c.Request.Context(), id, schema)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// respondWithServiceError converts service errors to HTTP responses.
func respondWithServiceError(c *gin.Context, err error) {
	var validationErr *services.ValidationError
	var notFoundErr *services.NotFoundError
	var conflictErr *services.ConflictError

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
	case errors.As(err, &conflictErr):
		models.RespondWithError(c, newConflictError(
			c.Request.URL.Path,
			conflictErr.Message,
		))
	default:
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an unexpected error occurred",
		))
	}
}

// newConflictError creates a conflict error response.
func newConflictError(instance, detail string) *models.ProblemDetails {
	return &models.ProblemDetails{
		Type:     "https://philotes.io/errors/conflict",
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   detail,
		Instance: instance,
	}
}
