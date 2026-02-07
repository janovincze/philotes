// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// QueryDataSourceHandler handles query data source HTTP requests.
type QueryDataSourceHandler struct {
	service *services.QueryDataSourceService
}

// NewQueryDataSourceHandler creates a new QueryDataSourceHandler.
func NewQueryDataSourceHandler(service *services.QueryDataSourceService) *QueryDataSourceHandler {
	return &QueryDataSourceHandler{service: service}
}

// Create creates a new query data source.
// POST /api/v1/query/datasources
func (h *QueryDataSourceHandler) Create(c *gin.Context) {
	var req models.CreateQueryDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	ds, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.QueryDataSourceResponse{DataSource: ds})
}

// List lists all query data sources.
// GET /api/v1/query/datasources
func (h *QueryDataSourceHandler) List(c *gin.Context) {
	dataSources, err := h.service.List(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.QueryDataSourceListResponse{
		DataSources: dataSources,
		TotalCount:  len(dataSources),
	})
}

// Get retrieves a query data source by ID.
// GET /api/v1/query/datasources/:id
func (h *QueryDataSourceHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid data source ID format",
		))
		return
	}

	ds, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.QueryDataSourceResponse{DataSource: ds})
}

// Update updates a query data source.
// PUT /api/v1/query/datasources/:id
func (h *QueryDataSourceHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid data source ID format",
		))
		return
	}

	var req models.UpdateQueryDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	ds, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.QueryDataSourceResponse{DataSource: ds})
}

// Delete deletes a query data source.
// DELETE /api/v1/query/datasources/:id
func (h *QueryDataSourceHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid data source ID format",
		))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConnection tests the connection to a query data source.
// POST /api/v1/query/datasources/:id/test
func (h *QueryDataSourceHandler) TestConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid data source ID format",
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
