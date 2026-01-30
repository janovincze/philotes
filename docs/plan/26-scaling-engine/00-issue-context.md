# Issue Context: SCALE-001 - Scaling Engine and Policy Framework

## Issue Details
- **Number:** #26
- **Title:** SCALE-001: Scaling Engine and Policy Framework
- **Type:** Feature
- **Priority:** High
- **Phase:** MVP
- **Milestone:** M3: Production Ready
- **Epic:** Scaling

## Goal
Build a central scaling engine that evaluates policies and triggers scaling actions across all components (workers, query engines, infrastructure nodes) based on CDC-specific metrics and cost constraints.

## Problem Statement
Raw KEDA scaling is reactive and metric-specific. Organizations need coordinated scaling with business rules: cost limits, schedules, multi-metric decisions, and scale-to-zero with proper warmup.

## Who Benefits
- Cost-conscious teams wanting intelligent resource allocation
- Operations teams managing complex scaling requirements
- Organizations with variable workloads (business hours, batch windows)

## How It's Used
Admins define scaling policies in the dashboard specifying rules, constraints, and schedules. The scaling engine continuously evaluates policies and triggers KEDA/cloud provider APIs to adjust resources.

## Acceptance Criteria
- [ ] Scaling policy data model (thresholds, cooldowns, schedules, cost limits)
- [ ] Policy evaluation engine with configurable intervals
- [ ] Multi-metric decision making (lag AND buffer depth AND CPU)
- [ ] Scaling action executor with provider abstraction
- [ ] KEDA integration for pod-level scaling
- [ ] Cloud provider integration for node-level scaling (Hetzner, OVHcloud, Scaleway)
- [ ] Scaling history and audit log
- [ ] Dry-run mode for policy testing
- [ ] Cost estimation before scaling decisions

## Policy Model (from issue)
```go
type ScalingPolicy struct {
    ID              uuid.UUID
    Name            string
    TargetType      string   // "cdc-worker", "trino", "risingwave", "nodes"
    TargetID        *uuid.UUID // nil for infrastructure

    ScaleUpRules    []ScalingRule
    ScaleDownRules  []ScalingRule

    MinReplicas     int
    MaxReplicas     int
    CooldownSeconds int

    Schedules       []ScalingSchedule

    MaxHourlyCost   *float64
    ScaleToZero     bool
}

type ScalingRule struct {
    Metric          string   // "cdc_lag_seconds", "buffer_depth", "cpu_percent"
    Threshold       float64
    Operator        string   // "gt", "lt", "gte", "lte"
    Duration        string   // "5m"
    ScaleBy         int      // +2 or -1
}

type ScalingSchedule struct {
    CronExpression  string   // "0 8 * * 1-5"
    DesiredReplicas int
    Timezone        string
}
```

## Metrics Sources
- Prometheus (CDC metrics, query engine metrics)
- Cloud provider APIs (node CPU/memory)
- Philotes internal metrics (buffer depth, active pipelines)

## Dependencies
- OBS-001 (metrics must be available) ✅ Completed
- INFRA-001 (Kubernetes Helm Charts) ✅ Completed

## Blocks
- SCALE-002: Dashboard Scaling Configuration UI
- SCALE-003: Infrastructure Node Auto-scaling
- SCALE-004: Scale-to-Zero Implementation
