# Implementation Plan: Issue #11 - Dashboard Framework

## Overview

Create the foundational Next.js 14+ dashboard with TypeScript, Tailwind CSS, shadcn/ui, responsive sidebar, theme support, and API client scaffolding.

## Phase 1: Project Setup

### Task 1.1: Initialize Next.js Project
```bash
cd /Volumes/ExternalSSD/dev/philotes
npx create-next-app@latest web --typescript --tailwind --eslint --app --src-dir --import-alias "@/*"
```

### Task 1.2: Install Dependencies
```bash
cd web
npm install @tanstack/react-query zustand react-hook-form @hookform/resolvers zod
npm install recharts lucide-react class-variance-authority clsx tailwind-merge
npm install next-themes  # For dark/light mode
npm install -D @types/node
```

### Task 1.3: Initialize shadcn/ui
```bash
npx shadcn@latest init
npx shadcn@latest add button card input label select separator sheet skeleton switch tabs toast
```

## Phase 2: Core Layout Components

### Task 2.1: Create Layout Structure

**Files to create:**
- `src/components/layout/sidebar.tsx` - Collapsible navigation sidebar
- `src/components/layout/header.tsx` - Top header with theme toggle
- `src/components/layout/main-nav.tsx` - Navigation menu items
- `src/components/layout/mobile-nav.tsx` - Mobile responsive nav
- `src/components/layout/user-nav.tsx` - User menu (placeholder for auth)

### Task 2.2: Theme Provider Setup

**Files to create:**
- `src/components/theme-provider.tsx` - next-themes provider
- `src/components/theme-toggle.tsx` - Dark/light mode toggle button

### Task 2.3: Root Layout

**Update `src/app/layout.tsx`:**
- Wrap with ThemeProvider
- Add Sidebar and Header
- Configure fonts (Inter)
- Add TanStack Query provider

## Phase 3: Utility Functions & Configuration

### Task 3.1: Utility Functions

**Files to create:**
- `src/lib/utils.ts` - cn() class merge utility
- `src/lib/format.ts` - Date/number formatters
- `src/lib/constants.ts` - App constants, routes

### Task 3.2: Environment Configuration

**Files to create:**
- `src/env.ts` - Environment variable validation with zod
- `.env.example` - Environment template
- `.env.local` - Local development config

## Phase 4: API Client Setup

### Task 4.1: Create API Client

**Files to create:**
- `src/lib/api/client.ts` - Base fetch wrapper with error handling
- `src/lib/api/types.ts` - API response/request types
- `src/lib/api/sources.ts` - Source API functions
- `src/lib/api/pipelines.ts` - Pipeline API functions
- `src/lib/api/health.ts` - Health check API

### Task 4.2: React Query Setup

**Files to create:**
- `src/lib/providers.tsx` - QueryClientProvider setup
- `src/lib/hooks/use-sources.ts` - Source data hooks
- `src/lib/hooks/use-pipelines.ts` - Pipeline data hooks
- `src/lib/hooks/use-health.ts` - Health check hooks

## Phase 5: Error Handling & Loading States

### Task 5.1: Error Components

**Files to create:**
- `src/app/error.tsx` - Root error boundary
- `src/app/not-found.tsx` - 404 page
- `src/components/error/error-card.tsx` - Error display component
- `src/components/error/api-error.tsx` - API error display

### Task 5.2: Loading States

**Files to create:**
- `src/app/loading.tsx` - Root loading state
- `src/components/ui/loading-spinner.tsx` - Spinner component
- `src/components/skeleton/page-skeleton.tsx` - Page skeleton loader

## Phase 6: Dashboard Pages

### Task 6.1: Home Dashboard

**Files to create:**
- `src/app/page.tsx` - Dashboard overview with:
  - System health status card
  - Sources summary card
  - Pipelines summary card
  - Recent activity feed (placeholder)

### Task 6.2: Sources Page Scaffold

**Files to create:**
- `src/app/sources/page.tsx` - Sources list page
- `src/app/sources/loading.tsx` - Loading state
- `src/components/sources/source-card.tsx` - Source display card
- `src/components/sources/sources-list.tsx` - Sources grid/list

### Task 6.3: Pipelines Page Scaffold

**Files to create:**
- `src/app/pipelines/page.tsx` - Pipelines list page
- `src/app/pipelines/loading.tsx` - Loading state
- `src/components/pipelines/pipeline-card.tsx` - Pipeline display card
- `src/components/pipelines/pipeline-status-badge.tsx` - Status indicator

## Phase 7: State Management

### Task 7.1: Zustand Stores

**Files to create:**
- `src/lib/store/ui-store.ts` - UI state (sidebar collapsed, theme)
- `src/lib/store/index.ts` - Store exports

---

## File Summary

| Category | Files | Description |
|----------|-------|-------------|
| Setup | 5 | package.json, tsconfig, next.config, tailwind.config, .env |
| Layout | 6 | Sidebar, header, nav components |
| Theme | 2 | Theme provider and toggle |
| Utils | 4 | cn, format, constants, env |
| API | 6 | Client, types, endpoints, hooks |
| Error | 5 | Error boundaries, components |
| Pages | 6 | Dashboard, sources, pipelines |
| Store | 2 | UI store |

**Total: ~36 files**

---

## Verification

```bash
# Build check
cd web && npm run build

# Lint check
npm run lint

# Type check
npx tsc --noEmit

# Development server
npm run dev
```

---

## Navigation Structure

```
Dashboard (/)
├── Sources (/sources)
│   ├── List all sources
│   └── [future] Create/Edit source
├── Pipelines (/pipelines)
│   ├── List all pipelines
│   └── [future] Create/Edit pipeline
├── Alerts (/alerts) [placeholder]
└── Settings (/settings) [placeholder]
```

---

## Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|----------------|
| Next.js 14+ with App Router | create-next-app with --app flag |
| TypeScript strict mode | tsconfig.json with strict: true |
| Tailwind CSS + shadcn/ui | shadcn init + component installs |
| Dark/light theme | next-themes provider + toggle |
| Responsive layout | Mobile-first Tailwind + Sheet for mobile nav |
| Navigation sidebar | Collapsible sidebar component |
| API client generation | Manual TypeScript client (OpenAPI gen optional) |
| Error boundary + loading | error.tsx + loading.tsx per route |
