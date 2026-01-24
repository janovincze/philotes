# Implementation Plan: CDC-004 - End-to-End Pipeline Orchestration

## Summary

Implement pipeline orchestration with lifecycle management, backpressure handling, error handling with retries, dead-letter queue, health checks, and configuration validation.

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │         Pipeline Coordinator         │
                    │  ┌─────────────────────────────┐    │
                    │  │      State Machine          │    │
                    │  │ Starting→Running→Paused→... │    │
                    │  └─────────────────────────────┘    │
                    └─────────────────────────────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         ↓                          ↓                          ↓
   ┌──────────┐              ┌──────────┐              ┌──────────┐
   │  Source  │ ─backpressure→│  Buffer  │ ─backpressure→│  Writer  │
   └──────────┘              └──────────┘              └──────────┘
                                    │
                                    ↓ (on failure after retries)
                             ┌──────────┐
                             │   DLQ    │
                             └──────────┘
```

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cdc/pipeline/state.go` | Pipeline state machine |
| `internal/cdc/pipeline/backpressure.go` | Backpressure controller |
| `internal/cdc/pipeline/retry.go` | Retry with exponential backoff |
| `internal/cdc/deadletter/deadletter.go` | DLQ interface and types |
| `internal/cdc/deadletter/postgres.go` | PostgreSQL DLQ implementation |
| `internal/cdc/health/health.go` | Health check system |
| `deployments/docker/init-scripts/04-deadletter-schema.sql` | DLQ table |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add retry, DLQ, health configs |
| `internal/cdc/pipeline/pipeline.go` | Integrate state machine, backpressure, retry |
| `internal/cdc/buffer/batch.go` | Add retry logic and DLQ |
| `cmd/philotes-worker/main.go` | Add health endpoints |

## Task Breakdown

### Phase 1: State Machine
1. Create pipeline state enum (Starting, Running, Paused, Stopping, Stopped, Failed)
2. Define valid state transitions
3. Integrate state machine into Pipeline

### Phase 2: Retry Logic
4. Create retry policy configuration
5. Implement exponential backoff
6. Add retry logic to batch processor

### Phase 3: Dead-Letter Queue
7. Create DLQ schema
8. Implement DLQ manager interface
9. Integrate DLQ with batch processor

### Phase 4: Backpressure
10. Add buffer size monitoring
11. Implement backpressure signals
12. Integrate backpressure with source reader

### Phase 5: Health Checks
13. Create health check interface
14. Implement component health checks
15. Add HTTP health endpoints to worker

### Phase 6: Configuration & Validation
16. Add new configuration options
17. Implement config validation
18. Update worker initialization

## Key Interfaces

```go
// State represents the pipeline state.
type State int

const (
    StateStarting State = iota
    StateRunning
    StatePaused
    StateStopping
    StateStopped
    StateFailed
)

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
}

// DeadLetterManager handles failed events.
type DeadLetterManager interface {
    Write(ctx context.Context, event FailedEvent) error
    Read(ctx context.Context, limit int) ([]FailedEvent, error)
    Retry(ctx context.Context, eventID int64) error
    Delete(ctx context.Context, eventID int64) error
}

// HealthChecker provides health status.
type HealthChecker interface {
    Check(ctx context.Context) HealthStatus
    Name() string
}
```

## Configuration Additions

```go
// RetryConfig holds retry policy configuration.
type RetryConfig struct {
    MaxAttempts     int           `env:"PHILOTES_RETRY_MAX_ATTEMPTS"`
    InitialInterval time.Duration `env:"PHILOTES_RETRY_INITIAL_INTERVAL"`
    MaxInterval     time.Duration `env:"PHILOTES_RETRY_MAX_INTERVAL"`
    Multiplier      float64       `env:"PHILOTES_RETRY_MULTIPLIER"`
}

// DeadLetterConfig holds DLQ configuration.
type DeadLetterConfig struct {
    Enabled   bool          `env:"PHILOTES_DLQ_ENABLED"`
    Retention time.Duration `env:"PHILOTES_DLQ_RETENTION"`
}

// HealthConfig holds health check configuration.
type HealthConfig struct {
    Enabled    bool   `env:"PHILOTES_HEALTH_ENABLED"`
    ListenAddr string `env:"PHILOTES_HEALTH_LISTEN_ADDR"`
}
```

## DLQ Schema

```sql
CREATE TABLE philotes.dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    original_event_id BIGINT,
    source_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    event_data JSONB NOT NULL,
    error_message TEXT NOT NULL,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_retry_at TIMESTAMPTZ
);
```

## Verification

1. `make build` - Compiles successfully
2. `make test` - All tests pass
3. Manual verification:
   - Pipeline state transitions work correctly
   - Retries happen on transient failures
   - Failed events go to DLQ after max retries
   - Health endpoint returns correct status
   - Backpressure slows down source when buffer is full
