# Issue #98 - Unified Quick Start Wizard

## Issue Details

**Title:** feat(dashboard): Unified Quick Start Wizard (5 steps)
**State:** OPEN
**Labels:** epic:dashboard, phase:mvp, priority:high
**Milestone:** MVP: Dashboard Click-to-Query

## Goal

Create a streamlined 5-step wizard that replaces the current 13+ step setup/onboarding flow, getting users from fresh install to seeing their data in under 15 minutes.

## Problem Statement

Current flow has two separate wizards (Setup: 6 steps, Onboarding: 7 steps) that are confusing and include steps not critical for MVP (admin user, SSO, alerts). Users want to see value quickly.

## Current vs Proposed Flow

| Current (13 steps) | Proposed (5 steps) |
|-------------------|-------------------|
| Welcome | **1. Quick Start** (Health + Welcome) |
| Health Check | |
| Admin User | (Defer to Settings) |
| SSO Setup | (Defer to Settings) |
| Connect | **2. Connect** (Database credentials) |
| Source DB | |
| Tables | **3. Select Tables** (Pick what to replicate) |
| Configure | (Auto-generate pipeline name) |
| Pipeline | |
| Review | **4. Verify & Preview** (See your data!) |
| Verify Data | |
| Alerts | (Defer to Settings) |
| Done | **5. Complete** (Success + Query access) |

## Acceptance Criteria

- [ ] New `/quickstart` route with unified wizard
- [ ] Step 1: Health check + welcome (no auth required)
- [ ] Step 2: Database connection with inline test
- [ ] Step 3: Table selection with "Select All" option
- [ ] Step 4: Data verification with sample row preview
- [ ] Step 5: Success screen with links to Query/Pipelines
- [ ] Auto-create source and pipeline (minimal user input)
- [ ] Smart defaults (port 5432, SSL prefer)
- [ ] Redirect `/setup` and `/onboarding` to `/quickstart`

## Technical Notes

- Reuse existing connection form components
- Leverage existing health check endpoint
- Use confetti animation (already exists) for success

## Dependencies

- Requires #97 (Query Page) for "Query Your Data" link in completion step - **COMPLETED**

## Related Files (from session summary)

### Existing Components to Reuse
- `web/src/components/setup/setup-wizard.tsx` - Wizard pattern
- `web/src/components/setup/step-connect.tsx` - Connection form
- `web/src/components/setup/step-tables.tsx` - Table selection
- `web/src/components/onboarding/step-cluster-health.tsx` - Health check
- `web/src/components/query/results-table.tsx` - Data display

### API Endpoints
- `GET /api/v1/health` - Health check
- `POST /api/v1/sources` - Create source
- `GET /api/v1/sources/{id}/test-connection` - Test connection
- `GET /api/v1/sources/{id}/tables` - Discover tables
- `POST /api/v1/pipelines` - Create pipeline
- `POST /api/v1/pipelines/{id}/start` - Start pipeline
- `GET /api/v1/onboarding/verify-data` - Verify data arrival
- `POST /api/v1/query/execute` - Execute query for preview
