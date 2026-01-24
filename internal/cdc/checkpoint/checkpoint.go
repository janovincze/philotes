// Package checkpoint provides checkpoint management for CDC pipelines.
package checkpoint

import (
	"context"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
)

// Manager handles checkpoint persistence and retrieval.
type Manager interface {
	// Save persists a checkpoint.
	Save(ctx context.Context, checkpoint cdc.Checkpoint) error

	// Load retrieves the latest checkpoint for a source.
	Load(ctx context.Context, sourceID string) (*cdc.Checkpoint, error)

	// Delete removes a checkpoint for a source.
	Delete(ctx context.Context, sourceID string) error

	// Close releases any resources held by the manager.
	Close() error
}

// Config holds configuration for checkpoint managers.
type Config struct {
	// Enabled indicates whether checkpointing is enabled.
	Enabled bool

	// Interval is the minimum time between checkpoint saves.
	Interval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:  true,
		Interval: 10 * time.Second,
	}
}
