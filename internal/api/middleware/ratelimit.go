package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/janovincze/philotes/internal/api/models"
)

// RateLimitConfig holds rate limiter configuration.
type RateLimitConfig struct {
	// RequestsPerSecond is the rate limit in requests per second.
	RequestsPerSecond float64

	// BurstSize is the maximum burst size.
	BurstSize int

	// PerClient enables per-client rate limiting based on IP.
	PerClient bool

	// ClientTTL is how long to keep inactive client limiters before cleanup.
	// Defaults to 1 hour if not set.
	ClientTTL time.Duration

	// CleanupInterval is how often to run the cleanup routine.
	// Defaults to 10 minutes if not set.
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns a RateLimitConfig with sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		PerClient:         true,
		ClientTTL:         time.Hour,
		CleanupInterval:   10 * time.Minute,
	}
}

// RateLimiter returns a middleware that limits request rate.
func RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.PerClient {
		return perClientRateLimiter(cfg)
	}
	return globalRateLimiter(cfg)
}

// globalRateLimiter creates a single limiter for all requests.
func globalRateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			models.RespondWithError(c, models.NewRateLimitedError(c.Request.URL.Path))
			c.Abort()
			return
		}
		c.Next()
	}
}

// clientLimiter holds a rate limiter and its last access time.
type clientLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// rateLimiterStore holds the shared state for per-client rate limiting.
// Using a singleton pattern ensures only one cleanup goroutine runs globally.
type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*clientLimiter
	once     sync.Once
	ttl      time.Duration
	interval time.Duration
}

// cleanup runs periodically to remove stale client limiters.
func (s *rateLimiterStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for ip, cl := range s.limiters {
		if now.Sub(cl.lastAccess) > s.ttl {
			delete(s.limiters, ip)
		}
	}
}

// startCleanup starts the cleanup goroutine exactly once.
func (s *rateLimiterStore) startCleanup() {
	s.once.Do(func() {
		go func() {
			ticker := time.NewTicker(s.interval)
			defer ticker.Stop()

			for range ticker.C {
				s.cleanup()
			}
		}()
	})
}

// getOrCreateLimiter returns the limiter for a client IP, creating one if needed.
func (s *rateLimiterStore) getOrCreateLimiter(clientIP string, rps float64, burst int) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cl, exists := s.limiters[clientIP]
	if !exists {
		cl = &clientLimiter{
			limiter:    rate.NewLimiter(rate.Limit(rps), burst),
			lastAccess: now,
		}
		s.limiters[clientIP] = cl
	} else {
		cl.lastAccess = now
	}
	return cl.limiter
}

// perClientRateLimiter creates a limiter per client IP with automatic cleanup.
func perClientRateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	// Set defaults if not configured
	clientTTL := cfg.ClientTTL
	if clientTTL == 0 {
		clientTTL = time.Hour
	}
	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 10 * time.Minute
	}

	// Create store for this rate limiter instance
	store := &rateLimiterStore{
		limiters: make(map[string]*clientLimiter),
		ttl:      clientTTL,
		interval: cleanupInterval,
	}

	// Start cleanup goroutine (only once per store instance)
	store.startCleanup()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		limiter := store.getOrCreateLimiter(clientIP, cfg.RequestsPerSecond, cfg.BurstSize)

		if !limiter.Allow() {
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", formatFloat(cfg.RequestsPerSecond))
			c.Header("X-RateLimit-Remaining", "0")
			models.RespondWithError(c, models.NewRateLimitedError(c.Request.URL.Path))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", formatFloat(cfg.RequestsPerSecond))
		c.Next()
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.0f", f)
}
