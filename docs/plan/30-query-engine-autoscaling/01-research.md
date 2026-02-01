# Research: Query Engine Auto-scaling

## Executive Summary

The Philotes codebase has comprehensive scaling infrastructure already in place. The scaling engine (Issue #26), scale-to-zero (Issue #29), and Trino integration (Issue #21) provide all the foundational components needed. This issue focuses on applying existing patterns to Trino-specific metrics.

## Existing Infrastructure

### 1. KEDA ScaledObject Pattern

**Location:** `charts/philotes-worker/templates/scaledobject.yaml`

Features already implemented:
- Multi-trigger support (Prometheus, PostgreSQL, CPU/Memory)
- Scale-to-zero capability
- Advanced scaling policies with separate scale-up/scale-down behavior
- TriggerAuthentication for secure password management
- Pod Disruption Budget support

### 2. Scaling Engine Architecture

**Location:** `internal/scaling/`

Core components:
- **Manager** - Periodic evaluation loop
- **Evaluator** - Prometheus queries and rule evaluation
- **Executor** - KEDA and Kubernetes integration
- **Types** - Policy, Rule, Schedule, Decision models

**Target Types Already Defined:**
```go
const (
    TargetTrino TargetType = "trino"
    TargetRisingWave TargetType = "risingwave"
)
```

### 3. Trino Metrics

**ServiceMonitor:** `charts/trino/templates/servicemonitor.yaml`
- Scrape path: `/v1/jmx/mbean`
- Interval: 30s
- Timeout: 10s

**Key Trino Metrics Available:**
- `trino_queued_queries` - Queries waiting to execute
- `trino_running_queries` - Currently executing queries
- `trino_blocked_queries` - Queries blocked on resources
- `trino_active_workers` - Active worker nodes
- JVM memory metrics

### 4. Scale-to-Zero Pattern

**Location:** `internal/scaling/idle/` and `internal/scaling/wake/`

Already supports:
- Idle state tracking
- Wake triggers (manual, scheduled, webhook, api_request)
- Cost savings metrics

## Key Files to Create/Modify

### Create
| File | Purpose |
|------|---------|
| `charts/trino/templates/scaledobject.yaml` | KEDA ScaledObject for Trino workers |
| `charts/trino/templates/triggerauthentication.yaml` | Auth for Prometheus triggers |
| `internal/scaling/query/metrics.go` | Query engine metrics collector |
| `internal/scaling/query/policy.go` | Query-specific scaling policies |

### Modify
| File | Changes |
|------|---------|
| `charts/trino/values.yaml` | Add KEDA configuration section |
| `charts/trino/templates/hpa.yaml` | Make conditional on KEDA disable |
| `internal/config/config.go` | Add QueryScalingConfig |

## Recommended Approach

1. **Phase 1: KEDA ScaledObject for Trino**
   - Create ScaledObject template mirroring worker pattern
   - Configure Prometheus triggers for query metrics
   - Support scale-to-zero with activation threshold

2. **Phase 2: Query Metrics Integration**
   - Add query-specific metrics collector
   - Expose metrics via Prometheus endpoint
   - Track queue depth, latency, concurrency

3. **Phase 3: API Endpoints**
   - Add endpoints for query scaling policies
   - Integrate with existing scaling service

4. **Phase 4: Documentation**
   - Document metrics and configuration
   - Provide example scaling policies

## Trino JMX Metrics Mapping

| Trino JMX Bean | Prometheus Metric | Scaling Use |
|----------------|-------------------|-------------|
| `trino.server:name=Query,type=QueryManager` | `trino_queued_queries` | Scale up |
| `trino.server:name=Query,type=QueryManager` | `trino_running_queries` | Scale up |
| `trino.execution:name=TaskExecutor` | `trino_active_workers` | Scale decision |
| JVM Memory | `jvm_memory_used_bytes` | Scale down |

## Blockers

None - all infrastructure is in place.
