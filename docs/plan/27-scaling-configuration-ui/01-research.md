# Research Findings - Issue #27: Scaling Configuration UI

## Executive Summary

**Key Finding:** The scaling backend (SCALE-001) is **FULLY IMPLEMENTED** and production-ready. This issue is purely frontend work - building the dashboard UI to interact with existing APIs.

**No blockers found.** Implementation can proceed immediately.

---

## 1. Existing Backend Infrastructure

### API Endpoints Available

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/scaling/policies` | Create policy |
| GET | `/api/v1/scaling/policies` | List policies |
| GET | `/api/v1/scaling/policies/:id` | Get policy |
| PUT | `/api/v1/scaling/policies/:id` | Update policy |
| DELETE | `/api/v1/scaling/policies/:id` | Delete policy |
| POST | `/api/v1/scaling/policies/:id/enable` | Enable policy |
| POST | `/api/v1/scaling/policies/:id/disable` | Disable policy |
| POST | `/api/v1/scaling/policies/:id/evaluate` | Evaluate policy (dry-run) |
| GET | `/api/v1/scaling/policies/:id/state` | Get current scaling state |
| GET | `/api/v1/scaling/history` | List scaling history |
| GET | `/api/v1/scaling/policies/:id/history` | Get policy history |

### Backend Files

- `internal/scaling/types.go` - Policy, Rule, Schedule, History, State types
- `internal/scaling/service.go` - Business logic
- `internal/scaling/manager.go` - Background evaluation
- `internal/scaling/evaluator.go` - Prometheus metric evaluation
- `internal/scaling/executor.go` - Scaling execution
- `internal/scaling/repository.go` - Database persistence
- `internal/api/handlers/scaling.go` - HTTP handlers
- `internal/api/services/scaling.go` - API service layer
- `internal/api/models/scaling.go` - Request/response models

### Data Model

```go
type Policy struct {
    ID              uuid.UUID
    Name            string
    TargetType      TargetType  // "cdc-worker", "trino", "risingwave", "nodes"
    TargetID        *uuid.UUID
    MinReplicas     int
    MaxReplicas     int
    CooldownSeconds int
    MaxHourlyCost   *float64
    ScaleToZero     bool
    Enabled         bool
    ScaleUpRules    []Rule
    ScaleDownRules  []Rule
    Schedules       []Schedule
}

type Rule struct {
    Metric          string   // e.g., "cdc_lag_seconds"
    Operator        string   // "gt", "lt", "gte", "lte", "eq"
    Threshold       float64
    DurationSeconds int
    ScaleBy         int
}

type Schedule struct {
    CronExpression  string
    DesiredReplicas int
    Timezone        string
    Enabled         bool
}

type History struct {
    ID               uuid.UUID
    PolicyID         *uuid.UUID
    PolicyName       string
    Action           string  // "scale_up", "scale_down", "scheduled", "manual"
    TargetType       string
    PreviousReplicas int
    NewReplicas      int
    Reason           string
    TriggeredBy      string
    DryRun           bool
    ExecutedAt       time.Time
}

type State struct {
    CurrentReplicas   int
    LastScaleTime     *time.Time
    LastScaleAction   string
}
```

---

## 2. Dashboard Framework Analysis

### Available Components (shadcn/ui)

- button, card, badge, input, label, select, switch, tabs
- sheet, dropdown-menu, separator, skeleton, avatar
- table, sonner (toasts)
- **Missing:** Form component (need react-hook-form + zod)

### Charts (recharts v3.7.0)

- LineChart, AreaChart, BarChart
- ResponsiveContainer
- Tooltip, Legend

### Existing Patterns

**API Client Pattern:**
```typescript
// web/src/lib/api/sources.ts
export const sourcesApi = {
  list(): Promise<Source[]>,
  get(id: string): Promise<Source>,
  create(input): Promise<Source>,
  update(id, input): Promise<Source>,
  delete(id): Promise<void>,
}
```

**Hooks Pattern:**
```typescript
// web/src/lib/hooks/use-pipelines.ts
export function usePipelines() { ... }
export function useCreatePipeline() { ... }
```

### Formatting Utilities Available

- `formatNumber()` - K, M, B suffixes
- `formatDuration()` - Human-readable duration
- `formatRelativeTime()` - "5m ago" format

---

## 3. Navigation Structure

Current nav items: Dashboard, Sources, Pipelines, Alerts, Settings

**Proposed:** Add "Scaling" as top-level navigation item

---

## 4. Files to Create

### Frontend Types & API
1. `web/src/lib/api/types.ts` - Add scaling types
2. `web/src/lib/api/scaling.ts` - scalingApi module
3. `web/src/lib/hooks/use-scaling.ts` - Custom hooks

### Pages
4. `web/src/app/scaling/page.tsx` - Policies list
5. `web/src/app/scaling/new/page.tsx` - Create policy
6. `web/src/app/scaling/[id]/page.tsx` - Policy detail
7. `web/src/app/scaling/[id]/edit/page.tsx` - Edit policy

### Components
8. `web/src/components/scaling/scaling-policy-card.tsx` - Policy card
9. `web/src/components/scaling/scaling-policy-form.tsx` - Create/edit form
10. `web/src/components/scaling/rule-editor.tsx` - Rules editor
11. `web/src/components/scaling/schedule-editor.tsx` - Schedules editor
12. `web/src/components/scaling/scaling-history-table.tsx` - History table
13. `web/src/components/scaling/scale-state-card.tsx` - Current state display

### Dependencies to Add
```json
{
  "react-hook-form": "^7.x",
  "@hookform/resolvers": "^3.x",
  "zod": "^3.x"
}
```

---

## 5. Prometheus Metrics for Rule Selector

Available metrics for scaling rules:
- `philotes_cdc_lag_seconds` - Replication lag
- `philotes_buffer_depth` - Buffer queue depth
- `philotes_cdc_events_total` - Total events processed
- `philotes_cdc_errors_total` - Error count
- CPU/Memory metrics (from node exporter)

---

## 6. Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|----------------|
| Scaling policy create/edit forms | ScalingPolicyForm component |
| Visual policy builder | RuleEditor, ScheduleEditor |
| Current scale visualization | ScaleStateCard |
| Scaling event timeline | ScalingHistoryTable |
| Cost tracking dashboard | MaxHourlyCost field (display only for MVP) |
| Policy simulation/preview | Evaluate API with dry_run=true |
| Alerts for scaling limits | Defer to alerting integration |
| Per-pipeline scaling configuration | TargetID filter in UI |

---

## 7. Recommended Approach

1. **Phase 1: Core Infrastructure**
   - Add form library dependencies
   - Create TypeScript types
   - Build API client and hooks

2. **Phase 2: Policy Management**
   - Policies list page
   - Create/edit forms with rule and schedule editors
   - Policy detail view with state

3. **Phase 3: History & Visualization**
   - Scaling history table
   - Current scale visualization
   - Policy simulation (dry-run)

4. **Phase 4: Navigation & Polish**
   - Add to main navigation
   - Loading states and error handling
   - Responsive design
