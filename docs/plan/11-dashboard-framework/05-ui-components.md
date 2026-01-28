# UI Components Implementation - Issue #11

## Project Structure Created

```
web/
├── src/
│   ├── app/
│   │   ├── layout.tsx           # Root layout with providers
│   │   ├── page.tsx             # Dashboard home
│   │   ├── error.tsx            # Error boundary
│   │   ├── loading.tsx          # Loading state
│   │   ├── not-found.tsx        # 404 page
│   │   ├── sources/page.tsx     # Sources list
│   │   ├── pipelines/page.tsx   # Pipelines list
│   │   ├── alerts/page.tsx      # Alerts placeholder
│   │   └── settings/page.tsx    # Settings placeholder
│   ├── components/
│   │   ├── layout/
│   │   │   ├── sidebar.tsx      # Collapsible sidebar
│   │   │   ├── header.tsx       # Top header
│   │   │   └── main-nav.tsx     # Navigation menu
│   │   ├── ui/                  # shadcn/ui components (14 files)
│   │   ├── theme-provider.tsx   # next-themes wrapper
│   │   └── theme-toggle.tsx     # Dark/light toggle
│   └── lib/
│       ├── api/
│       │   ├── client.ts        # Fetch wrapper
│       │   ├── types.ts         # API types
│       │   ├── sources.ts       # Source endpoints
│       │   ├── pipelines.ts     # Pipeline endpoints
│       │   ├── health.ts        # Health endpoints
│       │   └── index.ts         # Exports
│       ├── hooks/
│       │   ├── use-health.ts    # Health hook
│       │   ├── use-sources.ts   # Source hooks
│       │   └── use-pipelines.ts # Pipeline hooks
│       ├── store/
│       │   └── ui-store.ts      # UI state (sidebar)
│       ├── providers.tsx        # QueryClientProvider
│       └── utils.ts             # cn() utility
├── .env.example
├── .env.local
├── package.json
├── tsconfig.json
├── next.config.ts
└── tailwind.config.ts
```

## Components Created

### Layout Components
- **Sidebar** - Collapsible navigation with icons, persisted state via Zustand
- **Header** - Fixed top bar with mobile menu, theme toggle, user dropdown
- **MainNav** - Navigation items with active state detection

### Page Components
- **Dashboard** - Overview with health status, sources/pipelines counts, recent pipelines
- **Sources** - List view with status badges, connection test buttons
- **Pipelines** - List view with status indicators, start/stop actions
- **Alerts** - Placeholder page
- **Settings** - Placeholder page

### API Integration
- **apiClient** - Base fetch wrapper with error handling, query params support
- **React Query hooks** - useSources, usePipelines, useHealth with cache invalidation
- **Type definitions** - Source, Pipeline, Health, ApiError interfaces

## Dependencies Installed

**Core:**
- next@16.1.6
- react@19.1.0
- typescript@5.x

**UI:**
- tailwindcss@4.x
- shadcn/ui components
- lucide-react (icons)
- next-themes (dark/light mode)

**State & Data:**
- @tanstack/react-query
- zustand (with persist middleware)
- zod, react-hook-form, @hookform/resolvers

**Utilities:**
- class-variance-authority
- clsx, tailwind-merge

## Verification

- [x] `npm run build` - Passes
- [x] `npm run lint` - Passes
- [x] TypeScript compiles without errors
- [x] All routes render correctly
