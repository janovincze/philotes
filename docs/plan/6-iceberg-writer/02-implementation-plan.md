# Implementation Plan: CDC-003 - Apache Iceberg Writer

## Summary

Implement an Apache Iceberg table writer that receives batched CDC events from the buffer and writes them to Iceberg tables via Lakekeeper REST catalog and MinIO/S3 storage.

## Architecture

```
Buffer Database (PostgreSQL)
         ↓
    Batch Processor
         ↓
    Iceberg Writer
         ↓
  ┌──────┴──────┐
  ↓             ↓
Lakekeeper   MinIO/S3
(metadata)   (data files)
```

## Implementation Strategy

**REST-first approach:**
1. Use Lakekeeper REST API for all Iceberg metadata operations
2. Write Parquet files directly using parquet-go
3. Upload to MinIO using minio-go client
4. Commit file metadata to Lakekeeper

This avoids implementing the full Iceberg spec in Go while still producing valid Iceberg tables.

## Files to Create

| File | Purpose |
|------|---------|
| `internal/iceberg/types.go` | Iceberg-specific types and constants |
| `internal/iceberg/catalog/catalog.go` | Catalog interface definition |
| `internal/iceberg/catalog/rest.go` | Lakekeeper REST client |
| `internal/iceberg/catalog/rest_test.go` | REST client tests |
| `internal/iceberg/schema/schema.go` | Schema conversion utilities |
| `internal/iceberg/schema/types.go` | Type mapping PostgreSQL → Iceberg |
| `internal/iceberg/writer/writer.go` | Writer interface and config |
| `internal/iceberg/writer/parquet.go` | Parquet file generation |
| `internal/iceberg/writer/s3.go` | MinIO/S3 operations |
| `internal/iceberg/writer/batch_handler.go` | BatchHandler implementation |
| `internal/iceberg/writer/writer_test.go` | Writer tests |

## Files to Modify

| File | Changes |
|------|---------|
| `go.mod` | Add parquet-go and minio-go dependencies |
| `cmd/philotes-worker/main.go` | Initialize Iceberg writer and batch processor |

## Task Breakdown

### Phase 1: Foundation

#### 1.1 Add Dependencies
```bash
go get github.com/xitongsys/parquet-go
go get github.com/minio/minio-go/v7
```

#### 1.2 Create Iceberg Types (`internal/iceberg/types.go`)
- Iceberg data types (boolean, int, long, float, double, string, etc.)
- Partition spec types
- Schema representation
- Snapshot metadata

### Phase 2: Catalog Client

#### 2.1 Catalog Interface (`internal/iceberg/catalog/catalog.go`)
```go
type Catalog interface {
    CreateNamespace(ctx, namespace string) error
    NamespaceExists(ctx, namespace string) (bool, error)
    CreateTable(ctx, namespace, table string, schema Schema, partitionSpec PartitionSpec) error
    TableExists(ctx, namespace, table string) (bool, error)
    LoadTable(ctx, namespace, table string) (*TableMetadata, error)
    CommitTable(ctx, namespace, table string, updates TableUpdates) error
}
```

#### 2.2 Lakekeeper REST Client (`internal/iceberg/catalog/rest.go`)
- HTTP client with retry logic
- JSON marshaling for Iceberg REST spec
- Namespace operations (create, list, exists)
- Table operations (create, load, commit)
- Error handling for REST responses

### Phase 3: Schema Handling

#### 3.1 Type Mapping (`internal/iceberg/schema/types.go`)
PostgreSQL to Iceberg type mapping:
- `integer/int4` → `int`
- `bigint/int8` → `long`
- `real/float4` → `float`
- `double precision/float8` → `double`
- `boolean` → `boolean`
- `text/varchar` → `string`
- `timestamp/timestamptz` → `timestamptz`
- `date` → `date`
- `uuid` → `uuid`
- `jsonb/json` → `string`
- `bytea` → `binary`

#### 3.2 Schema Conversion (`internal/iceberg/schema/schema.go`)
- Convert CDC event structure to Iceberg schema
- Add CDC system columns (_cdc_operation, _cdc_timestamp, _cdc_lsn)
- Schema evolution detection (new columns)

### Phase 4: Writer Implementation

#### 4.1 S3 Client (`internal/iceberg/writer/s3.go`)
```go
type S3Client interface {
    Upload(ctx, bucket, key string, data io.Reader) error
    GeneratePresignedURL(ctx, bucket, key string) (string, error)
    Delete(ctx, bucket, key string) error
}
```

#### 4.2 Parquet Writer (`internal/iceberg/writer/parquet.go`)
- Convert BufferedEvent slice to Parquet format
- Handle different column types
- Generate unique file names (UUID-based)
- Write to temporary file, then upload

#### 4.3 Main Writer (`internal/iceberg/writer/writer.go`)
```go
type Writer interface {
    // EnsureTable creates table if not exists
    EnsureTable(ctx, namespace, table string, schema Schema) error

    // WriteEvents writes a batch of events to Iceberg
    WriteEvents(ctx, namespace, table string, events []BufferedEvent) error

    // Close releases resources
    Close() error
}
```

#### 4.4 Batch Handler (`internal/iceberg/writer/batch_handler.go`)
- Implements `buffer.BatchHandler` signature
- Groups events by table
- Ensures tables exist
- Writes Parquet files
- Commits to catalog

### Phase 5: Integration

#### 5.1 Worker Integration
Update `cmd/philotes-worker/main.go`:
- Create Iceberg writer
- Create batch processor with Iceberg handler
- Start batch processor
- Graceful shutdown

### Phase 6: Testing

#### 6.1 Unit Tests
- Mock catalog for writer tests
- Mock S3 for upload tests
- Type conversion tests
- Schema generation tests

#### 6.2 Integration Tests (manual)
- End-to-end with Docker environment
- Verify Parquet files in MinIO
- Verify table metadata in Lakekeeper

## Configuration

Already defined in `internal/config/config.go`:
```go
IcebergConfig{
    CatalogURL: "http://localhost:8181",
    Warehouse:  "philotes",
}

StorageConfig{
    Endpoint:  "localhost:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Bucket:    "philotes",
    UseSSL:    false,
}
```

## Data Flow

1. **Batch Processor** calls handler with `[]BufferedEvent`
2. **Handler** groups events by `schema.table`
3. For each table:
   a. **EnsureTable** creates table if needed
   b. **WriteParquet** converts events to Parquet file
   c. **Upload** sends file to MinIO
   d. **Commit** updates Iceberg metadata via REST
4. Return success to batch processor
5. Batch processor marks events as processed

## File Naming Convention

```
s3://philotes/warehouse/{namespace}/{table}/data/{partition}/
    {uuid}-{timestamp}.parquet
```

Example:
```
s3://philotes/warehouse/cdc/public_users/data/date=2026-01-24/
    550e8400-e29b-41d4-a716-446655440000-1706122800.parquet
```

## Error Handling

- **Transient errors** (network, S3): Retry with exponential backoff
- **Schema mismatch**: Log warning, attempt schema evolution
- **Catalog errors**: Return error, let batch processor retry
- **Parquet errors**: Return error with event details

## Verification

1. `make build` - Compiles successfully
2. `make test` - All tests pass
3. Manual verification:
   - Start Docker environment
   - Insert data into source database
   - Verify Parquet files appear in MinIO
   - Query Lakekeeper for table metadata
