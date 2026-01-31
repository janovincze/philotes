// Package alerting provides the alerting framework for Philotes.
package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/janovincze/philotes/internal/config"
)

// Manager is the main alert manager that coordinates rule evaluation and notifications.
type Manager struct {
	repo      AlertRepository
	evaluator *Evaluator
	notifier  *Notifier
	logger    *slog.Logger
	config    config.AlertingConfig

	// Track pending alerts for duration-based evaluation
	pendingAlerts map[string]time.Time // fingerprint -> first triggered time
	mu            sync.RWMutex

	// Control channels
	stopCh    chan struct{}
	stoppedCh chan struct{}
	running   bool
	runMu     sync.Mutex
}

// NewManager creates a new alert manager.
func NewManager(repo AlertRepository, cfg config.AlertingConfig, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.PrometheusURL == "" {
		return nil, fmt.Errorf("prometheus URL is required")
	}

	evaluator := NewEvaluator(cfg.PrometheusURL, logger)
	notifier := NewNotifier(repo, nil, cfg.NotificationTimeout, logger)

	return &Manager{
		repo:          repo,
		evaluator:     evaluator,
		notifier:      notifier,
		logger:        logger.With("component", "alert-manager"),
		config:        cfg,
		pendingAlerts: make(map[string]time.Time),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}, nil
}

// SetChannelFactory sets the channel factory for the notifier.
// This should be called before Start() if custom channel implementations are needed.
func (m *Manager) SetChannelFactory(factory ChannelFactory) {
	m.notifier.channelFactory = factory
}

