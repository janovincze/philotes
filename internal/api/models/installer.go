// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentStatus represents the status of a deployment.
type DeploymentStatus string

const (
	// DeploymentStatusPending indicates the deployment is queued.
	DeploymentStatusPending DeploymentStatus = "pending"
	// DeploymentStatusProvisioning indicates infrastructure is being created.
	DeploymentStatusProvisioning DeploymentStatus = "provisioning"
	// DeploymentStatusConfiguring indicates the cluster is being configured.
	DeploymentStatusConfiguring DeploymentStatus = "configuring"
	// DeploymentStatusDeploying indicates applications are being deployed.
	DeploymentStatusDeploying DeploymentStatus = "deploying"
	// DeploymentStatusVerifying indicates health checks are running.
	DeploymentStatusVerifying DeploymentStatus = "verifying"
	// DeploymentStatusCompleted indicates the deployment succeeded.
	DeploymentStatusCompleted DeploymentStatus = "completed"
	// DeploymentStatusFailed indicates the deployment failed.
	DeploymentStatusFailed DeploymentStatus = "failed"
	// DeploymentStatusCancelled indicates the deployment was cancelled.
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

// DeploymentSize represents the size preset for a deployment.
type DeploymentSize string

const (
	// DeploymentSizeSmall is a small deployment (~€30/month).
	DeploymentSizeSmall DeploymentSize = "small"
	// DeploymentSizeMedium is a medium deployment (~€60/month).
	DeploymentSizeMedium DeploymentSize = "medium"
	// DeploymentSizeLarge is a large deployment (~€150/month).
	DeploymentSizeLarge DeploymentSize = "large"
)

// Deployment represents a cloud infrastructure deployment.
type Deployment struct {
	ID              uuid.UUID         `json:"id"`
	UserID          *uuid.UUID        `json:"user_id,omitempty"`
	Name            string            `json:"name"`
	Provider        string            `json:"provider"`
	Region          string            `json:"region"`
	Size            DeploymentSize    `json:"size"`
	Status          DeploymentStatus  `json:"status"`
	Environment     string            `json:"environment"`
	Config          *DeploymentConfig `json:"config,omitempty"`
	Outputs         *DeploymentOutput `json:"outputs,omitempty"`
	PulumiStackName string            `json:"pulumi_stack_name,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// DeploymentConfig holds the configuration for a deployment.
type DeploymentConfig struct {
	Domain        string `json:"domain,omitempty"`
	SSHPublicKey  string `json:"ssh_public_key,omitempty"`
	ChartVersion  string `json:"chart_version,omitempty"`
	WorkerCount   int    `json:"worker_count,omitempty"`
	StorageSizeGB int    `json:"storage_size_gb,omitempty"`
}

// DeploymentOutput holds the outputs from a completed deployment.
type DeploymentOutput struct {
	ControlPlaneIP string `json:"control_plane_ip,omitempty"`
	LoadBalancerIP string `json:"load_balancer_ip,omitempty"`
	Kubeconfig     string `json:"kubeconfig,omitempty"`
	DashboardURL   string `json:"dashboard_url,omitempty"`
	APIURL         string `json:"api_url,omitempty"`
}

// DeploymentLog represents a log entry for a deployment.
type DeploymentLog struct {
	ID           int64     `json:"id"`
	DeploymentID uuid.UUID `json:"deployment_id"`
	Level        string    `json:"level"`
	Step         string    `json:"step,omitempty"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}

// Provider represents a cloud provider with its configuration.
type Provider struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	LogoURL     string           `json:"logo_url,omitempty"`
	Regions     []ProviderRegion `json:"regions"`
	Sizes       []ProviderSize   `json:"sizes"`
}

// ProviderRegion represents a region for a cloud provider.
type ProviderRegion struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Location    string `json:"location"`
	IsDefault   bool   `json:"is_default,omitempty"`
	IsAvailable bool   `json:"is_available"`
}

