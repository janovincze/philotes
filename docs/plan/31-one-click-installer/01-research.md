# Research Findings: Issue #31 - One-Click Cloud Installer

## Existing Infrastructure Analysis

### 1. Frontend (Next.js)

**Location:** `/web/src/`
- Framework: Next.js 16, TypeScript, React 19, Tailwind CSS
- UI Library: shadcn/ui components
- State: Zustand + TanStack React Query
- **Key Finding:** Setup wizard already exists at `/web/src/components/setup/` with 6-step flow - pattern is reusable

### 2. Backend API (Go/Gin)

**Location:** `/internal/api/`
- Pattern: Handler → Service → Repository
- Auth: JWT + API keys (already implemented)
- Middleware: CORS, rate limiting, logging, metrics
- Routes: Centralized in `/internal/api/server.go`

### 3. Pulumi Integration

**Location:** `/deployments/pulumi/`
- All 5 providers implemented (Hetzner, Scaleway, OVH, Exoscale, Contabo)
- Cost estimation exists at `/pkg/output/cost.go`
- Full deployment orchestration in `/pkg/platform/`
- **Key Insight:** CLI deployment is complete - installer wraps via Automation API

### 4. Cost Estimation (Already Built)

| Provider | Range (EUR/month) |
|----------|-------------------|
| Hetzner | €4.35 - €28.99 |
| Scaleway | €4.99 - €35.99 |
| OVH | €6 - €208 |
| Exoscale | €7 - €224 |
| Contabo | €4.99 - €38.99 |

### 5. Missing Components

- WebSocket infrastructure for real-time progress
- Cloud provider OAuth handlers
- Deployment tracking database schema
- Pulumi Automation API wrapper

## Architecture Decision

**Recommendation: Integrated Installer** (Option A)
- Installer as new pages in existing Next.js app (`/install/*`)
- Shared API with dashboard
- Reuses auth, styling, infrastructure

## Implementation Components

### Backend (Go)

New routes under `/api/v1/installer/`:
```
POST   /deployments           - Create deployment
GET    /deployments           - List deployments
GET    /deployments/:id       - Get status
DELETE /deployments/:id       - Cancel/rollback
WS     /deployments/:id/logs  - Real-time logs
GET    /providers             - List providers with regions
GET    /providers/:id/pricing - Get pricing info
POST   /providers/:id/oauth   - OAuth callback
```

### Frontend (Next.js)

Wizard steps:
1. **Provider Selection** - Choose cloud provider
2. **Authentication** - OAuth to provider
3. **Size Selection** - Small/Medium/Large with costs
4. **Region Selection** - With latency indicators
5. **Configuration** - Domain, SSH key, etc.
6. **Review** - Summary before deploy
7. **Deployment** - Real-time progress
8. **Success** - Access credentials

### Database Schema

```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    config JSONB,
    outputs JSONB,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE deployment_logs (
    id SERIAL PRIMARY KEY,
    deployment_id UUID REFERENCES deployments(id),
    level VARCHAR(10),
    message TEXT,
    timestamp TIMESTAMP DEFAULT NOW()
);
```

## Estimated LOC

| Component | LOC |
|-----------|-----|
| Frontend (wizard, components) | 3,500-4,000 |
| Backend (handlers, services, repos) | 2,500-3,000 |
| Database schema | 300-400 |
| Pulumi Automation wrapper | 1,000-1,500 |
| WebSocket | 400-600 |
| **Total** | **~8,000-10,000** |

## Key Files to Reference

- Handler pattern: `/internal/api/handlers/pipelines.go`
- Service pattern: `/internal/api/services/pipeline.go`
- Setup wizard: `/web/src/components/setup/setup-wizard.tsx`
- Cost estimation: `/deployments/pulumi/pkg/output/cost.go`
- Platform deploy: `/deployments/pulumi/pkg/platform/platform.go`
