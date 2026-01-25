package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
)

// StubHandler returns not implemented errors for future endpoints.
type StubHandler struct{}

// NewStubHandler creates a new StubHandler.
func NewStubHandler() *StubHandler {
	return &StubHandler{}
}

// ListSources returns a not implemented error.
// GET /api/v1/sources
func (h *StubHandler) ListSources(c *gin.Context) {
	models.RespondWithError(c, models.NewNotImplementedError(c.Request.URL.Path))
}

// ListPipelines returns a not implemented error.
// GET /api/v1/pipelines
func (h *StubHandler) ListPipelines(c *gin.Context) {
	models.RespondWithError(c, models.NewNotImplementedError(c.Request.URL.Path))
}

// ListDestinations returns a not implemented error.
// GET /api/v1/destinations
func (h *StubHandler) ListDestinations(c *gin.Context) {
	models.RespondWithError(c, models.NewNotImplementedError(c.Request.URL.Path))
}
