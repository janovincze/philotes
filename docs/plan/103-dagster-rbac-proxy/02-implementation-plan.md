# Implementation Plan: Issue #103 - Dagster RBAC Proxy

## Overview

Build a Go-based GraphQL authorization proxy that provides Dagster+-level RBAC for Dagster OSS. The proxy intercepts all Dagster requests, applies permission checks, filters responses, and logs all actions.

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                      philotes-dagster-proxy                          │
│                                                                      │
│  ┌────────────┐    ┌──────────────┐    ┌──────────────────────────┐ │
│  │ Auth       │    │ GraphQL      │    │ Proxy Handler            │ │
│  │ Middleware │───►│ Parser       │───►│ (httputil.ReverseProxy)  │ │
│  └────────────┘    └──────────────┘    └──────────────────────────┘ │
│        │                 │                        │                  │
│        ▼                 ▼                        ▼                  │
│  ┌────────────┐    ┌──────────────┐    ┌──────────────────────────┐ │
│  │ Permission │    │ Response     │    │ Audit Logger             │ │
│  │ Matcher    │    │ Transformer  │    │                          │ │
│  └────────────┘    └──────────────┘    └──────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ Dagster         │
                    │ Webserver (OSS) │
                    └─────────────────┘
```

## Deliverables

1. **Go Service** - `cmd/philotes-dagster-proxy/main.go`
2. **Proxy Library** - `internal/dagster-proxy/`
3. **Database Schema** - Dagster permissions table
4. **Docker Integration** - Dockerfile and compose updates
5. **Tests** - Unit and integration tests

---

## Package Structure

```
cmd/philotes-dagster-proxy/
└── main.go

internal/dagster-proxy/
├── config/
│   └── config.go               # Configuration loading
├── handler/
│   ├── handler.go              # Main Gin router setup
│   ├── proxy.go                # HTTP reverse proxy
│   └── websocket.go            # WebSocket proxy
├── graphql/
│   ├── parser.go               # Parse GraphQL requests
│   ├── extractor.go            # Extract resource references
│   ├── transformer.go          # Filter responses
│   └── operations.go           # Known Dagster operations
├── auth/
│   ├── permissions.go          # Permission definitions & roles
│   ├── matcher.go              # Wildcard pattern matching
│   └── checker.go              # Permission check logic
├── audit/
│   └── logger.go               # Audit event logging
└── models/
    └── dagster.go              # Dagster-specific models

deployments/docker/
├── Dockerfile.dagster-proxy
└── init-scripts/
    └── 04-dagster-rbac-schema.sql
```

---

## Task Breakdown

### Phase 1: Core Infrastructure

#### Task 1.1: Configuration & Bootstrap
**Files:** `cmd/philotes-dagster-proxy/main.go`, `internal/dagster-proxy/config/config.go`

Create service entry point following existing pattern:
- Load config from environment
- Connect to PostgreSQL (reuse Philotes database)
- Initialize auth services (reuse from `internal/api/services`)
- Setup health checks
- Start Gin server

**Configuration:**
```go
type Config struct {
    ListenAddr        string        // :8080
    DagsterUpstreamURL string       // http://dagster-webserver:3000
    Database          DatabaseConfig
    Auth              AuthConfig
    LogLevel          string
}
```

#### Task 1.2: Basic Reverse Proxy
**Files:** `internal/dagster-proxy/handler/proxy.go`

Implement transparent proxy using `httputil.ReverseProxy`:
- Forward all requests to Dagster upstream
- Copy headers bidirectionally
- Handle error responses

```go
func NewProxyHandler(upstreamURL *url.URL) *ProxyHandler
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

#### Task 1.3: WebSocket Proxy
**Files:** `internal/dagster-proxy/handler/websocket.go`

Handle GraphQL subscriptions:
- Detect WebSocket upgrade requests
- Establish bidirectional connection
- Proxy messages between client and Dagster

---

