-- OIDC/SSO Schema for Philotes
-- This script creates the tables required for OIDC/SSO authentication

-- OIDC Providers table stores configured identity providers
CREATE TABLE IF NOT EXISTS philotes.oidc_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    issuer_url TEXT NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT ARRAY['openid', 'profile', 'email'],
    groups_claim VARCHAR(100) DEFAULT 'groups',
    role_mapping JSONB NOT NULL DEFAULT '{}',
    default_role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    enabled BOOLEAN NOT NULL DEFAULT true,
    auto_create_users BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for oidc_providers
CREATE INDEX IF NOT EXISTS idx_oidc_providers_name ON philotes.oidc_providers(name);
CREATE INDEX IF NOT EXISTS idx_oidc_providers_enabled ON philotes.oidc_providers(enabled);
CREATE INDEX IF NOT EXISTS idx_oidc_providers_provider_type ON philotes.oidc_providers(provider_type);

-- OIDC States table stores temporary state during OIDC flow
CREATE TABLE IF NOT EXISTS philotes.oidc_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL UNIQUE,
    nonce VARCHAR(255) NOT NULL,
    code_verifier VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL REFERENCES philotes.oidc_providers(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Indexes for oidc_states
CREATE INDEX IF NOT EXISTS idx_oidc_states_state ON philotes.oidc_states(state);
CREATE INDEX IF NOT EXISTS idx_oidc_states_expires_at ON philotes.oidc_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oidc_states_provider_id ON philotes.oidc_states(provider_id);

-- Alter users table to add OIDC fields
DO $$
BEGIN
    -- Add oidc_provider_id column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'users'
                   AND column_name = 'oidc_provider_id') THEN
        ALTER TABLE philotes.users
        ADD COLUMN oidc_provider_id UUID REFERENCES philotes.oidc_providers(id) ON DELETE SET NULL;
    END IF;

    -- Add oidc_subject column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'users'
                   AND column_name = 'oidc_subject') THEN
        ALTER TABLE philotes.users
        ADD COLUMN oidc_subject VARCHAR(255);
    END IF;

    -- Add oidc_groups column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'philotes'
                   AND table_name = 'users'
                   AND column_name = 'oidc_groups') THEN
        ALTER TABLE philotes.users
        ADD COLUMN oidc_groups TEXT[] DEFAULT '{}';
    END IF;

    -- Make password_hash nullable for OIDC-only users
    ALTER TABLE philotes.users
    ALTER COLUMN password_hash DROP NOT NULL;
END $$;

-- Create unique index on oidc_provider_id + oidc_subject combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_identity
    ON philotes.users(oidc_provider_id, oidc_subject)
    WHERE oidc_provider_id IS NOT NULL AND oidc_subject IS NOT NULL;

-- Create index for OIDC lookups
CREATE INDEX IF NOT EXISTS idx_users_oidc_provider_id ON philotes.users(oidc_provider_id);

-- Add comments for documentation
COMMENT ON TABLE philotes.oidc_providers IS 'Configured OIDC identity providers for SSO';
COMMENT ON TABLE philotes.oidc_states IS 'Temporary OIDC state storage for authorization flow';

COMMENT ON COLUMN philotes.oidc_providers.name IS 'Unique identifier for the provider (used in URLs)';
COMMENT ON COLUMN philotes.oidc_providers.display_name IS 'Human-readable name shown in UI';
COMMENT ON COLUMN philotes.oidc_providers.provider_type IS 'Type of provider: google, okta, azure_ad, auth0, generic';
COMMENT ON COLUMN philotes.oidc_providers.issuer_url IS 'OIDC issuer URL (discovery endpoint base)';
COMMENT ON COLUMN philotes.oidc_providers.client_secret_encrypted IS 'AES-256-GCM encrypted client secret';
COMMENT ON COLUMN philotes.oidc_providers.scopes IS 'OAuth scopes to request';
COMMENT ON COLUMN philotes.oidc_providers.groups_claim IS 'JWT claim containing user groups';
COMMENT ON COLUMN philotes.oidc_providers.role_mapping IS 'JSON mapping of groups to roles';
COMMENT ON COLUMN philotes.oidc_providers.default_role IS 'Default role for users without group mapping';
COMMENT ON COLUMN philotes.oidc_providers.auto_create_users IS 'Automatically create users on first login';

COMMENT ON COLUMN philotes.oidc_states.state IS 'Random state parameter for CSRF protection';
COMMENT ON COLUMN philotes.oidc_states.nonce IS 'Nonce for ID token replay protection';
COMMENT ON COLUMN philotes.oidc_states.code_verifier IS 'PKCE code verifier for token exchange';

COMMENT ON COLUMN philotes.users.oidc_provider_id IS 'OIDC provider used for authentication (NULL for local users)';
COMMENT ON COLUMN philotes.users.oidc_subject IS 'OIDC subject identifier (unique per provider)';
COMMENT ON COLUMN philotes.users.oidc_groups IS 'Groups from OIDC provider';

-- Grant permissions
GRANT ALL ON TABLE philotes.oidc_providers TO philotes;
GRANT ALL ON TABLE philotes.oidc_states TO philotes;

-- Function to cleanup expired OIDC states (run periodically)
CREATE OR REPLACE FUNCTION philotes.cleanup_expired_oidc_states()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM philotes.oidc_states
    WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
