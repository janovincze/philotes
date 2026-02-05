"""Main Dagster definitions for dagster-philotes examples.

This module provides the main Definitions object that Dagster loads.
It combines all example assets, jobs, and sensors.
"""

from dagster import (
    AssetExecutionContext,
    DefaultSensorStatus,
    Definitions,
    asset,
    define_asset_job,
)

from dagster_philotes import (
    PhilotesResource,
    get_pipeline_metrics_op,
    health_check_op,
    list_pipelines_op,
    philotes_pipeline_asset,
    pipeline_error_sensor,
    pipeline_lag_sensor,
    start_pipeline_op,
    stop_pipeline_op,
    wait_for_lag_op,
)

# ============================================================================
# CDC Pipeline Assets
# ============================================================================

# Example CDC pipeline assets - these represent Iceberg tables
# populated by Philotes CDC from PostgreSQL
orders_cdc = philotes_pipeline_asset(
    pipeline_id="orders-cdc",
    name="orders_iceberg",
    group_name="philotes_cdc",
    description="Orders table from e-commerce PostgreSQL database",
    max_lag_seconds=60.0,
)

customers_cdc = philotes_pipeline_asset(
    pipeline_id="customers-cdc",
    name="customers_iceberg",
    group_name="philotes_cdc",
    description="Customers table from e-commerce PostgreSQL database",
    max_lag_seconds=60.0,
)

products_cdc = philotes_pipeline_asset(
    pipeline_id="products-cdc",
    name="products_iceberg",
    group_name="philotes_cdc",
    description="Products table from e-commerce PostgreSQL database",
    max_lag_seconds=60.0,
)


# ============================================================================
# Downstream Assets
# ============================================================================


@asset(
    group_name="analytics",
    deps=[orders_cdc, customers_cdc, products_cdc],
)
def ecommerce_summary(
    context: AssetExecutionContext,
    philotes: PhilotesResource,
) -> dict[str, int]:
    """Summary metrics from all e-commerce CDC pipelines.

    This asset materializes after all CDC pipelines have caught up,
    providing a summary of the data ingestion status.
    """
    pipelines = ["orders-cdc", "customers-cdc", "products-cdc"]
    total_events = 0

    for pipeline_id in pipelines:
        try:
            metrics = philotes.get_pipeline_metrics(pipeline_id)
            total_events += metrics.events_processed
            context.log.info(
                f"Pipeline {pipeline_id}: {metrics.events_processed} events, "
                f"lag: {metrics.lag_seconds:.2f}s"
            )
        except Exception as e:
            context.log.warning(f"Could not get metrics for {pipeline_id}: {e}")

    return {"total_events_processed": total_events}


# ============================================================================
# Jobs
# ============================================================================

# Job to refresh all CDC assets
refresh_cdc_job = define_asset_job(
    name="refresh_cdc_job",
    selection=[orders_cdc, customers_cdc, products_cdc],
    description="Refresh all CDC pipeline assets (wait for lag to drop)",
)

# Full analytics pipeline
analytics_job = define_asset_job(
    name="analytics_job",
    selection="*",
    description="Run full analytics pipeline including CDC and summary",
)


# ============================================================================
# Sensors
# ============================================================================

# Create a placeholder job for sensors (normally you'd have a real job)
from dagster import job, op


@op
def notify_low_lag(context):
    """Placeholder op for lag notification."""
    context.log.info("CDC lag is low - ready for downstream processing!")


@op
def handle_pipeline_error(context):
    """Placeholder op for error handling."""
    context.log.error("Pipeline error detected - investigate!")


@job
def lag_notification_job():
    notify_low_lag()


@job
def error_handling_job():
    handle_pipeline_error()


# Sensor that triggers when orders CDC lag is low
orders_lag_sensor = pipeline_lag_sensor(
    pipeline_id="orders-cdc",
    job=lag_notification_job,
    lag_threshold_seconds=10.0,
    minimum_interval_seconds=60,
    default_status=DefaultSensorStatus.STOPPED,
)

# Sensor that triggers on pipeline errors
orders_error_sensor = pipeline_error_sensor(
    pipeline_id="orders-cdc",
    job=error_handling_job,
    minimum_interval_seconds=120,
    default_status=DefaultSensorStatus.STOPPED,
)


# ============================================================================
# Main Definitions
# ============================================================================

defs = Definitions(
    assets=[
        orders_cdc,
        customers_cdc,
        products_cdc,
        ecommerce_summary,
    ],
    jobs=[
        refresh_cdc_job,
        analytics_job,
        lag_notification_job,
        error_handling_job,
    ],
    sensors=[
        orders_lag_sensor,
        orders_error_sensor,
    ],
    resources={
        "philotes": PhilotesResource(
            # Default to local development settings
            # Override via PHILOTES_API_URL and PHILOTES_API_KEY env vars
            api_url="http://localhost:8080",
            api_key="",
            timeout=30.0,
        ),
    },
)
