// Package idle provides idle detection for scale-to-zero functionality.
package idle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/janovincze/philotes/internal/scaling"
)

var (
	// idleDurationGauge tracks idle duration for each policy.
	idleDurationGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "idle_duration_seconds",
			Help:      "Duration in seconds since last activity for a scaling policy",
		},
		[]string{"policy_id", "policy_name"},
	)

	// lastActivityGauge tracks the timestamp of last activity.
	lastActivityGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "last_activity_timestamp_seconds",
			Help:      "Unix timestamp of last activity for a scaling policy",
		},
		[]string{"policy_id", "policy_name"},
	)

	// scaledToZeroGauge indicates whether a policy is scaled to zero.
	scaledToZeroGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "is_scaled_to_zero",
			Help:      "Whether the scaling policy is currently scaled to zero (1=yes, 0=no)",
		},
		[]string{"policy_id", "policy_name"},
	)

	// scaledToZeroDurationGauge tracks how long a policy has been at zero.
	scaledToZeroDurationGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "scaled_to_zero_duration_seconds",
			Help:      "Duration in seconds a policy has been scaled to zero",
		},
		[]string{"policy_id", "policy_name"},
	)

	// wakeCounter tracks wake events.
	wakeCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "wake_events_total",
			Help:      "Total number of wake events by reason",
		},
		[]string{"policy_id", "policy_name", "reason"},
	)

	// scaleToZeroCounter tracks scale-to-zero events.
	scaleToZeroCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "scale_to_zero_events_total",
			Help:      "Total number of scale-to-zero events",
		},
		[]string{"policy_id", "policy_name"},
	)

	// costSavingsGauge tracks estimated cost savings.
	costSavingsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "philotes",
			Subsystem: "scaling",
			Name:      "estimated_savings_euros",
			Help:      "Estimated cost savings in euros from scale-to-zero",
		},
		[]string{"policy_id", "policy_name"},
	)
)

// MetricsCollector collects and exposes idle metrics.
type MetricsCollector struct {
	detector *Detector
	logger   *slog.Logger

	mu          sync.RWMutex
	policyNames map[string]string // policy_id -> policy_name

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(detector *Detector, logger *slog.Logger) *MetricsCollector {
	if logger == nil {
		logger = slog.Default()
	}

	return &MetricsCollector{
		detector:    detector,
		logger:      logger.With("component", "idle-metrics"),
		policyNames: make(map[string]string),
		stopCh:      make(chan struct{}),
	}
}

// Start begins the metrics collection loop.
func (c *MetricsCollector) Start(ctx context.Context, updateInterval time.Duration) {
	c.wg.Add(1)
	go c.runLoop(ctx, updateInterval)
}

// Stop stops the metrics collector.
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// RegisterPolicy registers a policy for metrics collection.
func (c *MetricsCollector) RegisterPolicy(policyID, policyName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policyNames[policyID] = policyName
}

// UnregisterPolicy removes a policy from metrics collection.
func (c *MetricsCollector) UnregisterPolicy(policyID string) {
	c.mu.Lock()
	name := c.policyNames[policyID]
	delete(c.policyNames, policyID)
	c.mu.Unlock()

	// Clean up metrics
	idleDurationGauge.DeleteLabelValues(policyID, name)
	lastActivityGauge.DeleteLabelValues(policyID, name)
	scaledToZeroGauge.DeleteLabelValues(policyID, name)
	scaledToZeroDurationGauge.DeleteLabelValues(policyID, name)
}

// runLoop is the main metrics collection loop.
func (c *MetricsCollector) runLoop(ctx context.Context, interval time.Duration) {
	defer c.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.updateMetrics(ctx)
		}
	}
}

// updateMetrics updates all idle metrics.
func (c *MetricsCollector) updateMetrics(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for policyID, policyName := range c.policyNames {
		state, err := c.detector.repo.GetIdleState(ctx, parseUUID(policyID))
		if err != nil {
			c.logger.Warn("failed to get idle state for metrics",
				"policy_id", policyID,
				"error", err,
			)
			continue
		}

		if state == nil {
			continue
		}

		// Update idle duration
		idleDuration := state.IdleDuration().Seconds()
		idleDurationGauge.WithLabelValues(policyID, policyName).Set(idleDuration)

		// Update last activity timestamp
		lastActivityGauge.WithLabelValues(policyID, policyName).Set(float64(state.LastActivityAt.Unix()))

		// Update scaled-to-zero status
		if state.IsScaledToZero {
			scaledToZeroGauge.WithLabelValues(policyID, policyName).Set(1)
			scaledToZeroDurationGauge.WithLabelValues(policyID, policyName).Set(state.ScaledToZeroDuration().Seconds())
		} else {
			scaledToZeroGauge.WithLabelValues(policyID, policyName).Set(0)
			scaledToZeroDurationGauge.WithLabelValues(policyID, policyName).Set(0)
		}
	}
}

// RecordWake records a wake event.
func (c *MetricsCollector) RecordWake(policyID, policyName string, reason scaling.WakeReason) {
	wakeCounter.WithLabelValues(policyID, policyName, reason.String()).Inc()
}

// RecordScaleToZero records a scale-to-zero event.
func (c *MetricsCollector) RecordScaleToZero(policyID, policyName string) {
	scaleToZeroCounter.WithLabelValues(policyID, policyName).Inc()
}

// UpdateCostSavings updates the cost savings metric.
func (c *MetricsCollector) UpdateCostSavings(policyID, policyName string, savingsEuros float64) {
	costSavingsGauge.WithLabelValues(policyID, policyName).Set(savingsEuros)
}

// parseUUID is a helper to parse UUID without error.
func parseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}
