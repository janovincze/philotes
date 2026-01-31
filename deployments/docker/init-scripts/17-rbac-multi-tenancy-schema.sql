-- 17-rbac-multi-tenancy-schema.sql
-- RBAC and Multi-Tenancy Foundation (Issue #20)
-- This migration adds tenant isolation and extends the authorization system.

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index on tenants
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_owner_user_id ON tenants(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_tenants_is_active ON tenants(is_active);

-- Create tenant_members table (user-tenant memberships with role)
CREATE TABLE IF NOT EXISTS tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    custom_permissions TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

-- Create indexes on tenant_members
CREATE INDEX IF NOT EXISTS idx_tenant_members_tenant_id ON tenant_members(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_user_id ON tenant_members(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_role ON tenant_members(role);

-- Create tenant_roles table (custom roles per tenant)
CREATE TABLE IF NOT EXISTS tenant_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Create indexes on tenant_roles
CREATE INDEX IF NOT EXISTS idx_tenant_roles_tenant_id ON tenant_roles(tenant_id);

-- Add tenant_id to sources table
ALTER TABLE sources ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_sources_tenant_id ON sources(tenant_id);

-- Add tenant_id to pipelines table
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_pipelines_tenant_id ON pipelines(tenant_id);

-- Add tenant_id to api_keys table
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);

-- Add tenant_id to audit_logs table
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id);

-- Create default system tenant for backward compatibility
-- This ensures existing data continues to work when multi-tenancy is disabled
INSERT INTO tenants (id, name, slug, is_active, settings)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default',
    'default',
    TRUE,
    '{"system": true}'::jsonb
)
ON CONFLICT (slug) DO NOTHING;

-- Migrate existing data to default tenant
UPDATE sources SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE pipelines SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE api_keys SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;

-- Add existing users as members of the default tenant
INSERT INTO tenant_members (tenant_id, user_id, role)
SELECT '00000000-0000-0000-0000-000000000001', id, role
FROM users
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- Update the default tenant owner to the first admin user (if any)
UPDATE tenants
SET owner_user_id = (
    SELECT id FROM users WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1
)
WHERE slug = 'default' AND owner_user_id IS NULL;

-- Create trigger to update updated_at on tenants
CREATE OR REPLACE FUNCTION update_tenants_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_tenants_updated_at ON tenants;
CREATE TRIGGER trigger_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_tenants_updated_at();

-- Create trigger to update updated_at on tenant_members
CREATE OR REPLACE FUNCTION update_tenant_members_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_tenant_members_updated_at ON tenant_members;
CREATE TRIGGER trigger_tenant_members_updated_at
    BEFORE UPDATE ON tenant_members
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_members_updated_at();

-- Create trigger to update updated_at on tenant_roles
CREATE OR REPLACE FUNCTION update_tenant_roles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_tenant_roles_updated_at ON tenant_roles;
CREATE TRIGGER trigger_tenant_roles_updated_at
    BEFORE UPDATE ON tenant_roles
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_roles_updated_at();

-- Add constraint to ensure role is valid
ALTER TABLE tenant_members DROP CONSTRAINT IF EXISTS tenant_members_role_check;
ALTER TABLE tenant_members ADD CONSTRAINT tenant_members_role_check
    CHECK (role IN ('admin', 'operator', 'viewer', 'custom'));

-- Comments for documentation
COMMENT ON TABLE tenants IS 'Multi-tenant organizations that own resources';
COMMENT ON TABLE tenant_members IS 'User memberships in tenants with assigned roles';
COMMENT ON TABLE tenant_roles IS 'Custom roles defined per tenant with specific permissions';
COMMENT ON COLUMN tenants.slug IS 'URL-friendly unique identifier for the tenant';
COMMENT ON COLUMN tenants.settings IS 'JSON settings for tenant-specific configuration';
COMMENT ON COLUMN tenant_members.role IS 'Role within tenant: admin, operator, viewer, or custom';
COMMENT ON COLUMN tenant_members.custom_permissions IS 'Additional permissions beyond the role defaults';
COMMENT ON COLUMN tenant_roles.permissions IS 'List of permission strings this role grants';
