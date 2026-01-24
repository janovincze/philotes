# Implementation Plan - CDC-001: pgstream Integration and CDC Worker Foundation

## Overview

Implement a CDC (Change Data Capture) worker service that uses the pgstream library to capture PostgreSQL WAL events. The worker will parse, normalize, and buffer events while maintaining checkpoints for exactly-once semantics.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Source DB      │────▶│  CDC Worker      │────▶│  Buffer DB      │
│  (PostgreSQL)   │     │  (pgstream)      │     │  (PostgreSQL)   │
└─────────────────┘     └──────────────────┘     └─────────────────┘
        │                       │                        │
        │                       ▼                        │
        │               ┌──────────────┐                 │
        │               │  Checkpoint  │◀────────────────┘
        │               │  Manager     │
        │               └──────────────┘
        │
        ▼
   WAL Events (INSERT, UPDATE, DELETE, DDL)
```

## Files to Create/Modify

### New Files

| File | Purpose |
|------|---------|
| `internal/cdc/types.go` | Core CDC types (Event, Operation, etc.) |
| `internal/cdc/source/source.go` | Source interface definition |
| `internal/cdc/source/postgres/reader.go` | pgstream-based PostgreSQL reader |
| `internal/cdc/source/postgres/config.go` | Reader configuration |
| `internal/cdc/checkpoint/checkpoint.go` | Checkpoint interface |
| `internal/cdc/checkpoint/postgres.go` | PostgreSQL checkpoint storage |
| `internal/cdc/pipeline/pipeline.go` | Pipeline orchestration |
| `internal/cdc/pipeline/worker.go` | Worker lifecycle management |
| `deployments/docker/init-scripts/02-cdc-schema.sql` | CDC tables schema |

### Modified Files

| File | Changes |
|------|---------|
| `go.mod` | Add pgstream, pgx dependencies |
| `internal/config/config.go` | Extend CDCConfig with source settings |
| `cmd/philotes-worker/main.go` | Implement worker initialization |

## Task Breakdown

### Phase 1: Dependencies & Configuration (Tasks 1-2)

**Task 1: Add dependencies to go.mod**
- Add `github.com/xataio/pgstream`
- Add `github.com/jackc/pgx/v5`
- Run `go mod tidy`

**Task 2: Extend configuration**
- Add `SourceConfig` struct for source DB connection
- Add `ReplicationConfig` for slot name, publication
- Add `CheckpointConfig` for interval, storage
- Update environment variable loading

### Phase 2: Core Types (Task 3)

**Task 3: Define CDC types**
- `Operation` enum (Insert, Update, Delete, Truncate)
- `Event` struct with all fields
- `Schema` struct for table schema tracking
- `Column` struct for column metadata

### Phase 3: Source Reader (Tasks 4-5)

**Task 4: Implement Source interface**
- Define `Source` interface with `Start`, `Stop`, `Events` channel
- Define `SourceConfig` for configuration

**Task 5: Implement PostgreSQL reader**
- Wrap pgstream library
- Handle connection and reconnection
- Parse WAL events to internal Event type
- Manage replication slot lifecycle

### Phase 4: Checkpoint Management (Tasks 6-7)

**Task 6: Create checkpoint schema**
- SQL migration for `cdc_checkpoints` table
- SQL migration for `cdc_schema_history` table

**Task 7: Implement checkpoint manager**
- Define `Checkpoint` interface
- Implement PostgreSQL checkpoint storage
- LSN tracking and persistence
- Recovery from last checkpoint

### Phase 5: Pipeline & Worker (Tasks 8-9)

**Task 8: Implement pipeline coordinator**
- Wire source → processing → checkpoint flow
- Error handling and recovery
- Graceful shutdown

**Task 9: Update worker main**
- Initialize pipeline with config
- Start/stop lifecycle
- Structured logging
- Signal handling

### Phase 6: Testing (Task 10)

**Task 10: Add tests**
- Unit tests for types and config
- Integration tests with test containers
- Mock source for pipeline tests

## Database Schema

### cdc_checkpoints table
```sql
CREATE TABLE cdc_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id TEXT NOT NULL UNIQUE,
    lsn TEXT NOT NULL,
    transaction_id BIGINT,
    committed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);
```

### cdc_schema_history table
```sql
CREATE TABLE cdc_schema_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    version INT NOT NULL,
    columns JSONB NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lsn TEXT NOT NULL,
    UNIQUE(schema_name, table_name, version)
);
```

## Configuration Environment Variables

```bash
# Source PostgreSQL
PHILOTES_CDC_SOURCE_HOST=localhost
PHILOTES_CDC_SOURCE_PORT=5433
PHILOTES_CDC_SOURCE_DATABASE=source
PHILOTES_CDC_SOURCE_USER=source
PHILOTES_CDC_SOURCE_PASSWORD=source

# Replication
PHILOTES_CDC_REPLICATION_SLOT=philotes_cdc
PHILOTES_CDC_PUBLICATION=philotes_pub

# Checkpoint
PHILOTES_CDC_CHECKPOINT_INTERVAL=10s
```

## Test Strategy

1. **Unit Tests**: Config parsing, event type conversion
2. **Integration Tests**:
   - Reader against real PostgreSQL with logical replication
   - Checkpoint persistence and recovery
3. **Pipeline Tests**: End-to-end with mock components

## Success Criteria

- [ ] Worker starts and connects to source PostgreSQL
- [ ] Replication slot is created/managed properly
- [ ] WAL events are parsed and normalized
- [ ] Checkpoints are persisted at configured intervals
- [ ] Graceful shutdown commits final checkpoint
- [ ] Reconnection works after connection loss
- [ ] All tests pass with `make test`
- [ ] Linting passes with `make lint`

## Estimated Scope

- ~15 new files
- ~2,500-3,000 lines of code
- Focus on core functionality, deferring buffer implementation to CDC-002
