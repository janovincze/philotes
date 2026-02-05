# Issue #103 - Full RBAC for Dagster OSS via GraphQL Proxy

## Summary

Build a GraphQL-aware authorization proxy in Go that provides Dagster+-level RBAC permissions without forking Dagster. The proxy intercepts all Dagster GraphQL requests, applies permission filtering, and logs all actions.

## Problem

Dagster OSS lacks native RBAC (it's a Dagster+ exclusive feature). To provide unified user management across Philotes with enterprise-grade RBAC, we need an authorization layer that sits between users and Dagster.

## Architecture

```
User Browser → Traefik → Philotes Dagster Proxy (Go) → Dagster Webserver (OSS)
```

The proxy:
1. Authenticates users via JWT/API key (reusing Philotes auth middleware)
2. Parses GraphQL queries/mutations to identify resources being accessed
3. Applies permission checks against Philotes RBAC system
4. Filters responses to only show authorized resources
5. Logs all actions for audit trail

## Permission Model

| Permission | Description |
|------------|-------------|
| `dagster:jobs:{name}:view` | See job definition and history |
| `dagster:jobs:{name}:execute` | Launch runs for this job |
| `dagster:assets:{key}:view` | See asset and materializations |
| `dagster:assets:{key}:materialize` | Trigger materialization |
| `dagster:schedules:{name}:view/control` | See/start/stop schedule |
| `dagster:sensors:{name}:view/control` | See/start/stop sensor |
| `dagster:runs:terminate` | Terminate visible runs |

Supports wildcards: `dagster:jobs:etl_*:execute`

## Pre-built Roles

- **dagster-viewer**: View all resources
- **dagster-operator**: Viewer + execute jobs, materialize assets
- **dagster-admin**: Full access including schedule/sensor control

## Multi-Tenancy

- **Shared Instance**: Filter resources by tenant prefix
- **Premium Tier**: Route to dedicated Dagster instances

## Acceptance Criteria

### Core Proxy
- [ ] GraphQL proxy intercepts all Dagster requests
- [ ] HTTP and WebSocket support (for subscriptions)
- [ ] JWT and API key authentication

### Permission Enforcement
- [ ] Per-job visibility filtering
- [ ] Per-job execution control
- [ ] Per-asset permissions
- [ ] Schedule/sensor control
- [ ] Run termination control
- [ ] Wildcard patterns

### Management
- [ ] Pre-built roles
- [ ] Custom role creation
- [ ] Dashboard UI for permissions

### Observability
- [ ] Full audit logging
- [ ] User attribution

## Dependencies

- Issue #24 (Dagster Integration) - **COMPLETED**

## New Files to Create

```
cmd/philotes-dagster-proxy/main.go
internal/dagster-proxy/
├── graphql/parser.go
├── graphql/transformer.go
├── auth/permissions.go
├── proxy/handler.go
└── audit/logger.go
```
