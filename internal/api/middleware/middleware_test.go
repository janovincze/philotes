package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_GeneratesID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get(RequestIDHeader)
	if requestID == "" {
		t.Error("expected X-Request-ID to be generated")
	}

	// Should be a valid UUID format (36 chars with dashes)
	if len(requestID) != 36 {
		t.Errorf("expected UUID format, got length %d", len(requestID))
	}
}

func TestRequestID_UsesProvidedID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, "my-custom-id")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get(RequestIDHeader)
	if requestID != "my-custom-id" {
		t.Errorf("expected 'my-custom-id', got '%s'", requestID)
	}
}

func TestRecovery_RecoversPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	router := gin.New()
	router.Use(Recovery(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Errorf("expected Content-Type 'application/problem+json', got '%s'", contentType)
	}
}

func TestRateLimiter_AllowsRequests(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
		PerClient:         false,
	}

	router := gin.New()
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First request should be allowed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLogger_LogsRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	router := gin.New()
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCORS_ReturnsMiddleware(t *testing.T) {
	// Test that CORS returns a valid middleware function
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false,
	}

	middleware := CORS(cfg)
	if middleware == nil {
		t.Error("expected CORS middleware to be non-nil")
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected default allowed origins to be ['*'], got %v", cfg.AllowedOrigins)
	}

	if cfg.AllowCredentials != false {
		t.Error("expected default allow credentials to be false")
	}

	if cfg.MaxAge == 0 {
		t.Error("expected default max age to be non-zero")
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if cfg.RequestsPerSecond != 100 {
		t.Errorf("expected RequestsPerSecond 100, got %f", cfg.RequestsPerSecond)
	}

	if cfg.BurstSize != 200 {
		t.Errorf("expected BurstSize 200, got %d", cfg.BurstSize)
	}

	if !cfg.PerClient {
		t.Error("expected PerClient to be true")
	}

	if cfg.ClientTTL == 0 {
		t.Error("expected ClientTTL to be non-zero")
	}

	if cfg.CleanupInterval == 0 {
		t.Error("expected CleanupInterval to be non-zero")
	}
}

func TestRateLimiter_PerClient(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
		PerClient:         true,
	}

	router := gin.New()
	router.Use(RateLimiter(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request should be allowed and include rate limit header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check rate limit header is set
	rateLimitHeader := w.Header().Get("X-RateLimit-Limit")
	if rateLimitHeader != "100" {
		t.Errorf("expected X-RateLimit-Limit '100', got '%s'", rateLimitHeader)
	}
}
