// Package main provides the entry point for the Philotes Management API service.
// The API provides REST endpoints for managing CDC pipelines, sources, and destinations.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/janovincze/philotes/internal/api"
	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/config"
)

func main() {
	// Setup structured logging
	logLevel := slog.LevelInfo
	if os.Getenv("PHILOTES_LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("starting Philotes API",
		"version", cfg.Version,
		"environment", cfg.Environment,
		"listen_addr", cfg.API.ListenAddr,
	)

	// Create health manager
	healthManager := health.NewManager(health.DefaultManagerConfig(), logger)

	// Register a basic API health checker
	healthManager.Register(health.NewComponentChecker("api", func(ctx context.Context) (health.Status, string, error) {
		return health.StatusHealthy, "API server is running", nil
	}))

	// Create server configuration
	serverCfg := api.ServerConfig{
		Config:        cfg,
		Logger:        logger,
		HealthManager: healthManager,
		CORSConfig: middleware.CORSConfig{
			AllowedOrigins:   cfg.API.CORSOrigins,
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		},
		RateLimitConfig: middleware.RateLimitConfig{
			RequestsPerSecond: cfg.API.RateLimitRPS,
			BurstSize:         cfg.API.RateLimitBurst,
			PerClient:         true,
		},
	}

	// Create and start server
	server := api.NewServer(serverCfg)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		logger.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		logger.Error("server error", "error", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop server gracefully", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
