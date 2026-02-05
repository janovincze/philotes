# Implementation Plan: Issue #24 - Dagster Integration

## Overview

Build `dagster-philotes`, a Python package that provides Dagster resources, assets, and sensors for orchestrating Philotes CDC pipelines within broader data workflows.

## Deliverables

1. **Python SDK** (`dagster-philotes/`) - Dagster integration package
2. **Docker Services** - Dagster webserver + daemon in docker-compose
3. **Examples** - Working DAGs demonstrating common patterns
4. **Documentation** - Usage guide and API reference

---

## Package Structure

```
dagster-philotes/
├── pyproject.toml              # Package metadata, dependencies
├── README.md                   # Usage documentation
├── dagster_philotes/
│   ├── __init__.py             # Public exports
│   ├── client.py               # PhilotesClient - HTTP wrapper
│   ├── resources.py            # PhilotesResource - Dagster resource
│   ├── assets.py               # Asset factories and helpers
│   ├── sensors.py              # Lag sensor, sync sensor
│   ├── ops.py                  # Ops for pipeline control
│   ├── models/
│   │   ├── __init__.py
│   │   ├── pipeline.py         # Pipeline, PipelineStatus
│   │   ├── source.py           # Source model
│   │   ├── metrics.py          # PipelineMetrics, TableMetrics
│   │   └── enums.py            # Status enums
│   └── exceptions.py           # Custom exceptions
├── tests/
│   ├── __init__.py
│   ├── conftest.py             # Fixtures
│   ├── test_client.py
│   ├── test_resources.py
│   ├── test_assets.py
│   └── test_sensors.py
└── examples/
    ├── __init__.py
    ├── definitions.py          # Main Dagster definitions
    ├── basic_pipeline.py       # Simple CDC monitoring
    ├── with_dbt.py             # CDC → dbt pattern
    └── multi_pipeline.py       # Multiple pipelines job
```

---

## Task Breakdown

### Task 1: Project Setup
**Files:** `dagster-philotes/pyproject.toml`, `README.md`

Create Python package with:
- pyproject.toml with dependencies (dagster, pydantic, httpx)
- Basic README with installation instructions
- Package structure

### Task 2: Data Models
**Files:** `dagster_philotes/models/*.py`, `exceptions.py`

Pydantic models matching Philotes API:
- `Pipeline`, `PipelineStatus`, `PipelineConfig`
- `Source`, `SourceStatus`
- `PipelineMetrics`, `TableMetrics`
- Custom exceptions: `PhilotesAPIError`, `PipelineNotFoundError`, etc.

### Task 3: HTTP Client
**Files:** `dagster_philotes/client.py`

`PhilotesClient` class:
- Authentication (API key header)
- CRUD for pipelines and sources
- Pipeline control (start/stop)
- Metrics retrieval
- Health checks
- Retry logic with exponential backoff
- Timeout configuration

### Task 4: Dagster Resource
**Files:** `dagster_philotes/resources.py`

`PhilotesResource` ConfigurableResource:
- Configuration: `api_url`, `api_key`, `timeout`
- Exposes `PhilotesClient` methods
- Dagster-compatible lifecycle

### Task 5: Asset Definitions
**Files:** `dagster_philotes/assets.py`

Asset factories:
- `philotes_pipeline_asset()` - Create asset for a pipeline
- `philotes_table_asset()` - Create asset for specific table
- Metadata extraction (lag, events, timestamps)
- `wait_for_lag()` helper for dependencies

### Task 6: Sensors
**Files:** `dagster_philotes/sensors.py`

Sensors:
- `pipeline_lag_sensor()` - Trigger when lag drops below threshold
- `pipeline_sync_sensor()` - Trigger when pipeline catches up
- `pipeline_error_sensor()` - Trigger on pipeline errors

### Task 7: Ops
**Files:** `dagster_philotes/ops.py`

Operations:
- `start_pipeline` - Start a CDC pipeline
- `stop_pipeline` - Stop a CDC pipeline
- `wait_for_pipeline` - Wait for pipeline to reach status
- `get_pipeline_metrics` - Fetch current metrics

### Task 8: Docker Integration
**Files:** `deployments/docker/docker-compose.yml`, `deployments/docker/dagster/`

Add Dagster services:
- `dagster-webserver` - UI on port 3001
- `dagster-daemon` - Schedules and sensors
- `dagster-postgres` - Dagster metadata database
- Network configuration for Philotes API access
- Volume for code location

### Task 9: Examples
**Files:** `dagster-philotes/examples/*.py`

Working examples:
- Basic pipeline monitoring
- CDC → dbt workflow
- Multi-pipeline coordination
- Sensor-triggered jobs

### Task 10: Tests
**Files:** `dagster-philotes/tests/*.py`

Unit tests:
- Client methods with mocked responses
- Resource configuration
- Asset metadata extraction
- Sensor evaluation logic

---

## API Design

### PhilotesClient

```python
class PhilotesClient:
    def __init__(self, api_url: str, api_key: str, timeout: float = 30.0):
        ...

    # Pipelines
    def list_pipelines(self) -> list[Pipeline]: ...
    def get_pipeline(self, pipeline_id: str) -> Pipeline: ...
    def create_pipeline(self, name: str, source_id: str, config: dict) -> Pipeline: ...
    def start_pipeline(self, pipeline_id: str) -> Pipeline: ...
    def stop_pipeline(self, pipeline_id: str) -> Pipeline: ...
    def get_pipeline_metrics(self, pipeline_id: str) -> PipelineMetrics: ...

    # Sources
    def list_sources(self) -> list[Source]: ...
    def get_source(self, source_id: str) -> Source: ...
    def test_source_connection(self, source_id: str) -> bool: ...

    # Health
    def health_check(self) -> bool: ...

    # Helpers
    def wait_for_lag(self, pipeline_id: str, max_lag_seconds: float, timeout: float) -> PipelineMetrics: ...
```

