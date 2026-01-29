// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"
)

// PipelineMetrics represents the current metrics for a pipeline.
type PipelineMetrics struct {
	// PipelineID is the ID of the pipeline.
	PipelineID uuid.UUID `json:"pipeline_id"`

	// Status is the current pipeline status.
	Status PipelineStatus `json:"status"`

	// EventsProcessed is the total number of events processed.
	EventsProcessed int64 `json:"events_processed"`

	// EventsPerSecond is the current rate of events being processed.
	EventsPerSecond float64 `json:"events_per_second"`

	// LagSeconds is the current replication lag in seconds.
	LagSeconds float64 `json:"lag_seconds"`

	// LagP95Seconds is the 95th percentile replication lag.
	LagP95Seconds float64 `json:"lag_p95_seconds"`

	// BufferDepth is the number of unprocessed events in the buffer.
	BufferDepth int64 `json:"buffer_depth"`

	// ErrorCount is the total number of errors encountered.
	ErrorCount int64 `json:"error_count"`

	// IcebergCommits is the total number of Iceberg commits made.
	IcebergCommits int64 `json:"iceberg_commits"`

	// IcebergBytesWritten is the total bytes written to Iceberg.
	IcebergBytesWritten int64 `json:"iceberg_bytes_written"`

	// LastEventAt is the timestamp of the last processed event.
	LastEventAt *time.Time `json:"last_event_at,omitempty"`

	// Uptime is the duration since the pipeline started.
	Uptime string `json:"uptime,omitempty"`

	// Tables contains per-table metrics.
	Tables []TableMetrics `json:"tables,omitempty"`
}

// TableMetrics represents metrics for a specific table.
type TableMetrics struct {
	// Schema is the source schema name.
	Schema string `json:"schema"`

	// Table is the source table name.
	Table string `json:"table"`

	// EventsProcessed is the total events processed for this table.
	EventsProcessed int64 `json:"events_processed"`

	// LagSeconds is the current replication lag for this table.
	LagSeconds float64 `json:"lag_seconds"`

	// LastEventAt is the timestamp of the last event for this table.
	LastEventAt *time.Time `json:"last_event_at,omitempty"`
}

// MetricsHistory represents historical metrics data for a pipeline.
type MetricsHistory struct {
	// PipelineID is the ID of the pipeline.
	PipelineID string `json:"pipeline_id"`

	// TimeRange is the time range of the data (e.g., "1h", "24h").
	TimeRange string `json:"time_range"`

	// DataPoints contains the time-series data.
	DataPoints []MetricsDataPoint `json:"data_points"`
}

// MetricsDataPoint represents a single point in the metrics time series.
type MetricsDataPoint struct {
	// Timestamp is the time of this data point.
	Timestamp time.Time `json:"timestamp"`

	// EventsPerSecond is the events per second at this point.
	EventsPerSecond float64 `json:"events_per_second"`

	// LagSeconds is the replication lag at this point.
	LagSeconds float64 `json:"lag_seconds"`

	// BufferDepth is the buffer depth at this point.
	BufferDepth int64 `json:"buffer_depth"`

	// ErrorCount is the cumulative error count at this point.
	ErrorCount int64 `json:"error_count"`
}

// PipelineMetricsResponse wraps pipeline metrics for API responses.
type PipelineMetricsResponse struct {
	Metrics *PipelineMetrics `json:"metrics"`
}

// MetricsHistoryResponse wraps metrics history for API responses.
type MetricsHistoryResponse struct {
	History *MetricsHistory `json:"history"`
}
