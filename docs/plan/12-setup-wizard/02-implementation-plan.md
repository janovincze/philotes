# Implementation Plan: Issue #12 - Setup Wizard

## Summary

Build a 6-step wizard that guides users through creating their first CDC pipeline. Backend APIs are complete - this is primarily frontend work with minor API client fixes.

## Approach

Create a new `/setup` route with a multi-step wizard component. Each step is a separate component that shares state via a wizard context. The wizard accumulates data through steps and creates resources (source, pipeline) at appropriate points.

## Wizard Steps

| Step | Name | Purpose | API Calls |
|------|------|---------|-----------|
| 1 | Welcome | Introduction, explain what they'll accomplish | None |
| 2 | Connect Database | Enter credentials, test connection | `POST /sources`, `POST /sources/:id/test` |
| 3 | Select Tables | Browse discovered tables, select which to replicate | `GET /sources/:id/tables` |
| 4 | Configure | Pipeline name, optional settings | None |
| 5 | Review | Summary before creation | None |
| 6 | Success | Create pipeline, show celebration | `POST /pipelines`, `POST /pipelines/:id/start` |

## Files to Create

### API Layer (2 files)
- `web/src/lib/api/sources.ts` - **Modify** to fix `discoverTables` return type
- `web/src/lib/hooks/use-setup.ts` - New hooks for wizard (test connection, discover tables)

### Components (8 files)
- `web/src/components/setup/setup-wizard.tsx` - Main wizard container with step management
- `web/src/components/setup/wizard-progress.tsx` - Step indicator/progress bar
- `web/src/components/setup/step-welcome.tsx` - Welcome step
- `web/src/components/setup/step-connect.tsx` - Database connection form
- `web/src/components/setup/step-tables.tsx` - Table selection with checkboxes
- `web/src/components/setup/step-configure.tsx` - Pipeline name and settings
- `web/src/components/setup/step-review.tsx` - Review summary
- `web/src/components/setup/step-success.tsx` - Success celebration

### Pages (1 file)
- `web/src/app/setup/page.tsx` - Setup wizard page

### Types (1 file)
- `web/src/lib/api/types.ts` - **Modify** to add `TableInfo`, `ColumnInfo`, `TableDiscoveryResponse`

## Component Architecture

```
SetupWizard (state management)
├── WizardProgress (step indicator)
├── StepWelcome
├── StepConnect
│   ├── ConnectionForm (credentials input)
│   └── ConnectionTest (test result display)
├── StepTables
│   └── TableSelector (checkbox list with search)
├── StepConfigure
│   └── PipelineConfigForm
├── StepReview
│   └── ReviewSummary
└── StepSuccess
    └── SuccessCelebration
```

## State Management

Use React Context for wizard state:

```typescript
interface WizardState {
  currentStep: number
  source: {
    name: string
    host: string
    port: number
    database_name: string
    username: string
    password: string
    ssl_mode: string
  } | null
  sourceId: string | null
  connectionTested: boolean
  selectedTables: string[]
  pipelineName: string
  pipelineId: string | null
}
```

## Task Breakdown

### Phase 1: API Layer
1. Add `TableInfo`, `ColumnInfo`, `TableDiscoveryResponse` types
2. Fix `sourcesApi.discoverTables()` to return full response
3. Create `useTestConnection()` mutation hook
4. Create `useDiscoverTables()` query hook

### Phase 2: Wizard Infrastructure
5. Create `WizardProgress` step indicator component
6. Create `SetupWizard` container with step management
7. Create wizard context and state management

### Phase 3: Wizard Steps
8. Create `StepWelcome` - intro and getting started
9. Create `StepConnect` - database connection form with test
10. Create `StepTables` - table discovery and selection
11. Create `StepConfigure` - pipeline name and settings
12. Create `StepReview` - summary before creation
13. Create `StepSuccess` - celebration and next steps

### Phase 4: Integration
14. Create `/setup` page
15. Add setup entry point (first-run detection or manual access)

## UI Design Notes

### Step Indicator
- Horizontal progress bar with numbered steps
- Current step highlighted, completed steps checked
- Step titles below numbers

### Connection Form
- Card layout with database icon
- Fields: Host, Port, Database, Username, Password, SSL Mode
- "Test Connection" button with loading state
- Success/error feedback with server info on success

### Table Selector
- Search input at top
- Checkbox list of tables grouped by schema
- Show column count per table
- "Select All" / "Deselect All" buttons
- Selected count indicator

### Review Summary
- Card layout showing:
  - Source details (host, database, user)
  - Selected tables count
  - Pipeline name
- "Create Pipeline" CTA button

### Success Screen
- Confetti animation or celebration icon
- "Pipeline created successfully" message
- "View Pipeline" and "Create Another" buttons

## Validation

### Step 2 (Connect)
- All fields required
- Port: 1-65535
- Host: non-empty
- Connection must be tested successfully before proceeding

### Step 3 (Tables)
- At least one table must be selected

### Step 4 (Configure)
- Pipeline name required, 1-255 chars

## Error Handling

- Connection test failures: Show error message with retry button
- Table discovery failures: Show error with retry
- Pipeline creation failures: Show error, allow retry without losing state

## Estimate

~2,500-3,000 LOC (lower than original 6,000 estimate due to existing infrastructure)

## Verification

```bash
# Build and lint
cd web && npm run build && npm run lint

# Manual testing
1. Navigate to /setup
2. Complete wizard flow end-to-end
3. Verify pipeline appears in /pipelines
4. Test error states (wrong credentials, no tables selected)
```
