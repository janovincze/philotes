"""CDC to dbt workflow example.

This example shows how to trigger dbt models after CDC pipelines
have caught up, ensuring dbt works with fresh data.
"""

from dagster import (
    AssetExecutionContext,
    DefaultSensorStatus,
    Definitions,
    RunRequest,
    SensorEvaluationContext,
    asset,
    define_asset_job,
    job,
    op,
    sensor,
)

from dagster_philotes import (
    PhilotesResource,
    PipelineStatus,
    philotes_pipeline_asset,
    pipeline_lag_sensor,
)

# ============================================================================
# CDC Pipeline Assets
# ============================================================================

orders_cdc = philotes_pipeline_asset(
    pipeline_id="orders-cdc",
    name="orders_iceberg",
    group_name="cdc",
    max_lag_seconds=30.0,
)

products_cdc = philotes_pipeline_asset(
    pipeline_id="products-cdc",
    name="products_iceberg",
    group_name="cdc",
    max_lag_seconds=30.0,
)


# ============================================================================
# dbt Model Assets
# ============================================================================


@asset(
    group_name="dbt_models",
    deps=[orders_cdc],
)
def dim_orders(context: AssetExecutionContext) -> None:
    """dbt model: dim_orders

    This would typically call dbt to run the dim_orders model.
    The dependency on orders_cdc ensures CDC has caught up first.
    """
    context.log.info("Running dbt model: dim_orders")
    # In practice, you would call dbt here:
    # subprocess.run(["dbt", "run", "--select", "dim_orders"])


@asset(
    group_name="dbt_models",
    deps=[products_cdc],
)
def dim_products(context: AssetExecutionContext) -> None:
    """dbt model: dim_products"""
    context.log.info("Running dbt model: dim_products")


@asset(
    group_name="dbt_models",
    deps=[dim_orders, dim_products],
)
def fact_order_items(context: AssetExecutionContext) -> None:
    """dbt model: fact_order_items

    Depends on both dim_orders and dim_products.
    """
    context.log.info("Running dbt model: fact_order_items")


@asset(
    group_name="analytics",
    deps=[fact_order_items],
)
def sales_dashboard_data(context: AssetExecutionContext) -> dict[str, str]:
    """Prepare data for sales dashboard.

    This runs after all dbt models are complete.
    """
    context.log.info("Preparing sales dashboard data")
    return {"status": "ready", "table": "fact_order_items"}


# ============================================================================
# Jobs
# ============================================================================

# Job that runs all dbt models
dbt_models_job = define_asset_job(
    name="dbt_models_job",
    selection=[dim_orders, dim_products, fact_order_items],
)

# Full pipeline job
full_pipeline_job = define_asset_job(
    name="full_pipeline_job",
    selection="*",  # All assets
)


# ============================================================================
# Sensors
# ============================================================================


@sensor(
    job=dbt_models_job,
    minimum_interval_seconds=60,
    default_status=DefaultSensorStatus.STOPPED,
)
def cdc_dbt_sensor(
    context: SensorEvaluationContext,
    philotes: PhilotesResource,
) -> RunRequest | None:
    """Trigger dbt models when all CDC pipelines have caught up.

    This sensor checks multiple pipelines and only triggers when
    all of them have low lag.
    """
    pipelines_to_check = ["orders-cdc", "products-cdc"]
    max_lag = 30.0

    for pipeline_id in pipelines_to_check:
        try:
            metrics = philotes.get_pipeline_metrics(pipeline_id)

            if metrics.status != PipelineStatus.RUNNING:
                context.log.info(f"Pipeline {pipeline_id} is not running")
                return None

            if metrics.lag_seconds > max_lag:
                context.log.info(
                    f"Pipeline {pipeline_id} lag ({metrics.lag_seconds}s) "
                    f"is above threshold ({max_lag}s)"
                )
                return None

        except Exception as e:
            context.log.warning(f"Failed to check pipeline {pipeline_id}: {e}")
            return None

    # All pipelines are caught up!
    context.log.info("All CDC pipelines caught up, triggering dbt models")
    return RunRequest(
        run_key=f"cdc_dbt_{context.cursor or 'initial'}",
        tags={"trigger": "cdc_caught_up"},
    )


# ============================================================================
# Alternative: Use the built-in sensor factory
# ============================================================================

# This creates a simpler sensor for a single pipeline
orders_dbt_sensor = pipeline_lag_sensor(
    pipeline_id="orders-cdc",
    job=dbt_models_job,
    lag_threshold_seconds=10.0,
    name="orders_dbt_sensor",
)


# ============================================================================
# Definitions
# ============================================================================

dbt_defs = Definitions(
    assets=[
        orders_cdc,
        products_cdc,
        dim_orders,
        dim_products,
        fact_order_items,
        sales_dashboard_data,
    ],
    jobs=[dbt_models_job, full_pipeline_job],
    sensors=[cdc_dbt_sensor, orders_dbt_sensor],
    resources={
        "philotes": PhilotesResource(
            api_url="http://localhost:8080",
            api_key="",
        ),
    },
)
