package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewRegistry(t *testing.T) {
	// NewRegistry should create a new registry with all metrics
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}

	// Gather metrics to verify they're registered
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Should have Go runtime metrics plus our custom metrics
	if len(mfs) == 0 {
		t.Error("expected metrics to be registered, got none")
	}
}

func TestRegisterWith(t *testing.T) {
	// Create a new registry
	reg := prometheus.NewRegistry()

	// RegisterWith should not panic on first call
	RegisterWith(reg)

	// Verify we can gather from the registry (even if empty before metrics are written)
	_, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	// Verify the allMetrics slice has expected count
	expectedCount := 17 // Total number of metrics defined
	if len(allMetrics) != expectedCount {
		t.Errorf("expected %d metrics in allMetrics, got %d", expectedCount, len(allMetrics))
	}
}

func TestMetricLabels(t *testing.T) {
	// Test that metrics can be used with expected labels without panicking
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "CDCEventsTotal",
			fn: func() {
				CDCEventsTotal.WithLabelValues("source1", "public.users", "INSERT").Inc()
			},
		},
		{
			name: "CDCLagSeconds",
			fn: func() {
				CDCLagSeconds.WithLabelValues("source1", "public.users").Set(1.5)
			},
		},
		{
			name: "CDCErrorsTotal",
			fn: func() {
				CDCErrorsTotal.WithLabelValues("source1", "buffer").Inc()
			},
		},
		{
			name: "CDCRetriesTotal",
			fn: func() {
				CDCRetriesTotal.WithLabelValues("source1").Inc()
			},
		},
		{
			name: "CDCPipelineState",
			fn: func() {
				CDCPipelineState.WithLabelValues("source1").Set(2)
			},
		},
		{
			name: "APIRequestsTotal",
			fn: func() {
				APIRequestsTotal.WithLabelValues("/api/v1/sources", "GET", "200").Inc()
			},
		},
		{
			name: "APIRequestDuration",
			fn: func() {
				APIRequestDuration.WithLabelValues("/api/v1/sources", "GET").Observe(0.05)
			},
		},
		{
			name: "IcebergCommitsTotal",
			fn: func() {
				IcebergCommitsTotal.WithLabelValues("source1", "public.users").Inc()
			},
		},
		{
			name: "IcebergCommitDuration",
			fn: func() {
				IcebergCommitDuration.WithLabelValues("source1", "public.users").Observe(2.5)
			},
		},
		{
			name: "BufferDepth",
			fn: func() {
				BufferDepth.WithLabelValues("source1").Set(100)
			},
		},
		{
			name: "BufferBatchesTotal",
			fn: func() {
				BufferBatchesTotal.WithLabelValues("source1", "success").Inc()
			},
		},
		{
			name: "BufferEventsProcessedTotal",
			fn: func() {
				BufferEventsProcessedTotal.WithLabelValues("source1").Add(50)
			},
		},
		{
			name: "BufferDLQTotal",
			fn: func() {
				BufferDLQTotal.WithLabelValues("source1").Inc()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}

func TestLabelConstants(t *testing.T) {
	// Verify label constants are defined correctly
	labels := map[string]string{
		"source":     LabelSource,
		"table":      LabelTable,
		"operation":  LabelOperation,
		"endpoint":   LabelEndpoint,
		"method":     LabelMethod,
		"status":     LabelStatus,
		"error_type": LabelErrorType,
	}

	for expected, got := range labels {
		if got != expected {
			t.Errorf("label constant mismatch: expected %q, got %q", expected, got)
		}
	}
}

func TestNamespaceAndSubsystems(t *testing.T) {
	if Namespace != "philotes" {
		t.Errorf("expected namespace 'philotes', got %q", Namespace)
	}

	subsystems := map[string]string{
		"cdc":     SubsystemCDC,
		"api":     SubsystemAPI,
		"iceberg": SubsystemIceberg,
		"buffer":  SubsystemBuffer,
	}

	for expected, got := range subsystems {
		if got != expected {
			t.Errorf("subsystem constant mismatch: expected %q, got %q", expected, got)
		}
	}
}
