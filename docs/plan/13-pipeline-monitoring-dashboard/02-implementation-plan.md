# Implementation Plan - Issue #13: Pipeline Monitoring Dashboard

## Overview

Implement a real-time pipeline monitoring dashboard with metrics visualization, status indicators, historical charts, and pipeline controls.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend (Next.js)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ MetricCard   │  │ MetricChart  │  │ PipelineMonitoringDash   │  │
│  │ (current)    │  │ (history)    │  │ (main view)              │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│            │               │                    │                   │
│            └───────────────┴────────────────────┘                   │
│                            │                                        │
│                    React Query (5s poll)                            │
└────────────────────────────┼────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Backend API (Go/Gin)                             │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ GET /api/v1/pipelines/:id/metrics                             │  │
│  │ GET /api/v1/pipelines/:id/metrics/history?range=1h            │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                            │                                        │
│                    Prometheus Client                                │
└────────────────────────────┼────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Prometheus                                     │
│  philotes_cdc_*, philotes_buffer_*, philotes_iceberg_*              │
└─────────────────────────────────────────────────────────────────────┘
```

## Task Breakdown

### Phase 1: Backend Metrics Endpoints

#### Task 1.1: Create Metrics Models
**File:** `internal/api/models/metrics.go` (new)

```go
type PipelineMetrics struct {
    PipelineID       uuid.UUID     `json:"pipeline_id"`
    Status           PipelineStatus `json:"status"`
    EventsProcessed  int64         `json:"events_processed"`
    EventsPerSecond  float64       `json:"events_per_second"`
    LagSeconds       float64       `json:"lag_seconds"`
    LagP95Seconds    float64       `json:"lag_p95_seconds"`
    BufferDepth      int64         `json:"buffer_depth"`
    ErrorCount       int64         `json:"error_count"`
    IcebergCommits   int64         `json:"iceberg_commits"`
    IcebergBytes     int64         `json:"iceberg_bytes_written"`
    LastEventAt      *time.Time    `json:"last_event_at,omitempty"`
    Uptime           string        `json:"uptime,omitempty"`
    Tables           []TableMetrics `json:"tables,omitempty"`
}

type TableMetrics struct {
    Schema          string    `json:"schema"`
    Table           string    `json:"table"`
    EventsProcessed int64     `json:"events_processed"`
    LagSeconds      float64   `json:"lag_seconds"`
    LastEventAt     *time.Time `json:"last_event_at,omitempty"`
}

type MetricsHistory struct {
    PipelineID string              `json:"pipeline_id"`
    TimeRange  string              `json:"time_range"`
    DataPoints []MetricsDataPoint  `json:"data_points"`
}

type MetricsDataPoint struct {
    Timestamp       time.Time `json:"timestamp"`
    EventsPerSecond float64   `json:"events_per_second"`
    LagSeconds      float64   `json:"lag_seconds"`
    BufferDepth     int64     `json:"buffer_depth"`
    ErrorCount      int64     `json:"error_count"`
}
```

#### Task 1.2: Create Prometheus Client Service
**File:** `internal/api/services/prometheus.go` (new)

- Connect to Prometheus HTTP API
- Query instant metrics
- Query range metrics for history
- Parse Prometheus response format

#### Task 1.3: Create Metrics Service
**File:** `internal/api/services/metrics.go` (new)

- `GetPipelineMetrics(ctx, pipelineID)` - Current metrics snapshot
- `GetPipelineMetricsHistory(ctx, pipelineID, timeRange)` - Historical data
- `GetTableMetrics(ctx, pipelineID)` - Per-table breakdown

#### Task 1.4: Create Metrics Handler
**File:** `internal/api/handlers/metrics.go` (new)

```go
// GET /api/v1/pipelines/:id/metrics
func (h *MetricsHandler) GetPipelineMetrics(c *gin.Context)

// GET /api/v1/pipelines/:id/metrics/history?range=1h
func (h *MetricsHandler) GetPipelineMetricsHistory(c *gin.Context)
```

#### Task 1.5: Register Routes
**File:** `internal/api/server.go` (modify)

Add metrics routes to the router.

---

### Phase 2: Frontend Types and API Client

#### Task 2.1: Add Metrics Types
**File:** `web/src/lib/api/types.ts` (modify)

```typescript
export interface PipelineMetrics {
  pipeline_id: string
  status: PipelineStatus
  events_processed: number
  events_per_second: number
  lag_seconds: number
  lag_p95_seconds: number
  buffer_depth: number
  error_count: number
  iceberg_commits: number
  iceberg_bytes_written: number
  last_event_at?: string
  uptime?: string
  tables?: TableMetrics[]
}

export interface TableMetrics {
  schema: string
  table: string
  events_processed: number
  lag_seconds: number
  last_event_at?: string
}

export interface MetricsDataPoint {
  timestamp: string
  events_per_second: number
  lag_seconds: number
  buffer_depth: number
  error_count: number
}

