// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/scaling"
)

// WakePolicyRequest represents a request to wake a specific policy.
type WakePolicyRequest struct {
	Reason       string `json:"reason,omitempty"`
	WaitForReady bool   `json:"wait_for_ready,omitempty"`
}

// GetReason returns the wake reason or default to manual.
func (r *WakePolicyRequest) GetReason() scaling.WakeReason {
	if r.Reason == "" {
		return scaling.WakeReasonManual
	}
	reason := scaling.WakeReason(r.Reason)
	if !reason.IsValid() {
		return scaling.WakeReasonManual
	}
	return reason
}

// WakeAllRequest represents a request to wake multiple policies.
type WakeAllRequest struct {
	PolicyIDs []uuid.UUID `json:"policy_ids,omitempty"`
	Reason    string      `json:"reason,omitempty"`
}

// GetReason returns the wake reason or default to manual.
func (r *WakeAllRequest) GetReason() scaling.WakeReason {
	if r.Reason == "" {
		return scaling.WakeReasonManual
	}
	reason := scaling.WakeReason(r.Reason)
	if !reason.IsValid() {
		return scaling.WakeReasonManual
	}
	return reason
}

// WakePolicyResponse represents the response for a wake operation.
type WakePolicyResponse struct {
	PolicyID              uuid.UUID `json:"policy_id"`
	PreviousReplicas      int       `json:"previous_replicas"`
	TargetReplicas        int       `json:"target_replicas"`
	Reason                string    `json:"reason"`
	Status                string    `json:"status"`
	EstimatedReadySeconds int       `json:"estimated_ready_seconds,omitempty"`
	Message               string    `json:"message"`
	Error                 string    `json:"error,omitempty"`
}

// WakeAllResponse represents the response for waking multiple policies.
type WakeAllResponse struct {
	Woken          int                  `json:"woken"`
	AlreadyRunning int                  `json:"already_running"`
	Failed         int                  `json:"failed"`
	Policies       []WakePolicyResponse `json:"policies"`
}

// IdleStateResponse represents the idle state of a policy.
type IdleStateResponse struct {
	PolicyID         uuid.UUID  `json:"policy_id"`
	LastActivityAt   time.Time  `json:"last_activity_at"`
	IdleSince        *time.Time `json:"idle_since,omitempty"`
	IdleDurationSecs float64    `json:"idle_duration_seconds"`
	IsScaledToZero   bool       `json:"is_scaled_to_zero"`
	ScaledToZeroAt   *time.Time `json:"scaled_to_zero_at,omitempty"`
	LastWakeAt       *time.Time `json:"last_wake_at,omitempty"`
	WakeReason       *string    `json:"wake_reason,omitempty"`
}

// CostSavingsResponse represents cost savings for a policy.
type CostSavingsResponse struct {
	PolicyID       uuid.UUID              `json:"policy_id"`
	Period         string                 `json:"period"`
	TotalIdleHours float64                `json:"total_idle_hours"`
	TotalZeroHours float64                `json:"total_scaled_to_zero_hours"`
	SavingsEuros   float64                `json:"estimated_savings_euros"`
	DailyBreakdown []DailySavingsResponse `json:"daily_breakdown,omitempty"`
}

// DailySavingsResponse represents daily cost savings.
type DailySavingsResponse struct {
	Date         string  `json:"date"`
	IdleHours    float64 `json:"idle_hours"`
	ZeroHours    float64 `json:"scaled_to_zero_hours"`
	SavingsEuros float64 `json:"savings_euros"`
}

// SavingsSummaryResponse represents overall cost savings summary.
type SavingsSummaryResponse struct {
	TotalIdleHours float64                `json:"total_idle_hours"`
	TotalZeroHours float64                `json:"total_scaled_to_zero_hours"`
	SavingsEuros   float64                `json:"total_savings_euros"`
	PolicyCount    int                    `json:"policy_count"`
	Policies       []PolicySavingsPreview `json:"policies,omitempty"`
}

// PolicySavingsPreview represents a preview of savings for a policy.
type PolicySavingsPreview struct {
	PolicyID     uuid.UUID `json:"policy_id"`
	PolicyName   string    `json:"policy_name,omitempty"`
	IdleHours    float64   `json:"idle_hours"`
	SavingsEuros float64   `json:"savings_euros"`
}

// ScaledToZeroListResponse represents a list of scaled-to-zero policies.
type ScaledToZeroListResponse struct {
	Policies   []IdleStateResponse `json:"policies"`
	TotalCount int                 `json:"total_count"`
}
