// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TargetType represents the type of scaling target.
type TargetType string

const (
	// TargetCDCWorker represents a CDC worker deployment.
	TargetCDCWorker TargetType = "cdc-worker"
	// TargetTrino represents a Trino query engine.
	TargetTrino TargetType = "trino"
	// TargetRisingWave represents a RisingWave streaming engine.
	TargetRisingWave TargetType = "risingwave"
	// TargetNodes represents infrastructure nodes.
	TargetNodes TargetType = "nodes"
)

// IsValid checks if the target type is valid.
func (t TargetType) IsValid() bool {
	switch t {
	case TargetCDCWorker, TargetTrino, TargetRisingWave, TargetNodes:
		return true
	}
	return false
}

// String returns the string representation of the target type.
func (t TargetType) String() string {
	return string(t)
}

// RuleType represents the type of scaling rule.
type RuleType string

const (
	// RuleTypeScaleUp indicates a scale-up rule.
	RuleTypeScaleUp RuleType = "scale_up"
	// RuleTypeScaleDown indicates a scale-down rule.
	RuleTypeScaleDown RuleType = "scale_down"
)

// IsValid checks if the rule type is valid.
func (r RuleType) IsValid() bool {
	switch r {
	case RuleTypeScaleUp, RuleTypeScaleDown:
		return true
	}
	return false
}

// Operator represents a comparison operator for scaling rules.
type Operator string

const (
	// OpGreaterThan represents the > operator.
	OpGreaterThan Operator = "gt"
	// OpLessThan represents the < operator.
	OpLessThan Operator = "lt"
	// OpGreaterThanEqual represents the >= operator.
	OpGreaterThanEqual Operator = "gte"
	// OpLessThanEqual represents the <= operator.
	OpLessThanEqual Operator = "lte"
	// OpEqual represents the == operator.
	OpEqual Operator = "eq"
)

// IsValid checks if the operator is valid.
func (o Operator) IsValid() bool {
	switch o {
	case OpGreaterThan, OpLessThan, OpGreaterThanEqual, OpLessThanEqual, OpEqual:
		return true
	}
	return false
}

// Evaluate evaluates the operator against the given values.
func (o Operator) Evaluate(value, threshold float64) bool {
	switch o {
	case OpGreaterThan:
		return value > threshold
	case OpLessThan:
		return value < threshold
	case OpGreaterThanEqual:
		return value >= threshold
	case OpLessThanEqual:
		return value <= threshold
	case OpEqual:
		return value == threshold
	}
	return false
}

// String returns the human-readable representation of the operator.
func (o Operator) String() string {
	switch o {
	case OpGreaterThan:
		return ">"
	case OpLessThan:
		return "<"
	case OpGreaterThanEqual:
		return ">="
	case OpLessThanEqual:
		return "<="
	case OpEqual:
		return "=="
	}
	return string(o)
}

// Action represents the type of scaling action.
type Action string

const (
	// ActionScaleUp indicates a scale-up action.
	ActionScaleUp Action = "scale_up"
	// ActionScaleDown indicates a scale-down action.
	ActionScaleDown Action = "scale_down"
	// ActionScheduled indicates a scheduled scaling action.
	ActionScheduled Action = "scheduled"
	// ActionManual indicates a manual scaling action.
	ActionManual Action = "manual"
)

// IsValid checks if the action is valid.
func (a Action) IsValid() bool {
	switch a {
	case ActionScaleUp, ActionScaleDown, ActionScheduled, ActionManual:
		return true
	}
	return false
}

// Policy represents a scaling policy configuration.
type Policy struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	TargetType      TargetType `json:"target_type"`
	TargetID        *uuid.UUID `json:"target_id,omitempty"`
	MinReplicas     int        `json:"min_replicas"`
	MaxReplicas     int        `json:"max_replicas"`
	CooldownSeconds int        `json:"cooldown_seconds"`
	MaxHourlyCost   *float64   `json:"max_hourly_cost,omitempty"`
	ScaleToZero     bool       `json:"scale_to_zero"`
	Enabled         bool       `json:"enabled"`
	ScaleUpRules    []Rule     `json:"scale_up_rules,omitempty"`
	ScaleDownRules  []Rule     `json:"scale_down_rules,omitempty"`
	Schedules       []Schedule `json:"schedules,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Validate validates the scaling policy.
func (p *Policy) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if !p.TargetType.IsValid() {
		return fmt.Errorf("invalid target_type: %s", p.TargetType)
	}
	if p.MinReplicas < 0 {
		return fmt.Errorf("min_replicas must be >= 0")
	}
	if p.MaxReplicas < 1 {
		return fmt.Errorf("max_replicas must be >= 1")
	}
	if p.MinReplicas > p.MaxReplicas {
		return fmt.Errorf("min_replicas (%d) cannot be greater than max_replicas (%d)", p.MinReplicas, p.MaxReplicas)
	}
	if p.CooldownSeconds < 0 {
		return fmt.Errorf("cooldown_seconds must be >= 0")
	}
	if p.MaxHourlyCost != nil && *p.MaxHourlyCost < 0 {
		return fmt.Errorf("max_hourly_cost must be >= 0")
	}

	for i := range p.ScaleUpRules {
		if err := p.ScaleUpRules[i].Validate(); err != nil {
			return fmt.Errorf("scale_up_rules[%d]: %w", i, err)
		}
		if p.ScaleUpRules[i].ScaleBy <= 0 {
			return fmt.Errorf("scale_up_rules[%d]: scale_by must be positive for scale-up rules", i)
		}
	}

	for i := range p.ScaleDownRules {
		if err := p.ScaleDownRules[i].Validate(); err != nil {
			return fmt.Errorf("scale_down_rules[%d]: %w", i, err)
		}
		if p.ScaleDownRules[i].ScaleBy >= 0 {
			return fmt.Errorf("scale_down_rules[%d]: scale_by must be negative for scale-down rules", i)
		}
	}

	for i, schedule := range p.Schedules {
		if err := schedule.Validate(); err != nil {
			return fmt.Errorf("schedules[%d]: %w", i, err)
		}
	}

	return nil
}

// CooldownDuration returns the cooldown as a time.Duration.
func (p *Policy) CooldownDuration() time.Duration {
	return time.Duration(p.CooldownSeconds) * time.Second
}

// ClampReplicas clamps the given replica count to policy limits.
func (p *Policy) ClampReplicas(replicas int) int {
	if replicas < p.MinReplicas {
		return p.MinReplicas
	}
	if replicas > p.MaxReplicas {
		return p.MaxReplicas
	}
	return replicas
}

// Rule represents a scaling rule definition.
type Rule struct {
	ID              uuid.UUID `json:"id"`
	PolicyID        uuid.UUID `json:"policy_id"`
	RuleType        RuleType  `json:"rule_type"`
	Metric          string    `json:"metric"`
	Operator        Operator  `json:"operator"`
	Threshold       float64   `json:"threshold"`
	DurationSeconds int       `json:"duration_seconds"`
	ScaleBy         int       `json:"scale_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// Validate validates the scaling rule.
