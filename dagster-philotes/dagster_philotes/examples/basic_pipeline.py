"""Basic pipeline monitoring example.

This example shows how to create Dagster assets that represent
Philotes CDC pipelines and wait for them to catch up.
"""

from dagster import AssetExecutionContext, Definitions, asset

from dagster_philotes import PhilotesResource, philotes_pipeline_asset

# Create an asset for a CDC pipeline using the factory
orders_cdc = philotes_pipeline_asset(
    pipeline_id="orders-cdc",
    name="orders_iceberg",
    group_name="ecommerce",
    max_lag_seconds=60.0,
    wait_timeout=300.0,
)

# Create another asset for a different pipeline
customers_cdc = philotes_pipeline_asset(
    pipeline_id="customers-cdc",
    name="customers_iceberg",
    group_name="ecommerce",
    max_lag_seconds=60.0,
)


# You can also create custom assets with more control
@asset(
    group_name="ecommerce",
    deps=[orders_cdc, customers_cdc],
)
def combined_report(
    context: AssetExecutionContext,
    philotes: PhilotesResource,
) -> dict[str, int]:
    """Generate a combined report after both CDCs are caught up.

    This asset depends on orders_iceberg and customers_iceberg,
    so it will only run after both pipelines have caught up.
    """
    orders_metrics = philotes.get_pipeline_metrics("orders-cdc")
    customers_metrics = philotes.get_pipeline_metrics("customers-cdc")

    total_events = orders_metrics.events_processed + customers_metrics.events_processed

    context.log.info(f"Total events across pipelines: {total_events}")

    return {
        "orders_events": orders_metrics.events_processed,
        "customers_events": customers_metrics.events_processed,
        "total_events": total_events,
    }


# Create definitions for this example
basic_defs = Definitions(
    assets=[orders_cdc, customers_cdc, combined_report],
    resources={
        "philotes": PhilotesResource(
            # These will be overridden by environment variables in production
            api_url="http://localhost:8080",
            api_key="",
        ),
    },
)
