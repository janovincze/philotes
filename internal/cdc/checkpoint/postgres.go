package checkpoint

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver

	"github.com/janovincze/philotes/internal/cdc"
)

// PostgresManager implements checkpoint persistence using PostgreSQL.
type PostgresManager struct {
	db     *sql.DB
	logger *slog.Logger
}

// PostgresConfig holds configuration for the PostgreSQL checkpoint manager.
type PostgresConfig struct {
	// DSN is the PostgreSQL connection string.
	DSN string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
}

// NewPostgresManager creates a new PostgreSQL checkpoint manager.
func NewPostgresManager(ctx context.Context, cfg PostgresConfig, logger *slog.Logger) (*PostgresManager, error) {
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
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
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
		logger: logger.With("component", "checkpoint-manager"),
	}, nil
}

// Save persists a checkpoint to the database.
func (m *PostgresManager) Save(ctx context.Context, checkpoint cdc.Checkpoint) error {
	var metadataJSON []byte
	var err error
	if checkpoint.Metadata != nil {
		metadataJSON, err = json.Marshal(checkpoint.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO philotes.cdc_checkpoints (source_id, lsn, transaction_id, committed_at, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source_id)
		DO UPDATE SET
			lsn = EXCLUDED.lsn,
			transaction_id = EXCLUDED.transaction_id,
			committed_at = EXCLUDED.committed_at,
			metadata = EXCLUDED.metadata
	`

	committedAt := checkpoint.CommittedAt
	if committedAt.IsZero() {
		committedAt = time.Now()
	}

	_, err = m.db.ExecContext(ctx, query,
		checkpoint.SourceID,
		checkpoint.LSN,
		checkpoint.TransactionID,
		committedAt,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}

	m.logger.Debug("checkpoint saved",
		"source_id", checkpoint.SourceID,
		"lsn", checkpoint.LSN,
	)

	return nil
}

// Load retrieves the latest checkpoint for a source.
func (m *PostgresManager) Load(ctx context.Context, sourceID string) (*cdc.Checkpoint, error) {
	query := `
		SELECT source_id, lsn, transaction_id, committed_at, metadata
		FROM philotes.cdc_checkpoints
		WHERE source_id = $1
	`

	var checkpoint cdc.Checkpoint
	var transactionID sql.NullInt64
	var metadataJSON []byte

	err := m.db.QueryRowContext(ctx, query, sourceID).Scan(
		&checkpoint.SourceID,
		&checkpoint.LSN,
		&transactionID,
		&checkpoint.CommittedAt,
		&metadataJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No checkpoint found
		}
		return nil, fmt.Errorf("load checkpoint: %w", err)
	}

	if transactionID.Valid {
		checkpoint.TransactionID = transactionID.Int64
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &checkpoint.Metadata); err != nil {
			m.logger.Warn("failed to unmarshal checkpoint metadata", "error", err)
		}
	}

	m.logger.Debug("checkpoint loaded",
		"source_id", checkpoint.SourceID,
		"lsn", checkpoint.LSN,
	)

	return &checkpoint, nil
}

// Delete removes a checkpoint for a source.
func (m *PostgresManager) Delete(ctx context.Context, sourceID string) error {
	query := `DELETE FROM philotes.cdc_checkpoints WHERE source_id = $1`

	_, err := m.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("delete checkpoint: %w", err)
	}

	m.logger.Debug("checkpoint deleted", "source_id", sourceID)

	return nil
}

// Close closes the database connection.
func (m *PostgresManager) Close() error {
	return m.db.Close()
}

// Ensure PostgresManager implements Manager interface.
var _ Manager = (*PostgresManager)(nil)
