# Issue #11 - DASH-001: Dashboard Framework and Core Layout

## Title
Dashboard Framework and Core Layout

## Labels
- epic:dashboard
- phase:mvp
- priority:medium
- type:feature

## Milestone
M2: Management Layer

## Goal
Create a modern, responsive web dashboard that provides visual management of CDC pipelines, making Philotes accessible to users who prefer GUIs over CLIs.

## Problem Statement
Not everyone is comfortable with command-line tools or API calls. A visual dashboard lowers the barrier to entry and provides at-a-glance status of all pipelines.

## Who Benefits
- Non-technical stakeholders who need pipeline visibility
- Data engineers who prefer visual interfaces
- Teams onboarding new members

## Acceptance Criteria
- [ ] Next.js 14+ with App Router
- [ ] TypeScript setup with strict mode
- [ ] Tailwind CSS + shadcn/ui components
- [ ] Dark/light theme support
- [ ] Responsive layout
- [ ] Navigation sidebar
- [ ] API client generation from OpenAPI spec
- [ ] Error boundary and loading states

## Tech Stack
- Next.js 14+ (React framework)
- TypeScript
- Tailwind CSS
- shadcn/ui (component library)
- TanStack Query (data fetching)
- Zustand (state management)

## Dependencies
- API-001 (completed)

## Blocks
- DASH-002 (Setup Wizard)
- DASH-003 (Pipeline Monitoring)

## Estimate
~10,000 LOC
