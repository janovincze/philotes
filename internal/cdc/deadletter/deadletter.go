// Package deadletter provides dead-letter queue functionality for failed CDC events.
package deadletter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
)

// ErrorType classifies the type of error that caused the event to fail.
type ErrorType string

const (
	// ErrorTypeTransient indicates a temporary error that may succeed on retry.
	ErrorTypeTransient ErrorType = "transient"
	// ErrorTypePermanent indicates a permanent error that will not succeed on retry.
	ErrorTypePermanent ErrorType = "permanent"
	// ErrorTypeValidation indicates a data validation error.
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeSchema indicates a schema-related error.
	ErrorTypeSchema ErrorType = "schema"
	// ErrorTypeUnknown indicates an unknown error type.
	ErrorTypeUnknown ErrorType = "unknown"
)

// FailedEvent represents a CDC event that failed processing.
type FailedEvent struct {
	// ID is the unique identifier for this dead-letter entry.
	ID int64 `json:"id"`

	// OriginalEventID is the ID of the original event (if available).
	OriginalEventID int64 `json:"original_event_id,omitempty"`

	// SourceID identifies the CDC source.
	SourceID string `json:"source_id"`

	// SchemaName is the database schema name.
	SchemaName string `json:"schema_name"`

	// TableName is the table name.
	TableName string `json:"table_name"`

	// Operation is the CDC operation type.
	Operation string `json:"operation"`

	// EventData contains the full event data as JSON.
	EventData json.RawMessage `json:"event_data"`

	// ErrorMessage is the error that caused the failure.
	ErrorMessage string `json:"error_message"`

	// ErrorType classifies the type of error.
	ErrorType ErrorType `json:"error_type"`

	// RetryCount is the number of times this event has been retried from DLQ.
	RetryCount int `json:"retry_count"`

	// CreatedAt is when the event was added to the DLQ.
	CreatedAt time.Time `json:"created_at"`

	// LastRetryAt is when the event was last retried.
	LastRetryAt *time.Time `json:"last_retry_at,omitempty"`

	// ExpiresAt is when the event will be deleted.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Manager defines the interface for dead-letter queue operations.
type Manager interface {
	// Write adds a failed event to the dead-letter queue.
	Write(ctx context.Context, event FailedEvent) error

	// Read retrieves failed events from the dead-letter queue.
	Read(ctx context.Context, limit int) ([]FailedEvent, error)

	// ReadBySource retrieves failed events for a specific source.
	ReadBySource(ctx context.Context, sourceID string, limit int) ([]FailedEvent, error)

	// ReadByTable retrieves failed events for a specific table.
	ReadByTable(ctx context.Context, schemaName, tableName string, limit int) ([]FailedEvent, error)

	// MarkRetried updates the retry count and last retry time.
	MarkRetried(ctx context.Context, eventID int64) error

	// Delete removes an event from the dead-letter queue.
	Delete(ctx context.Context, eventID int64) error

	// Cleanup removes expired events from the dead-letter queue.
	Cleanup(ctx context.Context) (int64, error)

	// Count returns the number of events in the dead-letter queue.
	Count(ctx context.Context) (int64, error)

	// Close releases any resources held by the manager.
	Close() error
}

// FromCDCEvent creates a FailedEvent from a CDC event and error.
func FromCDCEvent(event cdc.Event, err error, errType ErrorType, retention time.Duration) (FailedEvent, error) {
	eventData, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		return FailedEvent{}, marshalErr
	}

	now := time.Now()
	expiresAt := now.Add(retention)

	return FailedEvent{
		SourceID:     event.ID,
		SchemaName:   event.Schema,
		TableName:    event.Table,
		Operation:    string(event.Operation),
		EventData:    eventData,
		ErrorMessage: err.Error(),
		ErrorType:    errType,
		CreatedAt:    now,
		ExpiresAt:    &expiresAt,
	}, nil
}

// ToEvent converts a FailedEvent back to a cdc.Event.
func (f *FailedEvent) ToEvent() (cdc.Event, error) {
	var event cdc.Event
	if err := json.Unmarshal(f.EventData, &event); err != nil {
		return cdc.Event{}, err
	}
	return event, nil
}

// Stats holds dead-letter queue statistics.
type Stats struct {
	// TotalCount is the total number of events in the DLQ.
	TotalCount int64 `json:"total_count"`

	// BySource is the count grouped by source ID.
	BySource map[string]int64 `json:"by_source"`

	// ByErrorType is the count grouped by error type.
	ByErrorType map[ErrorType]int64 `json:"by_error_type"`

	// OldestEvent is the timestamp of the oldest event.
	OldestEvent *time.Time `json:"oldest_event,omitempty"`

	// NewestEvent is the timestamp of the newest event.
	NewestEvent *time.Time `json:"newest_event,omitempty"`
}
