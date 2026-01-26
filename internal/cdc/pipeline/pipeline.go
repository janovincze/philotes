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
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/cdc/source"
	"github.com/janovincze/philotes/internal/metrics"
)

// Pipeline orchestrates the CDC flow from source to checkpointing.
type Pipeline struct {
	source       source.Source
	checkpoint   checkpoint.Manager
	buffer       buffer.Manager
	logger       *slog.Logger
	config       Config
	stateMachine *StateMachine

	// Optional components
	backpressure *BackpressureController
	retryer      *Retryer

	mu      sync.RWMutex
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

	// RetryPolicy configures retry behavior.
	RetryPolicy RetryPolicy

	// BackpressureConfig configures backpressure handling.
	BackpressureConfig BackpressureConfig
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		CheckpointInterval: 10 * time.Second,
		CheckpointEnabled:  true,
		BufferEnabled:      true,
		RetryPolicy:        DefaultRetryPolicy(),
		BackpressureConfig: DefaultBackpressureConfig(),
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
	RetryCount        int64
	State             State
}

// New creates a new CDC pipeline.
func New(src source.Source, cp checkpoint.Manager, buf buffer.Manager, cfg Config, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}

	retryer := NewRetryer(cfg.RetryPolicy, logger)
	retryer.SetSourceName(src.Name())

	p := &Pipeline{
		source:       src,
		checkpoint:   cp,
		buffer:       buf,
		config:       cfg,
		logger:       logger.With("component", "pipeline", "source", src.Name()),
		stateMachine: NewStateMachine(),
		retryer:      retryer,
	}

	// Add state change listener for logging and metrics
	p.stateMachine.AddListener(func(from, to State) {
		p.logger.Info("pipeline state changed", "from", from, "to", to)
		p.mu.Lock()
		p.stats.State = to
		p.mu.Unlock()

		// Update pipeline state metric
		metrics.CDCPipelineState.WithLabelValues(src.Name()).Set(float64(to))
	})

	return p
}

// SetBackpressureController sets the backpressure controller.
func (p *Pipeline) SetBackpressureController(bp *BackpressureController) {
	p.backpressure = bp
	// Set the state machine so the controller can pause/resume the pipeline
	bp.SetStateMachine(p.stateMachine)
}

// Run starts the pipeline and blocks until context is cancelled or an error occurs.
func (p *Pipeline) Run(ctx context.Context) error {
	if err := p.stateMachine.Transition(StateRunning); err != nil {
		// Already starting or running
		return fmt.Errorf("pipeline state transition failed: %w", err)
	}

	defer func() {
		if err := p.stateMachine.Transition(StateStopped); err != nil {
			p.logger.Warn("failed to transition to stopped state", "error", err)
		}
	}()

	p.logger.Info("starting CDC pipeline")

	// Try to restore from last checkpoint
	if p.config.CheckpointEnabled && p.checkpoint != nil {
		if err := p.restoreCheckpoint(ctx); err != nil {
			p.logger.Warn("failed to restore checkpoint", "error", err)
			// Continue without checkpoint
		}
	}

	// Start backpressure controller if configured
	if p.backpressure != nil {
		go p.backpressure.Start(ctx)
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
			if err := p.stateMachine.Transition(StateStopping); err != nil {
				p.logger.Warn("failed to transition to stopping state", "error", err)
			}
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

				// Record error metric
				metrics.CDCErrorsTotal.WithLabelValues(p.source.Name(), "source").Inc()

				p.logger.Error("source error", "error", err)

				// Transition to failed state
				if transErr := p.stateMachine.Transition(StateFailed); transErr != nil {
					p.logger.Warn("failed to transition to failed state", "error", transErr)
				}

				return fmt.Errorf("source error: %w", err)
			}

		case event, ok := <-events:
			if !ok {
				p.logger.Info("event channel closed")
				return nil
			}

			// Check if we should process (not paused due to backpressure)
			if !p.stateMachine.CanProcess() {
				// Wait for resume or context cancellation
				p.logger.Debug("pipeline paused, waiting to resume")
				for !p.stateMachine.CanProcess() {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
					}
				}
			}

			if err := p.processEventWithRetry(ctx, event); err != nil {
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

// processEventWithRetry processes an event with retry logic.
func (p *Pipeline) processEventWithRetry(ctx context.Context, event cdc.Event) error {
	return p.retryer.Execute(ctx, func(ctx context.Context) error {
		return p.processEvent(ctx, event)
	})
}

func (p *Pipeline) processEvent(ctx context.Context, event cdc.Event) error {
	now := time.Now()

	p.mu.Lock()
	p.lastLSN = event.LSN
	p.stats.EventsProcessed++
	p.stats.LastEventTime = now
	p.mu.Unlock()

	// Record CDC event metric
	tableName := event.FullyQualifiedTable()
	metrics.CDCEventsTotal.WithLabelValues(p.source.Name(), tableName, string(event.Operation)).Inc()

	// Calculate and record lag if event has a timestamp
	if !event.Timestamp.IsZero() {
		lag := now.Sub(event.Timestamp).Seconds()
		metrics.CDCLagSeconds.WithLabelValues(p.source.Name(), tableName).Set(lag)
	}

	p.logger.Debug("processed event",
		"table", tableName,
		"operation", event.Operation,
		"lsn", event.LSN,
	)

	// Write event to buffer if enabled
	if p.config.BufferEnabled && p.buffer != nil {
		if err := p.buffer.Write(ctx, []cdc.Event{event}); err != nil {
			// Record buffer error metric
			metrics.CDCErrorsTotal.WithLabelValues(p.source.Name(), "buffer").Inc()

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
	stats := p.stats
	stats.State = p.stateMachine.State()
	return stats
}

// IsRunning returns whether the pipeline is currently running.
func (p *Pipeline) IsRunning() bool {
	return p.stateMachine.IsRunning()
}

// State returns the current pipeline state.
func (p *Pipeline) State() State {
	return p.stateMachine.State()
}

// Pause pauses the pipeline.
func (p *Pipeline) Pause() error {
	return p.stateMachine.Transition(StatePaused)
}

// Resume resumes the pipeline.
func (p *Pipeline) Resume() error {
	return p.stateMachine.Transition(StateRunning)
}

// HealthChecker returns a health checker for the pipeline.
func (p *Pipeline) HealthChecker() health.HealthChecker {
	return health.NewComponentChecker("pipeline", func(ctx context.Context) (health.Status, string, error) {
		state := p.stateMachine.State()
		switch state {
		case StateRunning:
			return health.StatusHealthy, "pipeline is running", nil
		case StatePaused:
			return health.StatusDegraded, "pipeline is paused", nil
		case StateStarting:
			return health.StatusDegraded, "pipeline is starting", nil
		case StateStopping:
			return health.StatusDegraded, "pipeline is stopping", nil
		case StateStopped:
			return health.StatusUnhealthy, "pipeline is stopped", nil
		case StateFailed:
			return health.StatusUnhealthy, "pipeline has failed", nil
		default:
			return health.StatusUnknown, "unknown state", nil
		}
	})
}
