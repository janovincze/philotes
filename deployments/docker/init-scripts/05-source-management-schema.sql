-- Source and Pipeline Management Schema for Philotes
-- This script creates the tables required for managing CDC sources and pipelines

-- Sources table stores registered source databases
CREATE TABLE IF NOT EXISTS philotes.sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL DEFAULT 'postgresql',
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 5432,
    database_name TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    ssl_mode TEXT NOT NULL DEFAULT 'prefer',
    slot_name TEXT,
    publication_name TEXT,
    status TEXT NOT NULL DEFAULT 'inactive',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying sources by status
CREATE INDEX IF NOT EXISTS idx_sources_status ON philotes.sources(status);
CREATE INDEX IF NOT EXISTS idx_sources_name ON philotes.sources(name);

-- Pipelines table stores pipeline definitions
CREATE TABLE IF NOT EXISTS philotes.pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    source_id UUID NOT NULL REFERENCES philotes.sources(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'stopped',
    config JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    stopped_at TIMESTAMPTZ
);

-- Indexes for pipelines
CREATE INDEX IF NOT EXISTS idx_pipelines_status ON philotes.pipelines(status);
CREATE INDEX IF NOT EXISTS idx_pipelines_source_id ON philotes.pipelines(source_id);
CREATE INDEX IF NOT EXISTS idx_pipelines_name ON philotes.pipelines(name);

-- Table mappings for pipelines
CREATE TABLE IF NOT EXISTS philotes.table_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES philotes.pipelines(id) ON DELETE CASCADE,
    source_schema TEXT NOT NULL DEFAULT 'public',
    source_table TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(pipeline_id, source_schema, source_table)
);

-- Indexes for table mappings
CREATE INDEX IF NOT EXISTS idx_table_mappings_pipeline_id ON philotes.table_mappings(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_table_mappings_enabled ON philotes.table_mappings(pipeline_id, enabled);

-- Add comments for documentation
COMMENT ON TABLE philotes.sources IS 'Registered source databases for CDC';
COMMENT ON TABLE philotes.pipelines IS 'CDC pipeline definitions linking sources to processing';
COMMENT ON TABLE philotes.table_mappings IS 'Table-level configurations for pipelines';

-- Grant permissions
GRANT ALL ON TABLE philotes.sources TO philotes;
GRANT ALL ON TABLE philotes.pipelines TO philotes;
GRANT ALL ON TABLE philotes.table_mappings TO philotes;
