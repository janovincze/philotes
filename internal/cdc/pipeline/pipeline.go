// Package pipeline provides CDC pipeline orchestration.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
	"github.com/janovincze/philotes/internal/cdc/buffer"
	"github.com/janovincze/philotes/internal/cdc/checkpoint"
	"github.com/janovincze/philotes/internal/cdc/source"
)

// Pipeline orchestrates the CDC flow from source to checkpointing.
type Pipeline struct {
	source      source.Source
	checkpoint  checkpoint.Manager
	buffer      buffer.Manager
	logger      *slog.Logger
	config      Config

	mu      sync.RWMutex
	running bool
	lastLSN string
	stats   Stats
}

// Config holds pipeline configuration.
type Config struct {
	// CheckpointInterval is how often to save checkpoints.
	CheckpointInterval time.Duration

	// CheckpointEnabled enables checkpoint saving.
	CheckpointEnabled bool

	// BufferEnabled enables writing events to buffer.
	BufferEnabled bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		CheckpointInterval: 10 * time.Second,
		CheckpointEnabled:  true,
		BufferEnabled:      true,
	}
}

// Stats holds pipeline statistics.
type Stats struct {
	EventsProcessed   int64
	EventsBuffered    int64
	LastEventTime     time.Time
	LastCheckpointLSN string
	LastCheckpointAt  time.Time
	Errors            int64
}

// New creates a new CDC pipeline.
func New(src source.Source, cp checkpoint.Manager, buf buffer.Manager, cfg Config, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}

	return &Pipeline{
		source:     src,
		checkpoint: cp,
		buffer:     buf,
		config:     cfg,
		logger:     logger.With("component", "pipeline", "source", src.Name()),
	}
}

// Run starts the pipeline and blocks until context is cancelled or an error occurs.
func (p *Pipeline) Run(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	p.logger.Info("starting CDC pipeline")

	// Try to restore from last checkpoint
	if p.config.CheckpointEnabled && p.checkpoint != nil {
		if err := p.restoreCheckpoint(ctx); err != nil {
			p.logger.Warn("failed to restore checkpoint", "error", err)
			// Continue without checkpoint
		}
	}

	// Start the source
	events, errors := p.source.Start(ctx)

	// Start checkpoint ticker if enabled
	var checkpointTicker *time.Ticker
	var checkpointCh <-chan time.Time
	if p.config.CheckpointEnabled && p.checkpoint != nil && p.config.CheckpointInterval > 0 {
		checkpointTicker = time.NewTicker(p.config.CheckpointInterval)
		checkpointCh = checkpointTicker.C
		defer checkpointTicker.Stop()
	}

	p.logger.Info("pipeline running, processing events")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline stopping", "reason", ctx.Err())
			// Save final checkpoint before exiting
			if p.config.CheckpointEnabled && p.checkpoint != nil {
				if err := p.saveCheckpoint(context.Background()); err != nil {
					p.logger.Error("failed to save final checkpoint", "error", err)
				}
			}
			return p.source.Stop(context.Background())

		case err := <-errors:
			if err != nil {
				p.mu.Lock()
				p.stats.Errors++
				p.mu.Unlock()
				p.logger.Error("source error", "error", err)
				return fmt.Errorf("source error: %w", err)
			}

		case event, ok := <-events:
			if !ok {
				p.logger.Info("event channel closed")
				return nil
			}
			if err := p.processEvent(ctx, event); err != nil {
				p.logger.Error("failed to process event", "error", err)
				// Continue processing other events
			}

		case <-checkpointCh:
			if err := p.saveCheckpoint(ctx); err != nil {
				p.logger.Error("failed to save checkpoint", "error", err)
			}
		}
	}
}

func (p *Pipeline) processEvent(ctx context.Context, event cdc.Event) error {
	p.mu.Lock()
	p.lastLSN = event.LSN
	p.stats.EventsProcessed++
	p.stats.LastEventTime = time.Now()
	p.mu.Unlock()

	p.logger.Debug("processed event",
		"table", event.FullyQualifiedTable(),
		"operation", event.Operation,
		"lsn", event.LSN,
	)

	// Write event to buffer if enabled
	if p.config.BufferEnabled && p.buffer != nil {
		if err := p.buffer.Write(ctx, []cdc.Event{event}); err != nil {
			p.logger.Error("failed to write event to buffer",
				"error", err,
				"lsn", event.LSN,
			)
			return fmt.Errorf("buffer write: %w", err)
		}

		p.mu.Lock()
		p.stats.EventsBuffered++
		p.mu.Unlock()
	}

	return nil
}

func (p *Pipeline) saveCheckpoint(ctx context.Context) error {
	p.mu.RLock()
	lsn := p.lastLSN
	p.mu.RUnlock()

	if lsn == "" {
		return nil // Nothing to checkpoint
	}

	checkpoint := cdc.Checkpoint{
		SourceID:    p.source.Name(),
		LSN:         lsn,
		CommittedAt: time.Now(),
	}

	if err := p.checkpoint.Save(ctx, checkpoint); err != nil {
		return err
	}

	p.mu.Lock()
	p.stats.LastCheckpointLSN = lsn
	p.stats.LastCheckpointAt = time.Now()
	p.mu.Unlock()

	p.logger.Debug("checkpoint saved", "lsn", lsn)

	return nil
}

func (p *Pipeline) restoreCheckpoint(ctx context.Context) error {
	checkpoint, err := p.checkpoint.Load(ctx, p.source.Name())
	if err != nil {
		return err
	}

	if checkpoint == nil {
		p.logger.Info("no checkpoint found, starting from beginning")
		return nil
	}

	p.mu.Lock()
	p.lastLSN = checkpoint.LSN
	p.stats.LastCheckpointLSN = checkpoint.LSN
	p.stats.LastCheckpointAt = checkpoint.CommittedAt
	p.mu.Unlock()

	p.logger.Info("restored checkpoint",
		"lsn", checkpoint.LSN,
		"committed_at", checkpoint.CommittedAt,
	)

	return nil
}

// Stats returns the current pipeline statistics.
func (p *Pipeline) Stats() Stats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// IsRunning returns whether the pipeline is currently running.
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}
