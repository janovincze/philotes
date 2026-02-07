# Implementation Plan: Issue #99 - E2E Tests with Playwright

## Overview

Set up Playwright E2E testing for the dashboard with page object models, three test suites covering core user journeys, and CI integration.

## Structure

```
tests/e2e/dashboard/
├── playwright.config.ts        # Playwright configuration
├── global-setup.ts             # Docker/API health wait
├── fixtures.ts                 # Custom test fixtures
├── pages/                      # Page Object Models
│   ├── base.page.ts
│   ├── quickstart.page.ts
│   ├── query.page.ts
│   └── pipelines.page.ts
├── quickstart.spec.ts          # Quickstart wizard flow test
├── query.spec.ts               # Query page flow test
└── pipelines.spec.ts           # Pipeline management test
```

## Tasks

### Task 1: Playwright Setup
- Install `@playwright/test` as devDependency in `web/package.json`
- Add `test:e2e` script to `web/package.json`
- Create `tests/e2e/dashboard/playwright.config.ts` with:
  - Base URL: `http://localhost:3001` (Next.js dev port)
  - Chromium-only for speed in CI
  - Timeout: 30s per test, 60s navigation
  - Screenshot on failure
  - HTML reporter

### Task 2: Global Setup & Fixtures
- `global-setup.ts`: Wait for API (`localhost:8080/health`) and Next.js (`localhost:3001`) to be ready
- `fixtures.ts`: Custom test fixture that provides page objects

### Task 3: Page Object Models
- `base.page.ts`: Common navigation, sidebar, toast helpers
- `quickstart.page.ts`: Wizard step navigation, form filling, verification
- `query.page.ts`: SQL editor interaction, template selection, results checking
- `pipelines.page.ts`: Pipeline list, status checking, start/stop actions

### Task 4: Add data-testid Attributes
Add minimal `data-testid` attributes to key interactive elements:
- Quickstart wizard: steps, buttons, form inputs
- Query page: editor, execute button, results table, templates
- Pipelines page: pipeline cards, status badges, action buttons
- Layout: sidebar nav items

### Task 5: Test Suites
- `quickstart.spec.ts`: Navigate → fill form → test connection → select tables → verify → complete
- `query.spec.ts`: Navigate → select template → execute → verify results → export CSV
- `pipelines.spec.ts`: Navigate → verify list → check status → interact with pipeline

### Task 6: CI Integration
- Add `test-e2e-dashboard` job to `.github/workflows/ci.yml`
- Requires: Docker Compose services, Node.js, pnpm, Playwright browsers
- Runs after Go tests
- Upload test report as artifact

## Key Decisions
- Tests live in `tests/e2e/dashboard/` (alongside existing Go E2E tests in `tests/e2e/`)
- Use pnpm (project already uses it via pnpm-lock.yaml)
- Chromium-only in CI for speed; all browsers locally
- Page objects provide stable selectors via data-testid
- Tests assume Docker environment is running (global-setup waits for readiness)
