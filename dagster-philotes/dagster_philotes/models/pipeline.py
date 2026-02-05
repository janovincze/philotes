"""Pipeline models for Philotes API."""

from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, Field


class PipelineStatus(str, Enum):
    """Pipeline status enum."""

    STOPPED = "stopped"
    STARTING = "starting"
    RUNNING = "running"
    STOPPING = "stopping"
    ERROR = "error"


class PipelineConfig(BaseModel):
    """Pipeline configuration."""

    batch_size: int = Field(default=1000, description="Batch size for CDC events")
    flush_interval_ms: int = Field(
        default=5000, description="Flush interval in milliseconds"
    )
    max_lag_seconds: float = Field(
        default=60.0, description="Maximum acceptable replication lag"
    )

    model_config = {"extra": "allow"}


class TableMapping(BaseModel):
    """Table mapping configuration."""

    id: str = Field(description="Mapping ID")
    pipeline_id: str = Field(description="Pipeline ID")
    source_schema: str = Field(description="Source schema name")
    source_table: str = Field(description="Source table name")
    enabled: bool = Field(default=True, description="Whether mapping is enabled")
    config: dict[str, Any] = Field(default_factory=dict, description="Table config")
    created_at: datetime | None = Field(default=None, description="Creation timestamp")


class Pipeline(BaseModel):
    """Philotes CDC pipeline."""

    id: str = Field(description="Pipeline ID")
    name: str = Field(description="Pipeline name")
    source_id: str = Field(description="Source database ID")
    status: PipelineStatus = Field(description="Current status")
    config: PipelineConfig = Field(
        default_factory=PipelineConfig, description="Pipeline configuration"
    )
    error_message: str | None = Field(default=None, description="Error message if any")
    tables: list[TableMapping] = Field(
        default_factory=list, description="Table mappings"
    )
    created_at: datetime | None = Field(default=None, description="Creation timestamp")
    updated_at: datetime | None = Field(default=None, description="Last update timestamp")
    started_at: datetime | None = Field(default=None, description="Last start timestamp")
    stopped_at: datetime | None = Field(default=None, description="Last stop timestamp")

    @property
    def is_running(self) -> bool:
        """Check if pipeline is running."""
        return self.status == PipelineStatus.RUNNING

    @property
    def is_healthy(self) -> bool:
        """Check if pipeline is healthy (running without errors)."""
        return self.status == PipelineStatus.RUNNING and self.error_message is None
