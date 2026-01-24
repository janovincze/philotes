# Research Findings: CDC-004 - End-to-End Pipeline Orchestration

## 1. Current Pipeline Implementation

**File: `internal/cdc/pipeline/pipeline.go` (253 lines)**

Current capabilities:
- Basic event processing loop with source, checkpoint, and buffer integration
- Processes events sequentially from source channel
- Saves checkpoints periodically (10s default)
- Writes events to buffer if enabled
- Stats tracking: EventsProcessed, EventsBuffered, LastEventTime, Errors

What's missing:
- No state machine (just `running` boolean)
- No backpressure handling
- No retry logic - fails immediately on error
- No error classification/categorization
- No dead-letter queue
- No health checks
- No configuration validation
- No hot reload capability

## 2. Worker Lifecycle Management

**File: `cmd/philotes-worker/main.go` (193 lines)**

Current implementation:
- Signal handling for SIGINT/SIGTERM
- Sequential component initialization
- Graceful shutdown with deferred Close() calls
- No component health monitoring
- No readiness/liveness probes

Issues:
- No coordinated startup sequence checking
- All components start independently
- No validation that all components are ready before pipeline starts

## 3. Batch Processing Error Handling

**File: `internal/cdc/buffer/batch.go` (207 lines)**

Current implementation:
- Processes batches on flush interval (5s default)
- Marks events as processed after handler succeeds
- Cleanup runs on separate goroutine

Gaps:
- Handler errors are logged but don't retry
- No exponential backoff
- Failed events are not moved to DLQ
- No metrics on failure rate

## 4. Configuration System

**File: `internal/config/config.go` (352 lines)**

Existing config:
- `CDC.Checkpoint.Interval`: 10s default
- `CDC.Buffer.Retention`: 168h default
- `CDC.Buffer.CleanupInterval`: 1h default
- `CDC.BatchSize`: 1000 default
- `CDC.FlushInterval`: 5s default

Missing config for:
- Retry policies (max attempts, backoff strategy)
- Dead-letter queue settings
- Health check intervals and timeouts
- Backpressure thresholds
- Hot reload settings

## 5. Key Gaps to Address

| Requirement | Current State | Gap |
|-------------|---------------|-----|
| Lifecycle Management | start/stop only | Need pause/resume, state machine |
| Backpressure | None | Need buffer queue size monitoring |
| Retry Logic | Immediate fail | Need exponential backoff config |
| Dead-Letter Queue | None | Need separate table + configuration |
| Health Checks | None | Need startup/liveness/readiness probes |
| State Machine | Boolean flag | Need proper state enum + transitions |
| Config Validation | Partial | Need comprehensive validation |
| Hot Reload | None | Need file watcher + signal handling |

## 6. Current Integration Architecture

```
PostgreSQL Source
       ↓
pgstream Reader ← START() with events/errors channels
       ↓
Pipeline.Run(ctx)
       ├→ restoreCheckpoint()
       ├→ For each event:
       │   ├→ processEvent()
       │   └→ buffer.Write(ctx, event)
       ├→ saveCheckpoint() on ticker
       └→ Stop on context cancellation
       ↓
BatchProcessor.Start(ctx)
       ├→ processLoop() on flush interval
       ├→ processBatch()
       │   ├→ ReadBatch()
       │   ├→ handler() [Iceberg writer]
       │   └→ MarkProcessed()
       └→ cleanupLoop()
```

## 7. Recommended File Structure

### New Files to Create:
```
internal/cdc/pipeline/
├── state.go           # State machine definition
├── backpressure.go    # Backpressure handling
└── retry.go           # Retry logic with exponential backoff

internal/cdc/deadletter/
├── deadletter.go      # DLQ interface and types
└── postgres.go        # PostgreSQL DLQ implementation

internal/cdc/health/
├── health.go          # Health check system
└── probes.go          # Liveness/readiness probes

deployments/docker/init-scripts/
└── 04-deadletter-schema.sql  # DLQ table schema
```

### Files to Modify:
- `internal/config/config.go` - Add retry, DLQ, health check configs
- `internal/cdc/pipeline/pipeline.go` - Integrate state machine, backpressure
- `internal/cdc/buffer/batch.go` - Add retry logic and DLQ integration
- `cmd/philotes-worker/main.go` - Add health endpoints, startup sequence

## 8. Existing Patterns to Follow

From checkpoint and buffer implementations:
- Use context for cancellation
- Implement Manager interface pattern
- Use slog for structured logging
- Synchronize with sync.RWMutex
- Use PostgreSQL for persistent storage
- Error wrapping with `fmt.Errorf("message: %w", err)`
