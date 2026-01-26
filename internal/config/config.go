// Package config provides configuration loading and management for Philotes services.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for Philotes services.
type Config struct {
	// Version is the application version
	Version string

	// Environment is the deployment environment (development, staging, production)
	Environment string

	// API configuration
	API APIConfig

	// Database configuration for the buffer/metadata database
	Database DatabaseConfig

	// CDC configuration
	CDC CDCConfig

	// Iceberg configuration
	Iceberg IcebergConfig

	// MinIO/S3 configuration
	Storage StorageConfig

	// Metrics configuration
	Metrics MetricsConfig

	// Alerting configuration
	Alerting AlertingConfig
}

// APIConfig holds API server configuration.
type APIConfig struct {
	// ListenAddr is the address to listen on (e.g., ":8080")
	ListenAddr string

	// BaseURL is the external base URL of the API
	BaseURL string

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration

	// CORSOrigins is a list of allowed CORS origins (use "*" for all)
	CORSOrigins []string

	// RateLimitRPS is the rate limit in requests per second
	RateLimitRPS float64

	// RateLimitBurst is the maximum burst size for rate limiting
	RateLimitBurst int
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	// Host is the database host
	Host string

	// Port is the database port
	Port int

	// Name is the database name
	Name string

	// User is the database user
	User string

	// Password is the database password
	Password string

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
}

// DSN returns the database connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

// CDCConfig holds CDC pipeline configuration.
type CDCConfig struct {
	// BufferSize is the event buffer size
	BufferSize int

	// BatchSize is the batch size for flushing events
	BatchSize int

	// FlushInterval is the interval for flushing events
	FlushInterval time.Duration

	// Source is the source PostgreSQL database configuration
	Source SourceConfig

	// Replication holds replication slot and publication settings
	Replication ReplicationConfig

	// Checkpoint holds checkpointing configuration
	Checkpoint CheckpointConfig

	// Buffer holds buffer database configuration
	Buffer BufferConfig

	// Retry holds retry policy configuration
	Retry RetryConfig

	// DeadLetter holds dead-letter queue configuration
	DeadLetter DeadLetterConfig

	// Health holds health check configuration
	Health HealthConfig

	// Backpressure holds backpressure configuration
	Backpressure BackpressureConfig
}

// RetryConfig holds retry policy configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int

	// InitialInterval is the initial backoff interval
	InitialInterval time.Duration

	// MaxInterval is the maximum backoff interval
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier
	Multiplier float64
}

// DeadLetterConfig holds dead-letter queue configuration.
type DeadLetterConfig struct {
	// Enabled enables the dead-letter queue
	Enabled bool

	// Retention is how long to keep dead-letter events
	Retention time.Duration
}

// HealthConfig holds health check configuration.
type HealthConfig struct {
	// Enabled enables health check endpoints
	Enabled bool

	// ListenAddr is the address for health check endpoints
	ListenAddr string

	// ReadinessTimeout is how long to wait for readiness checks
	ReadinessTimeout time.Duration
}

// BackpressureConfig holds backpressure configuration.
type BackpressureConfig struct {
	// Enabled enables backpressure handling
	Enabled bool

	// HighWatermark is the buffer size threshold to trigger pause
	HighWatermark int

	// LowWatermark is the buffer size threshold to resume processing
	LowWatermark int

	// CheckInterval is how often to check buffer size
	CheckInterval time.Duration
}

// BufferConfig holds buffer database configuration.
type BufferConfig struct {
	// Enabled enables event buffering
	Enabled bool

	// Retention is how long to keep processed events before cleanup
	Retention time.Duration

	// CleanupInterval is how often to run the cleanup job
	CleanupInterval time.Duration
}

// SourceConfig holds the source PostgreSQL database configuration.
type SourceConfig struct {
	// Host is the source database host
	Host string

	// Port is the source database port
	Port int

	// Database is the source database name
	Database string

	// User is the source database user
	User string

	// Password is the source database password
	Password string

	// SSLMode is the SSL mode for the source connection
	SSLMode string
}

// DSN returns the source database connection string.
func (s SourceConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		s.Host, s.Port, s.Database, s.User, s.Password, s.SSLMode,
	)
}

// URL returns the source database connection URL.
func (s SourceConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		s.User, s.Password, s.Host, s.Port, s.Database, s.SSLMode,
	)
}

