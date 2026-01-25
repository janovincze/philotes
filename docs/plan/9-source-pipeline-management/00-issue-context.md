# Issue Context - API-002: Source and Pipeline Management

## Issue Details
- **Number:** #9
- **Title:** API-002: Source and Pipeline Management
- **Priority:** High
- **Phase:** MVP
- **Milestone:** M2: Management Layer

## Description
CRUD operations for sources, pipelines, and destinations.

## Acceptance Criteria
- [ ] Source registration (PostgreSQL connection details)
- [ ] Connection testing endpoint
- [ ] Table discovery and selection
- [ ] Pipeline creation with source-to-destination mapping
- [ ] Pipeline status monitoring
- [ ] Table sync status tracking
- [ ] Batch operations (start/stop multiple pipelines)
- [ ] Configuration persistence (PostgreSQL metadata store)

## Data Models
```go
type Source struct {
    ID          uuid.UUID
    Name        string
    Type        string // "postgresql"
    Config      SourceConfig
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Pipeline struct {
    ID            uuid.UUID
    Name          string
    SourceID      uuid.UUID
    DestinationID uuid.UUID
    Tables        []TableMapping
    Status        PipelineStatus
    Config        PipelineConfig
}
```

## Dependencies
- API-001 (Complete) - Core Management API Framework

## Blocks
- API-003 - Advanced Pipeline Features

## Notes
This issue implements the core CRUD functionality for managing CDC sources, pipelines, and destinations. It builds on the Gin-based API framework from API-001 and requires:
1. Database schema for metadata storage
2. Repository layer for data access
3. Service layer for business logic
4. HTTP handlers for REST endpoints
5. Input validation and error handling
