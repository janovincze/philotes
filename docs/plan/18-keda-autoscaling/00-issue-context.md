# Issue #18 - KEDA Autoscaling Configuration

## Summary

**Title:** INFRA-003: KEDA Autoscaling Configuration
**Type:** Infrastructure
**Priority:** Medium
**Phase:** v1
**Milestone:** M3: Production Ready

## Goal

Enable automatic scaling of Philotes workers based on pipeline load using KEDA (Kubernetes Event-Driven Autoscaling), ensuring resources match demand without manual intervention.

## Problem Statement

Fixed replica counts waste resources during quiet periods and bottleneck during peak loads. KEDA scales based on actual metrics (lag, queue depth) rather than just CPU/memory.

## Who Benefits

- Cost-conscious teams wanting to scale to zero overnight
- High-volume users needing burst capacity
- Operations teams tired of manual scaling

## Acceptance Criteria

- [ ] KEDA ScaledObject definitions for CDC workers
- [ ] Prometheus scaler configuration
- [ ] Custom metrics adapter if needed
- [ ] Scaling triggers (lag, buffer depth, events/sec)
- [ ] Cooldown and stabilization periods
- [ ] Min/max replica bounds
- [ ] Scale-to-zero support with activation threshold

## Scaling Triggers Example

```yaml
triggers:
  - type: prometheus
    metadata:
      serverAddress: http://prometheus:9090
      metricName: philotes_cdc_lag_seconds
      threshold: '300'  # 5 minutes lag
      query: max(philotes_cdc_lag_seconds{pipeline="..."})
```

## Dependencies

- OBS-001 (metrics must be exposed) - Prometheus metrics already implemented (#14)
- INFRA-001 (Helm charts) - Completed (#16)

## Blocks

- SCALE-001 (policy layer uses KEDA)

## Estimate

~3,000 LOC
