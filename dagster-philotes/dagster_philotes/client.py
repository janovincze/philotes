"""HTTP client for Philotes API."""

import time
from typing import Any

import httpx

from dagster_philotes.exceptions import (
    PipelineLagTimeoutError,
    PipelineNotFoundError,
    PipelineNotRunningError,
    PhilotesAPIError,
    PhilotesAuthenticationError,
    SourceNotFoundError,
)
from dagster_philotes.models import (
    Pipeline,
    PipelineMetrics,
    PipelineStatus,
    Source,
)


class PhilotesClient:
    """HTTP client for Philotes API.

    This client provides methods to interact with the Philotes CDC platform API,
    including pipeline management, source configuration, and metrics retrieval.

    Example:
        ```python
        client = PhilotesClient(
            api_url="http://localhost:8080",
            api_key="pk_xxx",
        )

        # List all pipelines
        pipelines = client.list_pipelines()

        # Get metrics for a pipeline
        metrics = client.get_pipeline_metrics("my-pipeline-id")

        # Wait for lag to drop
        metrics = client.wait_for_lag("my-pipeline-id", max_lag_seconds=60.0)
        ```
    """

    def __init__(
        self,
        api_url: str,
        api_key: str = "",
        timeout: float = 30.0,
        max_retries: int = 3,
        retry_delay: float = 1.0,
    ):
        """Initialize the Philotes client.

        Args:
            api_url: Base URL of the Philotes API (e.g., "http://localhost:8080")
            api_key: API key for authentication (optional if auth is disabled)
            timeout: Request timeout in seconds
            max_retries: Maximum number of retries for failed requests
            retry_delay: Delay between retries in seconds
        """
        self.api_url = api_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_delay = retry_delay

        # Build headers
        headers: dict[str, str] = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"

        self._client = httpx.Client(
            base_url=self.api_url,
            headers=headers,
            timeout=timeout,
        )

    def close(self) -> None:
        """Close the HTTP client."""
        self._client.close()

    def __enter__(self) -> "PhilotesClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    def _request(
        self,
        method: str,
        path: str,
        **kwargs: Any,
    ) -> httpx.Response:
        """Make an HTTP request with retry logic.

        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            path: API path (e.g., "/api/v1/pipelines")
            **kwargs: Additional arguments to pass to httpx

        Returns:
            HTTP response

        Raises:
            PhilotesAPIError: If the request fails after retries
            PhilotesAuthenticationError: If authentication fails
        """
        last_error: Exception | None = None

        for attempt in range(self.max_retries):
            try:
                response = self._client.request(method, path, **kwargs)

                # Handle specific error codes
                if response.status_code == 401:
                    raise PhilotesAuthenticationError()
                if response.status_code == 404:
                    # Let caller handle 404s
                    return response
                if response.status_code >= 400:
                    raise PhilotesAPIError(
                        f"API request failed: {response.text}",
                        status_code=response.status_code,
                        response_body=response.text,
                    )

                return response

            except httpx.TimeoutException as e:
                last_error = e
                if attempt < self.max_retries - 1:
                    time.sleep(self.retry_delay * (attempt + 1))
            except httpx.RequestError as e:
                last_error = e
                if attempt < self.max_retries - 1:
                    time.sleep(self.retry_delay * (attempt + 1))

        raise PhilotesAPIError(f"Request failed after {self.max_retries} retries: {last_error}")

    # -------------------------------------------------------------------------
    # Health Check
    # -------------------------------------------------------------------------

    def health_check(self) -> bool:
        """Check if the Philotes API is healthy.

        Returns:
            True if API is healthy, False otherwise
        """
        try:
            response = self._request("GET", "/health")
            return response.status_code == 200
        except PhilotesAPIError:
            return False

    # -------------------------------------------------------------------------
    # Pipeline Operations
    # -------------------------------------------------------------------------

    def list_pipelines(self) -> list[Pipeline]:
        """List all pipelines.

        Returns:
            List of pipelines
        """
        response = self._request("GET", "/api/v1/pipelines")
        data = response.json()

        # Handle both list and wrapped response formats
        pipelines_data = data if isinstance(data, list) else data.get("pipelines", [])
        return [Pipeline.model_validate(p) for p in pipelines_data]

    def get_pipeline(self, pipeline_id: str) -> Pipeline:
        """Get a pipeline by ID.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            Pipeline object

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        response = self._request("GET", f"/api/v1/pipelines/{pipeline_id}")

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

        return Pipeline.model_validate(response.json())

    def create_pipeline(
        self,
        name: str,
        source_id: str,
        config: dict[str, Any] | None = None,
    ) -> Pipeline:
        """Create a new pipeline.

        Args:
            name: Pipeline name
            source_id: Source database ID
            config: Optional pipeline configuration

        Returns:
            Created pipeline
        """
        payload = {
            "name": name,
            "source_id": source_id,
        }
        if config:
            payload["config"] = config

        response = self._request("POST", "/api/v1/pipelines", json=payload)
        return Pipeline.model_validate(response.json())

    def update_pipeline(
        self,
        pipeline_id: str,
        name: str | None = None,
        config: dict[str, Any] | None = None,
    ) -> Pipeline:
        """Update a pipeline.

        Args:
            pipeline_id: Pipeline ID
            name: New name (optional)
            config: New configuration (optional)

        Returns:
            Updated pipeline
        """
        payload: dict[str, Any] = {}
        if name is not None:
            payload["name"] = name
        if config is not None:
            payload["config"] = config

        response = self._request("PUT", f"/api/v1/pipelines/{pipeline_id}", json=payload)

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

        return Pipeline.model_validate(response.json())

    def delete_pipeline(self, pipeline_id: str) -> None:
        """Delete a pipeline.

        Args:
            pipeline_id: Pipeline ID

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        response = self._request("DELETE", f"/api/v1/pipelines/{pipeline_id}")

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

    def start_pipeline(self, pipeline_id: str) -> Pipeline:
        """Start a pipeline.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            Updated pipeline

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        response = self._request("POST", f"/api/v1/pipelines/{pipeline_id}/start")

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

        return Pipeline.model_validate(response.json())

    def stop_pipeline(self, pipeline_id: str) -> Pipeline:
        """Stop a pipeline.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            Updated pipeline

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        response = self._request("POST", f"/api/v1/pipelines/{pipeline_id}/stop")

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

        return Pipeline.model_validate(response.json())

    def get_pipeline_status(self, pipeline_id: str) -> PipelineStatus:
        """Get pipeline status.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            Current pipeline status

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        pipeline = self.get_pipeline(pipeline_id)
        return pipeline.status

    def get_pipeline_metrics(self, pipeline_id: str) -> PipelineMetrics:
        """Get current metrics for a pipeline.

        Args:
            pipeline_id: Pipeline ID

        Returns:
            Pipeline metrics

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
        """
        response = self._request("GET", f"/api/v1/pipelines/{pipeline_id}/metrics")

        if response.status_code == 404:
            raise PipelineNotFoundError(pipeline_id)

        return PipelineMetrics.model_validate(response.json())

    # -------------------------------------------------------------------------
    # Source Operations
    # -------------------------------------------------------------------------

    def list_sources(self) -> list[Source]:
        """List all sources.

        Returns:
            List of sources
        """
        response = self._request("GET", "/api/v1/sources")
        data = response.json()

        sources_data = data if isinstance(data, list) else data.get("sources", [])
        return [Source.model_validate(s) for s in sources_data]

    def get_source(self, source_id: str) -> Source:
        """Get a source by ID.

        Args:
            source_id: Source ID

        Returns:
            Source object

        Raises:
            SourceNotFoundError: If source doesn't exist
        """
        response = self._request("GET", f"/api/v1/sources/{source_id}")

        if response.status_code == 404:
            raise SourceNotFoundError(source_id)

        return Source.model_validate(response.json())

    def test_source_connection(self, source_id: str) -> bool:
        """Test connection to a source.

        Args:
            source_id: Source ID

        Returns:
            True if connection successful

        Raises:
            SourceNotFoundError: If source doesn't exist
        """
        response = self._request("POST", f"/api/v1/sources/{source_id}/test")

        if response.status_code == 404:
            raise SourceNotFoundError(source_id)

        return response.status_code == 200

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
        """Wait for pipeline lag to drop below threshold.

        Args:
            pipeline_id: Pipeline ID
            max_lag_seconds: Maximum acceptable lag in seconds
            timeout: Maximum time to wait in seconds
            poll_interval: Time between checks in seconds

        Returns:
            Pipeline metrics when lag is acceptable

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
            PipelineNotRunningError: If pipeline is not running
            PipelineLagTimeoutError: If timeout is reached
        """
        start_time = time.time()

        while True:
            elapsed = time.time() - start_time
            if elapsed >= timeout:
                # Get final metrics for error message
                metrics = self.get_pipeline_metrics(pipeline_id)
                raise PipelineLagTimeoutError(
                    pipeline_id=pipeline_id,
                    max_lag_seconds=max_lag_seconds,
                    current_lag_seconds=metrics.lag_seconds,
                    timeout_seconds=timeout,
                )

            metrics = self.get_pipeline_metrics(pipeline_id)

            # Check if pipeline is running
            if metrics.status != PipelineStatus.RUNNING:
                raise PipelineNotRunningError(pipeline_id, metrics.status.value)

            # Check if lag is acceptable
            if metrics.is_within_lag(max_lag_seconds):
                return metrics

            time.sleep(poll_interval)

    def wait_for_status(
        self,
        pipeline_id: str,
        target_status: PipelineStatus,
        timeout: float = 60.0,
        poll_interval: float = 2.0,
    ) -> Pipeline:
        """Wait for pipeline to reach a specific status.

        Args:
            pipeline_id: Pipeline ID
            target_status: Status to wait for
            timeout: Maximum time to wait in seconds
            poll_interval: Time between checks in seconds

        Returns:
            Pipeline when target status is reached

        Raises:
            PipelineNotFoundError: If pipeline doesn't exist
            PhilotesTimeoutError: If timeout is reached
        """
        from dagster_philotes.exceptions import PhilotesTimeoutError

        start_time = time.time()

        while True:
            elapsed = time.time() - start_time
            if elapsed >= timeout:
                raise PhilotesTimeoutError(
                    f"pipeline {pipeline_id} to reach status {target_status.value}",
                    timeout,
                )

            pipeline = self.get_pipeline(pipeline_id)
            if pipeline.status == target_status:
                return pipeline

            time.sleep(poll_interval)
