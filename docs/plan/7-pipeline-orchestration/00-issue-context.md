# Issue Context: CDC-004 - End-to-End Pipeline Orchestration

## Issue Details
- **Number:** #7
- **Title:** CDC-004: End-to-End Pipeline Orchestration
- **Labels:** epic:cdc, phase:mvp, priority:high, type:feature
- **Milestone:** M1: Core Pipeline
- **Estimate:** ~10,000 LOC

## Description
Pipeline coordinator that ties CDC source, buffer, and Iceberg writer together.

## Acceptance Criteria
- [ ] Pipeline lifecycle management (start, stop, pause, resume)
- [ ] Backpressure handling between components
- [ ] Error handling with retry policies (exponential backoff)
- [ ] Dead-letter queue for failed events
- [ ] Health check endpoints
- [ ] Pipeline state machine
- [ ] Configuration validation
- [ ] Hot reload for configuration changes

## Dependencies
- CDC-001 (pgstream Integration) - COMPLETED
- CDC-002 (Buffer Database) - COMPLETED
- CDC-003 (Apache Iceberg Writer) - COMPLETED

## Blocks
- API-001 (Core Management API Framework)

## Current Architecture

```
PostgreSQL Source
       ↓
   pgstream Reader (CDC-001)
       ↓
   Pipeline.processEvent()
       ↓
   Buffer.Write() (CDC-002)
       ↓
   [PostgreSQL Buffer Table]
       ↓
   Batch Processor
       ↓
   Iceberg Writer (CDC-003)
       ↓
   Lakekeeper + MinIO
```

## What's Missing
- Pipeline state machine with proper lifecycle
- Backpressure handling
- Error handling with retries
- Dead-letter queue for failed events
- Health checks
- Configuration validation
- Hot reload capability
