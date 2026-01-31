# Implementation Plan: Scale-to-Zero (Issue #29)

## Overview

Implement scale-to-zero functionality to completely shut down Philotes workers during idle periods and quickly resume when activity returns.

## Approach

Leverage the existing scaling infrastructure:
- `ScaleToZero` boolean already exists in Policy struct and database
- Evaluator already supports scaling to 0 when `policy.ScaleToZero` is true
- Add idle detection, wake triggers, and cost tracking

## Files to Create

| File | Purpose |
|------|---------|
| `internal/scaling/idle/detector.go` | Idle detection service |
| `internal/scaling/idle/metrics.go` | Prometheus metrics for idle state |
| `internal/scaling/wake/trigger.go` | Wake trigger handling |
| `internal/api/handlers/wake.go` | Wake API endpoints |
| `internal/api/models/wake.go` | Wake request/response types |
| `internal/api/services/wake.go` | Wake service logic |
| `deployments/docker/init-scripts/15-scale-to-zero-schema.sql` | Database migrations |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/scaling/types.go` | Add IdleConfig, WakeReason types |
| `internal/scaling/manager.go` | Integrate idle detector |
| `internal/scaling/evaluator.go` | Idle-aware evaluation |
| `internal/config/config.go` | Add scale-to-zero config options |
| `internal/api/server.go` | Register wake endpoints |

## Task Breakdown

### Phase 1: Core Idle Detection (~1,500 LOC)

**Task 1.1: Database Schema**
- Add `idle_threshold_seconds` to scaling_policies
- Add `keep_alive_window_seconds` to scaling_policies
- Create `scaling_idle_state` table for tracking
- Create `scaling_cost_savings` table for reporting

**Task 1.2: Idle Detection Package**
- `idle/detector.go` - IdleDetector service
  - Track last activity timestamp per policy
  - Calculate idle duration
  - Expose idle state via Prometheus metrics
- `idle/metrics.go` - Prometheus metrics
  - `philotes_scaling_idle_duration_seconds` gauge
  - `philotes_scaling_last_activity_timestamp` gauge
  - `philotes_scaling_is_scaled_to_zero` gauge

**Task 1.3: Type Extensions**
- Add `IdleConfig` struct to types.go
- Add idle threshold fields to Policy struct
- Add WakeReason enum

### Phase 2: Wake Triggers (~1,500 LOC)

**Task 2.1: Wake Package**
- `wake/trigger.go` - WakeTrigger service
  - Handle wake requests
  - Update scaling state
  - Trigger scale-up

**Task 2.2: Wake API**
- `POST /api/v1/scaling/policies/:id/wake` - Wake specific policy
- `POST /api/v1/scaling/wake` - Wake all scaled-to-zero policies
- Request/response models
- Service layer implementation

**Task 2.3: Handler Registration**
- Add wake endpoints to scaling handler
- Update server.go to register routes

### Phase 3: Evaluator Integration (~1,000 LOC)

**Task 3.1: Evaluator Updates**
- Check idle state during evaluation
- Support idle-based scale-down rules
- Integrate with idle detector

**Task 3.2: Manager Updates**
- Start/stop idle detector
- Handle wake events
- Update state tracking

### Phase 4: Cost Tracking (~1,000 LOC)

**Task 4.1: Cost Savings Tracking**
- Track idle hours per day
- Calculate estimated savings based on instance pricing
- Store in cost_savings table

**Task 4.2: Cost Savings API**
- `GET /api/v1/scaling/policies/:id/savings` - Get cost savings report
- `GET /api/v1/scaling/savings/summary` - Get overall savings summary

### Phase 5: Configuration & Testing (~1,000 LOC)

**Task 5.1: Configuration**
- Add scale-to-zero config to config.go
- Environment variable support
- Default values

**Task 5.2: Integration Tests**
- Test idle detection
- Test wake triggers
- Test scale-down flow
- Test scale-up flow

## Database Schema

```sql
-- Migration: 15-scale-to-zero-schema.sql

-- Add idle configuration to scaling_policies
ALTER TABLE scaling_policies
ADD COLUMN IF NOT EXISTS idle_threshold_seconds INTEGER DEFAULT 1800,
ADD COLUMN IF NOT EXISTS keep_alive_window_seconds INTEGER DEFAULT 300,
ADD COLUMN IF NOT EXISTS cold_start_timeout_seconds INTEGER DEFAULT 120;

