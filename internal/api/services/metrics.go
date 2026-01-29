// Package services provides business logic for API resources.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
)

// MetricsService provides business logic for metrics operations.
type MetricsService struct {
	promClient   *PrometheusClient
	pipelineRepo *repositories.PipelineRepository
	logger       *slog.Logger
}

// NewMetricsService creates a new MetricsService.
func NewMetricsService(
	promClient *PrometheusClient,
	pipelineRepo *repositories.PipelineRepository,
	logger *slog.Logger,
) *MetricsService {
	return &MetricsService{
		promClient:   promClient,
		pipelineRepo: pipelineRepo,
		logger:       logger.With("component", "metrics-service"),
	}
}

// GetPipelineMetrics retrieves current metrics for a pipeline.
func (s *MetricsService) GetPipelineMetrics(ctx context.Context, pipelineID uuid.UUID) (*models.PipelineMetrics, error) {
	// Get pipeline to verify it exists and get its name for metric labels
	pipeline, err := s.pipelineRepo.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, &NotFoundError{Resource: "pipeline", ID: pipelineID.String()}
	}

	metrics := &models.PipelineMetrics{
		PipelineID: pipelineID,
		Status:     pipeline.Status,
	}

	// Calculate uptime if running
	if pipeline.Status == models.PipelineStatusRunning && pipeline.StartedAt != nil {
		uptime := time.Since(*pipeline.StartedAt)
		metrics.Uptime = formatDurationMetrics(uptime)
	}

	// Use pipeline name as the source label in Prometheus
	sourceName := pipeline.Name

	// Query all metrics in parallel for better performance
	type metricResult struct {
		name  string
		value interface{}
		err   error
	}
	results := make(chan metricResult, 10)

	// Events total
	go func() {
		query := fmt.Sprintf(`sum(philotes_cdc_events_total{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"events_total", GetScalarInt(r), err}
	}()

	// Events per second (rate over 1 minute)
	go func() {
		query := fmt.Sprintf(`sum(rate(philotes_cdc_events_total{source="%s"}[1m]))`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"events_per_sec", GetScalarValue(r), err}
	}()

	// Current lag
	go func() {
		query := fmt.Sprintf(`max(philotes_cdc_lag_seconds{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"lag_seconds", GetScalarValue(r), err}
	}()

	// Buffer depth
	go func() {
		query := fmt.Sprintf(`sum(philotes_buffer_depth{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"buffer_depth", GetScalarInt(r), err}
	}()

	// Error count
	go func() {
		query := fmt.Sprintf(`sum(philotes_cdc_errors_total{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"error_count", GetScalarInt(r), err}
	}()

	// Iceberg commits
	go func() {
		query := fmt.Sprintf(`sum(philotes_iceberg_commits_total{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"iceberg_commits", GetScalarInt(r), err}
	}()

	// Iceberg bytes written
	go func() {
		query := fmt.Sprintf(`sum(philotes_iceberg_bytes_written_total{source="%s"})`, sourceName)
		r, err := s.promClient.QueryInstant(ctx, query)
		results <- metricResult{"iceberg_bytes", GetScalarInt(r), err}
	}()

	// Collect results
	for i := 0; i < 7; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-results:
			// Log errors but continue - missing metrics shouldn't fail the request
			if r.err != nil {
				s.logger.Debug("failed to query metric", "metric", r.name, "error", r.err)
			}
			switch r.name {
			case "events_total":
				if v, ok := r.value.(int64); ok {
					metrics.EventsProcessed = v
				}
			case "events_per_sec":
				if v, ok := r.value.(float64); ok {
					metrics.EventsPerSecond = v
				}
			case "lag_seconds":
				if v, ok := r.value.(float64); ok {
					metrics.LagSeconds = v
				}
			case "buffer_depth":
				if v, ok := r.value.(int64); ok {
					metrics.BufferDepth = v
				}
			case "error_count":
				if v, ok := r.value.(int64); ok {
					metrics.ErrorCount = v
				}
			case "iceberg_commits":
				if v, ok := r.value.(int64); ok {
					metrics.IcebergCommits = v
				}
			case "iceberg_bytes":
				if v, ok := r.value.(int64); ok {
					metrics.IcebergBytesWritten = v
				}
			}
		}
	}

	// Get per-table metrics
	tableMetrics, err := s.getTableMetrics(ctx, sourceName, pipeline.Tables)
	if err != nil {
		s.logger.Debug("failed to get table metrics", "error", err)
		// Continue without table metrics
	} else {
		metrics.Tables = tableMetrics
	}

	return metrics, nil
}

// getTableMetrics retrieves metrics for each table in the pipeline.
func (s *MetricsService) getTableMetrics(ctx context.Context, sourceName string, tables []models.TableMapping) ([]models.TableMetrics, error) {
	var result []models.TableMetrics

	for _, table := range tables {
		tableName := fmt.Sprintf("%s.%s", table.SourceSchema, table.SourceTable)

		tm := models.TableMetrics{
			Schema: table.SourceSchema,
			Table:  table.SourceTable,
		}

		// Events for this table
		query := fmt.Sprintf(`sum(philotes_cdc_events_total{source="%s",table="%s"})`, sourceName, tableName)
		r, err := s.promClient.QueryInstant(ctx, query)
		if err == nil {
			tm.EventsProcessed = GetScalarInt(r)
		}

		// Lag for this table
		query = fmt.Sprintf(`philotes_cdc_lag_seconds{source="%s",table="%s"}`, sourceName, tableName)
		r, err = s.promClient.QueryInstant(ctx, query)
		if err == nil {
			tm.LagSeconds = GetScalarValue(r)
		}

		result = append(result, tm)
	}

	return result, nil
}

// TimeRange represents a parsed time range.
type TimeRange struct {
	Start time.Time
	End   time.Time
	Step  time.Duration
}

// ParseTimeRange parses a time range string (e.g., "1h", "24h", "7d") into start/end times.
func ParseTimeRange(rangeStr string) (*TimeRange, error) {
	now := time.Now()
	var duration time.Duration

	switch rangeStr {
	case "15m":
		duration = 15 * time.Minute
	case "1h":
		duration = time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		// Try parsing as duration
		var err error
		duration, err = time.ParseDuration(rangeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid time range: %s", rangeStr)
		}
	}

	// Calculate step based on duration to get approximately 100 data points
	step := duration / 100
	if step < time.Second {
		step = time.Second
	}

	return &TimeRange{
		Start: now.Add(-duration),
		End:   now,
		Step:  step,
	}, nil
}

// GetPipelineMetricsHistory retrieves historical metrics for a pipeline.
func (s *MetricsService) GetPipelineMetricsHistory(ctx context.Context, pipelineID uuid.UUID, rangeStr string) (*models.MetricsHistory, error) {
	// Get pipeline to verify it exists
	pipeline, err := s.pipelineRepo.GetByID(ctx, pipelineID)
	if err != nil {
		return nil, &NotFoundError{Resource: "pipeline", ID: pipelineID.String()}
	}

	// Parse time range
	tr, err := ParseTimeRange(rangeStr)
	if err != nil {
		return nil, &ValidationError{Errors: []models.FieldError{
			{Field: "range", Message: err.Error()},
		}}
	}

	sourceName := pipeline.Name

	// Query historical data
	history := &models.MetricsHistory{
		PipelineID: pipelineID.String(),
		TimeRange:  rangeStr,
	}

	// Get events rate history
	eventsRateQuery := fmt.Sprintf(`sum(rate(philotes_cdc_events_total{source="%s"}[1m]))`, sourceName)
	eventsResults, err := s.promClient.QueryRange(ctx, eventsRateQuery, tr.Start, tr.End, tr.Step)
	if err != nil {
		s.logger.Debug("failed to query events rate history", "error", err)
	}
	eventsPoints := ParseTimeSeriesValues(eventsResults)

	// Get lag history
	lagQuery := fmt.Sprintf(`max(philotes_cdc_lag_seconds{source="%s"})`, sourceName)
	lagResults, err := s.promClient.QueryRange(ctx, lagQuery, tr.Start, tr.End, tr.Step)
	if err != nil {
		s.logger.Debug("failed to query lag history", "error", err)
	}
	lagPoints := ParseTimeSeriesValues(lagResults)

	// Get buffer depth history
	bufferQuery := fmt.Sprintf(`sum(philotes_buffer_depth{source="%s"})`, sourceName)
	bufferResults, err := s.promClient.QueryRange(ctx, bufferQuery, tr.Start, tr.End, tr.Step)
	if err != nil {
		s.logger.Debug("failed to query buffer depth history", "error", err)
	}
	bufferPoints := ParseTimeSeriesValues(bufferResults)

	// Get error count history
	errorQuery := fmt.Sprintf(`sum(philotes_cdc_errors_total{source="%s"})`, sourceName)
	errorResults, err := s.promClient.QueryRange(ctx, errorQuery, tr.Start, tr.End, tr.Step)
	if err != nil {
		s.logger.Debug("failed to query error history", "error", err)
	}
	errorPoints := ParseTimeSeriesValues(errorResults)

	// Merge all time series data
	// Use events rate timestamps as the base since it's the most important metric
	dataPointsMap := make(map[int64]*models.MetricsDataPoint)

	for _, p := range eventsPoints {
		ts := p.Timestamp.Unix()
		dataPointsMap[ts] = &models.MetricsDataPoint{
			Timestamp:       p.Timestamp,
			EventsPerSecond: p.Value,
		}
	}

	for _, p := range lagPoints {
		ts := p.Timestamp.Unix()
		if dp, ok := dataPointsMap[ts]; ok {
			dp.LagSeconds = p.Value
		} else {
			dataPointsMap[ts] = &models.MetricsDataPoint{
				Timestamp:  p.Timestamp,
				LagSeconds: p.Value,
			}
		}
	}

	for _, p := range bufferPoints {
		ts := p.Timestamp.Unix()
		if dp, ok := dataPointsMap[ts]; ok {
			dp.BufferDepth = int64(p.Value)
		} else {
			dataPointsMap[ts] = &models.MetricsDataPoint{
				Timestamp:   p.Timestamp,
				BufferDepth: int64(p.Value),
			}
		}
	}

	for _, p := range errorPoints {
		ts := p.Timestamp.Unix()
		if dp, ok := dataPointsMap[ts]; ok {
			dp.ErrorCount = int64(p.Value)
		} else {
			dataPointsMap[ts] = &models.MetricsDataPoint{
				Timestamp:  p.Timestamp,
				ErrorCount: int64(p.Value),
			}
		}
	}

	// Convert map to sorted slice
	var dataPoints []models.MetricsDataPoint
	for _, dp := range dataPointsMap {
		dataPoints = append(dataPoints, *dp)
	}

	// Sort by timestamp
	sortDataPoints(dataPoints)

	history.DataPoints = dataPoints
	return history, nil
}

// sortDataPoints sorts data points by timestamp.
func sortDataPoints(points []models.MetricsDataPoint) {
	for i := 0; i < len(points)-1; i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j].Timestamp.Before(points[i].Timestamp) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
}

// formatDurationMetrics formats a duration as a human-readable string.
// Uses a different name to avoid conflict with pipeline.go's formatDuration.
func formatDurationMetrics(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	sec := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, sec)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}
