// Package alerting provides the alerting framework for Philotes.
package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertRepository defines the interface for alert data access.
// This is defined here to avoid circular imports with the repositories package.
type AlertRepository interface {
	// Rule operations
	ListRules(ctx context.Context, enabledOnly bool) ([]AlertRule, error)
	GetRule(ctx context.Context, id uuid.UUID) (*AlertRule, error)

	// Instance operations
	CreateInstance(ctx context.Context, instance *AlertInstance) (*AlertInstance, error)
	GetInstanceByFingerprint(ctx context.Context, ruleID uuid.UUID, fingerprint string) (*AlertInstance, error)
	ListInstances(ctx context.Context, status *AlertStatus, ruleID *uuid.UUID) ([]AlertInstance, error)
	UpdateInstance(ctx context.Context, id uuid.UUID, status AlertStatus, currentValue *float64, resolvedAt *time.Time) error

	// History operations
	CreateHistory(ctx context.Context, history *AlertHistory) (*AlertHistory, error)

	// Silence operations
	ListSilences(ctx context.Context, activeOnly bool) ([]AlertSilence, error)

	// Channel operations
	GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error)
	ListChannels(ctx context.Context, enabledOnly bool) ([]NotificationChannel, error)

	// Route operations
	ListRoutes(ctx context.Context, ruleID *uuid.UUID, enabledOnly bool) ([]AlertRoute, error)
}

// ChannelSender defines the interface for sending notifications through a channel.
type ChannelSender interface {
	Type() ChannelType
	Send(ctx context.Context, notification Notification) error
}

// ChannelFactory creates channel senders from configuration.
type ChannelFactory func(channelType ChannelType, config map[string]interface{}, logger *slog.Logger) (ChannelSender, error)

// Notifier dispatches notifications to configured channels.
type Notifier struct {
	repo           AlertRepository
	channelFactory ChannelFactory
	logger         *slog.Logger
	timeout        time.Duration

	// Track last notification time per alert+channel
	lastNotified map[string]time.Time // fingerprint:channel_id -> time
	mu           sync.RWMutex
}

