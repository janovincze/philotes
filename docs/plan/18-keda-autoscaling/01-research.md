# Research: KEDA Autoscaling Configuration (Issue #18)

## Key Files to Modify

### Existing Files

| File | Purpose | Status |
|------|---------|--------|
| `charts/philotes-worker/templates/scaledobject.yaml` | ScaledObject template | Has PostgreSQL stub, needs Prometheus scalers |
| `charts/philotes-worker/values.yaml` | KEDA configuration values | Has `keda` section, needs expansion |
| `charts/philotes-worker/templates/deployment.yaml` | Worker deployment | Already conditionally disables replicas for KEDA |
| `charts/philotes-worker/templates/servicemonitor.yaml` | Prometheus scraping | Exists, properly configured |

### New Files to Create

| File | Purpose |
|------|---------|
| `charts/philotes-worker/templates/triggerauthentication.yaml` | KEDA trigger auth for Prometheus |
| `charts/philotes-worker/templates/keda-rbac.yaml` | RBAC for KEDA metrics access (if needed) |

## Available Prometheus Metrics for Scaling

From `internal/metrics/metrics.go`:

### Primary Scaling Metrics

| Metric | Type | Labels | Use Case |
|--------|------|--------|----------|
| `philotes_cdc_lag_seconds` | Gauge | source, table | Scale when replication lag exceeds threshold |
| `philotes_buffer_depth` | Gauge | source | Scale based on unprocessed event queue depth |
| `philotes_buffer_events_processed_total` | Counter | source | Rate-based scaling (events/sec) |

### Secondary/Monitoring Metrics

| Metric | Type | Labels | Use Case |
|--------|------|--------|----------|
| `philotes_cdc_events_total` | Counter | source, table, operation | Track event processing rate |
| `philotes_buffer_batches_total` | Counter | source, status | Monitor batch processing rate |
| `philotes_cdc_pipeline_state` | Gauge | source | Avoid scaling failed pipelines |
| `philotes_cdc_errors_total` | Counter | source, error_type | Scale back if error rate increases |
| `philotes_buffer_dlq_total` | Counter | source | Detect DLQ growth issues |

## Current KEDA Configuration Stub

From `charts/philotes-worker/values.yaml` (lines 52-83):

```yaml
keda:
  enabled: false
  minReplicas: 1
  maxReplicas: 5
  pollingInterval: 30
  cooldownPeriod: 300
  postgresql:
    host: ""
    database: ""
    user: ""
    password: ""
    existingSecret: ""
    passwordKey: "password"
  triggers:
    - type: postgresql
      metadata:
        query: "SELECT COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn), 0) FROM pg_replication_slots WHERE slot_name = 'philotes_cdc'"
        targetQueryValue: "1000000"  # 1MB lag threshold
```

## Current ScaledObject Template

The existing `scaledobject.yaml` has:
- Basic structure with trigger range
- PostgreSQL authentication support
- TriggerAuthentication resource for password management
- Dynamic trigger configuration from values

## Recommended KEDA Scalers

### 1. Prometheus Scaler (Primary - Recommended)

```yaml
triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus:9090
      metricName: philotes_cdc_lag_seconds
      threshold: '300'  # 5 minutes lag
      query: max(philotes_cdc_lag_seconds)
```

**Benefits:**
- Works with metric aggregation
- More reliable than direct database queries
- Integrates with existing observability stack

### 2. PostgreSQL Scaler (Alternative)

Already stubbed - good for deployments without Prometheus.

### 3. CPU/Memory Scaler (Fallback)

Standard HPA-style scaling as a safety net.

## Implementation Approach

### Phase 1: Prometheus-based Scaling
- Add Prometheus scaler trigger
- Use `philotes_cdc_lag_seconds` as primary metric
- Default threshold: 300 seconds (5 min lag)
- Min replicas: 1, Max replicas: 5

### Phase 2: Multi-metric Scaling
- Add `philotes_buffer_depth` as secondary trigger
- Add rate-based scaling using `philotes_buffer_events_processed_total`
- Combine metrics with appropriate weights

### Phase 3: Advanced Features
- Scale-to-zero with activation threshold
- Separate scale-up and scale-down policies
- Advanced KEDA v2 scaling behaviors

## Architecture Decisions

### Scaling Strategy

1. **Primary Trigger:** Replication lag (`philotes_cdc_lag_seconds`)
   - Scale up when lag > 5 minutes (300s)
   - Scale down when lag < 1 minute (60s)

2. **Secondary Trigger:** Buffer depth (`philotes_buffer_depth`)
   - Scale up when depth > 8000 events
   - Scale down when depth < 2000 events

3. **Cooldown Periods:**
   - Scale-up: 60 seconds (react quickly to load)
   - Scale-down: 300 seconds (avoid thrashing)

### Scale-to-Zero Support

KEDA 2.8+ supports scale-to-zero with activation triggers:
- `minReplicaCount: 0`
- `activationThreshold` to wake up when metrics appear
- Separate `idleReplicaCount` for warm standby

## Blockers/Questions

1. **PostgreSQL LAG Query:** Current query returns bytes, may need adjustment for specific PostgreSQL versions
2. **Prometheus URL:** Needs to be configurable per environment
3. **Metric Aggregation:** Need to decide: scale per-source or aggregate all sources
4. **PDB Respect:** Verify scaling respects Pod Disruption Budget

## Existing Infrastructure

- ✓ Metrics exposed (issue #14)
- ✓ Helm charts structure (issue #16)
- ✓ Health check endpoints
- ✓ ScaledObject template stub
- ✓ ServiceMonitor template
