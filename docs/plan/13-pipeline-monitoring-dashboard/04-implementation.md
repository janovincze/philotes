# Implementation Details - Issue #13: Backend Metrics API

## Files Created

### 1. `internal/api/models/metrics.go`
Defines the metrics response types:
- `PipelineMetrics` - Current metrics snapshot for a pipeline
- `TableMetrics` - Per-table metrics
- `MetricsHistory` - Historical time-series data
- `MetricsDataPoint` - Single point in the time series

### 2. `internal/api/services/prometheus.go`
Prometheus HTTP client for querying metrics:
- `QueryInstant()` - Instant queries for current values
- `QueryRange()` - Range queries for historical data
- Helper functions: `GetScalarValue()`, `GetScalarInt()`, `ParseTimeSeriesValues()`
- `IsAvailable()` - Health check for Prometheus connectivity

### 3. `internal/api/services/metrics.go`
Business logic for metrics retrieval:
- `GetPipelineMetrics()` - Fetches current metrics (parallel queries for performance)
- `GetPipelineMetricsHistory()` - Fetches historical time-series data
- `ParseTimeRange()` - Parses time range strings (15m, 1h, 6h, 24h, 7d)
- `getTableMetrics()` - Per-table metrics breakdown

### 4. `internal/api/handlers/metrics.go`
HTTP handlers for metrics endpoints:
- `GET /api/v1/pipelines/:id/metrics` - Current metrics
- `GET /api/v1/pipelines/:id/metrics/history?range=1h` - Historical metrics

## Files Modified

### `internal/api/server.go`
- Added `metricsService` field to `Server` struct
- Added `MetricsService` field to `ServerConfig` struct
- Registered new metrics routes in `registerRoutes()`

## Prometheus Queries Used

| Metric | Query |
|--------|-------|
| Events total | `sum(philotes_cdc_events_total{source="<name>"})` |
| Events/sec | `sum(rate(philotes_cdc_events_total{source="<name>"}[1m]))` |
| Lag | `max(philotes_cdc_lag_seconds{source="<name>"})` |
| Buffer depth | `sum(philotes_buffer_depth{source="<name>"})` |
| Errors | `sum(philotes_cdc_errors_total{source="<name>"})` |
| Iceberg commits | `sum(philotes_iceberg_commits_total{source="<name>"})` |
| Iceberg bytes | `sum(philotes_iceberg_bytes_written_total{source="<name>"})` |

## API Response Examples

### GET /api/v1/pipelines/:id/metrics
```json
{
  "metrics": {
    "pipeline_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "running",
    "events_processed": 15000,
    "events_per_second": 250.5,
    "lag_seconds": 0.5,
    "lag_p95_seconds": 0,
    "buffer_depth": 100,
    "error_count": 2,
    "iceberg_commits": 150,
    "iceberg_bytes_written": 1073741824,
    "uptime": "2h30m15s",
    "tables": [
      {
        "schema": "public",
        "table": "users",
        "events_processed": 5000,
        "lag_seconds": 0.3
      }
    ]
  }
}
```

### GET /api/v1/pipelines/:id/metrics/history?range=1h
```json
{
  "history": {
    "pipeline_id": "123e4567-e89b-12d3-a456-426614174000",
    "time_range": "1h",
    "data_points": [
      {
        "timestamp": "2026-01-29T10:00:00Z",
        "events_per_second": 200,
        "lag_seconds": 0.4,
        "buffer_depth": 50,
        "error_count": 0
      }
    ]
  }
}
```

## Configuration

The Prometheus URL is configured via environment variable:
- `PHILOTES_PROMETHEUS_URL` (default: `http://localhost:9090`)

This is already used by the alerting and scaling services.

## Verification

```bash
# Build succeeds
go build ./...

# go vet passes
go vet ./...

# Tests pass
make test
```
