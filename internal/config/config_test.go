package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "0.1.0" {
		t.Errorf("Version = %v, want %v", cfg.Version, "0.1.0")
	}

	if cfg.Environment != "development" {
		t.Errorf("Environment = %v, want %v", cfg.Environment, "development")
	}

	if cfg.API.ListenAddr != ":8080" {
		t.Errorf("API.ListenAddr = %v, want %v", cfg.API.ListenAddr, ":8080")
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("Database.Port = %v, want %v", cfg.Database.Port, 5432)
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("PHILOTES_VERSION", "1.0.0")
	os.Setenv("PHILOTES_ENV", "production")
	os.Setenv("PHILOTES_DB_HOST", "db.example.com")
	os.Setenv("PHILOTES_DB_PORT", "5433")
	defer func() {
		os.Unsetenv("PHILOTES_VERSION")
		os.Unsetenv("PHILOTES_ENV")
		os.Unsetenv("PHILOTES_DB_HOST")
		os.Unsetenv("PHILOTES_DB_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %v, want %v", cfg.Version, "1.0.0")
	}

	if cfg.Environment != "production" {
		t.Errorf("Environment = %v, want %v", cfg.Environment, "production")
	}

	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %v, want %v", cfg.Database.Host, "db.example.com")
	}

	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port = %v, want %v", cfg.Database.Port, 5433)
	}
}

func TestDatabaseDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"
	if got := cfg.DSN(); got != expected {
		t.Errorf("DSN() = %v, want %v", got, expected)
	}
}

func TestGetDurationEnv(t *testing.T) {
	os.Setenv("TEST_DURATION", "30s")
	defer os.Unsetenv("TEST_DURATION")

	got := getDurationEnv("TEST_DURATION", 10*time.Second)
	if got != 30*time.Second {
		t.Errorf("getDurationEnv() = %v, want %v", got, 30*time.Second)
	}

	// Test default
	got = getDurationEnv("NONEXISTENT", 10*time.Second)
	if got != 10*time.Second {
		t.Errorf("getDurationEnv() = %v, want %v", got, 10*time.Second)
	}
}

func TestGetBoolEnv(t *testing.T) {
	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")

	got := getBoolEnv("TEST_BOOL", false)
	if got != true {
		t.Errorf("getBoolEnv() = %v, want %v", got, true)
	}

	// Test default
	got = getBoolEnv("NONEXISTENT", false)
	if got != false {
		t.Errorf("getBoolEnv() = %v, want %v", got, false)
	}
}
