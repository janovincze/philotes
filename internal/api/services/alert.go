// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/alerting"
	"github.com/janovincze/philotes/internal/alerting/channels"
	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
)

// AlertService provides business logic for alerting operations.
type AlertService struct {
	repo   *repositories.AlertRepository
	logger *slog.Logger
}

// NewAlertService creates a new AlertService.
func NewAlertService(repo *repositories.AlertRepository, logger *slog.Logger) *AlertService {
	return &AlertService{
		repo:   repo,
		logger: logger.With("component", "alert-service"),
	}
}

// Alert Rules

// CreateRule creates a new alert rule.
func (s *AlertService) CreateRule(ctx context.Context, req *models.CreateAlertRuleRequest) (*alerting.AlertRule, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Create rule
	rule, err := s.repo.CreateRule(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertRuleNameExists) {
			return nil, &ConflictError{Message: "alert rule with this name already exists"}
		}
		s.logger.Error("failed to create alert rule", "error", err)
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}

	s.logger.Info("alert rule created", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

// GetRule retrieves an alert rule by ID.
func (s *AlertService) GetRule(ctx context.Context, id uuid.UUID) (*alerting.AlertRule, error) {
	rule, err := s.repo.GetRule(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertRuleNotFound) {
			return nil, &NotFoundError{Resource: "alert rule", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}
	return rule, nil
}

// ListRules retrieves all alert rules with pagination.
func (s *AlertService) ListRules(ctx context.Context, limit, offset int) (*models.AlertRuleListResponse, error) {
	// Get total count first
	allRules, err := s.repo.ListRulesPaginated(ctx, false, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to count alert rules: %w", err)
	}
	total := len(allRules)

	// Get paginated rules
	var rules []alerting.AlertRule
	if limit > 0 {
		rules, err = s.repo.ListRulesPaginated(ctx, false, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to list alert rules: %w", err)
		}
	} else {
		rules = allRules
	}

	if rules == nil {
		rules = []alerting.AlertRule{}
	}

	return &models.AlertRuleListResponse{
		Rules:      rules,
		TotalCount: total,
	}, nil
}

// UpdateRule updates an alert rule.
func (s *AlertService) UpdateRule(ctx context.Context, id uuid.UUID, req *models.UpdateAlertRuleRequest) (*alerting.AlertRule, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Update rule
	rule, err := s.repo.UpdateRule(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertRuleNotFound) {
			return nil, &NotFoundError{Resource: "alert rule", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrAlertRuleNameExists) {
			return nil, &ConflictError{Message: "alert rule with this name already exists"}
		}
		return nil, fmt.Errorf("failed to update alert rule: %w", err)
	}

	s.logger.Info("alert rule updated", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

// DeleteRule deletes an alert rule.
func (s *AlertService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteRule(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertRuleNotFound) {
			return &NotFoundError{Resource: "alert rule", ID: id.String()}
		}
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	s.logger.Info("alert rule deleted", "id", id)
	return nil
}

// Alert Instances

// GetAlert retrieves an alert instance by ID.
func (s *AlertService) GetAlert(ctx context.Context, id uuid.UUID) (*alerting.AlertInstance, error) {
	alert, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertInstanceNotFound) {
			return nil, &NotFoundError{Resource: "alert", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	return alert, nil
}

// ListAlerts retrieves alert instances with optional filtering.
func (s *AlertService) ListAlerts(ctx context.Context, status string, severity string, limit, offset int) (*models.AlertInstanceListResponse, error) {
	var statusFilter *alerting.AlertStatus
	if status != "" {
		s := alerting.AlertStatus(status)
		if !s.IsValid() {
			return nil, &ValidationError{Errors: []models.FieldError{
				{Field: "status", Message: "status must be one of: firing, resolved"},
			}}
		}
		statusFilter = &s
	}

	alerts, err := s.repo.ListInstances(ctx, statusFilter, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}

	if alerts == nil {
		alerts = []alerting.AlertInstance{}
	}

	// Filter by severity if provided
	if severity != "" {
		sev := alerting.AlertSeverity(severity)
		if !sev.IsValid() {
			return nil, &ValidationError{Errors: []models.FieldError{
				{Field: "severity", Message: "severity must be one of: info, warning, critical"},
			}}
		}

		// Load rules to check severity
		filteredAlerts := make([]alerting.AlertInstance, 0)
		for _, alert := range alerts {
			rule, err := s.repo.GetRule(ctx, alert.RuleID)
			if err == nil && rule.Severity == sev {
				alert.Rule = rule
				filteredAlerts = append(filteredAlerts, alert)
			}
		}
		alerts = filteredAlerts
	}

	// Apply pagination
	total := len(alerts)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return &models.AlertInstanceListResponse{
		Alerts:     alerts[offset:end],
		TotalCount: total,
	}, nil
}

// AcknowledgeAlert acknowledges an alert instance.
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id uuid.UUID, req *models.AcknowledgeAlertRequest) error {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	// Check alert exists
	alert, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertInstanceNotFound) {
			return &NotFoundError{Resource: "alert", ID: id.String()}
		}
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Check if already acknowledged
	if alert.AcknowledgedAt != nil {
		return &ConflictError{Message: "alert has already been acknowledged"}
	}

	// Acknowledge the alert
	if err := s.repo.AcknowledgeInstance(ctx, id, req.AcknowledgedBy); err != nil {
		if errors.Is(err, repositories.ErrAlertInstanceNotFound) {
			return &NotFoundError{Resource: "alert", ID: id.String()}
		}
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	// Record acknowledgment in history
	rule, _ := s.repo.GetRule(ctx, alert.RuleID)
	history := &alerting.AlertHistory{
		AlertID:   id,
		RuleID:    alert.RuleID,
		EventType: alerting.EventAcknowledged,
		Message:   fmt.Sprintf("Alert acknowledged by %s", req.AcknowledgedBy),
		Metadata: map[string]any{
			"acknowledged_by": req.AcknowledgedBy,
		},
	}
	if req.Comment != "" {
		history.Metadata["comment"] = req.Comment
	}
	if _, err := s.repo.CreateHistory(ctx, history); err != nil {
		s.logger.Warn("failed to create acknowledgment history", "error", err)
	}

	ruleName := "unknown"
	if rule != nil {
		ruleName = rule.Name
	}
	s.logger.Info("alert acknowledged", "id", id, "rule", ruleName, "by", req.AcknowledgedBy)
	return nil
}

// GetAlertHistory retrieves history for an alert instance.
func (s *AlertService) GetAlertHistory(ctx context.Context, alertID uuid.UUID, limit, offset int) (*models.AlertHistoryResponse, error) {
	// Check alert exists
	_, err := s.repo.GetInstance(ctx, alertID)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertInstanceNotFound) {
			return nil, &NotFoundError{Resource: "alert", ID: alertID.String()}
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	// Get history
	history, err := s.repo.ListHistory(ctx, &alertID, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}

	if history == nil {
		history = []alerting.AlertHistory{}
	}

	// Apply pagination
	total := len(history)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return &models.AlertHistoryResponse{
		History:    history[offset:end],
		TotalCount: total,
	}, nil
}

// Silences

// CreateSilence creates a new alert silence.
func (s *AlertService) CreateSilence(ctx context.Context, req *models.CreateSilenceRequest) (*alerting.AlertSilence, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Create silence
	silence, err := s.repo.CreateSilence(ctx, req)
	if err != nil {
		s.logger.Error("failed to create silence", "error", err)
		return nil, fmt.Errorf("failed to create silence: %w", err)
	}

	s.logger.Info("silence created", "id", silence.ID, "created_by", silence.CreatedBy)
	return silence, nil
}

// GetSilence retrieves a silence by ID.
func (s *AlertService) GetSilence(ctx context.Context, id uuid.UUID) (*alerting.AlertSilence, error) {
	silence, err := s.repo.GetSilence(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSilenceNotFound) {
			return nil, &NotFoundError{Resource: "silence", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get silence: %w", err)
	}
	return silence, nil
}

// ListSilences retrieves silences with optional filtering.
func (s *AlertService) ListSilences(ctx context.Context, active bool, limit, offset int) (*models.SilenceListResponse, error) {
	silences, err := s.repo.ListSilences(ctx, active)
	if err != nil {
		return nil, fmt.Errorf("failed to list silences: %w", err)
	}

	if silences == nil {
		silences = []alerting.AlertSilence{}
	}

	// Apply pagination
	total := len(silences)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return &models.SilenceListResponse{
		Silences:   silences[offset:end],
		TotalCount: total,
	}, nil
}

// DeleteSilence deletes a silence.
func (s *AlertService) DeleteSilence(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteSilence(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrSilenceNotFound) {
			return &NotFoundError{Resource: "silence", ID: id.String()}
		}
		return fmt.Errorf("failed to delete silence: %w", err)
	}

	s.logger.Info("silence deleted", "id", id)
	return nil
}

// Notification Channels

// CreateChannel creates a new notification channel.
func (s *AlertService) CreateChannel(ctx context.Context, req *models.CreateChannelRequest) (*alerting.NotificationChannel, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Create channel
	channel, err := s.repo.CreateChannel(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNameExists) {
			return nil, &ConflictError{Message: "notification channel with this name already exists"}
		}
		s.logger.Error("failed to create notification channel", "error", err)
		return nil, fmt.Errorf("failed to create notification channel: %w", err)
	}

	s.logger.Info("notification channel created", "id", channel.ID, "name", channel.Name, "type", channel.Type)
	return channel, nil
}

// GetChannel retrieves a notification channel by ID.
func (s *AlertService) GetChannel(ctx context.Context, id uuid.UUID) (*alerting.NotificationChannel, error) {
	channel, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNotFound) {
			return nil, &NotFoundError{Resource: "notification channel", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}
	return channel, nil
}

// ListChannels retrieves notification channels with pagination.
func (s *AlertService) ListChannels(ctx context.Context, limit, offset int) (*models.ChannelListResponse, error) {
	channelList, err := s.repo.ListChannels(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification channels: %w", err)
	}

	if channelList == nil {
		channelList = []alerting.NotificationChannel{}
	}

	// Apply pagination
	total := len(channelList)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return &models.ChannelListResponse{
		Channels:   channelList[offset:end],
		TotalCount: total,
	}, nil
}

// UpdateChannel updates a notification channel.
func (s *AlertService) UpdateChannel(ctx context.Context, id uuid.UUID, req *models.UpdateChannelRequest) (*alerting.NotificationChannel, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Update channel
	channel, err := s.repo.UpdateChannel(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNotFound) {
			return nil, &NotFoundError{Resource: "notification channel", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrChannelNameExists) {
			return nil, &ConflictError{Message: "notification channel with this name already exists"}
		}
		return nil, fmt.Errorf("failed to update notification channel: %w", err)
	}

	s.logger.Info("notification channel updated", "id", channel.ID, "name", channel.Name)
	return channel, nil
}

// DeleteChannel deletes a notification channel.
func (s *AlertService) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteChannel(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNotFound) {
			return &NotFoundError{Resource: "notification channel", ID: id.String()}
		}
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	s.logger.Info("notification channel deleted", "id", id)
	return nil
}

// TestChannel tests a notification channel by sending a test notification.
func (s *AlertService) TestChannel(ctx context.Context, id uuid.UUID) (*models.TestChannelResponse, error) {
	// Get channel
	channel, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNotFound) {
			return nil, &NotFoundError{Resource: "notification channel", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}

	// Create channel sender
	sender, err := channels.NewChannel(channel.Type, channel.Config, s.logger)
	if err != nil {
		return &models.TestChannelResponse{
			Success:     false,
			Message:     "Failed to initialize channel",
			ErrorDetail: err.Error(),
		}, nil
	}

	// Test the channel
	if err := sender.Test(ctx); err != nil {
		s.logger.Error("channel test failed", "channel_id", id, "channel_name", channel.Name, "error", err)
		return &models.TestChannelResponse{
			Success:     false,
			Message:     "Channel test failed",
			ErrorDetail: err.Error(),
		}, nil
	}

	s.logger.Info("channel test successful", "id", id, "name", channel.Name, "type", channel.Type)
	return &models.TestChannelResponse{
		Success: true,
		Message: "Test notification sent successfully",
	}, nil
}

// Summary

// GetSummary retrieves alert statistics summary.
func (s *AlertService) GetSummary(ctx context.Context) (*models.AlertSummaryResponse, error) {
	summary, err := s.repo.GetAlertSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert summary: %w", err)
	}
	return summary, nil
}

// Routes

// CreateRoute creates a new alert route.
func (s *AlertService) CreateRoute(ctx context.Context, req *models.CreateRouteRequest) (*alerting.AlertRoute, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Verify rule exists
	_, err := s.repo.GetRule(ctx, req.RuleID)
	if err != nil {
		if errors.Is(err, repositories.ErrAlertRuleNotFound) {
			return nil, &NotFoundError{Resource: "alert rule", ID: req.RuleID.String()}
		}
		return nil, fmt.Errorf("failed to verify alert rule: %w", err)
	}

	// Verify channel exists
	_, err = s.repo.GetChannel(ctx, req.ChannelID)
	if err != nil {
		if errors.Is(err, repositories.ErrChannelNotFound) {
			return nil, &NotFoundError{Resource: "notification channel", ID: req.ChannelID.String()}
		}
		return nil, fmt.Errorf("failed to verify notification channel: %w", err)
	}

	// Create route
	route, err := s.repo.CreateRoute(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrRouteExists) {
			return nil, &ConflictError{Message: "alert route already exists for this rule and channel"}
		}
		s.logger.Error("failed to create alert route", "error", err)
		return nil, fmt.Errorf("failed to create alert route: %w", err)
	}

	s.logger.Info("alert route created", "id", route.ID, "rule_id", route.RuleID, "channel_id", route.ChannelID)
	return route, nil
}

