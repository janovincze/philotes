# Implementation Plan - API-001: Core Management API Framework

## Overview

Implement a RESTful management API using the Gin framework with OpenAPI 3.0 specification. This API will serve as the management layer for Philotes, providing endpoints for health checks, metrics, and resource stubs for future CRUD operations.

## Approach

**API-First Design:** Write OpenAPI 3.0 specification first, then implement handlers that conform to it.

**Key Design Decisions:**
1. Use **Gin framework** - mature, well-documented, excellent middleware support
2. Follow existing patterns - slog logging, config-based, graceful shutdown
3. Integrate with existing **health.Manager** for health endpoints
4. Use middleware stack for cross-cutting concerns (logging, CORS, rate limiting)
5. Return structured JSON errors with RFC 7807 problem details format

## Dependencies to Add

```go
// go.mod additions
github.com/gin-gonic/gin v1.10.0
github.com/gin-contrib/cors v1.7.2
golang.org/x/time/rate // for rate limiting (standard library pattern)
```

## Package Structure

```
internal/api/
├── server.go           # Server setup, configuration, graceful shutdown
├── router.go           # Route registration, versioning
├── middleware/
│   ├── logging.go      # Request/response logging with slog
│   ├── cors.go         # CORS configuration
│   ├── ratelimit.go    # Rate limiting middleware
│   ├── recovery.go     # Panic recovery with structured errors
│   └── requestid.go    # Request ID generation
├── handlers/
│   ├── health.go       # Health, liveness, readiness endpoints
│   ├── config.go       # System configuration endpoint
│   └── version.go      # Version information endpoint
├── models/
│   ├── error.go        # RFC 7807 problem details
│   └── response.go     # Common response wrappers
└── validation/
    └── validator.go    # Request validation utilities

api/openapi/
└── openapi.yaml        # OpenAPI 3.0 specification
```

## Task Breakdown

### Task 1: Add Dependencies and Update go.mod
- Add Gin framework and gin-contrib/cors
- Run `go mod tidy`

### Task 2: Create OpenAPI 3.0 Specification
- Define API info, servers, paths
- Define health endpoints (/health, /health/live, /health/ready)
- Define version endpoint (/api/v1/version)
- Define config endpoint (/api/v1/config)
- Define error schemas (RFC 7807)
- Add stubs for future resources (sources, pipelines, destinations)

### Task 3: Implement API Models
- `internal/api/models/error.go` - RFC 7807 Problem Details
- `internal/api/models/response.go` - Common response wrappers

### Task 4: Implement Middleware Stack
- `internal/api/middleware/logging.go` - slog-based request logging
- `internal/api/middleware/cors.go` - CORS configuration
- `internal/api/middleware/ratelimit.go` - Token bucket rate limiting
- `internal/api/middleware/recovery.go` - Panic recovery
- `internal/api/middleware/requestid.go` - X-Request-ID handling

### Task 5: Implement Handlers
- `internal/api/handlers/health.go` - Integrate with health.Manager
- `internal/api/handlers/version.go` - Return version info
- `internal/api/handlers/config.go` - Return safe config subset

### Task 6: Implement Server and Router
- `internal/api/server.go` - Server struct, Start/Stop methods
- `internal/api/router.go` - Route registration with versioning

### Task 7: Update main.go Entry Point
- Wire up API server with config
- Create health.Manager and register checkers
- Implement graceful shutdown

### Task 8: Extend Configuration
- Add CORS origins to config
- Add rate limit settings to config

### Task 9: Write Tests
- Unit tests for middleware
- Handler tests with httptest
- Integration test for server startup/shutdown

### Task 10: Update Makefile
- Add API-specific build target if needed
- Ensure `make build` includes api binary

## API Endpoints (v1)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Overall health status (integrates health.Manager) |
| GET | `/health/live` | Kubernetes liveness probe |
| GET | `/health/ready` | Kubernetes readiness probe |
| GET | `/api/v1/version` | API and service version info |
| GET | `/api/v1/config` | Safe configuration subset |
| GET | `/metrics` | Prometheus metrics (existing pattern) |

### Future Endpoints (Stubs in OpenAPI)
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/sources` | Source management (API-002) |
| GET/POST | `/api/v1/pipelines` | Pipeline management (API-002) |
| GET/POST | `/api/v1/destinations` | Destination management (API-002) |

## Error Response Format (RFC 7807)

```json
{
  "type": "https://philotes.io/errors/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "The request body contains invalid fields",
  "instance": "/api/v1/sources",
  "errors": [
    {"field": "host", "message": "host is required"}
  ]
}
```

## Configuration Extensions

```go
// Add to APIConfig
type APIConfig struct {
    // Existing fields...
    ListenAddr   string
    BaseURL      string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration

    // New fields
    CORSOrigins     []string      // Allowed CORS origins
    RateLimitRPS    float64       // Requests per second limit
    RateLimitBurst  int           // Burst size for rate limiting
}
```

## Test Strategy

1. **Unit Tests:**
   - Middleware functions in isolation
   - Handler logic with mocked dependencies
   - Model serialization

2. **Integration Tests:**
   - Server startup/shutdown
   - Full request/response cycle
   - Health endpoint integration

3. **Manual Testing:**
   - curl commands in docs
   - Postman collection export

## Files to Create

| File | Lines (Est.) | Purpose |
|------|--------------|---------|
| `api/openapi/openapi.yaml` | ~300 | OpenAPI 3.0 specification |
| `internal/api/server.go` | ~150 | Server setup and lifecycle |
| `internal/api/router.go` | ~80 | Route registration |
| `internal/api/middleware/logging.go` | ~60 | Request logging |
| `internal/api/middleware/cors.go` | ~40 | CORS middleware |
| `internal/api/middleware/ratelimit.go` | ~80 | Rate limiting |
| `internal/api/middleware/recovery.go` | ~50 | Panic recovery |
| `internal/api/middleware/requestid.go` | ~40 | Request ID |
| `internal/api/handlers/health.go` | ~100 | Health endpoints |
| `internal/api/handlers/version.go` | ~40 | Version endpoint |
| `internal/api/handlers/config.go` | ~60 | Config endpoint |
| `internal/api/models/error.go` | ~80 | Error types |
| `internal/api/models/response.go` | ~40 | Response wrappers |
| `cmd/philotes-api/main.go` | ~120 | Updated entry point |
| `internal/config/config.go` | +30 | Config extensions |
| Tests | ~400 | Unit and integration tests |

**Total Estimate:** ~1,700 LOC (focused scope for framework only)

## Success Criteria

- [ ] Gin-based HTTP server starts and responds
- [ ] OpenAPI 3.0 spec is complete and valid
- [ ] All middleware functions correctly (logging, CORS, rate limiting)
- [ ] Health endpoints integrate with existing health.Manager
- [ ] Structured error responses follow RFC 7807
- [ ] API versioning works (/api/v1/...)
- [ ] Graceful shutdown works
- [ ] All tests pass
- [ ] `make build` and `make test` succeed
