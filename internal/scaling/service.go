// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Service provides business logic for scaling operations.
type Service struct {
	repo   *Repository
	logger *slog.Logger
}

// NewService creates a new scaling service.
func NewService(repo *Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repo:   repo,
		logger: logger.With("component", "scaling-service"),
	}
}

// CreatePolicy creates a new scaling policy with its rules and schedules.
func (s *Service) CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	// Validate the policy
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	// Validate cron expressions in schedules
	for i, schedule := range policy.Schedules {
		if err := s.validateCronExpression(schedule.CronExpression, schedule.Timezone); err != nil {
			return nil, fmt.Errorf("invalid schedule[%d]: %w", i, err)
		}
	}

	// Create the policy
	created, err := s.repo.CreatePolicy(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	// Create rules
	for i := range policy.ScaleUpRules {
		policy.ScaleUpRules[i].PolicyID = created.ID
		policy.ScaleUpRules[i].RuleType = RuleTypeScaleUp
		if _, err := s.repo.CreateRule(ctx, &policy.ScaleUpRules[i]); err != nil {
			return nil, fmt.Errorf("failed to create scale-up rule: %w", err)
		}
	}

	for i := range policy.ScaleDownRules {
		policy.ScaleDownRules[i].PolicyID = created.ID
		policy.ScaleDownRules[i].RuleType = RuleTypeScaleDown
		if _, err := s.repo.CreateRule(ctx, &policy.ScaleDownRules[i]); err != nil {
			return nil, fmt.Errorf("failed to create scale-down rule: %w", err)
		}
	}

	// Create schedules
	for i := range policy.Schedules {
		policy.Schedules[i].PolicyID = created.ID
		if _, err := s.repo.CreateSchedule(ctx, &policy.Schedules[i]); err != nil {
			return nil, fmt.Errorf("failed to create schedule: %w", err)
		}
	}

	// Initialize scaling state
	state := &State{
		PolicyID:        created.ID,
		CurrentReplicas: policy.MinReplicas,
	}
	if _, err := s.repo.CreateState(ctx, state); err != nil {
		s.logger.Warn("failed to create initial scaling state", "error", err)
	}

	s.logger.Info("created scaling policy",
		"policy_id", created.ID,
		"policy_name", created.Name,
		"target_type", created.TargetType,
	)

	// Return the full policy with rules and schedules
	return s.GetPolicy(ctx, created.ID)
}

// GetPolicy retrieves a policy by ID with all its rules and schedules.
func (s *Service) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	// Load rules
	rules, err := s.repo.GetRulesForPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Separate scale-up and scale-down rules
	for i := range rules {
		if rules[i].RuleType == RuleTypeScaleUp {
			policy.ScaleUpRules = append(policy.ScaleUpRules, rules[i])
		} else {
			policy.ScaleDownRules = append(policy.ScaleDownRules, rules[i])
		}
	}

	// Load schedules
	schedules, err := s.repo.GetSchedulesForPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}
	policy.Schedules = schedules

	return policy, nil
}

