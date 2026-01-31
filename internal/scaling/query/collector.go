// Package query provides query engine metrics collection and scaling policy evaluation.
package query

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/janovincze/philotes/internal/config"
)

// Metrics represents query engine metrics.
type Metrics struct {
	QueryEngine    string    `json:"query_engine"`
	QueuedQueries  int       `json:"queued_queries"`
	RunningQueries int       `json:"running_queries"`
	BlockedQueries int       `json:"blocked_queries"`
	ActiveWorkers  int       `json:"active_workers"`
	P95LatencyMs   *float64  `json:"p95_latency_ms,omitempty"`
	CollectedAt    time.Time `json:"collected_at"`
}

// Collector collects metrics from query engines.
type Collector struct {
	trinoCfg   config.TrinoConfig
	httpClient *http.Client
	logger     *slog.Logger

	mu          sync.RWMutex
	lastMetrics map[string]*Metrics
}

// NewCollector creates a new metrics collector.
func NewCollector(trinoCfg config.TrinoConfig, logger *slog.Logger) *Collector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Collector{
		trinoCfg: trinoCfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:      logger.With("component", "query-metrics-collector"),
		lastMetrics: make(map[string]*Metrics),
	}
}

// Collect collects metrics from all enabled query engines.
func (c *Collector) Collect(ctx context.Context) (map[string]*Metrics, error) {
	metrics := make(map[string]*Metrics)

	// Collect Trino metrics
	if c.trinoCfg.Enabled {
		trinoMetrics, err := c.collectTrinoMetrics(ctx)
		if err != nil {
			c.logger.Warn("failed to collect Trino metrics", "error", err)
		} else {
			metrics["trino"] = trinoMetrics
		}
	}

	// Update cached metrics
	c.mu.Lock()
	c.lastMetrics = metrics
	c.mu.Unlock()

	return metrics, nil
}

// GetCached returns the last collected metrics.
func (c *Collector) GetCached() map[string]*Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*Metrics)
	for k, v := range c.lastMetrics {
		result[k] = v
	}
	return result
}

// trinoClusterStats represents Trino cluster stats response.
type trinoClusterStats struct {
	RunningQueries           int `json:"runningQueries"`
	QueuedQueries            int `json:"queuedQueries"`
	BlockedQueries           int `json:"blockedQueries"`
	ActiveWorkers            int `json:"activeWorkers"`
	RunningDrivers           int `json:"runningDrivers"`
	TotalAvailableProcessors int `json:"totalAvailableProcessors"`
}

// collectTrinoMetrics collects metrics from Trino coordinator.
func (c *Collector) collectTrinoMetrics(ctx context.Context) (*Metrics, error) {
	url := strings.TrimSuffix(c.trinoCfg.URL, "/") + "/v1/cluster"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.trinoCfg.Username != "" {
		req.SetBasicAuth(c.trinoCfg.Username, c.trinoCfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("Trino returned status %d (failed to read body: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("Trino returned status %d: %s", resp.StatusCode, string(body))
	}

	var stats trinoClusterStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Metrics{
		QueryEngine:    "trino",
		QueuedQueries:  stats.QueuedQueries,
		RunningQueries: stats.RunningQueries,
		BlockedQueries: stats.BlockedQueries,
		ActiveWorkers:  stats.ActiveWorkers,
		CollectedAt:    time.Now(),
	}, nil
}

// StartPeriodicCollection starts periodic metric collection.
func (c *Collector) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect immediately on start
	if _, err := c.Collect(ctx); err != nil {
		c.logger.Error("initial metric collection failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := c.Collect(ctx); err != nil {
				c.logger.Error("metric collection failed", "error", err)
			}
		}
	}
}
