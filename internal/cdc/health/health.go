// Package health provides health check functionality for CDC components.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is healthy.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is unhealthy.
	StatusUnhealthy Status = "unhealthy"
	// StatusDegraded indicates the component is degraded but functional.
	StatusDegraded Status = "degraded"
	// StatusUnknown indicates the health status is unknown.
	StatusUnknown Status = "unknown"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	// Name is the name of the component.
	Name string `json:"name"`

	// Status is the health status.
	Status Status `json:"status"`

	// Message provides additional details.
	Message string `json:"message,omitempty"`

	// Duration is how long the check took.
	Duration time.Duration `json:"duration_ms"`

	// LastCheck is when the check was last performed.
	LastCheck time.Time `json:"last_check"`

	// Error is the error message if the check failed.
	Error string `json:"error,omitempty"`
}

// Checker is a function that performs a health check.
type Checker func(ctx context.Context) CheckResult

// HealthChecker defines the interface for health check providers.
type HealthChecker interface {
	// Check performs the health check.
	Check(ctx context.Context) CheckResult

	// Name returns the name of the component.
	Name() string
}

// Manager manages health checks for multiple components.
type Manager struct {
	mu       sync.RWMutex
	checkers []HealthChecker
	results  map[string]CheckResult
	logger   *slog.Logger
	timeout  time.Duration
}

// ManagerConfig holds configuration for the health manager.
type ManagerConfig struct {
	// Timeout is the timeout for individual health checks.
	Timeout time.Duration
}

// DefaultManagerConfig returns a ManagerConfig with sensible defaults.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		Timeout: 5 * time.Second,
	}
}

// NewManager creates a new health manager.
func NewManager(cfg ManagerConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{
		checkers: make([]HealthChecker, 0),
		results:  make(map[string]CheckResult),
		logger:   logger.With("component", "health-manager"),
		timeout:  cfg.Timeout,
	}
}

// Register adds a health checker to the manager.
func (m *Manager) Register(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
	m.logger.Debug("registered health checker", "name", checker.Name())
}

// CheckAll performs health checks on all registered components.
func (m *Manager) CheckAll(ctx context.Context) map[string]CheckResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	results := make(map[string]CheckResult)

	for _, checker := range m.checkers {
		checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
		result := checker.Check(checkCtx)
		cancel()

		results[checker.Name()] = result
		m.results[checker.Name()] = result
	}

	return results
}

// GetResult returns the last result for a specific checker.
func (m *Manager) GetResult(name string) (CheckResult, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result, ok := m.results[name]
	return result, ok
}

// IsHealthy returns true if all components are healthy.
func (m *Manager) IsHealthy(ctx context.Context) bool {
	results := m.CheckAll(ctx)
	for _, result := range results {
		if result.Status != StatusHealthy && result.Status != StatusDegraded {
			return false
		}
	}
	return true
}

// IsReady returns true if the system is ready to serve requests.
func (m *Manager) IsReady(ctx context.Context) bool {
	return m.IsHealthy(ctx)
}

// OverallStatus computes the overall health status.
type OverallStatus struct {
	// Status is the overall status.
	Status Status `json:"status"`

	// Components contains individual component results.
	Components map[string]CheckResult `json:"components"`

	// Timestamp is when the status was computed.
	Timestamp time.Time `json:"timestamp"`
}

// GetOverallStatus returns the overall health status.
func (m *Manager) GetOverallStatus(ctx context.Context) OverallStatus {
	results := m.CheckAll(ctx)

	overall := OverallStatus{
		Status:     StatusHealthy,
		Components: results,
		Timestamp:  time.Now(),
	}

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			overall.Status = StatusUnhealthy
			return overall
		case StatusDegraded:
			if overall.Status == StatusHealthy {
				overall.Status = StatusDegraded
			}
		case StatusUnknown:
			if overall.Status == StatusHealthy {
				overall.Status = StatusUnknown
			}
		}
	}

	return overall
}

// Server provides HTTP endpoints for health checks.
type Server struct {
	manager *Manager
	logger  *slog.Logger
	server  *http.Server
}

// ServerConfig holds configuration for the health server.
type ServerConfig struct {
	// ListenAddr is the address to listen on.
	ListenAddr string

	// ReadTimeout is the read timeout for HTTP requests.
	ReadTimeout time.Duration

	// WriteTimeout is the write timeout for HTTP responses.
	WriteTimeout time.Duration
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ListenAddr:   ":8081",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// NewServer creates a new health server.
func NewServer(manager *Manager, cfg ServerConfig, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		manager: manager,
		logger:  logger.With("component", "health-server"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)

	s.server = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return s
}

// Start starts the health server.
func (s *Server) Start() error {
	s.logger.Info("starting health server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the health server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping health server")
	return s.server.Shutdown(ctx)
}

// handleHealth returns the overall health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := s.manager.GetOverallStatus(r.Context())

	w.Header().Set("Content-Type", "application/json")

	if status.Status == StatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else if status.Status == StatusDegraded {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.logger.Error("failed to encode health response", "error", err)
	}
}

// handleLiveness returns whether the service is alive.
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness is always true if the server is responding
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"alive","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleReadiness returns whether the service is ready.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.manager.IsReady(r.Context()) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	}
}

// DatabaseChecker checks database connectivity.
type DatabaseChecker struct {
	name string
	ping func(ctx context.Context) error
}

// NewDatabaseChecker creates a new database health checker.
func NewDatabaseChecker(name string, ping func(ctx context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{name: name, ping: ping}
}

// Name returns the name of the component.
func (c *DatabaseChecker) Name() string {
	return c.name
}

// Check performs the health check.
func (c *DatabaseChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      c.name,
		LastCheck: start,
	}

	err := c.ping(ctx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		result.Message = "database connection failed"
	} else {
		result.Status = StatusHealthy
		result.Message = "database connection successful"
	}

	return result
}

// ComponentChecker checks a generic component.
type ComponentChecker struct {
	name  string
	check func(ctx context.Context) (Status, string, error)
}

// NewComponentChecker creates a new component health checker.
func NewComponentChecker(name string, check func(ctx context.Context) (Status, string, error)) *ComponentChecker {
	return &ComponentChecker{name: name, check: check}
}

// Name returns the name of the component.
func (c *ComponentChecker) Name() string {
	return c.name
}

// Check performs the health check.
func (c *ComponentChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:      c.name,
		LastCheck: start,
	}

	status, message, err := c.check(ctx)
	result.Duration = time.Since(start)
	result.Status = status
	result.Message = message

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

// Ensure interfaces are implemented.
var (
	_ HealthChecker = (*DatabaseChecker)(nil)
	_ HealthChecker = (*ComponentChecker)(nil)
)
