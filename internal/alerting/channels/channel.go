// Package channels provides notification channel implementations for the alerting framework.
package channels

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/janovincze/philotes/internal/alerting"
)

// Channel is the interface that notification channels must implement.
type Channel interface {
	// Type returns the channel type identifier.
	Type() alerting.ChannelType

	// Send sends a notification through the channel.
	Send(ctx context.Context, notification alerting.Notification) error

	// Test sends a test notification to verify the channel configuration.
	Test(ctx context.Context) error
}

// NewChannel creates a channel from its type and configuration.
// This is the factory function for creating channel implementations.
func NewChannel(channelType alerting.ChannelType, config map[string]interface{}, logger *slog.Logger) (Channel, error) {
	if logger == nil {
		logger = slog.Default()
	}

	switch channelType {
	case alerting.ChannelSlack:
		return NewSlackChannel(config, logger)
	case alerting.ChannelEmail:
		return NewEmailChannel(config, logger)
	case alerting.ChannelWebhook:
		return NewWebhookChannel(config, logger)
	case alerting.ChannelPagerDuty:
		return nil, fmt.Errorf("PagerDuty channel not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// getStringConfig safely retrieves a string value from the config map.
func getStringConfig(config map[string]interface{}, key string) (string, bool) {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

// getIntConfig safely retrieves an int value from the config map.
func getIntConfig(config map[string]interface{}, key string) (int, bool) {
	if v, ok := config[key]; ok {
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		}
	}
	return 0, false
}

// getBoolConfig safely retrieves a bool value from the config map.
func getBoolConfig(config map[string]interface{}, key string) (bool, bool) {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// getStringSliceConfig safely retrieves a string slice from the config map.
func getStringSliceConfig(config map[string]interface{}, key string) ([]string, bool) {
	if v, ok := config[key]; ok {
		switch s := v.(type) {
		case []string:
			return s, true
		case []interface{}:
			result := make([]string, 0, len(s))
			for _, item := range s {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			if len(result) > 0 {
				return result, true
			}
		}
	}
	return nil, false
}

// getMapConfig safely retrieves a map from the config.
func getMapConfig(config map[string]interface{}, key string) (map[string]string, bool) {
	if v, ok := config[key]; ok {
		switch m := v.(type) {
		case map[string]string:
			return m, true
		case map[string]interface{}:
			result := make(map[string]string)
			for k, val := range m {
				if str, ok := val.(string); ok {
					result[k] = str
				}
			}
			if len(result) > 0 {
				return result, true
			}
		}
	}
	return nil, false
}

// SeverityColor returns a color code for a given severity level.
func SeverityColor(severity alerting.AlertSeverity) string {
	switch severity {
	case alerting.SeverityCritical:
		return "#dc3545" // Red
	case alerting.SeverityWarning:
		return "#ffc107" // Orange/Yellow
	case alerting.SeverityInfo:
		return "#17a2b8" // Blue
	default:
		return "#6c757d" // Gray
	}
}

// FormatAlertTitle creates a formatted title for an alert notification.
func FormatAlertTitle(notification alerting.Notification) string {
	status := "FIRING"
	if notification.Event == alerting.EventResolved {
		status = "RESOLVED"
	}

	severity := ""
	if notification.Rule != nil {
		severity = string(notification.Rule.Severity)
	}

	ruleName := "Unknown Rule"
	if notification.Rule != nil {
		ruleName = notification.Rule.Name
	}

	if severity != "" {
		return fmt.Sprintf("[%s] %s: %s", status, severity, ruleName)
	}
	return fmt.Sprintf("[%s] %s", status, ruleName)
}

// FormatAlertDescription creates a formatted description for an alert notification.
func FormatAlertDescription(notification alerting.Notification) string {
	if notification.Rule != nil && notification.Rule.Description != "" {
		return notification.Rule.Description
	}

	if notification.Alert != nil && notification.Alert.CurrentValue != nil && notification.Rule != nil {
		return fmt.Sprintf("Metric %s is %s %.2f (threshold: %.2f)",
			notification.Rule.MetricName,
			notification.Rule.Operator.String(),
			*notification.Alert.CurrentValue,
			notification.Rule.Threshold,
		)
	}

	return "Alert triggered"
}
