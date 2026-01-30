# Implementation Plan: SCALE-001 - Scaling Engine and Policy Framework

## Overview

Implement a central scaling engine that evaluates policies and triggers scaling actions based on CDC-specific metrics, schedules, and cost constraints. This MVP focuses on the core engine with KEDA integration via interface abstraction.

## Scope

### In Scope (This Issue)
- Scaling policy data model and database schema
- Full CRUD REST API for policies, rules, schedules
- Background policy evaluation engine
- Prometheus metric evaluation (reuse alerting patterns)
- Multi-metric decision making (AND logic for rules)
- Scaling history and audit log
- Dry-run mode for testing policies
- Executor interface with KEDA stub implementation

### Out of Scope (Future Issues)
- Cloud provider node scaling (SCALE-003)
- Dashboard UI (SCALE-002)
- Scale-to-zero warmup (SCALE-004)
- Cost estimation (needs pricing APIs)

## Files to Create

### Database
| File | Purpose |
|------|---------|
| `deployments/docker/init-scripts/07-scaling-schema.sql` | Database schema |

### Core Scaling Package (`internal/scaling/`)
| File | Purpose |
|------|---------|
| `types.go` | Domain models: Policy, Rule, Schedule, History |
| `repository.go` | Database operations |
| `service.go` | Business logic and validation |
| `evaluator.go` | Prometheus metric evaluation |
| `executor.go` | Executor interface and KEDA stub |
| `manager.go` | Background evaluation loop |

### API Layer
| File | Purpose |
|------|---------|
| `internal/api/models/scaling.go` | Request/response DTOs |
| `internal/api/handlers/scaling.go` | HTTP endpoint handlers |

### Configuration
| File | Purpose |
|------|---------|
| `internal/config/config.go` | Add ScalingConfig (modify) |

### Tests
| File | Purpose |
|------|---------|
| `internal/scaling/types_test.go` | Model tests |
| `internal/scaling/evaluator_test.go` | Evaluator tests |
| `internal/scaling/service_test.go` | Service tests |
| `internal/scaling/manager_test.go` | Manager tests |

## Task Breakdown

### Task 1: Database Schema
Create `07-scaling-schema.sql` with tables:
- `scaling_policies` - Main policy configuration
- `scaling_rules` - Scale up/down rules per policy
- `scaling_schedules` - Cron-based scheduling per policy
- `scaling_history` - Audit log of scaling actions

### Task 2: Core Types (`internal/scaling/types.go`)
```go
type ScalingPolicy struct {
    ID              uuid.UUID
    Name            string
    TargetType      TargetType  // CDCWorker, Trino, RisingWave, Nodes
    TargetID        *uuid.UUID
    MinReplicas     int
    MaxReplicas     int
    CooldownSeconds int
    MaxHourlyCost   *float64
    ScaleToZero     bool
    Enabled         bool
    ScaleUpRules    []ScalingRule
    ScaleDownRules  []ScalingRule
    Schedules       []ScalingSchedule
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type ScalingRule struct {
    ID        uuid.UUID
    PolicyID  uuid.UUID
    RuleType  RuleType  // ScaleUp, ScaleDown
    Metric    string
    Operator  Operator  // GT, LT, GTE, LTE
    Threshold float64
    Duration  time.Duration
    ScaleBy   int
}

type ScalingSchedule struct {
    ID              uuid.UUID
    PolicyID        uuid.UUID
    CronExpression  string
    DesiredReplicas int
    Timezone        string
}

type ScalingHistory struct {
    ID               uuid.UUID
    PolicyID         *uuid.UUID
    PolicyName       string
    Action           ScalingAction
    TargetType       TargetType
    TargetID         *uuid.UUID
    PreviousReplicas int
    NewReplicas      int
    Reason           string
    DryRun           bool
    ExecutedAt       time.Time
}
```

### Task 3: Repository (`internal/scaling/repository.go`)
CRUD operations for all tables:
- `CreatePolicy`, `GetPolicy`, `ListPolicies`, `UpdatePolicy`, `DeletePolicy`
- `CreateRule`, `GetRulesForPolicy`, `DeleteRule`
- `CreateSchedule`, `GetSchedulesForPolicy`, `DeleteSchedule`
- `CreateHistory`, `GetHistoryForPolicy`, `ListRecentHistory`

### Task 4: Service (`internal/scaling/service.go`)
Business logic:
- Policy validation (min < max, valid cron expressions)
- Rule validation (valid operators, positive thresholds)
- Aggregated policy loading (policy + rules + schedules)

### Task 5: Evaluator (`internal/scaling/evaluator.go`)
Prometheus integration (follow alerting/evaluator.go pattern):
- Query Prometheus for metric values
- Check if metric matches rule conditions
- Track duration-based thresholds (metric must exceed for X time)
- Return scaling decision (scale up, scale down, or no action)

