package deadletter

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// PostgresManager implements Manager using PostgreSQL.
type PostgresManager struct {
	db        *sql.DB
	logger    *slog.Logger
	retention time.Duration
}

// PostgresConfig holds configuration for the PostgreSQL DLQ manager.
type PostgresConfig struct {
	// Retention is how long to keep events in the DLQ.
	Retention time.Duration
}

// DefaultPostgresConfig returns a PostgresConfig with sensible defaults.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Retention: 7 * 24 * time.Hour, // 7 days
	}
}

// NewPostgresManager creates a new PostgreSQL-backed DLQ manager.
func NewPostgresManager(db *sql.DB, cfg PostgresConfig, logger *slog.Logger) *PostgresManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresManager{
		db:        db,
		logger:    logger.With("component", "dlq-manager"),
		retention: cfg.Retention,
	}
}

// Write adds a failed event to the dead-letter queue.
func (m *PostgresManager) Write(ctx context.Context, event FailedEvent) error {
	query := `
		INSERT INTO philotes.dead_letter_events (
			original_event_id, source_id, schema_name, table_name, operation,
			event_data, error_message, error_type, retry_count, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var id int64
	err := m.db.QueryRowContext(ctx, query,
		nullableInt64(event.OriginalEventID),
		event.SourceID,
		event.SchemaName,
		event.TableName,
		event.Operation,
		event.EventData,
		event.ErrorMessage,
		string(event.ErrorType),
		event.RetryCount,
		event.CreatedAt,
		event.ExpiresAt,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert dead letter event: %w", err)
	}

	m.logger.Debug("event added to DLQ",
		"id", id,
		"source_id", event.SourceID,
		"table", fmt.Sprintf("%s.%s", event.SchemaName, event.TableName),
		"error_type", event.ErrorType,
	)

	return nil
}

// Read retrieves failed events from the dead-letter queue.
func (m *PostgresManager) Read(ctx context.Context, limit int) ([]FailedEvent, error) {
	query := `
		SELECT id, original_event_id, source_id, schema_name, table_name,
		       operation, event_data, error_message, error_type, retry_count,
		       created_at, last_retry_at, expires_at
		FROM philotes.dead_letter_events
		ORDER BY created_at ASC
		LIMIT $1
	`

	return m.queryEvents(ctx, query, limit)
}

// ReadBySource retrieves failed events for a specific source.
func (m *PostgresManager) ReadBySource(ctx context.Context, sourceID string, limit int) ([]FailedEvent, error) {
	query := `
		SELECT id, original_event_id, source_id, schema_name, table_name,
		       operation, event_data, error_message, error_type, retry_count,
		       created_at, last_retry_at, expires_at
		FROM philotes.dead_letter_events
		WHERE source_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	return m.queryEvents(ctx, query, sourceID, limit)
}

// ReadByTable retrieves failed events for a specific table.
func (m *PostgresManager) ReadByTable(ctx context.Context, schemaName, tableName string, limit int) ([]FailedEvent, error) {
	query := `
		SELECT id, original_event_id, source_id, schema_name, table_name,
		       operation, event_data, error_message, error_type, retry_count,
		       created_at, last_retry_at, expires_at
		FROM philotes.dead_letter_events
		WHERE schema_name = $1 AND table_name = $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	return m.queryEvents(ctx, query, schemaName, tableName, limit)
}

// queryEvents executes a query and returns the resulting events.
func (m *PostgresManager) queryEvents(ctx context.Context, query string, args ...any) ([]FailedEvent, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query dead letter events: %w", err)
	}
	defer rows.Close()

	var events []FailedEvent
	for rows.Next() {
		var event FailedEvent
		var originalEventID sql.NullInt64
		var errorType sql.NullString
		var lastRetryAt sql.NullTime
		var expiresAt sql.NullTime

		err := rows.Scan(
			&event.ID,
			&originalEventID,
			&event.SourceID,
			&event.SchemaName,
			&event.TableName,
			&event.Operation,
			&event.EventData,
			&event.ErrorMessage,
			&errorType,
			&event.RetryCount,
			&event.CreatedAt,
			&lastRetryAt,
			&expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dead letter event: %w", err)
		}

		if originalEventID.Valid {
			event.OriginalEventID = originalEventID.Int64
		}
		if errorType.Valid {
			event.ErrorType = ErrorType(errorType.String)
		}
		if lastRetryAt.Valid {
			event.LastRetryAt = &lastRetryAt.Time
		}
		if expiresAt.Valid {
			event.ExpiresAt = &expiresAt.Time
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dead letter events: %w", err)
	}

	return events, nil
}

// MarkRetried updates the retry count and last retry time.
func (m *PostgresManager) MarkRetried(ctx context.Context, eventID int64) error {
	query := `
		UPDATE philotes.dead_letter_events
		SET retry_count = retry_count + 1, last_retry_at = $2
		WHERE id = $1
	`

	result, err := m.db.ExecContext(ctx, query, eventID, time.Now())
	if err != nil {
		return fmt.Errorf("mark event retried: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %d", eventID)
	}

	return nil
}

// Delete removes an event from the dead-letter queue.
func (m *PostgresManager) Delete(ctx context.Context, eventID int64) error {
	query := `DELETE FROM philotes.dead_letter_events WHERE id = $1`

	result, err := m.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("delete dead letter event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %d", eventID)
	}

	m.logger.Debug("event deleted from DLQ", "id", eventID)
	return nil
}

// Cleanup removes expired events from the dead-letter queue.
func (m *PostgresManager) Cleanup(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM philotes.dead_letter_events
		WHERE expires_at IS NOT NULL AND expires_at < $1
	`

	result, err := m.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("cleanup expired events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		m.logger.Info("cleaned up expired DLQ events", "count", rowsAffected)
	}

	return rowsAffected, nil
}

