# dagster-philotes

Dagster integration for [Philotes](https://github.com/janovincze/philotes) CDC platform.

## Overview

`dagster-philotes` provides Dagster resources, assets, and sensors for orchestrating Philotes CDC pipelines within broader data workflows. This enables you to:

- Represent CDC pipelines as Dagster assets with lag monitoring
- Trigger downstream jobs when CDC catches up
- Coordinate CDC with dbt, ML training, and other data workflows
- Monitor replication lag and pipeline health in Dagster UI

## Installation

```bash
pip install dagster-philotes
```

Or install from source:

```bash
cd dagster-philotes
pip install -e ".[dev]"
```

## Quick Start

### 1. Configure the Philotes Resource

```python
from dagster import Definitions
from dagster_philotes import PhilotesResource

defs = Definitions(
    resources={
        "philotes": PhilotesResource(
            api_url="http://localhost:8080",
            api_key="your-api-key",
        ),
    },
)
```

### 2. Create Pipeline Assets

```python
from dagster_philotes import philotes_pipeline_asset

# Create an asset representing a CDC pipeline
orders_cdc = philotes_pipeline_asset(
    pipeline_id="orders-cdc",
    name="orders_iceberg",
    max_lag_seconds=60.0,
)
```

### 3. Use Sensors for Downstream Triggers

```python
from dagster import job, op
from dagster_philotes import pipeline_lag_sensor

@op
def run_dbt_models():
    # Run dbt models after CDC catches up
    pass

@job
def downstream_dbt():
    run_dbt_models()

# Sensor triggers when lag < 10 seconds
orders_caught_up = pipeline_lag_sensor(
    pipeline_id="orders-cdc",
    job=downstream_dbt,
    lag_threshold_seconds=10.0,
)
```

## API Reference

### PhilotesResource

Dagster resource for connecting to Philotes API.

```python
from dagster_philotes import PhilotesResource

resource = PhilotesResource(
    api_url="http://localhost:8080",  # Philotes API URL
    api_key="pk_xxx",                  # API key (optional if auth disabled)
    timeout=30.0,                      # Request timeout in seconds
)
```

### PhilotesClient

Low-level HTTP client for Philotes API.

```python
from dagster_philotes import PhilotesClient

client = PhilotesClient(
    api_url="http://localhost:8080",
    api_key="pk_xxx",
)

# List all pipelines
pipelines = client.list_pipelines()

# Get pipeline metrics
metrics = client.get_pipeline_metrics("pipeline-id")

# Wait for lag to drop below threshold
metrics = client.wait_for_lag("pipeline-id", max_lag_seconds=60.0, timeout=300.0)
```

### Asset Factories

Create Dagster assets from Philotes pipelines.

```python
from dagster_philotes import philotes_pipeline_asset

# Basic usage
asset = philotes_pipeline_asset(pipeline_id="my-pipeline")

# With configuration
asset = philotes_pipeline_asset(
    pipeline_id="my-pipeline",
    name="custom_asset_name",
    group_name="cdc_pipelines",
    max_lag_seconds=120.0,
)
```

### Sensors

Create sensors that trigger on pipeline conditions.

```python
from dagster_philotes import pipeline_lag_sensor, pipeline_sync_sensor

# Trigger when lag drops below threshold
lag_sensor = pipeline_lag_sensor(
    pipeline_id="my-pipeline",
    job=my_downstream_job,
    lag_threshold_seconds=10.0,
    minimum_interval_seconds=30,
)

# Trigger when pipeline is fully caught up
sync_sensor = pipeline_sync_sensor(
    pipeline_id="my-pipeline",
    job=my_downstream_job,
)
```

### Ops

Operations for pipeline control.

```python
from dagster_philotes import start_pipeline_op, stop_pipeline_op, wait_for_pipeline_op

@job
def manage_pipeline():
    started = start_pipeline_op()
    wait_for_pipeline_op(started)
    # ... do work ...
    stop_pipeline_op()
```

## Examples

See the [examples](./examples/) directory for complete working examples:

- `basic_pipeline.py` - Simple CDC pipeline monitoring
- `with_dbt.py` - CDC → dbt workflow
- `definitions.py` - Full Dagster definitions

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PHILOTES_API_URL` | Philotes API URL | `http://localhost:8080` |
| `PHILOTES_API_KEY` | API key for authentication | (none) |

### Docker Compose

When running with Docker Compose, the Philotes API is available at:

```yaml
PHILOTES_API_URL: http://philotes-api:8080
```

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run linter
ruff check .

# Run type checker
mypy dagster_philotes
```

## License

Apache 2.0