### PhilotesResource

```python
class PhilotesResource(ConfigurableResource):
    api_url: str
    api_key: str = ""
    timeout: float = 30.0

    def get_client(self) -> PhilotesClient: ...

    # Convenience methods delegating to client
    def get_pipeline(self, pipeline_id: str) -> Pipeline: ...
    def wait_for_lag(self, pipeline_id: str, max_lag_seconds: float) -> PipelineMetrics: ...
```

### Asset Factory

```python
def philotes_pipeline_asset(
    pipeline_id: str,
    name: str | None = None,
    group_name: str = "philotes",
    max_lag_seconds: float = 60.0,
) -> AssetsDefinition:
    """
    Create a Dagster asset representing a Philotes CDC pipeline.

    The asset materializes when the pipeline's replication lag
    drops below max_lag_seconds.
    """
```

### Sensor Factory

```python
def pipeline_lag_sensor(
    pipeline_id: str,
    job: JobDefinition,
    lag_threshold_seconds: float = 10.0,
    minimum_interval_seconds: int = 30,
) -> SensorDefinition:
    """
    Create a sensor that triggers a job when pipeline lag is low.
    """
```

---

## Docker Compose Addition

```yaml
services:
  # ... existing services ...

  dagster-webserver:
    image: dagster/dagster-webserver:latest
    ports:
      - "3001:3000"
    environment:
      DAGSTER_HOME: /opt/dagster/dagster_home
      PHILOTES_API_URL: http://philotes-api:8080
      PHILOTES_API_KEY: ${PHILOTES_API_KEY:-}
    volumes:
      - ./dagster/dagster.yaml:/opt/dagster/dagster_home/dagster.yaml
      - ./dagster/workspace.yaml:/opt/dagster/dagster_home/workspace.yaml
      - ../../dagster-philotes:/opt/dagster/app
    depends_on:
      - dagster-postgres
      - philotes-api
    networks:
      - philotes-network

  dagster-daemon:
    image: dagster/dagster-daemon:latest
    environment:
      DAGSTER_HOME: /opt/dagster/dagster_home
      PHILOTES_API_URL: http://philotes-api:8080
      PHILOTES_API_KEY: ${PHILOTES_API_KEY:-}
    volumes:
      - ./dagster/dagster.yaml:/opt/dagster/dagster_home/dagster.yaml
      - ./dagster/workspace.yaml:/opt/dagster/dagster_home/workspace.yaml
      - ../../dagster-philotes:/opt/dagster/app
    depends_on:
      - dagster-postgres
      - philotes-api
    networks:
      - philotes-network

  dagster-postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: dagster
      POSTGRES_PASSWORD: dagster
      POSTGRES_DB: dagster
    volumes:
      - dagster-postgres-data:/var/lib/postgresql/data
    networks:
      - philotes-network

volumes:
  dagster-postgres-data:
```

---

## Verification

### Unit Tests
```bash
cd dagster-philotes
pip install -e ".[dev]"
pytest tests/ -v
```

### Integration Test
```bash
# Start full environment
docker compose -f deployments/docker/docker-compose.yml up -d

# Verify Dagster UI accessible
curl http://localhost:3001/health

# Run example DAG
# (via Dagster UI or dagster job execute)
```

### Manual Testing Checklist
- [ ] Dagster webserver starts and shows UI
- [ ] PhilotesResource connects to API
- [ ] Pipeline assets show correct metadata
- [ ] Lag sensor triggers downstream jobs
- [ ] Example DAGs execute successfully

---

## Estimated Effort

| Task | Effort |
|------|--------|
| Project Setup | 0.5 day |
| Data Models | 0.5 day |
| HTTP Client | 1 day |
| Dagster Resource | 0.5 day |
| Asset Definitions | 1 day |
| Sensors | 0.5 day |
| Ops | 0.5 day |
| Docker Integration | 1 day |
| Examples | 0.5 day |
| Tests | 1 day |
| **Total** | **~7 days** |

---

## Files to Create

| File | Purpose |
|------|---------|
| `dagster-philotes/pyproject.toml` | Package configuration |
| `dagster-philotes/README.md` | Documentation |
| `dagster-philotes/dagster_philotes/__init__.py` | Public exports |
| `dagster-philotes/dagster_philotes/client.py` | HTTP client |
| `dagster-philotes/dagster_philotes/resources.py` | Dagster resource |
| `dagster-philotes/dagster_philotes/assets.py` | Asset factories |
| `dagster-philotes/dagster_philotes/sensors.py` | Sensors |
| `dagster-philotes/dagster_philotes/ops.py` | Operations |
| `dagster-philotes/dagster_philotes/models/*.py` | Data models |
| `dagster-philotes/dagster_philotes/exceptions.py` | Exceptions |
| `dagster-philotes/tests/*.py` | Unit tests |
| `dagster-philotes/examples/*.py` | Example DAGs |
| `deployments/docker/docker-compose.yml` | Docker services (modify) |
| `deployments/docker/dagster/dagster.yaml` | Dagster config |
| `deployments/docker/dagster/workspace.yaml` | Workspace config |

---

## Success Criteria

1. `pip install dagster-philotes` works
2. Dagster UI shows Philotes pipelines as assets
3. Sensors trigger jobs when CDC catches up
4. Example DAGs demonstrate real workflows
5. Documentation covers common use cases
