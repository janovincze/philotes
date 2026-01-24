package buffer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/janovincze/philotes/internal/cdc"
)

// mockManager implements Manager for testing.
type mockManager struct {
	mu             sync.Mutex
	writtenEvents  []cdc.Event
	readBatchCalls int
	processedIDs   []int64
	cleanupCalls   int
	eventsToReturn []BufferedEvent
	statsFn        func() Stats
}

func newMockManager() *mockManager {
	return &mockManager{}
}

func (m *mockManager) Write(ctx context.Context, events []cdc.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenEvents = append(m.writtenEvents, events...)
	return nil
}

func (m *mockManager) ReadBatch(ctx context.Context, sourceID string, limit int) ([]BufferedEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBatchCalls++

	if len(m.eventsToReturn) > 0 {
		events := m.eventsToReturn
		m.eventsToReturn = nil // Return events only once
		return events, nil
	}
	return nil, nil
}

func (m *mockManager) MarkProcessed(ctx context.Context, eventIDs []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedIDs = append(m.processedIDs, eventIDs...)
	return nil
}

func (m *mockManager) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupCalls++
	return 0, nil
}

func (m *mockManager) Stats(ctx context.Context) (Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statsFn != nil {
		return m.statsFn(), nil
	}
	return Stats{}, nil
}

func (m *mockManager) Close() error {
	return nil
}

func (m *mockManager) getWrittenEvents() []cdc.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]cdc.Event, len(m.writtenEvents))
	copy(events, m.writtenEvents)
	return events
}

func (m *mockManager) getProcessedIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]int64, len(m.processedIDs))
	copy(ids, m.processedIDs)
	return ids
}

func (m *mockManager) setEventsToReturn(events []BufferedEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventsToReturn = events
}

func TestNewBatchProcessor(t *testing.T) {
	manager := newMockManager()
	handler := func(ctx context.Context, events []BufferedEvent) error {
		return nil
	}
	cfg := DefaultBatchConfig()
	cfg.SourceID = "test-source"

	processor := NewBatchProcessor(manager, handler, cfg, nil)

	if processor == nil {
		t.Fatal("Expected processor to not be nil")
	}

	if processor.IsRunning() {
		t.Error("Expected processor to not be running initially")
	}
}

func TestBatchProcessorStartStop(t *testing.T) {
	manager := newMockManager()
	handler := func(ctx context.Context, events []BufferedEvent) error {
		return nil
	}
	cfg := DefaultBatchConfig()
	cfg.SourceID = "test-source"
	cfg.FlushInterval = 10 * time.Millisecond

	processor := NewBatchProcessor(manager, handler, cfg, nil)

	ctx := context.Background()

	// Start the processor
	err := processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	if !processor.IsRunning() {
		t.Error("Expected processor to be running after Start")
	}

	// Starting again should be a no-op
	err = processor.Start(ctx)
	if err != nil {
		t.Fatalf("Second start should not error: %v", err)
	}

	// Give it time to run at least one iteration
	time.Sleep(50 * time.Millisecond)

	// Stop the processor
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = processor.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}

	if processor.IsRunning() {
		t.Error("Expected processor to not be running after Stop")
	}

	// Stopping again should be a no-op
	err = processor.Stop(stopCtx)
	if err != nil {
		t.Fatalf("Second stop should not error: %v", err)
	}
}

func TestBatchProcessorProcessesEvents(t *testing.T) {
	manager := newMockManager()

	// Set up events to return
	testEvents := []BufferedEvent{
		{ID: 1, Event: cdc.Event{LSN: "0/1"}},
		{ID: 2, Event: cdc.Event{LSN: "0/2"}},
	}
	manager.setEventsToReturn(testEvents)

	var processedEvents []BufferedEvent
	var mu sync.Mutex
	handler := func(ctx context.Context, events []BufferedEvent) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents = append(processedEvents, events...)
		return nil
	}

	cfg := DefaultBatchConfig()
	cfg.SourceID = "test-source"
	cfg.FlushInterval = 10 * time.Millisecond
	cfg.CleanupInterval = 0 // Disable cleanup

	processor := NewBatchProcessor(manager, handler, cfg, nil)

	ctx := context.Background()
	err := processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	processor.Stop(stopCtx)

	// Check that events were processed
	mu.Lock()
	defer mu.Unlock()

	if len(processedEvents) != 2 {
		t.Errorf("Expected 2 processed events, got %d", len(processedEvents))
	}

	// Check that events were marked as processed
	processedIDs := manager.getProcessedIDs()
	if len(processedIDs) != 2 {
		t.Errorf("Expected 2 processed IDs, got %d", len(processedIDs))
	}
}

func TestBatchProcessorContextCancellation(t *testing.T) {
	manager := newMockManager()
	handler := func(ctx context.Context, events []BufferedEvent) error {
		return nil
	}

	cfg := DefaultBatchConfig()
	cfg.SourceID = "test-source"
	cfg.FlushInterval = time.Hour // Long interval

	processor := NewBatchProcessor(manager, handler, cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())

	err := processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Cancel the context
	cancel()

	// Wait briefly for shutdown
	time.Sleep(50 * time.Millisecond)

	// The processor should have stopped due to context cancellation
	// (though IsRunning may still be true until Stop is called explicitly)
}
