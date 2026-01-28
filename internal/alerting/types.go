// Package alerting provides the alerting framework for Philotes.
package alerting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	// SeverityInfo indicates an informational alert.
	SeverityInfo AlertSeverity = "info"
	// SeverityWarning indicates a warning alert.
	SeverityWarning AlertSeverity = "warning"
	// SeverityCritical indicates a critical alert.
	SeverityCritical AlertSeverity = "critical"
)

// IsValid checks if the severity is valid.
func (s AlertSeverity) IsValid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	}
	return false
}

// AlertStatus represents the status of an alert instance.
type AlertStatus string

const (
	// StatusFiring indicates the alert is currently firing.
	StatusFiring AlertStatus = "firing"
	// StatusResolved indicates the alert has been resolved.
	StatusResolved AlertStatus = "resolved"
)

// IsValid checks if the status is valid.
func (s AlertStatus) IsValid() bool {
	switch s {
	case StatusFiring, StatusResolved:
		return true
	}
	return false
}

// Operator represents a comparison operator for alert rules.
type Operator string

const (
	// OpGreaterThan represents the > operator.
	OpGreaterThan Operator = "gt"
	// OpLessThan represents the < operator.
	OpLessThan Operator = "lt"
	// OpEqual represents the == operator.
	OpEqual Operator = "eq"
	// OpGreaterThanEqual represents the >= operator.
	OpGreaterThanEqual Operator = "gte"
	// OpLessThanEqual represents the <= operator.
	OpLessThanEqual Operator = "lte"
)

// IsValid checks if the operator is valid.
func (o Operator) IsValid() bool {
	switch o {
	case OpGreaterThan, OpLessThan, OpEqual, OpGreaterThanEqual, OpLessThanEqual:
		return true
	}
	return false
}

// Evaluate evaluates the operator against the given values.
func (o Operator) Evaluate(value, threshold float64) bool {
	switch o {
	case OpGreaterThan:
		return value > threshold
	case OpLessThan:
		return value < threshold
	case OpEqual:
		return value == threshold
	case OpGreaterThanEqual:
		return value >= threshold
	case OpLessThanEqual:
		return value <= threshold
	}
	return false
}

// String returns the human-readable representation of the operator.
func (o Operator) String() string {
	switch o {
	case OpGreaterThan:
		return ">"
	case OpLessThan:
		return "<"
	case OpEqual:
		return "=="
	case OpGreaterThanEqual:
		return ">="
	case OpLessThanEqual:
		return "<="
	}
	return string(o)
}

// ChannelType represents the type of notification channel.
type ChannelType string

const (
	// ChannelSlack represents a Slack notification channel.
	ChannelSlack ChannelType = "slack"
	// ChannelEmail represents an email notification channel.
	ChannelEmail ChannelType = "email"
	// ChannelWebhook represents a webhook notification channel.
	ChannelWebhook ChannelType = "webhook"
	// ChannelPagerDuty represents a PagerDuty notification channel.
	ChannelPagerDuty ChannelType = "pagerduty"
)

// IsValid checks if the channel type is valid.
func (c ChannelType) IsValid() bool {
	switch c {
	case ChannelSlack, ChannelEmail, ChannelWebhook, ChannelPagerDuty:
		return true
	}
	return false
}

// EventType represents the type of alert event.
type EventType string

const (
	// EventFired indicates an alert was fired.
	EventFired EventType = "fired"
	// EventResolved indicates an alert was resolved.
	EventResolved EventType = "resolved"
	// EventAcknowledged indicates an alert was acknowledged.
	EventAcknowledged EventType = "acknowledged"
	// EventNotificationSent indicates a notification was sent.
	EventNotificationSent EventType = "notification_sent"
	// EventNotificationFailed indicates a notification failed.
	EventNotificationFailed EventType = "notification_failed"
)

