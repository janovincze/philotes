// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"
)

// OnboardingStep represents a step in the onboarding wizard.
type OnboardingStep int

const (
	// StepClusterHealth verifies cluster components are healthy.
	StepClusterHealth OnboardingStep = 1
	// StepAdminUser creates the first admin user.
	StepAdminUser OnboardingStep = 2
	// StepSSOConfig configures SSO/OIDC (optional).
	StepSSOConfig OnboardingStep = 3
	// StepSourceDatabase connects the first source database.
	StepSourceDatabase OnboardingStep = 4
	// StepCreatePipeline creates the first CDC pipeline.
	StepCreatePipeline OnboardingStep = 5
	// StepVerifyData verifies data flow to Iceberg.
	StepVerifyData OnboardingStep = 6
	// StepAlerts configures alerting (optional).
	StepAlerts OnboardingStep = 7
)

// TotalOnboardingSteps is the total number of steps in the wizard.
const TotalOnboardingSteps = 7

// OptionalSteps are steps that can be skipped.
var OptionalSteps = map[OnboardingStep]bool{
	StepSSOConfig: true,
	StepAlerts:    true,
}

// OnboardingProgress tracks wizard state for resumability.
type OnboardingProgress struct {
	ID             uuid.UUID              `json:"id"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	CurrentStep    int                    `json:"current_step"`
	CompletedSteps []int                  `json:"completed_steps"`
	StepData       map[string]interface{} `json:"step_data"`
	Metrics        *OnboardingMetrics     `json:"metrics,omitempty"`
	StartedAt      time.Time              `json:"started_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// OnboardingMetrics tracks analytics for the wizard.
type OnboardingMetrics struct {
	TimePerStep  map[int]int64 `json:"time_per_step"`  // step -> milliseconds
	TotalTimeMs  int64         `json:"total_time_ms"`
	StepsSkipped []int         `json:"steps_skipped"`
}

// ClusterHealthResponse provides extended health information for onboarding.
// Note: ComponentHealth is defined in response.go
type ClusterHealthResponse struct {
	Overall    string                     `json:"overall"` // healthy, unhealthy, degraded
	Components map[string]ComponentHealth `json:"components"`

	// Summary flags for quick checks
	APIReady         bool `json:"api_ready"`
	BufferDBReady    bool `json:"buffer_db_ready"`
	MinIOReady       bool `json:"minio_ready"`
	LakekeeperReady  bool `json:"lakekeeper_ready"`
	AllCriticalReady bool `json:"all_critical_ready"`

	// K8s-specific (optional)
	PodsHealthy *int `json:"pods_healthy,omitempty"`
	PodsTotal   *int `json:"pods_total,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// GetOnboardingProgressRequest identifies the progress to retrieve.
type GetOnboardingProgressRequest struct {
	SessionID string `form:"session_id"`
}

// SaveOnboardingProgressRequest updates wizard progress.
type SaveOnboardingProgressRequest struct {
	SessionID      string                 `json:"session_id,omitempty"`
	CurrentStep    int                    `json:"current_step" binding:"required,min=1,max=7"`
	CompletedSteps []int                  `json:"completed_steps"`
	StepData       map[string]interface{} `json:"step_data"`
	StepSkipped    *int                   `json:"step_skipped,omitempty"`
	StepTimeMs     *int64                 `json:"step_time_ms,omitempty"`
}

// Validate validates the save progress request.
func (r *SaveOnboardingProgressRequest) Validate() []FieldError {
	var errors []FieldError
	if r.CurrentStep < 1 || r.CurrentStep > TotalOnboardingSteps {
		errors = append(errors, FieldError{
			Field:   "current_step",
			Message: "current_step must be between 1 and 7",
		})
	}
	for _, step := range r.CompletedSteps {
		if step < 1 || step > TotalOnboardingSteps {
			errors = append(errors, FieldError{
				Field:   "completed_steps",
				Message: "completed_steps must contain values between 1 and 7",
			})
			break
		}
	}
	return errors
}

// OnboardingProgressResponse wraps the progress for API response.
type OnboardingProgressResponse struct {
	Progress *OnboardingProgress `json:"progress"`
}

// DataVerificationRequest requests verification of data in Iceberg.
type DataVerificationRequest struct {
	PipelineID string `json:"pipeline_id" binding:"required"`
	TableName  string `json:"table_name" binding:"required"`
	MaxWaitSec int    `json:"max_wait_sec"` // defaults to 60
}

// Validate validates the data verification request.
func (r *DataVerificationRequest) Validate() []FieldError {
	var errors []FieldError
	if r.PipelineID == "" {
		errors = append(errors, FieldError{Field: "pipeline_id", Message: "pipeline_id is required"})
	}
	if r.TableName == "" {
		errors = append(errors, FieldError{Field: "table_name", Message: "table_name is required"})
	}
	return errors
}

// DataVerificationResponse shows results of querying Iceberg.
type DataVerificationResponse struct {
	Success      bool                     `json:"success"`
	RowCount     int64                    `json:"row_count"`
	SampleRows   []map[string]interface{} `json:"sample_rows,omitempty"`
	QueryTimeMs  int64                    `json:"query_time_ms"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Name            string `json:"name"`
	GenerateAPIKey  *bool  `json:"generate_api_key"` // defaults to true
}

// Validate validates the registration request.
func (r *RegisterRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Email == "" {
		errors = append(errors, FieldError{Field: "email", Message: "email is required"})
	}
	if len(r.Password) < 8 {
		errors = append(errors, FieldError{Field: "password", Message: "password must be at least 8 characters"})
	}
	if r.Password != r.ConfirmPassword {
		errors = append(errors, FieldError{Field: "confirm_password", Message: "passwords do not match"})
	}
	return errors
}

// RegisterResponse represents a registration response.
type RegisterResponse struct {
	User   *User   `json:"user"`
	Token  string  `json:"token"`
	APIKey *string `json:"api_key,omitempty"` // plaintext, shown once
}

// AdminExistsResponse indicates whether an admin user exists.
type AdminExistsResponse struct {
	Exists bool `json:"exists"`
}
