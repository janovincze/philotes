// Package models provides request and response types for the API.
package models

import (
	"time"

	"github.com/google/uuid"
)

// QueryEngine represents the type of query engine.
type QueryEngine string

const (
	// QueryEngineTrino is the Trino query engine.
	QueryEngineTrino QueryEngine = "trino"
	// QueryEngineRisingWave is the RisingWave query engine.
	QueryEngineRisingWave QueryEngine = "risingwave"
)

// QueryScalingPolicy represents a query engine auto-scaling policy.
type QueryScalingPolicy struct {
	ID                       uuid.UUID   `json:"id"`
	Name                     string      `json:"name"`
	QueryEngine              QueryEngine `json:"query_engine"`
	Enabled                  bool        `json:"enabled"`
	MinReplicas              int         `json:"min_replicas"`
	MaxReplicas              int         `json:"max_replicas"`
	CooldownSeconds          int         `json:"cooldown_seconds"`
	ScaleToZero              bool        `json:"scale_to_zero"`
	QueuedQueriesThreshold   int         `json:"queued_queries_threshold"`
	RunningQueriesThreshold  int         `json:"running_queries_threshold"`
	LatencyThresholdSeconds  int         `json:"latency_threshold_seconds"`
	ScheduleEnabled          bool        `json:"schedule_enabled"`
	BusinessHoursMinReplicas *int        `json:"business_hours_min_replicas,omitempty"`
	BusinessHoursStart       *string     `json:"business_hours_start,omitempty"`
	BusinessHoursEnd         *string     `json:"business_hours_end,omitempty"`
	BusinessHoursTimezone    *string     `json:"business_hours_timezone,omitempty"`
	CreatedAt                time.Time   `json:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at"`
}

// CreateQueryScalingPolicyRequest is the request to create a query scaling policy.
type CreateQueryScalingPolicyRequest struct {
	Name                     string      `json:"name" binding:"required"`
	QueryEngine              QueryEngine `json:"query_engine" binding:"required"`
	Enabled                  *bool       `json:"enabled,omitempty"`
	MinReplicas              *int        `json:"min_replicas,omitempty"`
	MaxReplicas              *int        `json:"max_replicas,omitempty"`
	CooldownSeconds          *int        `json:"cooldown_seconds,omitempty"`
	ScaleToZero              *bool       `json:"scale_to_zero,omitempty"`
	QueuedQueriesThreshold   *int        `json:"queued_queries_threshold,omitempty"`
	RunningQueriesThreshold  *int        `json:"running_queries_threshold,omitempty"`
	LatencyThresholdSeconds  *int        `json:"latency_threshold_seconds,omitempty"`
	ScheduleEnabled          *bool       `json:"schedule_enabled,omitempty"`
	BusinessHoursMinReplicas *int        `json:"business_hours_min_replicas,omitempty"`
	BusinessHoursStart       *string     `json:"business_hours_start,omitempty"`
	BusinessHoursEnd         *string     `json:"business_hours_end,omitempty"`
	BusinessHoursTimezone    *string     `json:"business_hours_timezone,omitempty"`
}

// UpdateQueryScalingPolicyRequest is the request to update a query scaling policy.
type UpdateQueryScalingPolicyRequest struct {
	Name                     *string `json:"name,omitempty"`
	Enabled                  *bool   `json:"enabled,omitempty"`
	MinReplicas              *int    `json:"min_replicas,omitempty"`
	MaxReplicas              *int    `json:"max_replicas,omitempty"`
	CooldownSeconds          *int    `json:"cooldown_seconds,omitempty"`
	ScaleToZero              *bool   `json:"scale_to_zero,omitempty"`
	QueuedQueriesThreshold   *int    `json:"queued_queries_threshold,omitempty"`
	RunningQueriesThreshold  *int    `json:"running_queries_threshold,omitempty"`
	LatencyThresholdSeconds  *int    `json:"latency_threshold_seconds,omitempty"`
	ScheduleEnabled          *bool   `json:"schedule_enabled,omitempty"`
	BusinessHoursMinReplicas *int    `json:"business_hours_min_replicas,omitempty"`
	BusinessHoursStart       *string `json:"business_hours_start,omitempty"`
	BusinessHoursEnd         *string `json:"business_hours_end,omitempty"`
	BusinessHoursTimezone    *string `json:"business_hours_timezone,omitempty"`
}

// QueryScalingPolicyListResponse is the response for listing query scaling policies.
type QueryScalingPolicyListResponse struct {
	Policies []QueryScalingPolicy `json:"policies"`
	Total    int                  `json:"total"`
}

// QueryScalingHistoryEntry represents a scaling action in history.
type QueryScalingHistoryEntry struct {
	ID               uuid.UUID   `json:"id"`
	PolicyID         *uuid.UUID  `json:"policy_id,omitempty"`
	QueryEngine      QueryEngine `json:"query_engine"`
	Action           string      `json:"action"`
	PreviousReplicas int         `json:"previous_replicas"`
	NewReplicas      int         `json:"new_replicas"`
	TriggerReason    string      `json:"trigger_reason,omitempty"`
	TriggerValue     *float64    `json:"trigger_value,omitempty"`
	ExecutedAt       time.Time   `json:"executed_at"`
}

// QueryScalingHistoryResponse is the response for query scaling history.
type QueryScalingHistoryResponse struct {
	History []QueryScalingHistoryEntry `json:"history"`
	Total   int                        `json:"total"`
}

// QueryScalingMetrics represents current query engine metrics.
type QueryScalingMetrics struct {
	QueryEngine    QueryEngine `json:"query_engine"`
	QueuedQueries  int         `json:"queued_queries"`
	RunningQueries int         `json:"running_queries"`
	BlockedQueries int         `json:"blocked_queries"`
	ActiveWorkers  int         `json:"active_workers"`
	P95LatencyMs   *float64    `json:"p95_latency_ms,omitempty"`
	CollectedAt    time.Time   `json:"collected_at"`
}

// QueryScalingMetricsResponse is the response for query scaling metrics.
type QueryScalingMetricsResponse struct {
	Metrics []QueryScalingMetrics `json:"metrics"`
}
