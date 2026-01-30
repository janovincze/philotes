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
	"github.com/janovincze/philotes/internal/installer"
	"github.com/janovincze/philotes/internal/metrics"
)

// Server is the HTTP API server.
type Server struct {
	cfg                   *config.Config
	logger                *slog.Logger
	healthManager         *health.Manager
	sourceService         *services.SourceService
	pipelineService       *services.PipelineService
	alertService          *services.AlertService
	metricsService        *services.MetricsService
	installerService      *services.InstallerService
	installerLogHub       *installer.LogHub
	installerOrchestrator *installer.DeploymentOrchestrator
	authService           *services.AuthService
	apiKeyService         *services.APIKeyService
	oauthService          *services.OAuthService
	oidcService           *services.OIDCService
	onboardingService     *services.OnboardingService
	httpServer            *http.Server
	router                *gin.Engine
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

	// InstallerService is the installer service for deployment operations.
	InstallerService *services.InstallerService

	// InstallerLogHub is the WebSocket log hub for deployment streaming.
	InstallerLogHub *installer.LogHub

	// InstallerOrchestrator is the deployment orchestrator for progress tracking.
	InstallerOrchestrator *installer.DeploymentOrchestrator

	// AuthService is the auth service for authentication.
	AuthService *services.AuthService

	// APIKeyService is the API key service for API key management.
	APIKeyService *services.APIKeyService

	// OAuthService is the OAuth service for cloud provider authentication.
	OAuthService *services.OAuthService

	// OIDCService is the OIDC service for SSO authentication.
	OIDCService *services.OIDCService

	// OnboardingService is the onboarding service for post-installation wizard.
	OnboardingService *services.OnboardingService

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
		cfg:                   serverCfg.Config,
		logger:                logger.With("component", "api-server"),
		healthManager:         serverCfg.HealthManager,
		sourceService:         serverCfg.SourceService,
		pipelineService:       serverCfg.PipelineService,
		alertService:          serverCfg.AlertService,
		metricsService:        serverCfg.MetricsService,
		installerService:      serverCfg.InstallerService,
		installerLogHub:       serverCfg.InstallerLogHub,
		installerOrchestrator: serverCfg.InstallerOrchestrator,
		authService:           serverCfg.AuthService,
		apiKeyService:         serverCfg.APIKeyService,
		oauthService:          serverCfg.OAuthService,
		oidcService:           serverCfg.OIDCService,
		onboardingService:     serverCfg.OnboardingService,
		router:                router,
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
		authHandler = handlers.NewAuthHandler(s.authService, s.apiKeyService)
	}
	if s.apiKeyService != nil {
		apiKeyHandler = handlers.NewAPIKeyHandler(s.apiKeyService)
	}

	// Create onboarding handler
	var onboardingHandler *handlers.OnboardingHandler
	if s.onboardingService != nil {
		onboardingHandler = handlers.NewOnboardingHandler(s.onboardingService)
	}

	// Create OIDC handler
	var oidcHandler *handlers.OIDCHandler
	if s.oidcService != nil {
		oidcHandler = handlers.NewOIDCHandler(s.oidcService)
	}

	// Configure auth middleware
	authConfig := middleware.AuthConfig{
		Enabled:       s.cfg.Auth.Enabled,
		AuthService:   s.authService,
		APIKeyService: s.apiKeyService,
		APIKeyPrefix:  s.cfg.Auth.APIKeyPrefix,
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

		// Onboarding endpoints (mostly public for initial setup)
		if onboardingHandler != nil {
			onboardingHandler.Register(v1)
		}

		// OIDC SSO endpoints (registered by handler)
		if oidcHandler != nil {
			oidcHandler.Register(v1, requireAuth)
		}

		// Source endpoints (protected when auth is enabled)
		if sourceHandler != nil {
			sources := v1.Group("/sources")
			sources.Use(requireAuth)
			sources.POST("", sourceHandler.Create)
			sources.GET("", sourceHandler.List)
			sources.GET("/:id", sourceHandler.Get)
			sources.PUT("/:id", sourceHandler.Update)
			sources.DELETE("/:id", sourceHandler.Delete)
			sources.POST("/:id/test", sourceHandler.TestConnection)
			sources.GET("/:id/tables", sourceHandler.DiscoverTables)
		}

		// Pipeline endpoints (protected when auth is enabled)
		if pipelineHandler != nil {
			pipelines := v1.Group("/pipelines")
			pipelines.Use(requireAuth)
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

		// Alert endpoints (protected when auth is enabled)
		// Note: alertHandler.Register adds /alerts/* routes to the passed group
		if alertHandler != nil {
			protected := v1.Group("")
			protected.Use(requireAuth)
			alertHandler.Register(protected)
		}

		// Installer endpoints
		if s.installerService != nil {
			installerHandler := handlers.NewInstallerHandler(s.installerService, s.installerLogHub, s.installerOrchestrator)

			// Provider endpoints (public - no auth required for browsing)
			installerGroup := v1.Group("/installer")
			installerGroup.GET("/providers", installerHandler.ListProviders)
			installerGroup.GET("/providers/:id", installerHandler.GetProvider)
			installerGroup.GET("/providers/:id/estimate", installerHandler.GetCostEstimate)

			// OAuth endpoints (registered if OAuth service is available)
			if s.oauthService != nil {
				oauthHandler := handlers.NewOAuthHandler(s.oauthService)

				// OAuth provider info (public)
				installerGroup.GET("/oauth/providers", oauthHandler.GetOAuthProviders)

				// OAuth flow endpoints (start auth requires optional auth, callback is public)
				installerGroup.POST("/oauth/:provider/authorize", oauthHandler.Authorize)
				installerGroup.GET("/oauth/:provider/callback", oauthHandler.Callback)

				// Manual credential storage (public for now, will store with session/user)
				installerGroup.POST("/credentials/:provider", oauthHandler.StoreCredential)

				// Protected credential endpoints
				credentialsGroup := installerGroup.Group("/credentials")
				credentialsGroup.Use(requireAuth)
				credentialsGroup.GET("", oauthHandler.ListCredentials)
				credentialsGroup.DELETE("/:provider", oauthHandler.DeleteCredential)
			}

			// Deployment endpoints (protected when auth is enabled)
			deployments := installerGroup.Group("/deployments")
			deployments.Use(requireAuth)
			deployments.POST("", installerHandler.CreateDeployment)
			deployments.GET("", installerHandler.ListDeployments)
			deployments.GET("/:id", installerHandler.GetDeployment)
			deployments.POST("/:id/cancel", installerHandler.CancelDeployment)
			deployments.DELETE("/:id", installerHandler.DeleteDeployment)
			deployments.GET("/:id/logs", installerHandler.GetDeploymentLogs)
			// WebSocket endpoint for real-time log streaming
			deployments.GET("/:id/logs/stream", installerHandler.StreamDeploymentLogs)
			// Progress tracking endpoints
			deployments.GET("/:id/progress", installerHandler.GetDeploymentProgress)
			deployments.POST("/:id/retry", installerHandler.RetryDeployment)
			deployments.GET("/:id/cleanup-preview", installerHandler.GetCleanupResources)
			deployments.GET("/:id/retry-info", installerHandler.GetRetryInfo)
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
