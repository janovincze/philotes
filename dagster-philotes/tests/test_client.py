"""Tests for PhilotesClient."""

import pytest
from pytest_httpx import HTTPXMock

from dagster_philotes import (
    PhilotesClient,
    Pipeline,
    PipelineMetrics,
    PipelineNotFoundError,
    PipelineStatus,
    PhilotesAPIError,
    PhilotesAuthenticationError,
    Source,
)


class TestPhilotesClient:
    """Tests for PhilotesClient."""

    def test_init(self):
        """Test client initialization."""
        client = PhilotesClient(
            api_url="http://localhost:8080",
            api_key="test-key",
            timeout=60.0,
        )
        assert client.api_url == "http://localhost:8080"
        assert client.api_key == "test-key"
        assert client.timeout == 60.0

    def test_init_strips_trailing_slash(self):
        """Test that trailing slashes are stripped from API URL."""
        client = PhilotesClient(api_url="http://localhost:8080/")
        assert client.api_url == "http://localhost:8080"

    def test_health_check_success(self, httpx_mock: HTTPXMock):
        """Test successful health check."""
        httpx_mock.add_response(
            url="http://localhost:8080/health",
            json={"status": "ok"},
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            assert client.health_check() is True

    def test_health_check_failure(self, httpx_mock: HTTPXMock):
        """Test failed health check."""
        httpx_mock.add_response(
            url="http://localhost:8080/health",
            status_code=500,
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            assert client.health_check() is False

    def test_list_pipelines(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test listing pipelines."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines",
            json=api_responses["pipelines"],
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            pipelines = client.list_pipelines()

        assert len(pipelines) == 1
        assert pipelines[0].id == "test-pipeline-123"
        assert isinstance(pipelines[0], Pipeline)

    def test_get_pipeline(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test getting a single pipeline."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/test-pipeline-123",
            json=api_responses["pipeline"],
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            pipeline = client.get_pipeline("test-pipeline-123")

        assert pipeline.id == "test-pipeline-123"
        assert pipeline.name == "test-pipeline"
        assert pipeline.status == PipelineStatus.RUNNING

    def test_get_pipeline_not_found(self, httpx_mock: HTTPXMock):
        """Test getting a non-existent pipeline."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/nonexistent",
            status_code=404,
            json={"error": "not found"},
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            with pytest.raises(PipelineNotFoundError) as exc_info:
                client.get_pipeline("nonexistent")

        assert "nonexistent" in str(exc_info.value)

    def test_start_pipeline(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test starting a pipeline."""
        response = api_responses["pipeline"].copy()
        response["status"] = "running"

        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/test-pipeline-123/start",
            method="POST",
            json=response,
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            pipeline = client.start_pipeline("test-pipeline-123")

        assert pipeline.status == PipelineStatus.RUNNING

    def test_stop_pipeline(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test stopping a pipeline."""
        response = api_responses["pipeline"].copy()
        response["status"] = "stopped"

        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/test-pipeline-123/stop",
            method="POST",
            json=response,
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            pipeline = client.stop_pipeline("test-pipeline-123")

        assert pipeline.status == PipelineStatus.STOPPED

    def test_get_pipeline_metrics(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test getting pipeline metrics."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/test-pipeline-123/metrics",
            json=api_responses["metrics"],
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            metrics = client.get_pipeline_metrics("test-pipeline-123")

        assert isinstance(metrics, PipelineMetrics)
        assert metrics.pipeline_id == "test-pipeline-123"
        assert metrics.events_processed == 10000
        assert metrics.lag_seconds == 5.2

    def test_list_sources(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test listing sources."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/sources",
            json=[api_responses["source"]],
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            sources = client.list_sources()

        assert len(sources) == 1
        assert isinstance(sources[0], Source)

    def test_authentication_error(self, httpx_mock: HTTPXMock):
        """Test authentication error handling."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines",
            status_code=401,
            json={"error": "unauthorized"},
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            with pytest.raises(PhilotesAuthenticationError):
                client.list_pipelines()

    def test_api_error(self, httpx_mock: HTTPXMock):
        """Test generic API error handling."""
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines",
            status_code=500,
            json={"error": "internal server error"},
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            with pytest.raises(PhilotesAPIError) as exc_info:
                client.list_pipelines()

        assert exc_info.value.status_code == 500


class TestPhilotesClientHelpers:
    """Tests for PhilotesClient helper methods."""

    def test_wait_for_lag_immediate(self, httpx_mock: HTTPXMock, api_responses: dict):
        """Test wait_for_lag when lag is already acceptable."""
        # Lag is 5.2s which is below threshold of 60s
        httpx_mock.add_response(
            url="http://localhost:8080/api/v1/pipelines/test-pipeline-123/metrics",
            json=api_responses["metrics"],
        )

        with PhilotesClient(api_url="http://localhost:8080") as client:
            metrics = client.wait_for_lag(
                pipeline_id="test-pipeline-123",
                max_lag_seconds=60.0,
                timeout=10.0,
            )

        assert metrics.lag_seconds == 5.2

    def test_metrics_is_caught_up(self, sample_metrics: PipelineMetrics):
        """Test is_caught_up property."""
        # Lag is 5.2s which is > 1s, so not caught up
        assert sample_metrics.is_caught_up is False

        # Create metrics with very low lag
        caught_up = PipelineMetrics(
            pipeline_id="test",
            status=PipelineStatus.RUNNING,
            lag_seconds=0.5,
        )
        assert caught_up.is_caught_up is True

    def test_metrics_is_within_lag(self, sample_metrics: PipelineMetrics):
        """Test is_within_lag method."""
        assert sample_metrics.is_within_lag(10.0) is True
        assert sample_metrics.is_within_lag(5.0) is False

    def test_metrics_to_dagster_metadata(self, sample_metrics: PipelineMetrics):
        """Test conversion to Dagster metadata."""
        metadata = sample_metrics.to_dagster_metadata()

        assert metadata["status"] == "running"
        assert metadata["events_processed"] == 10000
        assert metadata["lag_seconds"] == 5.2
