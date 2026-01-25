package models

import "time"

// VersionResponse contains version information.
type VersionResponse struct {
	Version    string `json:"version"`
	APIVersion string `json:"api_version"`
	GoVersion  string `json:"go_version,omitempty"`
	BuildTime  string `json:"build_time,omitempty"`
	GitCommit  string `json:"git_commit,omitempty"`
}

// ConfigResponse contains safe configuration information.
type ConfigResponse struct {
	Environment string       `json:"environment"`
	API         APIConfig    `json:"api"`
	CDC         CDCConfig    `json:"cdc,omitempty"`
	Metrics     MetricConfig `json:"metrics,omitempty"`
}

// APIConfig contains API configuration (safe subset).
type APIConfig struct {
	ListenAddr string `json:"listen_addr"`
	BaseURL    string `json:"base_url"`
}

// CDCConfig contains CDC configuration (safe subset).
type CDCConfig struct {
	BufferSize    int    `json:"buffer_size"`
	BatchSize     int    `json:"batch_size"`
	FlushInterval string `json:"flush_interval"`
}

// MetricConfig contains metrics configuration.
type MetricConfig struct {
	Enabled    bool   `json:"enabled"`
	ListenAddr string `json:"listen_addr"`
}

// HealthResponse represents the overall health status.
type HealthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
	Timestamp  time.Time                  `json:"timestamp"`
}

// ComponentHealth represents the health of a single component.
type ComponentHealth struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	DurationMs int64     `json:"duration_ms"`
	LastCheck  time.Time `json:"last_check"`
	Error      string    `json:"error,omitempty"`
}

// LivenessResponse represents the liveness probe response.
type LivenessResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadinessResponse represents the readiness probe response.
type ReadinessResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}
