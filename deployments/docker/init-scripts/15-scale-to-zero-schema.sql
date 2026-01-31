-- Scale-to-Zero Schema Migration
-- Adds idle detection and cost tracking for scale-to-zero functionality

-- Add idle configuration columns to scaling_policies
ALTER TABLE scaling_policies
ADD COLUMN IF NOT EXISTS idle_threshold_seconds INTEGER DEFAULT 1800,
ADD COLUMN IF NOT EXISTS keep_alive_window_seconds INTEGER DEFAULT 300,
ADD COLUMN IF NOT EXISTS cold_start_timeout_seconds INTEGER DEFAULT 120;

COMMENT ON COLUMN scaling_policies.idle_threshold_seconds IS 'Seconds of inactivity before scaling to zero';
COMMENT ON COLUMN scaling_policies.keep_alive_window_seconds IS 'Grace period to prevent flapping';
COMMENT ON COLUMN scaling_policies.cold_start_timeout_seconds IS 'Maximum time to wait for cold start';

-- Idle state tracking table
CREATE TABLE IF NOT EXISTS scaling_idle_state (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    idle_since TIMESTAMP WITH TIME ZONE,
    scaled_to_zero_at TIMESTAMP WITH TIME ZONE,
    last_wake_at TIMESTAMP WITH TIME ZONE,
    wake_reason TEXT,
    is_scaled_to_zero BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(policy_id)
);

COMMENT ON TABLE scaling_idle_state IS 'Tracks idle state for scale-to-zero policies';
COMMENT ON COLUMN scaling_idle_state.last_activity_at IS 'Timestamp of last activity (event, API call)';
COMMENT ON COLUMN scaling_idle_state.idle_since IS 'When the policy became idle (null if active)';
COMMENT ON COLUMN scaling_idle_state.scaled_to_zero_at IS 'When the policy was scaled to zero';
COMMENT ON COLUMN scaling_idle_state.wake_reason IS 'Reason for last wake: manual, scheduled, webhook, api_request';

CREATE INDEX IF NOT EXISTS idx_scaling_idle_state_policy ON scaling_idle_state(policy_id);
CREATE INDEX IF NOT EXISTS idx_scaling_idle_state_scaled_to_zero ON scaling_idle_state(is_scaled_to_zero) WHERE is_scaled_to_zero = TRUE;

-- Cost savings tracking table
CREATE TABLE IF NOT EXISTS scaling_cost_savings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES scaling_policies(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    idle_seconds BIGINT DEFAULT 0,
    scaled_to_zero_seconds BIGINT DEFAULT 0,
    estimated_savings_cents INTEGER DEFAULT 0,
    hourly_cost_cents INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(policy_id, date)
);

COMMENT ON TABLE scaling_cost_savings IS 'Tracks cost savings from scale-to-zero';
COMMENT ON COLUMN scaling_cost_savings.idle_seconds IS 'Total seconds idle (but not scaled to zero) for the day';
COMMENT ON COLUMN scaling_cost_savings.scaled_to_zero_seconds IS 'Total seconds scaled to zero for the day';
COMMENT ON COLUMN scaling_cost_savings.estimated_savings_cents IS 'Estimated cost savings in cents';
COMMENT ON COLUMN scaling_cost_savings.hourly_cost_cents IS 'Hourly cost rate used for calculation';

CREATE INDEX IF NOT EXISTS idx_scaling_cost_savings_policy_date ON scaling_cost_savings(policy_id, date DESC);

-- Trigger to update updated_at on scaling_idle_state
CREATE OR REPLACE FUNCTION update_scaling_idle_state_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_scaling_idle_state_updated_at ON scaling_idle_state;
CREATE TRIGGER trigger_scaling_idle_state_updated_at
    BEFORE UPDATE ON scaling_idle_state
    FOR EACH ROW
    EXECUTE FUNCTION update_scaling_idle_state_updated_at();

-- Trigger to update updated_at on scaling_cost_savings
DROP TRIGGER IF EXISTS trigger_scaling_cost_savings_updated_at ON scaling_cost_savings;
CREATE TRIGGER trigger_scaling_cost_savings_updated_at
    BEFORE UPDATE ON scaling_cost_savings
    FOR EACH ROW
    EXECUTE FUNCTION update_scaling_idle_state_updated_at();
