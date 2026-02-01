# Implementation Plan - Issue #92

## Overview

Implement a complete E2E demonstration of Philotes CDC pipeline with:
1. E-commerce sample database with seed data
2. Getting Started guide
3. Comprehensive E2E tests

## Approach

We'll use an **e-commerce dataset** (customers, products, orders, order_items) because:
- Demonstrates relational data with foreign keys
- Natural CRUD operations (orders being created, updated, fulfilled)
- Various data types (UUID, timestamps, JSON, numeric, text)
- Realistic for CDC use cases

## Task Breakdown

### Phase 1: Sample Database Setup

#### Task 1.1: Create Sample Database Schema
**File**: `deployments/docker/init-scripts-source/01-ecommerce-schema.sql`

Tables:
- `customers` - Customer data (UUID, name, email, JSON metadata, timestamps)
- `products` - Product catalog (UUID, name, price, inventory, JSON attributes)
- `orders` - Order headers (UUID, customer FK, status, totals, timestamps)
- `order_items` - Order line items (UUID, order FK, product FK, quantity, price)

All tables will have:
- UUID primary keys
- created_at/updated_at timestamps
- Indexes for common queries

#### Task 1.2: Create Seed Data Script
**File**: `deployments/docker/init-scripts-source/02-ecommerce-seed.sql`

Seed data:
- 100 customers
- 50 products
- 500 orders with ~2000 order items
- Realistic data distribution

#### Task 1.3: Update Docker Compose
**File**: `deployments/docker/docker-compose.yml`

- Add volume mount for postgres-source init scripts
- Ensure logical replication is enabled (already done)

### Phase 2: Getting Started Guide

#### Task 2.1: Create Guide Structure
**File**: `docs/guides/getting-started.md`

Sections:
1. Prerequisites
2. Quick Start
3. Starting the Environment
4. Creating Your First Source
5. Creating a Pipeline
6. Starting CDC Replication
7. Verifying Data in Iceberg
8. Querying with Trino
9. Monitoring with Grafana
10. Next Steps
11. Troubleshooting

Include:
- cURL examples for all API calls
- Expected responses
- Trino SQL queries
- Screenshots (optional)

### Phase 3: E2E Tests

#### Task 3.1: Create E2E Test Infrastructure
**File**: `tests/e2e/setup_test.go`

- Test suite setup/teardown
- API client helpers
- Polling utilities for async operations
- Database connection helpers

#### Task 3.2: Source Management E2E Tests
**File**: `tests/e2e/source_test.go`

Tests:
- Create source with valid credentials
- Test source connection
- Discover tables from source
- List and get sources
- Delete source

#### Task 3.3: Pipeline Management E2E Tests
**File**: `tests/e2e/pipeline_test.go`

Tests:
- Create pipeline with table mappings
- Start pipeline and verify running
- Check pipeline status
- Stop pipeline gracefully
- Delete pipeline

#### Task 3.4: CDC Data Flow E2E Tests
**File**: `tests/e2e/cdc_flow_test.go`

Tests:
- Full flow: source → pipeline → insert data → verify in Iceberg
- INSERT operations captured
- UPDATE operations captured
- DELETE operations captured
- Batch writes verified
- Trino query verification

#### Task 3.5: Data Generator Utility
**File**: `tools/datagen/main.go`

CLI tool for generating continuous data:
- Configurable insert rate
- Random customer/order generation
- Update and delete operations
- Useful for demos and load testing

### Phase 4: Bug Fixes (As Discovered)

Document and fix any issues found during testing:
- API endpoint bugs
- CDC pipeline issues
- Docker compose problems
- Schema issues

## Files to Create/Modify

### New Files
| File | Purpose |
|------|---------|
| `deployments/docker/init-scripts-source/01-ecommerce-schema.sql` | Sample schema |
| `deployments/docker/init-scripts-source/02-ecommerce-seed.sql` | Seed data |
| `docs/guides/getting-started.md` | Usage guide |
| `tests/e2e/setup_test.go` | E2E test infrastructure |
| `tests/e2e/source_test.go` | Source E2E tests |
| `tests/e2e/pipeline_test.go` | Pipeline E2E tests |
| `tests/e2e/cdc_flow_test.go` | CDC flow E2E tests |
| `tools/datagen/main.go` | Data generator tool |

### Modified Files
| File | Change |
|------|--------|
| `deployments/docker/docker-compose.yml` | Add source init scripts volume |
| `Makefile` | Add e2e test target |
| `README.md` | Reference getting-started guide |

## Test Strategy

### E2E Test Requirements
- Tests require docker-compose environment running
- Use build tag `//go:build e2e` to separate from unit tests
- Tests should be idempotent (clean up after themselves)
- Use reasonable timeouts for async operations (30s for pipeline start, 60s for CDC)

### Test Execution
```bash
# Start environment
make docker-up

# Run E2E tests
make test-e2e

# Or manually
go test -tags=e2e -v ./tests/e2e/...
```

## API Schema Reference

### Create Source
```json
POST /api/v1/sources
{
  "name": "ecommerce-source",
  "host": "postgres-source",
  "port": 5432,
  "database_name": "source",
  "username": "source",
  "password": "source",
  "ssl_mode": "disable",
  "slot_name": "philotes_slot",
  "publication_name": "philotes_pub"
}
```

### Create Pipeline
```json
POST /api/v1/pipelines
{
  "name": "ecommerce-pipeline",
  "source_id": "<source-uuid>",
  "destination_type": "iceberg",
  "config": {
    "batch_size": 1000,
    "flush_interval": "10s"
  }
}
```

### Add Table Mapping
```json
POST /api/v1/pipelines/:id/tables
{
  "source_schema": "public",
  "source_table": "customers",
  "destination_schema": "ecommerce",
  "destination_table": "customers",
  "enabled": true
}
```

## Estimated Complexity

| Phase | Tasks | Complexity |
|-------|-------|------------|
| Phase 1: Sample DB | 3 | Medium |
| Phase 2: Guide | 1 | Medium |
| Phase 3: E2E Tests | 5 | High |
| Phase 4: Bug Fixes | TBD | Variable |

## Success Criteria

1. `docker compose up` starts with sample data loaded
2. New user can follow guide and see data in Iceberg
3. All E2E tests pass
4. `make test-e2e` works in CI
5. README references the new guide
