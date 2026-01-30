# Research Findings: SCALE-001 - Scaling Engine and Policy Framework

## 1. Existing Scaling/Metrics Infrastructure

### KEDA Configuration (Already Implemented)
- **Location:** `charts/philotes-worker/templates/scaledobject.yaml`
- Supports multiple trigger types:
  - Prometheus (lag, buffer depth, throughput metrics)
  - PostgreSQL (direct query-based scaling)
  - CPU/Memory (fallback scalers)
- Helm values in `charts/philotes-worker/values.yaml` (lines 53-163)
- Features: cooldown periods, advanced scaling policies, scale-to-zero support

### Available Prometheus Metrics
- `philotes_cdc_lag_seconds` - replication lag
- `philotes_buffer_depth` - unprocessed events count
- `philotes_cdc_events_total` - total events processed
- `philotes_buffer_batches_total` - batch processing count
- Full metric definitions in `internal/metrics/metrics.go`

## 2. API and Service Patterns to Follow

### Layer Architecture
```
Handler → Service → Repository → Database
```

Example from `internal/api/handlers/pipelines.go`:
- **Handlers:** Parse requests, call service methods, return responses
- **Services:** Business logic, validation, error handling
- **Repositories:** Database operations

### Error Handling Patterns
- Located in `internal/api/models/error.go`
- RFC 7807 Problem Details responses
- Custom error types: NotFoundError, ConflictError, ValidationError

## 3. Database Patterns

### Location
- Schema files: `deployments/docker/init-scripts/`
- Example: `06-alerting-schema.sql`

### Key Patterns
- PostgreSQL with UUID primary keys (`gen_random_uuid()`)
- JSONB columns for flexible configuration
- `TIMESTAMPTZ` for timezone-aware timestamps
- Proper indexing and foreign key relationships

## 4. Background Worker Pattern (AlertManager)

**File:** `internal/alerting/manager.go`

```go
type Manager struct {
    pendingAlerts map[string]time.Time  // State tracking
    mu            sync.RWMutex           // Thread safety
    stopCh        chan struct{}          // Graceful shutdown
    stoppedCh     chan struct{}
    running       bool
}

// Key methods:
// - Start() - spawns evaluation goroutine
// - Stop() - graceful shutdown via channels
// - evaluationLoop() - periodic evaluation with ticker
```

## 5. Configuration Pattern

**File:** `internal/config/config.go`

```go
type AlertingConfig struct {
    Enabled              bool
    EvaluationInterval   time.Duration
    PrometheusURL        string
    // ...
}
```

Environment variables: `PHILOTES_ALERTING_ENABLED`, etc.

## 6. Prometheus Integration

**File:** `internal/alerting/evaluator.go`

- HTTP POST to `/api/v1/query`
- 30-second timeout
- JSON response parsing
- Proper error handling

## 7. Key Files to Create

### New Package: `internal/scaling/`
- `types.go` - ScalingPolicy, ScalingRule, ScalingSchedule models
- `repository.go` - Database operations
- `service.go` - Business logic
- `evaluator.go` - Policy evaluation (Prometheus queries)
- `executor.go` - Scaling action execution
- `manager.go` - Background evaluation loop
- `providers/` - Cloud provider integrations

### API Layer: `internal/api/`
- `models/scaling.go` - Request/response types
- `handlers/scaling.go` - HTTP endpoints
- `services/scaling.go` - API service layer

### Database Migration
- `deployments/docker/init-scripts/07-scaling-schema.sql`

## 8. Recommended Architecture

### Phase 1: Core Data Model & CRUD API
- Database schema for policies, rules, schedules, history
- REST API endpoints for policy management
- Follow existing handler → service → repository pattern

### Phase 2: Evaluation Engine
- Background manager with evaluation loop
- Multi-metric decision making (AND/OR logic)
- Duration-based thresholds (metric must exceed for X time)

### Phase 3: Scaling Executor
- Provider abstraction interface
- KEDA executor (patch ScaledObject replicas)
- Future: cloud provider executors (Hetzner, OVH, Scaleway)

### Phase 4: Audit & Dry-Run
- Scaling history table
- Dry-run mode flag
- Cost estimation (future)

## 9. Database Schema (Proposed)

```sql
-- Scaling policies
CREATE TABLE scaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    target_type TEXT NOT NULL,  -- "cdc-worker", "trino", "nodes"
    target_id UUID,
    min_replicas INT DEFAULT 1,
    max_replicas INT DEFAULT 10,
    cooldown_seconds INT DEFAULT 300,
    max_hourly_cost FLOAT8,
    scale_to_zero BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Scaling rules
CREATE TABLE scaling_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES scaling_policies(id) ON DELETE CASCADE,
    rule_type TEXT NOT NULL,  -- "scale_up", "scale_down"
    metric TEXT NOT NULL,
    operator TEXT NOT NULL,   -- "gt", "lt", "gte", "lte"
    threshold FLOAT8 NOT NULL,
    duration TEXT NOT NULL,   -- "5m", "10m"
    scale_by INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Scaling schedules
CREATE TABLE scaling_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES scaling_policies(id) ON DELETE CASCADE,
    cron_expression TEXT NOT NULL,
    desired_replicas INT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Scaling history
CREATE TABLE scaling_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES scaling_policies(id) ON DELETE SET NULL,
    policy_name TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID,
    previous_replicas INT,
    new_replicas INT,
    reason TEXT,
    dry_run BOOLEAN DEFAULT false,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 10. Dependencies to Add

### Required
- `github.com/robfig/cron/v3` - Cron expression parsing

### Future (for cloud provider scaling)
- `k8s.io/client-go` - Kubernetes API client
- Hetzner Cloud SDK
- OVHcloud SDK
- Scaleway SDK

## 11. Blockers & Concerns

1. **Kubernetes Client:** Not yet in codebase, needed for KEDA integration
2. **Cloud Provider SDKs:** Defer to SCALE-003 (node-level scaling)
3. **Cost Estimation:** Defer to future - need pricing API integrations
4. **Multi-metric Logic:** Need to support AND/OR conditions between rules

## 12. Recommended Scope for This Issue

Focus on **MVP functionality**:
1. ✅ Policy data model and database schema
2. ✅ Full CRUD REST API for policies
3. ✅ Background evaluation engine
4. ✅ Prometheus metric evaluation
5. ✅ Scaling history/audit log
6. ✅ Dry-run mode
7. ⏳ KEDA integration (stub/interface only - defer actual k8s client)
8. ⏳ Cloud provider integration (defer to SCALE-003)
9. ⏳ Cost estimation (defer - need pricing APIs)

This provides a working scaling engine with policy management and evaluation, ready for KEDA and cloud provider executors in follow-up issues.
