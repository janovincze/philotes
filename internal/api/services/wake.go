// Package services provides business logic for API endpoints.
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/scaling"
	"github.com/janovincze/philotes/internal/scaling/idle"
	"github.com/janovincze/philotes/internal/scaling/wake"
)

// WakeService handles wake operations for scaled-to-zero policies.
type WakeService struct {
	wakeTrigger  *wake.Trigger
	idleDetector *idle.Detector
	idleRepo     idle.Repository
}

// NewWakeService creates a new WakeService.
func NewWakeService(
	wakeTrigger *wake.Trigger,
	idleDetector *idle.Detector,
	idleRepo idle.Repository,
) *WakeService {
	return &WakeService{
		wakeTrigger:  wakeTrigger,
		idleDetector: idleDetector,
		idleRepo:     idleRepo,
	}
}

// WakePolicy wakes a specific policy.
func (s *WakeService) WakePolicy(ctx context.Context, policyID uuid.UUID, reason scaling.WakeReason) (*models.WakePolicyResponse, error) {
	result, err := s.wakeTrigger.Wake(ctx, policyID, reason)
	if err != nil {
		return nil, fmt.Errorf("failed to wake policy: %w", err)
	}

	return &models.WakePolicyResponse{
		PolicyID:              result.PolicyID,
		PreviousReplicas:      result.PreviousReplicas,
		TargetReplicas:        result.TargetReplicas,
		Reason:                string(result.Reason),
		Status:                string(result.Status),
		EstimatedReadySeconds: result.EstimatedReadySeconds,
		Message:               result.Message,
		Error:                 result.Error,
	}, nil
}

// WakeAll wakes all scaled-to-zero policies or specific policies.
func (s *WakeService) WakeAll(ctx context.Context, policyIDs []uuid.UUID, reason scaling.WakeReason) (*models.WakeAllResponse, error) {
	var results []wake.WakeResult
	var err error

	if len(policyIDs) > 0 {
		results, err = s.wakeTrigger.WakeMultiple(ctx, policyIDs, reason)
	} else {
		results, err = s.wakeTrigger.WakeAll(ctx, reason)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to wake policies: %w", err)
	}

	response := &models.WakeAllResponse{
		Policies: make([]models.WakePolicyResponse, 0, len(results)),
	}

	for _, result := range results {
		switch result.Status {
		case wake.WakeStatusCompleted:
			if result.PreviousReplicas > 0 {
				response.AlreadyRunning++
			} else {
				response.Woken++
			}
		case wake.WakeStatusFailed:
			response.Failed++
		}

		response.Policies = append(response.Policies, models.WakePolicyResponse{
			PolicyID:              result.PolicyID,
			PreviousReplicas:      result.PreviousReplicas,
			TargetReplicas:        result.TargetReplicas,
			Reason:                string(result.Reason),
			Status:                string(result.Status),
			EstimatedReadySeconds: result.EstimatedReadySeconds,
			Message:               result.Message,
			Error:                 result.Error,
		})
	}

	return response, nil
}

// GetIdleState returns the idle state for a policy.
func (s *WakeService) GetIdleState(ctx context.Context, policyID uuid.UUID) (*models.IdleStateResponse, error) {
	state, err := s.idleDetector.GetIdleState(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle state: %w", err)
	}

	if state == nil {
		return nil, &NotFoundError{
			Resource: "idle_state",
			ID:       policyID.String(),
		}
	}

	var wakeReason *string
	if state.WakeReason != nil {
		s := state.WakeReason.String()
		wakeReason = &s
	}

	return &models.IdleStateResponse{
		PolicyID:         state.PolicyID,
		LastActivityAt:   state.LastActivityAt,
		IdleSince:        state.IdleSince,
		IdleDurationSecs: state.IdleDuration().Seconds(),
		IsScaledToZero:   state.IsScaledToZero,
		ScaledToZeroAt:   state.ScaledToZeroAt,
		LastWakeAt:       state.LastWakeAt,
		WakeReason:       wakeReason,
	}, nil
}