// NewNotifier creates a new notifier.
func NewNotifier(repo AlertRepository, channelFactory ChannelFactory, timeout time.Duration, logger *slog.Logger) *Notifier {
	if logger == nil {
		logger = slog.Default()
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Notifier{
		repo:           repo,
		channelFactory: channelFactory,
		logger:         logger.With("component", "alert-notifier"),
		timeout:        timeout,
		lastNotified:   make(map[string]time.Time),
	}
}

// Notify sends notifications for an alert to all configured channels.
func (n *Notifier) Notify(ctx context.Context, alert AlertInstance, rule AlertRule, eventType EventType) error {
	n.logger.Debug("processing notification",
		"alert_id", alert.ID,
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"event_type", eventType,
	)

	// Get routes for this rule
	routes, err := n.repo.ListRoutes(ctx, &rule.ID, true)
	if err != nil {
		return fmt.Errorf("failed to list routes for rule %s: %w", rule.ID, err)
	}

	if len(routes) == 0 {
		n.logger.Debug("no routes configured for rule",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
		)
		return nil
	}

	var notifyErrors []error

	for _, route := range routes {
		// Check if we should skip due to repeat interval
		if !n.shouldNotify(alert.Fingerprint, route.ChannelID, route.RepeatIntervalSeconds, eventType) {
			n.logger.Debug("skipping notification due to repeat interval",
				"alert_fingerprint", alert.Fingerprint,
				"channel_id", route.ChannelID,
			)
			continue
		}

		// Get the channel
		channel, err := n.repo.GetChannel(ctx, route.ChannelID)
		if err != nil {
			n.logger.Error("failed to get channel",
				"channel_id", route.ChannelID,
				"error", err,
			)
			notifyErrors = append(notifyErrors, fmt.Errorf("failed to get channel %s: %w", route.ChannelID, err))
			continue
		}

		if !channel.Enabled {
			n.logger.Debug("skipping disabled channel",
				"channel_id", channel.ID,
				"channel_name", channel.Name,
			)
			continue
		}

		// Create the notification
		notification := Notification{
			Alert:   &alert,
			Rule:    &rule,
			Channel: channel,
			Route:   &route,
			Event:   eventType,
		}

		// Send the notification
		if err := n.sendNotification(ctx, notification); err != nil {
			notifyErrors = append(notifyErrors, err)
			// Record notification failure in history
			n.recordNotificationEvent(ctx, alert, rule, channel, EventNotificationFailed, err.Error())
		} else {
			// Update last notified time
			n.updateLastNotified(alert.Fingerprint, channel.ID)
			// Record successful notification in history
			n.recordNotificationEvent(ctx, alert, rule, channel, EventNotificationSent, "")
		}
	}

	if len(notifyErrors) > 0 {
		return fmt.Errorf("notification errors: %v", notifyErrors)
	}

	return nil
}

// shouldNotify checks if we should send a notification based on repeat interval.
func (n *Notifier) shouldNotify(fingerprint string, channelID uuid.UUID, repeatIntervalSeconds int, eventType EventType) bool {
	// Always notify on resolved events
	if eventType == EventResolved {
		return true
	}

	key := fmt.Sprintf("%s:%s", fingerprint, channelID.String())

	n.mu.RLock()
	lastTime, exists := n.lastNotified[key]
	n.mu.RUnlock()

	if !exists {
		return true
	}

	repeatInterval := time.Duration(repeatIntervalSeconds) * time.Second
	return time.Since(lastTime) >= repeatInterval
}

// updateLastNotified updates the last notification time for an alert+channel.
func (n *Notifier) updateLastNotified(fingerprint string, channelID uuid.UUID) {
	key := fmt.Sprintf("%s:%s", fingerprint, channelID.String())

	n.mu.Lock()
	n.lastNotified[key] = time.Now()
	n.mu.Unlock()
}

// sendNotification sends a notification through a channel.
func (n *Notifier) sendNotification(ctx context.Context, notification Notification) error {
	if notification.Channel == nil {
		return fmt.Errorf("notification channel is nil")
	}

	if n.channelFactory == nil {
		return fmt.Errorf("channel factory not configured")
	}

	// Create the channel sender
	sender, err := n.channelFactory(notification.Channel.Type, notification.Channel.Config, n.logger)
	if err != nil {
		return fmt.Errorf("failed to create channel sender for %s: %w", notification.Channel.Type, err)
	}

	// Create a context with timeout
	sendCtx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	n.logger.Info("sending notification",
		"channel_type", notification.Channel.Type,
		"channel_name", notification.Channel.Name,
		"rule_name", notification.Rule.Name,
		"event", notification.Event,
	)

	if err := sender.Send(sendCtx, notification); err != nil {
		n.logger.Error("failed to send notification",
			"channel_type", notification.Channel.Type,
			"channel_name", notification.Channel.Name,
			"error", err,
		)
		return fmt.Errorf("failed to send notification via %s: %w", notification.Channel.Name, err)
	}

	n.logger.Info("notification sent successfully",
		"channel_type", notification.Channel.Type,
		"channel_name", notification.Channel.Name,
	)

	return nil
}

// recordNotificationEvent records a notification event in history.
func (n *Notifier) recordNotificationEvent(ctx context.Context, alert AlertInstance, rule AlertRule, channel *NotificationChannel, eventType EventType, errMsg string) {
	history := &AlertHistory{
		AlertID:   alert.ID,
		RuleID:    rule.ID,
		EventType: eventType,
		Message:   fmt.Sprintf("Notification to %s (%s)", channel.Name, channel.Type),
		Metadata: map[string]any{
			"channel_id":   channel.ID.String(),
			"channel_name": channel.Name,
			"channel_type": string(channel.Type),
		},
	}

	if errMsg != "" {
		history.Message = fmt.Sprintf("Failed to notify %s (%s): %s", channel.Name, channel.Type, errMsg)
		history.Metadata["error"] = errMsg
	}

	if alert.CurrentValue != nil {
		history.Value = alert.CurrentValue
	}

	if _, err := n.repo.CreateHistory(ctx, history); err != nil {
		n.logger.Error("failed to create notification history",
			"alert_id", alert.ID,
			"error", err,
		)
	}
}

// ClearLastNotified clears the last notification time for an alert.
// This should be called when an alert is resolved.
func (n *Notifier) ClearLastNotified(fingerprint string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Clear all entries for this fingerprint
	prefix := fingerprint + ":"
	for key := range n.lastNotified {
		if strings.HasPrefix(key, prefix) {
			delete(n.lastNotified, key)
		}
	}
}

// NotifyBatch sends notifications for multiple alerts efficiently.
func (n *Notifier) NotifyBatch(ctx context.Context, alerts []struct {
	Alert AlertInstance
	Rule  AlertRule
	Event EventType
}) error {
	var errors []error

	for _, a := range alerts {
		if err := n.Notify(ctx, a.Alert, a.Rule, a.Event); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch notification had %d errors", len(errors))
	}

	return nil
}
