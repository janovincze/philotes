-- 16-query-scaling-schema.sql
-- Schema for query engine auto-scaling policies and history

-- Query scaling policies for Trino and RisingWave
CREATE TABLE IF NOT EXISTS query_scaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    query_engine VARCHAR(50) NOT NULL,  -- 'trino', 'risingwave'
    enabled BOOLEAN DEFAULT TRUE,
    min_replicas INTEGER DEFAULT 1,
    max_replicas INTEGER DEFAULT 10,
    cooldown_seconds INTEGER DEFAULT 300,
    scale_to_zero BOOLEAN DEFAULT FALSE,

    -- Trigger thresholds
    queued_queries_threshold INTEGER DEFAULT 5,
    running_queries_threshold INTEGER DEFAULT 10,
    latency_threshold_seconds INTEGER DEFAULT 30,

    -- Schedule-based scaling
    schedule_enabled BOOLEAN DEFAULT FALSE,
    business_hours_min_replicas INTEGER,
    business_hours_start TIME,
    business_hours_end TIME,
    business_hours_timezone VARCHAR(50) DEFAULT 'UTC',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_replicas CHECK (min_replicas <= max_replicas),
    CONSTRAINT chk_min_replicas_positive CHECK (min_replicas >= 0),
    CONSTRAINT chk_max_replicas_positive CHECK (max_replicas > 0),
    CONSTRAINT chk_cooldown_positive CHECK (cooldown_seconds >= 0),
    CONSTRAINT chk_query_engine CHECK (query_engine IN ('trino', 'risingwave'))
);

-- Query scaling action history
CREATE TABLE IF NOT EXISTS query_scaling_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES query_scaling_policies(id) ON DELETE CASCADE,
    query_engine VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,  -- 'scale_up', 'scale_down', 'scale_to_zero', 'wake'
    previous_replicas INTEGER NOT NULL,
    new_replicas INTEGER NOT NULL,
    trigger_reason VARCHAR(255),
    trigger_value DECIMAL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_query_scaling_policies_engine ON query_scaling_policies(query_engine);
CREATE INDEX IF NOT EXISTS idx_query_scaling_policies_enabled ON query_scaling_policies(enabled);
CREATE INDEX IF NOT EXISTS idx_query_scaling_history_policy ON query_scaling_history(policy_id);
CREATE INDEX IF NOT EXISTS idx_query_scaling_history_engine ON query_scaling_history(query_engine);
CREATE INDEX IF NOT EXISTS idx_query_scaling_history_executed ON query_scaling_history(executed_at DESC);

-- Trigger for updating updated_at timestamp
CREATE OR REPLACE FUNCTION update_query_scaling_policies_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS query_scaling_policies_updated_at ON query_scaling_policies;
CREATE TRIGGER query_scaling_policies_updated_at
    BEFORE UPDATE ON query_scaling_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_query_scaling_policies_updated_at();

-- Comments
COMMENT ON TABLE query_scaling_policies IS 'Query engine auto-scaling policies for Trino and RisingWave';
COMMENT ON TABLE query_scaling_history IS 'History of query engine scaling actions';
COMMENT ON COLUMN query_scaling_policies.query_engine IS 'Query engine type: trino, risingwave';
COMMENT ON COLUMN query_scaling_policies.scale_to_zero IS 'Allow scaling to zero workers when idle';
COMMENT ON COLUMN query_scaling_policies.queued_queries_threshold IS 'Scale up when queued queries exceed this value';
COMMENT ON COLUMN query_scaling_policies.running_queries_threshold IS 'Scale up when running queries exceed this value';
COMMENT ON COLUMN query_scaling_policies.latency_threshold_seconds IS 'Scale up when p95 query latency exceeds this value';
