// Package metrics provides Prometheus metrics for Philotes components.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var registerOnce sync.Once

const (
	// Namespace is the Prometheus namespace for all Philotes metrics.
	Namespace = "philotes"

	// Subsystem constants for metric organization.
	SubsystemCDC     = "cdc"
	SubsystemAPI     = "api"
	SubsystemIceberg = "iceberg"
	SubsystemBuffer  = "buffer"
)

// Label constants for consistent labeling across metrics.
const (
	LabelSource    = "source"
	LabelTable     = "table"
	LabelOperation = "operation"
	LabelEndpoint  = "endpoint"
	LabelMethod    = "method"
	LabelStatus    = "status"
	LabelErrorType = "error_type"
)

var (
	// CDC Metrics

	// CDCEventsTotal counts the total number of CDC events processed.
	CDCEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCDC,
			Name:      "events_total",
			Help:      "Total number of CDC events processed",
		},
		[]string{LabelSource, LabelTable, LabelOperation},
	)

	// CDCLagSeconds tracks the replication lag in seconds.
	CDCLagSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCDC,
			Name:      "lag_seconds",
			Help:      "Current replication lag in seconds",
		},
		[]string{LabelSource, LabelTable},
	)

	// CDCErrorsTotal counts the total number of CDC errors.
	CDCErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCDC,
			Name:      "errors_total",
			Help:      "Total number of CDC errors",
		},
		[]string{LabelSource, LabelErrorType},
	)

	// CDCRetriesTotal counts the total number of retry attempts.
	CDCRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCDC,
			Name:      "retries_total",
			Help:      "Total number of retry attempts",
		},
		[]string{LabelSource},
	)

	// CDCPipelineState represents the current state of the pipeline.
	// Values: 0=stopped, 1=starting, 2=running, 3=paused, 4=failed
	CDCPipelineState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemCDC,
			Name:      "pipeline_state",
			Help:      "Current pipeline state (0=stopped, 1=starting, 2=running, 3=paused, 4=failed)",
		},
		[]string{LabelSource},
	)

	// API Metrics

	// APIRequestsTotal counts the total number of API requests.
	APIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemAPI,
			Name:      "requests_total",
			Help:      "Total number of API requests",
		},
		[]string{LabelEndpoint, LabelMethod, LabelStatus},
	)

	// APIRequestDuration tracks the duration of API requests.
	APIRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemAPI,
			Name:      "request_duration_seconds",
			Help:      "Duration of API requests in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{LabelEndpoint, LabelMethod},
	)

	// APIRequestSize tracks the size of API request bodies.
	APIRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemAPI,
			Name:      "request_size_bytes",
			Help:      "Size of API request bodies in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 10MB
		},
		[]string{LabelEndpoint, LabelMethod},
	)

	// APIResponseSize tracks the size of API response bodies.
	APIResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemAPI,
			Name:      "response_size_bytes",
			Help:      "Size of API response bodies in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 10MB
		},
		[]string{LabelEndpoint, LabelMethod},
	)

	// Iceberg Metrics

	// IcebergCommitsTotal counts the total number of Iceberg commits.
	IcebergCommitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemIceberg,
			Name:      "commits_total",
			Help:      "Total number of Iceberg commits",
		},
		[]string{LabelSource, LabelTable},
	)

	// IcebergCommitDuration tracks the duration of Iceberg commits.
	IcebergCommitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: SubsystemIceberg,
			Name:      "commit_duration_seconds",
			Help:      "Duration of Iceberg commits in seconds",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{LabelSource, LabelTable},
	)

	// IcebergFilesWrittenTotal counts the total number of Parquet files written.
	IcebergFilesWrittenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemIceberg,
			Name:      "files_written_total",
			Help:      "Total number of Parquet files written to Iceberg",
		},
		[]string{LabelSource, LabelTable},
	)

	// IcebergBytesWrittenTotal counts the total bytes written to Iceberg.
	IcebergBytesWrittenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemIceberg,
			Name:      "bytes_written_total",
			Help:      "Total bytes written to Iceberg",
		},
		[]string{LabelSource, LabelTable},
	)

	// Buffer Metrics

	// BufferDepth tracks the number of unprocessed events in the buffer.
	BufferDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SubsystemBuffer,
			Name:      "depth",
			Help:      "Number of unprocessed events in the buffer",
		},
		[]string{LabelSource},
	)

	// BufferBatchesTotal counts the total number of batches processed.
	BufferBatchesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemBuffer,
			Name:      "batches_total",
			Help:      "Total number of batches processed",
		},
		[]string{LabelSource, LabelStatus},
	)

	// BufferEventsProcessedTotal counts the total events processed from buffer.
	BufferEventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemBuffer,
			Name:      "events_processed_total",
			Help:      "Total number of events processed from buffer",
		},
		[]string{LabelSource},
	)

	// BufferDLQTotal counts events sent to dead letter queue.
	BufferDLQTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SubsystemBuffer,
			Name:      "dlq_total",
			Help:      "Total number of events sent to dead letter queue",
		},
		[]string{LabelSource},
	)

	// allMetrics contains all metrics for registration.
	allMetrics = []prometheus.Collector{
		// CDC
		CDCEventsTotal,
		CDCLagSeconds,
		CDCErrorsTotal,
		CDCRetriesTotal,
		CDCPipelineState,
		// API
		APIRequestsTotal,
		APIRequestDuration,
		APIRequestSize,
		APIResponseSize,
		// Iceberg
		IcebergCommitsTotal,
		IcebergCommitDuration,
		IcebergFilesWrittenTotal,
		IcebergBytesWrittenTotal,
		// Buffer
		BufferDepth,
		BufferBatchesTotal,
		BufferEventsProcessedTotal,
		BufferDLQTotal,
	}
)

// Register registers all Philotes metrics with the default Prometheus registry.
// It is safe to call multiple times; subsequent calls are no-ops.
func Register() {
	registerOnce.Do(func() {
		for _, m := range allMetrics {
			prometheus.MustRegister(m)
		}
	})
}

// RegisterWith registers all Philotes metrics with the given registry.
func RegisterWith(reg prometheus.Registerer) {
	for _, m := range allMetrics {
		reg.MustRegister(m)
	}
}

// NewRegistry creates a new Prometheus registry with all Philotes metrics
// and standard Go runtime collectors.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()

	// Register standard collectors
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register Philotes metrics
	RegisterWith(reg)

	return reg
}
