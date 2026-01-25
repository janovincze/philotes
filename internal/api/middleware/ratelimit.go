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

// perClientRateLimiter creates a limiter per client IP with automatic cleanup.
func perClientRateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	var mu sync.Mutex
	limiters := make(map[string]*clientLimiter)

	// Set defaults if not configured
	clientTTL := cfg.ClientTTL
	if clientTTL == 0 {
		clientTTL = time.Hour
	}
	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 10 * time.Minute
	}

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, cl := range limiters {
				if now.Sub(cl.lastAccess) > clientTTL {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()
		cl, exists := limiters[clientIP]
		if !exists {
			cl = &clientLimiter{
				limiter:    rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize),
				lastAccess: now,
			}
			limiters[clientIP] = cl
		} else {
			cl.lastAccess = now
		}
		limiter := cl.limiter
		mu.Unlock()

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
