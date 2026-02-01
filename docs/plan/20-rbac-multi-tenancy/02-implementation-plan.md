# Implementation Plan: RBAC and Multi-Tenancy Foundation

## Summary

Implement role-based access control (RBAC) and multi-tenancy for Philotes. The existing AUTH-001 infrastructure provides a solid foundation with user roles (admin/operator/viewer), granular permissions, and middleware patterns. This implementation adds tenant isolation and extends the authorization system.

## Approach

1. **Database-First** - Create tenant and role tables, add tenant_id to existing resources
2. **Models & Repositories** - Add Tenant model and extend existing models/repos
3. **Middleware** - Add tenant extraction and isolation middleware
4. **Services** - Create tenant management service
5. **Handlers** - Create tenant API endpoints, extend existing handlers
6. **Configuration** - Add multi-tenancy config options

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `deployments/docker/init-scripts/17-rbac-multi-tenancy-schema.sql` | Database schema for tenants and roles | ~150 |
| `internal/api/models/tenant.go` | Tenant, TenantUser, Role models | ~200 |
| `internal/api/repositories/tenant.go` | Tenant CRUD repository | ~300 |
| `internal/api/repositories/role.go` | Custom role repository | ~200 |
| `internal/api/services/tenant.go` | Tenant management service | ~400 |
| `internal/api/handlers/tenant.go` | Tenant API handlers | ~350 |
| `internal/api/middleware/tenant.go` | Tenant extraction/isolation middleware | ~150 |

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/api/models/auth.go` | Extend AuthContext with TenantID | ~30 |
| `internal/api/models/source.go` | Add TenantID field | ~10 |
| `internal/api/models/pipeline.go` | Add TenantID field | ~10 |
| `internal/api/repositories/source.go` | Add tenant-scoped queries | ~50 |
| `internal/api/repositories/pipeline.go` | Add tenant-scoped queries | ~50 |
| `internal/api/handlers/sources.go` | Add tenant context to operations | ~40 |
| `internal/api/handlers/pipelines.go` | Add tenant context to operations | ~40 |
| `internal/api/middleware/auth.go` | Populate tenant in auth context | ~30 |
| `internal/api/server.go` | Register tenant routes | ~30 |
| `internal/config/config.go` | Add MultiTenancyConfig | ~40 |

## Database Schema

### New Tables

```sql
-- Tenants table
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    owner_user_id UUID REFERENCES users(id),
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User-Tenant memberships with role
CREATE TABLE tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    custom_permissions TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

-- Custom roles per tenant (optional)
CREATE TABLE tenant_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);
```

### Modify Existing Tables

```sql
-- Add tenant_id to sources
ALTER TABLE sources ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_sources_tenant_id ON sources(tenant_id);

-- Add tenant_id to pipelines
ALTER TABLE pipelines ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_pipelines_tenant_id ON pipelines(tenant_id);

-- Add tenant_id to api_keys
ALTER TABLE api_keys ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);

-- Add tenant_id to audit_logs
ALTER TABLE audit_logs ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL;
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
```

## API Endpoints

### Tenant Management

| Method | Path | Description | Permission |
|--------|------|-------------|------------|
| `GET` | `/api/v1/tenants` | List tenants (user's tenants) | authenticated |
| `POST` | `/api/v1/tenants` | Create tenant | `tenants:write` |
| `GET` | `/api/v1/tenants/:id` | Get tenant details | member |
| `PUT` | `/api/v1/tenants/:id` | Update tenant | tenant admin |
| `DELETE` | `/api/v1/tenants/:id` | Delete tenant | owner |

### Tenant Members

| Method | Path | Description | Permission |
|--------|------|-------------|------------|
| `GET` | `/api/v1/tenants/:id/members` | List members | member |
| `POST` | `/api/v1/tenants/:id/members` | Add member | tenant admin |
| `PUT` | `/api/v1/tenants/:id/members/:user_id` | Update member role | tenant admin |
| `DELETE` | `/api/v1/tenants/:id/members/:user_id` | Remove member | tenant admin |

### Custom Roles (optional)

| Method | Path | Description | Permission |
|--------|------|-------------|------------|
| `GET` | `/api/v1/tenants/:id/roles` | List custom roles | member |
| `POST` | `/api/v1/tenants/:id/roles` | Create custom role | tenant admin |
| `PUT` | `/api/v1/tenants/:id/roles/:role_id` | Update role | tenant admin |
| `DELETE` | `/api/v1/tenants/:id/roles/:role_id` | Delete role | tenant admin |

## New Permissions

```go
const (
    PermissionTenantsRead  = "tenants:read"
    PermissionTenantsWrite = "tenants:write"
    PermissionMembersRead  = "members:read"
    PermissionMembersWrite = "members:write"
    PermissionRolesRead    = "roles:read"
    PermissionRolesWrite   = "roles:write"
)
```

## Middleware Flow

```
Request → Authenticate → ExtractTenant → RequireTenant → RequirePermission → Handler
```

1. **Authenticate**: Extract user from JWT/API key (existing)
2. **ExtractTenant**: Get tenant from header `X-Tenant-ID` or JWT claims
3. **RequireTenant**: Verify user is member of tenant
4. **RequirePermission**: Check permission within tenant context

## Configuration

```go
type MultiTenancyConfig struct {
    // Enabled enables multi-tenancy mode
    Enabled bool

    // DefaultTenantID is the default tenant for single-tenant mode
    DefaultTenantID string

    // AutoCreateTenant creates tenant on first user signup
    AutoCreateTenant bool

    // AllowCrossTenantAccess allows super-admins to access all tenants
    AllowCrossTenantAccess bool
}
```

## Task Order

1. Create database migration schema
2. Add MultiTenancyConfig to config.go
3. Create Tenant model and request/response types
4. Create TenantRepository
5. Create TenantService
6. Create tenant middleware (ExtractTenant, RequireTenant)
7. Create TenantHandler
8. Extend AuthContext with TenantID
9. Extend Source/Pipeline models with TenantID
10. Extend Source/Pipeline repositories with tenant-scoped queries
11. Update Source/Pipeline handlers to use tenant context
12. Register routes in server.go
13. Create default tenant migration for existing data
14. Run lint and tests

## Backward Compatibility

- Multi-tenancy is **disabled by default**
- When disabled, a "default" system tenant is used
- Existing data migrated to default tenant
- Single-tenant deployments work unchanged

## Verification

1. `go build ./...` - Verify compilation
2. `make lint` - Verify code quality
3. `make test` - Verify tests pass
4. Manual test: Create tenant, add member, verify isolation
5. Manual test: Verify existing resources work with default tenant

## Estimate

~1,800 LOC total (reduced from 10,000 LOC estimate due to existing infrastructure)
