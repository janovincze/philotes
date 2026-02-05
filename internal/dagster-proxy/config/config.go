// Package config provides configuration for the Dagster proxy service.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the Dagster proxy configuration.
type Config struct {
	// ListenAddr is the address to listen on (e.g., ":8080")
	ListenAddr string

	// DagsterUpstreamURL is the URL of the Dagster webserver
	DagsterUpstreamURL string

	// Database configuration
	Database DatabaseConfig

	// Auth configuration
	Auth AuthConfig

	// LogLevel is the logging level (debug, info, warn, error)
	LogLevel string

	// RequestTimeout is the timeout for proxied requests
	RequestTimeout time.Duration

	// Version is the service version
	Version string

	// Environment is the deployment environment
	Environment string
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	Database    string
	SSLMode     string
	MaxOpenConn int
	MaxIdleConn int
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Enabled      bool
	JWTSecret    string
	APIKeyPrefix string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:         getEnv("DAGSTER_PROXY_LISTEN_ADDR", ":8080"),
		DagsterUpstreamURL: getEnv("DAGSTER_PROXY_UPSTREAM_URL", "http://dagster-webserver:3000"),
		LogLevel:           getEnv("PHILOTES_LOG_LEVEL", "info"),
		RequestTimeout:     getDurationEnv("DAGSTER_PROXY_REQUEST_TIMEOUT", 60*time.Second),
		Version:            getEnv("PHILOTES_VERSION", "dev"),
		Environment:        getEnv("PHILOTES_ENVIRONMENT", "development"),

		Database: DatabaseConfig{
			Host:        getEnv("PHILOTES_DATABASE_HOST", "localhost"),
			Port:        getIntEnv("PHILOTES_DATABASE_PORT", 5432),
			User:        getEnv("PHILOTES_DATABASE_USER", "philotes"),
			Password:    getEnv("PHILOTES_DATABASE_PASSWORD", "philotes"),
			Database:    getEnv("PHILOTES_DATABASE_NAME", "philotes"),
			SSLMode:     getEnv("PHILOTES_DATABASE_SSLMODE", "disable"),
			MaxOpenConn: getIntEnv("PHILOTES_DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConn: getIntEnv("PHILOTES_DATABASE_MAX_IDLE_CONNS", 5),
		},

		Auth: AuthConfig{
			Enabled:      getBoolEnv("PHILOTES_AUTH_ENABLED", true),
			JWTSecret:    getEnv("PHILOTES_AUTH_JWT_SECRET", ""),
			APIKeyPrefix: getEnv("PHILOTES_AUTH_API_KEY_PREFIX", "pk_"),
		},
	}

	// Validate required configuration
	if cfg.DagsterUpstreamURL == "" {
		return nil, fmt.Errorf("DAGSTER_PROXY_UPSTREAM_URL is required")
	}

	if cfg.Auth.Enabled && len(cfg.Auth.JWTSecret) < 32 {
		return nil, fmt.Errorf("PHILOTES_AUTH_JWT_SECRET must be at least 32 characters when auth is enabled")
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
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