### Phase 2: GraphQL Parsing & Permission Model

#### Task 2.1: GraphQL Request Parser
**Files:** `internal/dagster-proxy/graphql/parser.go`

Parse incoming GraphQL requests:
```go
type GraphQLRequest struct {
    Query         string                 `json:"query"`
    OperationName string                 `json:"operationName"`
    Variables     map[string]interface{} `json:"variables"`
}

type ParsedOperation struct {
    Type       OperationType  // Query, Mutation, Subscription
    Name       string
    Fields     []FieldSelection
    Variables  map[string]interface{}
}

func ParseRequest(body []byte) (*ParsedOperation, error)
```

#### Task 2.2: Resource Extractor
**Files:** `internal/dagster-proxy/graphql/extractor.go`

Extract resource references from operations:
```go
type ResourceRef struct {
    Type   ResourceType  // Job, Asset, Schedule, Sensor, Run
    Name   string        // job name, asset key, etc.
    Action ActionType    // View, Execute, Control, Terminate
}

func ExtractResources(op *ParsedOperation) []ResourceRef
```

**Dagster Operations Mapping:**

| Operation | Resource Type | Action |
|-----------|--------------|--------|
| `jobsOrError` | Job | View |
| `launchRun` | Job | Execute |
| `assetNodes` | Asset | View |
| `launchPartitionBackfill` | Asset | Materialize |
| `startSchedule` | Schedule | Control |
| `stopSensor` | Sensor | Control |
| `terminateRun` | Run | Terminate |

#### Task 2.3: Permission Definitions
**Files:** `internal/dagster-proxy/auth/permissions.go`

Define Dagster-specific permissions:
```go
// Permission format: dagster:{resource_type}:{resource_name}:{action}
const (
    PermissionDagsterJobsView       = "dagster:jobs:*:view"
    PermissionDagsterJobsExecute    = "dagster:jobs:*:execute"
    PermissionDagsterAssetsView     = "dagster:assets:*:view"
    PermissionDagsterAssetsMaterialize = "dagster:assets:*:materialize"
    PermissionDagsterSchedulesView  = "dagster:schedules:*:view"
    PermissionDagsterSchedulesControl = "dagster:schedules:*:control"
    PermissionDagsterSensorsView    = "dagster:sensors:*:view"
    PermissionDagsterSensorsControl = "dagster:sensors:*:control"
    PermissionDagsterRunsTerminate  = "dagster:runs:*:terminate"
)

// Pre-built roles
var DagsterRoles = map[string][]string{
    "dagster-viewer": {
        "dagster:jobs:*:view",
        "dagster:assets:*:view",
        "dagster:schedules:*:view",
        "dagster:sensors:*:view",
    },
    "dagster-operator": {
        "dagster:jobs:*:view",
        "dagster:jobs:*:execute",
        "dagster:assets:*:view",
        "dagster:assets:*:materialize",
        "dagster:schedules:*:view",
        "dagster:sensors:*:view",
    },
    "dagster-admin": {
        "dagster:*:*:*", // Full access
    },
}
```

#### Task 2.4: Wildcard Permission Matcher
**Files:** `internal/dagster-proxy/auth/matcher.go`

Pattern matching with wildcards:
```go
// MatchPermission checks if a permission pattern matches a required permission
// Pattern: dagster:jobs:etl_*:execute
// Required: dagster:jobs:etl_orders:execute
// Returns: true
func MatchPermission(pattern, required string) bool

// HasDagsterPermission checks if user has required Dagster permission
func HasDagsterPermission(perms []string, resourceType, resourceName, action string) bool
```

---

### Phase 3: Request Interception & Response Filtering

#### Task 3.1: Auth Middleware Integration
**Files:** `internal/dagster-proxy/handler/handler.go`

Integrate with Philotes auth:
- Reuse `AuthService` and `APIKeyService` from `internal/api/services`
- Extract AuthContext from request
- Inject into request context for downstream handlers

