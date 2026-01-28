// Package scaling provides the auto-scaling engine for Philotes.
package scaling

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/janovincze/philotes/internal/config"
)

// Manager is the main scaling manager that coordinates policy evaluation and execution.
type Manager struct {
	repo      *Repository
	evaluator *Evaluator
	executor  Executor
	logger    *slog.Logger
	config    config.ScalingConfig

	// Control channels
	stopCh    chan struct{}
	stoppedCh chan struct{}
	running   bool
	runMu     sync.Mutex
}

// NewManager creates a new scaling manager.
func NewManager(repo *Repository, executor Executor, cfg config.ScalingConfig, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.PrometheusURL == "" {
		return nil, fmt.Errorf("prometheus URL is required")
	}

	evaluator := NewEvaluator(cfg.PrometheusURL, logger)

	return &Manager{
		repo:      repo,
		evaluator: evaluator,
		executor:  executor,
		logger:    logger.With("component", "scaling-manager"),
		config:    cfg,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

// SetExecutor sets the executor (useful for testing).
func (m *Manager) SetExecutor(executor Executor) {
	m.executor = executor
}

// Start starts the scaling manager evaluation loop.
func (m *Manager) Start(ctx context.Context) error {
	m.runMu.Lock()
	if m.running {
		m.runMu.Unlock()
		return fmt.Errorf("manager is already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.stoppedCh = make(chan struct{})
	m.runMu.Unlock()

	m.logger.Info("starting scaling manager",
		"evaluation_interval", m.config.EvaluationInterval,
		"prometheus_url", m.config.PrometheusURL,
	)

	go m.evaluationLoop(ctx)

	return nil
}

// Stop gracefully stops the scaling manager.
func (m *Manager) Stop() error {
	m.runMu.Lock()
	if !m.running {
		m.runMu.Unlock()
		return nil
	}
	m.runMu.Unlock()

	m.logger.Info("stopping scaling manager")

	close(m.stopCh)

	// Wait for the evaluation loop to finish
	<-m.stoppedCh

	m.runMu.Lock()
	m.running = false
	m.runMu.Unlock()

	m.logger.Info("scaling manager stopped")

	return nil
}

// IsRunning returns whether the manager is running.
func (m *Manager) IsRunning() bool {
	m.runMu.Lock()
	defer m.runMu.Unlock()
	return m.running
}

// evaluationLoop runs the periodic policy evaluation.
func (m *Manager) evaluationLoop(ctx context.Context) {
	defer close(m.stoppedCh)

	ticker := time.NewTicker(m.config.EvaluationInterval)
	defer ticker.Stop()

	// Run initial evaluation
	m.runEvaluation(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("context canceled, stopping evaluation loop")
			return
		case <-m.stopCh:
			m.logger.Info("stop signal received, stopping evaluation loop")
			return
		case <-ticker.C:
			m.runEvaluation(ctx)
		}
	}
}

// runEvaluation performs a single evaluation cycle.
func (m *Manager) runEvaluation(ctx context.Context) {
	m.logger.Debug("starting evaluation cycle")
	start := time.Now()

	if err := m.evaluatePolicies(ctx); err != nil {
		m.logger.Error("evaluation cycle failed", "error", err)
	}

	m.logger.Debug("evaluation cycle completed",
		"duration", time.Since(start),
	)
}

// evaluatePolicies evaluates all enabled scaling policies.
func (m *Manager) evaluatePolicies(ctx context.Context) error {
	// Load all enabled policies
	policies, err := m.repo.ListPolicies(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}

	if len(policies) == 0 {
		m.logger.Debug("no enabled policies to evaluate")
		return nil
	}

	m.logger.Debug("evaluating policies", "count", len(policies))

	for i := range policies {
		if err := m.evaluatePolicy(ctx, &policies[i]); err != nil {
			m.logger.Error("failed to evaluate policy",
				"policy_id", policies[i].ID,
				"policy_name", policies[i].Name,
				"error", err,
			)
		}
	}

	return nil
}

// evaluatePolicy evaluates a single scaling policy.
func (m *Manager) evaluatePolicy(ctx context.Context, policy *Policy) error {
	// Load rules for the policy
	rules, err := m.repo.GetRulesForPolicy(ctx, policy.ID)
	if err != nil {
		return fmt.Errorf("failed to get rules: %w", err)
	}

	// Separate rules by type
	for i := range rules {
		if rules[i].RuleType == RuleTypeScaleUp {
			policy.ScaleUpRules = append(policy.ScaleUpRules, rules[i])
		} else {
			policy.ScaleDownRules = append(policy.ScaleDownRules, rules[i])
		}
	}

	// Get current state
	state, err := m.repo.GetState(ctx, policy.ID)
	if err != nil {
		// Create initial state if not found
		state = &State{
			PolicyID:        policy.ID,
			CurrentReplicas: policy.MinReplicas,
		}
		if _, createErr := m.repo.CreateState(ctx, state); createErr != nil {
			m.logger.Warn("failed to create initial state", "error", createErr)
		}
	}

	// Sync current replicas from executor
	currentReplicas, err := m.executor.GetCurrentReplicas(ctx, policy.TargetType, policy.TargetID)
	if err != nil {
		m.logger.Warn("failed to get current replicas from executor, using state",
			"policy_id", policy.ID,
			"error", err,
		)
	} else {
		state.CurrentReplicas = currentReplicas
	}

	// Evaluate the policy
	decision, err := m.evaluator.EvaluatePolicy(ctx, policy, state)
	if err != nil {
		return fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Execute the decision if needed
	if decision.ShouldExecute {
		if err := m.executeDecision(ctx, decision, state); err != nil {
			return fmt.Errorf("failed to execute decision: %w", err)
		}
	} else {
		m.logger.Debug("no scaling needed",
			"policy_id", policy.ID,
			"policy_name", policy.Name,
			"reason", decision.Reason,
		)
	}

	// Update state with pending conditions
	if err := m.repo.UpdateState(ctx, state); err != nil {
		m.logger.Warn("failed to update state", "error", err)
	}

	return nil
}

// executeDecision executes a scaling decision.
func (m *Manager) executeDecision(ctx context.Context, decision *Decision, state *State) error {
	policy := decision.Policy

	m.logger.Info("executing scaling decision",
		"policy_id", policy.ID,
		"policy_name", policy.Name,
		"action", decision.Action,
		"current_replicas", decision.CurrentReplicas,
		"desired_replicas", decision.DesiredReplicas,
		"reason", decision.Reason,
	)

	// Execute the scaling action
	if err := m.executor.Scale(ctx, policy.TargetType, policy.TargetID, decision.DesiredReplicas, false); err != nil {
		return fmt.Errorf("failed to scale: %w", err)
	}

	// Record history
	history := &History{
		PolicyID:         &policy.ID,
		PolicyName:       policy.Name,
		Action:           decision.Action,
		TargetType:       policy.TargetType,
		TargetID:         policy.TargetID,
		PreviousReplicas: decision.CurrentReplicas,
		NewReplicas:      decision.DesiredReplicas,
		Reason:           decision.Reason,
		TriggeredBy:      decision.TriggeredBy,
		DryRun:           false,
		ExecutedAt:       time.Now(),
	}

	if _, err := m.repo.CreateHistory(ctx, history); err != nil {
		m.logger.Warn("failed to create history", "error", err)
	}

	// Update state
	now := time.Now()
	state.CurrentReplicas = decision.DesiredReplicas
	state.LastScaleTime = &now
	state.LastScaleAction = string(decision.Action)

	// Clear pending conditions that triggered
	if state.PendingConditions != nil {
		for key := range state.PendingConditions {
			delete(state.PendingConditions, key)
		}
	}

	if err := m.repo.UpdateState(ctx, state); err != nil {
		m.logger.Warn("failed to update state after scaling", "error", err)
	}

	return nil
}

// EvaluateNow triggers an immediate evaluation of all policies.
func (m *Manager) EvaluateNow(ctx context.Context) error {
	return m.evaluatePolicies(ctx)
}

// EvaluatePolicyNow triggers an immediate evaluation of a specific policy.
// If dryRun is true, the scaling action is not executed.
func (m *Manager) EvaluatePolicyNow(ctx context.Context, policyID string, dryRun bool) (*Decision, error) {
	// Parse the policy ID
	policy, err := m.repo.GetPolicyByName(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	// Load rules
	rules, err := m.repo.GetRulesForPolicy(ctx, policy.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	for i := range rules {
		if rules[i].RuleType == RuleTypeScaleUp {
			policy.ScaleUpRules = append(policy.ScaleUpRules, rules[i])
		} else {
			policy.ScaleDownRules = append(policy.ScaleDownRules, rules[i])
		}
	}

	// Get current state
	state, err := m.repo.GetState(ctx, policy.ID)
	if err != nil {
		state = &State{
			PolicyID:        policy.ID,
			CurrentReplicas: policy.MinReplicas,
		}
	}

	// Sync current replicas from executor
	currentReplicas, err := m.executor.GetCurrentReplicas(ctx, policy.TargetType, policy.TargetID)
	if err == nil {
		state.CurrentReplicas = currentReplicas
	}

	// Evaluate the policy
	decision, err := m.evaluator.EvaluatePolicy(ctx, policy, state)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Execute if not dry run
	if decision.ShouldExecute && !dryRun {
		if err := m.executeDecision(ctx, decision, state); err != nil {
			return nil, fmt.Errorf("failed to execute decision: %w", err)
		}
	} else if decision.ShouldExecute && dryRun {
		// Record dry-run history
		history := &History{
			PolicyID:         &policy.ID,
			PolicyName:       policy.Name,
			Action:           decision.Action,
			TargetType:       policy.TargetType,
			TargetID:         policy.TargetID,
			PreviousReplicas: decision.CurrentReplicas,
			NewReplicas:      decision.DesiredReplicas,
			Reason:           decision.Reason,
			TriggeredBy:      decision.TriggeredBy,
			DryRun:           true,
			ExecutedAt:       time.Now(),
		}

		if _, err := m.repo.CreateHistory(ctx, history); err != nil {
			m.logger.Warn("failed to create dry-run history", "error", err)
		}
	}

	return decision, nil
}

// Evaluator returns the underlying evaluator (useful for testing).
func (m *Manager) Evaluator() *Evaluator {
	return m.evaluator
}

// Executor returns the underlying executor (useful for testing).
func (m *Manager) Executor() Executor {
	return m.executor
}
