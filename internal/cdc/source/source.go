// Package source provides CDC source implementations for capturing changes from databases.
package source

import (
	"context"

	"github.com/janovincze/philotes/internal/cdc"
)

// Source represents a CDC source that produces events from a database.
type Source interface {
	// Start begins capturing CDC events. The returned channel will receive events
	// until the context is cancelled or an error occurs.
	Start(ctx context.Context) (<-chan cdc.Event, <-chan error)

	// Stop gracefully stops the source and releases resources.
	Stop(ctx context.Context) error

	// LastLSN returns the last processed LSN, or empty string if none.
	LastLSN() string

	// Name returns the name/identifier of this source.
	Name() string
}

// Config holds common configuration for CDC sources.
type Config struct {
	// Name is a unique identifier for this source.
	Name string

	// StartLSN is the LSN to start from (empty means start from current).
	StartLSN string
}
