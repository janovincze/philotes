// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/alerting"
)

// AlertRule represents an alert rule in API responses.
type AlertRule struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	MetricName      string                 `json:"metric_name"`
	Operator        alerting.Operator      `json:"operator"`
	Threshold       float64                `json:"threshold"`
	DurationSeconds int                    `json:"duration_seconds"`
	Severity        alerting.AlertSeverity `json:"severity"`
	Labels          map[string]string      `json:"labels,omitempty"`
	Annotations     map[string]string      `json:"annotations,omitempty"`
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// CreateAlertRuleRequest represents a request to create an alert rule.
type CreateAlertRuleRequest struct {
	Name            string                 `json:"name" binding:"required,min=1,max=255"`
	Description     string                 `json:"description,omitempty"`
	MetricName      string                 `json:"metric_name" binding:"required"`
	Operator        alerting.Operator      `json:"operator" binding:"required"`
	Threshold       float64                `json:"threshold"`
	DurationSeconds int                    `json:"duration_seconds,omitempty"`
	Severity        alerting.AlertSeverity `json:"severity,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty"`
	Annotations     map[string]string      `json:"annotations,omitempty"`
	Enabled         *bool                  `json:"enabled,omitempty"`
}

// Validate validates the create alert rule request.
func (r *CreateAlertRuleRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if r.MetricName == "" {
		errors = append(errors, FieldError{Field: "metric_name", Message: "metric_name is required"})
	}
	if !r.Operator.IsValid() {
		errors = append(errors, FieldError{Field: "operator", Message: "operator must be one of: gt, lt, eq, gte, lte"})
	}
	if r.DurationSeconds < 0 {
		errors = append(errors, FieldError{Field: "duration_seconds", Message: "duration_seconds cannot be negative"})
	}
	if r.Severity != "" && !r.Severity.IsValid() {
		errors = append(errors, FieldError{Field: "severity", Message: "severity must be one of: info, warning, critical"})
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateAlertRuleRequest) ApplyDefaults() {
	if r.Severity == "" {
		r.Severity = alerting.SeverityWarning
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}
}

// UpdateAlertRuleRequest represents a request to update an alert rule.
type UpdateAlertRuleRequest struct {
	Name            *string                 `json:"name,omitempty"`
	Description     *string                 `json:"description,omitempty"`
	MetricName      *string                 `json:"metric_name,omitempty"`
	Operator        *alerting.Operator      `json:"operator,omitempty"`
	Threshold       *float64                `json:"threshold,omitempty"`
	DurationSeconds *int                    `json:"duration_seconds,omitempty"`
	Severity        *alerting.AlertSeverity `json:"severity,omitempty"`
	Labels          map[string]string       `json:"labels,omitempty"`
	Annotations     map[string]string       `json:"annotations,omitempty"`
	Enabled         *bool                   `json:"enabled,omitempty"`
}

// Validate validates the update alert rule request.
func (r *UpdateAlertRuleRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}
	if r.MetricName != nil && *r.MetricName == "" {
		errors = append(errors, FieldError{Field: "metric_name", Message: "metric_name cannot be empty"})
	}
	if r.Operator != nil && !r.Operator.IsValid() {
		errors = append(errors, FieldError{Field: "operator", Message: "operator must be one of: gt, lt, eq, gte, lte"})
	}
	if r.DurationSeconds != nil && *r.DurationSeconds < 0 {
		errors = append(errors, FieldError{Field: "duration_seconds", Message: "duration_seconds cannot be negative"})
	}
	if r.Severity != nil && !r.Severity.IsValid() {
		errors = append(errors, FieldError{Field: "severity", Message: "severity must be one of: info, warning, critical"})
	}

	return errors
}

// AlertRuleResponse wraps an alert rule for API responses.
type AlertRuleResponse struct {
	Rule *alerting.AlertRule `json:"rule"`
}

// AlertRuleListResponse wraps a list of alert rules for API responses.
type AlertRuleListResponse struct {
	Rules      []alerting.AlertRule `json:"rules"`
	TotalCount int                  `json:"total_count"`
}

// AlertInstance represents an alert instance in API responses.
type AlertInstance struct {
	ID             uuid.UUID            `json:"id"`
	RuleID         uuid.UUID            `json:"rule_id"`
	Fingerprint    string               `json:"fingerprint"`
	Status         alerting.AlertStatus `json:"status"`
	Labels         map[string]string    `json:"labels,omitempty"`
	Annotations    map[string]string    `json:"annotations,omitempty"`
	CurrentValue   *float64             `json:"current_value,omitempty"`
	FiredAt        time.Time            `json:"fired_at"`
	ResolvedAt     *time.Time           `json:"resolved_at,omitempty"`
	AcknowledgedAt *time.Time           `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string               `json:"acknowledged_by,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	Rule           *alerting.AlertRule  `json:"rule,omitempty"`
}

