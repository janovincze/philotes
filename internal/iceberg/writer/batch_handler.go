package writer

import (
	"context"

	"github.com/janovincze/philotes/internal/cdc/buffer"
)

// BatchHandler returns a buffer.BatchHandler function that writes events to Iceberg.
// This is the integration point between the CDC buffer and the Iceberg writer.
func BatchHandler(w Writer) buffer.BatchHandler {
	return func(ctx context.Context, events []buffer.BufferedEvent) error {
		return w.WriteEvents(ctx, events)
	}
}
