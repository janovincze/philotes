// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
)

// PipelineService provides business logic for pipeline operations.
type PipelineService struct {
	repo       *repositories.PipelineRepository
	sourceRepo *repositories.SourceRepository
	logger     *slog.Logger
}

// NewPipelineService creates a new PipelineService.
func NewPipelineService(
	repo *repositories.PipelineRepository,
	sourceRepo *repositories.SourceRepository,
	logger *slog.Logger,
) *PipelineService {
	return &PipelineService{
		repo:       repo,
		sourceRepo: sourceRepo,
		logger:     logger.With("component", "pipeline-service"),
	}
}

// Create creates a new pipeline.
func (s *PipelineService) Create(ctx context.Context, req *models.CreatePipelineRequest) (*models.Pipeline, error) {
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		return nil, &ValidationError{Errors: errors}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Verify source exists
	_, err := s.sourceRepo.GetByID(ctx, req.SourceID)
	if err != nil {
		if errors.Is(err, repositories.ErrSourceNotFound) {
			return nil, &NotFoundError{Resource: "source", ID: req.SourceID.String()}
		}
		return nil, fmt.Errorf("failed to verify source: %w", err)
	}

	// Create pipeline
	pipeline, err := s.repo.Create(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNameExists) {
			return nil, &ConflictError{Message: "pipeline with this name already exists"}
		}
		if errors.Is(err, repositories.ErrTableMappingExists) {
			return nil, &ConflictError{Message: "duplicate table mapping in request"}
		}
		s.logger.Error("failed to create pipeline", "error", err)
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	s.logger.Info("pipeline created", "id", pipeline.ID, "name", pipeline.Name, "source_id", pipeline.SourceID)
	return pipeline, nil
}

// Get retrieves a pipeline by ID.
func (s *PipelineService) Get(ctx context.Context, id uuid.UUID) (*models.Pipeline, error) {
	pipeline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return nil, &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}
	return pipeline, nil
}

// List retrieves all pipelines.
func (s *PipelineService) List(ctx context.Context) ([]models.Pipeline, error) {
	pipelines, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	if pipelines == nil {
		pipelines = []models.Pipeline{}
	}
	return pipelines, nil
}

// Update updates a pipeline.
func (s *PipelineService) Update(ctx context.Context, id uuid.UUID, req *models.UpdatePipelineRequest) (*models.Pipeline, error) {
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		return nil, &ValidationError{Errors: errors}
	}

	// Check pipeline exists and is not running
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return nil, &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if existing.Status == models.PipelineStatusRunning || existing.Status == models.PipelineStatusStarting {
		return nil, &ConflictError{Message: "cannot update a running pipeline"}
	}

	// Update pipeline
	pipeline, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNameExists) {
			return nil, &ConflictError{Message: "pipeline with this name already exists"}
		}
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	s.logger.Info("pipeline updated", "id", pipeline.ID, "name", pipeline.Name)
	return pipeline, nil
}

// Delete deletes a pipeline.
func (s *PipelineService) Delete(ctx context.Context, id uuid.UUID) error {
	// Check pipeline exists and is not running
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	if existing.Status == models.PipelineStatusRunning || existing.Status == models.PipelineStatusStarting {
		return &ConflictError{Message: "cannot delete a running pipeline"}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}

	s.logger.Info("pipeline deleted", "id", id)
	return nil
}

// Start starts a pipeline.
func (s *PipelineService) Start(ctx context.Context, id uuid.UUID) error {
	// Get pipeline
	pipeline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	// Check if already running
	if pipeline.Status == models.PipelineStatusRunning || pipeline.Status == models.PipelineStatusStarting {
		return &ConflictError{Message: "pipeline is already running"}
	}

	// Update status to starting
	if err := s.repo.UpdateStatus(ctx, id, models.PipelineStatusStarting, ""); err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	// TODO: In the future, this will integrate with the CDC pipeline orchestrator
	// For now, we just update the status to running
	if err := s.repo.UpdateStatus(ctx, id, models.PipelineStatusRunning, ""); err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	s.logger.Info("pipeline started", "id", id, "name", pipeline.Name)
	return nil
}

