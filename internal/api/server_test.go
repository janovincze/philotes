package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/config"
)

func TestServer_HealthEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response.Status)
	}
}

func TestServer_LivenessEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.LivenessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "alive" {
		t.Errorf("expected status 'alive', got '%s'", response.Status)
	}
}

func TestServer_ReadinessEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ReadinessResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ready" {
		t.Errorf("expected status 'ready', got '%s'", response.Status)
	}
}

func TestServer_VersionEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.VersionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Version != "0.1.0-test" {
		t.Errorf("expected version '0.1.0-test', got '%s'", response.Version)
	}

	if response.APIVersion != "v1" {
		t.Errorf("expected api_version 'v1', got '%s'", response.APIVersion)
	}
}

func TestServer_ConfigEndpoint(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Environment != "test" {
		t.Errorf("expected environment 'test', got '%s'", response.Environment)
	}

	if response.API.ListenAddr != ":8080" {
		t.Errorf("expected listen_addr ':8080', got '%s'", response.API.ListenAddr)
	}
}

func TestServer_EndpointsWithoutServices(t *testing.T) {
	// When no services are configured, source/pipeline endpoints are not registered
	// and should return 404
	server := newTestServer(t)

	endpoints := []string{
		"/api/v1/sources",
		"/api/v1/pipelines",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			server.Router().ServeHTTP(w, req)

			// Without services configured, routes are not registered
			if w.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
			}
		})
	}
}

func TestServer_RequestID(t *testing.T) {
	server := newTestServer(t)

	// Test without X-Request-ID header (should generate one)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	// Test with X-Request-ID header (should use provided value)
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	w = httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	requestID = w.Header().Get("X-Request-ID")
	if requestID != "test-request-id" {
		t.Errorf("expected X-Request-ID 'test-request-id', got '%s'", requestID)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress logs during tests
	}))

	cfg := &config.Config{
		Version:     "0.1.0-test",
		Environment: "test",
		API: config.APIConfig{
			ListenAddr:     ":8080",
			BaseURL:        "http://localhost:8080",
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			CORSOrigins:    []string{"*"},
			RateLimitRPS:   100,
			RateLimitBurst: 200,
		},
		CDC: config.CDCConfig{
			BufferSize:    10000,
			BatchSize:     1000,
			FlushInterval: 5 * time.Second,
		},
		Metrics: config.MetricsConfig{
			Enabled:    true,
			ListenAddr: ":9090",
		},
	}

	serverCfg := DefaultServerConfig(cfg, logger)

	return NewServer(serverCfg)
}
