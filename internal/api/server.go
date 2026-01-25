// Package api provides the HTTP API server for Philotes management.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/handlers"
	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/config"
)

// Server is the HTTP API server.
type Server struct {
	cfg           *config.Config
	logger        *slog.Logger
	healthManager *health.Manager
	httpServer    *http.Server
	router        *gin.Engine
}

// ServerConfig holds server configuration options.
type ServerConfig struct {
	// Config is the application configuration.
	Config *config.Config

	// Logger is the structured logger.
	Logger *slog.Logger

	// HealthManager is the health check manager.
	HealthManager *health.Manager

	// CORSConfig is the CORS configuration.
	CORSConfig middleware.CORSConfig

	// RateLimitConfig is the rate limiting configuration.
	RateLimitConfig middleware.RateLimitConfig
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
func DefaultServerConfig(cfg *config.Config, logger *slog.Logger) ServerConfig {
	return ServerConfig{
		Config:          cfg,
		Logger:          logger,
		HealthManager:   nil,
		CORSConfig:      middleware.DefaultCORSConfig(),
		RateLimitConfig: middleware.DefaultRateLimitConfig(),
	}
}

// NewServer creates a new API server.
func NewServer(serverCfg ServerConfig) *Server {
	logger := serverCfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Set Gin mode based on environment
	if serverCfg.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Apply middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS(serverCfg.CORSConfig))
	router.Use(middleware.RateLimiter(serverCfg.RateLimitConfig))

	// Create server
	s := &Server{
		cfg:           serverCfg.Config,
		logger:        logger.With("component", "api-server"),
		healthManager: serverCfg.HealthManager,
		router:        router,
	}

	// Register routes
	s.registerRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         serverCfg.Config.API.ListenAddr,
		Handler:      router,
		ReadTimeout:  serverCfg.Config.API.ReadTimeout,
		WriteTimeout: serverCfg.Config.API.WriteTimeout,
		IdleTimeout:  serverCfg.Config.API.ReadTimeout * 4,
	}

	return s
}

// registerRoutes registers all API routes.
func (s *Server) registerRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.healthManager)
	versionHandler := handlers.NewVersionHandler(s.cfg.Version)
	configHandler := handlers.NewConfigHandler(s.cfg)
	stubHandler := handlers.NewStubHandler()

	// Health endpoints (no versioning)
	s.router.GET("/health", healthHandler.GetHealth)
	s.router.GET("/health/live", healthHandler.GetLiveness)
	s.router.GET("/health/ready", healthHandler.GetReadiness)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// System endpoints
		v1.GET("/version", versionHandler.GetVersion)
		v1.GET("/config", configHandler.GetConfig)

		// Stub endpoints for future implementation
		v1.GET("/sources", stubHandler.ListSources)
		v1.GET("/pipelines", stubHandler.ListPipelines)
		v1.GET("/destinations", stubHandler.ListDestinations)
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting API server", "addr", s.cfg.API.ListenAddr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Stop gracefully stops the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping API server")

	// Use a timeout context if none provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}

	return s.httpServer.Shutdown(ctx)
}

// Router returns the underlying Gin router for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}