// ListPolicies lists all scaling policies with optional filtering.
func (s *Service) ListPolicies(ctx context.Context, enabledOnly bool) ([]Policy, error) {
	policies, err := s.repo.ListPolicies(ctx, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	// Load rules and schedules for each policy
	for i := range policies {
		rules, err := s.repo.GetRulesForPolicy(ctx, policies[i].ID)
		if err != nil {
			s.logger.Warn("failed to get rules for policy",
				"policy_id", policies[i].ID,
				"error", err,
			)
			continue
		}

		for j := range rules {
			if rules[j].RuleType == RuleTypeScaleUp {
				policies[i].ScaleUpRules = append(policies[i].ScaleUpRules, rules[j])
			} else {
				policies[i].ScaleDownRules = append(policies[i].ScaleDownRules, rules[j])
			}
		}

		schedules, err := s.repo.GetSchedulesForPolicy(ctx, policies[i].ID)
		if err != nil {
			s.logger.Warn("failed to get schedules for policy",
				"policy_id", policies[i].ID,
				"error", err,
			)
			continue
		}
		policies[i].Schedules = schedules
	}

	return policies, nil
}

// UpdatePolicy updates a scaling policy.
func (s *Service) UpdatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	// Validate the policy
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy: %w", err)
	}

	// Validate cron expressions in schedules
	for i, schedule := range policy.Schedules {
		if err := s.validateCronExpression(schedule.CronExpression, schedule.Timezone); err != nil {
			return nil, fmt.Errorf("invalid schedule[%d]: %w", i, err)
		}
	}

	// Check policy exists
	existing, err := s.repo.GetPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	// Update the policy
	if updateErr := s.repo.UpdatePolicy(ctx, policy); updateErr != nil {
		return nil, fmt.Errorf("failed to update policy: %w", updateErr)
	}

	// Delete existing rules and recreate
	existingRules, err := s.repo.GetRulesForPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing rules: %w", err)
	}

	for i := range existingRules {
		if deleteErr := s.repo.DeleteRule(ctx, existingRules[i].ID); deleteErr != nil {
			s.logger.Warn("failed to delete rule", "rule_id", existingRules[i].ID, "error", deleteErr)
		}
	}

	// Create new rules
	for i := range policy.ScaleUpRules {
		policy.ScaleUpRules[i].PolicyID = policy.ID
		policy.ScaleUpRules[i].RuleType = RuleTypeScaleUp
		if _, createErr := s.repo.CreateRule(ctx, &policy.ScaleUpRules[i]); createErr != nil {
			return nil, fmt.Errorf("failed to create scale-up rule: %w", createErr)
		}
	}

	for i := range policy.ScaleDownRules {
		policy.ScaleDownRules[i].PolicyID = policy.ID
		policy.ScaleDownRules[i].RuleType = RuleTypeScaleDown
		if _, createErr := s.repo.CreateRule(ctx, &policy.ScaleDownRules[i]); createErr != nil {
			return nil, fmt.Errorf("failed to create scale-down rule: %w", createErr)
		}
	}

	// Delete existing schedules and recreate
	existingSchedules, err := s.repo.GetSchedulesForPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing schedules: %w", err)
	}

	for _, schedule := range existingSchedules {
		if deleteErr := s.repo.DeleteSchedule(ctx, schedule.ID); deleteErr != nil {
			s.logger.Warn("failed to delete schedule", "schedule_id", schedule.ID, "error", deleteErr)
		}
	}

	// Create new schedules
	for _, schedule := range policy.Schedules {
		schedule.PolicyID = policy.ID
		if _, err := s.repo.CreateSchedule(ctx, &schedule); err != nil {
			return nil, fmt.Errorf("failed to create schedule: %w", err)
		}
	}

	s.logger.Info("updated scaling policy",
		"policy_id", policy.ID,
		"policy_name", existing.Name,
	)

	return s.GetPolicy(ctx, policy.ID)
}

// DeletePolicy deletes a scaling policy.
func (s *Service) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	if err := s.repo.DeletePolicy(ctx, id); err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	s.logger.Info("deleted scaling policy",
		"policy_id", id,
		"policy_name", policy.Name,
	)

	return nil
}

// EnablePolicy enables a scaling policy.
func (s *Service) EnablePolicy(ctx context.Context, id uuid.UUID) error {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	policy.Enabled = true
	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to enable policy: %w", err)
	}

	s.logger.Info("enabled scaling policy",
		"policy_id", id,
		"policy_name", policy.Name,
	)

	return nil
}

// DisablePolicy disables a scaling policy.
func (s *Service) DisablePolicy(ctx context.Context, id uuid.UUID) error {
	policy, err := s.repo.GetPolicy(ctx, id)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	policy.Enabled = false
	if err := s.repo.UpdatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to disable policy: %w", err)
	}

	s.logger.Info("disabled scaling policy",
		"policy_id", id,
		"policy_name", policy.Name,
	)

	return nil
}

// GetHistory retrieves scaling history for a policy.
func (s *Service) GetHistory(ctx context.Context, policyID *uuid.UUID, limit int) ([]History, error) {
	if policyID != nil {
		return s.repo.GetHistoryForPolicy(ctx, *policyID, limit)
	}
	return s.repo.ListRecentHistory(ctx, limit)
}

// GetState retrieves the current scaling state for a policy.
func (s *Service) GetState(ctx context.Context, policyID uuid.UUID) (*State, error) {
	return s.repo.GetState(ctx, policyID)
}

// RecordHistory records a scaling action in the history.
func (s *Service) RecordHistory(ctx context.Context, history *History) (*History, error) {
	return s.repo.CreateHistory(ctx, history)
}

// validateCronExpression validates a cron expression with timezone.
func (s *Service) validateCronExpression(expr, timezone string) error {
	// Parse the timezone
	loc, err := parseTimezone(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	// Parse the cron expression with the timezone
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Verify schedule is valid by getting next occurrence
	_ = schedule.Next(loc.Now())

	return nil
}

// timeLocation wraps time.Location to provide a Now() method.
type timeLocation struct {
	loc *time.Location
}

func (t *timeLocation) Now() time.Time {
	return time.Now().In(t.loc)
}

// parseTimezone parses a timezone string and returns a timeLocation.
func parseTimezone(timezone string) (*timeLocation, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}
	return &timeLocation{loc: loc}, nil
}
