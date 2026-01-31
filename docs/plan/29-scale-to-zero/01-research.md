# Research: Scale-to-Zero Implementation

## Existing Infrastructure

### Scaling Core (Already Exists)

| File | Purpose |
|------|---------|
| `internal/scaling/types.go` | Core types: Policy, Rule, Schedule, Decision, State |
| `internal/scaling/manager.go` | Main scaling evaluation loop |
| `internal/scaling/evaluator.go` | Prometheus-based rule evaluation |
| `internal/scaling/executor.go` | Scaling execution (KEDA, Logging, Composite) |

**Key Discovery:** Scale-to-zero flag already exists in `Policy` struct (line 150 in types.go) and database schema (line 18 in 07-scaling-schema.sql). The evaluator already supports scaling to 0 if `policy.ScaleToZero` is true.

### Kubernetes Integration

| File | Purpose |
|------|---------|
| `internal/scaling/kubernetes/client.go` | K8s API wrapper |
| `internal/scaling/kubernetes/drain.go` | Node drain/cordon operations |
| `internal/scaling/kubernetes/monitor.go` | Cluster capacity monitoring |

### CDC Pipeline

| File | Purpose |
|------|---------|
| `internal/cdc/pipeline/pipeline.go` | Main pipeline orchestration |
| `internal/cdc/pipeline/state.go` | State machine with transitions |
| `internal/cdc/checkpoint/checkpoint.go` | Checkpoint interface |
| `internal/cdc/buffer/buffer.go` | Buffer interface |

**Key Discovery:** Pipeline already tracks `LastEventTime` in stats. Graceful shutdown saves checkpoint on SIGTERM.

### API Layer

| File | Purpose |
|------|---------|
| `internal/api/handlers/scaling.go` | Scaling API endpoints |
| `internal/api/handlers/nodepool.go` | Node pool endpoints |

### Database Schema

`deployments/docker/init-scripts/07-scaling-schema.sql`:
- `scaling_policies` - Policy configuration with `scale_to_zero` boolean
- `scaling_rules` - Metric-based rules
- `scaling_history` - Audit log
- `scaling_state` - Current state tracking

## What's Already Working

1. **Scale-to-zero flag** in Policy struct and database
2. **Scale-down logic** in evaluator (line 157)
3. **Policy evaluation framework** with Prometheus integration
4. **State tracking** with database persistence
5. **Checkpoint system** for CDC position recovery
6. **Health monitoring** with liveness/readiness probes
7. **Kubernetes integration** for scaling deployments
8. **Graceful shutdown** saving final checkpoint

## What Needs to Be Built

### 1. Idle Detection System

```go
// New metrics to expose
cdc_last_event_timestamp_seconds  // Gauge: timestamp of last event
cdc_idle_duration_seconds         // Gauge: seconds since last event
cdc_buffer_is_empty               // Gauge: 1 if buffer empty, 0 otherwise
```

### 2. Wake Triggers

- `POST /api/v1/scaling/policies/:id/wake` - Manual wake endpoint
- `POST /api/v1/scaling/wake` - Wake all scaled-to-zero policies
- Webhook integration for external triggers
- Scheduled wake-up support

### 3. Keep-Alive Mechanism

Prevent flapping with:
- Grace period before scale-down
- Cool-down after scale-up
- Activity spike detection

### 4. Worker Lifecycle

- Graceful stop with checkpoint flush
- Fast cold-start from checkpoint
- Pod lifecycle hooks

### 5. Cost Tracking

- Track idle time per pipeline
- Calculate cost savings
- Report in dashboard

## Database Schema Additions

```sql
-- Add to scaling_policies
ALTER TABLE scaling_policies ADD COLUMN idle_threshold_seconds INTEGER DEFAULT 1800;
ALTER TABLE scaling_policies ADD COLUMN keep_alive_window_seconds INTEGER DEFAULT 300;
ALTER TABLE scaling_policies ADD COLUMN cold_start_timeout_seconds INTEGER DEFAULT 120;

-- New table for tracking idle state
CREATE TABLE scaling_idle_state (
    policy_id UUID PRIMARY KEY REFERENCES scaling_policies(id),
    last_activity_at TIMESTAMP WITH TIME ZONE,
    idle_since TIMESTAMP WITH TIME ZONE,
    scaled_to_zero_at TIMESTAMP WITH TIME ZONE,
    wake_reason TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Track cost savings
CREATE TABLE scaling_cost_savings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES scaling_policies(id),
    date DATE NOT NULL,
    idle_hours DECIMAL(10,2),
    estimated_savings DECIMAL(10,2),
    currency TEXT DEFAULT 'EUR',
    UNIQUE(policy_id, date)
);
```

## Recommended Architecture

### Scale-Down Flow

```
1. Pipeline publishes metrics:
   - cdc_buffer_unprocessed_events = 0
   - cdc_idle_duration_seconds > threshold

2. Scaling evaluator checks rule:
   - condition: buffer empty AND idle > 30min
   - duration: must be true for 5 minutes
   - action: scale to 0

3. Executor.Scale(policy, 0):
   - Update KEDA ScaledObject
   - Kubernetes scales deployment to 0
   - Pod receives SIGTERM

4. Worker graceful shutdown:
   - Flush checkpoint with final LSN
   - Close database connections
   - Exit cleanly
```

### Scale-Up (Wake) Flow

```
1. Trigger received:
   - API call: POST /scaling/policies/:id/wake
   - Schedule trigger
   - Webhook from source database

2. Wake handler:
   - Validate policy exists
   - Check if currently at 0
   - Update scaling_state.desired_replicas = 1

3. KEDA scales deployment:
   - New pod starts
   - Worker reads checkpoint
   - Resumes from last LSN

4. Update state:
   - Record wake reason
   - Clear idle tracking
```

## Files to Create/Modify

### New Files

| File | Purpose |
|------|---------|
| `internal/scaling/idle/detector.go` | Idle detection logic |
| `internal/scaling/idle/metrics.go` | Prometheus metrics for idle state |
| `internal/scaling/wake/handler.go` | Wake trigger handling |
| `internal/scaling/wake/scheduler.go` | Scheduled wake-up |
| `internal/api/handlers/wake.go` | Wake API endpoints |
| `deployments/docker/init-scripts/15-scale-to-zero-schema.sql` | DB migrations |

### Modify Existing

| File | Changes |
|------|---------|
| `internal/scaling/evaluator.go` | Add idle-aware evaluation |
| `internal/scaling/manager.go` | Integrate idle detector |
| `internal/cdc/pipeline/pipeline.go` | Expose idle metrics |
| `internal/api/server.go` | Register wake endpoints |
| `internal/config/config.go` | Add scale-to-zero config |

## Patterns to Follow

1. **State Management** - Use JSONB in PostgreSQL
2. **Health Checks** - Register custom checkers
3. **Graceful Lifecycle** - Use context.Done()
4. **Metrics** - slog + Prometheus
5. **History Recording** - Audit all scaling decisions

## Open Questions

1. **Worker Pod Association** - Use deployment name pattern or add tracking table?
2. **Source Database Listener** - Implement lightweight WAL listener for wake trigger?
3. **Multi-Pipeline** - Scale individual pipelines or entire worker deployment?
