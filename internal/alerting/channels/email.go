package channels

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"strings"
	"time"

	"github.com/janovincze/philotes/internal/alerting"
)

// EmailChannel implements the Channel interface for SMTP email.
type EmailChannel struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
	to       []string
	useTLS   bool
	logger   *slog.Logger
}

// NewEmailChannel creates a new email notification channel.
func NewEmailChannel(config map[string]interface{}, logger *slog.Logger) (*EmailChannel, error) {
	smtpHost, ok := getStringConfig(config, "smtp_host")
	if !ok || smtpHost == "" {
		return nil, fmt.Errorf("email channel requires smtp_host configuration")
	}

	smtpPort, ok := getIntConfig(config, "smtp_port")
	if !ok {
		smtpPort = 587 // Default to submission port
	}

	from, ok := getStringConfig(config, "from")
	if !ok || from == "" {
		return nil, fmt.Errorf("email channel requires from configuration")
	}

	to, ok := getStringSliceConfig(config, "to")
	if !ok || len(to) == 0 {
		return nil, fmt.Errorf("email channel requires to configuration with at least one recipient")
	}

	username, _ := getStringConfig(config, "username")
	password, _ := getStringConfig(config, "password")

	useTLS := true
	if v, ok := getBoolConfig(config, "use_tls"); ok {
		useTLS = v
	}

	return &EmailChannel{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
		to:       to,
		useTLS:   useTLS,
		logger:   logger.With("component", "email-channel"),
	}, nil
}

// Type returns the channel type.
func (c *EmailChannel) Type() alerting.ChannelType {
	return alerting.ChannelEmail
}

// Send sends a notification via email.
func (c *EmailChannel) Send(ctx context.Context, notification alerting.Notification) error {
	subject := FormatAlertTitle(notification)
	body, err := c.buildHTMLBody(notification)
	if err != nil {
		return fmt.Errorf("failed to build email body: %w", err)
	}

	c.logger.Debug("sending email notification",
		"smtp_host", c.smtpHost,
		"from", c.from,
		"to", c.to,
	)

	if err := c.sendEmail(ctx, subject, body); err != nil {
		return err
	}

	c.logger.Info("email notification sent successfully",
		"rule_name", notification.Rule.Name,
		"event", notification.Event,
		"recipients", c.to,
	)

	return nil
}

// Test sends a test email to verify the channel configuration.
func (c *EmailChannel) Test(ctx context.Context) error {
	subject := "[TEST] Philotes Alert Test"
	body, err := c.buildTestEmailBody()
	if err != nil {
		return fmt.Errorf("failed to build test email body: %w", err)
	}
	return c.sendEmail(ctx, subject, body)
}

