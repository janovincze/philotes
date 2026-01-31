package buffer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver

	"github.com/janovincze/philotes/internal/cdc"
)

// PostgresManager implements buffer persistence using PostgreSQL.
type PostgresManager struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresManager creates a new PostgreSQL buffer manager.
func NewPostgresManager(ctx context.Context, cfg Config, logger *slog.Logger) (*PostgresManager, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &PostgresManager{
		db:     db,
		logger: logger.With("component", "buffer-manager"),
	}, nil
}

// Write stores events in the buffer.
func (m *PostgresManager) Write(ctx context.Context, events []cdc.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO philotes.cdc_events (
			source_id, schema_name, table_name, operation, lsn,
			transaction_id, key_columns, before_data, after_data,
			event_time, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		keyColumnsJSON, err := jsonMarshalNullable(event.KeyColumns)
		if err != nil {
			return fmt.Errorf("marshal key columns: %w", err)
		}

		beforeDataJSON, err := jsonMarshalNullable(event.Before)
		if err != nil {
			return fmt.Errorf("marshal before data: %w", err)
		}

		afterDataJSON, err := jsonMarshalNullable(event.After)
		if err != nil {
			return fmt.Errorf("marshal after data: %w", err)
		}

		metadataJSON, err := jsonMarshalNullable(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			event.ID,            // source_id (using event ID as source identifier)
			event.Schema,        // schema_name
			event.Table,         // table_name
			event.Operation,     // operation
			event.LSN,           // lsn
			event.TransactionID, // transaction_id
			keyColumnsJSON,      // key_columns
			beforeDataJSON,      // before_data
			afterDataJSON,       // after_data
			event.Timestamp,     // event_time
			metadataJSON,        // metadata
		)
		if err != nil {
			return fmt.Errorf("insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	m.logger.Debug("events written to buffer", "count", len(events))
	return nil
}

// ReadBatch retrieves a batch of unprocessed events for a source.
func (m *PostgresManager) ReadBatch(ctx context.Context, sourceID string, limit int) ([]BufferedEvent, error) {
	query := `
		SELECT id, source_id, schema_name, table_name, operation, lsn,
			   transaction_id, key_columns, before_data, after_data,
			   event_time, metadata, created_at, processed_at
		FROM philotes.cdc_events
		WHERE processed_at IS NULL AND source_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := m.db.QueryContext(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []BufferedEvent
	for rows.Next() {
		var be BufferedEvent
		var transactionID sql.NullInt64
		var keyColumnsJSON, beforeDataJSON, afterDataJSON, metadataJSON []byte

		err := rows.Scan(
			&be.ID,
			&be.Event.ID,
			&be.Event.Schema,
			&be.Event.Table,
			&be.Event.Operation,
			&be.Event.LSN,
			&transactionID,
			&keyColumnsJSON,
			&beforeDataJSON,
			&afterDataJSON,
			&be.Event.Timestamp,
			&metadataJSON,
			&be.CreatedAt,
			&be.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		if transactionID.Valid {
			be.Event.TransactionID = transactionID.Int64
		}

		if err := jsonUnmarshalNullable(keyColumnsJSON, &be.Event.KeyColumns); err != nil {
			m.logger.Warn("failed to unmarshal key columns", "error", err)
		}
		if err := jsonUnmarshalNullable(beforeDataJSON, &be.Event.Before); err != nil {
			m.logger.Warn("failed to unmarshal before data", "error", err)
		}
		if err := jsonUnmarshalNullable(afterDataJSON, &be.Event.After); err != nil {
			m.logger.Warn("failed to unmarshal after data", "error", err)
		}
		if err := jsonUnmarshalNullable(metadataJSON, &be.Event.Metadata); err != nil {
			m.logger.Warn("failed to unmarshal metadata", "error", err)
		}

		events = append(events, be)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

// MarkProcessed marks events as processed by their IDs.
func (m *PostgresManager) MarkProcessed(ctx context.Context, eventIDs []int64) error {
	if len(eventIDs) == 0 {
		return nil
	}

	// Build the query with placeholders
	query := `UPDATE philotes.cdc_events SET processed_at = NOW() WHERE id = ANY($1)`

	result, err := m.db.ExecContext(ctx, query, eventIDs)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	m.logger.Debug("events marked as processed", "count", rowsAffected)

	return nil
}

// Cleanup removes old processed events based on retention policy.
func (m *PostgresManager) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)

	query := `DELETE FROM philotes.cdc_events WHERE processed_at IS NOT NULL AND processed_at < $1`

	result, err := m.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup events: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		m.logger.Info("cleaned up old events", "deleted", rowsDeleted, "retention", retention)
	}

	return rowsDeleted, nil
}

// Stats returns buffer statistics.
func (m *PostgresManager) Stats(ctx context.Context) (Stats, error) {
	var stats Stats

	// Get total and unprocessed counts
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE processed_at IS NULL) as unprocessed,
			MIN(created_at) FILTER (WHERE processed_at IS NULL) as oldest_unprocessed
		FROM philotes.cdc_events
	`

	var oldestUnprocessed sql.NullTime
	err := m.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalEvents,
		&stats.UnprocessedEvents,
		&oldestUnprocessed,
	)
	if err != nil {
		return Stats{}, fmt.Errorf("query stats: %w", err)
	}

	if oldestUnprocessed.Valid {
		stats.OldestUnprocessed = &oldestUnprocessed.Time
		stats.Lag = time.Since(oldestUnprocessed.Time)
	}

	return stats, nil
}

// Close closes the database connection.
func (m *PostgresManager) Close() error {
	return m.db.Close()
}

// jsonMarshalNullable marshals a value to JSON, returning nil for nil/empty values.
func jsonMarshalNullable(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}

	// Check for empty slices/maps
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return nil, nil
		}
	case map[string]any:
		if len(val) == 0 {
			return nil, nil
		}
	}

	return json.Marshal(v)
}

// jsonUnmarshalNullable unmarshals JSON, handling nil gracefully.
func jsonUnmarshalNullable(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// Ensure PostgresManager implements Manager interface.
var _ Manager = (*PostgresManager)(nil)