// Stop stops a pipeline.
func (s *PipelineService) Stop(ctx context.Context, id uuid.UUID) error {
	// Get pipeline
	pipeline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	// Check if already stopped
	if pipeline.Status == models.PipelineStatusStopped || pipeline.Status == models.PipelineStatusStopping {
		return &ConflictError{Message: "pipeline is already stopped"}
	}

	// Update status to stopping
	if err := s.repo.UpdateStatus(ctx, id, models.PipelineStatusStopping, ""); err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	// TODO: In the future, this will integrate with the CDC pipeline orchestrator
	// For now, we just update the status to stopped
	if err := s.repo.UpdateStatus(ctx, id, models.PipelineStatusStopped, ""); err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	s.logger.Info("pipeline stopped", "id", id, "name", pipeline.Name)
	return nil
}

// GetStatus gets the status of a pipeline.
func (s *PipelineService) GetStatus(ctx context.Context, id uuid.UUID) (*models.PipelineStatusResponse, error) {
	pipeline, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return nil, &NotFoundError{Resource: "pipeline", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	status := &models.PipelineStatusResponse{
		ID:           pipeline.ID,
		Name:         pipeline.Name,
		Status:       pipeline.Status,
		ErrorMessage: pipeline.ErrorMessage,
		StartedAt:    pipeline.StartedAt,
	}

	// Calculate uptime if running
	if pipeline.Status == models.PipelineStatusRunning && pipeline.StartedAt != nil {
		uptime := time.Since(*pipeline.StartedAt)
		status.Uptime = formatDuration(uptime)
	}

	// TODO: Get actual event counts from CDC pipeline when integrated

	return status, nil
}

// AddTableMapping adds a table mapping to a pipeline.
func (s *PipelineService) AddTableMapping(ctx context.Context, pipelineID uuid.UUID, req *models.AddTableMappingRequest) (*models.TableMapping, error) {
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		return nil, &ValidationError{Errors: errors}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Check pipeline exists and is not running
	pipeline, err := s.repo.GetByID(ctx, pipelineID)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return nil, &NotFoundError{Resource: "pipeline", ID: pipelineID.String()}
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	if pipeline.Status == models.PipelineStatusRunning || pipeline.Status == models.PipelineStatusStarting {
		return nil, &ConflictError{Message: "cannot modify a running pipeline"}
	}

	// Add table mapping
	mapping, err := s.repo.AddTableMapping(ctx, pipelineID, req)
	if err != nil {
		if errors.Is(err, repositories.ErrTableMappingExists) {
			return nil, &ConflictError{Message: "table mapping already exists"}
		}
		return nil, fmt.Errorf("failed to add table mapping: %w", err)
	}

	s.logger.Info("table mapping added",
		"pipeline_id", pipelineID,
		"schema", mapping.SourceSchema,
		"table", mapping.SourceTable,
	)

	return mapping, nil
}

// RemoveTableMapping removes a table mapping from a pipeline.
func (s *PipelineService) RemoveTableMapping(ctx context.Context, pipelineID, mappingID uuid.UUID) error {
	// Check pipeline exists and is not running
	pipeline, err := s.repo.GetByID(ctx, pipelineID)
	if err != nil {
		if errors.Is(err, repositories.ErrPipelineNotFound) {
			return &NotFoundError{Resource: "pipeline", ID: pipelineID.String()}
		}
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	if pipeline.Status == models.PipelineStatusRunning || pipeline.Status == models.PipelineStatusStarting {
		return &ConflictError{Message: "cannot modify a running pipeline"}
	}

	if err := s.repo.RemoveTableMapping(ctx, pipelineID, mappingID); err != nil {
		return fmt.Errorf("failed to remove table mapping: %w", err)
	}

	s.logger.Info("table mapping removed", "pipeline_id", pipelineID, "mapping_id", mappingID)
	return nil
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
