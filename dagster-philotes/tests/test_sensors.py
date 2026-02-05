"""Tests for Philotes sensors."""

from unittest.mock import MagicMock

import pytest
from dagster import build_sensor_context, job, op

from dagster_philotes import (
    PhilotesResource,
    PipelineMetrics,
    PipelineStatus,
    pipeline_error_sensor,
    pipeline_lag_sensor,
    pipeline_sync_sensor,
)
from dagster_philotes.models import Pipeline


@op
def dummy_op():
    pass


@job
def dummy_job():
    dummy_op()


class TestPipelineLagSensor:
    """Tests for pipeline_lag_sensor."""

    def test_sensor_triggers_when_lag_below_threshold(self, sample_metrics: PipelineMetrics):
        """Test sensor triggers when lag is below threshold."""
        sensor = pipeline_lag_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
            lag_threshold_seconds=10.0,
        )

        # Mock the resource
        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.return_value = sample_metrics  # lag = 5.2s

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.run_requests is not None
        assert len(result.run_requests) == 1

    def test_sensor_skips_when_lag_above_threshold(
        self, sample_metrics_high_lag: PipelineMetrics
    ):
        """Test sensor skips when lag is above threshold."""
        sensor = pipeline_lag_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
            lag_threshold_seconds=10.0,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.return_value = sample_metrics_high_lag  # lag = 120.5s

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.skip_message is not None
        assert "above threshold" in result.skip_message

    def test_sensor_skips_when_pipeline_not_running(self):
        """Test sensor skips when pipeline is not running."""
        sensor = pipeline_lag_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
            lag_threshold_seconds=10.0,
        )

        stopped_metrics = PipelineMetrics(
            pipeline_id="test-pipeline",
            status=PipelineStatus.STOPPED,
            lag_seconds=0.0,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.return_value = stopped_metrics

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.skip_message is not None
        assert "not running" in result.skip_message

    def test_sensor_handles_api_error(self):
        """Test sensor handles API errors gracefully."""
        sensor = pipeline_lag_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
            lag_threshold_seconds=10.0,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.side_effect = Exception("API error")

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.skip_message is not None
        assert "Failed to get pipeline metrics" in result.skip_message


class TestPipelineSyncSensor:
    """Tests for pipeline_sync_sensor."""

    def test_sensor_triggers_when_caught_up(self):
        """Test sensor triggers when pipeline is caught up."""
        sensor = pipeline_sync_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
        )

        caught_up_metrics = PipelineMetrics(
            pipeline_id="test-pipeline",
            status=PipelineStatus.RUNNING,
            lag_seconds=0.5,  # < 1 second = caught up
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.return_value = caught_up_metrics

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.run_requests is not None
        assert len(result.run_requests) == 1

    def test_sensor_skips_when_not_caught_up(self, sample_metrics: PipelineMetrics):
        """Test sensor skips when pipeline is not caught up."""
        sensor = pipeline_sync_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline_metrics.return_value = sample_metrics  # lag = 5.2s

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.skip_message is not None
        assert "not yet caught up" in result.skip_message


class TestPipelineErrorSensor:
    """Tests for pipeline_error_sensor."""

    def test_sensor_triggers_on_error_status(self):
        """Test sensor triggers when pipeline is in error state."""
        sensor = pipeline_error_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
        )

        error_pipeline = Pipeline(
            id="test-pipeline",
            name="test",
            source_id="source-1",
            status=PipelineStatus.ERROR,
            error_message="Connection lost",
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline.return_value = error_pipeline

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.run_requests is not None
        assert len(result.run_requests) == 1
        assert result.run_requests[0].tags["philotes_error_type"] == "pipeline_error"

    def test_sensor_skips_when_healthy(self, sample_pipeline: Pipeline):
        """Test sensor skips when pipeline is healthy."""
        sensor = pipeline_error_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline.return_value = sample_pipeline  # RUNNING status

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.skip_message is not None
        assert "healthy" in result.skip_message

    def test_sensor_triggers_on_api_error(self):
        """Test sensor triggers when API is unreachable."""
        sensor = pipeline_error_sensor(
            pipeline_id="test-pipeline",
            job=dummy_job,
        )

        mock_resource = MagicMock(spec=PhilotesResource)
        mock_resource.get_pipeline.side_effect = Exception("Connection refused")

        context = build_sensor_context(resources={"philotes": mock_resource})

        result = sensor.evaluate_tick(context)

        assert result.run_requests is not None
        assert result.run_requests[0].tags["philotes_error_type"] == "api_error"
