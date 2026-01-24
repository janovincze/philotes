# Research Findings - CDC-001: pgstream Integration and CDC Worker Foundation

## 1. Existing Codebase Structure

### Worker Entry Point (`cmd/philotes-worker/main.go`)
- Minimal implementation with placeholder "Worker initialization not yet implemented"
- Has proper signal handling (SIGINT, SIGTERM)
- Loads config and calls `run(ctx, cfg)` function
- Pattern: Config loading → graceful shutdown handling

### Config System (`internal/config/config.go`)
- Well-structured environment variable configuration
- Already includes `CDCConfig` struct with:
  - `BufferSize: int` (default 10,000)
  - `BatchSize: int` (default 1,000)
  - `FlushInterval: time.Duration` (default 5s)
- Pattern: `getEnv()`, `getIntEnv()`, `getBoolEnv()`, `getDurationEnv()` helpers
- DSN builder for database connections

### CDC Directory Structure (`internal/cdc/`)
- Four subdirectories already created:
  - `source/postgres/` - Empty, ready for pgstream integration
  - `buffer/` - Empty, ready for event buffering
  - `checkpoint/` - Empty, ready for checkpointing logic
  - `pipeline/` - Empty, ready for orchestration
- No existing implementation code

## 2. PostgreSQL Setup (docker-compose.yml)

**Buffer DB (port 5432):**
- User: `philotes`, Password: `philotes`, DB: `philotes`
- Configured for logical replication:
  - `wal_level=logical`
  - `max_replication_slots=10`
  - `max_wal_senders=10`

**Source DB (port 5433):**
- User: `source`, Password: `source`, DB: `source`
- Same replication configuration for testing

## 3. pgstream Library Research

**Key Insights:**
- Installation: `go get github.com/xataio/pgstream`
- Core API:
  - Configuration accepts: PostgresURL, ReplicationSlot, Publication
  - Provides listener for CDC events
  - Supports snapshot and replication modes
- Schema Tracking: Handles DDL changes alongside data modifications
- Replication Slots: Managed for crash recovery

## 4. Suggested Component Structure

From `.claude/commands/cdc.md`:

```
PostgreSQL → pgstream (CDC Reader) → Event Buffer (PostgreSQL) → Iceberg Writer
```

**Component Files:**
- `source/postgres/reader.go` - pgstream wrapper
- `source/postgres/config.go` - Connection configuration
- `source/postgres/schema.go` - Schema tracking
- `buffer/postgres_buffer.go` - Event persistence
- `checkpoint/postgres_checkpoint.go` - Checkpointing with LSN tracking
- `pipeline/pipeline.go` - Orchestration

**Event Type Pattern:**
```go
type CDCEvent struct {
    ID, Table, Schema, Operation, LSN string
    TransactionID int64
    Timestamp time.Time
    Before, After map[string]interface{}
    KeyColumns []string
}
```

## 5. Code Patterns to Follow

**Error Handling:**
- Wrap with context: `fmt.Errorf("context: %w", err)`
- Custom error types for domain errors

**Configuration Pattern:**
- Environment variables via `PHILOTES_*` prefix
- Sensible defaults
- Helper functions for type conversion

**Service Pattern:**
- `New()` constructor
- `Start(ctx)` for async operations
- `Stop(ctx)` for graceful shutdown

## 6. Key Blockers and Questions

**Blockers:**
1. Need to add `github.com/xataio/pgstream` to `go.mod`
2. Buffer DB schema tables needed:
   - `cdc_events` - Event storage
   - `cdc_checkpoints` - Checkpoint tracking

**Design Decisions:**
1. Error recovery on connection drop
2. Schema evolution handling
3. Dead letter queue location
4. Metrics selection
5. LSN rollback handling

## 7. Configuration Extensions Needed

The existing `CDCConfig` needs expansion:
- PostgreSQL source connection details
- Replication slot name
- Publication name
- Schema tracking options
- Checkpoint interval
- Reconnection policies

## 8. Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `internal/cdc/source/postgres/reader.go` | Create | pgstream integration |
| `internal/cdc/source/postgres/config.go` | Create | PostgreSQL connection config |
| `internal/cdc/source/postgres/schema.go` | Create | Schema tracking |
| `internal/cdc/source/types.go` | Create | CDCEvent type definitions |
| `internal/cdc/buffer/postgres_buffer.go` | Create | Event buffer implementation |
| `internal/cdc/checkpoint/postgres_checkpoint.go` | Create | Checkpointing |
| `cmd/philotes-worker/main.go` | Modify | Implement worker initialization |
| `internal/config/config.go` | Modify | Extend CDCConfig |
| `go.mod` | Modify | Add pgstream dependency |

## 9. Recommended Implementation Phases

1. **Phase 1: Foundation** - Config extensions, pgstream dependency, reader wrapper
2. **Phase 2: Event Processing** - CDCEvent type, buffer interface, schema detection
3. **Phase 3: Checkpointing** - Checkpoint interface, tables, LSN tracking
4. **Phase 4: Pipeline Orchestration** - Coordinator, reader→buffer flow, shutdown
5. **Phase 5: Testing & Observability** - Tests, metrics, logging
