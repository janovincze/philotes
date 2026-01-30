// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/installer"
)

// InstallerService provides business logic for deployment operations.
type InstallerService struct {
	repo   *repositories.DeploymentRepository
	logger *slog.Logger
}

// NewInstallerService creates a new InstallerService.
func NewInstallerService(
	repo *repositories.DeploymentRepository,
	logger *slog.Logger,
) *InstallerService {
	return &InstallerService{
		repo:   repo,
		logger: logger.With("component", "installer-service"),
	}
}

// GetProviders returns all supported cloud providers.
func (s *InstallerService) GetProviders(_ context.Context) []models.Provider {
	return installer.GetProviders()
}

// GetProvider returns a single provider by ID.
func (s *InstallerService) GetProvider(_ context.Context, providerID string) (*models.Provider, error) {
	provider := installer.GetProvider(providerID)
	if provider == nil {
		return nil, &NotFoundError{Resource: "provider", ID: providerID}
	}
	return provider, nil
}

// CreateDeployment creates a new deployment.
func (s *InstallerService) CreateDeployment(ctx context.Context, req *models.CreateDeploymentRequest, userID *uuid.UUID) (*models.Deployment, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Validate provider exists
	if !installer.ValidateProvider(req.Provider) {
		return nil, &ValidationError{
			Errors: []models.FieldError{
				{Field: "provider", Message: "unsupported provider"},
			},
		}
	}

	// Validate region for provider
	if !installer.ValidateRegion(req.Provider, req.Region) {
		return nil, &ValidationError{
			Errors: []models.FieldError{
				{Field: "region", Message: "invalid region for provider"},
			},
		}
	}

	// Get size configuration for cost estimation
	sizeConfig := installer.GetSizeConfig(req.Provider, req.Size)
	if sizeConfig == nil {
		return nil, &ValidationError{
			Errors: []models.FieldError{
				{Field: "size", Message: "invalid size for provider"},
			},
		}
	}

	// Create deployment model
	deployment := &models.Deployment{
		UserID:      userID,
		Name:        req.Name,
		Provider:    req.Provider,
		Region:      req.Region,
		Size:        req.Size,
		Status:      models.DeploymentStatusPending,
		Environment: req.Environment,
		Config: &models.DeploymentConfig{
			Domain:        req.Domain,
			SSHPublicKey:  req.SSHPublicKey,
			ChartVersion:  req.ChartVersion,
			WorkerCount:   sizeConfig.WorkerCount,
			StorageSizeGB: sizeConfig.StorageSizeGB,
		},
	}

	// Override worker count and storage if specified in request
	if req.WorkerCount > 0 {
		deployment.Config.WorkerCount = req.WorkerCount
	}
	if req.StorageSizeGB > 0 {
		deployment.Config.StorageSizeGB = req.StorageSizeGB
	}

	// Create in database
	created, err := s.repo.Create(ctx, deployment)
	if err != nil {
		if errors.Is(err, repositories.ErrDeploymentNameExists) {
			return nil, &ConflictError{Message: "deployment with this name already exists"}
		}
		s.logger.Error("failed to create deployment", "error", err)
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	s.logger.Info("deployment created",
		"id", created.ID,
		"name", created.Name,
		"provider", created.Provider,
		"region", created.Region,
		"size", created.Size,
	)

	return created, nil
}

// GetDeployment retrieves a deployment by ID.
func (s *InstallerService) GetDeployment(ctx context.Context, id uuid.UUID) (*models.Deployment, error) {
	deployment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrDeploymentNotFound) {
			return nil, &NotFoundError{Resource: "deployment", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return deployment, nil
}

// ListDeployments retrieves all deployments, optionally filtered by user ID.
func (s *InstallerService) ListDeployments(ctx context.Context, userID *uuid.UUID) ([]models.Deployment, error) {
	deployments, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	if deployments == nil {
		deployments = []models.Deployment{}
	}
	return deployments, nil
}

// CancelDeployment cancels a deployment.
func (s *InstallerService) CancelDeployment(ctx context.Context, id uuid.UUID) error {
	// Get deployment
	deployment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrDeploymentNotFound) {
			return &NotFoundError{Resource: "deployment", ID: id.String()}
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Check if can be canceled
	if deployment.Status == models.DeploymentStatusCompleted {
		return &ConflictError{Message: "cannot cancel a completed deployment"}
	}
	if deployment.Status == models.DeploymentStatusCancelled {
		return &ConflictError{Message: "deployment is already canceled"}
	}
	if deployment.Status == models.DeploymentStatusFailed {
		return &ConflictError{Message: "deployment has already failed"}
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, id, models.DeploymentStatusCancelled, ""); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Add cancellation log
	if err := s.repo.AddLog(ctx, id, "info", "canceled", "Deployment canceled by user"); err != nil {
		s.logger.Warn("failed to add cancellation log", "deployment_id", id, "error", err)
	}

	s.logger.Info("deployment canceled", "id", id)
	return nil
}

// DeleteDeployment deletes a deployment.
func (s *InstallerService) DeleteDeployment(ctx context.Context, id uuid.UUID) error {
	// Get deployment to check status
	deployment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrDeploymentNotFound) {
			return &NotFoundError{Resource: "deployment", ID: id.String()}
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Don't allow deletion of active deployments
	if deployment.Status == models.DeploymentStatusProvisioning ||
		deployment.Status == models.DeploymentStatusConfiguring ||
		deployment.Status == models.DeploymentStatusDeploying ||
		deployment.Status == models.DeploymentStatusVerifying {
		return &ConflictError{Message: "cannot delete an active deployment"}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrDeploymentNotFound) {
			return &NotFoundError{Resource: "deployment", ID: id.String()}
		}
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	s.logger.Info("deployment deleted", "id", id)
	return nil
}

// GetDeploymentLogs retrieves logs for a deployment.
func (s *InstallerService) GetDeploymentLogs(ctx context.Context, deploymentID uuid.UUID, limit int) ([]models.DeploymentLog, error) {
	// Verify deployment exists
	_, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, repositories.ErrDeploymentNotFound) {
			return nil, &NotFoundError{Resource: "deployment", ID: deploymentID.String()}
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	logs, err := s.repo.GetLogs(ctx, deploymentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}
	if logs == nil {
		logs = []models.DeploymentLog{}
	}
	return logs, nil
}

// GetCostEstimate calculates the cost estimate for a deployment configuration.
func (s *InstallerService) GetCostEstimate(_ context.Context, providerID string, sizeID models.DeploymentSize) (*models.CostEstimate, error) {
	provider := installer.GetProvider(providerID)
	if provider == nil {
		return nil, &NotFoundError{Resource: "provider", ID: providerID}
	}

	sizeConfig := installer.GetSizeConfig(providerID, sizeID)
	if sizeConfig == nil {
		return nil, &ValidationError{
			Errors: []models.FieldError{
				{Field: "size", Message: "invalid size for provider"},
			},
		}
	}

	// Build cost estimate based on size configuration
	estimate := &models.CostEstimate{
		Provider: providerID,
		Size:     string(sizeID),
		Total:    sizeConfig.MonthlyCostEUR,
		Currency: "EUR",
	}

	// Calculate component costs (simplified - actual breakdown may vary)
	// These are rough estimates based on typical distributions
	estimate.ControlPlane = sizeConfig.MonthlyCostEUR * 0.2 // ~20% for control plane
	estimate.Workers = sizeConfig.MonthlyCostEUR * 0.6      // ~60% for workers
	estimate.Storage = sizeConfig.MonthlyCostEUR * 0.1      // ~10% for storage
	estimate.LoadBalancer = sizeConfig.MonthlyCostEUR * 0.1 // ~10% for LB

	return estimate, nil
}

// RetryDeployment initiates a retry of a failed deployment.
func (s *InstallerService) RetryDeployment(ctx context.Context, id uuid.UUID, deployment *models.Deployment, orchestrator *installer.DeploymentOrchestrator) error {
	// Check if deployment can be retried (should be failed)
	if deployment.Status != models.DeploymentStatusFailed {
		return &ConflictError{Message: "only failed deployments can be retried"}
	}

	// Update status to pending
	if err := s.repo.UpdateStatus(ctx, id, models.DeploymentStatusPending, ""); err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Add retry log
	if err := s.repo.AddLog(ctx, id, "info", "retry", "Retrying deployment"); err != nil {
		s.logger.Warn("failed to add retry log", "deployment_id", id, "error", err)
	}

	// Build deployment config for orchestrator
	cfg := &installer.DeploymentConfig{
		DeploymentID: id,
		StackName:    deployment.PulumiStackName,
		Provider:     deployment.Provider,
		Region:       deployment.Region,
		Environment:  deployment.Environment,
		Size:         deployment.Size,
		Config:       deployment.Config,
	}

	// Create status callback to update database
	statusCallback := func(status string, err error) {
		dbStatus := models.DeploymentStatus(status)
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		if updateErr := s.repo.UpdateStatus(ctx, id, dbStatus, errMsg); updateErr != nil {
			s.logger.Error("failed to update deployment status", "deployment_id", id, "error", updateErr)
		}
	}

	// Start retry
	if err := orchestrator.RetryDeployment(ctx, id, cfg, statusCallback); err != nil {
		// Revert to failed status
		if revertErr := s.repo.UpdateStatus(ctx, id, models.DeploymentStatusFailed, err.Error()); revertErr != nil {
			s.logger.Error("failed to revert deployment status", "deployment_id", id, "error", revertErr)
		}
		return fmt.Errorf("failed to start retry: %w", err)
	}

	s.logger.Info("deployment retry initiated", "id", id)
	return nil
}
