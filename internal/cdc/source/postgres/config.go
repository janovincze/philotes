// Package postgres provides a PostgreSQL CDC source implementation using pgstream.
package postgres

import (
	"time"

	"github.com/janovincze/philotes/internal/cdc/source"
)

// Config holds configuration for the PostgreSQL CDC source.
type Config struct {
	source.Config

	// ConnectionURL is the PostgreSQL connection URL.
	ConnectionURL string

	// SlotName is the name of the replication slot.
	SlotName string

	// PublicationName is the name of the publication to subscribe to.
	PublicationName string

	// Tables is a list of tables to capture (empty means all tables in publication).
	Tables []string

	// ReconnectInterval is the interval between reconnection attempts.
	ReconnectInterval time.Duration

	// MaxReconnectAttempts is the maximum number of reconnection attempts (0 = unlimited).
	MaxReconnectAttempts int

	// EventBufferSize is the size of the internal event buffer.
	EventBufferSize int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Config: source.Config{
			Name: "postgres",
		},
		SlotName:             "philotes_cdc",
		PublicationName:      "philotes_pub",
		ReconnectInterval:    5 * time.Second,
		MaxReconnectAttempts: 0, // unlimited
		EventBufferSize:      1000,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.ConnectionURL == "" {
		return ErrMissingConnectionURL
	}
	if c.SlotName == "" {
		return ErrMissingSlotName
	}
	if c.PublicationName == "" {
		return ErrMissingPublicationName
	}
	return nil
}