// ReplicationConfig holds replication slot and publication settings.
type ReplicationConfig struct {
	// SlotName is the name of the replication slot
	SlotName string

	// PublicationName is the name of the publication to subscribe to
	PublicationName string

	// Tables is a list of tables to replicate (empty means all tables in publication)
	Tables []string
}

// CheckpointConfig holds checkpointing configuration.
type CheckpointConfig struct {
	// Enabled enables checkpointing
	Enabled bool

	// Interval is the interval between checkpoints
	Interval time.Duration
}

// IcebergConfig holds Apache Iceberg configuration.
type IcebergConfig struct {
	// CatalogURL is the Lakekeeper REST catalog URL
	CatalogURL string

	// Warehouse is the warehouse name
	Warehouse string
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	// Endpoint is the S3/MinIO endpoint
	Endpoint string

	// AccessKey is the access key
	AccessKey string

	// SecretKey is the secret key
	SecretKey string

	// Bucket is the default bucket name
	Bucket string

	// UseSSL enables SSL for the connection
	UseSSL bool
}

// MetricsConfig holds metrics/observability configuration.
type MetricsConfig struct {
	// Enabled enables metrics collection
	Enabled bool

	// ListenAddr is the address for the metrics endpoint
	ListenAddr string
}

// AlertingConfig holds alerting framework configuration.
type AlertingConfig struct {
	// Enabled enables the alerting framework
	Enabled bool

	// EvaluationInterval is the interval between rule evaluations
	EvaluationInterval time.Duration

	// NotificationTimeout is the timeout for sending notifications
	NotificationTimeout time.Duration

	// PrometheusURL is the URL of the Prometheus server to query metrics from
	PrometheusURL string

	// RetentionDays is the number of days to retain alert history
	RetentionDays int
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Version:     getEnv("PHILOTES_VERSION", "0.1.0"),
		Environment: getEnv("PHILOTES_ENV", "development"),

		API: APIConfig{
			ListenAddr:     getEnv("PHILOTES_API_LISTEN_ADDR", ":8080"),
			BaseURL:        getEnv("PHILOTES_API_BASE_URL", "http://localhost:8080"),
			ReadTimeout:    getDurationEnv("PHILOTES_API_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:   getDurationEnv("PHILOTES_API_WRITE_TIMEOUT", 15*time.Second),
			CORSOrigins:    getSliceEnv("PHILOTES_API_CORS_ORIGINS", []string{"*"}),
			RateLimitRPS:   getFloatEnv("PHILOTES_API_RATE_LIMIT_RPS", 100),
			RateLimitBurst: getIntEnv("PHILOTES_API_RATE_LIMIT_BURST", 200),
		},

		Database: DatabaseConfig{
			Host:         getEnv("PHILOTES_DB_HOST", "localhost"),
			Port:         getIntEnv("PHILOTES_DB_PORT", 5432),
			Name:         getEnv("PHILOTES_DB_NAME", "philotes"),
			User:         getEnv("PHILOTES_DB_USER", "philotes"),
			Password:     getEnv("PHILOTES_DB_PASSWORD", "philotes"),
			SSLMode:      getEnv("PHILOTES_DB_SSLMODE", "disable"),
			MaxOpenConns: getIntEnv("PHILOTES_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getIntEnv("PHILOTES_DB_MAX_IDLE_CONNS", 5),
		},

		CDC: CDCConfig{
			BufferSize:    getIntEnv("PHILOTES_CDC_BUFFER_SIZE", 10000),
			BatchSize:     getIntEnv("PHILOTES_CDC_BATCH_SIZE", 1000),
			FlushInterval: getDurationEnv("PHILOTES_CDC_FLUSH_INTERVAL", 5*time.Second),
			Source: SourceConfig{
				Host:     getEnv("PHILOTES_CDC_SOURCE_HOST", "localhost"),
				Port:     getIntEnv("PHILOTES_CDC_SOURCE_PORT", 5433),
				Database: getEnv("PHILOTES_CDC_SOURCE_DATABASE", "source"),
				User:     getEnv("PHILOTES_CDC_SOURCE_USER", "source"),
				Password: getEnv("PHILOTES_CDC_SOURCE_PASSWORD", "source"),
				SSLMode:  getEnv("PHILOTES_CDC_SOURCE_SSLMODE", "disable"),
			},
			Replication: ReplicationConfig{
				SlotName:        getEnv("PHILOTES_CDC_REPLICATION_SLOT", "philotes_cdc"),
				PublicationName: getEnv("PHILOTES_CDC_PUBLICATION", "philotes_pub"),
				Tables:          getSliceEnv("PHILOTES_CDC_TABLES", nil),
			},
			Checkpoint: CheckpointConfig{
				Enabled:  getBoolEnv("PHILOTES_CDC_CHECKPOINT_ENABLED", true),
				Interval: getDurationEnv("PHILOTES_CDC_CHECKPOINT_INTERVAL", 10*time.Second),
			},
			Buffer: BufferConfig{
				Enabled:         getBoolEnv("PHILOTES_BUFFER_ENABLED", true),
				Retention:       getDurationEnv("PHILOTES_BUFFER_RETENTION", 168*time.Hour), // 7 days
				CleanupInterval: getDurationEnv("PHILOTES_BUFFER_CLEANUP_INTERVAL", time.Hour),
			},
			Retry: RetryConfig{
				MaxAttempts:     getIntEnv("PHILOTES_RETRY_MAX_ATTEMPTS", 3),
				InitialInterval: getDurationEnv("PHILOTES_RETRY_INITIAL_INTERVAL", time.Second),
				MaxInterval:     getDurationEnv("PHILOTES_RETRY_MAX_INTERVAL", 30*time.Second),
				Multiplier:      getFloatEnv("PHILOTES_RETRY_MULTIPLIER", 2.0),
			},
			DeadLetter: DeadLetterConfig{
				Enabled:   getBoolEnv("PHILOTES_DLQ_ENABLED", true),
				Retention: getDurationEnv("PHILOTES_DLQ_RETENTION", 168*time.Hour), // 7 days
			},
			Health: HealthConfig{
				Enabled:          getBoolEnv("PHILOTES_HEALTH_ENABLED", true),
				ListenAddr:       getEnv("PHILOTES_HEALTH_LISTEN_ADDR", ":8081"),
				ReadinessTimeout: getDurationEnv("PHILOTES_HEALTH_READINESS_TIMEOUT", 5*time.Second),
			},
			Backpressure: BackpressureConfig{
				Enabled:       getBoolEnv("PHILOTES_BACKPRESSURE_ENABLED", true),
				HighWatermark: getIntEnv("PHILOTES_BACKPRESSURE_HIGH_WATERMARK", 8000),
				LowWatermark:  getIntEnv("PHILOTES_BACKPRESSURE_LOW_WATERMARK", 5000),
				CheckInterval: getDurationEnv("PHILOTES_BACKPRESSURE_CHECK_INTERVAL", time.Second),
			},
		},

		Iceberg: IcebergConfig{
			CatalogURL: getEnv("PHILOTES_ICEBERG_CATALOG_URL", "http://localhost:8181"),
			Warehouse:  getEnv("PHILOTES_ICEBERG_WAREHOUSE", "philotes"),
		},

		Storage: StorageConfig{
			Endpoint:  getEnv("PHILOTES_STORAGE_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("PHILOTES_STORAGE_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("PHILOTES_STORAGE_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("PHILOTES_STORAGE_BUCKET", "philotes"),
			UseSSL:    getBoolEnv("PHILOTES_STORAGE_USE_SSL", false),
		},

		Metrics: MetricsConfig{
			Enabled:    getBoolEnv("PHILOTES_METRICS_ENABLED", true),
			ListenAddr: getEnv("PHILOTES_METRICS_LISTEN_ADDR", ":9090"),
		},

		Alerting: AlertingConfig{
			Enabled:             getBoolEnv("PHILOTES_ALERTING_ENABLED", true),
			EvaluationInterval:  getDurationEnv("PHILOTES_ALERTING_EVALUATION_INTERVAL", 30*time.Second),
			NotificationTimeout: getDurationEnv("PHILOTES_ALERTING_NOTIFICATION_TIMEOUT", 10*time.Second),
			PrometheusURL:       getEnv("PHILOTES_PROMETHEUS_URL", "http://localhost:9090"),
			RetentionDays:       getIntEnv("PHILOTES_ALERTING_RETENTION_DAYS", 30),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var result []string
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
