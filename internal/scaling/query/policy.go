// Package query provides query engine metrics collection and scaling policy evaluation.
package query

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/config"
)

// Policy represents a query scaling policy.
type Policy struct {
	ID                      uuid.UUID
	Name                    string
	QueryEngine             string
	Enabled                 bool
	MinReplicas             int
	MaxReplicas             int
	CooldownSeconds         int
	ScaleToZero             bool
	QueuedQueriesThreshold  int
	RunningQueriesThreshold int
	LatencyThresholdSeconds int
}

// ScalingDecision represents a scaling decision.
type ScalingDecision struct {
	PolicyID        uuid.UUID
	QueryEngine     string
	Action          string // "scale_up", "scale_down", "scale_to_zero", "wake", "none"
	CurrentReplicas int
	DesiredReplicas int
	TriggerReason   string
	TriggerValue    float64
}

// Evaluator evaluates query scaling policies against collected metrics.
type Evaluator struct {
	cfg       config.QueryScalingConfig
	collector *Collector
	logger    *slog.Logger

	// Track last scaling action time per policy to enforce cooldown
	mu            sync.RWMutex
	lastScaleTime map[uuid.UUID]time.Time
}

// NewEvaluator creates a new policy evaluator.
func NewEvaluator(cfg config.QueryScalingConfig, collector *Collector, logger *slog.Logger) *Evaluator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Evaluator{
		cfg:           cfg,
		collector:     collector,
		logger:        logger.With("component", "query-policy-evaluator"),
		lastScaleTime: make(map[uuid.UUID]time.Time),
	}
}

// Evaluate evaluates a policy against current metrics and returns a scaling decision.
func (e *Evaluator) Evaluate(_ context.Context, policy *Policy, currentReplicas int) (*ScalingDecision, error) {
	if !policy.Enabled {
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "none",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: currentReplicas,
			TriggerReason:   "policy disabled",
		}, nil
	}

	// Check cooldown
	e.mu.RLock()
	lastScale, ok := e.lastScaleTime[policy.ID]
	e.mu.RUnlock()
	if ok {
		cooldownDuration := time.Duration(policy.CooldownSeconds) * time.Second
		if time.Since(lastScale) < cooldownDuration {
			return &ScalingDecision{
				PolicyID:        policy.ID,
				QueryEngine:     policy.QueryEngine,
				Action:          "none",
				CurrentReplicas: currentReplicas,
				DesiredReplicas: currentReplicas,
				TriggerReason:   "in cooldown",
			}, nil
		}
	}

	// Get metrics for this query engine
	metrics := e.collector.GetCached()
	queryMetrics, ok := metrics[policy.QueryEngine]
	if !ok {
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "none",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: currentReplicas,
			TriggerReason:   "no metrics available",
		}, nil
	}

	// Evaluate scale-up triggers
	decision := e.evaluateScaleUp(policy, queryMetrics, currentReplicas)
	if decision != nil {
		return decision, nil
	}

	// Evaluate scale-down triggers (including scale-to-zero)
	decision = e.evaluateScaleDown(policy, queryMetrics, currentReplicas)
	if decision != nil {
		return decision, nil
	}

	return &ScalingDecision{
		PolicyID:        policy.ID,
		QueryEngine:     policy.QueryEngine,
		Action:          "none",
		CurrentReplicas: currentReplicas,
		DesiredReplicas: currentReplicas,
		TriggerReason:   "within thresholds",
	}, nil
}

