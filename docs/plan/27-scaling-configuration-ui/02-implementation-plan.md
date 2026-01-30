# Implementation Plan - Issue #27: Scaling Configuration UI

## Overview

Build a comprehensive dashboard UI for configuring and monitoring auto-scaling policies. The backend API is already complete (SCALE-001), so this is purely frontend work.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend (Next.js)                            │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    /scaling route                              │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐  │  │
│  │  │ List Page  │  │ Detail Page│  │ Create/Edit Form Page  │  │  │
│  │  └────────────┘  └────────────┘  └────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Components                                 │  │
│  │  ScalingPolicyCard │ ScalingPolicyForm │ RuleEditor          │  │
│  │  ScheduleEditor    │ ScalingHistoryTable│ ScaleStateCard     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │               API Client + React Query Hooks                   │  │
│  │  scalingApi │ useScalingPolicies │ useCreatePolicy            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Backend API (Already Implemented)                │
│  POST/GET/PUT/DELETE /api/v1/scaling/policies                       │
│  GET /api/v1/scaling/history                                        │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Task Breakdown

### Phase 1: Dependencies & Types (Day 1)

#### Task 1.1: Add Form Dependencies
```bash
cd web && npm install react-hook-form @hookform/resolvers zod
```

#### Task 1.2: Add shadcn/ui Form Component
```bash
npx shadcn@latest add form
```

#### Task 1.3: Add TypeScript Types
**File:** `web/src/lib/api/types.ts` (modify)

```typescript
// Scaling Types
export type ScalingTargetType = "cdc-worker" | "trino" | "risingwave" | "nodes"
export type ScalingAction = "scale_up" | "scale_down" | "scheduled" | "manual"
export type RuleOperator = "gt" | "lt" | "gte" | "lte" | "eq"

export interface ScalingRule {
  metric: string
  operator: RuleOperator
  threshold: number
  duration_seconds: number
  scale_by: number
}

export interface ScalingSchedule {
  cron_expression: string
  desired_replicas: number
  timezone: string
  enabled: boolean
}

export interface ScalingPolicy {
  id: string
  name: string
  target_type: ScalingTargetType
  target_id?: string
  min_replicas: number
  max_replicas: number
  cooldown_seconds: number
  max_hourly_cost?: number
  scale_to_zero: boolean
  enabled: boolean
  scale_up_rules: ScalingRule[]
  scale_down_rules: ScalingRule[]
  schedules: ScalingSchedule[]
  created_at: string
  updated_at: string
}

export interface ScalingState {
  policy_id: string
  current_replicas: number
  last_scale_time?: string
  last_scale_action?: string
}

export interface ScalingHistory {
  id: string
  policy_id?: string
  policy_name: string
  action: ScalingAction
  target_type: ScalingTargetType
  target_id?: string
  previous_replicas: number
  new_replicas: number
  reason: string
  triggered_by: string
  dry_run: boolean
  executed_at: string
}

export interface CreateScalingPolicyInput {
  name: string
  target_type: ScalingTargetType
  target_id?: string
  min_replicas: number
  max_replicas: number
  cooldown_seconds?: number
  max_hourly_cost?: number
  scale_to_zero?: boolean
  enabled?: boolean
  scale_up_rules?: ScalingRule[]
  scale_down_rules?: ScalingRule[]
  schedules?: ScalingSchedule[]
}
```

#### Task 1.4: Create API Client
**File:** `web/src/lib/api/scaling.ts` (new)

```typescript
export const scalingApi = {
  // Policies
  listPolicies(): Promise<ScalingPolicy[]>
  getPolicy(id: string): Promise<ScalingPolicy>
  createPolicy(input: CreateScalingPolicyInput): Promise<ScalingPolicy>
  updatePolicy(id: string, input: Partial<CreateScalingPolicyInput>): Promise<ScalingPolicy>
  deletePolicy(id: string): Promise<void>
  enablePolicy(id: string): Promise<ScalingPolicy>
  disablePolicy(id: string): Promise<ScalingPolicy>
  evaluatePolicy(id: string, dryRun?: boolean): Promise<EvaluationResult>

  // State
  getPolicyState(id: string): Promise<ScalingState>

  // History
  listHistory(policyId?: string): Promise<ScalingHistory[]>
}
```

