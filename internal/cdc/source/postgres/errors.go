package postgres

import "errors"

var (
	// ErrMissingConnectionURL is returned when the connection URL is not provided.
	ErrMissingConnectionURL = errors.New("postgres: connection URL is required")

	// ErrMissingSlotName is returned when the replication slot name is not provided.
	ErrMissingSlotName = errors.New("postgres: replication slot name is required")

	// ErrMissingPublicationName is returned when the publication name is not provided.
	ErrMissingPublicationName = errors.New("postgres: publication name is required")

	// ErrAlreadyStarted is returned when Start is called on an already started source.
	ErrAlreadyStarted = errors.New("postgres: source already started")

	// ErrNotStarted is returned when Stop is called on a source that hasn't started.
	ErrNotStarted = errors.New("postgres: source not started")

	// ErrConnectionFailed is returned when the connection to PostgreSQL fails.
	ErrConnectionFailed = errors.New("postgres: connection failed")

	// ErrReplicationFailed is returned when replication streaming fails.
	ErrReplicationFailed = errors.New("postgres: replication failed")
)