// Start starts the alert manager evaluation loop.
func (m *Manager) Start(ctx context.Context) error {
	m.runMu.Lock()
	if m.running {
		m.runMu.Unlock()
		return fmt.Errorf("manager is already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.stoppedCh = make(chan struct{})
	m.runMu.Unlock()

	m.logger.Info("starting alert manager",
		"evaluation_interval", m.config.EvaluationInterval,
		"prometheus_url", m.config.PrometheusURL,
	)

	go m.evaluationLoop(ctx)

	return nil
}

// Stop gracefully stops the alert manager.
func (m *Manager) Stop() error {
	m.runMu.Lock()
	if !m.running {
		m.runMu.Unlock()
		return nil
	}
	m.runMu.Unlock()

	m.logger.Info("stopping alert manager")

	close(m.stopCh)

	// Wait for the evaluation loop to finish
	<-m.stoppedCh

	m.runMu.Lock()
	m.running = false
	m.runMu.Unlock()

	m.logger.Info("alert manager stopped")

	return nil
}

// evaluationLoop runs the periodic rule evaluation.
func (m *Manager) evaluationLoop(ctx context.Context) {
	defer close(m.stoppedCh)

	ticker := time.NewTicker(m.config.EvaluationInterval)
	defer ticker.Stop()

	// Run initial evaluation
	m.runEvaluation(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context cancelled, stopping evaluation loop")
			return
		case <-m.stopCh:
			m.logger.Info("stop signal received, stopping evaluation loop")
			return
		case <-ticker.C:
			m.runEvaluation(ctx)
		}
	}
}

// runEvaluation performs a single evaluation cycle.
func (m *Manager) runEvaluation(ctx context.Context) {
	m.logger.Debug("starting evaluation cycle")
	start := time.Now()

	if err := m.evaluateRules(ctx); err != nil {
		m.logger.Error("evaluation cycle failed", "error", err)
	}

	m.logger.Debug("evaluation cycle completed",
		"duration", time.Since(start),
	)
}

// evaluateRules evaluates all enabled alert rules.
func (m *Manager) evaluateRules(ctx context.Context) error {
	// Load all enabled rules
	rules, err := m.repo.ListRules(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	if len(rules) == 0 {
		m.logger.Debug("no enabled rules to evaluate")
		return nil
	}

	m.logger.Debug("evaluating rules", "count", len(rules))

	// Track which fingerprints we've seen in this cycle for resolution detection
	seenFingerprints := make(map[string]bool)

	for _, rule := range rules {
		results, err := m.evaluator.Evaluate(ctx, rule)
		if err != nil {
			m.logger.Error("failed to evaluate rule",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err,
			)
			continue
		}

		for _, result := range results {
			fingerprint := GenerateFingerprint(rule.ID, result.Labels)
			seenFingerprints[fingerprint] = true

			if err := m.processEvaluation(ctx, rule, result, fingerprint); err != nil {
				m.logger.Error("failed to process evaluation",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"error", err,
				)
			}
		}
	}

	// Check for alerts that should be resolved (no longer firing)
	if err := m.checkForResolutions(ctx, seenFingerprints); err != nil {
		m.logger.Error("failed to check for resolutions", "error", err)
	}

	return nil
}

// processEvaluation processes a single evaluation result.
func (m *Manager) processEvaluation(ctx context.Context, rule AlertRule, result EvaluationResult, fingerprint string) error {
	if result.ShouldFire {
		return m.handleFiring(ctx, rule, result, fingerprint)
	}
	return m.handleNotFiring(ctx, rule, fingerprint)
}

// handleFiring handles a rule that is firing.
func (m *Manager) handleFiring(ctx context.Context, rule AlertRule, result EvaluationResult, fingerprint string) error {
	m.mu.Lock()
	firstTriggered, isPending := m.pendingAlerts[fingerprint]
	if !isPending {
		m.pendingAlerts[fingerprint] = time.Now()
		m.mu.Unlock()
		m.logger.Debug("alert pending, waiting for duration",
			"rule_name", rule.Name,
			"fingerprint", fingerprint,
			"duration_seconds", rule.DurationSeconds,
		)
		return nil
	}
	m.mu.Unlock()

	// Check if the duration has been exceeded
	duration := time.Duration(rule.DurationSeconds) * time.Second
	if time.Since(firstTriggered) < duration {
		m.logger.Debug("alert still pending",
			"rule_name", rule.Name,
			"fingerprint", fingerprint,
			"pending_for", time.Since(firstTriggered),
			"required_duration", duration,
		)
		return nil
	}

	// Duration exceeded, fire the alert
	return m.fireAlert(ctx, rule, result, fingerprint)
}

// handleNotFiring handles a rule that is not firing.
func (m *Manager) handleNotFiring(ctx context.Context, rule AlertRule, fingerprint string) error {
	m.mu.Lock()
	_, isPending := m.pendingAlerts[fingerprint]
	if isPending {
		delete(m.pendingAlerts, fingerprint)
		m.logger.Debug("cleared pending alert",
			"rule_name", rule.Name,
			"fingerprint", fingerprint,
		)
	}
	m.mu.Unlock()

	return nil
}

// fireAlert creates and fires an alert.
func (m *Manager) fireAlert(ctx context.Context, rule AlertRule, result EvaluationResult, fingerprint string) error {
	m.logger.Info("firing alert",
		"rule_name", rule.Name,
		"fingerprint", fingerprint,
		"value", result.Value,
		"threshold", rule.Threshold,
	)

	// Check if this alert is silenced
	silenced, err := m.checkSilenced(ctx, result.Labels)
	if err != nil {
		m.logger.Warn("failed to check silences", "error", err)
	}
	if silenced {
		m.logger.Info("alert is silenced, skipping notification",
			"rule_name", rule.Name,
			"fingerprint", fingerprint,
		)
		return nil
	}

	// Check if an instance already exists
	existing, err := m.repo.GetInstanceByFingerprint(ctx, rule.ID, fingerprint)
	if err != nil {
		// Check if this is a "not found" error (expected when creating new alerts)
		if !isNotFoundError(err) {
			return fmt.Errorf("failed to check existing alert instance: %w", err)
		}
		// Instance not found, proceed to create new one
	} else if existing != nil && existing.Status == StatusFiring {
		// Update the existing instance with new value
		if err := m.repo.UpdateInstance(ctx, existing.ID, StatusFiring, &result.Value, nil); err != nil {
			m.logger.Warn("failed to update existing alert instance", "error", err)
		}
		// Send notification (respecting repeat interval)
		return m.notifier.Notify(ctx, *existing, rule, EventFired)
	}

	// Create a new alert instance
	instance := &AlertInstance{
		RuleID:       rule.ID,
		Fingerprint:  fingerprint,
		Status:       StatusFiring,
		Labels:       result.Labels,
		Annotations:  rule.Annotations,
		CurrentValue: &result.Value,
		FiredAt:      time.Now(),
	}

	created, err := m.repo.CreateInstance(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create alert instance: %w", err)
	}

	// Record in history
	if _, err := m.repo.CreateHistory(ctx, &AlertHistory{
		AlertID:   created.ID,
		RuleID:    rule.ID,
		EventType: EventFired,
		Message:   fmt.Sprintf("Alert fired: %s %s %.2f", rule.MetricName, rule.Operator.String(), result.Value),
		Value:     &result.Value,
	}); err != nil {
		m.logger.Warn("failed to create alert history", "error", err)
	}

	// Clear from pending
	m.mu.Lock()
	delete(m.pendingAlerts, fingerprint)
	m.mu.Unlock()

	// Send notification
	return m.notifier.Notify(ctx, *created, rule, EventFired)
}

// resolveAlert resolves an alert instance.
func (m *Manager) resolveAlert(ctx context.Context, instance AlertInstance) error {
	m.logger.Info("resolving alert",
		"alert_id", instance.ID,
		"fingerprint", instance.Fingerprint,
	)

	now := time.Now()
	if err := m.repo.UpdateInstance(ctx, instance.ID, StatusResolved, nil, &now); err != nil {
		return fmt.Errorf("failed to update alert instance: %w", err)
	}

	// Get the rule
	rule, err := m.repo.GetRule(ctx, instance.RuleID)
	if err != nil {
		m.logger.Warn("failed to get rule for resolved alert", "error", err)
		return nil
	}

	// Record in history
	if _, err := m.repo.CreateHistory(ctx, &AlertHistory{
		AlertID:   instance.ID,
		RuleID:    instance.RuleID,
		EventType: EventResolved,
		Message:   "Alert resolved",
	}); err != nil {
		m.logger.Warn("failed to create alert history", "error", err)
	}

	// Clear notification tracking
	m.notifier.ClearLastNotified(instance.Fingerprint)

	// Update instance for notification
	instance.Status = StatusResolved
	instance.ResolvedAt = &now

	// Send notification
	return m.notifier.Notify(ctx, instance, *rule, EventResolved)
}

// checkForResolutions checks for alerts that should be resolved.
func (m *Manager) checkForResolutions(ctx context.Context, seenFingerprints map[string]bool) error {
	// Get all firing alerts
	status := StatusFiring
	firingAlerts, err := m.repo.ListInstances(ctx, &status, nil)
	if err != nil {
		return fmt.Errorf("failed to list firing alerts: %w", err)
	}

	for _, alert := range firingAlerts {
		// If this fingerprint wasn't seen in the current evaluation, resolve it
		if !seenFingerprints[alert.Fingerprint] {
			if err := m.resolveAlert(ctx, alert); err != nil {
				m.logger.Error("failed to resolve alert",
					"alert_id", alert.ID,
					"error", err,
				)
			}
		}
	}

	return nil
}

// checkSilenced checks if an alert should be silenced based on its labels.
func (m *Manager) checkSilenced(ctx context.Context, labels map[string]string) (bool, error) {
	silences, err := m.repo.ListSilences(ctx, true)
	if err != nil {
		return false, fmt.Errorf("failed to list silences: %w", err)
	}

	for _, silence := range silences {
		if silence.IsActive() && silence.Matches(labels) {
			return true, nil
		}
	}

	return false, nil
}

// EvaluateNow triggers an immediate evaluation of all rules.
// This is useful for testing or when rules have been updated.
func (m *Manager) EvaluateNow(ctx context.Context) error {
	return m.evaluateRules(ctx)
}

// GetPendingAlerts returns the currently pending alerts.
func (m *Manager) GetPendingAlerts() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range m.pendingAlerts {
		result[k] = v
	}
	return result
}

// IsRunning returns whether the manager is running.
func (m *Manager) IsRunning() bool {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	return m.running
}

// Evaluator returns the underlying evaluator (useful for testing).
func (m *Manager) Evaluator() *Evaluator {
	return m.evaluator
}

// Notifier returns the underlying notifier (useful for testing).
func (m *Manager) Notifier() *Notifier {
	return m.notifier
}

// isNotFoundError checks if an error indicates a "not found" condition.
// This is used to distinguish between "instance doesn't exist" (expected) and
// actual errors like database connection failures.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "no rows")
}
