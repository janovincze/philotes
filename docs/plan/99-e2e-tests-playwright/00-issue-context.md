# Issue #99 - E2E Tests with Playwright

## Title
test(dashboard): E2E Tests with Playwright

## Goal
Create end-to-end tests that verify the actual user journey through the dashboard UI using Playwright.

## Acceptance Criteria
- Playwright configuration (`tests/e2e/dashboard/playwright.config.ts`)
- Global setup/teardown for Docker environment
- Page object models for reusability
- Test scenarios: Quickstart flow, Query flow, Pipeline management
- CI integration in `.github/workflows/ci.yml`

## Test Scenarios
1. **Quickstart Flow** - /quickstart → health check → DB credentials → test connection → select tables → pipeline starts → data preview → complete → visible on /pipelines
2. **Query Flow** - /query → select template → execute → verify results → export CSV
3. **Pipeline Management** - /pipelines → stop → verify status → start → verify metrics

## Dependencies
- #97 (Query Page) — exists
- #98 (Quick Start Wizard) — exists

## Labels
- epic:dashboard, phase:mvp, priority:medium
- Milestone: MVP: Dashboard Click-to-Query
