-- Philotes Buffer Database Schema
-- This script initializes the buffer database for CDC events

-- Create schema for CDC events
CREATE SCHEMA IF NOT EXISTS cdc;

-- CDC Sources - tracks registered source databases
CREATE TABLE IF NOT EXISTS cdc.sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    source_type VARCHAR(50) NOT NULL DEFAULT 'postgresql',
    connection_config JSONB NOT NULL,
    tables JSONB DEFAULT '[]'::jsonb,
    replication_slot VARCHAR(255),
    publication VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- CDC Events - buffered events waiting to be written to Iceberg
CREATE TABLE IF NOT EXISTS cdc.events (
    id BIGSERIAL PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES cdc.sources(id),
    table_name VARCHAR(255) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    lsn VARCHAR(50) NOT NULL,
    transaction_id BIGINT,
    event_data BYTEA NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for efficient batch retrieval
CREATE INDEX IF NOT EXISTS idx_cdc_events_source_created
    ON cdc.events(source_id, created_at);

-- CDC Checkpoints - tracks replication position per source
CREATE TABLE IF NOT EXISTS cdc.checkpoints (
    source_id UUID PRIMARY KEY REFERENCES cdc.sources(id),
    lsn VARCHAR(50) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- CDC Pipelines - pipeline configurations
CREATE TABLE IF NOT EXISTS cdc.pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    source_id UUID NOT NULL REFERENCES cdc.sources(id),
    destination_config JSONB NOT NULL,
    table_mappings JSONB DEFAULT '[]'::jsonb,
    batch_size INTEGER NOT NULL DEFAULT 1000,
    flush_interval_ms INTEGER NOT NULL DEFAULT 5000,
    status VARCHAR(50) NOT NULL DEFAULT 'stopped',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Schema for Lakekeeper (if needed)
CREATE SCHEMA IF NOT EXISTS lakekeeper;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_sources_updated_at
    BEFORE UPDATE ON cdc.sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pipelines_updated_at
    BEFORE UPDATE ON cdc.pipelines
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_checkpoints_updated_at
    BEFORE UPDATE ON cdc.checkpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions
GRANT ALL ON SCHEMA cdc TO philotes;
GRANT ALL ON ALL TABLES IN SCHEMA cdc TO philotes;
GRANT ALL ON ALL SEQUENCES IN SCHEMA cdc TO philotes;
GRANT ALL ON SCHEMA lakekeeper TO philotes;
