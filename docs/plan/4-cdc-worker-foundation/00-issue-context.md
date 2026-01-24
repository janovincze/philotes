# Issue Context - CDC-001: pgstream Integration and CDC Worker Foundation

## Issue Details
- **Number:** #4
- **Title:** CDC-001: pgstream Integration and CDC Worker Foundation
- **Labels:** epic:cdc, phase:mvp, priority:critical, type:feature
- **Milestone:** M1: Core Pipeline

## Description
Core CDC worker service using pgstream library for PostgreSQL logical replication.

## Estimate
~12,000 LOC

## Acceptance Criteria
- [ ] pgstream library integration with configuration
- [ ] WAL event parsing and normalization
- [ ] Connection management with reconnection logic
- [ ] Replication slot management
- [ ] Schema log table creation and tracking
- [ ] Checkpoint management for exactly-once semantics
- [ ] Structured logging with correlation IDs
- [ ] Graceful shutdown handling

## Key Dependencies
- [github.com/xataio/pgstream](https://github.com/xataio/pgstream) - PostgreSQL CDC library
- Standard Go libraries for PostgreSQL (pgx)

## Configuration Structure
```go
type CDCConfig struct {
    Source      PostgresConfig
    Replication ReplicationConfig
    Buffer      BufferConfig
    Checkpoint  CheckpointConfig
}
```

## Dependencies
- FOUND-001 (completed)

## Blocks
- CDC-002: Buffer Database Implementation
- OBS-001: Observability
