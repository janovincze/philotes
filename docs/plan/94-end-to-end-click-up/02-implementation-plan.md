# Implementation Plan - Issue #94: End-to-End Click-Up Experience

## Approach Overview

This issue spans backend and frontend across multiple areas. We'll implement in two parallel tracks:

**Track A: Backend (Critical Path)**
1. Warehouse bootstrap via Lakekeeper management API
2. Pipeline pre-flight checks (source connectivity + warehouse)
3. Real data verification via Trino query

**Track B: Frontend**
4. Wire test connection button on sources list page
5. Source detail page (`/sources/[id]`)
6. Source create page (`/sources/new`)
7. Alerts UI (replace "coming soon" with real CRUD)
8. Add Grafana link to sidebar

## Task Breakdown

### Task 1: Warehouse Bootstrap (`EnsureWarehouse`)
**Files to modify:**
- `internal/iceberg/catalog/catalog.go` - Add `EnsureWarehouse` to interface
- `internal/iceberg/catalog/rest.go` - Implement using Lakekeeper management API
- `internal/config/config.go` - Add `StorageProfile` config for MinIO bucket

**Implementation:**
- Add `EnsureWarehouse(ctx context.Context) error` to `Catalog` interface
- In `RESTCatalog`, implement:
  1. `GET /management/v1/warehouse` - list warehouses, check if ours exists
  2. `POST /management/v1/warehouse` - create with S3 storage profile pointing to MinIO
- The warehouse config comes from existing `Config.Warehouse` field
- Storage profile uses existing MinIO/S3 config from `StorageConfig`

### Task 2: Pipeline Pre-flight Checks
**Files to modify:**
- `internal/api/services/pipeline.go` - Add pre-flight to `Start()`

**Implementation:**
- Before setting status to "starting", run checks:
  1. Get pipeline's source, call `SourceService.TestConnection()`
  2. Call `Catalog.EnsureWarehouse()` to ensure Lakekeeper warehouse exists
  3. Verify pipeline has at least one table mapping
- Return descriptive errors if any check fails
- Requires injecting `SourceService` and `Catalog` into `PipelineService`

### Task 3: Wire `VerifyDataFlow` to Real Trino Query
**Files to modify:**
- `internal/api/services/onboarding.go` - Replace stub with Trino query

**Implementation:**
- Inject `QueryService` into `OnboardingService`
- In `VerifyDataFlow()`:
  1. Build SQL: `SELECT COUNT(*) as cnt FROM {catalog}.{schema}.{table} LIMIT 1`
  2. Use `QueryService.ExecuteUserQuery()` to run it
  3. Poll with backoff until rows found or maxWait exceeded
  4. Return real row count and sample rows

### Task 4: Wire Test Connection Button (Sources List)
**Files to modify:**
- `web/src/app/sources/page.tsx` - Add onClick handler

**Implementation:**
- Import `useTestSourceConnection` hook
- Add onClick to "Test Connection" button calling the mutation
- Show loading state during test, toast on success/failure
- Update source status badge reactively

### Task 5: Source Detail Page
**Files to create:**
- `web/src/app/sources/[id]/page.tsx`

**Implementation:**
- Use `useSource(id)` hook to fetch source data
- Show: source info (name, host, port, database, SSL mode, status)
- Actions: test connection, edit (inline or modal), delete with confirmation
- Table discovery: call `useDiscoverTables` and display discovered tables
- Associated pipelines section (filter pipelines by source_id)

### Task 6: Source Create Page
**Files to create:**
- `web/src/app/sources/new/page.tsx`

**Implementation:**
- Form with: name, host, port, database_name, username, password, ssl_mode
- Follow existing form patterns (shadcn/ui form components)
- Test connection before save
- On success, redirect to `/sources/[id]`

### Task 7: Alerts UI
**Files to modify:**
- `web/src/app/alerts/page.tsx` - Replace "coming soon" stub

**Files to create:**
- `web/src/lib/api/alerts.ts` - API client for alerts endpoints
- `web/src/lib/hooks/use-alerts.ts` - React Query hooks
- `web/src/components/alerts/alert-rules-list.tsx` - Alert rules table
- `web/src/components/alerts/alert-rule-form.tsx` - Create/edit alert rule form
- `web/src/components/alerts/active-alerts.tsx` - Active alert instances
- `web/src/components/alerts/notification-channels.tsx` - Channels management

**Implementation:**
- Alerts page with tabs: Active Alerts | Rules | Notification Channels
- Active Alerts tab: list alert instances, acknowledge button
- Rules tab: CRUD table for alert rules
- Channels tab: CRUD for notification channels (Slack, Email, Webhook, PagerDuty)
- Use existing backend endpoints (`/api/v1/alerts/*`, `/api/v1/notifications/*`)

### Task 8: Add Grafana Link to Sidebar
**Files to modify:**
- `web/src/components/layout/sidebar.tsx` (or equivalent navigation component)

**Implementation:**
- Add external link to Grafana dashboard (default: `http://localhost:3000`)
- Use BarChart3 or similar icon from lucide-react

## Test Strategy

### Backend Tests
- Unit test for `EnsureWarehouse()` with mock HTTP responses
- Unit test for pipeline pre-flight checks (mock source service + catalog)
- Unit test for `VerifyDataFlow()` with mock query service

### Frontend Tests
- Verify source detail page renders with mock data
- Verify source create form validation
- Verify alerts page renders tabs and data

## Implementation Order

```
Task 1 (Warehouse Bootstrap) ──► Task 2 (Pre-flight Checks) ──► Task 3 (VerifyDataFlow)
                                                                        │
Task 4 (Wire Test Connection) ──────────────────────────────────────────┤
Task 5 (Source Detail Page) ────────────────────────────────────────────┤
Task 6 (Source Create Page) ────────────────────────────────────────────┤
Task 7 (Alerts UI) ────────────────────────────────────────────────────►│
Task 8 (Grafana Link) ─────────────────────────────────────────────────►└── Done
```

Backend tasks (1-3) are sequential. Frontend tasks (4-8) are independent.
