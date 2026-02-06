// Package handler provides HTTP handlers for the Dagster proxy.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/api/services"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/dagster-proxy/audit"
	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

// ServerConfig holds configuration for the proxy server.
type ServerConfig struct {
	ListenAddr     string
	UpstreamURL    *url.URL
	Logger         *slog.Logger
	HealthManager  *health.Manager
	AuthService    *services.AuthService
	APIKeyService  *services.APIKeyService
	AuditRepo      *repositories.AuditRepository
	AuthEnabled    bool
	APIKeyPrefix   string
	RequestTimeout time.Duration
}

// Server is the Dagster proxy HTTP server.
type Server struct {
	cfg        ServerConfig
	httpServer *http.Server
	router     *gin.Engine
	proxy      *ProxyHandler
	checker    *auth.PermissionChecker
	auditor    *audit.Logger
}

// NewServer creates a new proxy server.
func NewServer(cfg ServerConfig) *Server {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Create permission checker
	checker := auth.NewPermissionChecker(cfg.Logger)

	// Create audit logger
	auditor := audit.NewLogger(cfg.AuditRepo, cfg.Logger)

	// Create proxy handler
	proxy := NewProxyHandler(ProxyConfig{
		UpstreamURL:    cfg.UpstreamURL,
		Logger:         cfg.Logger,
		Checker:        checker,
		Auditor:        auditor,
		AuthEnabled:    cfg.AuthEnabled,
		RequestTimeout: cfg.RequestTimeout,
	})

	s := &Server{
		cfg:     cfg,
		router:  router,
		proxy:   proxy,
		checker: checker,
		auditor: auditor,
	}

	s.setupRoutes()

	return s
}

// setupRoutes configures the HTTP routes.
func (s *Server) setupRoutes() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Request logging
	s.router.Use(s.requestLogger())

	// Health endpoints (no auth required)
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/health/ready", s.readinessHandler)
	s.router.GET("/health/live", s.livenessHandler)

	// GraphQL proxy endpoint
	graphql := s.router.Group("/graphql")
	// Add auth middleware if enabled
	if s.cfg.AuthEnabled {
		graphql.Use(s.authMiddleware())
	}
	// Handle both GET (WebSocket upgrade) and POST (GraphQL)
	graphql.GET("", s.proxy.HandleWebSocket)
	graphql.POST("", s.proxy.HandleHTTP)

	// Proxy all other requests to Dagster (static assets, etc.)
	s.router.NoRoute(s.proxy.HandlePassthrough)
}

// requestLogger returns a middleware that logs requests.
func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		s.cfg.Logger.Debug("request completed",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration", time.Since(start),
			"client_ip", c.ClientIP(),
		)
	}
}

// authMiddleware returns a middleware that authenticates requests.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := s.extractAuthContext(c)
		if authCtx == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"errors": []gin.H{
					{
						"message":    "Authentication required",
						"extensions": gin.H{"code": "UNAUTHENTICATED"},
					},
				},
			})
			c.Abort()
			return
		}

		// Store auth context for downstream handlers
		c.Set("auth_context", authCtx)
		c.Next()
	}
}

// extractAuthContext attempts to extract auth credentials from the request.
func (s *Server) extractAuthContext(c *gin.Context) *auth.Context {
	// Check for API key in X-API-Key header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return s.validateAPIKey(c, apiKey)
	}

	// Check for Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil
	}

	// Parse "Bearer <token>" format
	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) {
		return nil
	}

	token := authHeader[len(bearerPrefix):]

	// Check if it's an API key or JWT
	if len(token) > len(s.cfg.APIKeyPrefix) && token[:len(s.cfg.APIKeyPrefix)] == s.cfg.APIKeyPrefix {
		return s.validateAPIKey(c, token)
	}

	return s.validateJWT(c, token)
}

// validateAPIKey validates an API key and returns auth context.
func (s *Server) validateAPIKey(c *gin.Context, key string) *auth.Context {
	if s.cfg.APIKeyService == nil {
		return nil
	}

	user, apiKey, err := s.cfg.APIKeyService.Validate(c.Request.Context(), key)
	if err != nil {
		s.cfg.Logger.Debug("API key validation failed", "error", err)
		return nil
	}

	return &auth.Context{
		UserID:      user.ID,
		UserEmail:   user.Email,
		Permissions: apiKey.Permissions,
		IsAPIKey:    true,
		APIKeyID:    &apiKey.ID,
	}
}

// validateJWT validates a JWT token and returns auth context.
func (s *Server) validateJWT(c *gin.Context, token string) *auth.Context {
	if s.cfg.AuthService == nil {
		return nil
	}

	claims, err := s.cfg.AuthService.ValidateJWT(token)
	if err != nil {
		s.cfg.Logger.Debug("JWT validation failed", "error", err)
		return nil
	}

	// Get user to verify they're still active
	user, err := s.cfg.AuthService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil || !user.IsActive {
		return nil
	}

	return &auth.Context{
		UserID:      user.ID,
		UserEmail:   user.Email,
		Role:        string(user.Role),
		Permissions: claims.Permissions,
		IsAPIKey:    false,
	}
}

// healthHandler returns the overall health status.
func (s *Server) healthHandler(c *gin.Context) {
	result := s.cfg.HealthManager.GetOverallStatus(c.Request.Context())

	status := http.StatusOK
	if result.Status == health.StatusUnhealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, result)
}

// readinessHandler returns readiness status.
func (s *Server) readinessHandler(c *gin.Context) {
	result := s.cfg.HealthManager.GetOverallStatus(c.Request.Context())

	if result.Status == health.StatusUnhealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// livenessHandler returns liveness status.
func (s *Server) livenessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:         s.cfg.ListenAddr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: s.cfg.RequestTimeout + 10*time.Second, // Allow extra time for response
		IdleTimeout:  120 * time.Second,
	}

	s.cfg.Logger.Info("starting HTTP server", "addr", s.cfg.ListenAddr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.cfg.Logger.Info("stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
