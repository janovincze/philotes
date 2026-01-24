// Package config provides configuration loading and management for Philotes services.
package config

import (
	"fmt"
	"os"
	"strconv"
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

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Version:     getEnv("PHILOTES_VERSION", "0.1.0"),
		Environment: getEnv("PHILOTES_ENV", "development"),

		API: APIConfig{
			ListenAddr:   getEnv("PHILOTES_API_LISTEN_ADDR", ":8080"),
			BaseURL:      getEnv("PHILOTES_API_BASE_URL", "http://localhost:8080"),
			ReadTimeout:  getDurationEnv("PHILOTES_API_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("PHILOTES_API_WRITE_TIMEOUT", 15*time.Second),
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
