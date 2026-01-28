// Package scaling provides the auto-scaling engine for Philotes.
package scaling

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

// Evaluator queries Prometheus and evaluates scaling rules.
type Evaluator struct {
	prometheusURL string
	httpClient    *http.Client
	logger        *slog.Logger
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

// NewEvaluator creates a new scaling evaluator.
func NewEvaluator(prometheusURL string, logger *slog.Logger) *Evaluator {
	if logger == nil {
		logger = slog.Default()
	}

	return &Evaluator{
		prometheusURL: strings.TrimSuffix(prometheusURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With("component", "scaling-evaluator"),
	}
}

// EvaluateRule queries Prometheus and checks if the rule condition is met.
func (e *Evaluator) EvaluateRule(ctx context.Context, rule *ScalingRule) (*EvaluationResult, error) {
	value, err := e.queryMetric(ctx, rule.Metric)
	if err != nil {
		return nil, fmt.Errorf("failed to query metric: %w", err)
	}

	triggered := rule.Evaluate(value)

	result := &EvaluationResult{
		Rule:        rule,
		MetricValue: value,
		Triggered:   triggered,
	}

	e.logger.Debug("evaluated rule",
		"rule_id", rule.ID,
		"metric", rule.Metric,
		"value", value,
		"threshold", rule.Threshold,
		"operator", rule.Operator,
		"triggered", triggered,
	)

	return result, nil
}

// EvaluatePolicy evaluates all rules for a policy and returns a scaling decision.
func (e *Evaluator) EvaluatePolicy(ctx context.Context, policy *ScalingPolicy, state *ScalingState) (*ScalingDecision, error) {
	decision := &ScalingDecision{
		Policy:          policy,
		CurrentReplicas: state.CurrentReplicas,
		DesiredReplicas: state.CurrentReplicas,
		ShouldExecute:   false,
	}

	// Check cooldown period
	if state.IsInCooldown(policy.CooldownDuration()) {
		cooldownRemaining := policy.CooldownDuration() - time.Since(*state.LastScaleTime)
		decision.CooldownRemaining = cooldownRemaining
		decision.Reason = fmt.Sprintf("in cooldown (%.0fs remaining)", cooldownRemaining.Seconds())
		e.logger.Debug("policy in cooldown",
			"policy_id", policy.ID,
			"cooldown_remaining", cooldownRemaining,
		)
		return decision, nil
	}

	// Evaluate scale-up rules (any rule triggering will scale up)
	for i := range policy.ScaleUpRules {
		rule := &policy.ScaleUpRules[i]
		result, err := e.EvaluateRule(ctx, rule)
		if err != nil {
			e.logger.Warn("failed to evaluate scale-up rule",
				"rule_id", rule.ID,
				"error", err,
			)
			continue
		}

		if result.Triggered {
			// Check if condition has been true for required duration
			conditionMet, duration := e.checkDurationCondition(state, rule.ID.String(), rule.Duration())
			if conditionMet {
				newReplicas := policy.ClampReplicas(state.CurrentReplicas + rule.ScaleBy)
				if newReplicas > state.CurrentReplicas {
					decision.Action = ActionScaleUp
					decision.DesiredReplicas = newReplicas
					decision.ShouldExecute = true
					decision.Reason = fmt.Sprintf("rule triggered: %s %s %.2f (actual: %.2f) for %s",
						rule.Metric, rule.Operator.String(), rule.Threshold, result.MetricValue, duration)
					decision.TriggeredBy = fmt.Sprintf("rule:%s", rule.ID)
					return decision, nil
				}
			} else {
				// Update pending condition
				e.updatePendingCondition(state, rule.ID.String())
			}
		} else {
			// Clear pending condition
			e.clearPendingCondition(state, rule.ID.String())
		}
	}

	// Evaluate scale-down rules (any rule triggering will scale down)
	for i := range policy.ScaleDownRules {
		rule := &policy.ScaleDownRules[i]
		result, err := e.EvaluateRule(ctx, rule)
		if err != nil {
			e.logger.Warn("failed to evaluate scale-down rule",
				"rule_id", rule.ID,
				"error", err,
			)
			continue
		}

		if result.Triggered {
			// Check if condition has been true for required duration
			conditionMet, duration := e.checkDurationCondition(state, rule.ID.String(), rule.Duration())
			if conditionMet {
				newReplicas := policy.ClampReplicas(state.CurrentReplicas + rule.ScaleBy)
				// Check for scale-to-zero
				if newReplicas < state.CurrentReplicas && (newReplicas > 0 || policy.ScaleToZero) {
					decision.Action = ActionScaleDown
					decision.DesiredReplicas = newReplicas
					decision.ShouldExecute = true
					decision.Reason = fmt.Sprintf("rule triggered: %s %s %.2f (actual: %.2f) for %s",
						rule.Metric, rule.Operator.String(), rule.Threshold, result.MetricValue, duration)
					decision.TriggeredBy = fmt.Sprintf("rule:%s", rule.ID)
					return decision, nil
				}
			} else {
				// Update pending condition
				e.updatePendingCondition(state, rule.ID.String())
			}
		} else {
			// Clear pending condition
			e.clearPendingCondition(state, rule.ID.String())
		}
	}

	decision.Reason = "no scaling rules triggered"
	return decision, nil
}

// checkDurationCondition checks if a condition has been true for the required duration.
func (e *Evaluator) checkDurationCondition(state *ScalingState, conditionKey string, requiredDuration time.Duration) (bool, time.Duration) {
	if requiredDuration == 0 {
		return true, 0
	}

	if state.PendingConditions == nil {
		return false, 0
	}

	firstTriggered, exists := state.PendingConditions[conditionKey]
	if !exists {
		return false, 0
	}

	elapsed := time.Since(firstTriggered)
	return elapsed >= requiredDuration, elapsed
}

// updatePendingCondition updates the pending condition tracking.
func (e *Evaluator) updatePendingCondition(state *ScalingState, conditionKey string) {
	if state.PendingConditions == nil {
		state.PendingConditions = make(map[string]time.Time)
	}

	if _, exists := state.PendingConditions[conditionKey]; !exists {
		state.PendingConditions[conditionKey] = time.Now()
	}
}

// clearPendingCondition clears a pending condition.
func (e *Evaluator) clearPendingCondition(state *ScalingState, conditionKey string) {
	if state.PendingConditions != nil {
		delete(state.PendingConditions, conditionKey)
	}
}

// queryMetric queries Prometheus for a metric value.
func (e *Evaluator) queryMetric(ctx context.Context, metric string) (float64, error) {
	queryURL := fmt.Sprintf("%s/api/v1/query", e.prometheusURL)
	reqURL, err := url.Parse(queryURL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse prometheus URL: %w", err)
	}

	params := url.Values{}
	params.Set("query", metric)
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	e.logger.Debug("querying prometheus",
		"url", reqURL.String(),
		"query", metric,
	)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		e.logger.Error("prometheus returned non-OK status",
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return 0, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var promResp prometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return 0, fmt.Errorf("failed to parse prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return 0, fmt.Errorf("prometheus query failed: %s - %s", promResp.ErrorType, promResp.Error)
	}

	if len(promResp.Data.Result) == 0 {
		return 0, fmt.Errorf("no data returned for metric: %s", metric)
	}

	// Parse the first result value
	result := promResp.Data.Result[0]
	if len(result.Value) != 2 {
		return 0, fmt.Errorf("unexpected value format: expected [timestamp, value], got %v", result.Value)
	}

	valueStr, ok := result.Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("failed to parse value as string: %v", result.Value[1])
	}

	var value float64
	if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
		return 0, fmt.Errorf("failed to parse value: %w", err)
	}

	return value, nil
}

// QueryMetric is a public method to query a metric value (useful for testing).
func (e *Evaluator) QueryMetric(ctx context.Context, metric string) (float64, error) {
	return e.queryMetric(ctx, metric)
}

// SetHTTPClient allows setting a custom HTTP client (useful for testing).
func (e *Evaluator) SetHTTPClient(client *http.Client) {
	e.httpClient = client
}
