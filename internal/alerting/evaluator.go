// Package alerting provides the alerting framework for Philotes.
package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Evaluator queries Prometheus and evaluates alert rules.
type Evaluator struct {
	prometheusURL string
	httpClient    *http.Client
	logger        *slog.Logger
}

// MetricValue represents a single metric value from Prometheus.
type MetricValue struct {
	Labels map[string]string
	Value  float64
	Time   time.Time
}

// prometheusResponse represents the response from Prometheus query API.
type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []prometheusResult `json:"result"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// prometheusResult represents a single result from Prometheus query.
type prometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// NewEvaluator creates a new alert evaluator.
func NewEvaluator(prometheusURL string, logger *slog.Logger) *Evaluator {
	if logger == nil {
		logger = slog.Default()
	}

	return &Evaluator{
		prometheusURL: strings.TrimSuffix(prometheusURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "alert-evaluator"),
	}
}

// Evaluate queries Prometheus and checks if the rule condition is met.
// Returns one EvaluationResult per metric series returned by Prometheus.
func (e *Evaluator) Evaluate(ctx context.Context, rule AlertRule) ([]EvaluationResult, error) {
	metrics, err := e.queryPrometheus(ctx, rule.MetricName, rule.Labels)
	if err != nil {
		return nil, fmt.Errorf("failed to query prometheus: %w", err)
	}

	if len(metrics) == 0 {
		e.logger.Debug("no metrics found for rule",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"metric_name", rule.MetricName,
		)
		return []EvaluationResult{}, nil
	}

	results := make([]EvaluationResult, 0, len(metrics))
	now := time.Now()

	for _, m := range metrics {
		shouldFire := rule.Operator.Evaluate(m.Value, rule.Threshold)

		// Merge rule labels with metric labels (metric labels take precedence for specificity)
		mergedLabels := make(map[string]string)
		for k, v := range rule.Labels {
			mergedLabels[k] = v
		}
		for k, v := range m.Labels {
			mergedLabels[k] = v
		}

		result := EvaluationResult{
			Rule:        &rule,
			Value:       m.Value,
			Labels:      mergedLabels,
			ShouldFire:  shouldFire,
			EvaluatedAt: now,
		}

		results = append(results, result)

		e.logger.Debug("evaluated rule",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"value", m.Value,
			"threshold", rule.Threshold,
			"operator", rule.Operator,
			"should_fire", shouldFire,
		)
	}

	return results, nil
}

// queryPrometheus queries the Prometheus HTTP API.
func (e *Evaluator) queryPrometheus(ctx context.Context, metricName string, labels map[string]string) ([]MetricValue, error) {
	// Build the PromQL query
	query := e.buildQuery(metricName, labels)

	// Construct the URL
	queryURL := fmt.Sprintf("%s/api/v1/query", e.prometheusURL)
	reqURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus URL: %w", err)
	}

	params := url.Values{}
	params.Set("query", query)
	reqURL.RawQuery = params.Encode()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	e.logger.Debug("querying prometheus",
		"url", reqURL.String(),
		"query", query,
	)

	// Execute the request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		e.logger.Error("prometheus returned non-OK status",
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var promResp prometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to parse prometheus response: %w", err)
	}

	// Check for Prometheus API errors
	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s - %s", promResp.ErrorType, promResp.Error)
	}

	// Convert results to MetricValue slice
	metrics := make([]MetricValue, 0, len(promResp.Data.Result))
	for _, result := range promResp.Data.Result {
		mv, err := e.parseResult(result)
		if err != nil {
			e.logger.Warn("failed to parse prometheus result",
				"error", err,
				"result", result,
			)
			continue
		}
		metrics = append(metrics, mv)
	}

	return metrics, nil
}

// buildQuery constructs a PromQL query from metric name and labels.
func (e *Evaluator) buildQuery(metricName string, labels map[string]string) string {
	if len(labels) == 0 {
		return metricName
	}

	// Build label selectors
	selectors := make([]string, 0, len(labels))
	for k, v := range labels {
		// Escape double quotes in label values
		escapedValue := strings.ReplaceAll(v, `"`, `\"`)
		selectors = append(selectors, fmt.Sprintf(`%s="%s"`, k, escapedValue))
	}

	return fmt.Sprintf("%s{%s}", metricName, strings.Join(selectors, ","))
}

// parseResult parses a single Prometheus result into a MetricValue.
func (e *Evaluator) parseResult(result prometheusResult) (MetricValue, error) {
	mv := MetricValue{
		Labels: result.Metric,
	}

	// Value is an array: [timestamp, "value"]
	if len(result.Value) != 2 {
		return mv, fmt.Errorf("unexpected value format: expected [timestamp, value], got %v", result.Value)
	}

	// Parse timestamp (Unix seconds as float64)
	timestamp, ok := result.Value[0].(float64)
	if !ok {
		return mv, fmt.Errorf("failed to parse timestamp: %v", result.Value[0])
	}
	mv.Time = time.Unix(int64(timestamp), 0)

	// Parse value (string representation of float)
	valueStr, ok := result.Value[1].(string)
	if !ok {
		return mv, fmt.Errorf("failed to parse value as string: %v", result.Value[1])
	}

	var value float64
	if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
		return mv, fmt.Errorf("failed to parse value: %w", err)
	}
	mv.Value = value

	return mv, nil
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (e *Evaluator) SetHTTPClient(client *http.Client) {
	e.httpClient = client
}
