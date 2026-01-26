# Implementation Plan - Issue #14: Prometheus Metrics Integration

## Overview

Add comprehensive Prometheus metrics to Philotes for monitoring CDC pipeline health, API performance, and enabling auto-scaling decisions via KEDA.

## Approach

Following the existing patterns in the codebase (health check system, middleware pattern), we will:

1. Create a centralized `internal/metrics` package for metric definitions
2. Add a metrics middleware for API request instrumentation
3. Instrument CDC pipeline and buffer for event metrics
4. Instrument Iceberg writer for commit metrics
5. Expose `/metrics` endpoint via the existing API server
6. Create Grafana dashboards for visualization

## Files to Create

| File | Purpose |
|------|---------|
| `internal/metrics/metrics.go` | Registry initialization, common types |
| `internal/metrics/collector.go` | Custom collector for CDC stats |
| `internal/api/middleware/metrics.go` | HTTP request metrics middleware |
| `deployments/docker/grafana/provisioning/dashboards/philotes.json` | Main Grafana dashboard |
| `deployments/docker/grafana/provisioning/dashboards/dashboard.yml` | Dashboard provisioning config |
| `deployments/docker/grafana/provisioning/datasources/datasource.yml` | Prometheus datasource config |

## Files to Modify

| File | Changes |
|------|---------|
| `go.mod` | Add `prometheus/client_golang` dependency |
| `internal/api/server.go` | Add metrics middleware and `/metrics` endpoint |
| `internal/cdc/pipeline/pipeline.go` | Add CDC event metrics calls |
| `internal/cdc/buffer/batch.go` | Add batch processing metrics |
| `internal/iceberg/writer/writer.go` | Add Iceberg commit metrics |
| `internal/config/config.go` | Metrics already configured, minor updates if needed |

## Task Breakdown

### Phase 1: Foundation (Core Metrics Infrastructure)

#### Task 1.1: Add Prometheus client dependency
```bash
go get github.com/prometheus/client_golang@latest
```

#### Task 1.2: Create metrics package (`internal/metrics/metrics.go`)
- Initialize Prometheus registry
- Define metric namespaces and subsystems
- Create helper functions for metric registration
- Define common label constants

```go
const (
    Namespace = "philotes"
    SubsystemCDC = "cdc"
    SubsystemAPI = "api"
    SubsystemIceberg = "iceberg"
    SubsystemBuffer = "buffer"
)

var (
    // CDC Metrics
    CDCEventsTotal *prometheus.CounterVec
    CDCLagSeconds *prometheus.GaugeVec
    CDCErrorsTotal *prometheus.CounterVec

    // API Metrics
    APIRequestsTotal *prometheus.CounterVec
    APIRequestDuration *prometheus.HistogramVec

    // Iceberg Metrics
    IcebergCommitsTotal *prometheus.CounterVec
    IcebergCommitDuration *prometheus.HistogramVec

    // Buffer Metrics
    BufferDepth *prometheus.GaugeVec
    BufferBatchesProcessed *prometheus.CounterVec
)
```

#### Task 1.3: Create CDC stats collector (`internal/metrics/collector.go`)
- Implement `prometheus.Collector` interface for pipeline stats
- Expose existing `pipeline.Stats` and `buffer.Stats` as metrics
- Support multiple pipelines via labels

### Phase 2: API Metrics

#### Task 2.1: Create metrics middleware (`internal/api/middleware/metrics.go`)
- Follow existing `logging.go` pattern
- Track request count, latency histogram
- Use path template (not full URL) to avoid cardinality explosion
- Labels: `endpoint`, `method`, `status`

```go
func Metrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start)

        metrics.APIRequestsTotal.WithLabelValues(
            c.FullPath(), c.Request.Method, strconv.Itoa(c.Writer.Status()),
        ).Inc()

        metrics.APIRequestDuration.WithLabelValues(
            c.FullPath(), c.Request.Method,
        ).Observe(duration.Seconds())
    }
}
```

#### Task 2.2: Register `/metrics` endpoint in server
- Add Prometheus HTTP handler to router
- Add metrics middleware to middleware chain (before logger)
- Ensure Go runtime metrics are exported

