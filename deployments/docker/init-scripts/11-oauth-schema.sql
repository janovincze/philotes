-- OAuth Schema for Philotes
-- This script creates the tables required for OAuth authentication with cloud providers

-- OAuth states table stores temporary PKCE state during OAuth flow
CREATE TABLE IF NOT EXISTS philotes.oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(255) NOT NULL,
    redirect_uri TEXT NOT NULL,
    user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for oauth_states
CREATE INDEX IF NOT EXISTS idx_oauth_states_state ON philotes.oauth_states(state);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON philotes.oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_states_user_id ON philotes.oauth_states(user_id);

-- Extend cloud_credentials table for OAuth tokens
DO $$
BEGIN
    -- Add credential_type column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'cloud_credentials'
                   AND column_name = 'credential_type') THEN
        ALTER TABLE philotes.cloud_credentials
        ADD COLUMN credential_type VARCHAR(20) NOT NULL DEFAULT 'manual';
    END IF;

    -- Add refresh_token_encrypted column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'cloud_credentials'
                   AND column_name = 'refresh_token_encrypted') THEN
        ALTER TABLE philotes.cloud_credentials
        ADD COLUMN refresh_token_encrypted BYTEA;
    END IF;

    -- Add token_expires_at column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'cloud_credentials'
                   AND column_name = 'token_expires_at') THEN
        ALTER TABLE philotes.cloud_credentials
        ADD COLUMN token_expires_at TIMESTAMPTZ;
    END IF;

    -- Add user_id column if not exists (for user-scoped credentials)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'cloud_credentials'
                   AND column_name = 'user_id') THEN
        ALTER TABLE philotes.cloud_credentials
        ADD COLUMN user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE;
    END IF;

    -- Make deployment_id nullable (credentials may be stored before deployment)
    ALTER TABLE philotes.cloud_credentials
    ALTER COLUMN deployment_id DROP NOT NULL;
END $$;

-- Create index for user_id on cloud_credentials
CREATE INDEX IF NOT EXISTS idx_cloud_credentials_user_id ON philotes.cloud_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_cloud_credentials_credential_type ON philotes.cloud_credentials(credential_type);

-- Add comments for documentation
COMMENT ON TABLE philotes.oauth_states IS 'Temporary OAuth state storage for PKCE flow (auto-expires after 10 minutes)';
COMMENT ON COLUMN philotes.oauth_states.state IS 'Random state parameter for CSRF protection';
COMMENT ON COLUMN philotes.oauth_states.code_verifier IS 'PKCE code verifier for token exchange';
COMMENT ON COLUMN philotes.oauth_states.redirect_uri IS 'Frontend URL to redirect after OAuth callback';
COMMENT ON COLUMN philotes.oauth_states.session_id IS 'Session ID for unauthenticated users';

COMMENT ON COLUMN philotes.cloud_credentials.credential_type IS 'Type of credential: oauth or manual';
COMMENT ON COLUMN philotes.cloud_credentials.refresh_token_encrypted IS 'Encrypted OAuth refresh token';
COMMENT ON COLUMN philotes.cloud_credentials.token_expires_at IS 'When the access token expires';

-- Grant permissions
GRANT ALL ON TABLE philotes.oauth_states TO philotes;

-- Function to cleanup expired OAuth states (run periodically)
CREATE OR REPLACE FUNCTION philotes.cleanup_expired_oauth_states()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM philotes.oauth_states
    WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
