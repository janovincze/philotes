-- Buffer Schema for Philotes CDC Events
-- This script creates the tables required for CDC event buffering

-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS philotes;

-- CDC Events buffer table
-- Stores events captured from source databases before forwarding to Iceberg
CREATE TABLE IF NOT EXISTS philotes.cdc_events (
    id BIGSERIAL PRIMARY KEY,

    -- Source identification
    source_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,

    -- Event details
    operation TEXT NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE', 'TRUNCATE')),
    lsn TEXT NOT NULL,
    transaction_id BIGINT,

    -- Event data (stored as JSONB for flexibility)
    key_columns JSONB,
    before_data JSONB,
    after_data JSONB,
    metadata JSONB,

    -- Timestamps
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ  -- NULL means unprocessed
);

-- Index for fetching unprocessed events efficiently
-- This is the primary query pattern: get oldest unprocessed events for a source
CREATE INDEX IF NOT EXISTS idx_cdc_events_unprocessed
    ON philotes.cdc_events (source_id, created_at)
    WHERE processed_at IS NULL;

-- Index for cleanup of processed events
-- Used by retention cleanup job
CREATE INDEX IF NOT EXISTS idx_cdc_events_processed
    ON philotes.cdc_events (processed_at)
    WHERE processed_at IS NOT NULL;

-- Index for querying by table
CREATE INDEX IF NOT EXISTS idx_cdc_events_table
    ON philotes.cdc_events (source_id, schema_name, table_name);

-- Index for LSN lookups (for replay/debugging)
CREATE INDEX IF NOT EXISTS idx_cdc_events_lsn
    ON philotes.cdc_events (source_id, lsn);

-- Comments for documentation
COMMENT ON TABLE philotes.cdc_events IS 'Buffer for CDC events before forwarding to Iceberg';
COMMENT ON COLUMN philotes.cdc_events.source_id IS 'Identifier for the source database/pipeline';
COMMENT ON COLUMN philotes.cdc_events.processed_at IS 'NULL = unprocessed, timestamp = when it was processed';
COMMENT ON COLUMN philotes.cdc_events.lsn IS 'PostgreSQL Log Sequence Number for ordering';

-- Grant permissions
GRANT ALL ON TABLE philotes.cdc_events TO philotes;
GRANT USAGE, SELECT ON SEQUENCE philotes.cdc_events_id_seq TO philotes;
