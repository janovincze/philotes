"""Sensors for monitoring Philotes pipelines."""

from dagster import (
    DefaultSensorStatus,
    JobDefinition,
    RunRequest,
    SensorDefinition,
    SensorEvaluationContext,
    SkipReason,
    sensor,
)

from dagster_philotes.models import PipelineStatus
from dagster_philotes.resources import PhilotesResource


def pipeline_lag_sensor(
    pipeline_id: str,
    job: JobDefinition,
    lag_threshold_seconds: float = 10.0,
    minimum_interval_seconds: int = 30,
    name: str | None = None,
    default_status: DefaultSensorStatus = DefaultSensorStatus.STOPPED,
) -> SensorDefinition:
    """Create a sensor that triggers a job when pipeline lag drops below threshold.

    This sensor monitors a Philotes pipeline and triggers the specified job
    when the replication lag drops below the configured threshold. This is
    useful for triggering downstream processing (like dbt runs) after CDC
    has caught up.

    Args:
        pipeline_id: The Philotes pipeline ID to monitor
        job: The Dagster job to trigger when lag is low
        lag_threshold_seconds: Trigger when lag drops below this value (default: 10.0)
        minimum_interval_seconds: Minimum time between sensor evaluations (default: 30)
        name: Sensor name (defaults to "{pipeline_id}_lag_sensor")
        default_status: Initial sensor status (default: STOPPED)

    Returns:
        A Dagster SensorDefinition

    Example:
        ```python
        from dagster import job, op
        from dagster_philotes import pipeline_lag_sensor

        @op
        def run_dbt():
            # Run dbt models
            pass

        @job
        def dbt_job():
            run_dbt()

        # Trigger dbt when CDC lag < 10 seconds
        orders_lag_sensor = pipeline_lag_sensor(
            pipeline_id="orders-cdc",
            job=dbt_job,
            lag_threshold_seconds=10.0,
        )
        ```
    """
    sensor_name = name or f"{pipeline_id.replace('-', '_')}_lag_sensor"

    @sensor(
        name=sensor_name,
        job=job,
        minimum_interval_seconds=minimum_interval_seconds,
        default_status=default_status,
    )
    def _lag_sensor(
        context: SensorEvaluationContext,
        philotes: PhilotesResource,
    ) -> RunRequest | SkipReason:
        try:
            metrics = philotes.get_pipeline_metrics(pipeline_id)
        except Exception as e:
            return SkipReason(f"Failed to get pipeline metrics: {e}")

        # Check if pipeline is running
        if metrics.status != PipelineStatus.RUNNING:
            return SkipReason(
                f"Pipeline is not running (status: {metrics.status.value})"
            )

        # Check lag threshold
        if metrics.lag_seconds <= lag_threshold_seconds:
            context.log.info(
                f"Pipeline '{pipeline_id}' lag ({metrics.lag_seconds:.2f}s) "
                f"is below threshold ({lag_threshold_seconds}s), triggering job"
            )
            return RunRequest(
                run_key=f"{pipeline_id}_{metrics.events_processed}",
                run_config={},
                tags={
                    "philotes_pipeline_id": pipeline_id,
                    "philotes_lag_seconds": str(metrics.lag_seconds),
                    "philotes_events_processed": str(metrics.events_processed),
                },
            )

        return SkipReason(
            f"Pipeline lag ({metrics.lag_seconds:.2f}s) is above threshold "
            f"({lag_threshold_seconds}s)"
        )

    return _lag_sensor


def pipeline_sync_sensor(
    pipeline_id: str,
    job: JobDefinition,
    minimum_interval_seconds: int = 60,
    name: str | None = None,
    default_status: DefaultSensorStatus = DefaultSensorStatus.STOPPED,
) -> SensorDefinition:
    """Create a sensor that triggers when pipeline is fully caught up.

    This sensor triggers when the pipeline's replication lag drops below
    1 second, indicating the pipeline has fully caught up with the source.

    Args:
        pipeline_id: The Philotes pipeline ID to monitor
        job: The Dagster job to trigger when caught up
        minimum_interval_seconds: Minimum time between sensor evaluations (default: 60)
        name: Sensor name (defaults to "{pipeline_id}_sync_sensor")
        default_status: Initial sensor status (default: STOPPED)

    Returns:
        A Dagster SensorDefinition
    """
    sensor_name = name or f"{pipeline_id.replace('-', '_')}_sync_sensor"

    @sensor(
        name=sensor_name,
        job=job,
        minimum_interval_seconds=minimum_interval_seconds,
        default_status=default_status,
    )
    def _sync_sensor(
        context: SensorEvaluationContext,
        philotes: PhilotesResource,
    ) -> RunRequest | SkipReason:
        try:
            metrics = philotes.get_pipeline_metrics(pipeline_id)
        except Exception as e:
            return SkipReason(f"Failed to get pipeline metrics: {e}")

        if metrics.status != PipelineStatus.RUNNING:
            return SkipReason(
                f"Pipeline is not running (status: {metrics.status.value})"
            )

        if metrics.is_caught_up:
            context.log.info(
                f"Pipeline '{pipeline_id}' is caught up "
                f"(lag: {metrics.lag_seconds:.2f}s), triggering job"
            )
            return RunRequest(
                run_key=f"{pipeline_id}_sync_{metrics.events_processed}",
                run_config={},
                tags={
                    "philotes_pipeline_id": pipeline_id,
                    "philotes_synced": "true",
                },
            )

        return SkipReason(
            f"Pipeline is not yet caught up (lag: {metrics.lag_seconds:.2f}s)"
        )

    return _sync_sensor


