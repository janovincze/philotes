# Issue Context: CDC-003 - Apache Iceberg Writer

## Issue Details
- **Number:** #6
- **Title:** CDC-003: Apache Iceberg Writer
- **Labels:** epic:cdc, phase:mvp, priority:critical, type:feature
- **Milestone:** M1: Core Pipeline
- **Estimate:** ~15,000 LOC

## Description
Iceberg table writer with REST catalog (Lakekeeper) integration.

## Acceptance Criteria
- [ ] Lakekeeper REST catalog client implementation
- [ ] Iceberg table creation with configurable partitioning
- [ ] Parquet file writing with proper Iceberg metadata
- [ ] Data file management (create, commit)
- [ ] Schema evolution handling (column add/drop/rename)
- [ ] Partition evolution support
- [ ] Manifest file generation
- [ ] Snapshot management
- [ ] S3/MinIO client integration

## Key Integrations
- [Lakekeeper REST API](https://docs.lakekeeper.io/)
- MinIO S3 API
- Parquet-go library for file writing

## Technical Note
Since Go doesn't have a native Iceberg library like Java/Python, this will require implementing the Iceberg spec or using REST API exclusively for metadata while writing Parquet files directly.

## Dependencies
- CDC-002 (Buffer Database) - COMPLETED

## Blocks
- CDC-004 (End-to-End Pipeline Orchestration)

## Architecture Context

```
Buffer Database → Batch Processor → Iceberg Writer
                                         ↓
                                    ┌────────────┐
                                    │ Lakekeeper │ (REST Catalog)
                                    └────────────┘
                                         ↓
                                    ┌────────────┐
                                    │   MinIO    │ (S3 Storage)
                                    └────────────┘
```

The Iceberg Writer receives batched CDC events from the buffer and:
1. Converts events to Parquet files
2. Uploads files to MinIO/S3
3. Commits metadata to Lakekeeper REST catalog
