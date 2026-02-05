"""Metrics models for Philotes API."""

from datetime import datetime

from pydantic import BaseModel, Field

from dagster_philotes.models.pipeline import PipelineStatus


class TableMetrics(BaseModel):
    """Metrics for a single table in a pipeline."""

    # Note: Using 'schema' directly as field name (valid in Pydantic v2)
    # This matches the JSON key from the Philotes API response
    schema: str = Field(description="Database schema name")
    table: str = Field(description="Table name")
    events_processed: int = Field(default=0, description="Total events processed")
    lag_seconds: float = Field(default=0.0, description="Replication lag in seconds")
    last_event_at: datetime | None = Field(
        default=None, description="Timestamp of last event"
    )

    @property
    def full_name(self) -> str:
        """Get fully qualified table name."""
        return f"{self.schema}.{self.table}"


class PipelineMetrics(BaseModel):
    """Metrics for a Philotes pipeline."""

    pipeline_id: str = Field(description="Pipeline ID")
    status: PipelineStatus = Field(description="Current pipeline status")
    events_processed: int = Field(default=0, description="Total events processed")
    events_per_second: float = Field(default=0.0, description="Current events/sec rate")
    lag_seconds: float = Field(default=0.0, description="Current replication lag")
    lag_p95_seconds: float = Field(default=0.0, description="P95 replication lag")
    buffer_depth: int = Field(default=0, description="Events in buffer")
    error_count: int = Field(default=0, description="Total error count")
    iceberg_commits: int = Field(default=0, description="Iceberg commits made")
    iceberg_bytes_written: int = Field(default=0, description="Bytes written to Iceberg")
    last_event_at: datetime | None = Field(
        default=None, description="Timestamp of last event"
    )
    uptime: str = Field(default="0s", description="Pipeline uptime")
    tables: list[TableMetrics] = Field(
        default_factory=list, description="Per-table metrics"
    )

    @property
    def is_caught_up(self) -> bool:
        """Check if pipeline has caught up (lag < 1 second)."""
        return self.lag_seconds < 1.0

    def is_within_lag(self, max_lag_seconds: float) -> bool:
        """Check if lag is within acceptable threshold."""
        return self.lag_seconds <= max_lag_seconds

    def to_dagster_metadata(self) -> dict[str, float | int | str]:
        """Convert to Dagster metadata dictionary."""
        return {
            "status": self.status.value,
            "events_processed": self.events_processed,
            "events_per_second": self.events_per_second,
            "lag_seconds": self.lag_seconds,
            "buffer_depth": self.buffer_depth,
            "error_count": self.error_count,
            "iceberg_commits": self.iceberg_commits,
            "uptime": self.uptime,
        }
