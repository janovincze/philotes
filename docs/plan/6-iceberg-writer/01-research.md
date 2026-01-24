# Research Findings: CDC-003 - Apache Iceberg Writer

## Overview

Comprehensive research of the Philotes codebase to inform the Apache Iceberg Writer implementation.

## 1. Configuration Already Complete

**IcebergConfig** in `internal/config/config.go`:
```go
type IcebergConfig struct {
    CatalogURL string  // Lakekeeper REST catalog URL (default: http://localhost:8181)
    Warehouse  string  // Warehouse name (default: philotes)
}
```

**StorageConfig** in `internal/config/config.go`:
```go
type StorageConfig struct {
    Endpoint  string  // MinIO endpoint (default: localhost:9000)
    AccessKey string  // Access key (default: minioadmin)
    SecretKey string  // Secret key (default: minioadmin)
    Bucket    string  // Bucket name (default: philotes)
    UseSSL    bool    // SSL enabled (default: false)
}
```

All environment variables are already mapped and loaded.

## 2. Integration Architecture

```
PostgreSQL (source) → pgstream CDC → Pipeline.processEvent()
                                            ↓
                                    buffer.Manager.Write()
                                            ↓
                                    [Buffer in PostgreSQL]
                                            ↓
                                    batch.BatchProcessor.Start()
                                            ↓
                                    handler(ctx, events)  ← ICEBERG WRITER HERE
                                            ↓
                                    buffer.Manager.MarkProcessed()
```

The batch processor already accepts a `BatchHandler` function:
```go
type BatchHandler func(ctx context.Context, events []BufferedEvent) error
```

## 3. Docker Environment

From `deployments/docker/docker-compose.yml`:
- **Lakekeeper**: Port 8181 with health checks
- **MinIO**: Port 9000 (API), Port 9001 (console)
- **Buckets**: "philotes" and "warehouse" pre-created
- Authentication disabled for MVP

## 4. Event Structure

From `internal/cdc/types.go`:
```go
type Event struct {
    ID            string
    LSN           string
    TransactionID int64
    Timestamp     time.Time
    Schema        string
    Table         string
    Operation     string
    KeyColumns    []string
    Before        map[string]any
    After         map[string]any
    Metadata      map[string]any
}
```

## 5. Dependencies

**Already in go.mod:**
- github.com/google/uuid
- github.com/jackc/pgx/v5
- slog, context, database/sql, encoding/json (built-in)

**Need to add:**
- `github.com/xitongsys/parquet-go` - Pure Go Parquet writing
- `github.com/minio/minio-go/v7` - MinIO/S3 client

## 6. Iceberg Implementation Strategy

**Recommended approach for MVP:**
1. Use **Lakekeeper REST API** for all metadata operations
2. Write **Parquet files directly** to MinIO/S3
3. No need to implement Iceberg spec internals

**Rationale:**
- Go lacks native Iceberg library (unlike Java/Python)
- Lakekeeper already running and configured
- Clear separation of concerns
- Easier testing and debugging

## 7. Code Patterns to Follow

From `internal/cdc/buffer/`:
- Interface definition with simple methods
- PostgreSQL implementation with connection pooling
- JSON serialization for complex data
- Structured logging with slog
- Proper error wrapping with fmt.Errorf

## 8. Files to Create

```
internal/iceberg/
├── catalog/
│   ├── catalog.go           # Interface for REST catalog operations
│   └── rest.go              # Lakekeeper REST client implementation
├── schema/
│   ├── schema.go            # Convert PostgreSQL types to Iceberg schema
│   └── evolution.go         # Handle schema changes
├── writer/
│   ├── writer.go            # Main Writer interface
│   ├── batch_handler.go     # Handler function for batch processor
│   ├── parquet.go           # Parquet file writing logic
│   └── s3.go                # MinIO/S3 upload operations
└── types.go                 # Iceberg-specific types
```

## 9. Integration with Worker

Current `cmd/philotes-worker/main.go` creates buffer manager but no batch processor.

Need to add batch processor startup with Iceberg handler.

## 10. Lakekeeper REST API

Key endpoints:
- `POST /catalog/v1/{prefix}/namespaces` - Create namespace
- `POST /catalog/v1/{prefix}/namespaces/{namespace}/tables` - Create table
- `POST /catalog/v1/{prefix}/namespaces/{namespace}/tables/{table}` - Commit data
- `GET /catalog/v1/{prefix}/namespaces/{namespace}/tables/{table}` - Get table metadata

## 11. Parquet File Structure

For CDC events, each batch becomes a Parquet file with:
- All columns from the after_data (for INSERT/UPDATE)
- System columns: _cdc_operation, _cdc_timestamp, _cdc_lsn
- Partitioned by date or configurable partition column
