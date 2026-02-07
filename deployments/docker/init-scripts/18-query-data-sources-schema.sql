-- Query Data Sources schema
-- Stores external data source configurations for query federation via Trino dynamic catalogs.

CREATE TABLE IF NOT EXISTS philotes.query_data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES philotes.tenants(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('postgresql', 'mysql')),
    catalog_name TEXT NOT NULL UNIQUE,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database_name TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL, -- TODO: encrypt at rest via Vault transit engine (see Issue #70 follow-up)
    ssl_mode TEXT NOT NULL DEFAULT 'prefer',
    extra_config JSONB NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'inactive' CHECK (status IN ('inactive', 'active', 'error')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

-- Index for listing by tenant
CREATE INDEX IF NOT EXISTS idx_query_data_sources_tenant_id ON philotes.query_data_sources(tenant_id);
