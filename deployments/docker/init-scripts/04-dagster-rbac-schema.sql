-- Dagster RBAC Schema
-- Provides role and permission management for Dagster proxy authorization

-- Dagster role assignments
-- Maps users to pre-built Dagster roles (dagster-viewer, dagster-operator, dagster-admin)
CREATE TABLE IF NOT EXISTS dagster_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('dagster-viewer', 'dagster-operator', 'dagster-admin')),
    -- Optional resource pattern for scoped roles (e.g., dagster:jobs:etl_*:*)
    resource_pattern VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role, COALESCE(resource_pattern, ''))
);

-- Custom Dagster permissions
-- For fine-grained permissions beyond pre-built roles
-- Format: dagster:{resource_type}:{resource_name}:{action}
CREATE TABLE IF NOT EXISTS dagster_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(255) NOT NULL CHECK (permission LIKE 'dagster:%'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, permission)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_dagster_role_assignments_user
    ON dagster_role_assignments(user_id);

CREATE INDEX IF NOT EXISTS idx_dagster_role_assignments_role
    ON dagster_role_assignments(role);

CREATE INDEX IF NOT EXISTS idx_dagster_permissions_user
    ON dagster_permissions(user_id);

CREATE INDEX IF NOT EXISTS idx_dagster_permissions_permission
    ON dagster_permissions(permission);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_dagster_role_assignments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_dagster_role_assignments_updated_at ON dagster_role_assignments;
CREATE TRIGGER trigger_dagster_role_assignments_updated_at
    BEFORE UPDATE ON dagster_role_assignments
    FOR EACH ROW
    EXECUTE FUNCTION update_dagster_role_assignments_updated_at();

-- View to get all Dagster permissions for a user (roles + custom)
CREATE OR REPLACE VIEW user_dagster_permissions AS
SELECT
    u.id AS user_id,
    u.email,
    COALESCE(
        ARRAY_AGG(DISTINCT
            CASE dra.role
                WHEN 'dagster-viewer' THEN unnest(ARRAY[
                    'dagster:jobs:*:view',
                    'dagster:assets:*:view',
                    'dagster:schedules:*:view',
                    'dagster:sensors:*:view',
                    'dagster:runs:*:view'
                ])
                WHEN 'dagster-operator' THEN unnest(ARRAY[
                    'dagster:jobs:*:view',
                    'dagster:jobs:*:execute',
                    'dagster:assets:*:view',
                    'dagster:assets:*:materialize',
                    'dagster:schedules:*:view',
                    'dagster:sensors:*:view',
                    'dagster:runs:*:view'
                ])
                WHEN 'dagster-admin' THEN 'dagster:*:*:*'
                ELSE NULL
            END
        ) FILTER (WHERE dra.role IS NOT NULL),
        ARRAY[]::VARCHAR[]
    ) || COALESCE(
        ARRAY_AGG(DISTINCT dp.permission) FILTER (WHERE dp.permission IS NOT NULL),
        ARRAY[]::VARCHAR[]
    ) AS permissions
FROM users u
LEFT JOIN dagster_role_assignments dra ON u.id = dra.user_id
LEFT JOIN dagster_permissions dp ON u.id = dp.user_id
GROUP BY u.id, u.email;

-- Function to check if a user has a specific Dagster permission
CREATE OR REPLACE FUNCTION user_has_dagster_permission(
    p_user_id UUID,
    p_required_permission VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_permissions VARCHAR[];
    v_permission VARCHAR;
BEGIN
    -- Get user's permissions
    SELECT permissions INTO v_permissions
    FROM user_dagster_permissions
    WHERE user_id = p_user_id;

    IF v_permissions IS NULL THEN
        RETURN FALSE;
    END IF;

    -- Check for exact match or wildcard match
    FOREACH v_permission IN ARRAY v_permissions LOOP
        IF v_permission = p_required_permission THEN
            RETURN TRUE;
        END IF;

        -- Check wildcard patterns
        IF v_permission = 'dagster:*:*:*' THEN
            RETURN TRUE;
        END IF;

        -- TODO: Add more sophisticated pattern matching
    END LOOP;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions for the API user
GRANT SELECT, INSERT, UPDATE, DELETE ON dagster_role_assignments TO philotes;
GRANT SELECT, INSERT, UPDATE, DELETE ON dagster_permissions TO philotes;
GRANT SELECT ON user_dagster_permissions TO philotes;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO philotes;

-- Add some example data for development (optional - commented out for production)
-- INSERT INTO dagster_role_assignments (user_id, role)
-- SELECT id, 'dagster-admin'
-- FROM users
-- WHERE role = 'admin'
-- ON CONFLICT DO NOTHING;

COMMENT ON TABLE dagster_role_assignments IS 'Maps users to pre-built Dagster roles for RBAC';
COMMENT ON TABLE dagster_permissions IS 'Custom per-user Dagster permissions for fine-grained access control';
COMMENT ON VIEW user_dagster_permissions IS 'Aggregated view of all Dagster permissions for each user';
