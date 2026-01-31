# Issue #29: SCALE-004 - Scale-to-Zero Implementation

## Overview

**Goal:** Enable Philotes to scale completely to zero during idle periods, eliminating infrastructure costs when no data is flowing, while quickly resuming when activity returns.

**Problem:** Traditional always-on deployments waste money during nights, weekends, and holidays. For development environments or low-volume pipelines, this can be 70%+ of the time.

**Value:** Running a development Philotes for €5/month instead of €50/month makes it accessible to individual developers and small teams.

## Acceptance Criteria

- [ ] Inactivity detection (no events, no API calls)
- [ ] Configurable idle threshold per pipeline/component
- [ ] Graceful scale-down with checkpoint flush
- [ ] Warmup triggers (CDC events, API requests, schedule)
- [ ] Fast cold-start optimization
- [ ] Keep-alive probes to prevent premature scaling
- [ ] Cost savings reporting

## Scale-to-Zero Flow

```
1. No events for [threshold] minutes
2. Flush checkpoints and close connections
3. Scale worker pods to 0
4. (Optional) Terminate worker nodes
5. On trigger: provision resources, restore from checkpoint
6. Resume CDC from last position
```

## Warmup Triggers

- Source database activity (via lightweight listener)
- Manual API call (wake endpoint)
- Scheduled time (cron)
- Dashboard access

## Dependencies

- SCALE-001: KEDA Integration (completed)
- SCALE-003: Infrastructure Node Auto-scaling (just completed in PR #87)

## Related Files from SCALE-003

The node auto-scaling implementation provides:
- `internal/scaling/nodepool/` - Node pool management
- `internal/scaling/kubernetes/` - K8s client, drain, monitor
- `internal/scaling/cloudprovider/` - Multi-provider support
- `internal/scaling/node_executor.go` - Scaling execution
