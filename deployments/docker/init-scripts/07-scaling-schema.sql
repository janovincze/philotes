-- Scaling Engine Schema
-- This schema supports the scaling policy framework for auto-scaling CDC workers,
-- query engines, and infrastructure nodes.

-- ============================================================================
-- Scaling Policies
-- ============================================================================

CREATE TABLE IF NOT EXISTS scaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    target_type TEXT NOT NULL CHECK (target_type IN ('cdc-worker', 'trino', 'risingwave', 'nodes')),
    target_id UUID,  -- NULL for infrastructure-level scaling (nodes)
    min_replicas INT NOT NULL DEFAULT 1 CHECK (min_replicas >= 0),
    max_replicas INT NOT NULL DEFAULT 10 CHECK (max_replicas >= 1),
    cooldown_seconds INT NOT NULL DEFAULT 300 CHECK (cooldown_seconds >= 0),
    max_hourly_cost FLOAT8,  -- NULL means no cost limit
    scale_to_zero BOOLEAN NOT NULL DEFAULT false,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure min <= max
    CONSTRAINT scaling_policies_min_max_check CHECK (min_replicas <= max_replicas)
);

CREATE INDEX IF NOT EXISTS idx_scaling_policies_enabled ON scaling_policies(enabled) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_scaling_policies_target_type ON scaling_policies(target_type);
CREATE INDEX IF NOT EXISTS idx_scaling_policies_target_id ON scaling_policies(target_id) WHERE target_id IS NOT NULL;

-- ============================================================================
-- Scaling Rules
-- ============================================================================

CREATE TABLE IF NOT EXISTS scaling_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    rule_type TEXT NOT NULL CHECK (rule_type IN ('scale_up', 'scale_down')),
    metric TEXT NOT NULL,  -- e.g., "philotes_cdc_lag_seconds", "philotes_buffer_depth"
    operator TEXT NOT NULL CHECK (operator IN ('gt', 'lt', 'gte', 'lte', 'eq')),
    threshold FLOAT8 NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    scale_by INT NOT NULL,  -- positive for scale up, negative for scale down
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure scale_by sign matches rule_type
    CONSTRAINT scaling_rules_scale_by_check CHECK (
        (rule_type = 'scale_up' AND scale_by > 0) OR
        (rule_type = 'scale_down' AND scale_by < 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_scaling_rules_policy_id ON scaling_rules(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_rules_type ON scaling_rules(policy_id, rule_type);

-- ============================================================================
-- Scaling Schedules
-- ============================================================================

CREATE TABLE IF NOT EXISTS scaling_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    cron_expression TEXT NOT NULL,  -- e.g., "0 8 * * 1-5" (weekdays at 8am)
    desired_replicas INT NOT NULL CHECK (desired_replicas >= 0),
    timezone TEXT NOT NULL DEFAULT 'UTC',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scaling_schedules_policy_id ON scaling_schedules(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_schedules_enabled ON scaling_schedules(enabled) WHERE enabled = true;

-- ============================================================================
-- Scaling History (Audit Log)
-- ============================================================================

CREATE TABLE IF NOT EXISTS scaling_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES scaling_policies(id) ON DELETE SET NULL,
    policy_name TEXT NOT NULL,  -- Denormalized for audit purposes
    action TEXT NOT NULL CHECK (action IN ('scale_up', 'scale_down', 'scheduled', 'manual')),
    target_type TEXT NOT NULL,
    target_id UUID,
    previous_replicas INT NOT NULL,
    new_replicas INT NOT NULL,
    reason TEXT,  -- Human-readable reason for scaling
    triggered_by TEXT,  -- "rule:<rule_id>", "schedule:<schedule_id>", "manual:<user>"
    dry_run BOOLEAN NOT NULL DEFAULT false,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scaling_history_policy_id ON scaling_history(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_history_executed_at ON scaling_history(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_scaling_history_target ON scaling_history(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_scaling_history_dry_run ON scaling_history(dry_run) WHERE dry_run = true;

-- ============================================================================
-- Scaling State (for tracking cooldowns and pending conditions)
-- ============================================================================

CREATE TABLE IF NOT EXISTS scaling_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE UNIQUE,
    current_replicas INT NOT NULL DEFAULT 0,
    last_scale_time TIMESTAMPTZ,
    last_scale_action TEXT,
    pending_conditions JSONB,  -- Tracks when conditions first became true for duration checks
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scaling_state_policy_id ON scaling_state(policy_id);

-- ============================================================================
-- Functions
-- ============================================================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_scaling_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
DROP TRIGGER IF EXISTS trigger_scaling_policies_updated_at ON scaling_policies;
CREATE TRIGGER trigger_scaling_policies_updated_at
    BEFORE UPDATE ON scaling_policies
    FOR EACH ROW
    EXECUTE FUNCTION update_scaling_updated_at();

DROP TRIGGER IF EXISTS trigger_scaling_state_updated_at ON scaling_state;
CREATE TRIGGER trigger_scaling_state_updated_at
    BEFORE UPDATE ON scaling_state
    FOR EACH ROW
    EXECUTE FUNCTION update_scaling_updated_at();
