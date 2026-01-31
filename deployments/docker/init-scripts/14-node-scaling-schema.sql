-- Node Auto-scaling Schema
-- Issue #28: Infrastructure Node Auto-scaling

-- Node Pools - groups of nodes with similar characteristics
CREATE TABLE IF NOT EXISTS philotes.node_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    provider TEXT NOT NULL CHECK (provider IN ('hetzner', 'scaleway', 'ovh', 'exoscale', 'contabo')),
    region TEXT NOT NULL,
    instance_type TEXT NOT NULL,
    image TEXT NOT NULL DEFAULT 'ubuntu-24.04',
    min_nodes INT NOT NULL DEFAULT 1 CHECK (min_nodes >= 0),
    max_nodes INT NOT NULL DEFAULT 10 CHECK (max_nodes >= 1),
    current_nodes INT NOT NULL DEFAULT 0 CHECK (current_nodes >= 0),
    labels JSONB NOT NULL DEFAULT '{}',
    taints JSONB NOT NULL DEFAULT '[]',
    user_data_template TEXT,
    ssh_key_id TEXT,
    network_id TEXT,
    firewall_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT node_pools_min_max_check CHECK (min_nodes <= max_nodes)
);

CREATE INDEX IF NOT EXISTS idx_node_pools_provider ON philotes.node_pools(provider);
CREATE INDEX IF NOT EXISTS idx_node_pools_enabled ON philotes.node_pools(enabled) WHERE enabled = true;

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION philotes.update_node_pools_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_node_pools_updated_at ON philotes.node_pools;
CREATE TRIGGER trigger_node_pools_updated_at
    BEFORE UPDATE ON philotes.node_pools
    FOR EACH ROW
    EXECUTE FUNCTION philotes.update_node_pools_updated_at();


-- Node Pool Nodes - individual nodes belonging to a pool
CREATE TABLE IF NOT EXISTS philotes.node_pool_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES philotes.node_pools(id) ON DELETE CASCADE,
    provider_id TEXT NOT NULL,  -- Cloud provider server ID
    node_name TEXT,             -- Kubernetes node name (set after join)
    status TEXT NOT NULL CHECK (status IN ('creating', 'joining', 'ready', 'draining', 'deleting', 'deleted', 'failed')),
    public_ip TEXT,
    private_ip TEXT,
    instance_type TEXT NOT NULL,
    hourly_cost FLOAT8,
    is_spot BOOLEAN NOT NULL DEFAULT false,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_node_pool_nodes_pool_id ON philotes.node_pool_nodes(pool_id);
CREATE INDEX IF NOT EXISTS idx_node_pool_nodes_status ON philotes.node_pool_nodes(status);
CREATE INDEX IF NOT EXISTS idx_node_pool_nodes_provider_id ON philotes.node_pool_nodes(provider_id);
CREATE INDEX IF NOT EXISTS idx_node_pool_nodes_node_name ON philotes.node_pool_nodes(node_name) WHERE node_name IS NOT NULL;

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION philotes.update_node_pool_nodes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_node_pool_nodes_updated_at ON philotes.node_pool_nodes;
CREATE TRIGGER trigger_node_pool_nodes_updated_at
    BEFORE UPDATE ON philotes.node_pool_nodes
    FOR EACH ROW
    EXECUTE FUNCTION philotes.update_node_pool_nodes_updated_at();


-- Node Scaling Operations - audit log of all scaling actions
CREATE TABLE IF NOT EXISTS philotes.node_scaling_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES philotes.node_pools(id) ON DELETE CASCADE,
    policy_id UUID REFERENCES philotes.scaling_policies(id) ON DELETE SET NULL,
    action TEXT NOT NULL CHECK (action IN ('scale_up', 'scale_down')),
    previous_count INT NOT NULL CHECK (previous_count >= 0),
    target_count INT NOT NULL CHECK (target_count >= 0),
    actual_count INT CHECK (actual_count >= 0),
    status TEXT NOT NULL CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled')),
    reason TEXT,
    triggered_by TEXT,  -- 'policy:<id>', 'schedule:<id>', 'manual', 'api'
    nodes_affected JSONB NOT NULL DEFAULT '[]',  -- Array of node IDs
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    estimated_cost_change FLOAT8,
    dry_run BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_node_scaling_operations_pool_id ON philotes.node_scaling_operations(pool_id);
CREATE INDEX IF NOT EXISTS idx_node_scaling_operations_policy_id ON philotes.node_scaling_operations(policy_id) WHERE policy_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_node_scaling_operations_status ON philotes.node_scaling_operations(status);
CREATE INDEX IF NOT EXISTS idx_node_scaling_operations_started_at ON philotes.node_scaling_operations(started_at DESC);


