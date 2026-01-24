-- CDC Schema for Philotes
-- This script creates the tables required for CDC checkpoint management

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS philotes;

-- Checkpoints table stores the last processed LSN for each source
CREATE TABLE IF NOT EXISTS philotes.cdc_checkpoints (
    source_id TEXT PRIMARY KEY,
    lsn TEXT NOT NULL,
    transaction_id BIGINT,
    committed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Index for querying by committed time
CREATE INDEX IF NOT EXISTS idx_cdc_checkpoints_committed_at
    ON philotes.cdc_checkpoints(committed_at);

-- Schema history table tracks DDL changes from source databases
CREATE TABLE IF NOT EXISTS philotes.cdc_schema_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    version INT NOT NULL,
    columns JSONB NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lsn TEXT NOT NULL,
    UNIQUE(source_id, schema_name, table_name, version)
);

-- Indexes for schema history queries
CREATE INDEX IF NOT EXISTS idx_cdc_schema_history_source
    ON philotes.cdc_schema_history(source_id);
CREATE INDEX IF NOT EXISTS idx_cdc_schema_history_table
    ON philotes.cdc_schema_history(source_id, schema_name, table_name);
CREATE INDEX IF NOT EXISTS idx_cdc_schema_history_captured
    ON philotes.cdc_schema_history(captured_at);

-- Add comment for documentation
COMMENT ON TABLE philotes.cdc_checkpoints IS 'Stores CDC checkpoint positions for exactly-once semantics';
COMMENT ON TABLE philotes.cdc_schema_history IS 'Tracks schema changes from source databases';

-- Grant permissions (adjust as needed for your setup)
GRANT ALL ON SCHEMA philotes TO philotes;
GRANT ALL ON ALL TABLES IN SCHEMA philotes TO philotes;
