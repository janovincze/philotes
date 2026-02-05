# Research: Issue #103 - Dagster RBAC Proxy

## Existing Authentication Architecture

### Auth Middleware (`internal/api/middleware/auth.go`)
- **Authenticate()**: Extracts credentials (JWT or API key) and sets AuthContext
- **RequireAuth()**: Enforces authentication requirement
- **RequirePermission(permission)**: Checks specific permission
- Supports both `X-API-Key` header and `Authorization: Bearer` token
- API keys use configurable prefix (default: `pk_`)

### Auth Models (`internal/api/models/auth.go`)
- **AuthContext**: Holds User, APIKey, Permissions, TenantID, TenantRole
- **HasPermission(string)**: Simple string matching (no wildcards currently)
- **UserRole**: Admin, Operator, Viewer with predefined permissions
- **Permission format**: `resource:action` (e.g., `sources:read`, `pipelines:write`)
- **JWTClaims**: Contains UserID, Email, Role, Permissions array

### Audit Logging (`internal/api/models/auth.go`)
- **AuditLog** model with: UserID, APIKeyID, Action, ResourceType, ResourceID, IPAddress, UserAgent, Details
- Predefined actions: login, login_failed, logout, api_key_created, unauthorized, forbidden, etc.
- AuditRepository provides Create() and List() methods

## Command Pattern (`cmd/philotes-api/main.go`)

Service bootstrap follows this pattern:
1. Setup structured logging with slog.JSONHandler
2. Load configuration via config.Load()
3. Initialize secret provider (Vault integration)
4. Open database connection
5. Create repositories (source, pipeline, user, apiKey, audit)
6. Create services (source, pipeline, auth, apiKey)
7. Setup health manager with component checkers
8. Create ServerConfig and start server
9. Handle graceful shutdown on SIGINT/SIGTERM

## Key Packages to Reuse

| Package | Import Path | Purpose |
|---------|-------------|---------|
| Gin Router | `github.com/gin-gonic/gin` | HTTP routing |
| JWT | `github.com/golang-jwt/jwt/v5` | Token validation |
| UUID | `github.com/google/uuid` | ID generation |
| pgx | `github.com/jackc/pgx/v5/stdlib` | PostgreSQL driver |
| slog | `log/slog` (stdlib) | Structured logging |
| httputil | `net/http/httputil` (stdlib) | ReverseProxy |
| websocket | `github.com/gorilla/websocket` | WebSocket support |

## Recommended GraphQL Packages

1. **github.com/vektah/gqlparser/v2** - Comprehensive GraphQL parser
   - Parse queries/mutations into AST
   - Validate against schema
   - Well-maintained, used by gqlgen

2. **encoding/json** (stdlib) - Parse GraphQL request body
   - Request format: `{"query": "...", "variables": {...}, "operationName": "..."}`

## File Structure for New Service

```
cmd/philotes-dagster-proxy/
└── main.go

internal/dagster-proxy/
├── config/
│   └── config.go           # Proxy-specific configuration
├── handler/
│   ├── proxy.go            # Main HTTP proxy handler
│   └── websocket.go        # WebSocket upgrade & proxy
├── graphql/
│   ├── parser.go           # Parse GraphQL operations
│   ├── extractor.go        # Extract resource refs from AST
│   └── transformer.go      # Filter response data
├── auth/
│   ├── permissions.go      # Dagster permission definitions
│   └── matcher.go          # Wildcard permission matching
└── audit/
    └── logger.go           # Audit event logging
```

## Dagster GraphQL Operations to Intercept

### Queries (Filter Response)
- `pipelinesOrError` / `jobsOrError` - List pipelines/jobs
- `pipelineOrError` / `jobOrError` - Single pipeline/job
- `assetNodes` / `assetsOrError` - List assets
- `scheduleOrError` / `schedulesOrError` - Schedules
- `sensorOrError` / `sensorsOrError` - Sensors
- `runsOrError` - List runs

### Mutations (Block/Allow)
- `launchRun` / `launchPipelineExecution` - Execute job
- `launchPartitionBackfill` - Backfill runs
- `terminateRun` - Terminate run
- `startSchedule` / `stopSchedule` - Control schedule
- `startSensor` / `stopSensor` - Control sensor
- `wipeAssets` - Delete asset data

## Multi-Tenancy Strategy

### Shared Instance (Default)
- All tenants use same Dagster instance
- Jobs prefixed with tenant ID: `tenant_abc123_orders_etl`
- Proxy filters by prefix pattern

### Isolated Instance (Premium)
- Dedicated Dagster per tenant
- Proxy routes based on tenant header
- Complete isolation

## Docker Integration

Add to `deployments/docker/docker-compose.yml`:
```yaml
dagster-proxy:
  build:
    context: ../..
    dockerfile: deployments/docker/Dockerfile.dagster-proxy
  ports:
    - "3002:8080"
  environment:
    DAGSTER_UPSTREAM_URL: http://dagster-webserver:3000
    PHILOTES_DATABASE_URL: postgresql://philotes:philotes@postgres:5432/philotes
    PHILOTES_AUTH_JWT_SECRET: ${PHILOTES_AUTH_JWT_SECRET}
  depends_on:
    - postgres
    - dagster-webserver
```

## Key Considerations

1. **WebSocket Support**: Dagster uses GraphQL subscriptions for real-time updates
2. **Response Streaming**: Large query results may need streaming
3. **Schema Versioning**: Dagster schema may change between versions
4. **Performance**: Minimize parsing overhead for high-throughput
5. **Error Handling**: Return proper GraphQL errors for denied operations
