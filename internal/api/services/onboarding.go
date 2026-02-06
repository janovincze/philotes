// Package services provides business logic for API endpoints.
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/cdc/health"
)

// OnboardingService errors.
var (
	ErrOnboardingNotFound     = errors.New("onboarding progress not found")
	ErrRegistrationDisabled   = errors.New("user registration is disabled")
	ErrDataVerificationFailed = errors.New("data verification failed")
)

// OnboardingService provides business logic for the onboarding wizard.
type OnboardingService struct {
	onboardingRepo *repositories.OnboardingRepository
	userRepo       *repositories.UserRepository
	healthManager  *health.Manager
	queryService   *QueryService
	logger         *slog.Logger
}

// NewOnboardingService creates a new OnboardingService.
func NewOnboardingService(
	onboardingRepo *repositories.OnboardingRepository,
	userRepo *repositories.UserRepository,
	healthManager *health.Manager,
	queryService *QueryService,
	logger *slog.Logger,
) *OnboardingService {
	if logger == nil {
		logger = slog.Default()
	}

	return &OnboardingService{
		onboardingRepo: onboardingRepo,
		userRepo:       userRepo,
		healthManager:  healthManager,
		queryService:   queryService,
		logger:         logger.With("component", "onboarding-service"),
	}
}

// GetClusterHealth returns extended health information for onboarding.
func (s *OnboardingService) GetClusterHealth(ctx context.Context) *models.ClusterHealthResponse {
	response := &models.ClusterHealthResponse{
		Overall:    "healthy",
		Components: make(map[string]models.ComponentHealth),
		Timestamp:  time.Now(),
		APIReady:   true, // If we're responding, API is ready
	}

	// If no health manager is configured, return basic healthy response
	if s.healthManager == nil {
		response.AllCriticalReady = true
		return response
	}

	// Get health status from health manager
	status := s.healthManager.GetOverallStatus(ctx)
	response.Overall = string(status.Status)

	// Convert component health
	for name, result := range status.Components {
		componentHealth := models.ComponentHealth{
			Name:       result.Name,
			Status:     string(result.Status),
			Message:    result.Message,
			DurationMs: result.Duration.Milliseconds(),
			LastCheck:  result.LastCheck,
		}
		if result.Error != "" {
			componentHealth.Error = result.Error
		}
		response.Components[name] = componentHealth

		// Set specific ready flags based on component names
		switch name {
		case "buffer_db", "buffer-db", "postgres", "postgresql":
			response.BufferDBReady = result.Status == health.StatusHealthy
		case "minio", "object_storage", "s3":
			response.MinIOReady = result.Status == health.StatusHealthy
		case "lakekeeper", "iceberg_catalog", "catalog":
			response.LakekeeperReady = result.Status == health.StatusHealthy
		}
	}

	// If no components registered, assume all are ready
	if len(status.Components) == 0 {
		response.BufferDBReady = true
		response.MinIOReady = true
		response.LakekeeperReady = true
	}

	// Determine if all critical components are ready
	response.AllCriticalReady = response.APIReady && response.BufferDBReady

	return response
}

// GetProgress retrieves onboarding progress.
func (s *OnboardingService) GetProgress(ctx context.Context, userID *uuid.UUID, sessionID string) (*models.OnboardingProgress, error) {
	// Try to get progress by user ID first
	if userID != nil {
		progress, err := s.onboardingRepo.GetByUserID(ctx, *userID)
		if err == nil {
			return progress, nil
		}
		if !errors.Is(err, repositories.ErrOnboardingNotFound) {
			return nil, err
		}
	}

	// Try to get progress by session ID
	if sessionID != "" {
		progress, err := s.onboardingRepo.GetBySessionID(ctx, sessionID)
		if err == nil {
			return progress, nil
		}
		if !errors.Is(err, repositories.ErrOnboardingNotFound) {
			return nil, err
		}
	}

	return nil, ErrOnboardingNotFound
}

// CreateProgress creates a new onboarding progress record.
func (s *OnboardingService) CreateProgress(ctx context.Context, userID *uuid.UUID, sessionID string) (*models.OnboardingProgress, error) {
	progress, err := s.onboardingRepo.Create(ctx, userID, sessionID)
	if err != nil {
		s.logger.Error("failed to create onboarding progress", "error", err)
		return nil, err
	}

	s.logger.Info("created onboarding progress",
		"progress_id", progress.ID,
		"user_id", userID,
		"session_id", sessionID,
	)

	return progress, nil
}

// SaveProgress saves onboarding progress.
func (s *OnboardingService) SaveProgress(ctx context.Context, progressID uuid.UUID, req *models.SaveOnboardingProgressRequest) (*models.OnboardingProgress, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	progress, err := s.onboardingRepo.Update(ctx, progressID, req)
	if err != nil {
		if errors.Is(err, repositories.ErrOnboardingNotFound) {
			return nil, ErrOnboardingNotFound
		}
		s.logger.Error("failed to update onboarding progress",
			"progress_id", progressID,
			"error", err,
		)
		return nil, err
	}

	s.logger.Debug("updated onboarding progress",
		"progress_id", progressID,
		"current_step", req.CurrentStep,
		"completed_steps", req.CompletedSteps,
	)

	return progress, nil
}