// AlertInstanceResponse wraps an alert instance for API responses.
type AlertInstanceResponse struct {
	Alert *alerting.AlertInstance `json:"alert"`
}

// AlertInstanceListResponse wraps a list of alert instances for API responses.
type AlertInstanceListResponse struct {
	Alerts     []alerting.AlertInstance `json:"alerts"`
	TotalCount int                      `json:"total_count"`
}

// AcknowledgeAlertRequest represents a request to acknowledge an alert.
type AcknowledgeAlertRequest struct {
	AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	Comment        string `json:"comment,omitempty"`
}

// Validate validates the acknowledge alert request.
func (r *AcknowledgeAlertRequest) Validate() []FieldError {
	var errors []FieldError

	if r.AcknowledgedBy == "" {
		errors = append(errors, FieldError{Field: "acknowledged_by", Message: "acknowledged_by is required"})
	}

	return errors
}

// AlertHistoryResponse wraps alert history for API responses.
type AlertHistoryResponse struct {
	History    []alerting.AlertHistory `json:"history"`
	TotalCount int                     `json:"total_count"`
}

// CreateSilenceRequest represents a request to create an alert silence.
type CreateSilenceRequest struct {
	Matchers  map[string]string `json:"matchers" binding:"required"`
	StartsAt  time.Time         `json:"starts_at" binding:"required"`
	EndsAt    time.Time         `json:"ends_at" binding:"required"`
	CreatedBy string            `json:"created_by" binding:"required"`
	Comment   string            `json:"comment,omitempty"`
}

// Validate validates the create silence request.
func (r *CreateSilenceRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Matchers == nil || len(r.Matchers) == 0 {
		errors = append(errors, FieldError{Field: "matchers", Message: "matchers is required and cannot be empty"})
	}
	if r.StartsAt.IsZero() {
		errors = append(errors, FieldError{Field: "starts_at", Message: "starts_at is required"})
	}
	if r.EndsAt.IsZero() {
		errors = append(errors, FieldError{Field: "ends_at", Message: "ends_at is required"})
	}
	if !r.StartsAt.IsZero() && !r.EndsAt.IsZero() && r.EndsAt.Before(r.StartsAt) {
		errors = append(errors, FieldError{Field: "ends_at", Message: "ends_at must be after starts_at"})
	}
	if r.CreatedBy == "" {
		errors = append(errors, FieldError{Field: "created_by", Message: "created_by is required"})
	}

	return errors
}

// SilenceResponse wraps a silence for API responses.
type SilenceResponse struct {
	Silence *alerting.AlertSilence `json:"silence"`
}

// SilenceListResponse wraps a list of silences for API responses.
type SilenceListResponse struct {
	Silences   []alerting.AlertSilence `json:"silences"`
	TotalCount int                     `json:"total_count"`
}

// CreateChannelRequest represents a request to create a notification channel.
type CreateChannelRequest struct {
	Name    string               `json:"name" binding:"required,min=1,max=255"`
	Type    alerting.ChannelType `json:"type" binding:"required"`
	Config  map[string]any       `json:"config" binding:"required"`
	Enabled *bool                `json:"enabled,omitempty"`
}