// GetRoute retrieves an alert route by ID.
func (s *AlertService) GetRoute(ctx context.Context, id uuid.UUID) (*alerting.AlertRoute, error) {
	route, err := s.repo.GetRoute(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrRouteNotFound) {
			return nil, &NotFoundError{Resource: "alert route", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get alert route: %w", err)
	}
	return route, nil
}

// ListRoutes retrieves alert routes with optional filtering.
func (s *AlertService) ListRoutes(ctx context.Context, ruleID *uuid.UUID, limit, offset int) (*models.RouteListResponse, error) {
	routes, err := s.repo.ListRoutes(ctx, ruleID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert routes: %w", err)
	}

	if routes == nil {
		routes = []alerting.AlertRoute{}
	}

	// Apply pagination
	total := len(routes)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total || limit == 0 {
		end = total
	}

	return &models.RouteListResponse{
		Routes:     routes[offset:end],
		TotalCount: total,
	}, nil
}

// UpdateRoute updates an alert route.
func (s *AlertService) UpdateRoute(ctx context.Context, id uuid.UUID, req *models.UpdateRouteRequest) (*alerting.AlertRoute, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Update route
	route, err := s.repo.UpdateRoute(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrRouteNotFound) {
			return nil, &NotFoundError{Resource: "alert route", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to update alert route: %w", err)
	}

	s.logger.Info("alert route updated", "id", route.ID)
	return route, nil
}

// DeleteRoute deletes an alert route.
func (s *AlertService) DeleteRoute(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteRoute(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrRouteNotFound) {
			return &NotFoundError{Resource: "alert route", ID: id.String()}
		}
		return fmt.Errorf("failed to delete alert route: %w", err)
	}

	s.logger.Info("alert route deleted", "id", id)
	return nil
}
