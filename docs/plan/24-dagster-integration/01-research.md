# Research Findings: Issue #24 - Dagster Integration

## 1. API Endpoints for Dagster Integration

### Core Pipeline Management
```
POST   /api/v1/pipelines              - Create new pipeline
GET    /api/v1/pipelines              - List all pipelines
GET    /api/v1/pipelines/:id          - Get specific pipeline
PUT    /api/v1/pipelines/:id          - Update pipeline
DELETE /api/v1/pipelines/:id          - Delete pipeline
POST   /api/v1/pipelines/:id/start    - Start pipeline
POST   /api/v1/pipelines/:id/stop     - Stop pipeline
GET    /api/v1/pipelines/:id/status   - Get pipeline status
POST   /api/v1/pipelines/:id/tables   - Add table mapping
DELETE /api/v1/pipelines/:id/tables/:mappingId - Remove table mapping
GET    /api/v1/pipelines/:id/metrics  - Get current metrics
GET    /api/v1/pipelines/:id/metrics/history?range=1h - Historical metrics
```

### Source Management
```
POST   /api/v1/sources                - Create source
GET    /api/v1/sources                - List sources
GET    /api/v1/sources/:id            - Get source
PUT    /api/v1/sources/:id            - Update source
DELETE /api/v1/sources/:id            - Delete source
POST   /api/v1/sources/:id/test       - Test connection
GET    /api/v1/sources/:id/tables     - Discover tables
```

### Health & Status
```
GET    /health                        - Overall health status
GET    /health/live                   - Liveness probe
GET    /health/ready                  - Readiness probe
GET    /api/v1/version                - API version info
GET    /metrics                       - Prometheus metrics
```

## 2. Data Models

### Pipeline
```python
Pipeline:
  - id: UUID
  - name: str
  - source_id: UUID
  - status: PipelineStatus (stopped|starting|running|stopping|error)
  - config: dict[str, Any]
  - error_message: str (optional)
  - tables: list[TableMapping]
  - created_at: datetime
  - updated_at: datetime
```

### Pipeline Metrics
```python
PipelineMetrics:
  - pipeline_id: UUID
  - status: PipelineStatus
  - events_processed: int
  - events_per_second: float
  - lag_seconds: float
  - buffer_depth: int
  - error_count: int
  - iceberg_commits: int
  - last_event_at: datetime (optional)
  - tables: list[TableMetrics]
```

### Source
```python
Source:
  - id: UUID
  - name: str
  - type: str (postgresql)
  - host: str
  - port: int
  - database_name: str
  - username: str
  - ssl_mode: str
  - status: SourceStatus
```

## 3. Authentication

**Recommended: API Keys** for Dagster-to-Philotes communication

- Generated via `/api/v1/api-keys`
- Sent as `Authorization: Bearer <key>` header
- Auth can be disabled: `PHILOTES_AUTH_ENABLED=false`

## 4. Recommended Package Structure

```
dagster-philotes/
├── pyproject.toml
├── README.md
├── dagster_philotes/
│   ├── __init__.py
│   ├── client.py           # HTTP client wrapper
│   ├── resources.py        # Dagster Resource definition
│   ├── assets.py           # Asset factory functions
│   ├── sensors.py          # Sensor definitions
│   ├── ops.py              # Op definitions
│   ├── models/
│   │   ├── __init__.py
│   │   ├── pipeline.py
│   │   ├── source.py
│   │   └── metrics.py
│   └── types.py            # Type definitions
├── tests/
│   ├── test_client.py
│   ├── test_resources.py
│   └── conftest.py
└── examples/
    ├── basic_pipeline.py
    ├── with_dbt.py
    └── multi_pipeline_job.py
```

## 5. Key Dependencies

```
dagster >= 1.6.0
dagster-webserver >= 1.6.0
pydantic >= 2.0
httpx >= 0.25.0
```

## 6. Docker Deployment

Current setup uses multi-stage builds with distroless images. Dagster will need:
- New service in `docker-compose.yml`
- Dagster webserver + daemon containers
- Network access to Philotes API
- PostgreSQL for Dagster's own metadata

## 7. Implementation Priorities

### Phase 1 (MVP)
1. Python HTTP client with auth
2. `PhilotesResource` for Dagster
3. Asset definition helpers
4. Basic lag sensor
5. Example DAGs

### Phase 2
1. Advanced sensors
2. Job templates
3. Integration tests

## 8. Blockers

- None for MVP
- Issue #103 (RBAC) needed for enterprise features
