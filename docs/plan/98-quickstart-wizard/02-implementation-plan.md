# Implementation Plan - Issue #98 Unified Quick Start Wizard

## Overview

Create a streamlined 5-step wizard at `/quickstart` that consolidates the current 13-step setup/onboarding flow. The wizard reuses existing components while providing a faster path to value.

## Architecture

### Wizard State
```typescript
interface QuickstartState {
  currentStep: number          // 1-5
  // Step 2: Connect
  sourceFormData: SourceFormData
  source: Source | null
  connectionTested: boolean
  // Step 3: Tables
  availableTables: TableInfo[]
  selectedTables: string[]
  // Step 4: Verify
  pipeline: Pipeline | null
  verificationResult: DataVerificationResponse | null
}
```

### Step Flow

| Step | Component | Primary Action |
|------|-----------|----------------|
| 1 | StepWelcome | Health check + Get Started |
| 2 | StepConnect | Create source + Test connection |
| 3 | StepTables | Select tables to replicate |
| 4 | StepVerify | Auto-create pipeline, start, verify data |
| 5 | StepComplete | Success + Navigation to Query/Pipelines |

## Files to Create

### 1. Route: `web/src/app/quickstart/page.tsx`
```typescript
// Simple page wrapper for QuickstartWizard
import { QuickstartWizard } from "@/components/quickstart/quickstart-wizard"

export default function QuickstartPage() {
  return (
    <div className="container max-w-4xl py-8">
      <QuickstartWizard />
    </div>
  )
}
```

### 2. Main Wizard: `web/src/components/quickstart/quickstart-wizard.tsx`
- Central state management (similar to setup-wizard.tsx)
- 5-step navigation
- Renders appropriate step component
- Progress indicator

### 3. Step 1 - Welcome: `web/src/components/quickstart/step-welcome.tsx`
- Health check using `useClusterHealth()` hook
- Welcome message explaining 5-step process
- "Get Started" button (enabled when health OK)
- Show health status indicators

### 4. Step 4 - Verify: `web/src/components/quickstart/step-verify.tsx`
- Auto-generates pipeline name from source name
- Creates pipeline with selected tables
- Starts pipeline automatically
- Polls for data arrival using `useVerifyDataFlow()`
- Shows sample data preview using `ResultsTable` component
- Progress indicators for each sub-step

### 5. Step 5 - Complete: `web/src/components/quickstart/step-complete.tsx`
- Confetti animation (reuse existing)
- Success message with stats
- Links to:
  - Query page (`/query`)
  - Pipeline details (`/pipelines/{id}`)
  - Create another source

## Files to Modify

### 1. Redirect Setup: `web/src/app/setup/page.tsx`
Add redirect to `/quickstart`:
```typescript
import { redirect } from "next/navigation"
export default function SetupPage() {
  redirect("/quickstart")
}
```

### 2. Redirect Onboarding: `web/src/app/onboarding/page.tsx`
Add redirect to `/quickstart`:
```typescript
import { redirect } from "next/navigation"
export default function OnboardingPage() {
  redirect("/quickstart")
}
```

### 3. Navigation: `web/src/components/layout/main-nav.tsx`
Update "Setup" link to point to `/quickstart`

## Reused Components (No Modifications)

| Component | From | Usage |
|-----------|------|-------|
| `StepConnect` | `setup/step-connect.tsx` | Step 2 (reuse directly) |
| `StepTables` | `setup/step-tables.tsx` | Step 3 (reuse directly) |
| `ResultsTable` | `query/results-table.tsx` | Step 4 data preview |
| `WizardProgress` | `setup/wizard-progress.tsx` | Progress indicator |

## Implementation Tasks

### Task 1: Create Quickstart Route and Wizard Shell
- Create `web/src/app/quickstart/page.tsx`
- Create `web/src/components/quickstart/quickstart-wizard.tsx` with state management
- Create `web/src/components/quickstart/wizard-progress.tsx` (5-step version)

### Task 2: Implement Step 1 (Welcome)
- Create `web/src/components/quickstart/step-welcome.tsx`
- Integrate `useClusterHealth()` hook
- Health status display with icons
- "Get Started" button

### Task 3: Wire Up Steps 2 & 3 (Connect & Tables)
- Import and use existing `StepConnect` component
- Import and use existing `StepTables` component
- Wire to wizard state

### Task 4: Implement Step 4 (Verify & Preview)
- Create `web/src/components/quickstart/step-verify.tsx`
- Auto-generate pipeline name
- Create pipeline via `useCreatePipeline()`
- Start pipeline via `useStartPipeline()`
- Verify data via `useVerifyDataFlow()`
- Display sample data with `ResultsTable`

### Task 5: Implement Step 5 (Complete)
- Create `web/src/components/quickstart/step-complete.tsx`
- Confetti animation
- Navigation links

### Task 6: Add Redirects
- Update `web/src/app/setup/page.tsx` to redirect
- Update `web/src/app/onboarding/page.tsx` to redirect
- Update navigation link

### Task 7: Testing & Polish
- Test full flow end-to-end
- Handle edge cases (connection failures, no tables, etc.)
- Loading states and error handling

## API Endpoints Used

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/onboarding/cluster/health` | GET | Health check |
| `/api/v1/sources` | POST | Create source |
| `/api/v1/sources/{id}/test` | POST | Test connection |
| `/api/v1/sources/{id}/tables` | GET | Discover tables |
| `/api/v1/pipelines` | POST | Create pipeline |
| `/api/v1/pipelines/{id}/start` | POST | Start pipeline |
| `/api/v1/onboarding/data/verify` | POST | Verify data arrival |
| `/api/v1/query/execute` | POST | Query for preview |

## Smart Defaults

- **Port**: 5432 (PostgreSQL default)
- **SSL Mode**: "prefer"
- **Source Name**: Auto-suggest from hostname
- **Pipeline Name**: `{source_name}-pipeline`
- **Table Selection**: Pre-select all discovered tables

## Error Handling

| Scenario | Handling |
|----------|----------|
| Health check fails | Show warning, disable proceed |
| Connection test fails | Show error, allow retry |
| No tables found | Show message, allow back |
| Pipeline creation fails | Show error, allow retry |
| Data verification timeout | Show warning, allow proceed anyway |

## Success Criteria

- [ ] User can complete wizard in < 5 minutes (excluding data sync time)
- [ ] All acceptance criteria from issue #98 met
- [ ] Existing `/setup` and `/onboarding` redirect to `/quickstart`
- [ ] Full flow works with sample database
