package pipeline

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetryer_Success(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)
	callCount := 0

	err := retryer.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryer_EventualSuccess(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)
	callCount := 0

	err := retryer.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestRetryer_MaxAttemptsExceeded(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)
	callCount := 0

	err := retryer.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Errorf("expected RetryError, got %T", err)
	}
	if retryErr.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", retryErr.Attempts)
	}
}

func TestRetryer_NonRetryableError(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)
	callCount := 0

	err := retryer.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		return NewNonRetryableError(errors.New("permanent error"))
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestRetryer_ContextCancelled(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryer.Execute(ctx, func(ctx context.Context) error {
		callCount++
		return errors.New("temporary error")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Errorf("expected RetryError, got %T", err)
	}
	if !errors.Is(retryErr.Err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", retryErr.Err)
	}
}

func TestRetryer_CalculateBackoff(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Jitter:          false,
	}

	retryer := NewRetryer(policy, nil)

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1 * time.Second}, // capped at MaxInterval
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := retryer.calculateBackoff(tt.attempt)
			if got != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts = 3, got %d", policy.MaxAttempts)
	}
	if policy.InitialInterval != time.Second {
		t.Errorf("expected InitialInterval = 1s, got %v", policy.InitialInterval)
	}
	if policy.MaxInterval != 30*time.Second {
		t.Errorf("expected MaxInterval = 30s, got %v", policy.MaxInterval)
	}
	if policy.Multiplier != 2.0 {
		t.Errorf("expected Multiplier = 2.0, got %f", policy.Multiplier)
	}
	if !policy.Jitter {
		t.Error("expected Jitter = true")
	}
}

func TestRetryableError(t *testing.T) {
	originalErr := errors.New("original error")

	retryable := NewRetryableError(originalErr)
	var r Retryable
	if !errors.As(retryable, &r) {
		t.Error("expected error to implement Retryable")
	}
	if !r.IsRetryable() {
		t.Error("expected IsRetryable() = true")
	}

	nonRetryable := NewNonRetryableError(originalErr)
	if !errors.As(nonRetryable, &r) {
		t.Error("expected error to implement Retryable")
	}
	if r.IsRetryable() {
		t.Error("expected IsRetryable() = false")
	}
}

func TestRetryError(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := &RetryError{
		Err:      originalErr,
		Attempts: 3,
		LastWait: 500 * time.Millisecond,
	}

	expectedMsg := "failed after 3 attempts: original error"
	if retryErr.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, retryErr.Error())
	}

	if !errors.Is(retryErr, originalErr) {
		t.Error("expected Unwrap to return original error")
	}
}
