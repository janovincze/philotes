-- Dead-letter queue schema for failed CDC events

CREATE TABLE IF NOT EXISTS philotes.dead_letter_events (
    id BIGSERIAL PRIMARY KEY,
    original_event_id BIGINT,
    source_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    operation TEXT NOT NULL,
    event_data JSONB NOT NULL,
    error_message TEXT NOT NULL,
    error_type TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_retry_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

-- Index for querying unprocessed events
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_source_id ON philotes.dead_letter_events(source_id);
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_created_at ON philotes.dead_letter_events(created_at);
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_expires_at ON philotes.dead_letter_events(expires_at) WHERE expires_at IS NOT NULL;

-- Index for table-specific queries
CREATE INDEX IF NOT EXISTS idx_dead_letter_events_table ON philotes.dead_letter_events(schema_name, table_name);

COMMENT ON TABLE philotes.dead_letter_events IS 'Stores CDC events that failed after maximum retry attempts';
COMMENT ON COLUMN philotes.dead_letter_events.original_event_id IS 'Reference to the original event ID in cdc_events table';
COMMENT ON COLUMN philotes.dead_letter_events.error_type IS 'Type of error (transient, permanent, validation, etc.)';
COMMENT ON COLUMN philotes.dead_letter_events.retry_count IS 'Number of times this event has been retried from DLQ';
COMMENT ON COLUMN philotes.dead_letter_events.expires_at IS 'When this event should be deleted (based on retention)';