def pipeline_error_sensor(
    pipeline_id: str,
    job: JobDefinition,
    minimum_interval_seconds: int = 60,
    name: str | None = None,
    default_status: DefaultSensorStatus = DefaultSensorStatus.STOPPED,
) -> SensorDefinition:
    """Create a sensor that triggers when pipeline encounters an error.

    This sensor monitors for pipeline errors and can trigger alerting or
    remediation workflows when issues are detected.

    Args:
        pipeline_id: The Philotes pipeline ID to monitor
        job: The Dagster job to trigger on error
        minimum_interval_seconds: Minimum time between sensor evaluations (default: 60)
        name: Sensor name (defaults to "{pipeline_id}_error_sensor")
        default_status: Initial sensor status (default: STOPPED)

    Returns:
        A Dagster SensorDefinition
    """
    sensor_name = name or f"{pipeline_id.replace('-', '_')}_error_sensor"

    @sensor(
        name=sensor_name,
        job=job,
        minimum_interval_seconds=minimum_interval_seconds,
        default_status=default_status,
    )
    def _error_sensor(
        context: SensorEvaluationContext,
        philotes: PhilotesResource,
    ) -> RunRequest | SkipReason:
        try:
            pipeline = philotes.get_pipeline(pipeline_id)
        except Exception as e:
            # API error might indicate infrastructure issue
            context.log.warning(f"Failed to get pipeline: {e}")
            return RunRequest(
                run_key=f"{pipeline_id}_api_error",
                run_config={},
                tags={
                    "philotes_pipeline_id": pipeline_id,
                    "philotes_error_type": "api_error",
                    "philotes_error_message": str(e),
                },
            )

        if pipeline.status == PipelineStatus.ERROR:
            context.log.warning(
                f"Pipeline '{pipeline_id}' is in error state: {pipeline.error_message}"
            )
            return RunRequest(
                run_key=f"{pipeline_id}_error_{pipeline.updated_at}",
                run_config={},
                tags={
                    "philotes_pipeline_id": pipeline_id,
                    "philotes_error_type": "pipeline_error",
                    "philotes_error_message": pipeline.error_message or "Unknown error",
                },
            )

        return SkipReason(f"Pipeline is healthy (status: {pipeline.status.value})")

    return _error_sensor


def pipeline_health_sensor(
    pipeline_ids: list[str],
    job: JobDefinition,
    lag_warning_threshold: float = 300.0,
    minimum_interval_seconds: int = 120,
    name: str = "philotes_health_sensor",
    default_status: DefaultSensorStatus = DefaultSensorStatus.STOPPED,
) -> SensorDefinition:
    """Create a sensor that monitors health of multiple pipelines.

    This sensor checks all specified pipelines and triggers when any of them
    are unhealthy (in error state or lag above warning threshold).

    Args:
        pipeline_ids: List of pipeline IDs to monitor
        job: The Dagster job to trigger on health issues
        lag_warning_threshold: Warn when lag exceeds this value (default: 300.0)
        minimum_interval_seconds: Minimum time between evaluations (default: 120)
        name: Sensor name
        default_status: Initial sensor status

    Returns:
        A Dagster SensorDefinition
    """

    @sensor(
        name=name,
        job=job,
        minimum_interval_seconds=minimum_interval_seconds,
        default_status=default_status,
    )
    def _health_sensor(
        context: SensorEvaluationContext,
        philotes: PhilotesResource,
    ) -> RunRequest | SkipReason:
        unhealthy_pipelines: list[dict[str, str]] = []

        for pipeline_id in pipeline_ids:
            try:
                pipeline = philotes.get_pipeline(pipeline_id)
                metrics = philotes.get_pipeline_metrics(pipeline_id)

                if pipeline.status == PipelineStatus.ERROR:
                    unhealthy_pipelines.append({
                        "pipeline_id": pipeline_id,
                        "issue": "error",
                        "message": pipeline.error_message or "Unknown error",
                    })
                elif metrics.lag_seconds > lag_warning_threshold:
                    unhealthy_pipelines.append({
                        "pipeline_id": pipeline_id,
                        "issue": "high_lag",
                        "message": f"Lag: {metrics.lag_seconds:.2f}s",
                    })
            except Exception as e:
                unhealthy_pipelines.append({
                    "pipeline_id": pipeline_id,
                    "issue": "unreachable",
                    "message": str(e),
                })

        if unhealthy_pipelines:
            context.log.warning(f"Found {len(unhealthy_pipelines)} unhealthy pipelines")
            return RunRequest(
                run_key=f"health_check_{len(unhealthy_pipelines)}",
                run_config={
                    "ops": {
                        "handle_health_issues": {
                            "config": {
                                "unhealthy_pipelines": unhealthy_pipelines,
                            }
                        }
                    }
                },
                tags={
                    "philotes_health_check": "failed",
                    "philotes_unhealthy_count": str(len(unhealthy_pipelines)),
                },
            )

        return SkipReason(f"All {len(pipeline_ids)} pipelines are healthy")

    return _health_sensor
