// Package handlers provides HTTP request handlers for the API.
package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// QueryScalingHandler handles query scaling API requests.
type QueryScalingHandler struct {
	service *services.QueryScalingService
	logger  *slog.Logger
}

// NewQueryScalingHandler creates a new QueryScalingHandler.
func NewQueryScalingHandler(service *services.QueryScalingService, logger *slog.Logger) *QueryScalingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &QueryScalingHandler{
		service: service,
		logger:  logger.With("handler", "query-scaling"),
	}
}

// RegisterRoutes registers the query scaling routes.
func (h *QueryScalingHandler) RegisterRoutes(r *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	queryScaling := r.Group("/query-scaling")
	queryScaling.Use(requireAuth)
	queryScaling.GET("/policies", h.ListPolicies)
	queryScaling.POST("/policies", h.CreatePolicy)
	queryScaling.GET("/policies/:id", h.GetPolicy)
	queryScaling.PUT("/policies/:id", h.UpdatePolicy)
	queryScaling.DELETE("/policies/:id", h.DeletePolicy)
	queryScaling.GET("/metrics", h.GetMetrics)
	queryScaling.GET("/history", h.GetHistory)
}

// ListPolicies lists all query scaling policies.
func (h *QueryScalingHandler) ListPolicies(c *gin.Context) {
	var queryEngine *models.QueryEngine
	if qe := c.Query("query_engine"); qe != "" {
		engine := models.QueryEngine(qe)
		queryEngine = &engine
	}

	resp, err := h.service.ListPolicies(c.Request.Context(), queryEngine)
	if err != nil {
		h.logger.Error("failed to list policies", "error", err)
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to list policies",
		))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreatePolicy creates a new query scaling policy.
func (h *QueryScalingHandler) CreatePolicy(c *gin.Context) {
	var req models.CreateQueryScalingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	// Validate query engine
	if req.QueryEngine != models.QueryEngineTrino && req.QueryEngine != models.QueryEngineRisingWave {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid query_engine: must be 'trino' or 'risingwave'",
		))
		return
	}

	policy, err := h.service.CreatePolicy(c.Request.Context(), &req)
	if err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetPolicy retrieves a query scaling policy by ID.
func (h *QueryScalingHandler) GetPolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	policy, err := h.service.GetPolicy(c.Request.Context(), id)
	if err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// UpdatePolicy updates a query scaling policy.
func (h *QueryScalingHandler) UpdatePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	var req models.UpdateQueryScalingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	policy, err := h.service.UpdatePolicy(c.Request.Context(), id, &req)
	if err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeletePolicy deletes a query scaling policy.
func (h *QueryScalingHandler) DeletePolicy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid policy ID format",
		))
		return
	}

	if err := h.service.DeletePolicy(c.Request.Context(), id); err != nil {
		h.respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMetrics retrieves current query engine metrics.
func (h *QueryScalingHandler) GetMetrics(c *gin.Context) {
	resp, err := h.service.GetMetrics(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get metrics", "error", err)
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to get metrics",
		))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetHistory retrieves query scaling history.
func (h *QueryScalingHandler) GetHistory(c *gin.Context) {
	var policyID *uuid.UUID
	if pidStr := c.Query("policy_id"); pidStr != "" {
		id, err := uuid.Parse(pidStr)
		if err != nil {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"invalid policy_id format",
			))
			return
		}
		policyID = &id
	}

	var queryEngine *models.QueryEngine
	if qe := c.Query("query_engine"); qe != "" {
		engine := models.QueryEngine(qe)
		queryEngine = &engine
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	resp, err := h.service.GetHistory(c.Request.Context(), policyID, queryEngine, limit)
	if err != nil {
		h.logger.Error("failed to get history", "error", err)
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"failed to get history",
		))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// respondWithServiceError converts service errors to HTTP responses.
func (h *QueryScalingHandler) respondWithServiceError(c *gin.Context, err error) {
	var notFoundErr *services.NotFoundError
	var validationErr *services.ValidationError
	var conflictErr *services.ConflictError

	switch {
	case errors.As(err, &notFoundErr):
		models.RespondWithError(c, models.NewNotFoundError(
			c.Request.URL.Path,
			notFoundErr.Error(),
		))
	case errors.As(err, &validationErr):
		models.RespondWithError(c, models.NewValidationError(
			c.Request.URL.Path,
			validationErr.Errors,
		))
	case errors.As(err, &conflictErr):
		models.RespondWithError(c, &models.ProblemDetails{
			Type:     "https://philotes.io/errors/conflict",
			Title:    "Conflict",
			Status:   http.StatusConflict,
			Detail:   conflictErr.Message,
			Instance: c.Request.URL.Path,
		})
	default:
		h.logger.Error("unexpected service error", "error", err)
		models.RespondWithError(c, models.NewInternalError(
			c.Request.URL.Path,
			"an unexpected error occurred",
		))
	}
}
