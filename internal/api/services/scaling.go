// Package services provides business logic for API resources.
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/scaling"
)

// ScalingService provides business logic for scaling operations.
type ScalingService struct {
	coreService *scaling.Service
	manager     *scaling.Manager
	logger      *slog.Logger
}

// NewScalingService creates a new ScalingService.
func NewScalingService(coreService *scaling.Service, manager *scaling.Manager, logger *slog.Logger) *ScalingService {
	return &ScalingService{
		coreService: coreService,
		manager:     manager,
		logger:      logger.With("component", "scaling-api-service"),
	}
}

// CreatePolicy creates a new scaling policy.
func (s *ScalingService) CreatePolicy(ctx context.Context, req *models.CreateScalingPolicyRequest) (*scaling.Policy, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Convert to domain model
	policy := req.ToScalingPolicy()

	// Create via core service
	created, err := s.coreService.CreatePolicy(ctx, policy)
	if err != nil {
		s.logger.Error("failed to create scaling policy", "error", err)
		return nil, fmt.Errorf("failed to create scaling policy: %w", err)
	}

	s.logger.Info("scaling policy created", "id", created.ID, "name", created.Name)
	return created, nil
}

// GetPolicy retrieves a scaling policy by ID.
func (s *ScalingService) GetPolicy(ctx context.Context, id uuid.UUID) (*scaling.Policy, error) {
	policy, err := s.coreService.GetPolicy(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}
	return policy, nil
}

// ListPolicies retrieves all scaling policies.
func (s *ScalingService) ListPolicies(ctx context.Context, enabledOnly bool) (*models.ScalingPolicyListResponse, error) {
	policies, err := s.coreService.ListPolicies(ctx, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list scaling policies: %w", err)
	}

	if policies == nil {
		policies = []scaling.Policy{}
	}

	return &models.ScalingPolicyListResponse{
		Policies:   policies,
		TotalCount: len(policies),
	}, nil
}

// UpdatePolicy updates a scaling policy.
func (s *ScalingService) UpdatePolicy(ctx context.Context, id uuid.UUID, req *models.UpdateScalingPolicyRequest) (*scaling.Policy, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Get existing policy
	policy, err := s.coreService.GetPolicy(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}

	// Apply updates
	req.ApplyToPolicy(policy)

	// Update via core service
	updated, err := s.coreService.UpdatePolicy(ctx, policy)
	if err != nil {
		s.logger.Error("failed to update scaling policy", "error", err)
		return nil, fmt.Errorf("failed to update scaling policy: %w", err)
	}

	s.logger.Info("scaling policy updated", "id", updated.ID, "name", updated.Name)
	return updated, nil
}

// DeletePolicy deletes a scaling policy.
func (s *ScalingService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if err := s.coreService.DeletePolicy(ctx, id); err != nil {
		return &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}
	return nil
}

// EnablePolicy enables a scaling policy.
func (s *ScalingService) EnablePolicy(ctx context.Context, id uuid.UUID) error {
	if err := s.coreService.EnablePolicy(ctx, id); err != nil {
		return &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}
	return nil
}

// DisablePolicy disables a scaling policy.
func (s *ScalingService) DisablePolicy(ctx context.Context, id uuid.UUID) error {
	if err := s.coreService.DisablePolicy(ctx, id); err != nil {
		return &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}
	return nil
}

// EvaluatePolicy evaluates a scaling policy and returns the decision.
func (s *ScalingService) EvaluatePolicy(ctx context.Context, id uuid.UUID, dryRun bool) (*models.EvaluatePolicyResponse, error) {
	// Get policy first
	policy, err := s.coreService.GetPolicy(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}

	// Evaluate via manager
	decision, err := s.manager.EvaluatePolicyNow(ctx, policy.Name, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	response := &models.EvaluatePolicyResponse{
		Decision:        decision,
		CurrentReplicas: decision.CurrentReplicas,
		WouldScale:      decision.ShouldExecute,
		Reason:          decision.Reason,
		DryRun:          dryRun,
	}

	if decision.ShouldExecute {
		response.Action = string(decision.Action)
		response.DesiredReplicas = decision.DesiredReplicas
	}

	return response, nil
}

// GetPolicyState retrieves the current scaling state for a policy.
func (s *ScalingService) GetPolicyState(ctx context.Context, id uuid.UUID) (*models.ScalingStateResponse, error) {
	// Get policy
	policy, err := s.coreService.GetPolicy(ctx, id)
	if err != nil {
		return nil, &NotFoundError{Resource: "scaling policy", ID: id.String()}
	}

	// Get state
	state, err := s.coreService.GetState(ctx, id)
	if err != nil {
		// Return empty state if not found
		return &models.ScalingStateResponse{
			PolicyID:   id,
			PolicyName: policy.Name,
			InCooldown: false,
		}, nil
	}

	response := &models.ScalingStateResponse{
		State:      state,
		PolicyID:   id,
		PolicyName: policy.Name,
		InCooldown: state.IsInCooldown(policy.CooldownDuration()),
	}

	if response.InCooldown && state.LastScaleTime != nil {
		cooldownEnds := state.LastScaleTime.Add(policy.CooldownDuration())
		response.CooldownEnds = &cooldownEnds
	}

	return response, nil
}

// ListHistory retrieves scaling history.
func (s *ScalingService) ListHistory(ctx context.Context, policyID *uuid.UUID, limit int) (*models.ScalingHistoryResponse, error) {
	history, err := s.coreService.GetHistory(ctx, policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaling history: %w", err)
	}

	if history == nil {
		history = []scaling.History{}
	}

	return &models.ScalingHistoryResponse{
		History:    history,
		TotalCount: len(history),
	}, nil
}