// evaluateScaleUp checks if scale-up is needed.
func (e *Evaluator) evaluateScaleUp(policy *Policy, metrics *Metrics, currentReplicas int) *ScalingDecision {
	if currentReplicas >= policy.MaxReplicas {
		return nil
	}

	// Check queued queries threshold
	if metrics.QueuedQueries >= policy.QueuedQueriesThreshold {
		desired := min(currentReplicas+1, policy.MaxReplicas)
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "scale_up",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: desired,
			TriggerReason:   fmt.Sprintf("queued_queries >= %d", policy.QueuedQueriesThreshold),
			TriggerValue:    float64(metrics.QueuedQueries),
		}
	}

	// Check running queries threshold
	if metrics.RunningQueries >= policy.RunningQueriesThreshold {
		desired := min(currentReplicas+1, policy.MaxReplicas)
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "scale_up",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: desired,
			TriggerReason:   fmt.Sprintf("running_queries >= %d", policy.RunningQueriesThreshold),
			TriggerValue:    float64(metrics.RunningQueries),
		}
	}

	// Check latency threshold (if we have latency metrics)
	if metrics.P95LatencyMs != nil && policy.LatencyThresholdSeconds > 0 {
		latencyThresholdMs := float64(policy.LatencyThresholdSeconds * 1000)
		if *metrics.P95LatencyMs >= latencyThresholdMs {
			desired := min(currentReplicas+1, policy.MaxReplicas)
			return &ScalingDecision{
				PolicyID:        policy.ID,
				QueryEngine:     policy.QueryEngine,
				Action:          "scale_up",
				CurrentReplicas: currentReplicas,
				DesiredReplicas: desired,
				TriggerReason:   fmt.Sprintf("p95_latency >= %dms", policy.LatencyThresholdSeconds*1000),
				TriggerValue:    *metrics.P95LatencyMs,
			}
		}
	}

	return nil
}

// evaluateScaleDown checks if scale-down is needed.
func (e *Evaluator) evaluateScaleDown(policy *Policy, metrics *Metrics, currentReplicas int) *ScalingDecision {
	if currentReplicas <= policy.MinReplicas && !policy.ScaleToZero {
		return nil
	}

	// Scale down when queries are below half the thresholds.
	// Note: Integer division is intentional - we use floor division to ensure
	// significant load reduction before scaling down (e.g., threshold 5 -> scale down at < 2).
	queuedBelowThreshold := metrics.QueuedQueries < policy.QueuedQueriesThreshold/2
	runningBelowThreshold := metrics.RunningQueries < policy.RunningQueriesThreshold/2

	if !queuedBelowThreshold || !runningBelowThreshold {
		return nil
	}

	// Check for scale-to-zero condition
	if policy.ScaleToZero && metrics.QueuedQueries == 0 && metrics.RunningQueries == 0 && metrics.BlockedQueries == 0 {
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "scale_to_zero",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: 0,
			TriggerReason:   "no active queries",
			TriggerValue:    0,
		}
	}

	// Regular scale down
	if currentReplicas > policy.MinReplicas {
		desired := max(currentReplicas-1, policy.MinReplicas)
		return &ScalingDecision{
			PolicyID:        policy.ID,
			QueryEngine:     policy.QueryEngine,
			Action:          "scale_down",
			CurrentReplicas: currentReplicas,
			DesiredReplicas: desired,
			TriggerReason:   "low query load",
			TriggerValue:    float64(metrics.RunningQueries),
		}
	}

	return nil
}

// RecordScaling records a scaling action for cooldown tracking.
func (e *Evaluator) RecordScaling(policyID uuid.UUID) {
	e.mu.Lock()
	e.lastScaleTime[policyID] = time.Now()
	e.mu.Unlock()
}

// DefaultPolicy returns a policy with default values from config.
func DefaultPolicy(cfg config.QueryScalingConfig, name, queryEngine string) *Policy {
	return &Policy{
		ID:                      uuid.New(),
		Name:                    name,
		QueryEngine:             queryEngine,
		Enabled:                 true,
		MinReplicas:             cfg.DefaultMinReplicas,
		MaxReplicas:             cfg.DefaultMaxReplicas,
		CooldownSeconds:         cfg.DefaultCooldownSeconds,
		ScaleToZero:             false,
		QueuedQueriesThreshold:  cfg.DefaultQueuedQueriesThreshold,
		RunningQueriesThreshold: cfg.DefaultRunningQueriesThreshold,
		LatencyThresholdSeconds: cfg.DefaultLatencyThreshold,
	}
}