// buildTestEmailBody builds the HTML body for a test email using template.
func (c *EmailChannel) buildTestEmailBody() (string, error) {
	const testEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 0; background-color: #f4f4f4; }
        .container { max-width: 600px; margin: 20px auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { background-color: #17a2b8; color: white; padding: 20px; }
        .header h2 { margin: 0; }
        .content { padding: 20px; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Test Notification</h2>
        </div>
        <div class="content">
            <p>This is a test notification from Philotes alerting system.</p>
            <p>If you received this email, your email channel configuration is working correctly.</p>
            <p><small>Sent at: {{.SentAt}}</small></p>
        </div>
        <div class="footer">
            Philotes Alerting System
        </div>
    </div>
</body>
</html>
`

	data := struct {
		SentAt string
	}{
		SentAt: time.Now().Format(time.RFC3339),
	}

	tmpl, err := template.New("test-email").Parse(testEmailTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse test email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute test email template: %w", err)
	}

	return buf.String(), nil
}

// sendEmail sends an email using SMTP.
func (c *EmailChannel) sendEmail(ctx context.Context, subject, body string) error {
	// Build the email message
	message := c.buildMessage(subject, body)

	// Create a channel to receive the result
	done := make(chan error, 1)

	go func() {
		var err error
		addr := fmt.Sprintf("%s:%d", c.smtpHost, c.smtpPort)

		if c.useTLS {
			err = c.sendWithTLS(addr, message)
		} else {
			err = c.sendPlain(addr, message)
		}
		done <- err
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// sendWithTLS sends email using STARTTLS.
func (c *EmailChannel) sendWithTLS(addr string, message []byte) error {
	// Connect to the server
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Say hello
	if err := conn.Hello("localhost"); err != nil {
		return fmt.Errorf("HELO failed: %w", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: c.smtpHost,
	}
	if err := conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	// Authenticate if credentials are provided
	if c.username != "" && c.password != "" {
		auth := smtp.PlainAuth("", c.username, c.password, c.smtpHost)
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set the sender
	if err := conn.Mail(c.from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set the recipients
	for _, recipient := range c.to {
		if err := conn.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	// Send the message body
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return conn.Quit()
}

// sendPlain sends email without TLS (not recommended for production).
func (c *EmailChannel) sendPlain(addr string, message []byte) error {
	var auth smtp.Auth
	if c.username != "" && c.password != "" {
		auth = smtp.PlainAuth("", c.username, c.password, c.smtpHost)
	}

	return smtp.SendMail(addr, auth, c.from, c.to, message)
}

// buildMessage builds the raw email message.
func (c *EmailChannel) buildMessage(subject, body string) []byte {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", c.from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(c.to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("\r\n")

	// Body
	buf.WriteString(body)

	return buf.Bytes()
}

// buildHTMLBody builds an HTML email body from a notification.
func (c *EmailChannel) buildHTMLBody(notification alerting.Notification) (string, error) {
	const emailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 0; background-color: #f4f4f4; }
        .container { max-width: 600px; margin: 20px auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { background-color: {{.HeaderColor}}; color: white; padding: 20px; }
        .header h2 { margin: 0; }
        .content { padding: 20px; }
        .field { margin-bottom: 15px; }
        .field-label { font-weight: bold; color: #666; font-size: 12px; text-transform: uppercase; }
        .field-value { margin-top: 5px; color: #333; }
        .labels { background-color: #f8f9fa; padding: 10px; border-radius: 4px; margin-top: 15px; }
        .label-item { display: inline-block; background-color: #e9ecef; padding: 2px 8px; border-radius: 3px; margin: 2px; font-size: 12px; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #666; }
        .status-badge { display: inline-block; padding: 4px 12px; border-radius: 4px; font-weight: bold; }
        .status-firing { background-color: #dc3545; color: white; }
        .status-resolved { background-color: #28a745; color: white; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>{{.Title}}</h2>
        </div>
        <div class="content">
            <div class="field">
                <div class="field-label">Status</div>
                <div class="field-value">
                    <span class="status-badge {{.StatusClass}}">{{.Status}}</span>
                </div>
            </div>

            {{if .Description}}
            <div class="field">
                <div class="field-label">Description</div>
                <div class="field-value">{{.Description}}</div>
            </div>
            {{end}}

            <div class="field">
                <div class="field-label">Severity</div>
                <div class="field-value">{{.Severity}}</div>
            </div>

            <div class="field">
                <div class="field-label">Metric</div>
                <div class="field-value">{{.Metric}}</div>
            </div>

            {{if .CurrentValue}}
            <div class="field">
                <div class="field-label">Current Value</div>
                <div class="field-value">{{.CurrentValue}}</div>
            </div>
            {{end}}

            <div class="field">
                <div class="field-label">Threshold</div>
                <div class="field-value">{{.Threshold}}</div>
            </div>

            <div class="field">
                <div class="field-label">Fired At</div>
                <div class="field-value">{{.FiredAt}}</div>
            </div>

            {{if .ResolvedAt}}
            <div class="field">
                <div class="field-label">Resolved At</div>
                <div class="field-value">{{.ResolvedAt}}</div>
            </div>
            {{end}}

            {{if .Labels}}
            <div class="labels">
                <div class="field-label">Labels</div>
                {{range $key, $value := .Labels}}
                <span class="label-item">{{$key}}={{$value}}</span>
                {{end}}
            </div>
            {{end}}
        </div>
        <div class="footer">
            Philotes Alerting System
        </div>
    </div>
</body>
</html>
`

	data := c.buildTemplateData(notification)

	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse email template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// emailTemplateData holds data for the email template.
type emailTemplateData struct {
	Title        string
	HeaderColor  string
	Status       string
	StatusClass  string
	Description  string
	Severity     string
	Metric       string
	CurrentValue string
	Threshold    string
	FiredAt      string
	ResolvedAt   string
	Labels       map[string]string
}

// buildTemplateData builds template data from a notification.
func (c *EmailChannel) buildTemplateData(notification alerting.Notification) emailTemplateData {
	data := emailTemplateData{
		Title:       FormatAlertTitle(notification),
		HeaderColor: SeverityColor(alerting.SeverityInfo),
		Status:      "FIRING",
		StatusClass: "status-firing",
		Labels:      make(map[string]string),
	}

	if notification.Event == alerting.EventResolved {
		data.Status = "RESOLVED"
		data.StatusClass = "status-resolved"
		data.HeaderColor = "#28a745"
	}

	if notification.Rule != nil {
		data.HeaderColor = SeverityColor(notification.Rule.Severity)
		data.Description = notification.Rule.Description
		data.Severity = string(notification.Rule.Severity)
		data.Metric = notification.Rule.MetricName
		data.Threshold = fmt.Sprintf("%s %.2f", notification.Rule.Operator.String(), notification.Rule.Threshold)
	}

	if notification.Alert != nil {
		if notification.Alert.CurrentValue != nil {
			data.CurrentValue = fmt.Sprintf("%.2f", *notification.Alert.CurrentValue)
		}
		data.FiredAt = notification.Alert.FiredAt.Format(time.RFC3339)
		if notification.Alert.ResolvedAt != nil {
			data.ResolvedAt = notification.Alert.ResolvedAt.Format(time.RFC3339)
		}
		data.Labels = notification.Alert.Labels
	}

	return data
}

// Ensure EmailChannel implements Channel interface.
var _ Channel = (*EmailChannel)(nil)