func (r *Rule) Validate() error {
	if r.Metric == "" {
		return fmt.Errorf("metric is required")
	}
	if !r.Operator.IsValid() {
		return fmt.Errorf("invalid operator: %s", r.Operator)
	}
	if r.DurationSeconds < 0 {
		return fmt.Errorf("duration_seconds must be >= 0")
	}
	if r.ScaleBy == 0 {
		return fmt.Errorf("scale_by cannot be zero")
	}
	return nil
}

// Duration returns the duration as a time.Duration.
func (r *Rule) Duration() time.Duration {
	return time.Duration(r.DurationSeconds) * time.Second
}

// Evaluate checks if the rule condition is met.
func (r *Rule) Evaluate(value float64) bool {
	return r.Operator.Evaluate(value, r.Threshold)
}

// Schedule represents a scheduled scaling configuration.
type Schedule struct {
	ID              uuid.UUID `json:"id"`
	PolicyID        uuid.UUID `json:"policy_id"`
	CronExpression  string    `json:"cron_expression"`
	DesiredReplicas int       `json:"desired_replicas"`
	Timezone        string    `json:"timezone"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
}

// Validate validates the scaling schedule.
func (s *Schedule) Validate() error {
	if s.CronExpression == "" {
		return fmt.Errorf("cron_expression is required")
	}
	if s.DesiredReplicas < 0 {
		return fmt.Errorf("desired_replicas must be >= 0")
	}
	if s.Timezone == "" {
		return fmt.Errorf("timezone is required")
	}
	// Timezone validation is done during cron parsing
	return nil
}

// History represents a scaling action audit log entry.
type History struct {
	ID               uuid.UUID  `json:"id"`
	PolicyID         *uuid.UUID `json:"policy_id,omitempty"`
	PolicyName       string     `json:"policy_name"`
	Action           Action     `json:"action"`
	TargetType       TargetType `json:"target_type"`
	TargetID         *uuid.UUID `json:"target_id,omitempty"`
	PreviousReplicas int        `json:"previous_replicas"`
	NewReplicas      int        `json:"new_replicas"`
	Reason           string     `json:"reason,omitempty"`
	TriggeredBy      string     `json:"triggered_by,omitempty"`
	DryRun           bool       `json:"dry_run"`
	ExecutedAt       time.Time  `json:"executed_at"`
}

// State represents the current scaling state for a policy.
type State struct {
	ID                uuid.UUID            `json:"id"`
	PolicyID          uuid.UUID            `json:"policy_id"`
	CurrentReplicas   int                  `json:"current_replicas"`
	LastScaleTime     *time.Time           `json:"last_scale_time,omitempty"`
	LastScaleAction   string               `json:"last_scale_action,omitempty"`
	PendingConditions map[string]time.Time `json:"pending_conditions,omitempty"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

// IsInCooldown checks if the policy is currently in cooldown.
func (s *State) IsInCooldown(cooldownDuration time.Duration) bool {
	if s.LastScaleTime == nil {
		return false
	}
	return time.Since(*s.LastScaleTime) < cooldownDuration
}

// Decision represents a scaling decision made by the evaluator.
type Decision struct {
	Policy            *Policy
	Action            Action
	CurrentReplicas   int
	DesiredReplicas   int
	Reason            string
	TriggeredBy       string // "rule:<id>", "schedule:<id>", "manual"
	ShouldExecute     bool
	CooldownRemaining time.Duration
}

// Delta returns the change in replicas.
func (d *Decision) Delta() int {
	return d.DesiredReplicas - d.CurrentReplicas
}

// EvaluationResult represents the result of evaluating a single rule.
type EvaluationResult struct {
	Rule        *Rule
	MetricValue float64
	Triggered   bool
	Duration    time.Duration // How long the condition has been true
}

// MetricValue represents a single metric value from Prometheus.
type MetricValue struct {
	Labels map[string]string
	Value  float64
	Time   time.Time
}
