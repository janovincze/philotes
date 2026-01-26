package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/janovincze/philotes/internal/metrics"
)

// RetryPolicy defines the retry behavior.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of attempts (including the first try).
	MaxAttempts int

	// InitialInterval is the initial backoff interval.
	InitialInterval time.Duration

	// MaxInterval is the maximum backoff interval.
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier.
	Multiplier float64

	// Jitter adds randomness to prevent thundering herd.
	Jitter bool
}

// DefaultRetryPolicy returns a RetryPolicy with sensible defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
	}
}

// RetryError wraps an error with retry information.
type RetryError struct {
	Err      error
	Attempts int
	LastWait time.Duration
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("failed after %d attempts: %v", e.Attempts, e.Err)
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

// Retryable marks an error as retryable.
type Retryable interface {
	IsRetryable() bool
}

// RetryableError wraps an error and marks it as retryable.
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func (e *RetryableError) IsRetryable() bool {
	return e.Retryable
}

// NewRetryableError wraps an error as retryable.
func NewRetryableError(err error) error {
	return &RetryableError{Err: err, Retryable: true}
}

// NewNonRetryableError wraps an error as non-retryable.
func NewNonRetryableError(err error) error {
	return &RetryableError{Err: err, Retryable: false}
}

// Retryer executes operations with retry logic.
type Retryer struct {
	policy     RetryPolicy
	logger     *slog.Logger
	sourceName string
}

// NewRetryer creates a new Retryer with the given policy.
func NewRetryer(policy RetryPolicy, logger *slog.Logger) *Retryer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Retryer{
		policy: policy,
		logger: logger.With("component", "retryer"),
	}
}

// SetSourceName sets the source name for metric labels.
func (r *Retryer) SetSourceName(name string) {
	r.sourceName = name
}

// Execute runs the operation with retry logic.
// Returns the first successful result or the last error after all retries.
func (r *Retryer) Execute(ctx context.Context, operation func(ctx context.Context) error) error {
	var lastErr error
	var lastWait time.Duration

	for attempt := 1; attempt <= r.policy.MaxAttempts; attempt++ {
		err := operation(ctx)
		if err == nil {
			if attempt > 1 {
				r.logger.Debug("operation succeeded after retry",
					"attempt", attempt,
					"total_wait", lastWait,
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryable(err) {
			r.logger.Debug("non-retryable error",
				"attempt", attempt,
				"error", err,
			)
			return &RetryError{
				Err:      err,
				Attempts: attempt,
				LastWait: lastWait,
			}
		}

		// Check if this was the last attempt
		if attempt >= r.policy.MaxAttempts {
			break
		}

		// Record retry metric
		if r.sourceName != "" {
			metrics.CDCRetriesTotal.WithLabelValues(r.sourceName).Inc()
		}

		// Calculate backoff
		wait := r.calculateBackoff(attempt)
		lastWait += wait

		r.logger.Debug("retrying after error",
			"attempt", attempt,
			"next_attempt", attempt+1,
			"wait", wait,
			"error", err,
		)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return &RetryError{
				Err:      ctx.Err(),
				Attempts: attempt,
				LastWait: lastWait,
			}
		case <-time.After(wait):
		}
	}

	return &RetryError{
		Err:      lastErr,
		Attempts: r.policy.MaxAttempts,
		LastWait: lastWait,
	}
}

// isRetryable determines if an error should be retried.
func (r *Retryer) isRetryable(err error) bool {
	// Check if the error implements Retryable interface
	var retryable Retryable
	if errors.As(err, &retryable) {
		return retryable.IsRetryable()
	}

	// By default, assume errors are retryable unless they're context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return true
}

// calculateBackoff calculates the backoff duration for the given attempt.
func (r *Retryer) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initialInterval * multiplier^(attempt-1)
	backoff := float64(r.policy.InitialInterval) * math.Pow(r.policy.Multiplier, float64(attempt-1))

	// Cap at max interval
	if backoff > float64(r.policy.MaxInterval) {
		backoff = float64(r.policy.MaxInterval)
	}

	duration := time.Duration(backoff)

	// Add jitter if enabled (±25%)
	if r.policy.Jitter {
		jitter := duration / 4
		duration = duration - jitter + time.Duration(rand.Int64N(int64(jitter*2)))
	}

	return duration
}

// ExecuteWithResult runs an operation that returns a value with retry logic.
func ExecuteWithResult[T any](ctx context.Context, r *Retryer, operation func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	err := r.Execute(ctx, func(ctx context.Context) error {
		var opErr error
		result, opErr = operation(ctx)
		if opErr != nil {
			lastErr = opErr
			return opErr
		}
		return nil
	})

	if err != nil {
		return result, err
	}
	if lastErr != nil {
		return result, lastErr
	}
	return result, nil
}
