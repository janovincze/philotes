// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/services"
)

// AlertHandler handles alerting-related HTTP requests.
type AlertHandler struct {
	service *services.AlertService
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(service *services.AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

// Register adds all alert routes to the router.
func (h *AlertHandler) Register(rg *gin.RouterGroup) {
	// Alert Rules
	rg.POST("/alerts/rules", h.CreateRule)
	rg.GET("/alerts/rules", h.ListRules)
	rg.GET("/alerts/rules/:id", h.GetRule)
	rg.PUT("/alerts/rules/:id", h.UpdateRule)
	rg.DELETE("/alerts/rules/:id", h.DeleteRule)

	// Alert Instances
	rg.GET("/alerts", h.ListAlerts)
	rg.GET("/alerts/summary", h.GetSummary)
	rg.GET("/alerts/:id", h.GetAlert)
	rg.POST("/alerts/:id/acknowledge", h.AcknowledgeAlert)
	rg.GET("/alerts/:id/history", h.GetAlertHistory)

	// Silences
	rg.POST("/alerts/silences", h.CreateSilence)
	rg.GET("/alerts/silences", h.ListSilences)
	rg.GET("/alerts/silences/:id", h.GetSilence)
	rg.DELETE("/alerts/silences/:id", h.DeleteSilence)

	// Notification Channels
	rg.POST("/notifications/channels", h.CreateChannel)
	rg.GET("/notifications/channels", h.ListChannels)
	rg.GET("/notifications/channels/:id", h.GetChannel)
	rg.PUT("/notifications/channels/:id", h.UpdateChannel)
	rg.DELETE("/notifications/channels/:id", h.DeleteChannel)
	rg.POST("/notifications/channels/:id/test", h.TestChannel)

	// Alert Routes
	rg.POST("/alerts/routes", h.CreateRoute)
	rg.GET("/alerts/routes", h.ListRoutes)
	rg.GET("/alerts/routes/:id", h.GetRoute)
	rg.PUT("/alerts/routes/:id", h.UpdateRoute)
	rg.DELETE("/alerts/routes/:id", h.DeleteRoute)
}

// Alert Rules

// CreateRule creates a new alert rule.
// POST /api/v1/alerts/rules
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var req models.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	rule, err := h.service.CreateRule(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.AlertRuleResponse{Rule: rule})
}

// GetRule retrieves an alert rule by ID.
// GET /api/v1/alerts/rules/:id
func (h *AlertHandler) GetRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid rule ID format",
		))
		return
	}

	rule, err := h.service.GetRule(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.AlertRuleResponse{Rule: rule})
}

// ListRules lists all alert rules.
// GET /api/v1/alerts/rules
func (h *AlertHandler) ListRules(c *gin.Context) {
	limit, offset := parsePagination(c)

	response, err := h.service.ListRules(c.Request.Context(), limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateRule updates an alert rule.
// PUT /api/v1/alerts/rules/:id
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid rule ID format",
		))
		return
	}

	var req models.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	rule, err := h.service.UpdateRule(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.AlertRuleResponse{Rule: rule})
}

// DeleteRule deletes an alert rule.
// DELETE /api/v1/alerts/rules/:id
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid rule ID format",
		))
		return
	}

	if err := h.service.DeleteRule(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Alert Instances

// GetAlert retrieves an alert instance by ID.
// GET /api/v1/alerts/:id
func (h *AlertHandler) GetAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid alert ID format",
		))
		return
	}

	alert, err := h.service.GetAlert(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.AlertInstanceResponse{Alert: alert})
}

// ListAlerts lists alert instances.
// GET /api/v1/alerts
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	limit, offset := parsePagination(c)
	status := c.Query("status")
	severity := c.Query("severity")

	response, err := h.service.ListAlerts(c.Request.Context(), status, severity, limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// AcknowledgeAlert acknowledges an alert instance.
// POST /api/v1/alerts/:id/acknowledge
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid alert ID format",
		))
		return
	}

	var req models.AcknowledgeAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	if err := h.service.AcknowledgeAlert(c.Request.Context(), id, &req); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alert acknowledged"})
}

// GetAlertHistory retrieves history for an alert instance.
// GET /api/v1/alerts/:id/history
func (h *AlertHandler) GetAlertHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid alert ID format",
		))
		return
	}

	limit, offset := parsePagination(c)

	response, err := h.service.GetAlertHistory(c.Request.Context(), id, limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Silences

// CreateSilence creates a new alert silence.
// POST /api/v1/alerts/silences
func (h *AlertHandler) CreateSilence(c *gin.Context) {
	var req models.CreateSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	silence, err := h.service.CreateSilence(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.SilenceResponse{Silence: silence})
}

// GetSilence retrieves a silence by ID.
// GET /api/v1/alerts/silences/:id
func (h *AlertHandler) GetSilence(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid silence ID format",
		))
		return
	}

	silence, err := h.service.GetSilence(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.SilenceResponse{Silence: silence})
}

