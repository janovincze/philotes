# Research Findings - Issue #94

## Key Findings

### Backend Status
| Area | Status | Notes |
|------|--------|-------|
| Warehouse Bootstrap | NOT IMPLEMENTED | No `EnsureWarehouse` in catalog; Lakekeeper management API at `/management/v1/warehouse` needed |
| Query Execution API | COMPLETE | Full Trino client in `services/query.go`, handler in `handlers/query.go` |
| Data Verification | STUB | `VerifyDataFlow()` in `services/onboarding.go` always returns success with 0 rows |
| Pipeline Pre-flight | NOT IMPLEMENTED | `Start()` just updates status, no validation |
| Source CRUD API | COMPLETE | Full CRUD + test connection + discover tables |
| Alerts API | COMPLETE | Full CRUD for rules, instances, silences, channels, routes |

### Frontend Status
| Area | Status | Notes |
|------|--------|-------|
| Source List Page | EXISTS | `/sources/page.tsx` - test connection button not wired |
| Source Detail Page | MISSING | `/sources/[id]` - 404 |
| Source Create Page | MISSING | `/sources/new` - 404 |
| Alerts Page | STUB | Shows "coming soon" with Bell icon |
| Query Page | COMPLETE | Full SQL editor with Trino |
| Sidebar Grafana Link | MISSING | No link to Grafana |

### Key Files
- `internal/iceberg/catalog/catalog.go` - Catalog interface (needs EnsureWarehouse)
- `internal/iceberg/catalog/rest.go` - REST implementation (needs management API)
- `internal/api/services/pipeline.go:157-185` - Pipeline.Start() (needs pre-flight)
- `internal/api/services/onboarding.go:232-271` - VerifyDataFlow() (stub)
- `web/src/app/sources/page.tsx` - Test connection not wired
- `web/src/app/alerts/page.tsx` - Coming soon stub
- `web/src/lib/hooks/use-sources.ts` - All hooks exist
- `web/src/lib/api/types.ts` - All types exist
