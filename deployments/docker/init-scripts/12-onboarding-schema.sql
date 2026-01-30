-- Onboarding Schema for Philotes
-- This script creates the tables required for the post-installation setup wizard

-- Onboarding progress table stores wizard state for resumability
CREATE TABLE IF NOT EXISTS philotes.onboarding_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    current_step INTEGER NOT NULL DEFAULT 1,
    completed_steps INTEGER[] NOT NULL DEFAULT '{}',
    step_data JSONB NOT NULL DEFAULT '{}',
    metrics JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Indexes for onboarding progress
CREATE INDEX IF NOT EXISTS idx_onboarding_progress_user_id ON philotes.onboarding_progress(user_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_progress_session_id ON philotes.onboarding_progress(session_id);
CREATE INDEX IF NOT EXISTS idx_onboarding_progress_completed_at ON philotes.onboarding_progress(completed_at);

-- Add comments for documentation
COMMENT ON TABLE philotes.onboarding_progress IS 'Tracks post-installation wizard progress for resumability';
COMMENT ON COLUMN philotes.onboarding_progress.session_id IS 'Browser session ID for anonymous progress tracking';
COMMENT ON COLUMN philotes.onboarding_progress.current_step IS 'Current wizard step (1-7)';
COMMENT ON COLUMN philotes.onboarding_progress.completed_steps IS 'Array of completed step numbers';
COMMENT ON COLUMN philotes.onboarding_progress.step_data IS 'JSON data collected from each step';
COMMENT ON COLUMN philotes.onboarding_progress.metrics IS 'Analytics data: time per step, skipped steps, etc.';

-- Grant permissions
GRANT ALL ON TABLE philotes.onboarding_progress TO philotes;