#### Task 3.2: Permission Checker
**Files:** `internal/dagster-proxy/auth/checker.go`

Check permissions before proxying:
```go
type PermissionChecker struct {
    matcher *Matcher
}

// CheckMutation blocks unauthorized mutations
func (c *PermissionChecker) CheckMutation(ctx context.Context, authCtx *AuthContext, op *ParsedOperation) error

// GetAllowedResources returns list of resources user can view
func (c *PermissionChecker) GetAllowedResources(authCtx *AuthContext, resourceType ResourceType) []string
```

#### Task 3.3: Response Transformer
**Files:** `internal/dagster-proxy/graphql/transformer.go`

Filter query responses:
```go
// TransformResponse filters response data based on permissions
func TransformResponse(body []byte, op *ParsedOperation, allowedResources []ResourceRef) ([]byte, error)
```

**Filtering Logic:**
- Parse JSON response
- Navigate to resource arrays (e.g., `data.jobsOrError.results`)
- Filter out resources user cannot view
- Reconstruct response

---

### Phase 4: Audit Logging

#### Task 4.1: Audit Logger
**Files:** `internal/dagster-proxy/audit/logger.go`

Log all Dagster actions:
```go
type AuditLogger struct {
    repo   *repositories.AuditRepository
    logger *slog.Logger
}

func (l *AuditLogger) LogDagsterAction(ctx context.Context, action DagsterAuditAction) error

type DagsterAuditAction struct {
    UserID       uuid.UUID
    Operation    string    // launchRun, startSchedule, etc.
    ResourceType string    // job, asset, schedule, sensor
    ResourceName string
    Allowed      bool
    Reason       string    // Permission granted/denied reason
    IPAddress    string
    UserAgent    string
}
```

---

### Phase 5: Database & Docker Integration

#### Task 5.1: Database Schema
**Files:** `deployments/docker/init-scripts/04-dagster-rbac-schema.sql`

```sql
-- Dagster role assignments (extends existing user roles)
CREATE TABLE dagster_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,  -- dagster-viewer, dagster-operator, dagster-admin
    resource_pattern VARCHAR(255),  -- Optional: dagster:jobs:etl_*:*
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role, resource_pattern)
);

-- Custom Dagster permissions (for fine-grained control)
CREATE TABLE dagster_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(255) NOT NULL,  -- dagster:jobs:orders_etl:execute
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, permission)
);

CREATE INDEX idx_dagster_role_assignments_user ON dagster_role_assignments(user_id);
CREATE INDEX idx_dagster_permissions_user ON dagster_permissions(user_id);
```

#### Task 5.2: Dockerfile
**Files:** `deployments/docker/Dockerfile.dagster-proxy`

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /philotes-dagster-proxy ./cmd/philotes-dagster-proxy

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /philotes-dagster-proxy /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/philotes-dagster-proxy"]
```

#### Task 5.3: Docker Compose Update
**Files:** `deployments/docker/docker-compose.yml`

```yaml
dagster-proxy:
  build:
    context: ../..
    dockerfile: deployments/docker/Dockerfile.dagster-proxy
  container_name: philotes-dagster-proxy
  ports:
    - "3002:8080"
  environment:
    DAGSTER_PROXY_LISTEN_ADDR: ":8080"
    DAGSTER_PROXY_UPSTREAM_URL: http://dagster-webserver:3000
    PHILOTES_DATABASE_HOST: postgres
    PHILOTES_DATABASE_PORT: 5432
    PHILOTES_DATABASE_USER: philotes
    PHILOTES_DATABASE_PASSWORD: philotes
    PHILOTES_DATABASE_NAME: philotes
    PHILOTES_AUTH_ENABLED: "true"
    PHILOTES_AUTH_JWT_SECRET: ${PHILOTES_AUTH_JWT_SECRET:-your-secret-key-min-32-chars-long}
  depends_on:
    postgres:
      condition: service_healthy
    dagster-webserver:
      condition: service_healthy
  healthcheck:
    test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 5
