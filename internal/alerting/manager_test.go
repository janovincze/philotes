package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/config"
)

// mockRepository is a mock implementation of AlertRepository for testing.
type mockRepository struct {
	rules          []AlertRule
	instances      []AlertInstance
	silences       []AlertSilence
	channels       []NotificationChannel
	routes         []AlertRoute
	histories      []AlertHistory
	getInstanceErr error
	listRulesErr   error
}

func (m *mockRepository) ListRules(ctx context.Context, enabledOnly bool) ([]AlertRule, error) {
	if m.listRulesErr != nil {
		return nil, m.listRulesErr
	}
	if enabledOnly {
		var result []AlertRule
		for _, r := range m.rules {
			if r.Enabled {
				result = append(result, r)
			}
		}
		return result, nil
	}
	return m.rules, nil
}

func (m *mockRepository) GetRule(ctx context.Context, id uuid.UUID) (*AlertRule, error) {
	for _, r := range m.rules {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("rule not found")
}

func (m *mockRepository) CreateInstance(ctx context.Context, instance *AlertInstance) (*AlertInstance, error) {
	instance.ID = uuid.New()
	instance.CreatedAt = time.Now()
	instance.UpdatedAt = time.Now()
	m.instances = append(m.instances, *instance)
	return instance, nil
}

func (m *mockRepository) GetInstanceByFingerprint(ctx context.Context, ruleID uuid.UUID, fingerprint string) (*AlertInstance, error) {
	if m.getInstanceErr != nil {
		return nil, m.getInstanceErr
	}
	for _, i := range m.instances {
		if i.RuleID == ruleID && i.Fingerprint == fingerprint {
			return &i, nil
		}
	}
	return nil, fmt.Errorf("instance not found")
}

func (m *mockRepository) ListInstances(ctx context.Context, status *AlertStatus, ruleID *uuid.UUID) ([]AlertInstance, error) {
	var result []AlertInstance
	for _, i := range m.instances {
		if status != nil && i.Status != *status {
			continue
		}
		if ruleID != nil && i.RuleID != *ruleID {
			continue
		}
		result = append(result, i)
	}
	return result, nil
}

func (m *mockRepository) UpdateInstance(ctx context.Context, id uuid.UUID, status AlertStatus, currentValue *float64, resolvedAt *time.Time) error {
	for i := range m.instances {
		if m.instances[i].ID == id {
			m.instances[i].Status = status
			m.instances[i].CurrentValue = currentValue
			m.instances[i].ResolvedAt = resolvedAt
			m.instances[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("instance not found")
}

func (m *mockRepository) CreateHistory(ctx context.Context, history *AlertHistory) (*AlertHistory, error) {
	history.ID = uuid.New()
	history.CreatedAt = time.Now()
	m.histories = append(m.histories, *history)
	return history, nil
}

func (m *mockRepository) ListSilences(ctx context.Context, activeOnly bool) ([]AlertSilence, error) {
	if activeOnly {
		var result []AlertSilence
		for _, s := range m.silences {
			if s.IsActive() {
				result = append(result, s)
			}
		}
		return result, nil
	}
	return m.silences, nil
}

func (m *mockRepository) GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error) {
	for _, c := range m.channels {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("channel not found")
}

func (m *mockRepository) ListChannels(ctx context.Context, enabledOnly bool) ([]NotificationChannel, error) {
	if enabledOnly {
		var result []NotificationChannel
		for _, c := range m.channels {
			if c.Enabled {
				result = append(result, c)
			}
		}
		return result, nil
	}
	return m.channels, nil
}

func (m *mockRepository) ListRoutes(ctx context.Context, ruleID *uuid.UUID, enabledOnly bool) ([]AlertRoute, error) {
	var result []AlertRoute
	for _, r := range m.routes {
		if ruleID != nil && r.RuleID != *ruleID {
			continue
		}
		if enabledOnly && !r.Enabled {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.AlertingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.AlertingConfig{
				PrometheusURL:      "http://localhost:9090",
				EvaluationInterval: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing prometheus URL",
			cfg: config.AlertingConfig{
				EvaluationInterval: 30 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{}
			m, err := NewManager(repo, tt.cfg, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if m == nil {
					t.Fatal("NewManager() returned nil")
				}
				if m.evaluator == nil {
					t.Error("evaluator should not be nil")
				}
				if m.notifier == nil {
					t.Error("notifier should not be nil")
				}
				if m.pendingAlerts == nil {
					t.Error("pendingAlerts should be initialized")
				}
			}
		})
	}
}

func TestManager_GetPendingAlerts(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 30 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Initially empty
	pending := m.GetPendingAlerts()
	if len(pending) != 0 {
		t.Errorf("expected no pending alerts, got %d", len(pending))
	}

	// Add a pending alert directly
	fp := "test-fingerprint"
	now := time.Now()
	m.mu.Lock()
	m.pendingAlerts[fp] = now
	m.mu.Unlock()

	// Should now have one pending alert
	pending = m.GetPendingAlerts()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending alert, got %d", len(pending))
	}
	if !pending[fp].Equal(now) {
		t.Errorf("pending alert time mismatch")
	}

	// Verify it's a copy (modifying returned map shouldn't affect internal state)
	delete(pending, fp)
	pending2 := m.GetPendingAlerts()
	if len(pending2) != 1 {
		t.Error("GetPendingAlerts should return a copy")
	}
}

func TestManager_IsRunning(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 100 * time.Millisecond,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Initially not running
	if m.IsRunning() {
		t.Error("manager should not be running initially")
	}

	// Start
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !m.IsRunning() {
		t.Error("manager should be running after Start")
	}

	// Stop
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if m.IsRunning() {
		t.Error("manager should not be running after Stop")
	}
}

func TestManager_StartTwice(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 100 * time.Millisecond,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("First Start() error = %v", err)
	}
	defer func() { _ = m.Stop() }() //nolint:errcheck

	// Second start should return error
	if err := m.Start(ctx); err == nil {
		t.Error("Second Start() should return error")
	}
}

func TestManager_StopWhenNotRunning(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 100 * time.Millisecond,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Stop without start should not error
	if err := m.Stop(); err != nil {
		t.Errorf("Stop() when not running should not error: %v", err)
	}
}

func TestManager_PendingAlertTracking(t *testing.T) {
	ruleID := uuid.New()

	// Create a test server that returns a firing metric
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := prometheusResponse{
			Status: "success",
			Data: struct {
				ResultType string             `json:"resultType"`
				Result     []prometheusResult `json:"result"`
			}{
				ResultType: "vector",
				Result: []prometheusResult{
					{
						Metric: map[string]string{"source": "db1"},
						Value:  []interface{}{float64(time.Now().Unix()), "100"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	repo := &mockRepository{
		rules: []AlertRule{
			{
				ID:              ruleID,
				Name:            "test-rule",
				MetricName:      "test_metric",
				Operator:        OpGreaterThan,
				Threshold:       50,
				DurationSeconds: 60, // 60 second duration
				Enabled:         true,
			},
		},
		getInstanceErr: fmt.Errorf("not found"), // No existing instance
	}

	cfg := config.AlertingConfig{
		PrometheusURL:      server.URL,
		EvaluationInterval: 10 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()

	// First evaluation should add to pending
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("EvaluateNow() error = %v", err)
	}

	pending := m.GetPendingAlerts()
	if len(pending) != 1 {
		t.Errorf("expected 1 pending alert after first evaluation, got %d", len(pending))
	}

	// Alert should not be created yet (duration not met)
	if len(repo.instances) != 0 {
		t.Errorf("expected no instances (duration not met), got %d", len(repo.instances))
	}
}

func TestManager_AlertFiringAfterDuration(t *testing.T) {
	ruleID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := prometheusResponse{
			Status: "success",
			Data: struct {
				ResultType string             `json:"resultType"`
				Result     []prometheusResult `json:"result"`
			}{
				ResultType: "vector",
				Result: []prometheusResult{
					{
						Metric: map[string]string{"source": "db1"},
						Value:  []interface{}{float64(time.Now().Unix()), "100"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	repo := &mockRepository{
		rules: []AlertRule{
			{
				ID:              ruleID,
				Name:            "test-rule",
				MetricName:      "test_metric",
				Operator:        OpGreaterThan,
				Threshold:       50,
				DurationSeconds: 0, // No duration requirement
				Enabled:         true,
			},
		},
		getInstanceErr: fmt.Errorf("not found"),
	}

	cfg := config.AlertingConfig{
		PrometheusURL:      server.URL,
		EvaluationInterval: 10 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()

	// First evaluation adds to pending
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("First EvaluateNow() error = %v", err)
	}

	// Second evaluation should fire (duration is 0)
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("Second EvaluateNow() error = %v", err)
	}

	// Alert should be created
	if len(repo.instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(repo.instances))
	}

	if repo.instances[0].Status != StatusFiring {
		t.Errorf("expected status %q, got %q", StatusFiring, repo.instances[0].Status)
	}

	// Should have history entry
	if len(repo.histories) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(repo.histories))
	}
}

func TestManager_AlertNotFiringClearsPending(t *testing.T) {
	ruleID := uuid.New()
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var value string
		if callCount == 1 {
			value = "100" // First call: firing
		} else {
			value = "30" // Subsequent calls: not firing
		}

		response := prometheusResponse{
			Status: "success",
			Data: struct {
				ResultType string             `json:"resultType"`
				Result     []prometheusResult `json:"result"`
			}{
				ResultType: "vector",
				Result: []prometheusResult{
					{
						Metric: map[string]string{"source": "db1"},
						Value:  []interface{}{float64(time.Now().Unix()), value},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	repo := &mockRepository{
		rules: []AlertRule{
			{
				ID:              ruleID,
				Name:            "test-rule",
				MetricName:      "test_metric",
				Operator:        OpGreaterThan,
				Threshold:       50,
				DurationSeconds: 60, // Long duration
				Enabled:         true,
			},
		},
		getInstanceErr: fmt.Errorf("not found"),
	}

	cfg := config.AlertingConfig{
		PrometheusURL:      server.URL,
		EvaluationInterval: 10 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()

	// First evaluation: firing, should add to pending
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("First EvaluateNow() error = %v", err)
	}

	pending := m.GetPendingAlerts()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending alert, got %d", len(pending))
	}

	// Second evaluation: not firing, should clear pending
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("Second EvaluateNow() error = %v", err)
	}

	pending = m.GetPendingAlerts()
	if len(pending) != 0 {
		t.Errorf("expected 0 pending alerts after not firing, got %d", len(pending))
	}
}

func TestManager_SilencedAlert(t *testing.T) {
	ruleID := uuid.New()
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := prometheusResponse{
			Status: "success",
			Data: struct {
				ResultType string             `json:"resultType"`
				Result     []prometheusResult `json:"result"`
			}{
				ResultType: "vector",
				Result: []prometheusResult{
					{
						Metric: map[string]string{"source": "db1"},
						Value:  []interface{}{float64(time.Now().Unix()), "100"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	repo := &mockRepository{
		rules: []AlertRule{
			{
				ID:              ruleID,
				Name:            "test-rule",
				MetricName:      "test_metric",
				Operator:        OpGreaterThan,
				Threshold:       50,
				DurationSeconds: 0,
				Enabled:         true,
			},
		},
		silences: []AlertSilence{
			{
				ID:       uuid.New(),
				Matchers: map[string]string{"source": "db1"},
				StartsAt: now.Add(-1 * time.Hour),
				EndsAt:   now.Add(1 * time.Hour),
			},
		},
		getInstanceErr: fmt.Errorf("not found"),
	}

	cfg := config.AlertingConfig{
		PrometheusURL:      server.URL,
		EvaluationInterval: 10 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()

	// First evaluation adds to pending
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("First EvaluateNow() error = %v", err)
	}

	// Second evaluation should try to fire but be silenced
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("Second EvaluateNow() error = %v", err)
	}

	// No instance should be created due to silence
	if len(repo.instances) != 0 {
		t.Errorf("expected 0 instances (silenced), got %d", len(repo.instances))
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 100 * time.Millisecond,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context should stop the evaluation loop
	cancel()

	// Wait for the evaluation loop to exit (stoppedCh closes)
	// Note: The manager's running flag is not updated on context cancellation,
	// only when Stop() is called. This is expected behavior - context cancellation
	// stops the evaluation loop but the manager still needs explicit Stop() to
	// clean up the running state properly. Here we just verify the loop exits.
	select {
	case <-m.stoppedCh:
		// Loop exited as expected
	case <-time.After(1 * time.Second):
		t.Error("evaluation loop should exit when context is cancelled")
	}
}

func TestManager_NoEnabledRules(t *testing.T) {
	ruleID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Prometheus should not be queried when no enabled rules")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &mockRepository{
		rules: []AlertRule{
			{
				ID:              ruleID,
				Name:            "disabled-rule",
				MetricName:      "test_metric",
				Operator:        OpGreaterThan,
				Threshold:       50,
				DurationSeconds: 0,
				Enabled:         false, // Disabled
			},
		},
	}

	cfg := config.AlertingConfig{
		PrometheusURL:      server.URL,
		EvaluationInterval: 10 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()

	// Should complete without error and without calling Prometheus
	err = m.EvaluateNow(ctx)
	if err != nil {
		t.Fatalf("EvaluateNow() error = %v", err)
	}
}

func TestManager_Accessors(t *testing.T) {
	repo := &mockRepository{}
	cfg := config.AlertingConfig{
		PrometheusURL:      "http://localhost:9090",
		EvaluationInterval: 30 * time.Second,
	}

	m, err := NewManager(repo, cfg, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.Evaluator() == nil {
		t.Error("Evaluator() should not return nil")
	}

	if m.Notifier() == nil {
		t.Error("Notifier() should not return nil")
	}
}