// ProviderSize represents a deployment size preset with pricing.
type ProviderSize struct {
	ID               DeploymentSize `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	MonthlyCostEUR   float64        `json:"monthly_cost_eur"`
	ControlPlaneType string         `json:"control_plane_type"`
	WorkerType       string         `json:"worker_type"`
	WorkerCount      int            `json:"worker_count"`
	StorageSizeGB    int            `json:"storage_size_gb"`
	VCPU             int            `json:"vcpu"`
	MemoryGB         int            `json:"memory_gb"`
}

// CreateDeploymentRequest represents a request to create a new deployment.
type CreateDeploymentRequest struct {
	Name          string               `json:"name" binding:"required,min=1,max=255"`
	Provider      string               `json:"provider" binding:"required"`
	Region        string               `json:"region" binding:"required"`
	Size          DeploymentSize       `json:"size" binding:"required"`
	Environment   string               `json:"environment,omitempty"`
	Domain        string               `json:"domain,omitempty"`
	SSHPublicKey  string               `json:"ssh_public_key,omitempty"`
	ChartVersion  string               `json:"chart_version,omitempty"`
	WorkerCount   int                  `json:"worker_count,omitempty"`
	StorageSizeGB int                  `json:"storage_size_gb,omitempty"`
	Credentials   *ProviderCredentials `json:"credentials,omitempty"`
}

// ProviderCredentials holds cloud provider authentication credentials.
type ProviderCredentials struct {
	// Hetzner
	HetznerToken string `json:"hetzner_token,omitempty"`

	// Scaleway
	ScalewayAccessKey string `json:"scaleway_access_key,omitempty"`
	ScalewaySecretKey string `json:"scaleway_secret_key,omitempty"`
	ScalewayProjectID string `json:"scaleway_project_id,omitempty"`

	// OVH
	OVHEndpoint          string `json:"ovh_endpoint,omitempty"`
	OVHApplicationKey    string `json:"ovh_application_key,omitempty"`
	OVHApplicationSecret string `json:"ovh_application_secret,omitempty"`
	OVHConsumerKey       string `json:"ovh_consumer_key,omitempty"`
	OVHServiceName       string `json:"ovh_service_name,omitempty"`

	// Exoscale
	ExoscaleAPIKey    string `json:"exoscale_api_key,omitempty"`
	ExoscaleAPISecret string `json:"exoscale_api_secret,omitempty"`

	// Contabo
	ContaboClientID     string `json:"contabo_client_id,omitempty"`
	ContaboClientSecret string `json:"contabo_client_secret,omitempty"`
	ContaboAPIUser      string `json:"contabo_api_user,omitempty"`
	ContaboAPIPassword  string `json:"contabo_api_password,omitempty"`
}

// Validate validates the create deployment request.
func (r *CreateDeploymentRequest) Validate() []FieldError {
	var errors []FieldError

	if r.Name == "" {
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	}

	validProviders := map[string]bool{
		"hetzner": true, "scaleway": true, "ovh": true, "exoscale": true, "contabo": true,
	}
	if !validProviders[r.Provider] {
		errors = append(errors, FieldError{Field: "provider", Message: "provider must be one of: hetzner, scaleway, ovh, exoscale, contabo"})
	}

	if r.Region == "" {
		errors = append(errors, FieldError{Field: "region", Message: "region is required"})
	}

	validSizes := map[DeploymentSize]bool{
		DeploymentSizeSmall: true, DeploymentSizeMedium: true, DeploymentSizeLarge: true,
	}
	if !validSizes[r.Size] {
		errors = append(errors, FieldError{Field: "size", Message: "size must be one of: small, medium, large"})
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateDeploymentRequest) ApplyDefaults() {
	if r.Environment == "" {
		r.Environment = "production"
	}
	if r.ChartVersion == "" {
		r.ChartVersion = "0.1.0"
	}
}

// DeploymentResponse wraps a deployment for API responses.
type DeploymentResponse struct {
	Deployment *Deployment `json:"deployment"`
}

// DeploymentListResponse wraps a list of deployments for API responses.
type DeploymentListResponse struct {
	Deployments []Deployment `json:"deployments"`
	TotalCount  int          `json:"total_count"`
}

// ProviderListResponse wraps a list of providers for API responses.
type ProviderListResponse struct {
	Providers []Provider `json:"providers"`
}

// ProviderResponse wraps a single provider for API responses.
type ProviderResponse struct {
	Provider *Provider `json:"provider"`
}

// DeploymentLogsResponse wraps deployment logs for API responses.
type DeploymentLogsResponse struct {
	Logs       []DeploymentLog `json:"logs"`
	TotalCount int             `json:"total_count"`
}

// CostEstimate represents a cost breakdown for a deployment.
type CostEstimate struct {
	Provider     string  `json:"provider"`
	Size         string  `json:"size"`
	ControlPlane float64 `json:"control_plane"`
	Workers      float64 `json:"workers"`
	Storage      float64 `json:"storage"`
	LoadBalancer float64 `json:"load_balancer"`
	Total        float64 `json:"total"`
	Currency     string  `json:"currency"`
}

// CostEstimateResponse wraps a cost estimate for API responses.
type CostEstimateResponse struct {
	Estimate *CostEstimate `json:"estimate"`
}
