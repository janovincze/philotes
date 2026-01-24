# Implementation Plan - CDC-002: Buffer Database Implementation

## Overview

Implement a PostgreSQL-based event buffer that stores CDC events for reliability, replay capability, and batch processing before forwarding to Iceberg.

## Architecture

```
PostgreSQL Source → pgstream Reader → Pipeline
                                         ↓
                                    Buffer Manager
                                         ↓
                                   PostgreSQL Buffer DB
                                         ↓
                                    Batch Processor
                                         ↓
                                    (Future: Iceberg Writer)
```

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cdc/buffer/buffer.go` | Manager interface and types |
| `internal/cdc/buffer/postgres.go` | PostgreSQL implementation |
| `internal/cdc/buffer/batch.go` | Batch processor for reading events |
| `internal/cdc/buffer/buffer_test.go` | Unit tests |
| `deployments/docker/init-scripts/03-buffer-schema.sql` | Buffer tables |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add BufferConfig with retention settings |
| `internal/cdc/pipeline/pipeline.go` | Integrate buffer writes |

## Task Breakdown

### Phase 1: Schema & Configuration

**Task 1: Create buffer database schema**
- Create `cdc_events` table with all required columns
- Add indexes for efficient querying
- Include `processed_at` for retention management

**Task 2: Extend configuration**
- Add `BufferConfig` struct with retention settings
- Add environment variable loading

### Phase 2: Buffer Manager

**Task 3: Define buffer interface**
- `Manager` interface with Write, ReadBatch, MarkProcessed, Cleanup methods
- `BufferedEvent` struct wrapping Event with database ID

**Task 4: Implement PostgreSQL buffer**
- Connection management
- Write events with JSON serialization
- Read unprocessed events in batches
- Mark events as processed
- Cleanup old processed events

### Phase 3: Batch Processor

**Task 5: Implement batch processor**
- Timer-based + size-based batch collection
- Read from buffer, forward to handler
- Mark processed on success

### Phase 4: Pipeline Integration

**Task 6: Integrate buffer into pipeline**
- Add buffer manager to Pipeline struct
- Write events to buffer in processEvent()
- Add batch processor goroutine

### Phase 5: Testing

**Task 7: Add tests**
- Unit tests for buffer operations
- Integration tests with PostgreSQL

## Database Schema

```sql
-- Buffer schema
CREATE SCHEMA IF NOT EXISTS philotes;

CREATE TABLE IF NOT EXISTS philotes.cdc_events (
    id BIGSERIAL PRIMARY KEY,
    source_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    lsn TEXT NOT NULL,
    transaction_id BIGINT,
    key_columns JSONB,
    before_data JSONB,
    after_data JSONB,
    event_time TIMESTAMPTZ NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

-- Index for fetching unprocessed events
CREATE INDEX IF NOT EXISTS idx_cdc_events_unprocessed
    ON philotes.cdc_events (source_id, created_at)
    WHERE processed_at IS NULL;

-- Index for cleanup of processed events
CREATE INDEX IF NOT EXISTS idx_cdc_events_processed
    ON philotes.cdc_events (processed_at)
    WHERE processed_at IS NOT NULL;
```

## Configuration

```bash
# Buffer settings
PHILOTES_BUFFER_ENABLED=true
PHILOTES_BUFFER_BATCH_SIZE=1000
PHILOTES_BUFFER_FLUSH_INTERVAL=5s
PHILOTES_BUFFER_RETENTION=168h  # 7 days
PHILOTES_BUFFER_CLEANUP_INTERVAL=1h
```

## Verification

1. `make build` - Compiles successfully
2. `make test` - All tests pass
3. Manual test:
   - Insert event to buffer
   - Read batch
   - Mark processed
   - Verify cleanup removes old events

## Out of Scope (Deferred to CDC-003)
- Iceberg writer integration
- Actual event forwarding to downstream
