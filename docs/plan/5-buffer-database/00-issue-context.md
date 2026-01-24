# Issue Context - CDC-002: Buffer Database Implementation

## Issue Details
- **Number:** #5
- **Title:** CDC-002: Buffer Database Implementation
- **Labels:** epic:cdc, phase:mvp, priority:critical, type:feature
- **Milestone:** M1: Core Pipeline

## Description
Intermediate buffer for CDC events to ensure reliability and replay capability.

## Estimate
~8,000 LOC

## Acceptance Criteria
- [ ] PostgreSQL-based event buffer with schema
- [ ] Event serialization/deserialization (Protocol Buffers or MessagePack)
- [ ] Batch processing with configurable size and timeout
- [ ] Position tracking per source table
- [ ] Cleanup of processed events (configurable retention)
- [ ] Metrics for buffer depth and lag
- [ ] Replay capability for failed batches

## Schema Design
```sql
CREATE TABLE cdc_events (
    id BIGSERIAL PRIMARY KEY,
    source_id UUID NOT NULL,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    key_columns JSONB,
    before_data JSONB,
    after_data JSONB,
    transaction_id BIGINT,
    lsn TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_cdc_events_unprocessed
ON cdc_events (source_id, table_name, created_at)
WHERE processed_at IS NULL;
```

## Dependencies
- CDC-001 (completed)

## Blocks
- CDC-003: Apache Iceberg Writer
