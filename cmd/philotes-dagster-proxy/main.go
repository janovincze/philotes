// Package main provides the entry point for the Philotes Dagster Proxy service.
// The proxy provides RBAC for Dagster OSS by intercepting GraphQL requests.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/api/services"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/config"
	proxyconfig "github.com/janovincze/philotes/internal/dagster-proxy/config"
	"github.com/janovincze/philotes/internal/dagster-proxy/handler"
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

	// Load proxy configuration
	proxyCfg, err := proxyconfig.Load()
	if err != nil {
		logger.Error("failed to load proxy config", "error", err)
		os.Exit(1)
	}

	logger.Info("starting Philotes Dagster Proxy",
		"version", proxyCfg.Version,
		"environment", proxyCfg.Environment,
		"listen_addr", proxyCfg.ListenAddr,
		"upstream_url", proxyCfg.DagsterUpstreamURL,
	)

	// Parse upstream URL
	upstreamURL, err := url.Parse(proxyCfg.DagsterUpstreamURL)
	if err != nil {
		logger.Error("failed to parse upstream URL", "error", err)
		os.Exit(1)
	}

	// Initialize database connection
	db, err := sql.Open("pgx", proxyCfg.Database.DSN())
	if err != nil {
		logger.Error("failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(proxyCfg.Database.MaxOpenConn)
	db.SetMaxIdleConns(proxyCfg.Database.MaxIdleConn)

	// Verify database connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(dbCtx); err != nil {
		dbCancel()
		db.Close()
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	dbCancel()
	logger.Info("database connection established")

	// Create repositories
	userRepo := repositories.NewUserRepository(db)
	apiKeyRepo := repositories.NewAPIKeyRepository(db)
	auditRepo := repositories.NewAuditRepository(db)

	// Create auth services (only if auth is enabled)
	var authService *services.AuthService
	var apiKeyService *services.APIKeyService

	if proxyCfg.Auth.Enabled {
		// Load main config for auth settings
		mainCfg, err := config.Load()
		if err != nil {
			logger.Warn("failed to load main config, using proxy config for auth", "error", err)
			// Create minimal auth config from proxy config
			authCfg := &config.AuthConfig{
				Enabled:      proxyCfg.Auth.Enabled,
				JWTSecret:    proxyCfg.Auth.JWTSecret,
				APIKeyPrefix: proxyCfg.Auth.APIKeyPrefix,
			}
			authService = services.NewAuthService(userRepo, auditRepo, authCfg, logger)
			apiKeyService = services.NewAPIKeyService(apiKeyRepo, userRepo, auditRepo, authCfg, logger)
		} else {
			authService = services.NewAuthService(userRepo, auditRepo, &mainCfg.Auth, logger)
			apiKeyService = services.NewAPIKeyService(apiKeyRepo, userRepo, auditRepo, &mainCfg.Auth, logger)
		}
	}

	// Create health manager
	healthManager := health.NewManager(health.DefaultManagerConfig(), logger)

	// Register health checkers
	healthManager.Register(health.NewComponentChecker("proxy", func(ctx context.Context) (health.Status, string, error) {
		return health.StatusHealthy, "Dagster proxy is running", nil
	}))
	healthManager.Register(health.NewComponentChecker("database", func(ctx context.Context) (health.Status, string, error) {
		if err := db.PingContext(ctx); err != nil {
			return health.StatusUnhealthy, "database connection failed", err
		}
		return health.StatusHealthy, "database connection OK", nil
	}))

	// Create server configuration
	serverCfg := handler.ServerConfig{
		ListenAddr:     proxyCfg.ListenAddr,
		UpstreamURL:    upstreamURL,
		Logger:         logger,
		HealthManager:  healthManager,
		AuthService:    authService,
		APIKeyService:  apiKeyService,
		AuditRepo:      auditRepo,
		AuthEnabled:    proxyCfg.Auth.Enabled,
		APIKeyPrefix:   proxyCfg.Auth.APIKeyPrefix,
		RequestTimeout: proxyCfg.RequestTimeout,
	}

	// Create and start server
	server := handler.NewServer(serverCfg)

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
	}

	logger.Info("server stopped")
}