// ListScaledToZero returns all policies currently scaled to zero.
func (s *WakeService) ListScaledToZero(ctx context.Context) (*models.ScaledToZeroListResponse, error) {
	states, err := s.idleRepo.ListScaledToZero(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list scaled-to-zero policies: %w", err)
	}

	response := &models.ScaledToZeroListResponse{
		Policies:   make([]models.IdleStateResponse, 0, len(states)),
		TotalCount: len(states),
	}

	for i := range states {
		var wakeReason *string
		if states[i].WakeReason != nil {
			s := states[i].WakeReason.String()
			wakeReason = &s
		}

		response.Policies = append(response.Policies, models.IdleStateResponse{
			PolicyID:         states[i].PolicyID,
			LastActivityAt:   states[i].LastActivityAt,
			IdleSince:        states[i].IdleSince,
			IdleDurationSecs: states[i].IdleDuration().Seconds(),
			IsScaledToZero:   states[i].IsScaledToZero,
			ScaledToZeroAt:   states[i].ScaledToZeroAt,
			LastWakeAt:       states[i].LastWakeAt,
			WakeReason:       wakeReason,
		})
	}

	return response, nil
}

// GetCostSavings returns cost savings for a policy.
func (s *WakeService) GetCostSavings(ctx context.Context, policyID uuid.UUID, days int) (*models.CostSavingsResponse, error) {
	if days <= 0 {
		days = 30
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	savings, err := s.idleRepo.GetCostSavings(ctx, policyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost savings: %w", err)
	}

	response := &models.CostSavingsResponse{
		PolicyID:       policyID,
		Period:         fmt.Sprintf("last_%d_days", days),
		DailyBreakdown: make([]models.DailySavingsResponse, 0, len(savings)),
	}

	for i := range savings {
		response.TotalIdleHours += savings[i].IdleHours()
		response.TotalZeroHours += savings[i].ScaledToZeroHours()
		response.SavingsEuros += savings[i].EstimatedSavingsEuros()

		response.DailyBreakdown = append(response.DailyBreakdown, models.DailySavingsResponse{
			Date:         savings[i].Date.Format("2006-01-02"),
			IdleHours:    savings[i].IdleHours(),
			ZeroHours:    savings[i].ScaledToZeroHours(),
			SavingsEuros: savings[i].EstimatedSavingsEuros(),
		})
	}

	return response, nil
}

// GetSavingsSummary returns overall cost savings summary.
func (s *WakeService) GetSavingsSummary(ctx context.Context) (*models.SavingsSummaryResponse, error) {
	// Get all idle states to find policies
	states, err := s.idleRepo.ListIdleStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list idle states: %w", err)
	}

	response := &models.SavingsSummaryResponse{
		PolicyCount: len(states),
		Policies:    make([]models.PolicySavingsPreview, 0, len(states)),
	}

	for i := range states {
		summary, err := s.idleRepo.GetTotalSavings(ctx, states[i].PolicyID)
		if err != nil {
			continue
		}

		idleHours := float64(summary.TotalIdleSeconds) / 3600.0
		zeroHours := float64(summary.TotalScaledToZeroSeconds) / 3600.0
		savingsEuros := float64(summary.TotalSavingsCents) / 100.0

		response.TotalIdleHours += idleHours
		response.TotalZeroHours += zeroHours
		response.SavingsEuros += savingsEuros

		response.Policies = append(response.Policies, models.PolicySavingsPreview{
			PolicyID:     states[i].PolicyID,
			IdleHours:    idleHours,
			SavingsEuros: savingsEuros,
		})
	}

	return response, nil
}

// RecordActivity records activity for a policy.
func (s *WakeService) RecordActivity(ctx context.Context, policyID uuid.UUID) error {
	return s.idleDetector.RecordActivity(ctx, policyID)
}
