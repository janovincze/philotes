# Research Findings - Issue #13: Pipeline Monitoring Dashboard

## 1. Dashboard Framework Analysis (DASH-001)

**Location:** `/web/` (Next.js 14 with App Router)

### Existing Infrastructure
- **Layout components:** Sidebar, Header, MainNav, MainContent
- **Styling:** Tailwind CSS 4.x + shadcn/ui (14 components installed)
- **Theme:** next-themes for dark/light mode
- **Icons:** lucide-react

### State Management
- **React Query** v5.90.20 - Data fetching with automatic caching
- **Zustand** v5.0.10 - UI state (sidebar collapse)
- Query client: 60s stale time, no window focus refetch

### Existing Pages
- `/pipelines/page.tsx` - Pipeline list with status badges, start/stop controls
- `/sources/page.tsx` - Source management
- `/alerts/page.tsx` - Placeholder
- `/settings/page.tsx` - Placeholder

### Component Patterns
- Card-based layout with shadcn/ui Card components
- `PipelineStatusBadge` component with color-coded status
- Skeleton loaders for loading states
- Error boundaries implemented

---

## 2. Backend API Analysis

### Existing Pipeline Endpoints
```
GET    /api/v1/pipelines              - List all pipelines
GET    /api/v1/pipelines/:id          - Get single pipeline
GET    /api/v1/pipelines/:id/status   - Get pipeline status
POST   /api/v1/pipelines/:id/start    - Start pipeline
POST   /api/v1/pipelines/:id/stop     - Stop pipeline
GET    /api/v1/pipelines/:id/tables   - Get table mappings
```

### Pipeline Model (`internal/api/models/pipeline.go`)
```go
type Pipeline struct {
    ID, Name, SourceID, Status, Config, ErrorMessage
    Tables []TableMapping
    CreatedAt, UpdatedAt, StartedAt, StoppedAt
}

type PipelineStatusResponse struct {
    ID, Name, Status, ErrorMessage
    EventsProcessed int64
    LastEventAt, StartedAt *time.Time
    Uptime string
}
```

### Current GetStatus Implementation
- Returns basic status info
- Has TODO for CDC metrics integration
- No actual metrics data yet

---

## 3. Metrics Infrastructure (OBS-001)

**Location:** `/internal/metrics/metrics.go`

### Available Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `philotes_cdc_events_total` | Counter | source, table, operation |
| `philotes_cdc_lag_seconds` | Gauge | source, table |
| `philotes_cdc_errors_total` | Counter | source, error_type |
| `philotes_cdc_retries_total` | Counter | source |
| `philotes_cdc_pipeline_state` | Gauge | source |
| `philotes_buffer_depth` | Gauge | source |
| `philotes_buffer_events_processed_total` | Counter | source |
| `philotes_buffer_dlq_total` | Counter | source |
| `philotes_iceberg_commits_total` | Counter | source, table |
| `philotes_iceberg_files_written_total` | Counter | source, table |
| `philotes_iceberg_bytes_written_total` | Counter | source, table |

### Access Points
- `/metrics` endpoint on API server
- Prometheus: `http://localhost:9090` (via docker-compose)

---

## 4. Frontend API Client

### Existing Hooks (`web/src/lib/hooks/use-pipelines.ts`)
- `usePipelines()` - Lists all pipelines
- `usePipeline(id)` - Gets single pipeline
- `useStartPipeline()` - Mutation for starting
- `useStopPipeline()` - Mutation for stopping

### Types (`web/src/lib/api/types.ts`)
- `Pipeline`, `PipelineStatus`, `TableMapping` defined
- No metrics types yet

---

## 5. Chart Library

**recharts v3.7.0** already in `package.json`:
- LineChart, AreaChart, BarChart available
- ResponsiveContainer for responsive design
- Perfect for time-series metrics

---

## 6. Gaps Identified

### Backend Gaps
1. **No metrics endpoint** - Need `/api/v1/pipelines/:id/metrics`
2. **No historical data** - Need Prometheus range query integration
3. **Status endpoint incomplete** - Doesn't return actual CDC metrics

### Frontend Gaps
1. **No pipeline detail page** - `/pipelines/[id]/page.tsx` doesn't exist
2. **No metrics components** - Need chart wrappers, metric cards
3. **No real-time polling** - Need refetchInterval on hooks
4. **No metrics types** - Need TypeScript interfaces

---

## 7. Implementation Approach

### Strategy: Backend Proxy for Metrics
- Add metrics endpoint in Go API
- API queries Prometheus and returns aggregated data
- Better security, performance, and abstraction than direct frontend queries

### Real-Time Updates
- Use React Query's `refetchInterval: 5000` (5 seconds)
- Simple, no WebSocket infrastructure needed

### Phased Implementation
1. **Phase 1:** Backend metrics endpoint + basic metrics display
2. **Phase 2:** Historical charts with time range selector
3. **Phase 3:** Per-table breakdown + error log viewer
