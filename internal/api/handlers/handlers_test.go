package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler_GetHealth_NoManager(t *testing.T) {
	handler := NewHealthHandler(nil)

	router := gin.New()
	router.GET("/health", handler.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

func TestHealthHandler_GetHealth_WithManager(t *testing.T) {
	manager := health.NewManager(health.DefaultManagerConfig(), nil)
	manager.Register(health.NewComponentChecker("test", func(ctx context.Context) (health.Status, string, error) {
		return health.StatusHealthy, "test is healthy", nil
	}))

	handler := NewHealthHandler(manager)

	router := gin.New()
	router.GET("/health", handler.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

	if _, ok := response.Components["test"]; !ok {
		t.Error("expected 'test' component in response")
	}
}

func TestHealthHandler_GetLiveness(t *testing.T) {
	handler := NewHealthHandler(nil)

	router := gin.New()
	router.GET("/health/live", handler.GetLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

func TestHealthHandler_GetReadiness_NoManager(t *testing.T) {
	handler := NewHealthHandler(nil)

	router := gin.New()
	router.GET("/health/ready", handler.GetReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

func TestVersionHandler_GetVersion(t *testing.T) {
	handler := NewVersionHandler("1.2.3")

	router := gin.New()
	router.GET("/version", handler.GetVersion)

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.VersionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", response.Version)
	}

	if response.APIVersion != "v1" {
		t.Errorf("expected api_version 'v1', got '%s'", response.APIVersion)
	}

	if response.GoVersion == "" {
		t.Error("expected go_version to be set")
	}
}

func TestConfigHandler_GetConfig(t *testing.T) {
	cfg := &config.Config{
		Environment: "test",
		API: config.APIConfig{
			ListenAddr: ":8080",
			BaseURL:    "http://localhost:8080",
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

	handler := NewConfigHandler(cfg)

	router := gin.New()
	router.GET("/config", handler.GetConfig)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

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

	if response.CDC.BufferSize != 10000 {
		t.Errorf("expected buffer_size 10000, got %d", response.CDC.BufferSize)
	}
}