-- Idle state tracking
CREATE TABLE IF NOT EXISTS scaling_idle_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    idle_since TIMESTAMP WITH TIME ZONE,
    scaled_to_zero_at TIMESTAMP WITH TIME ZONE,
    last_wake_at TIMESTAMP WITH TIME ZONE,
    wake_reason TEXT,
    is_scaled_to_zero BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(policy_id)
);

-- Cost savings tracking
CREATE TABLE IF NOT EXISTS scaling_cost_savings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    idle_seconds BIGINT DEFAULT 0,
    scaled_to_zero_seconds BIGINT DEFAULT 0,
    estimated_savings_cents INTEGER DEFAULT 0,
    hourly_cost_cents INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(policy_id, date)
);

CREATE INDEX IF NOT EXISTS idx_scaling_idle_state_policy ON scaling_idle_state(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_cost_savings_policy_date ON scaling_cost_savings(policy_id, date);
```

## API Design

### Wake Endpoints

**POST /api/v1/scaling/policies/:id/wake**
```json
// Request
{
  "reason": "manual",  // optional: "manual", "scheduled", "webhook", "api_request"
  "wait_for_ready": true  // optional: wait for worker to be ready
}

// Response
{
  "policy_id": "uuid",
  "previous_replicas": 0,
  "target_replicas": 1,
  "wake_reason": "manual",
  "message": "Wake initiated",
  "estimated_ready_seconds": 30
}
```

**POST /api/v1/scaling/wake**
```json
// Request
{
  "policy_ids": ["uuid1", "uuid2"],  // optional: wake specific policies
  "reason": "manual"
}

// Response
{
  "woken": 2,
  "already_running": 1,
  "policies": [
    {"policy_id": "uuid1", "status": "waking"},
    {"policy_id": "uuid2", "status": "waking"},
    {"policy_id": "uuid3", "status": "already_running"}
  ]
}
```

**GET /api/v1/scaling/policies/:id/savings**
```json
// Response
{
  "policy_id": "uuid",
  "period": "last_30_days",
  "total_idle_hours": 450.5,
  "total_scaled_to_zero_hours": 380.2,
  "estimated_savings_euros": 45.50,
  "daily_breakdown": [
    {"date": "2026-01-30", "idle_hours": 16.5, "savings_euros": 1.65},
    {"date": "2026-01-29", "idle_hours": 18.0, "savings_euros": 1.80}
  ]
}
```

## Configuration

```go
// In config.go
type ScaleToZeroConfig struct {
    // DefaultIdleThreshold is the default idle duration before scale-to-zero
    DefaultIdleThreshold time.Duration `envconfig:"SCALE_TO_ZERO_IDLE_THRESHOLD" default:"30m"`

    // DefaultKeepAliveWindow prevents flapping by requiring sustained idle
    DefaultKeepAliveWindow time.Duration `envconfig:"SCALE_TO_ZERO_KEEP_ALIVE" default:"5m"`

    // ColdStartTimeout is max time to wait for worker to become ready
    ColdStartTimeout time.Duration `envconfig:"SCALE_TO_ZERO_COLD_START_TIMEOUT" default:"2m"`

    // IdleCheckInterval is how often to check idle state
    IdleCheckInterval time.Duration `envconfig:"SCALE_TO_ZERO_CHECK_INTERVAL" default:"1m"`

    // EnableCostTracking enables cost savings tracking
    EnableCostTracking bool `envconfig:"SCALE_TO_ZERO_COST_TRACKING" default:"true"`
}
```

## Test Strategy

1. **Unit Tests**
   - Idle detector logic
   - Wake trigger handling
   - Cost calculations

2. **Integration Tests**
   - Full scale-down flow
   - Full scale-up flow
   - API endpoint tests

3. **Manual Testing**
   - Scale to zero after idle threshold
   - Wake via API call
   - Verify checkpoint preservation

## Dependencies

- Existing scaling infrastructure (complete)
- Node auto-scaling (SCALE-003) - complete
- Prometheus metrics endpoint

## Estimated Effort

- Phase 1: ~1,500 LOC
- Phase 2: ~1,500 LOC
- Phase 3: ~1,000 LOC
- Phase 4: ~1,000 LOC
- Phase 5: ~1,000 LOC
- **Total: ~6,000 LOC** (matches issue estimate)