// Validate validates the create channel request.
func (r *CreateChannelRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if !r.Type.IsValid() {
		errors = append(errors, FieldError{Field: "type", Message: "type must be one of: slack, email, webhook, pagerduty"})
	}
	if r.Config == nil || len(r.Config) == 0 {
		errors = append(errors, FieldError{Field: "config", Message: "config is required and cannot be empty"})
	}

	// Validate channel-specific config
	switch r.Type {
	case alerting.ChannelSlack:
		if _, ok := r.Config["webhook_url"]; !ok {
			errors = append(errors, FieldError{Field: "config.webhook_url", Message: "webhook_url is required for Slack channels"})
		}
	case alerting.ChannelEmail:
		if _, ok := r.Config["smtp_host"]; !ok {
			errors = append(errors, FieldError{Field: "config.smtp_host", Message: "smtp_host is required for email channels"})
		}
		if _, ok := r.Config["to"]; !ok {
			errors = append(errors, FieldError{Field: "config.to", Message: "to is required for email channels"})
		}
	case alerting.ChannelWebhook:
		if _, ok := r.Config["url"]; !ok {
			errors = append(errors, FieldError{Field: "config.url", Message: "url is required for webhook channels"})
		}
	case alerting.ChannelPagerDuty:
		if _, ok := r.Config["routing_key"]; !ok {
			errors = append(errors, FieldError{Field: "config.routing_key", Message: "routing_key is required for PagerDuty channels"})
		}
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateChannelRequest) ApplyDefaults() {
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
}

// UpdateChannelRequest represents a request to update a notification channel.
type UpdateChannelRequest struct {
	Name    *string        `json:"name,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
	Enabled *bool          `json:"enabled,omitempty"`
}

// Validate validates the update channel request.
func (r *UpdateChannelRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}

	return errors
}

// ChannelResponse wraps a notification channel for API responses.
type ChannelResponse struct {
	Channel *alerting.NotificationChannel `json:"channel"`
}

// ChannelListResponse wraps a list of notification channels for API responses.
type ChannelListResponse struct {
	Channels   []alerting.NotificationChannel `json:"channels"`
	TotalCount int                            `json:"total_count"`
}

// CreateRouteRequest represents a request to create an alert route.
type CreateRouteRequest struct {
	RuleID                uuid.UUID `json:"rule_id" binding:"required"`
	ChannelID             uuid.UUID `json:"channel_id" binding:"required"`
	RepeatIntervalSeconds *int      `json:"repeat_interval_seconds,omitempty"`
	GroupWaitSeconds      *int      `json:"group_wait_seconds,omitempty"`
	GroupIntervalSeconds  *int      `json:"group_interval_seconds,omitempty"`
	Enabled               *bool     `json:"enabled,omitempty"`
}

// Validate validates the create route request.
func (r *CreateRouteRequest) Validate() []FieldError {
	var errors []FieldError

	if r.RuleID == uuid.Nil {
		errors = append(errors, FieldError{Field: "rule_id", Message: "rule_id is required"})
	}
	if r.ChannelID == uuid.Nil {
		errors = append(errors, FieldError{Field: "channel_id", Message: "channel_id is required"})
	}
	if r.RepeatIntervalSeconds != nil && *r.RepeatIntervalSeconds < 0 {
		errors = append(errors, FieldError{Field: "repeat_interval_seconds", Message: "repeat_interval_seconds cannot be negative"})
	}
	if r.GroupWaitSeconds != nil && *r.GroupWaitSeconds < 0 {
		errors = append(errors, FieldError{Field: "group_wait_seconds", Message: "group_wait_seconds cannot be negative"})
	}
	if r.GroupIntervalSeconds != nil && *r.GroupIntervalSeconds < 0 {
		errors = append(errors, FieldError{Field: "group_interval_seconds", Message: "group_interval_seconds cannot be negative"})
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateRouteRequest) ApplyDefaults() {
	if r.RepeatIntervalSeconds == nil {
		val := 3600 // 1 hour
		r.RepeatIntervalSeconds = &val
	}
	if r.GroupWaitSeconds == nil {
		val := 30
		r.GroupWaitSeconds = &val
	}
	if r.GroupIntervalSeconds == nil {
		val := 300 // 5 minutes
		r.GroupIntervalSeconds = &val
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
}

// UpdateRouteRequest represents a request to update an alert route.
type UpdateRouteRequest struct {
	RepeatIntervalSeconds *int  `json:"repeat_interval_seconds,omitempty"`
	GroupWaitSeconds      *int  `json:"group_wait_seconds,omitempty"`
	GroupIntervalSeconds  *int  `json:"group_interval_seconds,omitempty"`
	Enabled               *bool `json:"enabled,omitempty"`
}

// Validate validates the update route request.
func (r *UpdateRouteRequest) Validate() []FieldError {
	var errors []FieldError

	if r.RepeatIntervalSeconds != nil && *r.RepeatIntervalSeconds < 0 {
		errors = append(errors, FieldError{Field: "repeat_interval_seconds", Message: "repeat_interval_seconds cannot be negative"})
	}
	if r.GroupWaitSeconds != nil && *r.GroupWaitSeconds < 0 {
		errors = append(errors, FieldError{Field: "group_wait_seconds", Message: "group_wait_seconds cannot be negative"})
	}
	if r.GroupIntervalSeconds != nil && *r.GroupIntervalSeconds < 0 {
		errors = append(errors, FieldError{Field: "group_interval_seconds", Message: "group_interval_seconds cannot be negative"})
	}

	return errors
}

// RouteResponse wraps an alert route for API responses.
type RouteResponse struct {
	Route *alerting.AlertRoute `json:"route"`
}

// RouteListResponse wraps a list of alert routes for API responses.
type RouteListResponse struct {
	Routes     []alerting.AlertRoute `json:"routes"`
	TotalCount int                   `json:"total_count"`
}

// TestChannelRequest represents a request to test a notification channel.
type TestChannelRequest struct {
	Message string `json:"message,omitempty"`
}

// TestChannelResponse represents the result of testing a notification channel.
type TestChannelResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

// AlertSummaryResponse provides a summary of alert statistics.
type AlertSummaryResponse struct {
	TotalRules      int `json:"total_rules"`
	EnabledRules    int `json:"enabled_rules"`
	FiringAlerts    int `json:"firing_alerts"`
	ResolvedAlerts  int `json:"resolved_alerts"`
	ActiveSilences  int `json:"active_silences"`
	TotalChannels   int `json:"total_channels"`
	EnabledChannels int `json:"enabled_channels"`
}