// AlertRule represents an alert rule definition.
type AlertRule struct {
	ID              uuid.UUID         `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	MetricName      string            `json:"metric_name"`
	Operator        Operator          `json:"operator"`
	Threshold       float64           `json:"threshold"`
	DurationSeconds int               `json:"duration_seconds"`
	Severity        AlertSeverity     `json:"severity"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Enabled         bool              `json:"enabled"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AlertInstance represents an active or resolved alert instance.
type AlertInstance struct {
	ID             uuid.UUID         `json:"id"`
	RuleID         uuid.UUID         `json:"rule_id"`
	Fingerprint    string            `json:"fingerprint"`
	Status         AlertStatus       `json:"status"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	CurrentValue   *float64          `json:"current_value,omitempty"`
	FiredAt        time.Time         `json:"fired_at"`
	ResolvedAt     *time.Time        `json:"resolved_at,omitempty"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string            `json:"acknowledged_by,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`

	// Rule is optionally populated when loading instances with their rules.
	Rule *AlertRule `json:"rule,omitempty"`
}

// GenerateFingerprint generates a unique fingerprint for an alert instance
// based on the rule ID and labels.
func GenerateFingerprint(ruleID uuid.UUID, labels map[string]string) string {
	// Sort label keys for deterministic fingerprint
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the fingerprint data
	data := map[string]any{
		"rule_id": ruleID.String(),
		"labels":  labels,
	}

	// Convert to JSON for consistent serialization
	// Error is intentionally ignored as this map structure is always marshalable
	jsonData, _ := json.Marshal(data) //nolint:errcheck

	// Generate SHA256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// AlertHistory represents an audit trail entry for an alert.
type AlertHistory struct {
	ID        uuid.UUID      `json:"id"`
	AlertID   uuid.UUID      `json:"alert_id"`
	RuleID    uuid.UUID      `json:"rule_id"`
	EventType EventType      `json:"event_type"`
	Message   string         `json:"message,omitempty"`
	Value     *float64       `json:"value,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// AlertSilence represents a temporary alert suppression rule.
type AlertSilence struct {
	ID        uuid.UUID         `json:"id"`
	Matchers  map[string]string `json:"matchers"`
	StartsAt  time.Time         `json:"starts_at"`
	EndsAt    time.Time         `json:"ends_at"`
	CreatedBy string            `json:"created_by"`
	Comment   string            `json:"comment,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// IsActive checks if the silence is currently active.
func (s *AlertSilence) IsActive() bool {
	now := time.Now()
	return now.After(s.StartsAt) && now.Before(s.EndsAt)
}

// Matches checks if the given labels match the silence matchers.
func (s *AlertSilence) Matches(labels map[string]string) bool {
	for key, value := range s.Matchers {
		if labelValue, ok := labels[key]; !ok || labelValue != value {
			return false
		}
	}
	return true
}

// NotificationChannel represents a notification channel configuration.
type NotificationChannel struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Type      ChannelType    `json:"type"`
	Config    map[string]any `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// AlertRoute represents a routing rule linking an alert rule to a notification channel.
type AlertRoute struct {
	ID                    uuid.UUID `json:"id"`
	RuleID                uuid.UUID `json:"rule_id"`
	ChannelID             uuid.UUID `json:"channel_id"`
	RepeatIntervalSeconds int       `json:"repeat_interval_seconds"`
	GroupWaitSeconds      int       `json:"group_wait_seconds"`
	GroupIntervalSeconds  int       `json:"group_interval_seconds"`
	Enabled               bool      `json:"enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	// Channel is optionally populated when loading routes with their channels.
	Channel *NotificationChannel `json:"channel,omitempty"`
}

// EvaluationResult represents the result of evaluating an alert rule.
type EvaluationResult struct {
	Rule         *AlertRule
	Value        float64
	Labels       map[string]string
	ShouldFire   bool
	EvaluatedAt  time.Time
	ErrorMessage string
}

// Notification represents a notification to be sent.
type Notification struct {
	Alert   *AlertInstance
	Rule    *AlertRule
	Channel *NotificationChannel
	Route   *AlertRoute
	Event   EventType
}