// ListSilences lists silences.
// GET /api/v1/alerts/silences
func (h *AlertHandler) ListSilences(c *gin.Context) {
	limit, offset := parsePagination(c)
	active := c.Query("active") == "true"

	response, err := h.service.ListSilences(c.Request.Context(), active, limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSilence deletes a silence.
// DELETE /api/v1/alerts/silences/:id
func (h *AlertHandler) DeleteSilence(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid silence ID format",
		))
		return
	}

	if err := h.service.DeleteSilence(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Notification Channels

// CreateChannel creates a new notification channel.
// POST /api/v1/notifications/channels
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	var req models.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	channel, err := h.service.CreateChannel(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.ChannelResponse{Channel: channel})
}

// GetChannel retrieves a notification channel by ID.
// GET /api/v1/notifications/channels/:id
func (h *AlertHandler) GetChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid channel ID format",
		))
		return
	}

	channel, err := h.service.GetChannel(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ChannelResponse{Channel: channel})
}

// ListChannels lists notification channels.
// GET /api/v1/notifications/channels
func (h *AlertHandler) ListChannels(c *gin.Context) {
	limit, offset := parsePagination(c)

	response, err := h.service.ListChannels(c.Request.Context(), limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateChannel updates a notification channel.
// PUT /api/v1/notifications/channels/:id
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid channel ID format",
		))
		return
	}

	var req models.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	channel, err := h.service.UpdateChannel(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.ChannelResponse{Channel: channel})
}

// DeleteChannel deletes a notification channel.
// DELETE /api/v1/notifications/channels/:id
func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid channel ID format",
		))
		return
	}

	if err := h.service.DeleteChannel(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// TestChannel tests a notification channel.
// POST /api/v1/notifications/channels/:id/test
func (h *AlertHandler) TestChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid channel ID format",
		))
		return
	}

	result, err := h.service.TestChannel(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Summary

// GetSummary retrieves alert statistics summary.
// GET /api/v1/alerts/summary
func (h *AlertHandler) GetSummary(c *gin.Context) {
	summary, err := h.service.GetSummary(c.Request.Context())
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}

// Alert Routes

// CreateRoute creates a new alert route.
// POST /api/v1/alerts/routes
func (h *AlertHandler) CreateRoute(c *gin.Context) {
	var req models.CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	route, err := h.service.CreateRoute(c.Request.Context(), &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, models.RouteResponse{Route: route})
}

// GetRoute retrieves an alert route by ID.
// GET /api/v1/alerts/routes/:id
func (h *AlertHandler) GetRoute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid route ID format",
		))
		return
	}

	route, err := h.service.GetRoute(c.Request.Context(), id)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.RouteResponse{Route: route})
}

// ListRoutes lists alert routes.
// GET /api/v1/alerts/routes
func (h *AlertHandler) ListRoutes(c *gin.Context) {
	limit, offset := parsePagination(c)

	var ruleID *uuid.UUID
	if ruleIDStr := c.Query("rule_id"); ruleIDStr != "" {
		id, err := uuid.Parse(ruleIDStr)
		if err != nil {
			models.RespondWithError(c, models.NewBadRequestError(
				c.Request.URL.Path,
				"invalid rule_id format",
			))
			return
		}
		ruleID = &id
	}

	response, err := h.service.ListRoutes(c.Request.Context(), ruleID, limit, offset)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateRoute updates an alert route.
// PUT /api/v1/alerts/routes/:id
func (h *AlertHandler) UpdateRoute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid route ID format",
		))
		return
	}

	var req models.UpdateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid request body: "+err.Error(),
		))
		return
	}

	route, err := h.service.UpdateRoute(c.Request.Context(), id, &req)
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.RouteResponse{Route: route})
}

// DeleteRoute deletes an alert route.
// DELETE /api/v1/alerts/routes/:id
func (h *AlertHandler) DeleteRoute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		models.RespondWithError(c, models.NewBadRequestError(
			c.Request.URL.Path,
			"invalid route ID format",
		))
		return
	}

	if err := h.service.DeleteRoute(c.Request.Context(), id); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// parsePagination extracts pagination parameters from the query string.
func parsePagination(c *gin.Context) (limit, offset int) {
	limit = 100 // default limit
	offset = 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000 // max limit
			}
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}