-- Instance Type Pricing - cached pricing information for cost-aware scaling
CREATE TABLE IF NOT EXISTS philotes.instance_type_pricing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL CHECK (provider IN ('hetzner', 'scaleway', 'ovh', 'exoscale', 'contabo')),
    instance_type TEXT NOT NULL,
    region TEXT NOT NULL,
    hourly_cost FLOAT8 NOT NULL CHECK (hourly_cost >= 0),
    cpu_cores INT NOT NULL CHECK (cpu_cores > 0),
    memory_mb INT NOT NULL CHECK (memory_mb > 0),
    disk_gb INT CHECK (disk_gb >= 0),
    supports_spot BOOLEAN NOT NULL DEFAULT false,
    spot_hourly_cost FLOAT8 CHECK (spot_hourly_cost >= 0),
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(provider, instance_type, region)
);

CREATE INDEX IF NOT EXISTS idx_instance_type_pricing_provider_region ON philotes.instance_type_pricing(provider, region);


-- Seed initial pricing data for common instance types
INSERT INTO philotes.instance_type_pricing (provider, instance_type, region, hourly_cost, cpu_cores, memory_mb, disk_gb, supports_spot)
VALUES
    -- Hetzner Cloud (EUR prices converted to USD approx)
    ('hetzner', 'cx22', 'nbg1', 0.0065, 2, 4096, 40, false),
    ('hetzner', 'cx32', 'nbg1', 0.0130, 4, 8192, 80, false),
    ('hetzner', 'cx42', 'nbg1', 0.0260, 8, 16384, 160, false),
    ('hetzner', 'cx52', 'nbg1', 0.0520, 16, 32768, 320, false),
    ('hetzner', 'cx22', 'fsn1', 0.0065, 2, 4096, 40, false),
    ('hetzner', 'cx32', 'fsn1', 0.0130, 4, 8192, 80, false),
    ('hetzner', 'cx22', 'hel1', 0.0065, 2, 4096, 40, false),
    ('hetzner', 'cx32', 'hel1', 0.0130, 4, 8192, 80, false),

    -- Scaleway
    ('scaleway', 'DEV1-S', 'fr-par-1', 0.0099, 2, 2048, 20, false),
    ('scaleway', 'DEV1-M', 'fr-par-1', 0.0199, 3, 4096, 40, false),
    ('scaleway', 'DEV1-L', 'fr-par-1', 0.0379, 4, 8192, 80, false),
    ('scaleway', 'DEV1-XL', 'fr-par-1', 0.0599, 4, 12288, 120, false),
    ('scaleway', 'DEV1-S', 'nl-ams-1', 0.0099, 2, 2048, 20, false),
    ('scaleway', 'DEV1-M', 'nl-ams-1', 0.0199, 3, 4096, 40, false),

    -- OVHcloud
    ('ovh', 'd2-2', 'GRA11', 0.0069, 1, 2048, 25, false),
    ('ovh', 'd2-4', 'GRA11', 0.0138, 2, 4096, 50, false),
    ('ovh', 'd2-8', 'GRA11', 0.0275, 4, 8192, 50, false),
    ('ovh', 'b2-7', 'GRA11', 0.0413, 2, 7168, 50, false),

    -- Exoscale
    ('exoscale', 'standard.small', 'ch-gva-2', 0.0234, 2, 2048, 50, false),
    ('exoscale', 'standard.medium', 'ch-gva-2', 0.0456, 2, 4096, 100, false),
    ('exoscale', 'standard.large', 'ch-gva-2', 0.0912, 4, 8192, 200, false),
    ('exoscale', 'standard.small', 'de-fra-1', 0.0234, 2, 2048, 50, false),
    ('exoscale', 'standard.medium', 'de-fra-1', 0.0456, 2, 4096, 100, false),

    -- Contabo
    ('contabo', 'VPS-S', 'EU', 0.0069, 4, 8192, 50, false),
    ('contabo', 'VPS-M', 'EU', 0.0125, 6, 16384, 100, false),
    ('contabo', 'VPS-L', 'EU', 0.0194, 8, 30720, 200, false)
ON CONFLICT (provider, instance_type, region) DO UPDATE SET
    hourly_cost = EXCLUDED.hourly_cost,
    cpu_cores = EXCLUDED.cpu_cores,
    memory_mb = EXCLUDED.memory_mb,
    disk_gb = EXCLUDED.disk_gb,
    supports_spot = EXCLUDED.supports_spot,
    last_updated = NOW();
