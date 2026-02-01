# Implementation Plan: Query Engine Auto-scaling

## Summary

Implement KEDA-based auto-scaling for Trino query engine workers based on query metrics. This extends the existing Trino Helm chart with KEDA ScaledObject support, following the patterns established in the philotes-worker chart.

## Approach

1. **KEDA ScaledObject for Trino** - Primary scaling mechanism using Prometheus metrics
2. **Query Metrics Integration** - Expose Trino query metrics via the existing API
3. **Configuration** - Add KEDA configuration to Trino values.yaml
4. **Documentation** - Document scaling configuration and metrics

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `charts/trino/templates/scaledobject.yaml` | KEDA ScaledObject for Trino workers | ~120 |
| `internal/scaling/query/collector.go` | Query metrics collector | ~200 |
| `internal/scaling/query/policy.go` | Query-specific policy defaults | ~150 |
| `internal/api/handlers/queryscaling.go` | API handlers for query scaling | ~150 |
| `internal/api/models/queryscaling.go` | Request/response types | ~80 |
| `internal/api/services/queryscaling.go` | Query scaling service | ~200 |
| `deployments/docker/init-scripts/16-query-scaling-schema.sql` | Database schema | ~50 |

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `charts/trino/values.yaml` | Add KEDA configuration section | ~80 |
| `charts/trino/templates/hpa.yaml` | Conditionally disable when KEDA enabled | ~5 |
| `internal/config/config.go` | Add QueryScalingConfig | ~30 |
| `internal/api/server.go` | Register query scaling routes | ~10 |

## KEDA Configuration

### Prometheus Triggers for Trino

```yaml
keda:
  enabled: false
  pollingInterval: 30
  cooldownPeriod: 300
  minReplicaCount: 1
  maxReplicas: 10
  idleReplicaCount: 0  # Scale to zero support

  prometheus:
    enabled: true
    serverAddress: "http://prometheus:9090"

    # Scale up when queries are queued
    queuedQueriesTrigger:
      enabled: true
      metricName: "trino_queued_queries"
      query: "trino_queued_queries"
      threshold: "5"
      activationThreshold: "1"  # Wake from zero when any query queued

    # Scale up when running queries exceed threshold
    runningQueriesTrigger:
      enabled: true
      metricName: "trino_running_queries"
      query: "trino_running_queries"
      threshold: "10"

    # Scale based on query latency
    latencyTrigger:
      enabled: false
      metricName: "trino_query_execution_time_p95"
      query: "histogram_quantile(0.95, trino_query_execution_time_bucket)"
      threshold: "30"  # 30 seconds

  # Advanced scaling behavior
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 300
          policies:
            - type: Percent
              value: 50
              periodSeconds: 60
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
            - type: Pods
              value: 2
              periodSeconds: 30
```

## Database Schema

```sql
-- Query scaling policies
CREATE TABLE query_scaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    query_engine VARCHAR(50) NOT NULL,  -- 'trino', 'risingwave'
    enabled BOOLEAN DEFAULT TRUE,
    min_replicas INTEGER DEFAULT 1,
    max_replicas INTEGER DEFAULT 10,
    cooldown_seconds INTEGER DEFAULT 300,
    scale_to_zero BOOLEAN DEFAULT FALSE,

    -- Trigger thresholds
    queued_queries_threshold INTEGER DEFAULT 5,
    running_queries_threshold INTEGER DEFAULT 10,
    latency_threshold_seconds INTEGER DEFAULT 30,

    -- Schedule
    schedule_enabled BOOLEAN DEFAULT FALSE,
    business_hours_min_replicas INTEGER,
    business_hours_start TIME,
    business_hours_end TIME,
    business_hours_timezone VARCHAR(50),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Query scaling history
CREATE TABLE query_scaling_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES query_scaling_policies(id),
    query_engine VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,  -- 'scale_up', 'scale_down', 'scale_to_zero', 'wake'
    previous_replicas INTEGER NOT NULL,
    new_replicas INTEGER NOT NULL,
    trigger_reason VARCHAR(255),
    trigger_value DECIMAL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_query_scaling_history_policy ON query_scaling_history(policy_id);
CREATE INDEX idx_query_scaling_history_engine ON query_scaling_history(query_engine);
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/query-scaling/policies` | List query scaling policies |
| `POST` | `/api/v1/query-scaling/policies` | Create scaling policy |
| `GET` | `/api/v1/query-scaling/policies/:id` | Get policy details |
| `PUT` | `/api/v1/query-scaling/policies/:id` | Update policy |
| `DELETE` | `/api/v1/query-scaling/policies/:id` | Delete policy |
| `GET` | `/api/v1/query-scaling/metrics` | Get current query metrics |
| `GET` | `/api/v1/query-scaling/history` | Get scaling history |

## Task Order

1. Create database migration schema
2. Add KEDA configuration to Trino values.yaml
3. Create KEDA ScaledObject template
4. Update HPA to be conditional
5. Add QueryScalingConfig to config.go
6. Implement query metrics collector
7. Implement API handlers, models, services
8. Register routes in server.go
9. Run tests and lint
10. Update documentation

## Verification

1. `helm template charts/trino` - Verify KEDA templates render correctly
2. `go build ./...` - Verify Go code compiles
3. `make lint` - Verify code quality
4. `make test` - Verify tests pass
5. Manual test with KEDA installed in cluster

## Estimate

~1,100 LOC (reduced from 6,000 due to existing infrastructure)
