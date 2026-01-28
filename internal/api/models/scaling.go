// Package models provides API request and response types.
package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling"
)

// ScalingRuleRequest represents a scaling rule in API requests.
type ScalingRuleRequest struct {
	Metric          string           `json:"metric" binding:"required"`
	Operator        scaling.Operator `json:"operator" binding:"required"`
	Threshold       float64          `json:"threshold"`
	DurationSeconds int              `json:"duration_seconds,omitempty"`
	ScaleBy         int              `json:"scale_by" binding:"required"`
}

// ScalingScheduleRequest represents a scaling schedule in API requests.
type ScalingScheduleRequest struct {
	CronExpression  string `json:"cron_expression" binding:"required"`
	DesiredReplicas int    `json:"desired_replicas" binding:"required"`
	Timezone        string `json:"timezone,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

// CreateScalingPolicyRequest represents a request to create a scaling policy.
type CreateScalingPolicyRequest struct {
	Name            string                   `json:"name" binding:"required,min=1,max=255"`
	TargetType      scaling.TargetType       `json:"target_type" binding:"required"`
	TargetID        *uuid.UUID               `json:"target_id,omitempty"`
	MinReplicas     int                      `json:"min_replicas" binding:"required"`
	MaxReplicas     int                      `json:"max_replicas" binding:"required"`
	CooldownSeconds *int                     `json:"cooldown_seconds,omitempty"`
	MaxHourlyCost   *float64                 `json:"max_hourly_cost,omitempty"`
	ScaleToZero     *bool                    `json:"scale_to_zero,omitempty"`
	Enabled         *bool                    `json:"enabled,omitempty"`
	ScaleUpRules    []ScalingRuleRequest     `json:"scale_up_rules,omitempty"`
	ScaleDownRules  []ScalingRuleRequest     `json:"scale_down_rules,omitempty"`
	Schedules       []ScalingScheduleRequest `json:"schedules,omitempty"`
}

// Validate validates the create scaling policy request.
func (r *CreateScalingPolicyRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}
	if !r.TargetType.IsValid() {
		errors = append(errors, FieldError{Field: "target_type", Message: "target_type must be one of: cdc-worker, trino, risingwave, nodes"})
	}
	if r.MinReplicas < 0 {
		errors = append(errors, FieldError{Field: "min_replicas", Message: "min_replicas must be >= 0"})
	}
	if r.MaxReplicas < 1 {
		errors = append(errors, FieldError{Field: "max_replicas", Message: "max_replicas must be >= 1"})
	}
	if r.MinReplicas > r.MaxReplicas {
		errors = append(errors, FieldError{Field: "min_replicas", Message: "min_replicas cannot be greater than max_replicas"})
	}
	if r.CooldownSeconds != nil && *r.CooldownSeconds < 0 {
		errors = append(errors, FieldError{Field: "cooldown_seconds", Message: "cooldown_seconds must be >= 0"})
	}
	if r.MaxHourlyCost != nil && *r.MaxHourlyCost < 0 {
		errors = append(errors, FieldError{Field: "max_hourly_cost", Message: "max_hourly_cost must be >= 0"})
	}

	// Validate scale-up rules
	for i, rule := range r.ScaleUpRules {
		if rule.Metric == "" {
			errors = append(errors, FieldError{Field: "scale_up_rules[" + fmt.Sprintf("%d", i) + "].metric", Message: "metric is required"})
		}
		if !rule.Operator.IsValid() {
			errors = append(errors, FieldError{Field: "scale_up_rules[" + fmt.Sprintf("%d", i) + "].operator", Message: "operator must be one of: gt, lt, gte, lte, eq"})
		}
		if rule.ScaleBy <= 0 {
			errors = append(errors, FieldError{Field: "scale_up_rules[" + fmt.Sprintf("%d", i) + "].scale_by", Message: "scale_by must be positive for scale-up rules"})
		}
		if rule.DurationSeconds < 0 {
			errors = append(errors, FieldError{Field: "scale_up_rules[" + fmt.Sprintf("%d", i) + "].duration_seconds", Message: "duration_seconds must be >= 0"})
		}
	}

	// Validate scale-down rules
	for i, rule := range r.ScaleDownRules {
		if rule.Metric == "" {
			errors = append(errors, FieldError{Field: "scale_down_rules[" + fmt.Sprintf("%d", i) + "].metric", Message: "metric is required"})
		}
		if !rule.Operator.IsValid() {
			errors = append(errors, FieldError{Field: "scale_down_rules[" + fmt.Sprintf("%d", i) + "].operator", Message: "operator must be one of: gt, lt, gte, lte, eq"})
		}
		if rule.ScaleBy >= 0 {
			errors = append(errors, FieldError{Field: "scale_down_rules[" + fmt.Sprintf("%d", i) + "].scale_by", Message: "scale_by must be negative for scale-down rules"})
		}
		if rule.DurationSeconds < 0 {
			errors = append(errors, FieldError{Field: "scale_down_rules[" + fmt.Sprintf("%d", i) + "].duration_seconds", Message: "duration_seconds must be >= 0"})
		}
	}

	// Validate schedules
	for i, schedule := range r.Schedules {
		if schedule.CronExpression == "" {
			errors = append(errors, FieldError{Field: "schedules[" + fmt.Sprintf("%d", i) + "].cron_expression", Message: "cron_expression is required"})
		}
		if schedule.DesiredReplicas < 0 {
			errors = append(errors, FieldError{Field: "schedules[" + fmt.Sprintf("%d", i) + "].desired_replicas", Message: "desired_replicas must be >= 0"})
		}
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateScalingPolicyRequest) ApplyDefaults() {
	if r.CooldownSeconds == nil {
		cooldown := 300
		r.CooldownSeconds = &cooldown
	}
	if r.ScaleToZero == nil {
		scaleToZero := false
		r.ScaleToZero = &scaleToZero
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
	for i := range r.Schedules {
		if r.Schedules[i].Timezone == "" {
			r.Schedules[i].Timezone = "UTC"
		}
		if r.Schedules[i].Enabled == nil {
			enabled := true
			r.Schedules[i].Enabled = &enabled
		}
	}
}

// ToScalingPolicy converts the request to a scaling policy.
func (r *CreateScalingPolicyRequest) ToScalingPolicy() *scaling.Policy {
	policy := &scaling.Policy{
		Name:            r.Name,
		TargetType:      r.TargetType,
		TargetID:        r.TargetID,
		MinReplicas:     r.MinReplicas,
		MaxReplicas:     r.MaxReplicas,
		CooldownSeconds: *r.CooldownSeconds,
		MaxHourlyCost:   r.MaxHourlyCost,
		ScaleToZero:     *r.ScaleToZero,
		Enabled:         *r.Enabled,
	}

	for _, rule := range r.ScaleUpRules {
		policy.ScaleUpRules = append(policy.ScaleUpRules, scaling.Rule{
			RuleType:        scaling.RuleTypeScaleUp,
			Metric:          rule.Metric,
			Operator:        rule.Operator,
			Threshold:       rule.Threshold,
			DurationSeconds: rule.DurationSeconds,
			ScaleBy:         rule.ScaleBy,
		})
	}

	for _, rule := range r.ScaleDownRules {
		policy.ScaleDownRules = append(policy.ScaleDownRules, scaling.Rule{
			RuleType:        scaling.RuleTypeScaleDown,
			Metric:          rule.Metric,
			Operator:        rule.Operator,
			Threshold:       rule.Threshold,
			DurationSeconds: rule.DurationSeconds,
			ScaleBy:         rule.ScaleBy,
		})
	}

	for _, schedule := range r.Schedules {
		enabled := true
		if schedule.Enabled != nil {
			enabled = *schedule.Enabled
		}
		policy.Schedules = append(policy.Schedules, scaling.Schedule{
			CronExpression:  schedule.CronExpression,
			DesiredReplicas: schedule.DesiredReplicas,
			Timezone:        schedule.Timezone,
			Enabled:         enabled,
		})
	}

	return policy
}

// UpdateScalingPolicyRequest represents a request to update a scaling policy.
type UpdateScalingPolicyRequest struct {
	Name            *string                  `json:"name,omitempty"`
	MinReplicas     *int                     `json:"min_replicas,omitempty"`
	MaxReplicas     *int                     `json:"max_replicas,omitempty"`
	CooldownSeconds *int                     `json:"cooldown_seconds,omitempty"`
	MaxHourlyCost   *float64                 `json:"max_hourly_cost,omitempty"`
	ScaleToZero     *bool                    `json:"scale_to_zero,omitempty"`
	Enabled         *bool                    `json:"enabled,omitempty"`
	ScaleUpRules    []ScalingRuleRequest     `json:"scale_up_rules,omitempty"`
	ScaleDownRules  []ScalingRuleRequest     `json:"scale_down_rules,omitempty"`
	Schedules       []ScalingScheduleRequest `json:"schedules,omitempty"`
}

// Validate validates the update scaling policy request.
func (r *UpdateScalingPolicyRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name != nil && *r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name cannot be empty"})
	}
	if r.MinReplicas != nil && *r.MinReplicas < 0 {
		errors = append(errors, FieldError{Field: "min_replicas", Message: "min_replicas must be >= 0"})
	}
	if r.MaxReplicas != nil && *r.MaxReplicas < 1 {
		errors = append(errors, FieldError{Field: "max_replicas", Message: "max_replicas must be >= 1"})
	}
	if r.CooldownSeconds != nil && *r.CooldownSeconds < 0 {
		errors = append(errors, FieldError{Field: "cooldown_seconds", Message: "cooldown_seconds must be >= 0"})
	}
	if r.MaxHourlyCost != nil && *r.MaxHourlyCost < 0 {
		errors = append(errors, FieldError{Field: "max_hourly_cost", Message: "max_hourly_cost must be >= 0"})
	}

	return errors
}

// ApplyToPolicy applies the update to an existing policy.
func (r *UpdateScalingPolicyRequest) ApplyToPolicy(policy *scaling.Policy) {
	if r.Name != nil {
		policy.Name = *r.Name
	}
	if r.MinReplicas != nil {
		policy.MinReplicas = *r.MinReplicas
	}
	if r.MaxReplicas != nil {
		policy.MaxReplicas = *r.MaxReplicas
	}
	if r.CooldownSeconds != nil {
		policy.CooldownSeconds = *r.CooldownSeconds
	}
	if r.MaxHourlyCost != nil {
		policy.MaxHourlyCost = r.MaxHourlyCost
	}
	if r.ScaleToZero != nil {
		policy.ScaleToZero = *r.ScaleToZero
	}
	if r.Enabled != nil {
		policy.Enabled = *r.Enabled
	}

	// Replace rules if provided
	if r.ScaleUpRules != nil {
		policy.ScaleUpRules = make([]scaling.Rule, 0, len(r.ScaleUpRules))
		for _, rule := range r.ScaleUpRules {
			policy.ScaleUpRules = append(policy.ScaleUpRules, scaling.Rule{
				RuleType:        scaling.RuleTypeScaleUp,
				Metric:          rule.Metric,
				Operator:        rule.Operator,
				Threshold:       rule.Threshold,
				DurationSeconds: rule.DurationSeconds,
				ScaleBy:         rule.ScaleBy,
			})
		}
	}

	if r.ScaleDownRules != nil {
		policy.ScaleDownRules = make([]scaling.Rule, 0, len(r.ScaleDownRules))
		for _, rule := range r.ScaleDownRules {
			policy.ScaleDownRules = append(policy.ScaleDownRules, scaling.Rule{
				RuleType:        scaling.RuleTypeScaleDown,
				Metric:          rule.Metric,
				Operator:        rule.Operator,
				Threshold:       rule.Threshold,
				DurationSeconds: rule.DurationSeconds,
				ScaleBy:         rule.ScaleBy,
			})
		}
	}

	// Replace schedules if provided
	if r.Schedules != nil {
		policy.Schedules = make([]scaling.Schedule, 0, len(r.Schedules))
		for _, schedule := range r.Schedules {
			timezone := schedule.Timezone
			if timezone == "" {
				timezone = "UTC"
			}
			enabled := true
			if schedule.Enabled != nil {
				enabled = *schedule.Enabled
			}
			policy.Schedules = append(policy.Schedules, scaling.Schedule{
				CronExpression:  schedule.CronExpression,
				DesiredReplicas: schedule.DesiredReplicas,
				Timezone:        timezone,
				Enabled:         enabled,
			})
		}
	}
}

// ScalingPolicyResponse wraps a scaling policy for API responses.
type ScalingPolicyResponse struct {
	Policy *scaling.Policy `json:"policy"`
}

// ScalingPolicyListResponse wraps a list of scaling policies for API responses.
type ScalingPolicyListResponse struct {
	Policies   []scaling.Policy `json:"policies"`
	TotalCount int              `json:"total_count"`
}

// ScalingHistoryResponse wraps scaling history for API responses.
type ScalingHistoryResponse struct {
	History    []scaling.History `json:"history"`
	TotalCount int               `json:"total_count"`
}

// EvaluatePolicyRequest represents a request to evaluate a scaling policy.
type EvaluatePolicyRequest struct {
	DryRun bool `json:"dry_run"`
}

// EvaluatePolicyResponse represents the result of evaluating a scaling policy.
type EvaluatePolicyResponse struct {
	Decision        *scaling.Decision `json:"decision"`
	CurrentReplicas int               `json:"current_replicas"`
	WouldScale      bool              `json:"would_scale"`
	Action          string            `json:"action,omitempty"`
	DesiredReplicas int               `json:"desired_replicas,omitempty"`
	Reason          string            `json:"reason"`
	DryRun          bool              `json:"dry_run"`
}

// ScalingStateResponse wraps scaling state for API responses.
type ScalingStateResponse struct {
	State        *scaling.State `json:"state"`
	PolicyID     uuid.UUID      `json:"policy_id"`
	PolicyName   string         `json:"policy_name"`
	InCooldown   bool           `json:"in_cooldown"`
	CooldownEnds *time.Time     `json:"cooldown_ends,omitempty"`
}

// ScalingSummaryResponse provides a summary of scaling statistics.
type ScalingSummaryResponse struct {
	TotalPolicies        int `json:"total_policies"`
	EnabledPolicies      int `json:"enabled_policies"`
	TotalScaleUpEvents   int `json:"total_scale_up_events"`
	TotalScaleDownEvents int `json:"total_scale_down_events"`
	RecentScalingEvents  int `json:"recent_scaling_events"`
}