#### Task 1.5: Create React Query Hooks
**File:** `web/src/lib/hooks/use-scaling.ts` (new)

- `useScalingPolicies()` - List policies with polling
- `useScalingPolicy(id)` - Get single policy
- `useScalingState(id)` - Get policy state with polling
- `useScalingHistory(policyId?)` - Get history
- `useCreateScalingPolicy()` - Mutation
- `useUpdateScalingPolicy()` - Mutation
- `useDeleteScalingPolicy()` - Mutation
- `useEnableScalingPolicy()` - Mutation
- `useDisableScalingPolicy()` - Mutation

---

### Phase 2: Core Components (Day 2)

#### Task 2.1: Scaling Policy Card
**File:** `web/src/components/scaling/scaling-policy-card.tsx` (new)

Display policy summary:
- Name, target type, status badge (enabled/disabled)
- Min/max replicas
- Number of rules and schedules
- Action buttons: View, Edit, Enable/Disable, Delete

#### Task 2.2: Scale State Card
**File:** `web/src/components/scaling/scale-state-card.tsx` (new)

Display current scaling state:
- Current replicas (large number)
- Min/max bounds
- Last scale action and time
- Visual gauge/progress bar

#### Task 2.3: Rule Editor Component
**File:** `web/src/components/scaling/rule-editor.tsx` (new)

Edit scaling rules:
- Metric selector (dropdown with available metrics)
- Operator selector (gt, lt, gte, lte, eq)
- Threshold input (number)
- Duration input (seconds)
- Scale by input (positive/negative number)
- Add/remove rule buttons

#### Task 2.4: Schedule Editor Component
**File:** `web/src/components/scaling/schedule-editor.tsx` (new)

Edit schedules:
- Cron expression input with helper text
- Desired replicas input
- Timezone selector
- Enabled toggle
- Add/remove schedule buttons

---

### Phase 3: Policy Form (Day 3)

#### Task 3.1: Scaling Policy Form
**File:** `web/src/components/scaling/scaling-policy-form.tsx` (new)

Full form with sections:
1. **Basic Info:** Name, Target Type, Target ID (optional pipeline selector)
2. **Scale Limits:** Min replicas, Max replicas, Cooldown seconds
3. **Cost Control:** Max hourly cost (optional), Scale to zero toggle
4. **Scale Up Rules:** RuleEditor instances
5. **Scale Down Rules:** RuleEditor instances
6. **Schedules:** ScheduleEditor instances
7. **Status:** Enabled toggle

Use react-hook-form with zod validation.

---

### Phase 4: Pages (Day 4)

#### Task 4.1: Scaling Policies List Page
**File:** `web/src/app/scaling/page.tsx` (new)

- Page header with title and "New Policy" button
- Grid of ScalingPolicyCard components
- Empty state if no policies
- Loading skeleton

#### Task 4.2: Create Policy Page
**File:** `web/src/app/scaling/new/page.tsx` (new)

- Back link to list
- ScalingPolicyForm in create mode
- Success redirect to policy detail

#### Task 4.3: Policy Detail Page
**File:** `web/src/app/scaling/[id]/page.tsx` (new)

- Back link to list
- Policy header with name, status, actions
- ScaleStateCard showing current state
- Policy configuration summary (read-only)
- Recent scaling history (last 10 events)
- Edit/Delete buttons

#### Task 4.4: Edit Policy Page
**File:** `web/src/app/scaling/[id]/edit/page.tsx` (new)

- Back link to detail
- ScalingPolicyForm in edit mode
- Success redirect to detail page

---

### Phase 5: History & Navigation (Day 5)

