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

	// Health endpoints (no versioning)
	s.router.GET("/health", healthHandler.GetHealth)
	s.router.GET("/health/live", healthHandler.GetLiveness)
	s.router.GET("/health/ready", healthHandler.GetReadiness)

	// Metrics endpoint (no versioning)
	if s.cfg.Metrics.Enabled {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// System endpoints
		v1.GET("/version", versionHandler.GetVersion)
		v1.GET("/config", configHandler.GetConfig)

		// Source endpoints
		if sourceHandler != nil {
			v1.POST("/sources", sourceHandler.Create)
			v1.GET("/sources", sourceHandler.List)
			v1.GET("/sources/:id", sourceHandler.Get)
			v1.PUT("/sources/:id", sourceHandler.Update)
			v1.DELETE("/sources/:id", sourceHandler.Delete)
			v1.POST("/sources/:id/test", sourceHandler.TestConnection)
			v1.GET("/sources/:id/tables", sourceHandler.DiscoverTables)
		}

		// Pipeline endpoints
		if pipelineHandler != nil {
			v1.POST("/pipelines", pipelineHandler.Create)
			v1.GET("/pipelines", pipelineHandler.List)
			v1.GET("/pipelines/:id", pipelineHandler.Get)
			v1.PUT("/pipelines/:id", pipelineHandler.Update)
			v1.DELETE("/pipelines/:id", pipelineHandler.Delete)
			v1.POST("/pipelines/:id/start", pipelineHandler.Start)
			v1.POST("/pipelines/:id/stop", pipelineHandler.Stop)
			v1.GET("/pipelines/:id/status", pipelineHandler.GetStatus)
			v1.POST("/pipelines/:id/tables", pipelineHandler.AddTableMapping)
			v1.DELETE("/pipelines/:id/tables/:mappingId", pipelineHandler.RemoveTableMapping)
		}

		// Pipeline metrics endpoints
		if s.metricsService != nil {
			metricsHandler := handlers.NewMetricsHandler(s.metricsService)
			v1.GET("/pipelines/:id/metrics", metricsHandler.GetPipelineMetrics)
			v1.GET("/pipelines/:id/metrics/history", metricsHandler.GetPipelineMetricsHistory)
		}

		// Alert endpoints
		if alertHandler != nil {
			alertHandler.Register(v1)
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
