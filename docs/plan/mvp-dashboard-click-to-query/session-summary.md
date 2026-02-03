# MVP Dashboard Click-to-Query - Session Summary

## Session Date: 2026-02-02

## Vision
Users can go from fresh install to querying their replicated data in Trino **entirely through the dashboard** in under 15 minutes.

---

## Completed This Session

### 1. Created MVP Milestone & Issues
- **Milestone**: [MVP: Dashboard Click-to-Query](https://github.com/janovincze/philotes/milestone/7)
- **Issue #97**: Integrated Query Page with Monaco Editor (HIGH) - DONE
- **Issue #98**: Unified Quick Start Wizard (HIGH) - TODO
- **Issue #99**: Dashboard E2E Tests with Playwright (MEDIUM) - TODO
- **Issue #100**: Data Preview in Verification Step (MEDIUM) - TODO

### 2. Query Page Implementation (PR #101)
**Branch**: `feature/97-query-page`
**PR**: https://github.com/janovincze/philotes/pull/101

#### Backend Changes
- `internal/api/handlers/query.go` - Added `ExecuteQuery` handler
- `internal/api/models/query.go` - Added `QueryExecuteRequest`, `QueryExecuteResponse`, `QueryColumn`
- `internal/api/services/query.go` - Added `ExecuteUserQuery` with validation

**Endpoint**: `POST /api/v1/query/execute`
```json
{
  "sql": "SELECT * FROM iceberg.public.customers LIMIT 10",
  "catalog": "iceberg",
  "limit": 100
}
```

**Security**:
- SELECT-only validation (blocks INSERT, UPDATE, DELETE, etc.)
- Row limit enforcement (default 100, max 1000)
- Query timeout (30 seconds)

#### Frontend Changes
- `web/src/app/query/page.tsx` - Query page route
- `web/src/components/query/sql-editor.tsx` - Monaco editor with SQL highlighting
- `web/src/components/query/results-table.tsx` - Sortable table with CSV export
- `web/src/components/query/query-templates.tsx` - Sample query dropdown
- `web/src/lib/hooks/use-query.ts` - TanStack Query hooks
- `web/src/components/layout/main-nav.tsx` - Added "Query" nav item
- `web/src/lib/api/types.ts` - Added query-related types

**Features**:
- Monaco Editor with SQL syntax highlighting
- Ctrl+Enter to execute
- Auto-complete for SQL keywords
- Sortable, paginated results table
- CSV export
- Query templates dropdown
- Error display

---

## TODO Next Session

### Priority 1: Unified Quick Start Wizard (Issue #98)

**Goal**: Consolidate 13-step setup into streamlined 5-step flow.

**New Route**: `/quickstart`

**Steps**:
1. **Quick Start** - Health check + welcome (no auth required)
2. **Connect** - Database credentials with inline test
3. **Select Tables** - Checkbox selection, auto-create pipeline
4. **Verify & Preview** - Show pipeline status, display sample rows
5. **Complete** - Success with links to Query/Pipelines

**Files to Create**:
- `web/src/app/quickstart/page.tsx`
- `web/src/components/quickstart/quickstart-wizard.tsx`
- `web/src/components/quickstart/steps/step-quick-start.tsx`
- `web/src/components/quickstart/steps/step-connect.tsx`
- `web/src/components/quickstart/steps/step-select-tables.tsx`
- `web/src/components/quickstart/steps/step-verify.tsx`
- `web/src/components/quickstart/steps/step-complete.tsx`

**Key Decisions**:
- Skip authentication in wizard (defer to Settings)
- Auto-generate pipeline name from source name
- Auto-start pipeline after creation
- Show sample data in verify step using Query API

**Reuse Existing**:
- Connection form from `web/src/components/setup/step-connect.tsx`
- Table selection from `web/src/components/setup/step-tables.tsx`
- Health check from `web/src/components/onboarding/step-cluster-health.tsx`
- Confetti animation for success

### Priority 2: Dashboard E2E Tests (Issue #99)

**Goal**: Test actual user journey through UI with Playwright.

**Setup**:
```
tests/e2e/dashboard/
  playwright.config.ts
  global-setup.ts
  global-teardown.ts
  pages/
    quickstart-wizard.page.ts
    query.page.ts
    pipelines.page.ts
  specs/
    quickstart.spec.ts
    query.spec.ts
    pipeline-management.spec.ts
```

**Test Scenarios**:
1. Quickstart flow (connect → select → verify → complete)
2. Query execution and results display
3. Pipeline start/stop management

**CI Integration**: Add Playwright job to `.github/workflows/ci.yml`

### Priority 3: Data Preview Enhancement (Issue #100)

**Goal**: "See your data" moment in verification step.

**File to Modify**: `web/src/components/onboarding/step-data-verification.tsx`

**Enhancements**:
- Display actual sample rows in table (reuse ResultsTable component)
- Show schema information (columns, types)
- Row count badge
- "Open in Query Editor" button

---

## Key Decisions Made

1. **SQL Editor**: Full Monaco editor (VS Code-like experience)
2. **Authentication**: Skip in quickstart wizard for faster time-to-value
3. **Query Restrictions**: SELECT-only for MVP security
4. **Issue Organization**: New "MVP: Dashboard Click-to-Query" milestone

---

## Reference Files

### Existing Components to Reuse
- `web/src/components/setup/setup-wizard.tsx` - Wizard pattern
- `web/src/components/setup/step-connect.tsx` - Connection form
- `web/src/components/setup/step-tables.tsx` - Table selection
- `web/src/components/onboarding/step-cluster-health.tsx` - Health check
- `web/src/components/query/results-table.tsx` - Data display (NEW)

### API Endpoints
- `POST /api/v1/query/execute` - Execute SQL (NEW)
- `GET /api/v1/query/catalogs` - List catalogs
- `GET /api/v1/sources` - List sources
- `POST /api/v1/sources` - Create source
- `GET /api/v1/sources/{id}/test-connection` - Test connection
- `GET /api/v1/sources/{id}/tables` - Discover tables
- `POST /api/v1/pipelines` - Create pipeline
- `POST /api/v1/pipelines/{id}/start` - Start pipeline
- `GET /api/v1/onboarding/verify-data` - Verify data arrival

---

## Estimated Remaining Effort

| Task | Effort |
|------|--------|
| Quick Start Wizard | 4-5 days |
| E2E Tests | 2-3 days |
| Data Preview | 1-2 days |
| **Total** | **7-10 days** |

---

## How to Continue

1. Merge PR #101 (Query Page)
2. Create branch `feature/98-quickstart-wizard`
3. Start with wizard skeleton and step navigation
4. Implement each step, reusing existing components
5. Add E2E tests after wizard is complete
