# Implementation Plan - Issue #9: API-002 Source and Pipeline Management

## Overview

Implement CRUD operations for sources, pipelines, and destinations. This builds on the Gin-based API framework from API-001 and adds:
1. Database schema for metadata storage
2. Repository layer for data access
3. Service layer for business logic
4. HTTP handlers for REST endpoints

## Architecture

```
                        ┌─────────────┐
                        │   Handler   │  HTTP request/response
                        └──────┬──────┘
                               │
                        ┌──────▼──────┐
                        │   Service   │  Business logic, validation
                        └──────┬──────┘
                               │
                        ┌──────▼──────┐
                        │ Repository  │  Data access
                        └──────┬──────┘
                               │
                        ┌──────▼──────┐
                        │  PostgreSQL │  Storage
                        └─────────────┘
```

## Database Schema

### New Tables (add to `05-source-management-schema.sql`)

```sql
-- Sources table stores registered source databases
CREATE TABLE IF NOT EXISTS philotes.sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'postgresql',
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 5432,
    database TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,  -- encrypted at rest
    ssl_mode TEXT NOT NULL DEFAULT 'prefer',
    slot_name TEXT,
    publication_name TEXT,
    status TEXT NOT NULL DEFAULT 'inactive',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pipelines table stores pipeline definitions
CREATE TABLE IF NOT EXISTS philotes.pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    source_id UUID NOT NULL REFERENCES philotes.sources(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'stopped',
    config JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    stopped_at TIMESTAMPTZ
);

-- Table mappings for pipelines
CREATE TABLE IF NOT EXISTS philotes.table_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES philotes.pipelines(id) ON DELETE CASCADE,
    source_schema TEXT NOT NULL,
    source_table TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(pipeline_id, source_schema, source_table)
);
```

## API Endpoints

### Sources

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/sources` | Create a new source |
| GET | `/api/v1/sources` | List all sources |
| GET | `/api/v1/sources/:id` | Get source by ID |
| PUT | `/api/v1/sources/:id` | Update source |
| DELETE | `/api/v1/sources/:id` | Delete source |
| POST | `/api/v1/sources/:id/test` | Test source connection |
| GET | `/api/v1/sources/:id/tables` | Discover tables in source |

### Pipelines

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/pipelines` | Create a new pipeline |
| GET | `/api/v1/pipelines` | List all pipelines |
| GET | `/api/v1/pipelines/:id` | Get pipeline by ID |
| PUT | `/api/v1/pipelines/:id` | Update pipeline |
| DELETE | `/api/v1/pipelines/:id` | Delete pipeline |
| POST | `/api/v1/pipelines/:id/start` | Start pipeline |
| POST | `/api/v1/pipelines/:id/stop` | Stop pipeline |
| GET | `/api/v1/pipelines/:id/status` | Get pipeline status |

## Package Structure

```
internal/api/
├── handlers/
│   ├── sources.go         # Source CRUD handlers
│   └── pipelines.go       # Pipeline CRUD handlers
├── services/
│   ├── source.go          # Source business logic
│   └── pipeline.go        # Pipeline business logic
├── repositories/
│   ├── source.go          # Source data access
│   └── pipeline.go        # Pipeline data access
└── models/
    ├── source.go          # Source request/response models
    └── pipeline.go        # Pipeline request/response models
```

## Task Breakdown

### Phase 1: Database Schema
1. Create `05-source-management-schema.sql` with sources, pipelines, table_mappings tables

### Phase 2: API Models
2. Create `internal/api/models/source.go` - Source request/response types
3. Create `internal/api/models/pipeline.go` - Pipeline request/response types

### Phase 3: Repository Layer
4. Create `internal/api/repositories/source.go` - Source CRUD operations
5. Create `internal/api/repositories/pipeline.go` - Pipeline CRUD operations

### Phase 4: Service Layer
6. Create `internal/api/services/source.go` - Source business logic (including connection test)
7. Create `internal/api/services/pipeline.go` - Pipeline business logic (including start/stop)

### Phase 5: Handlers
8. Create `internal/api/handlers/sources.go` - Source HTTP handlers
9. Create `internal/api/handlers/pipelines.go` - Pipeline HTTP handlers

### Phase 6: Integration
10. Update `internal/api/server.go` - Wire up new handlers and dependencies
11. Update `cmd/philotes-api/main.go` - Initialize database connection for repositories

### Phase 7: Tests
12. Write unit tests for repositories
13. Write unit tests for services
14. Write integration tests for handlers

### Phase 8: Cleanup
15. Remove stub handlers (`internal/api/handlers/stubs.go`)
16. Update OpenAPI specification

## Files to Create

| File | Description | Est. Lines |
|------|-------------|------------|
| `deployments/docker/init-scripts/05-source-management-schema.sql` | Database schema | ~80 |
| `internal/api/models/source.go` | Source models | ~120 |
| `internal/api/models/pipeline.go` | Pipeline models | ~100 |
| `internal/api/repositories/source.go` | Source repository | ~200 |
| `internal/api/repositories/pipeline.go` | Pipeline repository | ~200 |
| `internal/api/services/source.go` | Source service | ~180 |
| `internal/api/services/pipeline.go` | Pipeline service | ~200 |
| `internal/api/handlers/sources.go` | Source handlers | ~250 |
| `internal/api/handlers/pipelines.go` | Pipeline handlers | ~250 |
| `internal/api/repositories/source_test.go` | Repository tests | ~200 |
| `internal/api/services/source_test.go` | Service tests | ~200 |
| `internal/api/handlers/sources_test.go` | Handler tests | ~250 |

**Total:** ~2,230 LOC

## Files to Modify

| File | Changes |
|------|---------|
| `internal/api/server.go` | Add database dependency, register new routes |
| `cmd/philotes-api/main.go` | Initialize database connection |
| `api/openapi/openapi.yaml` | Add source/pipeline endpoint specs |

## Key Design Decisions

### 1. Password Storage
- Store encrypted passwords in database
- Use Go's `crypto/aes` for encryption with key from config
- Never return passwords in API responses (mask or omit)

### 2. Connection Testing
- Use `database/sql` with pgx driver
- Set short timeout (5s) for connection test
- Return detailed error messages on failure

### 3. Table Discovery
- Query `information_schema.tables` on source database
- Filter by schema (default: public)
- Include column information for each table

### 4. Pipeline Lifecycle
- Pipelines are "logical" - they define what to sync
- Start/stop integrates with existing CDC pipeline orchestrator
- Status tracking in database with error messages

### 5. Error Handling
- Use RFC 7807 Problem Details (existing pattern)
- Add `ErrorTypeConflict` for duplicate name errors
- Validate at service layer before repository calls

## Verification

1. `make build` - Compiles successfully
2. `make lint` - No lint errors
3. `make test` - All tests pass
4. Manual testing:
   ```bash
   # Create source
   curl -X POST http://localhost:8080/api/v1/sources \
     -H "Content-Type: application/json" \
     -d '{"name":"test-source","host":"localhost","port":5432,"database":"test","username":"user","password":"pass"}'

   # Test connection
   curl -X POST http://localhost:8080/api/v1/sources/{id}/test

   # Discover tables
   curl http://localhost:8080/api/v1/sources/{id}/tables

   # Create pipeline
   curl -X POST http://localhost:8080/api/v1/pipelines \
     -H "Content-Type: application/json" \
     -d '{"name":"test-pipeline","source_id":"{source-id}","tables":[{"schema":"public","table":"users"}]}'
   ```
