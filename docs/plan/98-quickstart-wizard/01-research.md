# Research Findings - Issue #98 Unified Quick Start Wizard

## 1. Existing Wizard Patterns

### Setup Wizard (6 steps)
- **File**: `web/src/components/setup/setup-wizard.tsx`
- **Pattern**: Uses central state management with `WizardState` interface
- **Step Components**:
  1. `step-welcome.tsx` - Welcome screen with overview
  2. `step-connect.tsx` - Database connection form with inline testing
  3. `step-tables.tsx` - Table discovery and selection
  4. `step-configure.tsx` - Pipeline naming (auto-generates suggestions)
  5. `step-review.tsx` - Configuration review and pipeline creation
  6. `step-success.tsx` - Success screen with navigation options

### Onboarding Wizard (7 steps with persistence)
- **File**: `web/src/components/onboarding/onboarding-wizard.tsx`
- **Features**: Progress persistence, session tracking, optional steps support

## 2. Key Reusable Components

### Connection Form (Fully Reusable)
- **File**: `web/src/components/setup/step-connect.tsx`
- **Props**: `formData`, `onFormDataChange`, `source`, `onSourceCreated`, `connectionTested`, `onConnectionTested`, `onNext`, `onBack`
- **Capabilities**: Form validation, source creation, connection test with feedback, SSL mode selection

### Table Selection (Fully Reusable)
- **File**: `web/src/components/setup/step-tables.tsx`
- **Props**: `sourceId`, `availableTables`, `onTablesLoaded`, `selectedTables`, `onSelectedTablesChange`, `onNext`, `onBack`
- **Capabilities**: Auto-discovery, searchable list, Select All/Deselect All, loading states

### Data Verification
- **File**: `web/src/components/onboarding/step-data-verification.tsx`
- **Capabilities**: Polls pipeline status, attempts verification, displays row count, sample row preview

### Health Check
- **Hook**: `useClusterHealth()` from `web/src/lib/hooks/use-onboarding.ts`
- **Component**: `web/src/components/onboarding/step-cluster-health.tsx`

### Results Table
- **File**: `web/src/components/query/results-table.tsx`
- **Capabilities**: Sortable columns, pagination, CSV export

## 3. API Integration

### Available Hooks
```typescript
// Sources
useCreateSource() -> Promise<Source>
useTestSourceConnection(id) -> Promise<ConnectionTestResult>
useDiscoverTables(sourceId) -> Promise<TableDiscoveryResponse>

// Pipelines
useCreatePipeline() -> Promise<Pipeline>
useStartPipeline(id) -> Promise<Pipeline>
usePipelineStatus(id) -> polls every 2s

// Onboarding
useClusterHealth(enabled?, refetchInterval?) -> Promise<ClusterHealthResponse>
useVerifyDataFlow(data) -> Promise<DataVerificationResponse>

// Query
useQueryExecute() -> Promise<QueryExecuteResponse>
```

## 4. Step Mapping for 5-Step Wizard

| Step | Content | Reuse |
|------|---------|-------|
| 1. Quick Start | Health check + Welcome | `useClusterHealth()` hook |
| 2. Connect | Database credentials + test | `StepConnect` component |
| 3. Select Tables | Table discovery + selection | `StepTables` component |
| 4. Verify & Preview | Data verification + sample rows | `useVerifyDataFlow()` + `ResultsTable` |
| 5. Complete | Success + links | Custom with confetti |

## 5. Auto-Generation Strategy

- **Source Name**: `{hostname}-source` (e.g., "prod-db-source")
- **Pipeline Name**: `{source_name}-pipeline`
- **Tables**: Pre-select all discovered tables
- **Defaults**: port 5432, SSL prefer

## 6. Files to Create

1. `web/src/app/quickstart/page.tsx` - Route
2. `web/src/components/quickstart/quickstart-wizard.tsx` - Main wizard
3. `web/src/components/quickstart/step-welcome.tsx` - Health + welcome
4. `web/src/components/quickstart/step-preview.tsx` - Data preview
5. `web/src/components/quickstart/step-complete.tsx` - Success

## 7. Files to Modify

1. `web/src/app/setup/page.tsx` - Redirect to `/quickstart`
2. `web/src/app/onboarding/page.tsx` - Redirect to `/quickstart`

## 8. No Blockers

All necessary APIs and components exist. Key consideration: data verification may need 30+ seconds for first data to appear.
