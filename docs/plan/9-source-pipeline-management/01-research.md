# Research Findings - Issue #9: API-002 Source and Pipeline Management

## 1. Existing API Structure (API-001 Foundation)

### Server Setup
- **File:** `internal/api/server.go`
- Gin-based HTTP server with middleware chain
- Routes registered in `registerRoutes()` method (lines 103-127)

### Handler Pattern
All handlers follow this pattern:
```go
type HealthHandler struct {
    healthManager *health.Manager
}

func NewHealthHandler(healthManager *health.Manager) *HealthHandler {
    return &HealthHandler{healthManager: healthManager}
}

func (h *HealthHandler) GetHealth(c *gin.Context) {
    // Implementation
}
```

### Middleware Stack
- Request ID (`middleware/requestid.go`)
- Recovery (`middleware/recovery.go`)
- Logging (`middleware/logging.go`)
- CORS (`middleware/cors.go`)
- Rate Limiting (`middleware/ratelimit.go`)

### Error Handling
- RFC 7807 Problem Details pattern in `internal/api/models/error.go`
- Field-level validation support via `FieldError` struct
- `RespondWithError(c, err)` helper function

## 2. Database & Repository Patterns

### Connection Pooling
- Using `database/sql` with pgx driver (`jackc/pgx/v5`)
- Connection via DSN from config

### Repository Pattern Examples
**Checkpoint Manager** (`internal/cdc/checkpoint/postgres.go`):
```go
type PostgresManager struct {
    db     *sql.DB
    logger *slog.Logger
}

func NewPostgresManager(db *sql.DB, logger *slog.Logger) (*PostgresManager, error) {
    // Create manager with schema setup
}
```

**Buffer Manager** (`internal/cdc/buffer/postgres.go`):
- Event persistence with transactions
- Batch operations support

### Interface-Based Design
Each component defines a Manager interface:
```go
type Manager interface {
    Save(ctx context.Context, checkpoint cdc.Checkpoint) error
    Load(ctx context.Context, sourceID string) (*cdc.Checkpoint, error)
    Delete(ctx context.Context, sourceID string) error
    Close() error
}
```

## 3. CDC Domain Concepts

### Core Types (`internal/cdc/types.go`)
- `Event`: operation, LSN, timestamp, before/after data
- `Checkpoint`: source tracking with LSN
- `TableSchema`: column definitions with types and constraints
- `Column`: type, nullable, primary key flags

### Source Interface (`internal/cdc/source/source.go`)
```go
type Source interface {
    Start(ctx context.Context) (<-chan cdc.Event, <-chan error)
    Stop(ctx context.Context) error
    LastLSN() string
    Name() string
}
```

### Pipeline (`internal/cdc/pipeline/pipeline.go`)
- Orchestrates: Source → Buffer → Checkpoint
- State machine for lifecycle management
- Stats tracking with RWMutex

## 4. Configuration Pattern

### Centralized Config (`internal/config/config.go`)
```go
type Config struct {
    Version     string
    Environment string
    API         APIConfig
    Database    DatabaseConfig
    CDC         CDCConfig
    Iceberg     IcebergConfig
    Storage     StorageConfig
    Metrics     MetricsConfig
}
```

### Database Config
```go
type DatabaseConfig struct {
    Host         string
    Port         int
    Name         string
    User         string
    Password     string
    SSLMode      string
    MaxOpenConns int
    MaxIdleConns int
}

func (d DatabaseConfig) DSN() string {
    return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode)
}
```

## 5. Existing Database Schema

### Init Scripts Location
`deployments/docker/init-scripts/`

### Existing Tables
- `cdc_checkpoints` - Checkpoint persistence
- `cdc_schema_history` - Schema tracking
- `cdc_events` - Event buffer
- `dead_letter_events` - Failed events

### Need to Create
- `sources` - Source database configurations
- `pipelines` - Pipeline definitions
- `destinations` - Destination configurations
- `table_mappings` - Table-level sync configs

## 6. Validation Patterns

### Config Validation Example (`internal/cdc/source/postgres/config.go`)
```go
func (c *Config) Validate() error {
    if c.Host == "" {
        return fmt.Errorf("host is required")
    }
    if c.Port <= 0 {
        return fmt.Errorf("port must be positive")
    }
    // ...
}
```

### API Validation
- Use `models.NewValidationError()` for field-level errors
- Return RFC 7807 Problem Details format

## 7. Key Files Reference

| Purpose | File Path |
|---------|-----------|
| API Server | `internal/api/server.go` |
| Error Model | `internal/api/models/error.go` |
| Response Model | `internal/api/models/response.go` |
| Handler Pattern | `internal/api/handlers/health.go` |
| DB Pattern | `internal/cdc/checkpoint/postgres.go` |
| Validation | `internal/cdc/source/postgres/config.go` |
| Config | `internal/config/config.go` |
| CDC Types | `internal/cdc/types.go` |

## 8. Recommended Architecture

```
internal/api/
├── handlers/
│   ├── sources.go      # Source CRUD handlers
│   ├── pipelines.go    # Pipeline CRUD handlers
│   └── destinations.go # Destination CRUD handlers
├── services/
│   ├── source.go       # Source business logic
│   ├── pipeline.go     # Pipeline business logic
│   └── destination.go  # Destination business logic
├── repositories/
│   ├── source.go       # Source data access
│   ├── pipeline.go     # Pipeline data access
│   └── destination.go  # Destination data access
└── models/
    ├── source.go       # Source request/response models
    ├── pipeline.go     # Pipeline request/response models
    └── destination.go  # Destination request/response models
```

## 9. No Blockers Identified

The existing architecture provides solid foundations for implementing Issue #9 without any technical blockers.
