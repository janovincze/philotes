// Package buffer provides CDC event buffering for reliability and batch processing.
package buffer

import (
	"context"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
)

// Manager handles CDC event buffering operations.
type Manager interface {
	// Write stores events in the buffer.
	Write(ctx context.Context, events []cdc.Event) error

	// ReadBatch retrieves a batch of unprocessed events for a source.
	ReadBatch(ctx context.Context, sourceID string, limit int) ([]BufferedEvent, error)

	// MarkProcessed marks events as processed by their IDs.
	MarkProcessed(ctx context.Context, eventIDs []int64) error

	// Cleanup removes old processed events based on retention policy.
	Cleanup(ctx context.Context, retention time.Duration) (int64, error)

	// Stats returns buffer statistics.
	Stats(ctx context.Context) (Stats, error)

	// Close releases any resources held by the manager.
	Close() error
}

// BufferedEvent wraps a CDC event with buffer-specific metadata.
type BufferedEvent struct {
	// ID is the buffer database ID.
	ID int64

	// Event is the original CDC event.
	Event cdc.Event

	// CreatedAt is when the event was buffered.
	CreatedAt time.Time

	// ProcessedAt is when the event was processed (nil if unprocessed).
	ProcessedAt *time.Time
}

// Stats holds buffer statistics.
type Stats struct {
	// TotalEvents is the total number of events in the buffer.
	TotalEvents int64

	// UnprocessedEvents is the number of unprocessed events.
	UnprocessedEvents int64

	// OldestUnprocessed is the timestamp of the oldest unprocessed event.
	OldestUnprocessed *time.Time

	// Lag is the duration since the oldest unprocessed event.
	Lag time.Duration
}

// Config holds configuration for buffer managers.
type Config struct {
	// Enabled indicates whether buffering is enabled.
	Enabled bool

	// DSN is the database connection string.
	DSN string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// Retention is how long to keep processed events.
	Retention time.Duration

	// CleanupInterval is how often to run cleanup.
	CleanupInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		Retention:       168 * time.Hour, // 7 days
		CleanupInterval: time.Hour,
	}
}
