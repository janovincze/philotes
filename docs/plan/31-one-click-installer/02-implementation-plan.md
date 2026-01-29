# Implementation Plan: Issue #31 - One-Click Cloud Installer

## Summary

Build a web-based installer that allows users to deploy Philotes to European cloud providers through a guided wizard with real-time progress tracking.

## Scope

This is a large feature (~8,000-10,000 LOC) that should be implemented in phases.

## Phase 1: Backend Foundation (This PR)

### 1.1 Database Schema

Create migration for deployment tracking:
- `deployments` table (id, user_id, provider, region, size, status, config, outputs)
- `deployment_logs` table (deployment_id, level, message, timestamp)

### 1.2 API Endpoints

Create installer API handlers:
```
GET  /api/v1/installer/providers           - List providers
GET  /api/v1/installer/providers/:id       - Provider details with regions/pricing
POST /api/v1/installer/deployments         - Create deployment
GET  /api/v1/installer/deployments/:id     - Get deployment status
```

### 1.3 Deployment Service

Create service layer:
- Validate deployment configuration
- Calculate cost estimates (use existing cost.go)
- Store deployment in database

### 1.4 Pulumi Automation API Wrapper

Create wrapper for programmatic Pulumi execution:
- Initialize stack with config
- Run `pulumi up` programmatically
- Capture logs and outputs
- Support destroy for rollback

## Phase 2: Real-time Progress (Future PR)

- WebSocket endpoint for deployment logs
- Frontend progress component
- Log streaming from Pulumi

## Phase 3: Frontend Wizard (Future PR)

- Provider selection page
- Size/cost calculator
- Region selection
- Configuration form
- Review & deploy page
- Progress dashboard

## Phase 4: OAuth Integration (Future PR)

- Cloud provider OAuth flows
- Token encryption and storage
- Token cleanup after deployment

---

## Phase 1 Implementation Details

### Files to Create

| File | Description |
|------|-------------|
| `internal/api/handlers/installer.go` | HTTP handlers for installer API |
| `internal/api/services/installer.go` | Deployment orchestration logic |
| `internal/api/repositories/deployment.go` | Database operations |
| `internal/api/models/installer.go` | Request/response types |
| `internal/installer/pulumi.go` | Pulumi Automation API wrapper |
| `deployments/docker/init-scripts/10-installer-schema.sql` | Database schema |

### Files to Modify

| File | Changes |
|------|---------|
| `internal/api/server.go` | Register installer routes |
| `go.mod` | Add Pulumi Automation API dependency |

### API Schema

#### GET /api/v1/installer/providers

```json
{
  "providers": [
    {
      "id": "hetzner",
      "name": "Hetzner Cloud",
      "regions": ["nbg1", "fsn1", "hel1"],
      "sizes": {
        "small": { "monthly_cost_eur": 27, "nodes": 3 },
        "medium": { "monthly_cost_eur": 60, "nodes": 4 },
        "large": { "monthly_cost_eur": 150, "nodes": 6 }
      }
    }
  ]
}
```

#### POST /api/v1/installer/deployments

```json
{
  "provider": "hetzner",
  "region": "nbg1",
  "size": "small",
  "environment": "production",
  "domain": "philotes.example.com"
}
```

#### Response

```json
{
  "id": "uuid",
  "status": "pending",
  "provider": "hetzner",
  "region": "nbg1",
  "estimated_cost_eur": 27,
  "created_at": "2026-01-29T12:00:00Z"
}
```

### Deployment Status Flow

```
pending → provisioning → configuring → deploying → verifying → completed
                                                           ↓
                                                        failed
```

## Verification

```bash
# Build
go build ./...

# Test API
curl http://localhost:8080/api/v1/installer/providers
curl -X POST http://localhost:8080/api/v1/installer/deployments \
  -H "Content-Type: application/json" \
  -d '{"provider":"hetzner","region":"nbg1","size":"small"}'
```

## Notes

- Phase 1 focuses on backend foundation only
- Frontend and OAuth will be separate PRs
- Pulumi Automation API requires `github.com/pulumi/pulumi/sdk/v3/go/auto`
- Cost estimates reuse existing `/deployments/pulumi/pkg/output/cost.go`
