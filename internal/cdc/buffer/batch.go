package buffer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// BatchHandler is called when a batch of events is ready for processing.
type BatchHandler func(ctx context.Context, events []BufferedEvent) error

// BatchProcessor reads events from the buffer in batches and processes them.
type BatchProcessor struct {
	manager  Manager
	handler  BatchHandler
	logger   *slog.Logger
	config   BatchConfig

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
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
}

// DefaultBatchConfig returns a BatchConfig with sensible defaults.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		BatchSize:       1000,
		FlushInterval:   5 * time.Second,
		Retention:       168 * time.Hour, // 7 days
		CleanupInterval: time.Hour,
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
			if err := p.processBatch(ctx); err != nil {
				p.logger.Error("failed to process batch", "error", err)
			}
		}
	}
}

func (p *BatchProcessor) processBatch(ctx context.Context) error {
	// Read a batch of unprocessed events
	events, err := p.manager.ReadBatch(ctx, p.config.SourceID, p.config.BatchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	p.logger.Debug("processing batch", "count", len(events))

	// Call the handler to process the events
	if err := p.handler(ctx, events); err != nil {
		p.logger.Error("handler failed", "error", err, "count", len(events))
		return err
	}

	// Mark events as processed
	eventIDs := make([]int64, len(events))
	for i, e := range events {
		eventIDs[i] = e.ID
	}

	if err := p.manager.MarkProcessed(ctx, eventIDs); err != nil {
		return err
	}

	p.logger.Debug("batch processed successfully", "count", len(events))
	return nil
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
		}
	}
}

// IsRunning returns whether the processor is currently running.
func (p *BatchProcessor) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
