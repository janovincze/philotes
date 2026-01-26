package buffer

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/janovincze/philotes/internal/cdc/deadletter"
	"github.com/janovincze/philotes/internal/metrics"
)

// BatchHandler is called when a batch of events is ready for processing.
type BatchHandler func(ctx context.Context, events []BufferedEvent) error

// BatchProcessor reads events from the buffer in batches and processes them.
type BatchProcessor struct {
	manager    Manager
	handler    BatchHandler
	deadLetter deadletter.Manager
	logger     *slog.Logger
	config     BatchConfig

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
	stats   BatchStats
}

// BatchStats holds batch processing statistics.
type BatchStats struct {
	BatchesProcessed int64
	EventsProcessed  int64
	EventsFailed     int64
	RetryCount       int64
	DLQCount         int64
}

// BatchConfig holds configuration for the batch processor.
type BatchConfig struct {
	// SourceID identifies the source to read events from.
	SourceID string

	// BatchSize is the maximum number of events per batch.
	BatchSize int

	// FlushInterval is how often to check for new events.
	FlushInterval time.Duration

	// Retention is how long to keep processed events.
	Retention time.Duration

	// CleanupInterval is how often to run cleanup.
	CleanupInterval time.Duration

	// Retry configuration
	RetryMaxAttempts     int
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration
	RetryMultiplier      float64

	// DLQ configuration
	DLQEnabled   bool
	DLQRetention time.Duration
}

// DefaultBatchConfig returns a BatchConfig with sensible defaults.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		BatchSize:            1000,
		FlushInterval:        5 * time.Second,
		Retention:            168 * time.Hour, // 7 days
		CleanupInterval:      time.Hour,
		RetryMaxAttempts:     3,
		RetryInitialInterval: time.Second,
		RetryMaxInterval:     30 * time.Second,
		RetryMultiplier:      2.0,
		DLQEnabled:           true,
		DLQRetention:         168 * time.Hour, // 7 days
	}
}

// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(manager Manager, handler BatchHandler, cfg BatchConfig, logger *slog.Logger) *BatchProcessor {
	if logger == nil {
		logger = slog.Default()
	}

	return &BatchProcessor{
		manager: manager,
		handler: handler,
		logger:  logger.With("component", "batch-processor"),
		config:  cfg,
		stopCh:  make(chan struct{}),
	}
}

// SetDeadLetterManager sets the dead-letter queue manager.
func (p *BatchProcessor) SetDeadLetterManager(dlq deadletter.Manager) {
	p.deadLetter = dlq
}

// Start begins processing batches.
func (p *BatchProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.mu.Unlock()

	p.logger.Info("starting batch processor",
		"batch_size", p.config.BatchSize,
		"flush_interval", p.config.FlushInterval,
		"retry_max_attempts", p.config.RetryMaxAttempts,
		"dlq_enabled", p.config.DLQEnabled,
	)

	// Start the processing goroutine
	p.wg.Add(1)
	go p.processLoop(ctx)

	// Start the cleanup goroutine if retention is configured
	if p.config.CleanupInterval > 0 && p.config.Retention > 0 {
		p.wg.Add(1)
		go p.cleanupLoop(ctx)
	}

	return nil
}

// Stop stops the batch processor.
func (p *BatchProcessor) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	close(p.stopCh)

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("batch processor stopped")
	case <-ctx.Done():
		p.logger.Warn("batch processor stop timed out")
	}

	return nil
}

func (p *BatchProcessor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			// Update buffer depth metric
			p.updateBufferDepthMetric(ctx)

			if err := p.processBatchWithRetry(ctx); err != nil {
				p.logger.Error("failed to process batch", "error", err)
			}
		}
	}
}

// updateBufferDepthMetric updates the buffer depth gauge metric.
func (p *BatchProcessor) updateBufferDepthMetric(ctx context.Context) {
	stats, err := p.manager.Stats(ctx)
	if err != nil {
		p.logger.Debug("failed to get buffer stats for metrics", "error", err)
		return
	}
	metrics.BufferDepth.WithLabelValues(p.config.SourceID).Set(float64(stats.UnprocessedEvents))
}

