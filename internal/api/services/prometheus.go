// Package services provides business logic for API resources.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PrometheusClient provides methods to query Prometheus.
type PrometheusClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewPrometheusClient creates a new PrometheusClient.
func NewPrometheusClient(baseURL string, logger *slog.Logger) *PrometheusClient {
	return &PrometheusClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "prometheus-client"),
	}
}

// PrometheusResponse represents a response from the Prometheus API.
type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
	Error  string         `json:"error,omitempty"`
}

// PrometheusData represents the data portion of a Prometheus response.
type PrometheusData struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusResult `json:"result"`
}

// PrometheusResult represents a single result from Prometheus.
type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`  // For instant queries: [timestamp, value]
	Values [][]interface{}   `json:"values,omitempty"` // For range queries: [[timestamp, value], ...]
}

// QueryInstant performs an instant query against Prometheus.
func (c *PrometheusClient) QueryInstant(ctx context.Context, query string) ([]PrometheusResult, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus URL: %w", err)
	}

	u.Path = "/api/v1/query"
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("prometheus query failed", "query", query, "error", err)
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("prometheus returned non-200", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", promResp.Error)
	}

	return promResp.Data.Result, nil
}

// QueryRange performs a range query against Prometheus.
func (c *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]PrometheusResult, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus URL: %w", err)
	}

	u.Path = "/api/v1/query_range"
	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", step.String())
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("prometheus range query failed", "query", query, "error", err)
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("prometheus returned non-200", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", promResp.Error)
	}

	return promResp.Data.Result, nil
}

// GetScalarValue extracts a scalar float value from Prometheus instant query results.
// Returns 0 if no results or parsing fails.
func GetScalarValue(results []PrometheusResult) float64 {
	if len(results) == 0 {
		return 0
	}
	if len(results[0].Value) < 2 {
		return 0
	}

	valueStr, ok := results[0].Value[1].(string)
	if !ok {
		return 0
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

	return value
}

// GetScalarInt extracts a scalar int64 value from Prometheus instant query results.
// Returns 0 if no results or parsing fails.
func GetScalarInt(results []PrometheusResult) int64 {
	return int64(GetScalarValue(results))
}

// ParseTimeSeriesValues extracts time series data from range query results.
func ParseTimeSeriesValues(results []PrometheusResult) []TimeSeriesPoint {
	if len(results) == 0 {
		return nil
	}

	var points []TimeSeriesPoint
	for _, val := range results[0].Values {
		if len(val) < 2 {
			continue
		}

		// Parse timestamp
		tsFloat, ok := val[0].(float64)
		if !ok {
			continue
		}
		ts := time.Unix(int64(tsFloat), 0)

		// Parse value
		valueStr, ok := val[1].(string)
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		points = append(points, TimeSeriesPoint{
			Timestamp: ts,
			Value:     value,
		})
	}

	return points
}

// TimeSeriesPoint represents a single point in a time series.
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
}

// IsAvailable checks if Prometheus is reachable.
func (c *PrometheusClient) IsAvailable(ctx context.Context) bool {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return false
	}
	u.Path = "/-/healthy"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
