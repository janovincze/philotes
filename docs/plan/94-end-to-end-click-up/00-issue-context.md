# Issue #94 - End-to-End Click-Up Experience

## Summary
Enable a complete "click-up" experience where users can set up CDC pipelines from the dashboard with automatic warehouse creation, monitoring, and verification.

## Goals
1. Add a source database from the dashboard
2. Select replication mode (buffer-based CDC vs streaming)
3. Auto-create Lakekeeper warehouse and Iceberg tables
4. Auto-create and start pipelines with pre-flight checks
5. Monitor with Grafana/Prometheus
6. Query data verification

## Current Gaps
| Gap | Impact |
|-----|--------|
| No warehouse bootstrap | CDC fails if Lakekeeper warehouse doesn't exist |
| No query execution API | Can't verify data arrived in Iceberg |
| Data verification placeholder | Onboarding shows fake data |
| Source detail pages missing | Can't view/edit sources after creation |
| Test connection broken | Button exists but doesn't work |
| Alerts UI stubbed | Shows "coming soon" |

## Implementation Phases
### Phase 1: Backend (Critical Path)
- Add `EnsureWarehouse()` to Lakekeeper REST catalog
- Expose query execution API endpoint (`POST /api/v1/query/execute`)
- Wire `VerifyDataFlow()` to real Trino query
- Add pipeline start pre-flight checks

### Phase 2: Dashboard
- Create source detail page (`/sources/[id]`)
- Create source creation page (`/sources/new`)
- Wire test connection button
- Add replication mode selection to setup wizard
- Implement alerts UI (replace "coming soon")
- Add Grafana link to sidebar

### Phase 3: Verification
- Create E2E test for full click-up flow
- Manual testing checklist

## Acceptance Criteria
- Fresh deployment → setup wizard creates working pipeline
- Warehouse auto-created when pipeline starts
- Source detail page shows connection status
- Test connection button works
- Alerts page shows real alert rules
- Insert row in source → data appears in Iceberg within 60s
- Grafana dashboard accessible from sidebar
