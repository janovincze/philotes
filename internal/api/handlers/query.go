// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/services"
)

// QueryHandler handles query layer API endpoints.
type QueryHandler struct {
	service *services.QueryService
	logger  *slog.Logger
}

// NewQueryHandler creates a new QueryHandler.
func NewQueryHandler(service *services.QueryService, logger *slog.Logger) *QueryHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &QueryHandler{
		service: service,
		logger:  logger.With("component", "query-handler"),
	}
}

// RegisterRoutes registers the query layer routes.
func (h *QueryHandler) RegisterRoutes(r *gin.RouterGroup) {
	query := r.Group("/query")
	query.GET("/status", h.GetStatus)
	query.GET("/health", h.GetHealth)
	query.GET("/catalogs", h.ListCatalogs)
	query.GET("/catalogs/:catalog/schemas", h.ListSchemas)
	query.GET("/catalogs/:catalog/schemas/:schema/tables", h.ListTables)
	query.GET("/catalogs/:catalog/schemas/:schema/tables/:table", h.GetTableInfo)
}

// GetStatus godoc
// @Summary Get query layer status
// @Description Returns the current status of the Trino query layer
// @Tags query
// @Accept json
// @Produce json
// @Success 200 {object} models.QueryLayerStatus
// @Failure 500 {object} map[string]string
// @Router /query/status [get]
func (h *QueryHandler) GetStatus(c *gin.Context) {
	status, err := h.service.GetStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get query layer status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetHealth godoc
// @Summary Get query layer health
// @Description Returns health status of the Trino query layer
// @Tags query
// @Accept json
// @Produce json
// @Success 200 {object} models.QueryHealthResponse
// @Router /query/health [get]
func (h *QueryHandler) GetHealth(c *gin.Context) {
	health, err := h.service.GetHealth(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to check query layer health", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// ListCatalogs godoc
// @Summary List Trino catalogs
// @Description Returns all available Trino catalogs
// @Tags query
// @Accept json
// @Produce json
// @Success 200 {object} models.CatalogListResponse
// @Failure 500 {object} map[string]string
// @Router /query/catalogs [get]
func (h *QueryHandler) ListCatalogs(c *gin.Context) {
	catalogs, err := h.service.ListCatalogs(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list catalogs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, catalogs)
}

// ListSchemas godoc
// @Summary List schemas in a catalog
// @Description Returns all schemas in the specified catalog
// @Tags query
// @Accept json
// @Produce json
// @Param catalog path string true "Catalog name"
// @Success 200 {object} models.SchemaListResponse
// @Failure 500 {object} map[string]string
// @Router /query/catalogs/{catalog}/schemas [get]
func (h *QueryHandler) ListSchemas(c *gin.Context) {
	catalog := c.Param("catalog")

	schemas, err := h.service.ListSchemas(c.Request.Context(), catalog)
	if err != nil {
		h.logger.Error("failed to list schemas", "catalog", catalog, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schemas)
}

// ListTables godoc
// @Summary List tables in a schema
// @Description Returns all tables in the specified schema
// @Tags query
// @Accept json
// @Produce json
// @Param catalog path string true "Catalog name"
// @Param schema path string true "Schema name"
// @Success 200 {object} models.TableListResponse
// @Failure 500 {object} map[string]string
// @Router /query/catalogs/{catalog}/schemas/{schema}/tables [get]
func (h *QueryHandler) ListTables(c *gin.Context) {
	catalog := c.Param("catalog")
	schema := c.Param("schema")

	tables, err := h.service.ListTables(c.Request.Context(), catalog, schema)
	if err != nil {
		h.logger.Error("failed to list tables", "catalog", catalog, "schema", schema, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tables)
}

// GetTableInfo godoc
// @Summary Get table information
// @Description Returns detailed information about a table
// @Tags query
// @Accept json
// @Produce json
// @Param catalog path string true "Catalog name"
// @Param schema path string true "Schema name"
// @Param table path string true "Table name"
// @Success 200 {object} models.TableInfoResponse
// @Failure 500 {object} map[string]string
// @Router /query/catalogs/{catalog}/schemas/{schema}/tables/{table} [get]
func (h *QueryHandler) GetTableInfo(c *gin.Context) {
	catalog := c.Param("catalog")
	schema := c.Param("schema")
	table := c.Param("table")

	info, err := h.service.GetTableInfo(c.Request.Context(), catalog, schema, table)
	if err != nil {
		h.logger.Error("failed to get table info",
			"catalog", catalog,
			"schema", schema,
			"table", table,
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}
