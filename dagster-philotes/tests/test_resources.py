"""Tests for PhilotesResource."""

import os
from unittest.mock import MagicMock, patch

import pytest

from dagster_philotes import PhilotesResource, PipelineStatus


class TestPhilotesResource:
    """Tests for PhilotesResource."""

    def test_init_with_explicit_values(self):
        """Test resource initialization with explicit values."""
        resource = PhilotesResource(
            api_url="http://custom:9000",
            api_key="custom-key",
            timeout=60.0,
        )
        assert resource.api_url == "http://custom:9000"
        assert resource.api_key == "custom-key"
        assert resource.timeout == 60.0

    def test_init_from_environment(self):
        """Test resource initialization from environment variables."""
        with patch.dict(os.environ, {
            "PHILOTES_API_URL": "http://env-url:8080",
            "PHILOTES_API_KEY": "env-key",
        }):
            resource = PhilotesResource()
            assert resource.api_url == "http://env-url:8080"
            assert resource.api_key == "env-key"

    def test_get_client(self):
        """Test getting the underlying client."""
        resource = PhilotesResource(
            api_url="http://localhost:8080",
            api_key="test-key",
        )
        client = resource.get_client()

        assert client.api_url == "http://localhost:8080"
        assert client.api_key == "test-key"

    def test_get_client_returns_same_instance(self):
        """Test that get_client returns the same instance."""
        resource = PhilotesResource(api_url="http://localhost:8080")
        client1 = resource.get_client()
        client2 = resource.get_client()

        assert client1 is client2

    def test_delegate_methods(self):
        """Test that resource methods delegate to client."""
        resource = PhilotesResource(api_url="http://localhost:8080")

        # Mock the client
        mock_client = MagicMock()
        resource._client = mock_client

        # Test delegation
        resource.list_pipelines()
        mock_client.list_pipelines.assert_called_once()

        resource.get_pipeline("test-id")
        mock_client.get_pipeline.assert_called_once_with("test-id")

        resource.start_pipeline("test-id")
        mock_client.start_pipeline.assert_called_once_with("test-id")

        resource.stop_pipeline("test-id")
        mock_client.stop_pipeline.assert_called_once_with("test-id")

        resource.get_pipeline_metrics("test-id")
        mock_client.get_pipeline_metrics.assert_called_once_with("test-id")

    def test_wait_for_lag_delegation(self):
        """Test wait_for_lag delegates with correct arguments."""
        resource = PhilotesResource(api_url="http://localhost:8080")

        mock_client = MagicMock()
        resource._client = mock_client

        resource.wait_for_lag(
            pipeline_id="test-id",
            max_lag_seconds=30.0,
            timeout=120.0,
            poll_interval=10.0,
        )

        mock_client.wait_for_lag.assert_called_once_with(
            pipeline_id="test-id",
            max_lag_seconds=30.0,
            timeout=120.0,
            poll_interval=10.0,
        )

    def test_wait_for_status_delegation(self):
        """Test wait_for_status delegates with correct arguments."""
        resource = PhilotesResource(api_url="http://localhost:8080")

        mock_client = MagicMock()
        resource._client = mock_client

        resource.wait_for_status(
            pipeline_id="test-id",
            target_status=PipelineStatus.RUNNING,
            timeout=60.0,
        )

        mock_client.wait_for_status.assert_called_once_with(
            pipeline_id="test-id",
            target_status=PipelineStatus.RUNNING,
            timeout=60.0,
            poll_interval=2.0,
        )
