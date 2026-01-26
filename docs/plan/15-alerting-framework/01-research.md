# Research - Issue #15: Alerting Framework

## Existing Patterns Analysis

### Health Check System (`internal/cdc/health/health.go`)
- Manager pattern with thread-safe registration
- Extensible HealthChecker interface
- Component-based architecture with separate HTTP server
- Perfect precedent for alerting manager

### Metrics System (`internal/metrics/metrics.go`)
- Prometheus metric definitions with namespace organization
- Label constants for consistency (source, table, operation, etc.)
- Registry pattern with sync.Once
- Thread-safe metric recording
- Metrics already tracking: lag, errors, pipeline state, buffer depth

### API Layered Architecture
- Handlers (HTTP request/response)
- Services (business logic, validation, error handling)
- Repositories (data access)
- Models (request/response types)
- Custom error types: ValidationError, NotFoundError, ConflictError

## Available Metrics for Alert Rules

From `internal/metrics/metrics.go`:
- `philotes_cdc_lag_seconds` - replication lag (perfect for alert condition)
- `philotes_cdc_errors_total` - error count
- `philotes_cdc_pipeline_state` - pipeline status (0=stopped, 1=starting, 2=running, 3=paused, 4=failed)
- `philotes_buffer_depth` - unprocessed events count
- `philotes_buffer_dlq_total` - dead-letter queue growth

## Architecture Decision: Custom Alerting Engine

**Recommended: Build a Philotes-native alerting system**

Advantages:
- Consistent with existing architecture patterns
- Built-in database persistence
- Tighter integration with metrics
- Simpler operational model (no additional components)
- Full control over alert lifecycle

Alternative (Alertmanager integration) rejected for MVP because:
- Adds operational complexity
- Requires managing two systems
- Alert UI would need to query both systems

## Database Schema Design

```sql
-- Alert rules definition
CREATE TABLE IF NOT EXISTS philotes.alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    metric_name TEXT NOT NULL,
    operator TEXT NOT NULL,       -- 'gt', 'lt', 'eq', 'gte', 'lte'
    threshold FLOAT NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 300,
    severity TEXT NOT NULL DEFAULT 'warning',
    labels JSONB,                 -- label filters: {"source": "prod_db"}
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Active alert instances
CREATE TABLE IF NOT EXISTS philotes.alert_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES philotes.alert_rules(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,    -- hash of rule_id + labels for deduplication
    status TEXT NOT NULL DEFAULT 'firing',
    labels JSONB,
    current_value FLOAT,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    last_evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(fingerprint)
);

-- Alert history for audit trail
CREATE TABLE IF NOT EXISTS philotes.alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    event_type TEXT NOT NULL,     -- 'fired', 'resolved', 'acknowledged', 'silenced'
    message TEXT,
    value FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Alert silences
CREATE TABLE IF NOT EXISTS philotes.alert_silences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matchers JSONB NOT NULL,      -- label matchers
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification channels
CREATE TABLE IF NOT EXISTS philotes.notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,           -- 'slack', 'email', 'webhook'
    config JSONB NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Rule to channel routing
CREATE TABLE IF NOT EXISTS philotes.alert_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES philotes.alert_rules(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES philotes.notification_channels(id) ON DELETE CASCADE,
    repeat_interval_seconds INT DEFAULT 3600,
    UNIQUE(rule_id, channel_id)
);
```

## API Endpoints

### Alert Rules
- `POST   /api/v1/alerts/rules` - Create alert rule
- `GET    /api/v1/alerts/rules` - List rules
- `GET    /api/v1/alerts/rules/:id` - Get rule
- `PUT    /api/v1/alerts/rules/:id` - Update rule
- `DELETE /api/v1/alerts/rules/:id` - Delete rule

### Alert Instances
- `GET    /api/v1/alerts` - List active alerts
- `GET    /api/v1/alerts/:id` - Get alert details
- `POST   /api/v1/alerts/:id/acknowledge` - Acknowledge alert
- `GET    /api/v1/alerts/history` - Alert event history

### Silences
- `POST   /api/v1/alerts/silences` - Create silence
- `GET    /api/v1/alerts/silences` - List silences
- `DELETE /api/v1/alerts/silences/:id` - Delete silence

### Notification Channels
- `POST   /api/v1/notifications/channels` - Create channel
- `GET    /api/v1/notifications/channels` - List channels
- `GET    /api/v1/notifications/channels/:id` - Get channel
- `PUT    /api/v1/notifications/channels/:id` - Update channel
- `DELETE /api/v1/notifications/channels/:id` - Delete channel
- `POST   /api/v1/notifications/channels/:id/test` - Test channel

## Component Architecture

```
internal/alerting/
├── manager.go           # Alert manager (evaluation loop, state)
├── evaluator.go         # Rule evaluation logic
├── notifier.go          # Notification dispatcher
├── types.go             # Alert types and interfaces
├── channels/
│   ├── channel.go       # Channel interface
│   ├── slack.go         # Slack integration
│   ├── email.go         # Email integration
│   └── webhook.go       # Webhook integration
└── manager_test.go

internal/api/
├── handlers/
│   └── alerts.go        # HTTP handlers
├── services/
│   └── alert_service.go # Business logic
├── repositories/
│   └── alert_repository.go
└── models/
    └── alert.go         # Request/response models
```

## Key Files Referenced

- `internal/metrics/metrics.go` - Metric definitions to query
- `internal/cdc/health/health.go` - Manager pattern to follow
- `internal/api/server.go` - Server structure for integration
- `internal/api/services/pipeline.go` - Service pattern
- `internal/api/repositories/pipeline.go` - Repository pattern
- `internal/config/config.go` - Configuration pattern
