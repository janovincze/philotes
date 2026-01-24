package pipeline

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// BackpressureConfig holds configuration for backpressure handling.
type BackpressureConfig struct {
	// Enabled enables backpressure handling.
	Enabled bool

	// HighWatermark is the threshold to trigger pause.
	HighWatermark int

	// LowWatermark is the threshold to resume processing.
	LowWatermark int

	// CheckInterval is how often to check buffer size.
	CheckInterval time.Duration
}

// DefaultBackpressureConfig returns a BackpressureConfig with sensible defaults.
func DefaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		Enabled:       true,
		HighWatermark: 8000,
		LowWatermark:  5000,
		CheckInterval: time.Second,
	}
}

// BufferSizeFunc is a function that returns the current buffer size.
type BufferSizeFunc func(ctx context.Context) (int, error)

// BackpressureController monitors buffer size and signals pause/resume.
type BackpressureController struct {
	config       BackpressureConfig
	getSize      BufferSizeFunc
	stateMachine *StateMachine
	logger       *slog.Logger

	mu          sync.RWMutex
	paused      bool
	pausedAt    time.Time
	resumedAt   time.Time
	pauseCount  int64
	resumeCount int64
	lastSize    int
}

// NewBackpressureController creates a new BackpressureController.
func NewBackpressureController(
	config BackpressureConfig,
	getSize BufferSizeFunc,
	stateMachine *StateMachine,
	logger *slog.Logger,
) *BackpressureController {
	if logger == nil {
		logger = slog.Default()
	}

	return &BackpressureController{
		config:       config,
		getSize:      getSize,
		stateMachine: stateMachine,
		logger:       logger.With("component", "backpressure"),
	}
}

// Start begins monitoring buffer size.
// It runs until the context is cancelled.
func (c *BackpressureController) Start(ctx context.Context) {
	if !c.config.Enabled {
		c.logger.Info("backpressure controller disabled")
		return
	}

	ticker := time.NewTicker(c.config.CheckInterval)
	defer ticker.Stop()

	c.logger.Info("backpressure controller started",
		"high_watermark", c.config.HighWatermark,
		"low_watermark", c.config.LowWatermark,
		"check_interval", c.config.CheckInterval,
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("backpressure controller stopping")
			return
		case <-ticker.C:
			c.check(ctx)
		}
	}
}

// check performs a single buffer size check.
func (c *BackpressureController) check(ctx context.Context) {
	size, err := c.getSize(ctx)
	if err != nil {
		c.logger.Warn("failed to get buffer size", "error", err)
		return
	}

	c.mu.Lock()
	c.lastSize = size
	c.mu.Unlock()

	currentState := c.stateMachine.State()

	// Only act if we're in a state where we can pause/resume
	if currentState != StateRunning && currentState != StatePaused {
		return
	}

	if size >= c.config.HighWatermark && currentState == StateRunning {
		c.pause()
	} else if size <= c.config.LowWatermark && currentState == StatePaused {
		c.resume()
	}
}

// pause triggers a pause due to backpressure.
func (c *BackpressureController) pause() {
	if err := c.stateMachine.Transition(StatePaused); err != nil {
		c.logger.Warn("failed to transition to paused state", "error", err)
		return
	}

	c.mu.Lock()
	c.paused = true
	c.pausedAt = time.Now()
	c.pauseCount++
	c.mu.Unlock()

	c.logger.Warn("backpressure triggered, pausing pipeline",
		"buffer_size", c.lastSize,
		"high_watermark", c.config.HighWatermark,
	)
}

// resume resumes processing after backpressure clears.
func (c *BackpressureController) resume() {
	if err := c.stateMachine.Transition(StateRunning); err != nil {
		c.logger.Warn("failed to transition to running state", "error", err)
		return
	}

	c.mu.Lock()
	pauseDuration := time.Since(c.pausedAt)
	c.paused = false
	c.resumedAt = time.Now()
	c.resumeCount++
	c.mu.Unlock()

	c.logger.Info("backpressure cleared, resuming pipeline",
		"buffer_size", c.lastSize,
		"low_watermark", c.config.LowWatermark,
		"pause_duration", pauseDuration,
	)
}

// IsPaused returns whether the pipeline is currently paused due to backpressure.
func (c *BackpressureController) IsPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused
}

// SetStateMachine sets the state machine for the controller.
func (c *BackpressureController) SetStateMachine(sm *StateMachine) {
	c.stateMachine = sm
}

// Stats returns backpressure statistics.
func (c *BackpressureController) Stats() BackpressureStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return BackpressureStats{
		IsPaused:    c.paused,
		PausedAt:    c.pausedAt,
		ResumedAt:   c.resumedAt,
		PauseCount:  c.pauseCount,
		ResumeCount: c.resumeCount,
		LastSize:    c.lastSize,
	}
}

// BackpressureStats holds backpressure statistics.
type BackpressureStats struct {
	// IsPaused indicates if the pipeline is currently paused.
	IsPaused bool `json:"is_paused"`

	// PausedAt is when the pipeline was last paused.
	PausedAt time.Time `json:"paused_at,omitempty"`

	// ResumedAt is when the pipeline was last resumed.
	ResumedAt time.Time `json:"resumed_at,omitempty"`

	// PauseCount is the total number of pauses.
	PauseCount int64 `json:"pause_count"`

	// ResumeCount is the total number of resumes.
	ResumeCount int64 `json:"resume_count"`

	// LastSize is the last observed buffer size.
	LastSize int `json:"last_size"`
}
