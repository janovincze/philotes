"""Custom exceptions for dagster-philotes."""


class PhilotesError(Exception):
    """Base exception for Philotes errors."""

    pass


class PhilotesAPIError(PhilotesError):
    """Error communicating with Philotes API."""

    def __init__(
        self,
        message: str,
        status_code: int | None = None,
        response_body: str | None = None,
    ):
        super().__init__(message)
        self.status_code = status_code
        self.response_body = response_body

    def __str__(self) -> str:
        msg = super().__str__()
        if self.status_code:
            msg = f"[{self.status_code}] {msg}"
        return msg


class PhilotesAuthenticationError(PhilotesAPIError):
    """Authentication failed with Philotes API."""

    def __init__(self, message: str = "Authentication failed"):
        super().__init__(message, status_code=401)


class PhilotesNotFoundError(PhilotesAPIError):
    """Resource not found in Philotes API."""

    def __init__(self, resource_type: str, resource_id: str):
        super().__init__(
            f"{resource_type} not found: {resource_id}",
            status_code=404,
        )
        self.resource_type = resource_type
        self.resource_id = resource_id


class PipelineNotFoundError(PhilotesNotFoundError):
    """Pipeline not found."""

    def __init__(self, pipeline_id: str):
        super().__init__("Pipeline", pipeline_id)


class SourceNotFoundError(PhilotesNotFoundError):
    """Source not found."""

    def __init__(self, source_id: str):
        super().__init__("Source", source_id)


class PhilotesTimeoutError(PhilotesError):
    """Timeout waiting for Philotes operation."""

    def __init__(self, operation: str, timeout_seconds: float):
        super().__init__(f"Timeout after {timeout_seconds}s waiting for: {operation}")
        self.operation = operation
        self.timeout_seconds = timeout_seconds


class PipelineLagTimeoutError(PhilotesTimeoutError):
    """Timeout waiting for pipeline lag to drop."""

    def __init__(
        self,
        pipeline_id: str,
        max_lag_seconds: float,
        current_lag_seconds: float,
        timeout_seconds: float,
    ):
        super().__init__(
            f"pipeline {pipeline_id} lag to drop below {max_lag_seconds}s "
            f"(current: {current_lag_seconds}s)",
            timeout_seconds,
        )
        self.pipeline_id = pipeline_id
        self.max_lag_seconds = max_lag_seconds
        self.current_lag_seconds = current_lag_seconds


class PipelineNotRunningError(PhilotesError):
    """Pipeline is not in running state."""

    def __init__(self, pipeline_id: str, current_status: str):
        super().__init__(
            f"Pipeline {pipeline_id} is not running (status: {current_status})"
        )
        self.pipeline_id = pipeline_id
        self.current_status = current_status
