"""Dagster resources for Philotes integration."""

import os
from typing import Any

from dagster import ConfigurableResource, InitResourceContext
from pydantic import Field

from dagster_philotes.client import PhilotesClient
from dagster_philotes.models import Pipeline, PipelineMetrics, PipelineStatus, Source


class PhilotesResource(ConfigurableResource):
    """Dagster resource for connecting to the Philotes CDC platform.

    This resource provides access to the Philotes API for managing CDC pipelines,
    monitoring replication lag, and coordinating downstream workflows.

    Example:
        ```python
        from dagster import Definitions, asset
        from dagster_philotes import PhilotesResource

        @asset
        def my_asset(philotes: PhilotesResource):
            # Get pipeline metrics
            metrics = philotes.get_pipeline_metrics("my-pipeline-id")
            return {"lag_seconds": metrics.lag_seconds}

        defs = Definitions(
            assets=[my_asset],
            resources={
                "philotes": PhilotesResource(
                    api_url="http://localhost:8080",
                    api_key="pk_xxx",
                ),
            },
        )
        ```

    Configuration:
        api_url: Base URL of the Philotes API. Can also be set via
            PHILOTES_API_URL environment variable.
        api_key: API key for authentication. Can also be set via
            PHILOTES_API_KEY environment variable.
        timeout: Request timeout in seconds (default: 30.0)
        max_retries: Maximum retry attempts for failed requests (default: 3)
    """

    api_url: str = Field(
        default_factory=lambda: os.environ.get("PHILOTES_API_URL", "http://localhost:8080"),
        description="Philotes API URL",
    )
    api_key: str = Field(
        default_factory=lambda: os.environ.get("PHILOTES_API_KEY", ""),
        description="API key for authentication",
    )
    timeout: float = Field(
        default=30.0,
        description="Request timeout in seconds",
    )
    max_retries: int = Field(
        default=3,
        description="Maximum retry attempts",
    )

    _client: PhilotesClient | None = None

    def setup_for_execution(self, context: InitResourceContext) -> None:
        """Initialize the client when resource is set up."""
        self._client = PhilotesClient(
            api_url=self.api_url,
            api_key=self.api_key,
            timeout=self.timeout,
            max_retries=self.max_retries,
        )

    def teardown_after_execution(self, context: InitResourceContext) -> None:
        """Clean up the client when resource is torn down."""
        if self._client is not None:
            self._client.close()
            self._client = None

    def get_client(self) -> PhilotesClient:
        """Get the underlying HTTP client.

        Returns:
            PhilotesClient instance

        Raises:
            RuntimeError: If resource is not initialized
        """
        if self._client is None:
            # Create a new client if not in execution context
            self._client = PhilotesClient(
                api_url=self.api_url,
                api_key=self.api_key,
                timeout=self.timeout,
                max_retries=self.max_retries,
            )
        return self._client

    # -------------------------------------------------------------------------
    # Health Check
    # -------------------------------------------------------------------------

    def health_check(self) -> bool:
        """Check if the Philotes API is healthy."""
        return self.get_client().health_check()

    # -------------------------------------------------------------------------
    # Pipeline Operations
    # -------------------------------------------------------------------------

    def list_pipelines(self) -> list[Pipeline]:
        """List all pipelines."""
        return self.get_client().list_pipelines()

    def get_pipeline(self, pipeline_id: str) -> Pipeline:
        """Get a pipeline by ID."""
        return self.get_client().get_pipeline(pipeline_id)

    def create_pipeline(
        self,
        name: str,
        source_id: str,
        config: dict[str, Any] | None = None,
    ) -> Pipeline:
        """Create a new pipeline."""
        return self.get_client().create_pipeline(name, source_id, config)

    def start_pipeline(self, pipeline_id: str) -> Pipeline:
        """Start a pipeline."""
        return self.get_client().start_pipeline(pipeline_id)

    def stop_pipeline(self, pipeline_id: str) -> Pipeline:
        """Stop a pipeline."""
        return self.get_client().stop_pipeline(pipeline_id)

    def get_pipeline_status(self, pipeline_id: str) -> PipelineStatus:
        """Get pipeline status."""
        return self.get_client().get_pipeline_status(pipeline_id)

    def get_pipeline_metrics(self, pipeline_id: str) -> PipelineMetrics:
        """Get current metrics for a pipeline."""
        return self.get_client().get_pipeline_metrics(pipeline_id)

    # -------------------------------------------------------------------------
    # Source Operations
    # -------------------------------------------------------------------------

    def list_sources(self) -> list[Source]:
        """List all sources."""
        return self.get_client().list_sources()

    def get_source(self, source_id: str) -> Source:
        """Get a source by ID."""
        return self.get_client().get_source(source_id)

    def test_source_connection(self, source_id: str) -> bool:
        """Test connection to a source."""
        return self.get_client().test_source_connection(source_id)

    # -------------------------------------------------------------------------
    # Helpers
    # -------------------------------------------------------------------------

    def wait_for_lag(
        self,
        pipeline_id: str,
        max_lag_seconds: float = 60.0,
        timeout: float = 300.0,
        poll_interval: float = 5.0,
    ) -> PipelineMetrics:
        """Wait for pipeline lag to drop below threshold."""
        return self.get_client().wait_for_lag(
            pipeline_id=pipeline_id,
            max_lag_seconds=max_lag_seconds,
            timeout=timeout,
            poll_interval=poll_interval,
        )

    def wait_for_status(
        self,
        pipeline_id: str,
        target_status: PipelineStatus,
        timeout: float = 60.0,
        poll_interval: float = 2.0,
    ) -> Pipeline:
        """Wait for pipeline to reach a specific status."""
        return self.get_client().wait_for_status(
            pipeline_id=pipeline_id,
            target_status=target_status,
            timeout=timeout,
            poll_interval=poll_interval,
        )
