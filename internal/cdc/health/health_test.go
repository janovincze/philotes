package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	cfg := DefaultManagerConfig()
	mgr := NewManager(cfg, nil)

	if mgr == nil {
		t.Fatal("expected manager to be created")
	}
	if mgr.timeout != cfg.Timeout {
		t.Errorf("expected timeout %v, got %v", cfg.Timeout, mgr.timeout)
	}
}

func TestManager_Register(t *testing.T) {
	mgr := NewManager(DefaultManagerConfig(), nil)

	checker := NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
		return StatusHealthy, "ok", nil
	})

	mgr.Register(checker)

	if len(mgr.checkers) != 1 {
		t.Errorf("expected 1 checker, got %d", len(mgr.checkers))
	}
}

func TestManager_CheckAll(t *testing.T) {
	mgr := NewManager(DefaultManagerConfig(), nil)

	mgr.Register(NewComponentChecker("healthy", func(ctx context.Context) (Status, string, error) {
		return StatusHealthy, "ok", nil
	}))
	mgr.Register(NewComponentChecker("unhealthy", func(ctx context.Context) (Status, string, error) {
		return StatusUnhealthy, "failed", errors.New("test error")
	}))

	results := mgr.CheckAll(context.Background())

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	healthyResult, ok := results["healthy"]
	if !ok {
		t.Fatal("expected healthy result")
	}
	if healthyResult.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %v", healthyResult.Status)
	}

	unhealthyResult, ok := results["unhealthy"]
	if !ok {
		t.Fatal("expected unhealthy result")
	}
	if unhealthyResult.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %v", unhealthyResult.Status)
	}
	if unhealthyResult.Error != "test error" {
		t.Errorf("expected error message, got %q", unhealthyResult.Error)
	}
}

func TestManager_IsHealthy(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		healthy bool
	}{
		{"all healthy", StatusHealthy, true},
		{"degraded", StatusDegraded, true},
		{"unhealthy", StatusUnhealthy, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(DefaultManagerConfig(), nil)
			mgr.Register(NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
				return tt.status, "", nil
			}))

			if got := mgr.IsHealthy(context.Background()); got != tt.healthy {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.healthy)
			}
		})
	}
}

func TestManager_GetOverallStatus(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []Status
		expectedStatus Status
	}{
		{"all healthy", []Status{StatusHealthy, StatusHealthy}, StatusHealthy},
		{"one degraded", []Status{StatusHealthy, StatusDegraded}, StatusDegraded},
		{"one unhealthy", []Status{StatusHealthy, StatusUnhealthy}, StatusUnhealthy},
		{"degraded and unhealthy", []Status{StatusDegraded, StatusUnhealthy}, StatusUnhealthy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewManager(DefaultManagerConfig(), nil)
			for i, status := range tt.statuses {
				s := status // capture loop variable
				mgr.Register(NewComponentChecker(
					string(rune('a'+i)),
					func(ctx context.Context) (Status, string, error) {
						return s, "", nil
					},
				))
			}

			overall := mgr.GetOverallStatus(context.Background())
			if overall.Status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, overall.Status)
			}
		})
	}
}

func TestDatabaseChecker(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		checker := NewDatabaseChecker("test-db", func(ctx context.Context) error {
			return nil
		})

		if checker.Name() != "test-db" {
			t.Errorf("expected name test-db, got %s", checker.Name())
		}

		result := checker.Check(context.Background())
		if result.Status != StatusHealthy {
			t.Errorf("expected healthy status, got %v", result.Status)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		checker := NewDatabaseChecker("test-db", func(ctx context.Context) error {
			return errors.New("connection failed")
		})

		result := checker.Check(context.Background())
		if result.Status != StatusUnhealthy {
			t.Errorf("expected unhealthy status, got %v", result.Status)
		}
		if result.Error != "connection failed" {
			t.Errorf("expected error message, got %q", result.Error)
		}
	})
}

func TestComponentChecker(t *testing.T) {
	checker := NewComponentChecker("test-component", func(ctx context.Context) (Status, string, error) {
		return StatusDegraded, "high load", nil
	})

	if checker.Name() != "test-component" {
		t.Errorf("expected name test-component, got %s", checker.Name())
	}

	result := checker.Check(context.Background())
	if result.Status != StatusDegraded {
		t.Errorf("expected degraded status, got %v", result.Status)
	}
	if result.Message != "high load" {
		t.Errorf("expected message 'high load', got %q", result.Message)
	}
}

func TestServer_Liveness(t *testing.T) {
	mgr := NewManager(DefaultManagerConfig(), nil)
	server := NewServer(mgr, DefaultServerConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	server.handleLiveness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "alive") {
		t.Errorf("expected body to contain 'alive', got %q", body)
	}
}

func TestServer_Readiness(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		mgr := NewManager(DefaultManagerConfig(), nil)
		mgr.Register(NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
			return StatusHealthy, "ok", nil
		}))
		server := NewServer(mgr, DefaultServerConfig(), nil)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		server.handleReadiness(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("not ready", func(t *testing.T) {
		mgr := NewManager(DefaultManagerConfig(), nil)
		mgr.Register(NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
			return StatusUnhealthy, "failed", errors.New("not ready")
		}))
		server := NewServer(mgr, DefaultServerConfig(), nil)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		w := httptest.NewRecorder()

		server.handleReadiness(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}

func TestServer_Health(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		mgr := NewManager(DefaultManagerConfig(), nil)
		mgr.Register(NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
			return StatusHealthy, "ok", nil
		}))
		server := NewServer(mgr, DefaultServerConfig(), nil)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		mgr := NewManager(DefaultManagerConfig(), nil)
		mgr.Register(NewComponentChecker("test", func(ctx context.Context) (Status, string, error) {
			return StatusUnhealthy, "failed", nil
		}))
		server := NewServer(mgr, DefaultServerConfig(), nil)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}

func TestDefaultManagerConfig(t *testing.T) {
	cfg := DefaultManagerConfig()
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cfg.Timeout)
	}
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	if cfg.ListenAddr != ":8081" {
		t.Errorf("expected listen addr :8081, got %s", cfg.ListenAddr)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("expected write timeout 10s, got %v", cfg.WriteTimeout)
	}
}
