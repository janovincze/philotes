"""Asset definitions for Philotes pipelines."""

from typing import Any

from dagster import (
    AssetExecutionContext,
    AssetsDefinition,
    Output,
    asset,
)

from dagster_philotes.resources import PhilotesResource


def philotes_pipeline_asset(
    pipeline_id: str,
    name: str | None = None,
    group_name: str = "philotes",
    description: str | None = None,
    max_lag_seconds: float = 60.0,
    wait_timeout: float = 300.0,
    metadata: dict[str, Any] | None = None,
) -> AssetsDefinition:
    """Create a Dagster asset representing a Philotes CDC pipeline.

    This factory creates an asset that represents the state of a CDC pipeline.
    When materialized, it waits for the pipeline's replication lag to drop below
    the specified threshold, ensuring downstream assets have access to fresh data.

    Args:
        pipeline_id: The Philotes pipeline ID to monitor
        name: Asset name (defaults to pipeline_id with dashes replaced by underscores)
        group_name: Asset group name (default: "philotes")
        description: Asset description (auto-generated if not provided)
        max_lag_seconds: Maximum acceptable replication lag in seconds (default: 60.0)
        wait_timeout: Maximum time to wait for lag in seconds (default: 300.0)
        metadata: Additional static metadata for the asset

    Returns:
        A Dagster AssetsDefinition that can be included in Definitions

    Example:
        ```python
        from dagster import Definitions
        from dagster_philotes import philotes_pipeline_asset, PhilotesResource

        orders_cdc = philotes_pipeline_asset(
            pipeline_id="orders-cdc",
            name="orders_iceberg",
            max_lag_seconds=30.0,
        )

        defs = Definitions(
            assets=[orders_cdc],
            resources={"philotes": PhilotesResource(...)},
        )
        ```
    """
    asset_name = name or pipeline_id.replace("-", "_")
    asset_description = description or (
        f"Iceberg table populated by Philotes CDC pipeline '{pipeline_id}'. "
        f"Materializing this asset waits for replication lag to drop below {max_lag_seconds}s."
    )

    @asset(
        name=asset_name,
        group_name=group_name,
        description=asset_description,
    )
    def _pipeline_asset(
        context: AssetExecutionContext,
        philotes: PhilotesResource,
    ) -> Output[None]:
        context.log.info(f"Checking pipeline '{pipeline_id}' metrics...")

        # Get initial metrics
        initial_metrics = philotes.get_pipeline_metrics(pipeline_id)
        context.log.info(
            f"Pipeline status: {initial_metrics.status.value}, "
            f"lag: {initial_metrics.lag_seconds:.2f}s, "
            f"events processed: {initial_metrics.events_processed}"
        )

        # Wait for lag to be acceptable
        if not initial_metrics.is_within_lag(max_lag_seconds):
            context.log.info(
                f"Waiting for lag to drop below {max_lag_seconds}s "
                f"(current: {initial_metrics.lag_seconds:.2f}s)..."
            )

        final_metrics = philotes.wait_for_lag(
            pipeline_id=pipeline_id,
            max_lag_seconds=max_lag_seconds,
            timeout=wait_timeout,
        )

        context.log.info(
            f"Pipeline caught up! Lag: {final_metrics.lag_seconds:.2f}s, "
            f"events: {final_metrics.events_processed}"
        )

        # Build output metadata
        output_metadata = final_metrics.to_dagster_metadata()
        if metadata:
            output_metadata.update(metadata)

        return Output(
            value=None,
            metadata=output_metadata,
        )

    return _pipeline_asset


def philotes_table_asset(
    pipeline_id: str,
    schema_name: str,
    table_name: str,
    name: str | None = None,
    group_name: str = "philotes",
    description: str | None = None,
    max_lag_seconds: float = 60.0,
    wait_timeout: float = 300.0,
) -> AssetsDefinition:
    """Create a Dagster asset representing a specific table in a CDC pipeline.

    This is more granular than `philotes_pipeline_asset` - it creates an asset
    for a specific table within a pipeline, allowing table-level dependencies.

    Args:
        pipeline_id: The Philotes pipeline ID
        schema_name: Source schema name
        table_name: Source table name
        name: Asset name (defaults to "{schema}_{table}")
        group_name: Asset group name (default: "philotes")
        description: Asset description
        max_lag_seconds: Maximum acceptable replication lag in seconds
        wait_timeout: Maximum time to wait for lag in seconds

    Returns:
        A Dagster AssetsDefinition for the specific table

    Example:
        ```python
        orders_table = philotes_table_asset(
            pipeline_id="ecommerce-cdc",
            schema_name="public",
            table_name="orders",
            max_lag_seconds=30.0,
        )
        ```
    """
    asset_name = name or f"{schema_name}_{table_name}"
    full_table_name = f"{schema_name}.{table_name}"
    asset_description = description or (
        f"Iceberg table for '{full_table_name}' from CDC pipeline '{pipeline_id}'."
    )

    @asset(
        name=asset_name,
        group_name=group_name,
        description=asset_description,
    )
    def _table_asset(
        context: AssetExecutionContext,
        philotes: PhilotesResource,
    ) -> Output[None]:
        context.log.info(f"Checking table '{full_table_name}' in pipeline '{pipeline_id}'...")

        # Wait for pipeline lag
        metrics = philotes.wait_for_lag(
            pipeline_id=pipeline_id,
            max_lag_seconds=max_lag_seconds,
            timeout=wait_timeout,
        )

        # Find table-specific metrics
        table_metrics = None
        for tm in metrics.tables:
            if tm.schema_name == schema_name and tm.table == table_name:
                table_metrics = tm
                break

        output_metadata: dict[str, Any] = {
            "pipeline_id": pipeline_id,
            "schema": schema_name,
            "table": table_name,
            "pipeline_lag_seconds": metrics.lag_seconds,
            "pipeline_events_processed": metrics.events_processed,
        }

        if table_metrics:
            output_metadata.update({
                "table_lag_seconds": table_metrics.lag_seconds,
                "table_events_processed": table_metrics.events_processed,
            })
            context.log.info(
                f"Table '{full_table_name}' metrics: "
                f"lag={table_metrics.lag_seconds:.2f}s, "
                f"events={table_metrics.events_processed}"
            )
        else:
            context.log.warning(f"No table-specific metrics found for '{full_table_name}'")

        return Output(value=None, metadata=output_metadata)

    return _table_asset


def multi_pipeline_assets(
    pipelines: dict[str, str],
    group_name: str = "philotes",
    max_lag_seconds: float = 60.0,
    wait_timeout: float = 300.0,
) -> list[AssetsDefinition]:
    """Create multiple pipeline assets at once.

    Args:
        pipelines: Dictionary mapping asset names to pipeline IDs
        group_name: Asset group name
        max_lag_seconds: Maximum acceptable replication lag
        wait_timeout: Maximum time to wait for lag

    Returns:
        List of asset definitions

    Example:
        ```python
        cdc_assets = multi_pipeline_assets({
            "orders_iceberg": "orders-cdc",
            "customers_iceberg": "customers-cdc",
            "products_iceberg": "products-cdc",
        })
        ```
    """
    return [
        philotes_pipeline_asset(
            pipeline_id=pipeline_id,
            name=asset_name,
            group_name=group_name,
            max_lag_seconds=max_lag_seconds,
            wait_timeout=wait_timeout,
        )
        for asset_name, pipeline_id in pipelines.items()
    ]
