"""Ops for controlling Philotes pipelines."""

from dagster import Config, In, Nothing, OpExecutionContext, Out, op

from dagster_philotes.models import Pipeline, PipelineMetrics, PipelineStatus
from dagster_philotes.resources import PhilotesResource


class PipelineIdConfig(Config):
    """Configuration for pipeline ID."""

    pipeline_id: str


class StartPipelineConfig(Config):
    """Configuration for starting a pipeline."""

    pipeline_id: str
    wait_for_running: bool = True
    wait_timeout: float = 60.0


class StopPipelineConfig(Config):
    """Configuration for stopping a pipeline."""

    pipeline_id: str
    wait_for_stopped: bool = True
    wait_timeout: float = 60.0


class WaitForLagConfig(Config):
    """Configuration for waiting for pipeline lag."""

    pipeline_id: str
    max_lag_seconds: float = 60.0
    timeout: float = 300.0
    poll_interval: float = 5.0


@op(
    out=Out(Pipeline, description="Started pipeline"),
)
def start_pipeline_op(
    context: OpExecutionContext,
    config: StartPipelineConfig,
    philotes: PhilotesResource,
) -> Pipeline:
    """Start a Philotes CDC pipeline.

    This op starts a pipeline and optionally waits for it to reach
    the running state.

    Example:
        ```python
        @job
        def my_job():
            pipeline = start_pipeline_op()
            wait_for_lag_op(pipeline)
        ```
    """
    context.log.info(f"Starting pipeline '{config.pipeline_id}'...")
    pipeline = philotes.start_pipeline(config.pipeline_id)

    if config.wait_for_running:
        context.log.info("Waiting for pipeline to reach running state...")
        pipeline = philotes.wait_for_status(
            pipeline_id=config.pipeline_id,
            target_status=PipelineStatus.RUNNING,
            timeout=config.wait_timeout,
        )
        context.log.info(f"Pipeline is now running (started at: {pipeline.started_at})")

    return pipeline


@op(
    out=Out(Pipeline, description="Stopped pipeline"),
)
def stop_pipeline_op(
    context: OpExecutionContext,
    config: StopPipelineConfig,
    philotes: PhilotesResource,
) -> Pipeline:
    """Stop a Philotes CDC pipeline.

    This op stops a pipeline and optionally waits for it to reach
    the stopped state.
    """
    context.log.info(f"Stopping pipeline '{config.pipeline_id}'...")
    pipeline = philotes.stop_pipeline(config.pipeline_id)

    if config.wait_for_stopped:
        context.log.info("Waiting for pipeline to reach stopped state...")
        pipeline = philotes.wait_for_status(
            pipeline_id=config.pipeline_id,
            target_status=PipelineStatus.STOPPED,
            timeout=config.wait_timeout,
        )
        context.log.info("Pipeline is now stopped")

    return pipeline


@op(
    ins={"upstream": In(Nothing, description="Optional upstream dependency")},
    out=Out(PipelineMetrics, description="Pipeline metrics after lag check"),
)
def wait_for_lag_op(
    context: OpExecutionContext,
    config: WaitForLagConfig,
    philotes: PhilotesResource,
) -> PipelineMetrics:
    """Wait for a pipeline's replication lag to drop below threshold.

    This op blocks until the pipeline's lag is acceptable, making it
    useful for ensuring data freshness before downstream processing.

    Example:
        ```python
        @job
        def etl_job():
            metrics = wait_for_lag_op()
            process_data(metrics)
        ```
    """
    context.log.info(
        f"Waiting for pipeline '{config.pipeline_id}' lag to drop below "
        f"{config.max_lag_seconds}s (timeout: {config.timeout}s)..."
    )

    metrics = philotes.wait_for_lag(
        pipeline_id=config.pipeline_id,
        max_lag_seconds=config.max_lag_seconds,
        timeout=config.timeout,
        poll_interval=config.poll_interval,
    )

    context.log.info(
        f"Pipeline caught up! Lag: {metrics.lag_seconds:.2f}s, "
        f"events processed: {metrics.events_processed}"
    )

    return metrics


@op(
    out=Out(PipelineMetrics, description="Current pipeline metrics"),
)
def get_pipeline_metrics_op(
    context: OpExecutionContext,
    config: PipelineIdConfig,
    philotes: PhilotesResource,
) -> PipelineMetrics:
    """Get current metrics for a pipeline.

    This op retrieves the current metrics without waiting, useful for
    monitoring or conditional logic.
    """
    metrics = philotes.get_pipeline_metrics(config.pipeline_id)

    context.log.info(
        f"Pipeline '{config.pipeline_id}' metrics: "
        f"status={metrics.status.value}, "
        f"lag={metrics.lag_seconds:.2f}s, "
        f"events={metrics.events_processed}"
    )

    return metrics


@op(
    out=Out(Pipeline, description="Pipeline details"),
)
def get_pipeline_op(
    context: OpExecutionContext,
    config: PipelineIdConfig,
    philotes: PhilotesResource,
) -> Pipeline:
    """Get pipeline details.

    This op retrieves the current pipeline configuration and status.
    """
    pipeline = philotes.get_pipeline(config.pipeline_id)

    context.log.info(
        f"Pipeline '{pipeline.name}' (id={pipeline.id}): "
        f"status={pipeline.status.value}, "
        f"tables={len(pipeline.tables)}"
    )

    return pipeline


@op(
    out=Out(bool, description="Health check result"),
)
def health_check_op(
    context: OpExecutionContext,
    philotes: PhilotesResource,
) -> bool:
    """Check if the Philotes API is healthy.

    Returns True if the API is reachable and healthy.
    """
    is_healthy = philotes.health_check()

    if is_healthy:
        context.log.info("Philotes API is healthy")
    else:
        context.log.warning("Philotes API health check failed")

    return is_healthy


@op(
    out=Out(list[Pipeline], description="List of all pipelines"),
)
def list_pipelines_op(
    context: OpExecutionContext,
    philotes: PhilotesResource,
) -> list[Pipeline]:
    """List all pipelines.

    Returns a list of all configured pipelines.
    """
    pipelines = philotes.list_pipelines()

    context.log.info(f"Found {len(pipelines)} pipelines")
    for p in pipelines:
        context.log.info(f"  - {p.name} ({p.id}): {p.status.value}")

    return pipelines