#### Task 5.1: Scaling History Table
**File:** `web/src/components/scaling/scaling-history-table.tsx` (new)

- Columns: Time, Policy, Action, Target, Replicas (prev→new), Reason
- Action type badges with colors
- Sortable by time
- Filter by policy (optional)

#### Task 5.2: History Section/Page
Add history section to detail page and optionally `/scaling/history` global page.

#### Task 5.3: Update Navigation
**File:** `web/src/components/layout/sidebar.tsx` (modify)

Add Scaling nav item with icon (Scale or Maximize2).

#### Task 5.4: Export API Index
**File:** `web/src/lib/api/index.ts` (modify)

Add `scalingApi` export.

---

## File Summary

### New Files (13)

| File | Description |
|------|-------------|
| `web/src/lib/api/scaling.ts` | API client module |
| `web/src/lib/hooks/use-scaling.ts` | React Query hooks |
| `web/src/components/scaling/scaling-policy-card.tsx` | Policy card |
| `web/src/components/scaling/scaling-policy-form.tsx` | Create/edit form |
| `web/src/components/scaling/rule-editor.tsx` | Rules editor |
| `web/src/components/scaling/schedule-editor.tsx` | Schedules editor |
| `web/src/components/scaling/scaling-history-table.tsx` | History table |
| `web/src/components/scaling/scale-state-card.tsx` | State display |
| `web/src/app/scaling/page.tsx` | List page |
| `web/src/app/scaling/new/page.tsx` | Create page |
| `web/src/app/scaling/[id]/page.tsx` | Detail page |
| `web/src/app/scaling/[id]/edit/page.tsx` | Edit page |
| `web/src/components/ui/form.tsx` | shadcn form component |

### Modified Files (3)

| File | Change |
|------|--------|
| `web/src/lib/api/types.ts` | Add scaling types |
| `web/src/lib/api/index.ts` | Export scalingApi |
| `web/src/components/layout/sidebar.tsx` | Add Scaling nav |

---

## Validation Schema (Zod)

```typescript
const scalingPolicySchema = z.object({
  name: z.string().min(1, "Name is required").max(100),
  target_type: z.enum(["cdc-worker", "trino", "risingwave", "nodes"]),
  target_id: z.string().uuid().optional(),
  min_replicas: z.number().int().min(0).max(100),
  max_replicas: z.number().int().min(1).max(100),
  cooldown_seconds: z.number().int().min(60).max(3600).default(300),
  max_hourly_cost: z.number().min(0).optional(),
  scale_to_zero: z.boolean().default(false),
  enabled: z.boolean().default(true),
  scale_up_rules: z.array(ruleSchema).default([]),
  scale_down_rules: z.array(ruleSchema).default([]),
  schedules: z.array(scheduleSchema).default([]),
}).refine(data => data.max_replicas >= data.min_replicas, {
  message: "Max replicas must be >= min replicas",
  path: ["max_replicas"],
})
```

---

## Test Strategy

1. **Type Safety:** TypeScript compilation
2. **Build Verification:** `npm run build`
3. **Lint:** `npm run lint`
4. **Manual Testing:**
   - Create policy with rules and schedules
   - Edit existing policy
   - Enable/disable policy
   - View scaling history
   - Verify navigation works

---

## Estimated LOC

| Area | Lines |
|------|-------|
| Types | ~150 |
| API Client | ~100 |
| Hooks | ~150 |
| Components | ~800 |
| Pages | ~600 |
| Form/Validation | ~200 |
| **Total** | **~2,000** |

(Lower than 8,000 estimate due to existing backend and component library)

---

## Dependencies

- ✅ DASH-001 (Dashboard Framework) - Complete
- ✅ SCALE-001 (Scaling Engine Backend) - Complete

---

## Out of Scope (Phase 2)

- Cost estimation/calculation logic
- Cost projection charts
- Drag-and-drop rule builder
- Alert integration for scaling limits
- Advanced scheduling calendar view