export interface MetricsHistory {
  pipeline_id: string
  time_range: string
  data_points: MetricsDataPoint[]
}
```

#### Task 2.2: Add Metrics API Functions
**File:** `web/src/lib/api/metrics.ts` (new)

```typescript
export const metricsApi = {
  getPipelineMetrics: (id: string) => Promise<PipelineMetrics>
  getPipelineMetricsHistory: (id: string, range: string) => Promise<MetricsHistory>
}
```

#### Task 2.3: Add Metrics Hooks
**File:** `web/src/lib/hooks/use-metrics.ts` (new)

```typescript
export function usePipelineMetrics(pipelineId: string, options?: { refetchInterval?: number })
export function usePipelineMetricsHistory(pipelineId: string, timeRange: string)
```

---

### Phase 3: Frontend Components

#### Task 3.1: MetricCard Component
**File:** `web/src/components/metrics/metric-card.tsx` (new)

Single metric display with:
- Current value (large number)
- Label and icon
- Trend indicator (up/down arrow)
- Color coding (normal/warning/critical)

#### Task 3.2: MetricChart Component
**File:** `web/src/components/metrics/metric-chart.tsx` (new)

Line chart wrapper using recharts:
- ResponsiveContainer
- Time-series X-axis
- Tooltip with formatted values
- Multiple series support

#### Task 3.3: TimeRangeSelector Component
**File:** `web/src/components/metrics/time-range-selector.tsx` (new)

Time range picker with presets:
- Last 15 minutes, 1 hour, 6 hours, 24 hours, 7 days
- Custom date range (future enhancement)

#### Task 3.4: TableMetricsTable Component
**File:** `web/src/components/pipelines/table-metrics.tsx` (new)

Per-table breakdown showing:
- Table name
- Events processed
- Current lag
- Last event timestamp

#### Task 3.5: ErrorLogViewer Component
**File:** `web/src/components/pipelines/error-log-viewer.tsx` (new)

Error log display with:
- Error message
- Timestamp
- Error type filter

---

### Phase 4: Pipeline Detail Page

#### Task 4.1: Create Pipeline Detail Page
**File:** `web/src/app/pipelines/[id]/page.tsx` (new)

Main monitoring dashboard:
- Pipeline header (name, status, controls)
- Metrics grid (4 key metrics)
- Historical chart with time range selector
- Table metrics breakdown
- Error log viewer

#### Task 4.2: Pipeline Metrics Grid
**File:** `web/src/components/pipelines/pipeline-metrics-grid.tsx` (new)

Grid layout for key metrics:
- Events/second
- Replication lag
- Buffer depth
- Error count

#### Task 4.3: Auto-Refresh Controls
**File:** `web/src/components/pipelines/auto-refresh-toggle.tsx` (new)

Toggle for auto-refresh with interval selector:
- Off, 5s, 15s, 30s, 60s

---

### Phase 5: Integration and Polish

#### Task 5.1: Update Pipelines List
**File:** `web/src/app/pipelines/page.tsx` (modify)

- Add real-time status polling
- Show mini metrics (events/sec) on cards
- Link to detail page

#### Task 5.2: Loading States
Add skeleton loaders for:
- Metric cards
- Charts
- Table metrics

#### Task 5.3: Error Handling
- Handle API errors gracefully
- Show retry buttons
- Toast notifications for actions

---

## File Summary

### New Backend Files
| File | Description |
|------|-------------|
| `internal/api/models/metrics.go` | Metrics response types |
| `internal/api/services/prometheus.go` | Prometheus client |
| `internal/api/services/metrics.go` | Metrics business logic |
| `internal/api/handlers/metrics.go` | HTTP handlers |

### Modified Backend Files
| File | Change |
|------|--------|
| `internal/api/server.go` | Register metrics routes |

### New Frontend Files
| File | Description |
|------|-------------|
| `web/src/lib/api/metrics.ts` | Metrics API client |
| `web/src/lib/hooks/use-metrics.ts` | React Query hooks |
| `web/src/components/metrics/metric-card.tsx` | Single metric display |
| `web/src/components/metrics/metric-chart.tsx` | Line chart wrapper |
| `web/src/components/metrics/time-range-selector.tsx` | Time range picker |
| `web/src/components/pipelines/table-metrics.tsx` | Per-table breakdown |
| `web/src/components/pipelines/error-log-viewer.tsx` | Error log viewer |
| `web/src/components/pipelines/pipeline-metrics-grid.tsx` | Metrics grid |
| `web/src/components/pipelines/auto-refresh-toggle.tsx` | Refresh controls |
| `web/src/app/pipelines/[id]/page.tsx` | Pipeline detail page |

### Modified Frontend Files
| File | Change |
|------|--------|
| `web/src/lib/api/types.ts` | Add metrics types |
| `web/src/app/pipelines/page.tsx` | Add polling, mini metrics |

---

## Test Strategy

### Backend Tests
- Unit tests for metrics service
- Mock Prometheus responses
- Integration tests for API endpoints

### Frontend Tests
- Component tests for MetricCard, MetricChart
- Hook tests for usePipelineMetrics
- Page integration tests

---

## Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|----------------|
| Pipeline list with status indicators | Existing + polling |
| Real-time metrics display | MetricCard + 5s polling |
| Historical charts | MetricChart + time range |
| Per-table breakdown | TableMetricsTable |
| Error log viewer | ErrorLogViewer |
| Pipeline start/stop controls | Existing |
| Auto-refresh with configurable interval | AutoRefreshToggle |

---

## Dependencies

- DASH-001 (completed) - Dashboard framework
- OBS-001 (completed) - Prometheus metrics
- Prometheus running in docker-compose

---

## Estimated LOC

| Area | Lines |
|------|-------|
| Backend models | ~100 |
| Backend services | ~400 |
| Backend handlers | ~150 |
| Frontend types | ~100 |
| Frontend API/hooks | ~200 |
| Frontend components | ~800 |
| Frontend pages | ~400 |
| Tests | ~500 |
| **Total** | **~2,650** |

(Lower than 8,000 estimate due to reuse of existing patterns)