### Task 6: Executor Interface (`internal/scaling/executor.go`)
```go
type Executor interface {
    GetCurrentReplicas(ctx context.Context, targetType TargetType, targetID *uuid.UUID) (int, error)
    Scale(ctx context.Context, targetType TargetType, targetID *uuid.UUID, replicas int, dryRun bool) error
}

type KEDAExecutor struct {
    // Stub implementation - logs actions but doesn't connect to K8s
    logger *slog.Logger
}

type LoggingExecutor struct {
    // For dry-run and testing
    logger *slog.Logger
}
```

### Task 7: Manager (`internal/scaling/manager.go`)
Background evaluation loop (follow alerting/manager.go pattern):
- Load enabled policies from database
- Evaluate each policy's rules against current metrics
- Check schedule overrides
- Respect cooldown periods
- Execute scaling actions via executor
- Record history

### Task 8: API Models (`internal/api/models/scaling.go`)
Request/response DTOs:
- `CreatePolicyRequest`, `UpdatePolicyRequest`
- `PolicyResponse`, `PolicyListResponse`
- `ScalingHistoryResponse`

### Task 9: API Handlers (`internal/api/handlers/scaling.go`)
REST endpoints:
- `POST /api/v1/scaling/policies` - Create policy
- `GET /api/v1/scaling/policies` - List policies
- `GET /api/v1/scaling/policies/:id` - Get policy
- `PUT /api/v1/scaling/policies/:id` - Update policy
- `DELETE /api/v1/scaling/policies/:id` - Delete policy
- `POST /api/v1/scaling/policies/:id/evaluate` - Manual evaluation (dry-run)
- `GET /api/v1/scaling/history` - List scaling history

### Task 10: Configuration
Add to `internal/config/config.go`:
```go
type ScalingConfig struct {
    Enabled            bool
    EvaluationInterval time.Duration
    PrometheusURL      string
    DefaultCooldown    int
}
```

### Task 11: Wire Everything Together
- Register handlers in API router
- Initialize manager in main
- Add health check for scaling engine

### Task 12: Tests
- Unit tests for types, evaluator, service
- Integration test for manager evaluation loop

## API Design

### Create Policy
```http
POST /api/v1/scaling/policies
Content-Type: application/json

{
  "name": "cdc-worker-autoscaling",
  "target_type": "cdc-worker",
  "target_id": "550e8400-e29b-41d4-a716-446655440000",
  "min_replicas": 1,
  "max_replicas": 10,
  "cooldown_seconds": 300,
  "scale_to_zero": false,
  "scale_up_rules": [
    {
      "metric": "philotes_cdc_lag_seconds",
      "operator": "gt",
      "threshold": 60,
      "duration": "5m",
      "scale_by": 2
    }
  ],
  "scale_down_rules": [
    {
      "metric": "philotes_cdc_lag_seconds",
      "operator": "lt",
      "threshold": 10,
      "duration": "10m",
      "scale_by": -1
    }
  ],
  "schedules": [
    {
      "cron_expression": "0 8 * * 1-5",
      "desired_replicas": 5,
      "timezone": "Europe/Budapest"
    }
  ]
}
```

### Response
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "name": "cdc-worker-autoscaling",
  "target_type": "cdc-worker",
  "target_id": "550e8400-e29b-41d4-a716-446655440000",
  "min_replicas": 1,
  "max_replicas": 10,
  "cooldown_seconds": 300,
  "scale_to_zero": false,
  "enabled": true,
  "scale_up_rules": [...],
  "scale_down_rules": [...],
  "schedules": [...],
  "created_at": "2026-01-28T12:00:00Z",
  "updated_at": "2026-01-28T12:00:00Z"
}
```

## Test Strategy

1. **Unit Tests:**
   - Type validation and parsing
   - Evaluator metric queries (mock HTTP)
   - Service business logic
   - Cooldown calculation

2. **Integration Tests:**
   - Repository CRUD with real PostgreSQL
   - Manager evaluation loop with mock executor

3. **Manual Testing:**
   - Create policies via API
   - Verify evaluation logs
   - Test dry-run mode

## Verification

```bash
# Run tests
make test

# Run linter
make lint

# Build
make build

# Start services
make docker-up

# Test API
curl -X POST http://localhost:8080/api/v1/scaling/policies \
  -H "Content-Type: application/json" \
  -d '{"name":"test-policy","target_type":"cdc-worker","min_replicas":1,"max_replicas":5}'
```

## Dependencies

Add to `go.mod`:
```
github.com/robfig/cron/v3 v3.0.1
```

## Notes

- KEDA executor is a stub in this issue - actual Kubernetes integration deferred
- Cloud provider scaling deferred to SCALE-003
- Cost estimation deferred - needs pricing API integration
- Schedule evaluation uses `github.com/robfig/cron` for cron parsing
