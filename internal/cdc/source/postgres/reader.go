// Package postgres provides a PostgreSQL CDC source implementation using pgstream.
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xataio/pgstream/pkg/wal"
	"github.com/xataio/pgstream/pkg/wal/listener"
	pglistener "github.com/xataio/pgstream/pkg/wal/listener/postgres"
	pgreplication "github.com/xataio/pgstream/pkg/wal/replication/postgres"

	"github.com/janovincze/philotes/internal/cdc"
	"github.com/janovincze/philotes/internal/cdc/source"
)

// Reader is a PostgreSQL CDC source that uses pgstream for logical replication.
type Reader struct {
	config   Config
	logger   *slog.Logger
	listener listener.Listener

	events chan cdc.Event
	errors chan error

	mu        sync.RWMutex
	started   bool
	lastLSN   string
	stopOnce  sync.Once
	closeOnce sync.Once
}

// New creates a new PostgreSQL CDC reader with the given configuration.
func New(cfg Config, logger *slog.Logger) (*Reader, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Reader{
		config: cfg,
		logger: logger.With("component", "postgres-reader", "source", cfg.Name),
		events: make(chan cdc.Event, cfg.EventBufferSize),
		errors: make(chan error, 1),
	}, nil
}

// Start begins capturing CDC events from PostgreSQL.
func (r *Reader) Start(ctx context.Context) (<-chan cdc.Event, <-chan error) {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		r.errors <- ErrAlreadyStarted
		return r.events, r.errors
	}
	r.started = true
	r.mu.Unlock()

	go r.run(ctx)

	return r.events, r.errors
}

// Stop gracefully stops the reader.
func (r *Reader) Stop(ctx context.Context) error {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return ErrNotStarted
	}
	r.mu.Unlock()

	var err error
	r.stopOnce.Do(func() {
		if r.listener != nil {
			err = r.listener.Close()
		}
		r.closeOnce.Do(func() {
			close(r.events)
		})
	})

	return err
}

// LastLSN returns the last processed LSN.
func (r *Reader) LastLSN() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastLSN
}

// Name returns the name of this source.
func (r *Reader) Name() string {
	return r.config.Name
}

func (r *Reader) run(ctx context.Context) {
	r.logger.Info("starting PostgreSQL CDC reader",
		"slot", r.config.SlotName,
		"publication", r.config.PublicationName,
	)

	// Create the replication handler
	handlerCfg := pgreplication.Config{
		PostgresURL:         r.config.ConnectionURL,
		ReplicationSlotName: r.config.SlotName,
		IncludeTables:       r.config.Tables,
	}

	handler, err := pgreplication.NewHandler(ctx, handlerCfg)
	if err != nil {
		r.logger.Error("failed to create replication handler", "error", err)
		r.errors <- fmt.Errorf("%w: %v", ErrConnectionFailed, err)
		return
	}
	defer handler.Close()

	// Create the WAL listener with our event processor
	r.listener = pglistener.New(handler, r.processWALEvent)

	r.logger.Info("connected to PostgreSQL, starting replication")

	// Start listening - this blocks until context is cancelled or error
	if err := r.listener.Listen(ctx); err != nil {
		if ctx.Err() != nil {
			r.logger.Info("reader stopped", "reason", ctx.Err())
			return
		}
		r.logger.Error("replication failed", "error", err)
		r.errors <- fmt.Errorf("%w: %v", ErrReplicationFailed, err)
	}
}

func (r *Reader) processWALEvent(ctx context.Context, event *wal.Event) error {
	if event == nil {
		return nil
	}

	// Update last LSN
	r.mu.Lock()
	r.lastLSN = string(event.CommitPosition)
	r.mu.Unlock()

	// Handle keep-alive events (no data, just checkpoint position)
	if event.Data == nil {
		return nil
	}

	cdcEvent, err := r.convertEvent(event)
	if err != nil {
		r.logger.Warn("failed to convert WAL event", "error", err)
		return nil // Don't fail on conversion errors, log and continue
	}

	select {
	case r.events <- cdcEvent:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (r *Reader) convertEvent(event *wal.Event) (cdc.Event, error) {
	data := event.Data

	// Parse timestamp
	ts, err := data.GetTimestamp()
	if err != nil {
		ts = time.Now() // Fallback to current time
	}

	// Convert operation
	op := r.convertOperation(data.Action)

	// Extract column data
	before, after, keyColumns := r.extractColumnData(data, op)

	return cdc.Event{
		ID:            uuid.New().String(),
		LSN:           data.LSN,
		TransactionID: 0, // pgstream doesn't expose transaction ID directly
		Timestamp:     ts,
		Schema:        data.Schema,
		Table:         data.Table,
		Operation:     op,
		Before:        before,
		After:         after,
		KeyColumns:    keyColumns,
		Metadata: map[string]any{
			"commit_position": string(event.CommitPosition),
		},
	}, nil
}

func (r *Reader) convertOperation(action string) cdc.Operation {
	switch action {
	case "I":
		return cdc.OperationInsert
	case "U":
		return cdc.OperationUpdate
	case "D":
		return cdc.OperationDelete
	case "T":
		return cdc.OperationTruncate
	default:
		return cdc.Operation(action)
	}
}

func (r *Reader) extractColumnData(data *wal.Data, op cdc.Operation) (before, after map[string]any, keyColumns []string) {
	// Extract key columns from identity
	for _, col := range data.Identity {
		keyColumns = append(keyColumns, col.Name)
	}

	// For INSERT and UPDATE, columns contain the new values (after)
	// For DELETE, identity contains the old values (before)
	// For UPDATE, identity contains the old key values (before)

	switch op {
	case cdc.OperationInsert:
		after = r.columnsToMap(data.Columns)
	case cdc.OperationUpdate:
		before = r.columnsToMap(data.Identity)
		after = r.columnsToMap(data.Columns)
	case cdc.OperationDelete:
		before = r.columnsToMap(data.Identity)
	case cdc.OperationTruncate:
		// No row data for truncate
	}

	return before, after, keyColumns
}

func (r *Reader) columnsToMap(columns []wal.Column) map[string]any {
	if len(columns) == 0 {
		return nil
	}
	result := make(map[string]any, len(columns))
	for _, col := range columns {
		result[col.Name] = col.Value
	}
	return result
}

// Ensure Reader implements source.Source interface.
var _ source.Source = (*Reader)(nil)