// Count returns the number of events in the dead-letter queue.
func (m *PostgresManager) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM philotes.dead_letter_events`

	var count int64
	err := m.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count dead letter events: %w", err)
	}

	return count, nil
}

// Close releases any resources held by the manager.
func (m *PostgresManager) Close() error {
	// The database connection is managed externally, so we don't close it here.
	return nil
}

// GetStats returns statistics about the dead-letter queue.
func (m *PostgresManager) GetStats(ctx context.Context) (Stats, error) {
	stats := Stats{
		BySource:    make(map[string]int64),
		ByErrorType: make(map[ErrorType]int64),
	}

	// Total count
	count, err := m.Count(ctx)
	if err != nil {
		return stats, err
	}
	stats.TotalCount = count

	// By source
	sourceQuery := `
		SELECT source_id, COUNT(*) as count
		FROM philotes.dead_letter_events
		GROUP BY source_id
	`
	sourceRows, err := m.db.QueryContext(ctx, sourceQuery)
	if err != nil {
		return stats, fmt.Errorf("query by source: %w", err)
	}
	defer sourceRows.Close()

	for sourceRows.Next() {
		var sourceID string
		var count int64
		if err := sourceRows.Scan(&sourceID, &count); err != nil {
			return stats, fmt.Errorf("scan source row: %w", err)
		}
		stats.BySource[sourceID] = count
	}

	// By error type
	errorQuery := `
		SELECT error_type, COUNT(*) as count
		FROM philotes.dead_letter_events
		WHERE error_type IS NOT NULL
		GROUP BY error_type
	`
	errorRows, err := m.db.QueryContext(ctx, errorQuery)
	if err != nil {
		return stats, fmt.Errorf("query by error type: %w", err)
	}
	defer errorRows.Close()

	for errorRows.Next() {
		var errorType string
		var count int64
		if err := errorRows.Scan(&errorType, &count); err != nil {
			return stats, fmt.Errorf("scan error type row: %w", err)
		}
		stats.ByErrorType[ErrorType(errorType)] = count
	}

	// Oldest and newest
	timeQuery := `
		SELECT MIN(created_at), MAX(created_at)
		FROM philotes.dead_letter_events
	`
	var oldest, newest sql.NullTime
	if err := m.db.QueryRowContext(ctx, timeQuery).Scan(&oldest, &newest); err != nil {
		return stats, fmt.Errorf("query time range: %w", err)
	}
	if oldest.Valid {
		stats.OldestEvent = &oldest.Time
	}
	if newest.Valid {
		stats.NewestEvent = &newest.Time
	}

	return stats, nil
}

// nullableInt64 converts an int64 to a sql.NullInt64.
// Note: Event IDs are always positive (BIGSERIAL), so 0 indicates no original event.
func nullableInt64(v int64) sql.NullInt64 {
	if v <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

// Ensure PostgresManager implements Manager.
var _ Manager = (*PostgresManager)(nil)
