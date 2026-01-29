# Research Findings - Issue #10: Authentication System

## 1. API Framework & Architecture

**Framework:** Gin Web Framework (v1.10.1)
- **Server:** `/internal/api/server.go`
- **Entry point:** `/cmd/philotes-api/main.go`

**API Routes Structure:**
```
/health, /health/live, /health/ready  - Health checks (no auth)
/metrics                               - Prometheus (no auth)
/api/v1/sources/*                     - Sources CRUD
/api/v1/pipelines/*                   - Pipelines CRUD
/api/v1/scaling/*                     - Scaling policies
/api/v1/alerts/*                      - Alerts
```

## 2. Current Middleware Stack

Located in `/internal/api/middleware/`:
1. `RequestID()` - Generates/propagates X-Request-ID
2. `Recovery()` - Panic recovery
3. `Metrics()` - Prometheus instrumentation
4. `Logger()` - Structured logging (slog)
5. `CORS()` - CORS headers (already allows `Authorization`)
6. `RateLimiter()` - Per-client rate limiting

**No authentication middleware exists yet.**

## 3. Database & Models

**Database:** PostgreSQL via `pgx/v5`

**Existing tables:**
- `philotes.sources`
- `philotes.pipelines`
- `philotes.table_mappings`

**No user/auth tables exist.**

**Data Access Pattern:**
- Repository layer → Service layer → Handlers
- Models in `/internal/api/models/`
- Typed errors: `ValidationError`, `NotFoundError`, `ConflictError`

## 4. Configuration

**Pattern:** Environment variables via `/internal/config/config.go`
- Supports HashiCorp Vault for secrets
- Config sections: `APIConfig`, `DatabaseConfig`, `VaultConfig`

## 5. Dependencies Available

From `go.mod`:
- `golang.org/x/crypto` v0.46.0 (available for password hashing)
- `github.com/google/uuid` v1.6.0 (for UUIDs)
- `golang.org/x/time` v0.14.0 (for rate limiting)

**Need to add:** `github.com/golang-jwt/jwt/v5` for JWT handling

## 6. Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| API Key Format | Prefix-based (`pk_live_...`) | Easy identification |
| Key Storage | SHA256 hash | Never store plaintext |
| JWT Algorithm | HS256 | Simple, sufficient for single-service |
| Permissions | String array | Flexible, simple start |
| Rate Limiting | Per-API-key | Extend existing limiter |

## 7. Files to Create

| File | Purpose |
|------|---------|
| `internal/api/models/auth.go` | User, APIKey, JWT claims |
| `internal/api/repositories/user.go` | User CRUD |
| `internal/api/repositories/api_key.go` | API key CRUD |
| `internal/api/repositories/audit.go` | Audit logging |
| `internal/api/services/auth.go` | Auth business logic |
| `internal/api/services/api_key.go` | API key management |
| `internal/api/middleware/auth.go` | Auth middleware |
| `internal/api/handlers/auth.go` | Login/logout handlers |
| `internal/api/handlers/api_keys.go` | API key endpoints |

## 8. Files to Modify

| File | Changes |
|------|---------|
| `internal/api/server.go` | Add auth middleware, register routes |
| `internal/config/config.go` | Add auth config section |
| `cmd/philotes-api/main.go` | Initialize auth services |
| `deployments/docker/init-scripts/` | Add auth tables |
| `go.mod` | Add JWT dependency |
