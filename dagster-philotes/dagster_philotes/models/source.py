"""Source models for Philotes API."""

from datetime import datetime
from enum import Enum

from pydantic import BaseModel, Field


class SourceStatus(str, Enum):
    """Source status enum."""

    INACTIVE = "inactive"
    ACTIVE = "active"
    ERROR = "error"


class Source(BaseModel):
    """Philotes source database configuration."""

    id: str = Field(description="Source ID")
    name: str = Field(description="Source name")
    type: str = Field(default="postgresql", description="Database type")
    host: str = Field(description="Database host")
    port: int = Field(default=5432, description="Database port")
    database_name: str = Field(description="Database name")
    username: str = Field(description="Database username")
    ssl_mode: str = Field(default="disable", description="SSL mode")
    slot_name: str | None = Field(
        default=None, description="Replication slot name"
    )
    publication_name: str | None = Field(
        default=None, description="Publication name"
    )
    status: SourceStatus = Field(
        default=SourceStatus.INACTIVE, description="Source status"
    )
    created_at: datetime | None = Field(default=None, description="Creation timestamp")
    updated_at: datetime | None = Field(default=None, description="Last update timestamp")

    @property
    def connection_string(self) -> str:
        """Get a redacted connection string for display."""
        return f"postgresql://{self.username}:***@{self.host}:{self.port}/{self.database_name}"
