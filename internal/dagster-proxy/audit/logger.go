// Package audit provides audit logging for the Dagster proxy.
package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

// Audit action constants for Dagster operations.
const (
	ActionDagsterQuery          = "dagster_query"
	ActionDagsterMutation       = "dagster_mutation"
	ActionDagsterLaunchRun      = "dagster_launch_run"
	ActionDagsterTerminateRun   = "dagster_terminate_run"
	ActionDagsterStartSchedule  = "dagster_start_schedule"
	ActionDagsterStopSchedule   = "dagster_stop_schedule"
	ActionDagsterStartSensor    = "dagster_start_sensor"
	ActionDagsterStopSensor     = "dagster_stop_sensor"
	ActionDagsterMaterialize    = "dagster_materialize_asset"
	ActionDagsterAccessDenied   = "dagster_access_denied"
	ActionDagsterWebSocketOpen  = "dagster_websocket_open"
	ActionDagsterWebSocketClose = "dagster_websocket_close"
)

// Action represents an action to be logged.
type Action struct {
	AuthCtx      *auth.Context
	Operation    string
	ResourceType string
	ResourceName string
	Action       string
	Allowed      bool
	Reason       string
	IPAddress    string
	UserAgent    string
}

// Logger provides audit logging for Dagster operations.
type Logger struct {
	repo   *repositories.AuditRepository
	logger *slog.Logger
}

// NewLogger creates a new audit logger.
func NewLogger(repo *repositories.AuditRepository, logger *slog.Logger) *Logger {
	return &Logger{
		repo:   repo,
		logger: logger,
	}
}

// LogAction logs a Dagster action.
func (l *Logger) LogAction(ctx context.Context, action Action) {
	if l.repo == nil {
		// Log to structured logger if no repository
		l.logToLogger(action)
		return
	}

	// Determine audit action
	auditAction := l.mapActionToAuditAction(action)

	// Build details
	details := map[string]interface{}{
		"operation":     action.Operation,
		"resource_type": action.ResourceType,
		"resource_name": action.ResourceName,
		"action":        action.Action,
		"allowed":       action.Allowed,
	}

	if action.Reason != "" {
		details["reason"] = action.Reason
	}

	// Create audit log entry
	log := &models.AuditLog{
		Action:       auditAction,
		ResourceType: "dagster:" + action.ResourceType,
		IPAddress:    action.IPAddress,
		UserAgent:    action.UserAgent,
		Details:      details,
	}

	// Set user/API key info
	if action.AuthCtx != nil {
		log.UserID = &action.AuthCtx.UserID
		if action.AuthCtx.APIKeyID != nil {
			log.APIKeyID = action.AuthCtx.APIKeyID
		}
	}

	// Set resource ID if we have a valid resource name
	if action.ResourceName != "" && action.ResourceName != "*" {
		// Use resource name as a deterministic UUID for tracking
		resourceID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(action.ResourceType+"/"+action.ResourceName))
		log.ResourceID = &resourceID
	}

	// Save to database
	if err := l.repo.Create(ctx, log); err != nil {
		l.logger.Error("failed to create audit log",
			"error", err,
			"action", auditAction,
		)
	}

	// Also log to structured logger
	l.logToLogger(action)
}

// logToLogger logs the action to the structured logger.
func (l *Logger) logToLogger(action Action) {
	level := slog.LevelInfo
	if !action.Allowed {
		level = slog.LevelWarn
	}

	l.logger.Log(context.Background(), level, "dagster action",
		"operation", action.Operation,
		"resource_type", action.ResourceType,
		"resource_name", action.ResourceName,
		"action", action.Action,
		"allowed", action.Allowed,
		"reason", action.Reason,
		"user_id", l.getUserID(action.AuthCtx),
		"ip_address", action.IPAddress,
	)
}

// mapActionToAuditAction maps an action to an audit action constant.
func (l *Logger) mapActionToAuditAction(action Action) string {
	if !action.Allowed {
		return ActionDagsterAccessDenied
	}

	switch action.Action {
	case "execute":
		return ActionDagsterLaunchRun
	case "terminate":
		return ActionDagsterTerminateRun
	case "control":
		switch action.ResourceType {
		case "schedules":
			if action.Operation == "startSchedule" {
				return ActionDagsterStartSchedule
			}
			return ActionDagsterStopSchedule
		case "sensors":
			if action.Operation == "startSensor" {
				return ActionDagsterStartSensor
			}
			return ActionDagsterStopSensor
		}
	case "materialize":
		return ActionDagsterMaterialize
	case "view":
		return ActionDagsterQuery
	}

	return ActionDagsterMutation
}

// getUserID extracts user ID string from auth context.
func (l *Logger) getUserID(authCtx *auth.Context) string {
	if authCtx == nil {
		return ""
	}
	return authCtx.UserID.String()
}

// LogWebSocketOpen logs a WebSocket connection opening.
func (l *Logger) LogWebSocketOpen(ctx context.Context, authCtx *auth.Context, ipAddress, userAgent string) {
	l.LogAction(ctx, Action{
		AuthCtx:      authCtx,
		Operation:    "websocket_open",
		ResourceType: "subscription",
		ResourceName: "*",
		Action:       "view",
		Allowed:      true,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
}

// LogWebSocketClose logs a WebSocket connection closing.
func (l *Logger) LogWebSocketClose(ctx context.Context, authCtx *auth.Context, ipAddress, userAgent string) {
	l.LogAction(ctx, Action{
		AuthCtx:      authCtx,
		Operation:    "websocket_close",
		ResourceType: "subscription",
		ResourceName: "*",
		Action:       "view",
		Allowed:      true,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
}

// LogAccessDenied logs an access denied event.
func (l *Logger) LogAccessDenied(ctx context.Context, authCtx *auth.Context, operation, resourceType, resourceName, reason, ipAddress, userAgent string) {
	l.LogAction(ctx, Action{
		AuthCtx:      authCtx,
		Operation:    operation,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Action:       "denied",
		Allowed:      false,
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
}
