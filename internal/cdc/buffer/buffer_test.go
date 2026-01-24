package buffer

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true by default")
	}

	if cfg.MaxOpenConns != 10 {
		t.Errorf("Expected MaxOpenConns to be 10, got %d", cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns to be 5, got %d", cfg.MaxIdleConns)
	}

	expectedRetention := 168 * time.Hour
	if cfg.Retention != expectedRetention {
		t.Errorf("Expected Retention to be %v, got %v", expectedRetention, cfg.Retention)
	}

	expectedCleanupInterval := time.Hour
	if cfg.CleanupInterval != expectedCleanupInterval {
		t.Errorf("Expected CleanupInterval to be %v, got %v", expectedCleanupInterval, cfg.CleanupInterval)
	}
}

func TestDefaultBatchConfig(t *testing.T) {
	cfg := DefaultBatchConfig()

	if cfg.BatchSize != 1000 {
		t.Errorf("Expected BatchSize to be 1000, got %d", cfg.BatchSize)
	}

	expectedFlushInterval := 5 * time.Second
	if cfg.FlushInterval != expectedFlushInterval {
		t.Errorf("Expected FlushInterval to be %v, got %v", expectedFlushInterval, cfg.FlushInterval)
	}

	expectedRetention := 168 * time.Hour
	if cfg.Retention != expectedRetention {
		t.Errorf("Expected Retention to be %v, got %v", expectedRetention, cfg.Retention)
	}

	expectedCleanupInterval := time.Hour
	if cfg.CleanupInterval != expectedCleanupInterval {
		t.Errorf("Expected CleanupInterval to be %v, got %v", expectedCleanupInterval, cfg.CleanupInterval)
	}
}

func TestBufferedEvent(t *testing.T) {
	now := time.Now()
	be := BufferedEvent{
		ID:        123,
		CreatedAt: now,
	}

	if be.ID != 123 {
		t.Errorf("Expected ID to be 123, got %d", be.ID)
	}

	if !be.CreatedAt.Equal(now) {
		t.Errorf("Expected CreatedAt to be %v, got %v", now, be.CreatedAt)
	}

	if be.ProcessedAt != nil {
		t.Error("Expected ProcessedAt to be nil")
	}
}

func TestStats(t *testing.T) {
	now := time.Now()
	lag := 5 * time.Minute

	stats := Stats{
		TotalEvents:       100,
		UnprocessedEvents: 25,
		OldestUnprocessed: &now,
		Lag:               lag,
	}

	if stats.TotalEvents != 100 {
		t.Errorf("Expected TotalEvents to be 100, got %d", stats.TotalEvents)
	}

	if stats.UnprocessedEvents != 25 {
		t.Errorf("Expected UnprocessedEvents to be 25, got %d", stats.UnprocessedEvents)
	}

	if stats.OldestUnprocessed == nil {
		t.Error("Expected OldestUnprocessed to not be nil")
	} else if !stats.OldestUnprocessed.Equal(now) {
		t.Errorf("Expected OldestUnprocessed to be %v, got %v", now, *stats.OldestUnprocessed)
	}

	if stats.Lag != lag {
		t.Errorf("Expected Lag to be %v, got %v", lag, stats.Lag)
	}
}
