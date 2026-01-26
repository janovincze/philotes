# Issue Context - OBS-001: Metrics and Prometheus Integration

## Issue Details

- **Issue Number:** #14
- **Title:** OBS-001: Metrics and Prometheus Integration
- **Type:** Feature
- **Priority:** High
- **Epic:** Observability
- **Phase:** MVP
- **Milestone:** M1: Core Pipeline
- **Estimate:** ~6,000 LOC

## Goal

Expose comprehensive metrics via Prometheus so users can monitor Philotes health using standard observability tooling and enable auto-scaling based on pipeline load.

## Problem Statement

Without metrics, operators fly blind. They can't answer:
- "How much data are we processing?"
- "Why is replication slow?"

Prometheus integration enables standard monitoring workflows and Grafana dashboards.

## Who Benefits

- Operations teams with existing Prometheus/Grafana stacks
- Engineers debugging performance issues
- Capacity planners sizing infrastructure

## How It's Used

1. Prometheus scrapes the `/metrics` endpoint
2. Grafana dashboards visualize trends
3. Alertmanager fires alerts when thresholds are breached
4. KEDA uses metrics for auto-scaling decisions

## Acceptance Criteria

- [ ] Prometheus metrics endpoint (`/metrics`)
- [ ] CDC metrics (events/sec, lag, errors)
- [ ] API metrics (requests, latency, errors)
- [ ] Go runtime metrics
- [ ] Custom business metrics
- [ ] Metric labels for multi-pipeline support
- [ ] Grafana dashboard definitions (JSON)

## Key Metrics

```
philotes_cdc_events_total{source, table, operation}
philotes_cdc_lag_seconds{source, table}
philotes_iceberg_commits_total{destination, table}
philotes_buffer_depth{source}
philotes_api_requests_total{endpoint, method, status}
philotes_api_request_duration_seconds{endpoint, method}
```

## Dependencies

- CDC-001 (completed)

## Blocks

- OBS-002 (Alerting Framework)
- INFRA-003 (KEDA uses these metrics for auto-scaling)