```

---

### Phase 6: Testing

#### Task 6.1: Unit Tests
**Files:** `internal/dagster-proxy/*_test.go`

- GraphQL parser tests with real Dagster queries
- Permission matcher tests (wildcards, patterns)
- Response transformer tests

#### Task 6.2: Integration Tests
**Files:** `internal/dagster-proxy/integration_test.go`

- Full proxy flow with mocked Dagster
- Auth middleware integration
- WebSocket subscription handling

---

## API Design

### Dagster Proxy Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/graphql` | POST | GraphQL proxy (main endpoint) |
| `/graphql` | GET | WebSocket upgrade for subscriptions |
| `/health` | GET | Health check |
| `/health/ready` | GET | Readiness check |

### Internal Permission API (Optional, Phase 2)

| Path | Method | Description |
|------|--------|-------------|
| `/api/v1/dagster/roles` | GET | List Dagster roles |
| `/api/v1/dagster/roles/:userId` | PUT | Assign Dagster role |
| `/api/v1/dagster/permissions/:userId` | GET | Get user permissions |
| `/api/v1/dagster/permissions/:userId` | PUT | Set custom permissions |

---

## Files to Create

| File | Purpose |
|------|---------|
| `cmd/philotes-dagster-proxy/main.go` | Service entry point |
| `internal/dagster-proxy/config/config.go` | Configuration |
| `internal/dagster-proxy/handler/handler.go` | Router setup |
| `internal/dagster-proxy/handler/proxy.go` | HTTP proxy |
| `internal/dagster-proxy/handler/websocket.go` | WebSocket proxy |
| `internal/dagster-proxy/graphql/parser.go` | GraphQL parsing |
| `internal/dagster-proxy/graphql/extractor.go` | Resource extraction |
| `internal/dagster-proxy/graphql/transformer.go` | Response filtering |
| `internal/dagster-proxy/graphql/operations.go` | Dagster operations |
| `internal/dagster-proxy/auth/permissions.go` | Permission definitions |
| `internal/dagster-proxy/auth/matcher.go` | Pattern matching |
| `internal/dagster-proxy/auth/checker.go` | Permission checking |
| `internal/dagster-proxy/audit/logger.go` | Audit logging |
| `internal/dagster-proxy/models/dagster.go` | Models |
| `deployments/docker/Dockerfile.dagster-proxy` | Docker build |
| `deployments/docker/init-scripts/04-dagster-rbac-schema.sql` | Database schema |

## Files to Modify

| File | Changes |
|------|---------|
| `deployments/docker/docker-compose.yml` | Add dagster-proxy service |
| `Makefile` | Add build-dagster-proxy target |

---

## Verification

### Unit Tests
```bash
go test ./internal/dagster-proxy/... -v
```

### Integration Test
```bash
# Start environment
docker compose -f deployments/docker/docker-compose.yml up -d

# Verify proxy health
curl http://localhost:3002/health

# Test authenticated request
curl -X POST http://localhost:3002/graphql \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ jobsOrError { __typename } }"}'
```

### Manual Checklist
- [ ] Proxy forwards requests to Dagster
- [ ] Unauthenticated requests are rejected
- [ ] Viewers can see but not execute jobs
- [ ] Operators can execute jobs
- [ ] Admins have full access
- [ ] Wildcard permissions work
- [ ] Actions are audit logged
- [ ] WebSocket subscriptions work

---

## Success Criteria

1. Proxy intercepts all Dagster GraphQL requests
2. JWT and API key authentication works
3. Per-job, per-asset, per-schedule/sensor permissions enforced
4. Wildcard patterns supported (`dagster:jobs:etl_*:execute`)
5. Pre-built roles (Viewer, Operator, Admin) work correctly
6. All actions logged for audit
7. WebSocket subscriptions proxied correctly
8. Docker integration complete
