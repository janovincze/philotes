# Issue Context - SCALE-002: Dashboard Scaling Configuration UI

## Issue Details
- **Number:** #27
- **Title:** SCALE-002: Dashboard Scaling Configuration UI
- **Labels:** epic:scaling, phase:mvp, priority:high, type:feature
- **Milestone:** M3: Production Ready
- **Estimate:** ~8,000 LOC

## Goal
Provide a visual interface in the dashboard for configuring scaling policies, viewing scaling history, and understanding current resource allocation without writing YAML or code.

## Problem Statement
Scaling configuration is complex. Non-expert users struggle with KEDA manifests and cost calculations. A UI abstracts complexity and provides visual feedback on scaling behavior.

## Target Users
- Platform teams configuring scaling for their organization
- Cost-conscious managers reviewing resource usage
- Engineers debugging scaling behavior

## Value Proposition
A scaling UI demonstrates sophistication. Competitors require manual YAML editing. Philotes makes intelligent scaling accessible to non-experts.

## Acceptance Criteria
- [ ] Scaling policy create/edit forms
- [ ] Visual policy builder (drag-and-drop rules)
- [ ] Current scale visualization (pods, nodes)
- [ ] Scaling event timeline
- [ ] Cost tracking dashboard (hourly/daily/monthly)
- [ ] Policy simulation/preview
- [ ] Alerts for scaling limits reached
- [ ] Per-pipeline scaling configuration

## UI Components Required
- Policy editor with metric selector
- Schedule calendar view
- Cost projection charts
- Scaling event log with drill-down
- Resource utilization gauges

## Dependencies
- DASH-001 (Dashboard Framework) - ✅ Completed
- SCALE-001 (Scaling Engine) - Need to check status

## Technical Context
- Dashboard: Next.js 14+, TypeScript, Tailwind, shadcn/ui
- Backend: Go/Gin API
- Scaling: KEDA-based auto-scaling (planned)
- Metrics: Prometheus for scaling metrics
