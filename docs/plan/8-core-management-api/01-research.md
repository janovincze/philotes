# Research Findings - Issue #8: API-001 Core Management API Framework

## 1. Existing Project Structure & Patterns

### Module Structure
- Module: `github.com/janovincze/philotes` (Go 1.25.5)
- Well-organized internal package structure:
  - `/internal/config` - Configuration management
  - `/internal/cdc` - CDC pipeline implementation
  - `/internal/iceberg` - Iceberg/Lakekeeper integration
  - `/cmd` - Service entry points (worker, api, cli)
  - `/api/openapi` - OpenAPI specifications directory (exists but empty)

### Configuration Pattern (`internal/config/config.go`)
- Centralized `Config` struct with subsections (API, Database, CDC, Iceberg, Storage, Metrics)
- Environment variable-based loading using helper functions
- No Viper dependency - using standard library approach
- All config types are exported structs with struct tags

### Logging Pattern (`cmd/philotes-worker/main.go:28-32`)
- Uses standard library `log/slog` (structured logging)
- JSON output handler with configurable log levels
- Global logger set via `slog.SetDefault()`
- Component logging with context via `.With("component", "...")`

### Error Handling Pattern (`internal/cdc/source/postgres/errors.go`)
- Sentinel errors using `errors.New()` for predefined errors
- Prefixed error messages (e.g., "postgres: connection failed")
- Error wrapping with `fmt.Errorf(...%w)` for context preservation

### Graceful Shutdown Pattern (`cmd/philotes-api/main.go`)
- Context-based cancellation with signal handling
- Timeout for shutdown operations (30s)
- Goroutine-based concurrent operations with channels for errors

## 2. CDC Pipeline Integration

### Pipeline Types (`internal/cdc/pipeline/pipeline.go`)

**Pipeline Struct (lines 18-34):**
- Orchestrates CDC flow: source → checkpoint → buffer
- Includes StateMachine, BackpressureController, Retryer
- Thread-safe stats tracking with RWMutex

**Pipeline.Config (lines 36-63):**
- CheckpointInterval, CheckpointEnabled, BufferEnabled
- RetryPolicy, BackpressureConfig
- `DefaultConfig()` provides sensible defaults

**Pipeline.Stats (lines 65-75):**
- EventsProcessed, EventsBuffered, LastEventTime
- LastCheckpointLSN, LastCheckpointAt
- Errors, RetryCount, State

**Key Public Methods:**
- `Run(ctx context.Context) error` - Main pipeline loop
- `Stats() Stats` - Get current metrics
- `Pause()`, `Resume()` - Flow control
- `HealthChecker()` - Returns health.HealthChecker
- `State()` - Current pipeline state

### Pipeline State Machine (`internal/cdc/pipeline/state.go`)
- States: Starting → Running → Paused/Stopping → Stopped/Failed
- Validated state transitions
- State change listeners pattern

### CDC Data Types (`internal/cdc/types.go`)
- `Event` - CDC event with Before/After data, LSN, Operation
- `TableSchema` - Column definitions with metadata
- `Checkpoint` - LSN, TransactionID, SourceID, CommittedAt
- `Column` - Type, Nullable, PrimaryKey flags

### Source Interface (`internal/cdc/source/source.go`)
```go
type Source interface {
    Start(ctx context.Context) (<-chan cdc.Event, <-chan error)
    Stop(ctx context.Context) error
    LastLSN() string
    Name() string
}
```

### Buffer Manager Interface (`internal/cdc/buffer/buffer.go`)
```go
type Manager interface {
    Write(ctx context.Context, events []cdc.Event) error
    ReadBatch(ctx context.Context, sourceID string, limit int) ([]BufferedEvent, error)
    MarkProcessed(ctx context.Context, eventIDs []int64) error
    Cleanup(ctx context.Context, retention time.Duration) (int64, error)
    Stats(ctx context.Context) (Stats, error)
    Close() error
}
```

### Checkpoint Manager Interface (`internal/cdc/checkpoint/checkpoint.go`)
```go
type Manager interface {
    Save(ctx context.Context, checkpoint cdc.Checkpoint) error
    Load(ctx context.Context, sourceID string) (*cdc.Checkpoint, error)
    Delete(ctx context.Context, sourceID string) error
    Close() error
}
```

## 3. Iceberg Integration Models

### Catalog Interface (`internal/iceberg/catalog/catalog.go`)
- Namespace management: CreateNamespace, NamespaceExists
- Table operations: CreateTable, TableExists, LoadTable
- Snapshot management: CommitSnapshot

### Iceberg Types (`internal/iceberg/types.go`)
- `Schema` - SchemaID, Fields array
- `PartitionSpec` - SpecID, Fields with Transform
- `TableMetadata` - Format version, UUID, Location, Schemas
- `TableIdentifier` - Namespace.Name qualified name

## 4. Health Check System

### Health Package (`internal/cdc/health/health.go`)

**Key Types:**
- `Status` enum: StatusHealthy, StatusUnhealthy, StatusDegraded, StatusUnknown
- `CheckResult` - Name, Status, Message, Duration, LastCheck, Error
- `HealthChecker` interface - Check() and Name() methods
- `Manager` - Registers checkers, runs checks, caches results
- `Server` - HTTP server with /health, /health/live, /health/ready endpoints

**Existing Health Checkers:**
- `DatabaseChecker` - Ping-based database connectivity
- `ComponentChecker` - Generic component with custom check function

## 5. Dependencies Status

**Current go.mod includes:**
- ✅ `github.com/jackc/pgx/v5` - PostgreSQL driver
- ✅ `github.com/xataio/pgstream` - CDC streaming
- ✅ `github.com/minio/minio-go/v7` - S3/MinIO client
- ✅ `github.com/xitongsys/parquet-go` - Parquet writing
- ❌ NO Gin framework yet (needs to be added)
- ❌ NO OpenAPI/Swagger generator (needs to be added)

## 6. Existing API Configuration

Already in config (`internal/config/config.go`):
```go
API APIConfig {
    ListenAddr   string        // ":8080" default
    BaseURL      string        // "http://localhost:8080"
    ReadTimeout  time.Duration // 15s default
    WriteTimeout time.Duration // 15s default
}
```

## 7. Key Files Reference

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Configuration patterns |
| `cmd/philotes-worker/main.go` | Service initialization pattern |
| `internal/cdc/pipeline/pipeline.go` | Pipeline interface & stats |
| `internal/cdc/types.go` | CDC data models |
| `internal/cdc/health/health.go` | Health checking system |
| `internal/cdc/buffer/buffer.go` | Buffer manager interface |
| `internal/iceberg/types.go` | Iceberg data models |
| `cmd/philotes-api/main.go` | Current API entry point |

## 8. Recommended Approach

1. **Add Gin to go.mod** - Primary HTTP framework
2. **Create `/internal/api` package** - handlers, middleware, models
3. **Wire existing infrastructure** - config, slog logger, health.Manager
4. **Implement OpenAPI spec first** - API-first design
5. **Follow worker's initialization pattern** - Consistent service structure
6. **Extend config as needed** - CORS origins, rate limit settings
