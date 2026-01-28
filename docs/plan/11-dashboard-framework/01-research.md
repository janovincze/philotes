# Research: Issue #11 - Dashboard Framework

## Existing Frontend Setup

**Status:** No frontend exists yet
- No `/web` or `/dashboard` directory
- No `package.json`, `tsconfig.json`, or Next.js configuration
- CLAUDE.md mentions `Dashboard (Next.js)` as "In Progress"

**Directory:** Create `/web/` at project root level

## OpenAPI Specification

**Location:** `/api/openapi/openapi.yaml`

**Details:**
- API Version: 1.0.0
- Base URL: `http://localhost:8080` (dev), `https://api.philotes.io` (prod)
- Prefix: `/api/v1`
- CORS: Enabled with `*` origins

## Key API Endpoints

### Health & System
- `GET /health` - Overall health with component checks
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `GET /api/v1/version` - Version info

### Sources
- `POST /api/v1/sources` - Create source
- `GET /api/v1/sources` - List sources
- `GET /api/v1/sources/:id` - Get source
- `PUT /api/v1/sources/:id` - Update source
- `DELETE /api/v1/sources/:id` - Delete source
- `POST /api/v1/sources/:id/test` - Test connection
- `GET /api/v1/sources/:id/tables` - Discover tables

### Pipelines
- `POST /api/v1/pipelines` - Create pipeline
- `GET /api/v1/pipelines` - List pipelines
- `GET /api/v1/pipelines/:id` - Get pipeline
- `PUT /api/v1/pipelines/:id` - Update pipeline
- `DELETE /api/v1/pipelines/:id` - Delete pipeline
- `POST /api/v1/pipelines/:id/start` - Start
- `POST /api/v1/pipelines/:id/stop` - Stop
- `GET /api/v1/pipelines/:id/status` - Status

### Alerts
- Full alerting framework with rules, channels, silences

### Scaling
- Policy management endpoints

## Data Models

```typescript
// Source
{ id, name, type, host, port, database_name, username, ssl_mode, status, created_at, updated_at }

// Pipeline
{ id, name, source_id, status, config, error_message, tables, created_at, updated_at, started_at, stopped_at }

// Health
{ status, components: { [name]: { status, message, duration_ms } }, timestamp }
```

## Recommended Directory Structure

```
web/
├── src/
│   ├── app/                    # Next.js App Router
│   │   ├── layout.tsx          # Root layout
│   │   ├── page.tsx            # Dashboard home
│   │   ├── sources/            # Source management
│   │   ├── pipelines/          # Pipeline management
│   │   ├── alerts/             # Alert management
│   │   ├── settings/           # Settings
│   │   └── error.tsx           # Error boundary
│   ├── components/
│   │   ├── ui/                 # shadcn/ui components
│   │   ├── layout/             # Sidebar, header
│   │   └── ...                 # Feature components
│   ├── lib/
│   │   ├── api/                # API client
│   │   ├── hooks/              # React hooks
│   │   ├── store/              # Zustand stores
│   │   └── utils/              # Utilities
│   └── types/                  # TypeScript types
├── package.json
├── tsconfig.json
├── next.config.js
└── tailwind.config.ts
```

## Dependencies

**Core:**
- next@14.x, react@18.x, typescript
- tailwindcss, shadcn/ui
- @tanstack/react-query (data fetching)
- zustand (state management)
- react-hook-form + zod (forms)
- recharts (charts)
- lucide-react (icons)

## Development Environment

Available via `docker-compose`:
- API: `localhost:8080`
- PostgreSQL: `localhost:5432`
- MinIO: `localhost:9000`
- Prometheus: `localhost:9090`
- Grafana: `localhost:3000`
