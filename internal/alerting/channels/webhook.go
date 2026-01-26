package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/janovincze/philotes/internal/alerting"
)

// WebhookChannel implements the Channel interface for generic HTTP webhooks.
type WebhookChannel struct {
	url        string
	method     string
	headers    map[string]string
	httpClient *http.Client
	logger     *slog.Logger
}

// WebhookPayload represents the JSON payload sent to webhooks.
type WebhookPayload struct {
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Event     string                 `json:"event"`
	Alert     *WebhookAlertPayload   `json:"alert"`
	Rule      *WebhookRulePayload    `json:"rule,omitempty"`
	Channel   *WebhookChannelPayload `json:"channel,omitempty"`
}

// WebhookAlertPayload represents alert information in the webhook payload.
type WebhookAlertPayload struct {
	ID             string            `json:"id"`
	Fingerprint    string            `json:"fingerprint"`
	Status         string            `json:"status"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	CurrentValue   *float64          `json:"current_value,omitempty"`
	FiredAt        time.Time         `json:"fired_at"`
	ResolvedAt     *time.Time        `json:"resolved_at,omitempty"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string            `json:"acknowledged_by,omitempty"`
}

// WebhookRulePayload represents rule information in the webhook payload.
type WebhookRulePayload struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	MetricName      string            `json:"metric_name"`
	Operator        string            `json:"operator"`
	Threshold       float64           `json:"threshold"`
	DurationSeconds int               `json:"duration_seconds"`
	Severity        string            `json:"severity"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// WebhookChannelPayload represents channel information in the webhook payload.
type WebhookChannelPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// NewWebhookChannel creates a new webhook notification channel.
func NewWebhookChannel(config map[string]interface{}, logger *slog.Logger) (*WebhookChannel, error) {
	url, ok := getStringConfig(config, "url")
	if !ok || url == "" {
		return nil, fmt.Errorf("webhook channel requires url configuration")
	}

	method, _ := getStringConfig(config, "method")
	if method == "" {
		method = http.MethodPost
	}

	headers, _ := getMapConfig(config, "headers")
	if headers == nil {
		headers = make(map[string]string)
	}

	return &WebhookChannel{
		url:     url,
		method:  method,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "webhook-channel"),
	}, nil
}

// Type returns the channel type.
func (c *WebhookChannel) Type() alerting.ChannelType {
	return alerting.ChannelWebhook
}

// Send sends a notification via webhook.
func (c *WebhookChannel) Send(ctx context.Context, notification alerting.Notification) error {
	payload := c.buildPayload(notification)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	c.logger.Debug("sending webhook notification",
		"url", c.url,
		"method", c.method,
	)

	req, err := http.NewRequestWithContext(ctx, c.method, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Philotes-Alerting/1.0")

	// Set custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Accept 2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("webhook notification sent successfully",
		"url", c.url,
		"status_code", resp.StatusCode,
		"event", notification.Event,
	)

	return nil
}

// Test sends a test notification to verify the channel configuration.
func (c *WebhookChannel) Test(ctx context.Context) error {
	payload := WebhookPayload{
		Version:   "1.0",
		Timestamp: time.Now(),
		Event:     "test",
		Alert: &WebhookAlertPayload{
			ID:          "test-alert-id",
			Fingerprint: "test-fingerprint",
			Status:      "test",
			Labels: map[string]string{
				"test": "true",
			},
			FiredAt: time.Now(),
		},
		Rule: &WebhookRulePayload{
			ID:          "test-rule-id",
			Name:        "Test Rule",
			Description: "This is a test notification",
			MetricName:  "test_metric",
			Operator:    "gt",
			Threshold:   100,
			Severity:    "info",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal test payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, c.method, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Philotes-Alerting/1.0")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook test failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook test failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// buildPayload builds a webhook payload from a notification.
func (c *WebhookChannel) buildPayload(notification alerting.Notification) WebhookPayload {
	payload := WebhookPayload{
		Version:   "1.0",
		Timestamp: time.Now(),
		Event:     string(notification.Event),
	}

	if notification.Alert != nil {
		payload.Alert = &WebhookAlertPayload{
			ID:             notification.Alert.ID.String(),
			Fingerprint:    notification.Alert.Fingerprint,
			Status:         string(notification.Alert.Status),
			Labels:         notification.Alert.Labels,
			Annotations:    notification.Alert.Annotations,
			CurrentValue:   notification.Alert.CurrentValue,
			FiredAt:        notification.Alert.FiredAt,
			ResolvedAt:     notification.Alert.ResolvedAt,
			AcknowledgedAt: notification.Alert.AcknowledgedAt,
			AcknowledgedBy: notification.Alert.AcknowledgedBy,
		}
	}

	if notification.Rule != nil {
		payload.Rule = &WebhookRulePayload{
			ID:              notification.Rule.ID.String(),
			Name:            notification.Rule.Name,
			Description:     notification.Rule.Description,
			MetricName:      notification.Rule.MetricName,
			Operator:        string(notification.Rule.Operator),
			Threshold:       notification.Rule.Threshold,
			DurationSeconds: notification.Rule.DurationSeconds,
			Severity:        string(notification.Rule.Severity),
			Labels:          notification.Rule.Labels,
			Annotations:     notification.Rule.Annotations,
		}
	}

	if notification.Channel != nil {
		payload.Channel = &WebhookChannelPayload{
			ID:   notification.Channel.ID.String(),
			Name: notification.Channel.Name,
			Type: string(notification.Channel.Type),
		}
	}

	return payload
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (c *WebhookChannel) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// Ensure WebhookChannel implements Channel interface.
var _ Channel = (*WebhookChannel)(nil)
