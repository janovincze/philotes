"""Dagster integration for Philotes CDC platform.

This package provides Dagster resources, assets, sensors, and ops for
orchestrating Philotes CDC pipelines within broader data workflows.

Quick Start:
    ```python
    from dagster import Definitions
    from dagster_philotes import (
        PhilotesResource,
        philotes_pipeline_asset,
        pipeline_lag_sensor,
    )

    # Create asset for CDC pipeline
    orders_cdc = philotes_pipeline_asset(
        pipeline_id="orders-cdc",
        max_lag_seconds=60.0,
    )

    defs = Definitions(
        assets=[orders_cdc],
        resources={
            "philotes": PhilotesResource(
                api_url="http://localhost:8080",
                api_key="your-api-key",
            ),
        },
    )
    ```
"""

from dagster_philotes.assets import (
    multi_pipeline_assets,
    philotes_pipeline_asset,
    philotes_table_asset,
)
from dagster_philotes.client import PhilotesClient
from dagster_philotes.exceptions import (
    PhilotesAPIError,
    PhilotesAuthenticationError,
    PhilotesError,
    PhilotesNotFoundError,
    PhilotesTimeoutError,
    PipelineLagTimeoutError,
    PipelineNotFoundError,
    PipelineNotRunningError,
    SourceNotFoundError,
)
from dagster_philotes.models import (
    Pipeline,
    PipelineConfig,
    PipelineMetrics,
    PipelineStatus,
    Source,
    SourceStatus,
    TableMapping,
    TableMetrics,
)
from dagster_philotes.ops import (
    get_pipeline_metrics_op,
    get_pipeline_op,
    health_check_op,
    list_pipelines_op,
    start_pipeline_op,
    stop_pipeline_op,
    wait_for_lag_op,
)
from dagster_philotes.resources import PhilotesResource
from dagster_philotes.sensors import (
    pipeline_error_sensor,
    pipeline_health_sensor,
    pipeline_lag_sensor,
    pipeline_sync_sensor,
)

__version__ = "0.1.0"

__all__ = [
    # Version
    "__version__",
    # Client
    "PhilotesClient",
    # Resource
    "PhilotesResource",
    # Assets
    "philotes_pipeline_asset",
    "philotes_table_asset",
    "multi_pipeline_assets",
    # Sensors
    "pipeline_lag_sensor",
    "pipeline_sync_sensor",
    "pipeline_error_sensor",
    "pipeline_health_sensor",
    # Ops
    "start_pipeline_op",
    "stop_pipeline_op",
    "wait_for_lag_op",
    "get_pipeline_metrics_op",
    "get_pipeline_op",
    "health_check_op",
    "list_pipelines_op",
    # Models
    "Pipeline",
    "PipelineConfig",
    "PipelineMetrics",
    "PipelineStatus",
    "Source",
    "SourceStatus",
    "TableMapping",
    "TableMetrics",
    # Exceptions
    "PhilotesError",
    "PhilotesAPIError",
    "PhilotesAuthenticationError",
    "PhilotesNotFoundError",
    "PhilotesTimeoutError",
    "PipelineNotFoundError",
    "SourceNotFoundError",
    "PipelineLagTimeoutError",
    "PipelineNotRunningError",
]