```go
// In registerRoutes()
s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### Phase 3: CDC Metrics

#### Task 3.1: Instrument pipeline events
- Increment counter for each event processed
- Track events by source, table, operation (INSERT/UPDATE/DELETE)
- Update lag gauge based on event timestamps

Key instrumentation points in `pipeline.go`:
- `Run()` event loop: increment `philotes_cdc_events_total`
- Error handling: increment `philotes_cdc_errors_total`
- State changes: update `philotes_cdc_pipeline_state` gauge

#### Task 3.2: Instrument buffer metrics
- Track buffer depth (unprocessed events)
- Track batch processing success/failure
- Track DLQ events

Key instrumentation points in `batch.go`:
- After batch success: increment `philotes_buffer_batches_total`
- Buffer depth: gauge of `UnprocessedEvents`

### Phase 4: Iceberg Metrics

#### Task 4.1: Instrument writer
- Track commits per table
- Track commit duration histogram
- Track files written

Key instrumentation points in `writer.go`:
- `WriteEvents()` success: increment `philotes_iceberg_commits_total`
- Commit timing: observe `philotes_iceberg_commit_duration_seconds`

### Phase 5: Grafana Dashboards

#### Task 5.1: Create Prometheus datasource config
```yaml
# deployments/docker/grafana/provisioning/datasources/datasource.yml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    url: http://prometheus:9090
    access: proxy
    isDefault: true
```

#### Task 5.2: Create dashboard provisioning config
```yaml
# deployments/docker/grafana/provisioning/dashboards/dashboard.yml
apiVersion: 1
providers:
  - name: 'Philotes'
    folder: ''
    type: file
    options:
      path: /etc/grafana/provisioning/dashboards
```

#### Task 5.3: Create main Grafana dashboard
Dashboard panels:
- **CDC Overview**: Events/sec, lag, errors
- **API Performance**: Request rate, latency percentiles, error rate
- **Iceberg Health**: Commits/sec, commit latency
- **Buffer Status**: Depth gauge, batch processing rate
- **System Health**: Go goroutines, memory, CPU

## Metrics Reference

### CDC Metrics
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `philotes_cdc_events_total` | Counter | source, table, operation | Total CDC events processed |
| `philotes_cdc_lag_seconds` | Gauge | source, table | Replication lag in seconds |
| `philotes_cdc_errors_total` | Counter | source, error_type | Total CDC errors |
| `philotes_cdc_retries_total` | Counter | source | Total retry attempts |

### API Metrics
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `philotes_api_requests_total` | Counter | endpoint, method, status | Total API requests |
| `philotes_api_request_duration_seconds` | Histogram | endpoint, method | Request latency |

### Iceberg Metrics
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `philotes_iceberg_commits_total` | Counter | source, table | Total Iceberg commits |
| `philotes_iceberg_commit_duration_seconds` | Histogram | source, table | Commit latency |
| `philotes_iceberg_files_written_total` | Counter | source, table | Total Parquet files written |

### Buffer Metrics
| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `philotes_buffer_depth` | Gauge | source | Unprocessed events in buffer |
| `philotes_buffer_batches_total` | Counter | source, status | Batches processed |

### Go Runtime (automatic)
- `go_goroutines`
- `go_memstats_alloc_bytes`
- `process_resident_memory_bytes`
- `process_cpu_seconds_total`

## Testing Strategy

1. **Unit tests**: Test metric registration and label validation
2. **Integration tests**: Verify `/metrics` endpoint returns expected format
3. **Manual verification**:
   - Start docker-compose
   - Generate CDC events
   - Verify metrics in Prometheus UI (localhost:9090)
   - Verify Grafana dashboard (localhost:3000)

## Implementation Order

1. Add Prometheus dependency
2. Create `internal/metrics/metrics.go` with metric definitions
3. Create API metrics middleware and register `/metrics` endpoint
4. Instrument CDC pipeline
5. Instrument buffer
6. Instrument Iceberg writer
7. Create Grafana provisioning files
8. Create Grafana dashboard JSON
9. Test end-to-end

## Rollback Plan

All changes are additive. If issues arise:
- Metrics can be disabled via `PHILOTES_METRICS_ENABLED=false`
- Middleware can be removed from chain
- No existing functionality is modified

## Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|----------------|
| Prometheus metrics endpoint (/metrics) | Task 2.2 |
| CDC metrics (events/sec, lag, errors) | Tasks 1.2, 3.1, 3.2 |
| API metrics (requests, latency, errors) | Tasks 2.1, 2.2 |
| Go runtime metrics | Automatic via promhttp |
| Custom business metrics | Tasks 3.1, 4.1 |
| Metric labels for multi-pipeline support | All metrics use source/pipeline labels |
| Grafana dashboard definitions | Tasks 5.1-5.3 |
