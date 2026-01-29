// Package api provides the HTTP API server for Philotes management.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/janovincze/philotes/internal/api/handlers"
	"github.com/janovincze/philotes/internal/api/middleware"
	"github.com/janovincze/philotes/internal/api/services"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/config"
	"github.com/janovincze/philotes/internal/metrics"
)

// Server is the HTTP API server.
type Server struct {
	cfg             *config.Config
	logger          *slog.Logger
	healthManager   *health.Manager
	sourceService   *services.SourceService
	pipelineService *services.PipelineService
	alertService    *services.AlertService
	metricsService  *services.MetricsService
	authService     *services.AuthService
	apiKeyService   *services.APIKeyService
	httpServer      *http.Server
	router          *gin.Engine
}

// ServerConfig holds server configuration options.
type ServerConfig struct {
	// Config is the application configuration.
	Config *config.Config

	// Logger is the structured logger.
	Logger *slog.Logger

	// HealthManager is the health check manager.
	HealthManager *health.Manager

	// SourceService is the source service for source CRUD operations.
	SourceService *services.SourceService

	// PipelineService is the pipeline service for pipeline CRUD operations.
	PipelineService *services.PipelineService

	// AlertService is the alert service for alerting CRUD operations.
	AlertService *services.AlertService

	// MetricsService is the metrics service for pipeline metrics queries.
	MetricsService *services.MetricsService

	// AuthService is the auth service for authentication.
	AuthService *services.AuthService

	// APIKeyService is the API key service for API key management.
	APIKeyService *services.APIKeyService

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

	// Register Prometheus metrics
	if serverCfg.Config.Metrics.Enabled {
		metrics.Register()
	}

	// Apply middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))
	if serverCfg.Config.Metrics.Enabled {
		router.Use(middleware.Metrics())
	}
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS(serverCfg.CORSConfig))
	router.Use(middleware.RateLimiter(serverCfg.RateLimitConfig))

	// Create server
	s := &Server{
		cfg:             serverCfg.Config,
		logger:          logger.With("component", "api-server"),
		healthManager:   serverCfg.HealthManager,
		sourceService:   serverCfg.SourceService,
		pipelineService: serverCfg.PipelineService,
		alertService:    serverCfg.AlertService,
		metricsService:  serverCfg.MetricsService,
		authService:     serverCfg.AuthService,
		apiKeyService:   serverCfg.APIKeyService,
		router:          router,
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

	// Create source, pipeline, and alert handlers (may be nil if services not provided)
	var sourceHandler *handlers.SourceHandler
	var pipelineHandler *handlers.PipelineHandler
	var alertHandler *handlers.AlertHandler
	if s.sourceService != nil {
		sourceHandler = handlers.NewSourceHandler(s.sourceService)
	}
	if s.pipelineService != nil {
		pipelineHandler = handlers.NewPipelineHandler(s.pipelineService)
	}
	if s.alertService != nil {
		alertHandler = handlers.NewAlertHandler(s.alertService)
	}

	// Create auth handlers
	var authHandler *handlers.AuthHandler
	var apiKeyHandler *handlers.APIKeyHandler
	if s.authService != nil {
		authHandler = handlers.NewAuthHandler(s.authService)
	}
	if s.apiKeyService != nil {
		apiKeyHandler = handlers.NewAPIKeyHandler(s.apiKeyService)
	}

	// Configure auth middleware
	authConfig := middleware.AuthConfig{
		Enabled:       s.cfg.Auth.Enabled,
		AuthService:   s.authService,
		APIKeyService: s.apiKeyService,
	}

	// Auth middleware: extracts credentials but doesn't require auth
	authMiddleware := middleware.Authenticate(authConfig)

	// RequireAuth middleware: requires authentication
	requireAuth := middleware.RequireAuth(authConfig)

	// Health endpoints (no versioning, no auth)
	s.router.GET("/health", healthHandler.GetHealth)
	s.router.GET("/health/live", healthHandler.GetLiveness)
	s.router.GET("/health/ready", healthHandler.GetReadiness)

	// Metrics endpoint (no versioning, no auth)
	if s.cfg.Metrics.Enabled {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	v1.Use(authMiddleware) // Apply auth middleware to extract credentials
	{
		// System endpoints (public)
		v1.GET("/version", versionHandler.GetVersion)
		v1.GET("/config", configHandler.GetConfig)

		// Auth endpoints (registered by handler)
		if authHandler != nil {
			authHandler.Register(v1, requireAuth)
		}

		// API key endpoints (registered by handler)
		if apiKeyHandler != nil {
			apiKeyHandler.Register(v1, requireAuth)
		}

		// Source endpoints (protected when auth is enabled)
		if sourceHandler != nil {
			sources := v1.Group("/sources")
			sources.Use(requireAuth)
			{
				sources.POST("", sourceHandler.Create)
				sources.GET("", sourceHandler.List)
				sources.GET("/:id", sourceHandler.Get)
				sources.PUT("/:id", sourceHandler.Update)
				sources.DELETE("/:id", sourceHandler.Delete)
				sources.POST("/:id/test", sourceHandler.TestConnection)
				sources.GET("/:id/tables", sourceHandler.DiscoverTables)
			}
		}

		// Pipeline endpoints (protected when auth is enabled)
		if pipelineHandler != nil {
			pipelines := v1.Group("/pipelines")
			pipelines.Use(requireAuth)
			{
				pipelines.POST("", pipelineHandler.Create)
				pipelines.GET("", pipelineHandler.List)
				pipelines.GET("/:id", pipelineHandler.Get)
				pipelines.PUT("/:id", pipelineHandler.Update)
				pipelines.DELETE("/:id", pipelineHandler.Delete)
				pipelines.POST("/:id/start", pipelineHandler.Start)
				pipelines.POST("/:id/stop", pipelineHandler.Stop)
				pipelines.GET("/:id/status", pipelineHandler.GetStatus)
				pipelines.POST("/:id/tables", pipelineHandler.AddTableMapping)
				pipelines.DELETE("/:id/tables/:mappingId", pipelineHandler.RemoveTableMapping)

				// Pipeline metrics endpoints
				if s.metricsService != nil {
					metricsHandler := handlers.NewMetricsHandler(s.metricsService)
					pipelines.GET("/:id/metrics", metricsHandler.GetPipelineMetrics)
					pipelines.GET("/:id/metrics/history", metricsHandler.GetPipelineMetricsHistory)
				}
			}
		}

		// Alert endpoints (protected when auth is enabled)
		// Note: alertHandler.Register adds /alerts/* routes to the passed group
		if alertHandler != nil {
			protected := v1.Group("")
			protected.Use(requireAuth)
			alertHandler.Register(protected)
		}
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
