# Research Findings - Issue #14: Prometheus Metrics Integration

## 1. Current State

### Existing Infrastructure
- Prometheus and Grafana are already configured in `deployments/docker/docker-compose.yml`
- Prometheus config is set up at `deployments/docker/prometheus.yml`
- Prometheus expects metrics endpoints:
  - API: `host.docker.internal:8080/metrics`
  - Worker: `host.docker.internal:9091/metrics`
- Grafana is configured with provisioning directories for dashboards and datasources

### Configuration Already Exists
`internal/config/config.go` has `MetricsConfig`:
- `Enabled`: defaults to `true` (environment: `PHILOTES_METRICS_ENABLED`)
- `ListenAddr`: defaults to `:9090` (environment: `PHILOTES_METRICS_LISTEN_ADDR`)

### Missing Dependency
The Prometheus client library (`prometheus/client_golang`) is NOT in `go.mod` - needs to be added.

## 2. Key Files to Modify

### Primary Files
| File | Purpose |
|------|---------|
| `cmd/philotes-api/main.go` | Add metrics server initialization |
| `cmd/philotes-worker/main.go` | Add metrics server initialization |
| `internal/api/server.go` | Register `/metrics` endpoint |
| `internal/cdc/pipeline/pipeline.go` | Add CDC metrics |
| `internal/cdc/buffer/batch.go` | Add buffer/batch metrics |
| `internal/iceberg/writer/writer.go` | Add Iceberg commit metrics |
| `go.mod` | Add Prometheus client library |

### New Files to Create
| File | Purpose |
|------|---------|
| `internal/metrics/metrics.go` | Central metrics registry and initialization |
| `internal/metrics/cdc.go` | CDC-specific metrics definitions |
| `internal/metrics/api.go` | API-specific metrics definitions |
| `internal/metrics/iceberg.go` | Iceberg writer metrics definitions |
| `internal/api/middleware/metrics.go` | HTTP metrics middleware |
| `deployments/docker/grafana/provisioning/dashboards/*.json` | Grafana dashboards |

## 3. Existing Patterns to Follow

### Health Check Pattern (excellent precedent)
Located in `internal/cdc/health/health.go`:
- `Manager` struct with thread-safe `Register()` and `CheckAll()`
- `Server` on separate port with HTTP handler
- Extensible `HealthChecker` interface
- Component-based architecture

### Middleware Pattern
From `internal/api/middleware/`:
- Request logging middleware captures method, path, status, latency_ms
- All middleware follow gin HandlerFunc pattern
- Applied in `internal/api/server.go`

### Data Collection Pattern
Pipeline has `Stats` struct with:
- EventsProcessed, EventsBuffered, LastEventTime
- LastCheckpointLSN, LastCheckpointAt, Errors, RetryCount, State

Buffer has `Stats` struct with:
- TotalEvents, UnprocessedEvents, OldestUnprocessed, Lag
- BatchesProcessed, EventsProcessed, EventsFailed, RetryCount, DLQCount

## 4. Metrics to Implement

### CDC Metrics
```
philotes_cdc_events_total{source,table,operation}     - Counter
philotes_cdc_lag_seconds{source,table}                - Gauge
philotes_cdc_errors_total{source,table}               - Counter
philotes_cdc_retries_total{source}                    - Counter
philotes_buffer_depth{source}                         - Gauge
```

### API Metrics
```
philotes_api_requests_total{endpoint,method,status}           - Counter
philotes_api_request_duration_seconds{endpoint,method}        - Histogram
```

### Iceberg Metrics
```
philotes_iceberg_commits_total{source,table}                  - Counter
philotes_iceberg_commit_duration_seconds{source,table}        - Histogram
philotes_iceberg_files_written_total{source,table}            - Counter
```

### Go Runtime (automatic)
```
go_goroutines
go_memstats_alloc_bytes
process_resident_memory_bytes
process_cpu_seconds_total
```

## 5. Recommended Library

**prometheus/client_golang v1.18+**
- Industry standard Go Prometheus client
- Supports all metric types: Counter, Gauge, Histogram
- Automatic Go runtime metrics
- HTTP handler for `/metrics` endpoint

## 6. Implementation Approach

### Phase 1: Foundation
1. Add `prometheus/client_golang` to `go.mod`
2. Create `internal/metrics/metrics.go` with registry initialization
3. Add metrics middleware to API server
4. Expose `/metrics` endpoint in both API and Worker

### Phase 2: CDC Metrics
1. Instrument `internal/cdc/pipeline/pipeline.go`
2. Instrument `internal/cdc/buffer/batch.go`

### Phase 3: Iceberg Metrics
1. Instrument `internal/iceberg/writer/writer.go`

### Phase 4: Grafana Dashboards
1. Create dashboard JSON files for CDC, API, and system monitoring

## 7. Questions Resolved

| Question | Decision |
|----------|----------|
| Metrics port | Same as API (`:8080`) with `/metrics` endpoint |
| Histogram buckets | `.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10` |
| Table-level labels | Yes, but consider aggregation for high-cardinality cases |
| Health as metrics | Yes, expose as gauge with numeric status |
