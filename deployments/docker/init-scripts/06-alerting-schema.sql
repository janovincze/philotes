-- Alerting Framework Schema for Philotes
-- This script creates the tables required for the alerting framework

-- Alert rules table stores alert rule definitions
CREATE TABLE IF NOT EXISTS philotes.alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    metric_name TEXT NOT NULL,
    operator TEXT NOT NULL CHECK (operator IN ('gt', 'lt', 'eq', 'gte', 'lte')),
    threshold DOUBLE PRECISION NOT NULL,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'critical')),
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for alert_rules
CREATE INDEX IF NOT EXISTS idx_alert_rules_name ON philotes.alert_rules(name);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON philotes.alert_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_alert_rules_severity ON philotes.alert_rules(severity);
CREATE INDEX IF NOT EXISTS idx_alert_rules_metric_name ON philotes.alert_rules(metric_name);

-- Alert instances table stores active/resolved alert instances
CREATE TABLE IF NOT EXISTS philotes.alert_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES philotes.alert_rules(id) ON DELETE CASCADE,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'firing' CHECK (status IN ('firing', 'resolved')),
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    current_value DOUBLE PRECISION,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(rule_id, fingerprint)
);

-- Indexes for alert_instances
CREATE INDEX IF NOT EXISTS idx_alert_instances_rule_id ON philotes.alert_instances(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_instances_status ON philotes.alert_instances(status);
CREATE INDEX IF NOT EXISTS idx_alert_instances_fingerprint ON philotes.alert_instances(fingerprint);
CREATE INDEX IF NOT EXISTS idx_alert_instances_fired_at ON philotes.alert_instances(fired_at);

-- Alert history table stores audit trail for alerts
CREATE TABLE IF NOT EXISTS philotes.alert_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES philotes.alert_instances(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES philotes.alert_rules(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN ('fired', 'resolved', 'acknowledged', 'notification_sent', 'notification_failed')),
    message TEXT NOT NULL DEFAULT '',
    value DOUBLE PRECISION,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for alert_history
CREATE INDEX IF NOT EXISTS idx_alert_history_alert_id ON philotes.alert_history(alert_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_rule_id ON philotes.alert_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_event_type ON philotes.alert_history(event_type);
CREATE INDEX IF NOT EXISTS idx_alert_history_created_at ON philotes.alert_history(created_at);

-- Alert silences table stores temporary suppression rules
CREATE TABLE IF NOT EXISTS philotes.alert_silences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matchers JSONB NOT NULL DEFAULT '{}',
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

-- Indexes for alert_silences
CREATE INDEX IF NOT EXISTS idx_alert_silences_starts_at ON philotes.alert_silences(starts_at);
CREATE INDEX IF NOT EXISTS idx_alert_silences_ends_at ON philotes.alert_silences(ends_at);
CREATE INDEX IF NOT EXISTS idx_alert_silences_active ON philotes.alert_silences(starts_at, ends_at);

-- Notification channels table stores notification channel configurations
CREATE TABLE IF NOT EXISTS philotes.notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK (type IN ('slack', 'email', 'webhook', 'pagerduty')),
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for notification_channels
CREATE INDEX IF NOT EXISTS idx_notification_channels_name ON philotes.notification_channels(name);
CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON philotes.notification_channels(type);
CREATE INDEX IF NOT EXISTS idx_notification_channels_enabled ON philotes.notification_channels(enabled);

-- Alert routes table links alert rules to notification channels
CREATE TABLE IF NOT EXISTS philotes.alert_routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES philotes.alert_rules(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES philotes.notification_channels(id) ON DELETE CASCADE,
    repeat_interval_seconds INTEGER NOT NULL DEFAULT 3600,
    group_wait_seconds INTEGER NOT NULL DEFAULT 30,
    group_interval_seconds INTEGER NOT NULL DEFAULT 300,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(rule_id, channel_id)
);

-- Indexes for alert_routes
CREATE INDEX IF NOT EXISTS idx_alert_routes_rule_id ON philotes.alert_routes(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_routes_channel_id ON philotes.alert_routes(channel_id);
CREATE INDEX IF NOT EXISTS idx_alert_routes_enabled ON philotes.alert_routes(enabled);

-- Add comments for documentation
COMMENT ON TABLE philotes.alert_rules IS 'Alert rule definitions for metric-based alerting';
COMMENT ON TABLE philotes.alert_instances IS 'Active and resolved alert instances';
COMMENT ON TABLE philotes.alert_history IS 'Audit trail for alert state changes';
COMMENT ON TABLE philotes.alert_silences IS 'Temporary alert suppression rules';
COMMENT ON TABLE philotes.notification_channels IS 'Notification channel configurations (Slack, email, webhook)';
COMMENT ON TABLE philotes.alert_routes IS 'Routing rules linking alert rules to notification channels';

-- Grant permissions
GRANT ALL ON TABLE philotes.alert_rules TO philotes;
GRANT ALL ON TABLE philotes.alert_instances TO philotes;
GRANT ALL ON TABLE philotes.alert_history TO philotes;
GRANT ALL ON TABLE philotes.alert_silences TO philotes;
GRANT ALL ON TABLE philotes.notification_channels TO philotes;
GRANT ALL ON TABLE philotes.alert_routes TO philotes;