// CompleteOnboarding marks the onboarding as complete.
func (s *OnboardingService) CompleteOnboarding(ctx context.Context, progressID uuid.UUID) (*models.OnboardingProgress, error) {
	progress, err := s.onboardingRepo.Complete(ctx, progressID)
	if err != nil {
		if errors.Is(err, repositories.ErrOnboardingNotFound) {
			return nil, ErrOnboardingNotFound
		}
		s.logger.Error("failed to complete onboarding",
			"progress_id", progressID,
			"error", err,
		)
		return nil, err
	}

	s.logger.Info("completed onboarding",
		"progress_id", progressID,
		"total_time_ms", progress.Metrics.TotalTimeMs,
	)

	return progress, nil
}

// AssociateUser associates a user with an onboarding session.
func (s *OnboardingService) AssociateUser(ctx context.Context, progressID, userID uuid.UUID) error {
	err := s.onboardingRepo.AssociateUser(ctx, progressID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrOnboardingNotFound) {
			return ErrOnboardingNotFound
		}
		s.logger.Error("failed to associate user with onboarding",
			"progress_id", progressID,
			"user_id", userID,
			"error", err,
		)
		return err
	}

	s.logger.Info("associated user with onboarding",
		"progress_id", progressID,
		"user_id", userID,
	)

	return nil
}

// CheckAdminExists checks if an admin user already exists.
func (s *OnboardingService) CheckAdminExists(ctx context.Context) (bool, error) {
	return s.userRepo.HasAdminUser(ctx)
}

// VerifyDataFlow verifies that data is flowing to Iceberg by querying via Trino.
func (s *OnboardingService) VerifyDataFlow(ctx context.Context, req *models.DataVerificationRequest) (*models.DataVerificationResponse, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	maxWait := 60
	if req.MaxWaitSec > 0 {
		maxWait = req.MaxWaitSec
	}
	if maxWait > 300 {
		maxWait = 300
	}

	startTime := time.Now()

	s.logger.Info("verifying data flow",
		"pipeline_id", req.PipelineID,
		"table_name", req.TableName,
		"max_wait_sec", maxWait,
	)

	// If Trino query service is not available, return a graceful placeholder
	if s.queryService == nil {
		s.logger.Warn("query service not available, returning placeholder verification")
		return &models.DataVerificationResponse{
			Success:      false,
			RowCount:     0,
			QueryTimeMs:  time.Since(startTime).Milliseconds(),
			ErrorMessage: "Query layer (Trino) is not configured; cannot verify data flow",
		}, nil
	}

	// Validate table name to prevent SQL injection.
	// TableName may be qualified (catalog.schema.table), so validate each segment.
	for _, part := range strings.Split(req.TableName, ".") {
		if err := validateIdentifier(part, "table"); err != nil {
			return nil, &ValidationError{Errors: []models.FieldError{{Field: "table_name", Message: err.Error()}}}
		}
	}

	// Build a count query for the target table
	sql := "SELECT COUNT(*) as row_count FROM " + req.TableName

	// Poll with 5-second intervals until data is found or maxWait is exceeded
	pollInterval := 5 * time.Second
	deadline := startTime.Add(time.Duration(maxWait) * time.Second)

	for {
		queryReq := &models.QueryExecuteRequest{
			SQL:   sql,
			Limit: 1,
		}

		result, err := s.queryService.ExecuteUserQuery(ctx, queryReq)
		if err != nil {
			s.logger.Warn("verification query failed", "error", err)
		} else if result.Error == "" && result.RowCount > 0 && len(result.Rows) > 0 {
			// Extract the count from the result
			rowCount := int64(0)
			if cnt, ok := result.Rows[0]["row_count"]; ok {
				switch v := cnt.(type) {
				case float64:
					rowCount = int64(v)
				case int64:
					rowCount = v
				}
			}

			if rowCount > 0 {
				// Data found — fetch sample rows
				sampleReq := &models.QueryExecuteRequest{
					SQL:   "SELECT * FROM " + req.TableName,
					Limit: 10,
				}
				sampleResult, sampleErr := s.queryService.ExecuteUserQuery(ctx, sampleReq)
				var sampleRows []map[string]any
				if sampleErr == nil && sampleResult.Error == "" {
					sampleRows = sampleResult.Rows
				}

				return &models.DataVerificationResponse{
					Success:     true,
					RowCount:    rowCount,
					SampleRows:  sampleRows,
					QueryTimeMs: time.Since(startTime).Milliseconds(),
				}, nil
			}
		}

		// Check if we've exceeded the deadline
		if time.Now().After(deadline) {
			return &models.DataVerificationResponse{
				Success:      false,
				RowCount:     0,
				QueryTimeMs:  time.Since(startTime).Milliseconds(),
				ErrorMessage: "No data found in table after waiting",
			}, nil
		}

		// Wait before next poll
		select {
		case <-ctx.Done():
			return &models.DataVerificationResponse{
				Success:      false,
				RowCount:     0,
				QueryTimeMs:  time.Since(startTime).Milliseconds(),
				ErrorMessage: "Verification canceled",
			}, nil
		case <-time.After(pollInterval):
			// Continue polling
		}
	}
}

// GetOrCreateProgress gets existing progress or creates new one.
func (s *OnboardingService) GetOrCreateProgress(ctx context.Context, userID *uuid.UUID, sessionID string) (*models.OnboardingProgress, error) {
	// Try to get existing progress
	progress, err := s.GetProgress(ctx, userID, sessionID)
	if err == nil {
		return progress, nil
	}
	if !errors.Is(err, ErrOnboardingNotFound) {
		return nil, err
	}

	// Create new progress
	return s.CreateProgress(ctx, userID, sessionID)
}