func (p *BatchProcessor) processBatchWithRetry(ctx context.Context) error {
	// Read a batch of unprocessed events
	events, err := p.manager.ReadBatch(ctx, p.config.SourceID, p.config.BatchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	p.logger.Debug("processing batch", "count", len(events))

	// Try to process with retries
	var lastErr error
	for attempt := 1; attempt <= p.config.RetryMaxAttempts; attempt++ {
		err := p.handler(ctx, events)
		if err == nil {
			// Success - mark events as processed
			eventIDs := make([]int64, len(events))
			for i, e := range events {
				eventIDs[i] = e.ID
			}

			if markErr := p.manager.MarkProcessed(ctx, eventIDs); markErr != nil {
				return markErr
			}

			p.mu.Lock()
			p.stats.BatchesProcessed++
			p.stats.EventsProcessed += int64(len(events))
			p.mu.Unlock()

			// Record batch success metrics
			metrics.BufferBatchesTotal.WithLabelValues(p.config.SourceID, "success").Inc()
			metrics.BufferEventsProcessedTotal.WithLabelValues(p.config.SourceID).Add(float64(len(events)))

			p.logger.Debug("batch processed successfully", "count", len(events))
			return nil
		}

		lastErr = err
		p.mu.Lock()
		p.stats.RetryCount++
		p.mu.Unlock()

		p.logger.Warn("batch processing failed, retrying",
			"attempt", attempt,
			"max_attempts", p.config.RetryMaxAttempts,
			"error", err,
		)

		// Don't retry on last attempt
		if attempt >= p.config.RetryMaxAttempts {
			break
		}

		// Wait with exponential backoff
		wait := p.calculateBackoff(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}

	// Max retries exceeded - send to DLQ if enabled
	if p.config.DLQEnabled && p.deadLetter != nil {
		p.sendToDLQ(ctx, events, lastErr)
	}

	p.mu.Lock()
	p.stats.EventsFailed += int64(len(events))
	p.mu.Unlock()

	// Record batch failure metrics
	metrics.BufferBatchesTotal.WithLabelValues(p.config.SourceID, "failed").Inc()

	// Mark events as processed even though they failed (they're in DLQ now)
	eventIDs := make([]int64, len(events))
	for i, e := range events {
		eventIDs[i] = e.ID
	}
	if markErr := p.manager.MarkProcessed(ctx, eventIDs); markErr != nil {
		p.logger.Error("failed to mark failed events as processed", "error", markErr)
	}

	return lastErr
}

func (p *BatchProcessor) calculateBackoff(attempt int) time.Duration {
	backoff := float64(p.config.RetryInitialInterval) * math.Pow(p.config.RetryMultiplier, float64(attempt-1))
	if backoff > float64(p.config.RetryMaxInterval) {
		backoff = float64(p.config.RetryMaxInterval)
	}
	return time.Duration(backoff)
}

func (p *BatchProcessor) sendToDLQ(ctx context.Context, events []BufferedEvent, err error) {
	for _, bufferedEvent := range events {
		eventData, marshalErr := json.Marshal(bufferedEvent.Event)
		if marshalErr != nil {
			p.logger.Error("failed to marshal event for DLQ",
				"event_id", bufferedEvent.ID,
				"error", marshalErr,
			)
			continue
		}

		now := time.Now()
		expiresAt := now.Add(p.config.DLQRetention)

		failedEvent := deadletter.FailedEvent{
			OriginalEventID: bufferedEvent.ID,
			SourceID:        bufferedEvent.Event.ID,
			SchemaName:      bufferedEvent.Event.Schema,
			TableName:       bufferedEvent.Event.Table,
			Operation:       string(bufferedEvent.Event.Operation),
			EventData:       eventData,
			ErrorMessage:    err.Error(),
			ErrorType:       deadletter.ErrorTypeTransient,
			CreatedAt:       now,
			ExpiresAt:       &expiresAt,
		}

		if dlqErr := p.deadLetter.Write(ctx, failedEvent); dlqErr != nil {
			p.logger.Error("failed to write to DLQ",
				"event_id", bufferedEvent.ID,
				"error", dlqErr,
			)
		} else {
			p.mu.Lock()
			p.stats.DLQCount++
			p.mu.Unlock()

			// Record DLQ metric
			metrics.BufferDLQTotal.WithLabelValues(p.config.SourceID).Inc()

			p.logger.Info("event sent to DLQ",
				"event_id", bufferedEvent.ID,
				"table", bufferedEvent.Event.Schema+"."+bufferedEvent.Event.Table,
			)
		}
	}
}

// processBatch processes a batch of events.
// Deprecated: This wrapper is kept for backward compatibility.
// Use processBatchWithRetry instead.
func (p *BatchProcessor) processBatch(ctx context.Context) error {
	return p.processBatchWithRetry(ctx)
}

func (p *BatchProcessor) cleanupLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			deleted, err := p.manager.Cleanup(ctx, p.config.Retention)
			if err != nil {
				p.logger.Error("cleanup failed", "error", err)
			} else if deleted > 0 {
				p.logger.Info("cleanup completed", "deleted", deleted)
			}

			// Also cleanup DLQ if enabled
			if p.deadLetter != nil {
				dlqDeleted, dlqErr := p.deadLetter.Cleanup(ctx)
				if dlqErr != nil {
					p.logger.Error("DLQ cleanup failed", "error", dlqErr)
				} else if dlqDeleted > 0 {
					p.logger.Info("DLQ cleanup completed", "deleted", dlqDeleted)
				}
			}
		}
	}
}

// IsRunning returns whether the processor is currently running.
func (p *BatchProcessor) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Stats returns the batch processing statistics.
func (p *BatchProcessor) Stats() BatchStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}
