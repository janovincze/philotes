# Implementation Plan - Issue #15: Alerting Framework

## Summary

Build a custom alerting framework that evaluates Prometheus metrics, fires alerts when conditions are met, and sends notifications via Slack, email, and webhooks.

## Approach

Build a Philotes-native alerting engine following existing patterns (health check manager, API layered architecture). The alert manager runs an evaluation loop that queries metrics and manages alert state transitions.

## Files to Create

| File | Purpose | LOC |
|------|---------|-----|
| `internal/alerting/types.go` | Alert types, interfaces, constants | ~150 |
| `internal/alerting/manager.go` | Alert manager with evaluation loop | ~300 |
| `internal/alerting/evaluator.go` | Rule evaluation logic | ~200 |
| `internal/alerting/notifier.go` | Notification dispatcher | ~150 |
| `internal/alerting/channels/channel.go` | Channel interface | ~50 |
| `internal/alerting/channels/slack.go` | Slack integration | ~150 |
| `internal/alerting/channels/email.go` | Email integration | ~150 |
| `internal/alerting/channels/webhook.go` | Webhook integration | ~100 |
| `internal/api/handlers/alerts.go` | HTTP handlers | ~400 |
| `internal/api/services/alert_service.go` | Business logic | ~350 |
| `internal/api/repositories/alert_repository.go` | Data access | ~400 |
| `internal/api/models/alert.go` | Request/response models | ~200 |
| `deployments/docker/init-scripts/04-alerting-schema.sql` | Database schema | ~100 |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add AlertingConfig struct |
| `internal/api/server.go` | Register alert routes |
| `go.mod` | Add email dependency (if needed) |

## Task Breakdown

### Phase 1: Foundation (~1,000 LOC)

1. **Create database schema** (`deployments/docker/init-scripts/04-alerting-schema.sql`)
   - alert_rules table
   - alert_instances table
   - alert_history table
   - alert_silences table
   - notification_channels table
   - alert_routes table

2. **Define alerting types** (`internal/alerting/types.go`)
   - AlertRule struct
   - AlertInstance struct
   - AlertSeverity enum (info, warning, critical)
   - AlertStatus enum (firing, resolved)
   - Operator enum (gt, lt, eq, gte, lte)

3. **Add configuration** (`internal/config/config.go`)
   - AlertingConfig struct
   - EvaluationInterval
   - NotificationTimeout

### Phase 2: Data Layer (~600 LOC)

4. **Create API models** (`internal/api/models/alert.go`)
   - CreateAlertRuleRequest/Response
   - UpdateAlertRuleRequest
   - AlertRuleResponse
   - AlertInstanceResponse
   - CreateSilenceRequest
   - CreateChannelRequest

5. **Create repository** (`internal/api/repositories/alert_repository.go`)
   - AlertRepository interface
   - CRUD for alert_rules
   - CRUD for alert_instances
   - CRUD for alert_silences
   - CRUD for notification_channels

### Phase 3: Alert Engine (~650 LOC)

6. **Create evaluator** (`internal/alerting/evaluator.go`)
   - Query Prometheus metrics
   - Compare against thresholds
   - Check duration windows
   - Return evaluation results

7. **Create alert manager** (`internal/alerting/manager.go`)
   - Start/Stop evaluation loop
   - Load rules from database
   - Track pending alerts (for duration)
   - Fire/resolve alerts
   - Call notifier on state changes

### Phase 4: Notifications (~450 LOC)

8. **Create channel interface** (`internal/alerting/channels/channel.go`)
   - Channel interface with Send method
   - ChannelFactory to create channels by type

9. **Implement Slack channel** (`internal/alerting/channels/slack.go`)
   - HTTP webhook integration
   - Message formatting with alert details

10. **Implement webhook channel** (`internal/alerting/channels/webhook.go`)
    - Generic HTTP POST with JSON payload

11. **Implement email channel** (`internal/alerting/channels/email.go`)
    - SMTP integration
    - HTML email template

12. **Create notifier** (`internal/alerting/notifier.go`)
    - Load channels from database
    - Route alerts to channels
    - Track last notification time (repeat interval)

### Phase 5: API Layer (~750 LOC)

13. **Create alert service** (`internal/api/services/alert_service.go`)
    - CreateRule, UpdateRule, DeleteRule, GetRule, ListRules
    - ListAlerts, GetAlert, AcknowledgeAlert
    - CreateSilence, DeleteSilence, ListSilences
    - CreateChannel, UpdateChannel, DeleteChannel, TestChannel

14. **Create alert handlers** (`internal/api/handlers/alerts.go`)
    - Alert rule CRUD handlers
    - Alert instance handlers
    - Silence handlers
    - Notification channel handlers

15. **Register routes** (`internal/api/server.go`)
    - Add alert routes to router
    - Initialize alert manager

### Phase 6: Testing (~500 LOC)

16. **Unit tests**
    - evaluator_test.go
    - manager_test.go
    - alert_service_test.go

17. **Integration tests**
    - API endpoint tests

## Default Alert Rules

Create built-in alert rules during initialization:

1. **Pipeline Stopped** - `cdc_pipeline_state == 4` for 5 minutes (critical)
2. **High Replication Lag** - `cdc_lag_seconds > 300` for 10 minutes (warning)
3. **Very High Replication Lag** - `cdc_lag_seconds > 900` for 5 minutes (critical)
4. **High Error Rate** - `cdc_errors_total rate > 10/min` for 5 minutes (warning)
5. **Buffer Near Capacity** - `buffer_depth > 80%` for 5 minutes (warning)
6. **DLQ Growing** - `buffer_dlq_total increase > 10` in 5 minutes (warning)

## API Schema Examples

### Create Alert Rule
```json
POST /api/v1/alerts/rules
{
  "name": "high-lag-warning",
  "description": "Alert when replication lag exceeds 5 minutes",
  "metric_name": "philotes_cdc_lag_seconds",
  "operator": "gt",
  "threshold": 300,
  "duration_seconds": 600,
  "severity": "warning",
  "labels": {"source": "production"},
  "enabled": true
}
```

### Create Notification Channel
```json
POST /api/v1/notifications/channels
{
  "name": "ops-slack",
  "type": "slack",
  "config": {
    "webhook_url": "https://hooks.slack.com/services/...",
    "channel": "#alerts"
  },
  "enabled": true
}
```

### Create Silence
```json
POST /api/v1/alerts/silences
{
  "matchers": {
    "source": "staging"
  },
  "starts_at": "2024-01-15T00:00:00Z",
  "ends_at": "2024-01-15T12:00:00Z",
  "created_by": "admin",
  "comment": "Maintenance window"
}
```

## Verification Steps

1. `make build` - Builds successfully
2. `make lint` - No linting errors
3. `make test` - All tests pass
4. Manual testing:
   - Create alert rule via API
   - Create notification channel
   - Trigger alert condition
   - Verify notification received
   - Verify alert shows in API
   - Create silence, verify alert suppressed
   - Resolve condition, verify alert resolved

## Estimated LOC

| Category | LOC |
|----------|-----|
| Alerting Engine | ~850 |
| Notification Channels | ~450 |
| API Layer (handlers, services, repos, models) | ~1,350 |
| Database Schema | ~100 |
| Configuration | ~50 |
| Tests | ~500 |
| **Total** | **~3,300** |

## Dependencies

- No new Go dependencies required
- Uses standard library `net/smtp` for email
- Uses standard library `net/http` for Slack/webhook
