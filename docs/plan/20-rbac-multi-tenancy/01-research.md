# Research Findings - Issue #20: RBAC and Multi-Tenancy

## Existing Authentication Infrastructure

### User Model (`internal/api/models/auth.go`)
- User struct with: ID, Email, Name, Role, IsActive, LastLoginAt, OIDCProviderID, OIDCSubject, OIDCGroups
- **Three predefined roles**: `admin`, `operator`, `viewer`
- **Permission constants** already defined:
  - `sources:read`, `sources:write`
  - `pipelines:read`, `pipelines:write`, `pipelines:control`
  - `api-keys:read`, `api-keys:write`
  - `users:read`, `users:write`
  - `scaling:read`, `scaling:write`
  - `alerts:read`, `alerts:write`
- **RolePermissions map** maps roles to default permission sets

### Auth Context (`internal/api/models/auth.go`)
- JWTClaims include UserID, Email, Role, and Permissions
- AuthContext struct holds User, APIKey, Permissions, IsAPIKey flag
- `HasPermission()` method for checking individual permissions

### Authentication Middleware (`internal/api/middleware/auth.go`)
- `Authenticate()`: extracts credentials from headers
- `RequireAuth()`: enforces authentication
- `RequirePermission()`: checks specific permissions
- Auth context stored in gin.Context with `AuthContextKey`

### Database Schema (`deployments/docker/init-scripts/08-auth-schema.sql`)
- `users` table: id, email, password_hash, name, role, is_active, last_login_at
- `api_keys` table: id, user_id, name, key_prefix, key_hash, permissions (TEXT[])
- `audit_logs` table: id, user_id, api_key_id, action, resource_type, resource_id

## Resources Requiring Tenant Scoping

| Resource | Model File | Repository | Handler |
|----------|------------|------------|---------|
| Sources | `models/source.go` | `repositories/source.go` | `handlers/sources.go` |
| Pipelines | `models/pipeline.go` | `repositories/pipeline.go` | `handlers/pipelines.go` |
| Alerts | `models/alert.go` | `repositories/alert_repository.go` | `handlers/alerts.go` |
| Scaling Policies | `models/scaling.go` | `repositories/scaling.go` | `handlers/scaling.go` |
| API Keys | `models/auth.go` | `repositories/api_key.go` | `handlers/api_keys.go` |

**Current State**: None have tenant_id field - all resources are global.

## Key Patterns to Follow

### Middleware Pattern
- Gin middleware functions returning `gin.HandlerFunc`
- Store context values in gin.Context
- Chain middleware in `server.go`

### Service Layer Pattern
- Service takes repository + config + logger
- Returns domain models
- Handles validation and business logic

### Repository Pattern
- Take only `*sql.DB`
- Define error variables at package level
- Return domain models via `toModel()` conversion

## Summary

AUTH-001 infrastructure is solid. Permission constants, role definitions, and middleware patterns already exist. Primary work is:

1. Add tenant concept to data models and database
2. Create tenant management service and handlers
3. Extend resource repositories with tenant-scoped queries
4. Update handlers to include tenant context
5. Add tenant extraction to auth middleware
