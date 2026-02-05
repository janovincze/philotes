"""Pytest fixtures for dagster-philotes tests."""

import pytest

from dagster_philotes.models import (
    Pipeline,
    PipelineConfig,
    PipelineMetrics,
    PipelineStatus,
    Source,
    SourceStatus,
    TableMetrics,
)


@pytest.fixture
def sample_pipeline() -> Pipeline:
    """Create a sample pipeline for testing."""
    return Pipeline(
        id="test-pipeline-123",
        name="test-pipeline",
        source_id="source-456",
        status=PipelineStatus.RUNNING,
        config=PipelineConfig(batch_size=1000, flush_interval_ms=5000),
        tables=[],
    )


@pytest.fixture
def sample_pipeline_stopped() -> Pipeline:
    """Create a stopped pipeline for testing."""
    return Pipeline(
        id="test-pipeline-stopped",
        name="test-pipeline-stopped",
        source_id="source-456",
        status=PipelineStatus.STOPPED,
        config=PipelineConfig(),
        tables=[],
    )


@pytest.fixture
def sample_metrics() -> PipelineMetrics:
    """Create sample metrics for testing."""
    return PipelineMetrics(
        pipeline_id="test-pipeline-123",
        status=PipelineStatus.RUNNING,
        events_processed=10000,
        events_per_second=150.5,
        lag_seconds=5.2,
        lag_p95_seconds=8.1,
        buffer_depth=50,
        error_count=0,
        iceberg_commits=100,
        iceberg_bytes_written=1024000,
        uptime="2h 30m",
        tables=[
            TableMetrics(
                schema="public",
                table="orders",
                events_processed=5000,
                lag_seconds=5.0,
            ),
            TableMetrics(
                schema="public",
                table="customers",
                events_processed=5000,
                lag_seconds=5.5,
            ),
        ],
    )


@pytest.fixture
def sample_metrics_high_lag() -> PipelineMetrics:
    """Create sample metrics with high lag for testing."""
    return PipelineMetrics(
        pipeline_id="test-pipeline-123",
        status=PipelineStatus.RUNNING,
        events_processed=5000,
        events_per_second=50.0,
        lag_seconds=120.5,
        buffer_depth=1000,
        error_count=0,
        uptime="1h",
        tables=[],
    )


@pytest.fixture
def sample_source() -> Source:
    """Create a sample source for testing."""
    return Source(
        id="source-456",
        name="test-source",
        type="postgresql",
        host="localhost",
        port=5432,
        database_name="testdb",
        username="testuser",
        ssl_mode="disable",
        status=SourceStatus.ACTIVE,
    )


@pytest.fixture
def api_responses() -> dict:
    """Sample API responses for mocking."""
    return {
        "pipeline": {
            "id": "test-pipeline-123",
            "name": "test-pipeline",
            "source_id": "source-456",
            "status": "running",
            "config": {"batch_size": 1000},
            "tables": [],
        },
        "pipelines": [
            {
                "id": "test-pipeline-123",
                "name": "test-pipeline",
                "source_id": "source-456",
                "status": "running",
                "config": {},
                "tables": [],
            }
        ],
        "metrics": {
            "pipeline_id": "test-pipeline-123",
            "status": "running",
            "events_processed": 10000,
            "events_per_second": 150.5,
            "lag_seconds": 5.2,
            "buffer_depth": 50,
            "error_count": 0,
            "uptime": "2h 30m",
            "tables": [],
        },
        "source": {
            "id": "source-456",
            "name": "test-source",
            "type": "postgresql",
            "host": "localhost",
            "port": 5432,
            "database_name": "testdb",
            "username": "testuser",
            "ssl_mode": "disable",
            "status": "active",
        },
        "health": {"status": "ok"},
    }
