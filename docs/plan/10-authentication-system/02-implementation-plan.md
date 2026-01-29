# Implementation Plan - Issue #10: Authentication System

## Summary

Implement API key authentication for programmatic access and JWT token support for dashboard sessions. This secures the management API and provides a foundation for future SSO/OIDC integration.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Request Flow                                │
│                                                                  │
│  Client ──► Middleware ──► Handler ──► Service ──► Repository   │
│             (auth.go)                   (auth.go)   (auth.go)   │
│                │                                                 │
│                ▼                                                 │
│         ┌──────────────┐                                        │
│         │ Auth Check   │                                        │
│         ├──────────────┤                                        │
│         │ 1. X-API-Key │                                        │
│         │ 2. Bearer JWT│                                        │
│         │ 3. Optional  │ (for health/metrics)                   │
│         └──────────────┘                                        │
└─────────────────────────────────────────────────────────────────┘
```

## Database Schema

### Tables to Create

```sql
-- Users table (for dashboard/JWT authentication)
CREATE TABLE philotes.users (
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

-- API Keys table
CREATE TABLE philotes.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES philotes.users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit log table
CREATE TABLE philotes.audit_logs (
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

-- Indexes
CREATE INDEX idx_api_keys_key_hash ON philotes.api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON philotes.api_keys(user_id);
CREATE INDEX idx_audit_logs_user_id ON philotes.audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON philotes.audit_logs(created_at);
```

### Roles and Permissions

| Role | Permissions |
|------|-------------|
| `admin` | Full access: create, read, update, delete all resources |
| `operator` | Manage pipelines: start, stop, create, update pipelines and sources |
| `viewer` | Read-only: view sources, pipelines, metrics |

Permissions array format:
```
["sources:read", "sources:write", "pipelines:read", "pipelines:write", "pipelines:control"]
```

## Files to Create

| File | Purpose | ~LOC |
|------|---------|------|
| `internal/api/models/auth.go` | User, APIKey, JWT claims models | 150 |
| `internal/api/repositories/user.go` | User CRUD operations | 200 |
| `internal/api/repositories/api_key.go` | API key CRUD operations | 200 |
| `internal/api/repositories/audit.go` | Audit log operations | 100 |
| `internal/api/services/auth.go` | Auth business logic (JWT, password) | 250 |
| `internal/api/services/api_key.go` | API key generation, validation | 200 |
| `internal/api/middleware/auth.go` | Authentication middleware | 200 |
| `internal/api/handlers/auth.go` | Login, logout, me endpoints | 150 |
| `internal/api/handlers/api_keys.go` | API key management endpoints | 150 |
| `deployments/docker/init-scripts/002_auth_tables.sql` | Database migrations | 60 |

**Total: ~1,660 LOC**

## Files to Modify

| File | Changes |
|------|---------|
| `internal/api/server.go` | Add auth middleware, register auth routes, add AuthService |
| `internal/config/config.go` | Add AuthConfig section |
| `cmd/philotes-api/main.go` | Initialize auth repositories and services |
| `go.mod` | Add `github.com/golang-jwt/jwt/v5` |

## Task Breakdown

### Phase 1: Foundation (Models, Config, Database)

1. **Add JWT dependency**
   - `go get github.com/golang-jwt/jwt/v5`

2. **Add AuthConfig to config.go**
   ```go
   Auth: AuthConfig{
       Enabled:         getBoolEnv("PHILOTES_AUTH_ENABLED", false),
       JWTSecret:       getEnv("PHILOTES_AUTH_JWT_SECRET", ""),
       JWTExpiration:   getDurationEnv("PHILOTES_AUTH_JWT_EXPIRATION", 24*time.Hour),
       APIKeyPrefix:    getEnv("PHILOTES_AUTH_API_KEY_PREFIX", "pk_"),
       BCryptCost:      getIntEnv("PHILOTES_AUTH_BCRYPT_COST", 12),
   }
   ```

3. **Create auth models** (`internal/api/models/auth.go`)
   - User struct
   - APIKey struct
   - JWTClaims struct
   - CreateUserRequest, LoginRequest, etc.

4. **Create database migration** (`deployments/docker/init-scripts/002_auth_tables.sql`)

### Phase 2: Repositories

5. **Create UserRepository** (`internal/api/repositories/user.go`)
   - Create, GetByID, GetByEmail, Update, Delete, List
   - UpdateLastLogin

6. **Create APIKeyRepository** (`internal/api/repositories/api_key.go`)
   - Create, GetByID, GetByHash, List, Delete
   - UpdateLastUsed

7. **Create AuditRepository** (`internal/api/repositories/audit.go`)
   - Create, List (with filters)

### Phase 3: Services

8. **Create AuthService** (`internal/api/services/auth.go`)
   - Login (validate password, generate JWT)
   - ValidateJWT
   - HashPassword / VerifyPassword
   - GetCurrentUser

9. **Create APIKeyService** (`internal/api/services/api_key.go`)
   - GenerateAPIKey (returns plaintext key once)
   - ValidateAPIKey (by hash lookup)
   - RevokeAPIKey
   - ListAPIKeys

### Phase 4: Middleware

10. **Create auth middleware** (`internal/api/middleware/auth.go`)
    - `Authenticate()` - Extracts credentials, sets user in context
    - `RequireAuth()` - Returns 401 if not authenticated
    - `RequirePermission(perm string)` - Returns 403 if missing permission
    - Support both `X-API-Key` and `Authorization: Bearer` headers

### Phase 5: Handlers

11. **Create auth handlers** (`internal/api/handlers/auth.go`)
    - `POST /api/v1/auth/login` - Login with email/password
    - `POST /api/v1/auth/logout` - Logout (optional, for JWT blacklist)
    - `GET /api/v1/auth/me` - Get current user

12. **Create API key handlers** (`internal/api/handlers/api_keys.go`)
    - `POST /api/v1/api-keys` - Create API key
    - `GET /api/v1/api-keys` - List API keys
    - `DELETE /api/v1/api-keys/:id` - Revoke API key

### Phase 6: Integration

13. **Update server.go**
    - Add AuthService to ServerConfig
    - Apply `Authenticate()` middleware globally
    - Apply `RequireAuth()` to protected routes (v1 API group)
    - Keep health/metrics public
    - Register auth and API key routes

14. **Update main.go**
    - Initialize auth repositories
    - Initialize auth services
    - Pass to ServerConfig

15. **Create bootstrap admin user** (optional CLI command or env var)
    - `PHILOTES_AUTH_ADMIN_EMAIL` / `PHILOTES_AUTH_ADMIN_PASSWORD`

### Phase 7: Testing

16. **Manual verification**
    - Test login flow
    - Test API key generation
    - Test protected routes with API key
    - Test protected routes with JWT
    - Test unauthorized access returns 401
    - Test audit logging

## API Endpoints

### Auth Endpoints

```
POST /api/v1/auth/login
Body: { "email": "...", "password": "..." }
Response: { "token": "jwt...", "expires_at": "...", "user": {...} }

GET /api/v1/auth/me
Headers: Authorization: Bearer <jwt>
Response: { "user": {...} }
```

### API Key Endpoints

```
POST /api/v1/api-keys
Headers: Authorization: Bearer <jwt>
Body: { "name": "My Key", "permissions": ["sources:read", "pipelines:read"] }
Response: { "api_key": {...}, "key": "pk_live_abc123..." } // key shown only once

GET /api/v1/api-keys
Headers: Authorization: Bearer <jwt>
Response: { "api_keys": [...] }

DELETE /api/v1/api-keys/:id
Headers: Authorization: Bearer <jwt>
Response: 204 No Content
```

## API Key Format

```
pk_live_[32 random chars]
│  │    └── Random suffix (base62 encoded)
│  └── Environment indicator
└── Philotes key prefix
```

Example: `pk_live_a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6`

- First 8 chars (`pk_live_`) stored as `key_prefix` for identification
- Full key hashed with SHA256 and stored as `key_hash`
- Plaintext key shown only once at creation time

## Security Considerations

1. **Password Storage**: bcrypt with cost factor 12
2. **API Key Storage**: SHA256 hash, never store plaintext
3. **JWT**: HS256 algorithm, short expiration (24h default)
4. **Rate Limiting**: Existing rate limiter applies to all endpoints
5. **Audit Logging**: All auth events logged with IP and user agent

## Configuration Environment Variables

```bash
# Enable/disable auth (disabled by default for development)
PHILOTES_AUTH_ENABLED=true

# JWT signing secret (required when auth enabled)
PHILOTES_AUTH_JWT_SECRET=your-secret-key-min-32-chars

# JWT token expiration
PHILOTES_AUTH_JWT_EXPIRATION=24h

# API key prefix
PHILOTES_AUTH_API_KEY_PREFIX=pk_

# BCrypt cost factor
PHILOTES_AUTH_BCRYPT_COST=12

# Bootstrap admin (created on startup if doesn't exist)
PHILOTES_AUTH_ADMIN_EMAIL=admin@example.com
PHILOTES_AUTH_ADMIN_PASSWORD=changeme
```

## Verification Checklist

- [ ] `go build ./...` succeeds
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Login returns JWT token
- [ ] API key creation returns plaintext key once
- [ ] Protected routes return 401 without auth
- [ ] Protected routes work with valid API key
- [ ] Protected routes work with valid JWT
- [ ] Audit log captures auth events
- [ ] Docker compose starts with auth tables created
