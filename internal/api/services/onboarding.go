// Package services provides business logic for API endpoints.
package services

import (
	"context"
	"errors"
	"fmt"
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

// VerifyDataFlow verifies that data is flowing to Iceberg.
// It queries Trino to check if data exists in the specified table.
func (s *OnboardingService) VerifyDataFlow(ctx context.Context, req *models.DataVerificationRequest) (*models.DataVerificationResponse, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	// Set default max wait time
	maxWait := 60
	if req.MaxWaitSec > 0 {
		maxWait = req.MaxWaitSec
	}

	startTime := time.Now()

	s.logger.Info("verifying data flow",
		"pipeline_id", req.PipelineID,
		"table_name", req.TableName,
		"max_wait_sec", maxWait,
	)

	// Check if query service is available
	if s.queryService == nil {
		return &models.DataVerificationResponse{
			Success:      false,
			RowCount:     0,
			QueryTimeMs:  time.Since(startTime).Milliseconds(),
			ErrorMessage: "query service not configured",
		}, nil
	}

	// Parse table name (expected format: catalog.schema.table or schema.table)
	catalog, schema, table := s.parseTableName(req.TableName)

	// Poll for data with retries
	pollInterval := 2 * time.Second
	deadline := time.Now().Add(time.Duration(maxWait) * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		// Build count query
		query := fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s.%s.%s", catalog, schema, table)

		queryReq := &models.ExecuteQueryRequest{
			Query:          query,
			TimeoutSeconds: 30,
			Catalog:        catalog,
			Schema:         schema,
		}

		result, err := s.queryService.ExecuteQuery(ctx, queryReq)
		if err != nil {
			lastErr = err
			s.logger.Debug("data verification query failed, retrying",
				"error", err,
				"table", req.TableName,
			)
			time.Sleep(pollInterval)
			continue
		}

		// Extract count from result
		if len(result.Data) > 0 && len(result.Data[0]) > 0 {
			count := s.extractCount(result.Data[0][0])
			if count > 0 {
				// Data found - get sample rows
				sampleRows, err := s.getSampleRows(ctx, catalog, schema, table)
				if err != nil {
					s.logger.Warn("failed to get sample rows", "error", err)
				}

				return &models.DataVerificationResponse{
					Success:     true,
					RowCount:    count,
					SampleRows:  sampleRows,
					QueryTimeMs: time.Since(startTime).Milliseconds(),
				}, nil
			}
		}

		// No data yet, wait and retry
		time.Sleep(pollInterval)
	}

	// Timeout - no data found
	errMsg := "no data found in table after waiting"
	if lastErr != nil {
		errMsg = fmt.Sprintf("no data found: %v", lastErr)
	}

	return &models.DataVerificationResponse{
		Success:      false,
		RowCount:     0,
		QueryTimeMs:  time.Since(startTime).Milliseconds(),
		ErrorMessage: errMsg,
	}, nil
}

// parseTableName parses a table name into catalog, schema, and table parts.
// Supports formats: catalog.schema.table, schema.table, or just table.
func (s *OnboardingService) parseTableName(tableName string) (catalog, schema, table string) {
	parts := strings.Split(tableName, ".")
	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2]
	case 2:
		return "iceberg", parts[0], parts[1]
	default:
		return "iceberg", "cdc", tableName
	}
}

// extractCount extracts an integer count from various interface types.
func (s *OnboardingService) extractCount(val interface{}) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		var count int64
		fmt.Sscanf(v, "%d", &count)
		return count
	default:
		return 0
	}
}

// getSampleRows retrieves sample rows from the table.
func (s *OnboardingService) getSampleRows(ctx context.Context, catalog, schema, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s.%s LIMIT 5", catalog, schema, table)

	queryReq := &models.ExecuteQueryRequest{
		Query:          query,
		TimeoutSeconds: 30,
		Catalog:        catalog,
		Schema:         schema,
	}

	result, err := s.queryService.ExecuteQuery(ctx, queryReq)
	if err != nil {
		return nil, err
	}

	// Convert to sample rows format
	sampleRows := make([]map[string]interface{}, 0, len(result.Data))
	for _, row := range result.Data {
		rowMap := make(map[string]interface{})
		for i, col := range result.Columns {
			if i < len(row) {
				rowMap[col.Name] = row[i]
			}
		}
		sampleRows = append(sampleRows, rowMap)
	}

	return sampleRows, nil
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
