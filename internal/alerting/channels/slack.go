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

// SlackChannel implements the Channel interface for Slack webhooks.
type SlackChannel struct {
	webhookURL string
	channel    string
	username   string
	httpClient *http.Client
	logger     *slog.Logger
}

// SlackConfig holds configuration for a Slack channel.
type SlackConfig struct {
	WebhookURL string
	Channel    string
	Username   string
}

// slackMessage represents a Slack webhook message.
type slackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

// slackAttachment represents a Slack message attachment.
type slackAttachment struct {
	Fallback   string       `json:"fallback"`
	Color      string       `json:"color"`
	Title      string       `json:"title"`
	Text       string       `json:"text,omitempty"`
	Fields     []slackField `json:"fields,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	MarkdownIn []string     `json:"mrkdwn_in,omitempty"`
}

// slackField represents a field in a Slack attachment.
type slackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackChannel creates a new Slack notification channel.
func NewSlackChannel(config map[string]interface{}, logger *slog.Logger) (*SlackChannel, error) {
	webhookURL, ok := getStringConfig(config, "webhook_url")
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("slack channel requires webhook_url configuration")
	}

	channel, _ := getStringConfig(config, "channel")
	username, _ := getStringConfig(config, "username")

	if username == "" {
		username = "Philotes Alerts"
	}

	return &SlackChannel{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "slack-channel"),
	}, nil
}

// Type returns the channel type.
func (c *SlackChannel) Type() alerting.ChannelType {
	return alerting.ChannelSlack
}

// Send sends a notification to Slack.
func (c *SlackChannel) Send(ctx context.Context, notification alerting.Notification) error {
	msg := c.buildMessage(notification)

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	c.logger.Debug("sending slack notification",
		"webhook_url", c.webhookURL,
		"channel", c.channel,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non-OK status %d: %s", resp.StatusCode, string(body))
	}

	// Slack returns "ok" as the response body on success
	if string(body) != "ok" {
		return fmt.Errorf("slack returned unexpected response: %s", string(body))
	}

	c.logger.Info("slack notification sent successfully",
		"rule_name", notification.Rule.Name,
		"event", notification.Event,
	)

	return nil
}

// Test sends a test notification to verify the channel configuration.
func (c *SlackChannel) Test(ctx context.Context) error {
	msg := slackMessage{
		Channel:  c.channel,
		Username: c.username,
		Attachments: []slackAttachment{
			{
				Fallback: "Test notification from Philotes",
				Color:    SeverityColor(alerting.SeverityInfo),
				Title:    "Test Notification",
				Text:     "This is a test notification from Philotes alerting system.",
				Footer:   "Philotes",
				Fields: []slackField{
					{
						Title: "Status",
						Value: "Test",
						Short: true,
					},
					{
						Title: "Time",
						Value: time.Now().Format(time.RFC3339),
						Short: true,
					},
				},
				Timestamp: time.Now().Unix(),
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal test message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack test failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildMessage builds a Slack message from a notification.
func (c *SlackChannel) buildMessage(notification alerting.Notification) slackMessage {
	title := FormatAlertTitle(notification)
	description := FormatAlertDescription(notification)

	severity := alerting.SeverityInfo
	if notification.Rule != nil {
		severity = notification.Rule.Severity
	}

	color := SeverityColor(severity)
	if notification.Event == alerting.EventResolved {
		color = "#28a745" // Green for resolved
	}

	fields := make([]slackField, 0)

	if notification.Rule != nil {
		fields = append(fields, slackField{
			Title: "Severity",
			Value: string(notification.Rule.Severity),
			Short: true,
		})
		fields = append(fields, slackField{
			Title: "Metric",
			Value: notification.Rule.MetricName,
			Short: true,
		})
	}

	if notification.Alert != nil && notification.Alert.CurrentValue != nil {
		fields = append(fields, slackField{
			Title: "Current Value",
			Value: fmt.Sprintf("%.2f", *notification.Alert.CurrentValue),
			Short: true,
		})
	}

	if notification.Rule != nil {
		fields = append(fields, slackField{
			Title: "Threshold",
			Value: fmt.Sprintf("%s %.2f", notification.Rule.Operator.String(), notification.Rule.Threshold),
			Short: true,
		})
	}

	if notification.Alert != nil {
		fields = append(fields, slackField{
			Title: "Fired At",
			Value: notification.Alert.FiredAt.Format(time.RFC3339),
			Short: true,
		})

		if notification.Alert.ResolvedAt != nil {
			fields = append(fields, slackField{
				Title: "Resolved At",
				Value: notification.Alert.ResolvedAt.Format(time.RFC3339),
				Short: true,
			})
		}
	}

	// Add labels as fields
	if notification.Alert != nil && len(notification.Alert.Labels) > 0 {
		labelsStr := ""
		for k, v := range notification.Alert.Labels {
			if labelsStr != "" {
				labelsStr += ", "
			}
			labelsStr += fmt.Sprintf("%s=%s", k, v)
		}
		fields = append(fields, slackField{
			Title: "Labels",
			Value: labelsStr,
			Short: false,
		})
	}

	timestamp := time.Now().Unix()
	if notification.Alert != nil {
		timestamp = notification.Alert.FiredAt.Unix()
	}

	return slackMessage{
		Channel:  c.channel,
		Username: c.username,
		Attachments: []slackAttachment{
			{
				Fallback:   title + ": " + description,
				Color:      color,
				Title:      title,
				Text:       description,
				Fields:     fields,
				Footer:     "Philotes Alerting",
				Timestamp:  timestamp,
				MarkdownIn: []string{"text"},
			},
		},
	}
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (c *SlackChannel) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// Ensure SlackChannel implements Channel interface.
var _ Channel = (*SlackChannel)(nil)
