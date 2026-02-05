# Issue #24: ORCH-001 - Dagster Integration

## Summary

**Goal:** Integrate with Dagster as an optional orchestration layer for users who need to coordinate CDC pipelines with broader data workflows (dbt runs, ML training, etc.).

**Problem it solves:** CDC is often one part of a larger data pipeline. Users may want to trigger dbt models after CDC catches up, or coordinate multiple pipelines. Dagster provides this orchestration layer.

## Who Benefits

- Data teams with existing Dagster deployments
- Users needing pipeline dependencies (CDC → dbt → ML)
- Organizations wanting unified data orchestration

## How It's Used

- Dagster assets represent Philotes pipelines
- Sensors monitor replication lag
- Jobs coordinate downstream processing after CDC reaches a checkpoint
- All orchestration is visible in Dagster UI

## Acceptance Criteria

- [ ] Dagster resource for Philotes API
- [ ] Asset definitions for pipelines and tables
- [ ] Sensors for replication lag and sync completion
- [ ] Job templates for common patterns
- [ ] Dagster UI integration
- [ ] Example DAGs with dbt integration

## Example Asset

```python
@asset
def orders_iceberg(philotes: PhilotesResource):
    """Iceberg table populated by CDC from PostgreSQL."""
    pipeline = philotes.get_pipeline("orders-cdc")
    pipeline.wait_for_lag(max_seconds=60)
    return Output(
        metadata={"lag_seconds": pipeline.current_lag}
    )
```

## Dependencies

- API-002 (completed) - Philotes REST API must be available

## Blocks

- Issue #103 - Dagster RBAC via GraphQL Proxy (requires Dagster to be deployed first)

## Estimate

~4,000 LOC

## Labels

- `epic:orchestration`
- `phase:v1`
- `priority:medium`
- `type:feature`
