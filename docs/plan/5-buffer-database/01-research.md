# Research Findings - CDC-002: Buffer Database Implementation

## 1. Existing CDC Implementation (CDC-001)

### Key Files Reviewed
- `internal/cdc/types.go` - Core event structures
- `internal/cdc/pipeline/pipeline.go` - Pipeline orchestration
- `internal/cdc/source/postgres/reader.go` - PostgreSQL CDC source
- `internal/cdc/checkpoint/postgres.go` - Checkpoint persistence patterns

### Event Structure
The `Event` struct in types.go has all fields needed for buffer storage:
- ID, LSN, TransactionID, Timestamp
- Schema, Table, Operation type
- Before/After data as map[string]any
- KeyColumns for identifying records
- Metadata map for extensibility

### Pipeline Architecture
Current pipeline:
- Receives events from PostgreSQL source via pgstream
- Calls `processEvent()` method which currently only logs
- Comment at line 161-164 indicates where buffer integration should happen
- Maintains stats and checkpoint positions

### Data Flow
```
PostgreSQL (port 5433)
  → pgstream CDC Reader
  → events channel
  → Pipeline.processEvent() [BUFFER GOES HERE]
  → Checkpoint saved periodically
```

## 2. Database Patterns & Configuration

### Existing Database Schema
Buffer table schema exists in `01-init.sql`:
```sql
CREATE TABLE IF NOT EXISTS cdc.events (
    id BIGSERIAL PRIMARY KEY,
    source_id UUID NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    lsn VARCHAR(50) NOT NULL,
    transaction_id BIGINT,
    event_data BYTEA NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Configuration
`CDCConfig` in config.go already has:
- BufferSize: 10000 (default)
- BatchSize: 1000 (default)
- FlushInterval: 5 seconds (default)

### Checkpoint Pattern (postgres.go)
Demonstrates:
- Connection pooling with MaxOpenConns, MaxIdleConns
- UPSERT pattern for idempotent operations
- JSON serialization for metadata
- Proper Close() for cleanup

## 3. Integration Points

### Where Buffer Integrates
Pipeline's `processEvent()` method (line 148-167) is the integration point.
Currently it only updates stats and logs.

### Batch Processing Strategy
From existing config:
- `BatchSize`: 1000 (default)
- `FlushInterval`: 5 seconds (default)

## 4. Design Decisions

### Serialization Format
- Schema uses `event_data BYTEA` (binary)
- **Recommendation: JSON** for simplicity and debuggability
- Can upgrade to MessagePack later if needed

### Buffer Manager Interface
Follow checkpoint.Manager pattern:
- Write(ctx, events []Event) error
- ReadBatch(ctx, sourceID, limit) ([]BufferedEvent, error)
- MarkProcessed(ctx, eventIDs []int64) error
- GetStats(ctx) (BufferStats, error)
- Close() error

### Retention Policy
- Add `processed_at` nullable column
- Background cleanup: DELETE WHERE processed_at < NOW() - retention

## 5. File Structure

```
internal/cdc/buffer/
├── buffer.go           # Manager interface
├── postgres.go         # PostgreSQL implementation
├── config.go           # Configuration
├── types.go            # BufferedEvent struct
└── buffer_test.go      # Tests
```

## 6. Dependencies

**Already available:**
- `database/sql` (standard library)
- `jackc/pgx/v5` (in go.mod)
- `encoding/json` (standard library)

## 7. Files to Modify

| File | Changes |
|------|---------|
| `internal/cdc/buffer/*.go` | New - buffer implementation |
| `internal/cdc/pipeline/pipeline.go` | Add buffer integration |
| `internal/config/config.go` | Add buffer retention config |
| `deployments/docker/init-scripts/` | Update schema with processed_at |
