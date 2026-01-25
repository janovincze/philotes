// Package main provides the entry point for the Philotes Management API service.
// The API provides REST endpoints for managing CDC pipelines, sources, and destinations.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/janovincze/philotes/internal/api"
	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/api/services"
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

	// Initialize database connection
	db, err := sql.Open("pgx", cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// Verify database connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()
	if err := db.PingContext(dbCtx); err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// Create repositories
	sourceRepo := repositories.NewSourceRepository(db)
	pipelineRepo := repositories.NewPipelineRepository(db)

	// Create services
	sourceService := services.NewSourceService(sourceRepo, logger)
	pipelineService := services.NewPipelineService(pipelineRepo, sourceRepo, logger)

	// Create health manager
	healthManager := health.NewManager(health.DefaultManagerConfig(), logger)

	// Register health checkers
	healthManager.Register(health.NewComponentChecker("api", func(ctx context.Context) (health.Status, string, error) {
		return health.StatusHealthy, "API server is running", nil
	}))
	healthManager.Register(health.NewComponentChecker("database", func(ctx context.Context) (health.Status, string, error) {
		if err := db.PingContext(ctx); err != nil {
			return health.StatusUnhealthy, "database connection failed", err
		}
		return health.StatusHealthy, "database connection OK", nil
	}))

	// Create server configuration
	serverCfg := api.ServerConfig{
		Config:          cfg,
		Logger:          logger,
		HealthManager:   healthManager,
		SourceService:   sourceService,
		PipelineService: pipelineService,
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
