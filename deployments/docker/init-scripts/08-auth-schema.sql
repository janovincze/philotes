-- Authentication Schema for Philotes
-- This script creates the tables required for authentication and audit logging

-- Users table stores registered users for dashboard/JWT authentication
CREATE TABLE IF NOT EXISTS philotes.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for users
CREATE INDEX IF NOT EXISTS idx_users_email ON philotes.users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON philotes.users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON philotes.users(is_active);

-- API Keys table stores API keys for programmatic access
CREATE TABLE IF NOT EXISTS philotes.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for API keys
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON philotes.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON philotes.api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON philotes.api_keys(is_active);

-- Audit log table tracks authentication events
CREATE TABLE IF NOT EXISTS philotes.audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES philotes.users(id) ON DELETE SET NULL,
    api_key_id UUID REFERENCES philotes.api_keys(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON philotes.audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_api_key_id ON philotes.audit_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON philotes.audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON philotes.audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON philotes.audit_logs(resource_type, resource_id);

-- Add comments for documentation
COMMENT ON TABLE philotes.users IS 'Registered users for authentication';
COMMENT ON TABLE philotes.api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE philotes.audit_logs IS 'Audit trail for authentication and authorization events';

COMMENT ON COLUMN philotes.users.role IS 'User role: admin, operator, or viewer';
COMMENT ON COLUMN philotes.api_keys.key_prefix IS 'First 8 characters of the API key for identification';
COMMENT ON COLUMN philotes.api_keys.key_hash IS 'SHA256 hash of the full API key';
COMMENT ON COLUMN philotes.api_keys.permissions IS 'Array of permission strings';
COMMENT ON COLUMN philotes.audit_logs.action IS 'Type of action: login, login_failed, logout, api_key_created, etc.';

-- Grant permissions
GRANT ALL ON TABLE philotes.users TO philotes;
GRANT ALL ON TABLE philotes.api_keys TO philotes;
GRANT ALL ON TABLE philotes.audit_logs TO philotes;
