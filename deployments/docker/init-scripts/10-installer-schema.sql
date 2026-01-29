-- Installer Schema for Philotes
-- This script creates the tables required for the one-click cloud installer

-- Deployments table stores deployment configurations and status
CREATE TABLE IF NOT EXISTS philotes.deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES philotes.users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    environment VARCHAR(50) NOT NULL DEFAULT 'production',
    config JSONB NOT NULL DEFAULT '{}',
    outputs JSONB,
    pulumi_stack_name VARCHAR(255),
    pulumi_org VARCHAR(255),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for deployments
CREATE INDEX IF NOT EXISTS idx_deployments_user_id ON philotes.deployments(user_id);
CREATE INDEX IF NOT EXISTS idx_deployments_provider ON philotes.deployments(provider);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON philotes.deployments(status);
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON philotes.deployments(created_at);

-- Deployment logs table stores real-time deployment progress
CREATE TABLE IF NOT EXISTS philotes.deployment_logs (
    id SERIAL PRIMARY KEY,
    deployment_id UUID NOT NULL REFERENCES philotes.deployments(id) ON DELETE CASCADE,
    level VARCHAR(10) NOT NULL DEFAULT 'info',
    step VARCHAR(100),
    message TEXT NOT NULL,
    details JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for deployment logs
CREATE INDEX IF NOT EXISTS idx_deployment_logs_deployment_id ON philotes.deployment_logs(deployment_id);
CREATE INDEX IF NOT EXISTS idx_deployment_logs_timestamp ON philotes.deployment_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_deployment_logs_level ON philotes.deployment_logs(level);

-- Cloud credentials table stores encrypted provider credentials (temporary during deployment)
CREATE TABLE IF NOT EXISTS philotes.cloud_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES philotes.deployments(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    credentials_encrypted BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for cloud credentials
CREATE INDEX IF NOT EXISTS idx_cloud_credentials_deployment_id ON philotes.cloud_credentials(deployment_id);
CREATE INDEX IF NOT EXISTS idx_cloud_credentials_expires_at ON philotes.cloud_credentials(expires_at);

-- Add comments for documentation
COMMENT ON TABLE philotes.deployments IS 'Cloud infrastructure deployments created through the installer';
COMMENT ON TABLE philotes.deployment_logs IS 'Real-time log entries for deployment progress tracking';
COMMENT ON TABLE philotes.cloud_credentials IS 'Encrypted cloud provider credentials (temporary, auto-deleted after deployment)';

COMMENT ON COLUMN philotes.deployments.provider IS 'Cloud provider: hetzner, ovh, scaleway, exoscale, contabo';
COMMENT ON COLUMN philotes.deployments.size IS 'Deployment size: small, medium, large';
COMMENT ON COLUMN philotes.deployments.status IS 'Deployment status: pending, provisioning, configuring, deploying, verifying, completed, failed, cancelled';
COMMENT ON COLUMN philotes.deployments.config IS 'JSON configuration including domain, SSH key, chart version, etc.';
COMMENT ON COLUMN philotes.deployments.outputs IS 'JSON outputs including control plane IP, kubeconfig, load balancer IP, etc.';
COMMENT ON COLUMN philotes.deployment_logs.level IS 'Log level: debug, info, warn, error';
COMMENT ON COLUMN philotes.deployment_logs.step IS 'Current deployment step for progress tracking';

-- Grant permissions
GRANT ALL ON TABLE philotes.deployments TO philotes;
GRANT ALL ON TABLE philotes.deployment_logs TO philotes;
GRANT ALL ON TABLE philotes.cloud_credentials TO philotes;
GRANT USAGE, SELECT ON SEQUENCE philotes.deployment_logs_id_seq TO philotes;

-- Function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION philotes.update_deployment_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS trigger_update_deployment_timestamp ON philotes.deployments;
CREATE TRIGGER trigger_update_deployment_timestamp
    BEFORE UPDATE ON philotes.deployments
    FOR EACH ROW
    EXECUTE FUNCTION philotes.update_deployment_timestamp();

-- Function to cleanup expired credentials (run periodically)
CREATE OR REPLACE FUNCTION philotes.cleanup_expired_credentials()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM philotes.cloud_credentials
    WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;
