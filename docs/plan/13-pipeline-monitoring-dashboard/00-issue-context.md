# Issue Context - DASH-003: Pipeline Monitoring Dashboard

**Issue:** #13
**Branch:** feature/13-pipeline-monitoring-dashboard
**Epic:** dashboard
**Phase:** MVP
**Priority:** Medium
**Milestone:** M2: Management Layer

## Goal

Provide real-time visibility into pipeline health, data flow metrics, and historical performance so users can confidently rely on Philotes for their data needs.

## Problem Statement

"Is my data up to date?" is the most common question for any CDC system. Without monitoring, users are blind to lag, errors, or stalled pipelines until downstream systems complain.

## Target Users

- **Data engineers** monitoring production pipelines
- **Analysts** checking if their reports have fresh data
- **On-call engineers** troubleshooting issues

## Usage Pattern

Dashboard shows live metrics: events per second, replication lag, error rates. Historical charts help identify patterns. Alerts notify when thresholds are breached.

## Acceptance Criteria

- [ ] Pipeline list with status indicators (running/stopped/error)
- [ ] Real-time metrics display (events/sec, lag, errors)
- [ ] Historical charts (line graphs, time range selector)
- [ ] Per-table breakdown of replication status
- [ ] Error log viewer with filtering
- [ ] Pipeline start/stop controls
- [ ] Auto-refresh with configurable interval

## Key Metrics to Display

| Metric | Description |
|--------|-------------|
| Events processed | Total count and rate (events/sec) |
| Replication lag | Current, p95, max |
| Buffer depth | Events waiting to be processed |
| Error count | Grouped by error type |
| Iceberg commits | Commit count and files written |

## Dependencies

- **DASH-001** (completed) - Dashboard framework and core layout
- **OBS-001** (completed) - Metrics and Prometheus integration

## Technical Considerations

- Must integrate with existing Prometheus metrics from OBS-001
- Should use the dashboard framework established in DASH-001
- Real-time updates via polling or WebSocket
- Charts need responsive design for different screen sizes
