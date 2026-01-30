// Package installer provides the Pulumi Automation API wrapper for deployments.
package installer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"

	"github.com/janovincze/philotes/internal/api/models"
)

// DeploymentRunner manages Pulumi stack deployments using the Automation API.
type DeploymentRunner struct {
	workDir      string
	pulumiOrg    string
	logger       *slog.Logger
	mu           sync.RWMutex
	activeStacks map[uuid.UUID]*auto.Stack
}

// DeploymentRunnerConfig holds configuration for the DeploymentRunner.
type DeploymentRunnerConfig struct {
	// WorkDir is the directory containing the Pulumi project.
	WorkDir string
	// PulumiOrg is the Pulumi organization for stack naming.
	PulumiOrg string
	// Logger is the structured logger.
	Logger *slog.Logger
}

// NewDeploymentRunner creates a new DeploymentRunner.
func NewDeploymentRunner(cfg DeploymentRunnerConfig) *DeploymentRunner {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &DeploymentRunner{
		workDir:      cfg.WorkDir,
		pulumiOrg:    cfg.PulumiOrg,
		logger:       logger.With("component", "deployment-runner"),
		activeStacks: make(map[uuid.UUID]*auto.Stack),
	}
}

// DeploymentConfig holds configuration for a deployment.
type DeploymentConfig struct {
	// DeploymentID is the unique identifier for this deployment.
	DeploymentID uuid.UUID
	// StackName is the name of the Pulumi stack to create/use.
	StackName string
	// Provider is the cloud provider (hetzner, scaleway, ovh, exoscale, contabo).
	Provider string
	// Region is the cloud region.
	Region string
	// Environment is the deployment environment.
	Environment string
	// Size is the deployment size (small, medium, large).
	Size models.DeploymentSize
	// Config holds additional deployment configuration.
	Config *models.DeploymentConfig
	// Credentials holds cloud provider credentials.
	Credentials *models.ProviderCredentials
}

// WorkerCount returns the configured worker count, or a default based on size.
func (c *DeploymentConfig) WorkerCount() int {
	if c.Config != nil && c.Config.WorkerCount > 0 {
		return c.Config.WorkerCount
	}
	// Get from size config
	sizeConfig := GetSizeConfig(c.Provider, c.Size)
	if sizeConfig != nil {
		return sizeConfig.WorkerCount
	}
	return 2 // Default
}

// DeploymentResult holds the result of a deployment.
type DeploymentResult struct {
	// ControlPlaneIP is the IP address of the control plane node.
	ControlPlaneIP string
	// LoadBalancerIP is the IP address of the load balancer.
	LoadBalancerIP string
	// Kubeconfig is the Kubernetes configuration.
	Kubeconfig string
	// DashboardURL is the URL of the Philotes dashboard.
	DashboardURL string
	// APIURL is the URL of the Philotes API.
	APIURL string
}

// LogCallback is called with log messages during deployment.
type LogCallback func(level, step, message string)

// Deploy runs a deployment using the Pulumi Automation API.
func (r *DeploymentRunner) Deploy(ctx context.Context, cfg *DeploymentConfig, logCallback LogCallback) (*DeploymentResult, error) {
	r.logger.Info("starting deployment",
		"deployment_id", cfg.DeploymentID,
		"stack", cfg.StackName,
		"provider", cfg.Provider,
		"region", cfg.Region,
	)

	// Generate stack name if not provided
	stackName := cfg.StackName
	if stackName == "" {
		stackName = fmt.Sprintf("%s/%s-%s", r.pulumiOrg, cfg.Provider, cfg.DeploymentID.String()[:8])
	}

	logCallback("info", "initializing", "Initializing Pulumi stack")

	// Create or select the stack
	stack, err := r.createOrSelectStack(ctx, stackName)
	if err != nil {
		logCallback("error", "initializing", fmt.Sprintf("Failed to initialize stack: %v", err))
		return nil, fmt.Errorf("failed to create/select stack: %w", err)
	}

	// Store active stack for potential cancellation
	r.mu.Lock()
	r.activeStacks[cfg.DeploymentID] = &stack
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.activeStacks, cfg.DeploymentID)
		r.mu.Unlock()
	}()

	logCallback("info", "configuring", "Configuring deployment parameters")

	// Set stack configuration
	tempFiles, err := r.configureStack(ctx, stack, cfg)
	// Clean up temp files when function exits
	defer func() {
		for _, f := range tempFiles {
			if removeErr := os.Remove(f); removeErr != nil {
				r.logger.Debug("failed to remove temp file", "path", f, "error", removeErr)
			}
		}
	}()
	if err != nil {
		logCallback("error", "configuring", fmt.Sprintf("Failed to configure stack: %v", err))
		return nil, fmt.Errorf("failed to configure stack: %w", err)
	}

	logCallback("info", "provisioning", "Provisioning cloud infrastructure")

	// Create event stream channel for logging
	eventsChan := make(chan events.EngineEvent)

	// Start a goroutine to process events
	go func() {
		for event := range eventsChan {
			r.processEvent(event, logCallback)
		}
	}()

	// Run pulumi up
	result, err := stack.Up(ctx,
		optup.EventStreams(eventsChan),
		optup.ProgressStreams(io.Discard), // We handle logging via events
	)
	if err != nil {
		logCallback("error", "provisioning", fmt.Sprintf("Deployment failed: %v", err))
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	logCallback("info", "completed", "Deployment completed successfully")

	// Extract outputs
	outputs := result.Outputs
	deployResult := &DeploymentResult{}

	if v, ok := outputs["controlPlaneIP"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.ControlPlaneIP = s
		}
	}
	if v, ok := outputs["loadBalancerIP"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.LoadBalancerIP = s
		}
	}
	if v, ok := outputs["kubeconfig"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.Kubeconfig = s
		}
	}

	// Derive URLs from load balancer IP
	if deployResult.LoadBalancerIP != "" {
		if cfg.Config != nil && cfg.Config.Domain != "" {
			deployResult.DashboardURL = fmt.Sprintf("https://%s", cfg.Config.Domain)
			deployResult.APIURL = fmt.Sprintf("https://api.%s", cfg.Config.Domain)
		} else {
			deployResult.DashboardURL = fmt.Sprintf("http://%s", deployResult.LoadBalancerIP)
			deployResult.APIURL = fmt.Sprintf("http://%s:8080", deployResult.LoadBalancerIP)
		}
	}

	r.logger.Info("deployment completed",
		"deployment_id", cfg.DeploymentID,
		"control_plane_ip", deployResult.ControlPlaneIP,
		"load_balancer_ip", deployResult.LoadBalancerIP,
	)

	return deployResult, nil
}

// DeployWithTracker runs a deployment with progress tracking.
func (r *DeploymentRunner) DeployWithTracker(ctx context.Context, cfg *DeploymentConfig, logCallback LogCallback, tracker *ProgressTracker) (*DeploymentResult, error) {
	r.logger.Info("starting deployment with tracker",
		"deployment_id", cfg.DeploymentID,
		"provider", cfg.Provider,
		"region", cfg.Region,
	)

	// Complete auth step (credentials already validated)
	tracker.CompleteStep(cfg.DeploymentID, "auth")

	// Generate stack name if not provided
	stackName := cfg.StackName
	if stackName == "" {
		stackName = fmt.Sprintf("%s/%s-%s", r.pulumiOrg, cfg.Provider, cfg.DeploymentID.String()[:8])
	}

	// Start network step
	tracker.StartStep(cfg.DeploymentID, "network")
	logCallback("info", "network", "Initializing Pulumi stack")

	// Create or select the stack
	stack, err := r.createOrSelectStack(ctx, stackName)
	if err != nil {
		tracker.FailStep(cfg.DeploymentID, "network", err)
		logCallback("error", "network", fmt.Sprintf("Failed to initialize stack: %v", err))
		return nil, fmt.Errorf("failed to create/select stack: %w", err)
	}

	// Store active stack for potential cancellation
	r.mu.Lock()
	r.activeStacks[cfg.DeploymentID] = &stack
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.activeStacks, cfg.DeploymentID)
		r.mu.Unlock()
	}()

	logCallback("info", "network", "Configuring deployment parameters")

	// Set stack configuration
	tempFiles, err := r.configureStack(ctx, stack, cfg)
	defer func() {
		for _, f := range tempFiles {
			if removeErr := os.Remove(f); removeErr != nil {
				r.logger.Debug("failed to remove temp file", "path", f, "error", removeErr)
			}
		}
	}()
	if err != nil {
		tracker.FailStep(cfg.DeploymentID, "network", err)
		logCallback("error", "network", fmt.Sprintf("Failed to configure stack: %v", err))
		return nil, fmt.Errorf("failed to configure stack: %w", err)
	}

	logCallback("info", "network", "Provisioning cloud infrastructure")

	// Create event stream channel for logging
	eventsChan := make(chan events.EngineEvent)

	// Start a goroutine to process events with tracker
	go func() {
		for event := range eventsChan {
			r.processEventWithTracker(event, cfg.DeploymentID, logCallback, tracker)
		}
	}()

	// Run pulumi up
	result, err := stack.Up(ctx,
		optup.EventStreams(eventsChan),
		optup.ProgressStreams(io.Discard),
	)
	if err != nil {
		// Find the current step and fail it
		progress := tracker.GetProgress(cfg.DeploymentID)
		if progress != nil {
			for _, step := range progress.Steps {
				if step.Status == StepStatusInProgress {
					tracker.FailStep(cfg.DeploymentID, step.ID, err)
					break
				}
			}
		}
		logCallback("error", "provisioning", fmt.Sprintf("Deployment failed: %v", err))
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Complete remaining steps
	tracker.CompleteStep(cfg.DeploymentID, "health")
	tracker.CompleteStep(cfg.DeploymentID, "ssl")

	logCallback("info", "completed", "Deployment completed successfully")

	// Extract outputs
	outputs := result.Outputs
	deployResult := &DeploymentResult{}

	if v, ok := outputs["controlPlaneIP"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.ControlPlaneIP = s
		}
	}
	if v, ok := outputs["loadBalancerIP"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.LoadBalancerIP = s
		}
	}
	if v, ok := outputs["kubeconfig"]; ok {
		if s, ok := v.Value.(string); ok {
			deployResult.Kubeconfig = s
		}
	}

	// Derive URLs from load balancer IP
	if deployResult.LoadBalancerIP != "" {
		if cfg.Config != nil && cfg.Config.Domain != "" {
			deployResult.DashboardURL = fmt.Sprintf("https://%s", cfg.Config.Domain)
			deployResult.APIURL = fmt.Sprintf("https://api.%s", cfg.Config.Domain)
		} else {
			deployResult.DashboardURL = fmt.Sprintf("http://%s", deployResult.LoadBalancerIP)
			deployResult.APIURL = fmt.Sprintf("http://%s:8080", deployResult.LoadBalancerIP)
		}
	}

	r.logger.Info("deployment with tracker completed",
		"deployment_id", cfg.DeploymentID,
		"control_plane_ip", deployResult.ControlPlaneIP,
		"load_balancer_ip", deployResult.LoadBalancerIP,
	)

	return deployResult, nil
}

// DeployFromStep runs a deployment starting from a specific step (for retries).
func (r *DeploymentRunner) DeployFromStep(ctx context.Context, cfg *DeploymentConfig, fromStep string, logCallback LogCallback, tracker *ProgressTracker) (*DeploymentResult, error) {
	r.logger.Info("resuming deployment from step",
		"deployment_id", cfg.DeploymentID,
		"from_step", fromStep,
	)

	// For now, retry means running the full deployment again
	// The Pulumi state will handle idempotency for already-created resources
	return r.DeployWithTracker(ctx, cfg, logCallback, tracker)
}

// Destroy destroys a deployment.
func (r *DeploymentRunner) Destroy(ctx context.Context, stackName string, logCallback LogCallback) error {
	r.logger.Info("destroying deployment", "stack", stackName)

	logCallback("info", "destroying", "Destroying cloud infrastructure")

	// Select the stack
	stack, err := auto.SelectStackLocalSource(ctx, stackName, r.workDir)
	if err != nil {
		logCallback("error", "destroying", fmt.Sprintf("Failed to select stack: %v", err))
		return fmt.Errorf("failed to select stack: %w", err)
	}

	// Create event stream channel for logging
	eventsChan := make(chan events.EngineEvent)

	// Start a goroutine to process events
	go func() {
		for event := range eventsChan {
			r.processEvent(event, logCallback)
		}
	}()

	// Run pulumi destroy
	_, err = stack.Destroy(ctx,
		optdestroy.EventStreams(eventsChan),
		optdestroy.ProgressStreams(io.Discard),
	)
	if err != nil {
		logCallback("error", "destroying", fmt.Sprintf("Destroy failed: %v", err))
		return fmt.Errorf("destroy failed: %w", err)
	}

	logCallback("info", "completed", "Infrastructure destroyed successfully")

	// Remove the stack
	if err := stack.Workspace().RemoveStack(ctx, stackName); err != nil {
		r.logger.Warn("failed to remove stack", "stack", stackName, "error", err)
	}

	r.logger.Info("deployment destroyed", "stack", stackName)
	return nil
}

// Cancel cancels an active deployment.
func (r *DeploymentRunner) Cancel(deploymentID uuid.UUID) error {
	r.mu.RLock()
	stack, ok := r.activeStacks[deploymentID]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active deployment found for %s", deploymentID)
	}

	// Cancel the stack operation
	if err := stack.Cancel(context.Background()); err != nil {
		return fmt.Errorf("failed to cancel deployment: %w", err)
	}

	r.logger.Info("deployment canceled", "deployment_id", deploymentID)
	return nil
}

// createOrSelectStack creates a new stack or selects an existing one.
func (r *DeploymentRunner) createOrSelectStack(ctx context.Context, stackName string) (auto.Stack, error) {
	// Try to create the stack
	stack, err := auto.NewStackLocalSource(ctx, stackName, r.workDir)
	if err != nil {
		// If stack exists, select it
		if strings.Contains(err.Error(), "already exists") {
			stack, err = auto.SelectStackLocalSource(ctx, stackName, r.workDir)
			if err != nil {
				return auto.Stack{}, fmt.Errorf("failed to select existing stack: %w", err)
			}
			return stack, nil
		}
		return auto.Stack{}, fmt.Errorf("failed to create stack: %w", err)
	}

	return stack, nil
}

// configureStack sets the stack configuration.
// Returns the path to any temp files created (for cleanup) and an error if any.
func (r *DeploymentRunner) configureStack(ctx context.Context, stack auto.Stack, cfg *DeploymentConfig) (tempFiles []string, err error) {
	// Get size configuration
	sizeConfig := GetSizeConfig(cfg.Provider, cfg.Size)
	if sizeConfig == nil {
		return nil, fmt.Errorf("invalid size %s for provider %s", cfg.Size, cfg.Provider)
	}

	// Set basic configuration
	configs := map[string]string{
		"philotes:provider":         cfg.Provider,
		"philotes:region":           cfg.Region,
		"philotes:environment":      cfg.Environment,
		"philotes:controlPlaneType": sizeConfig.ControlPlaneType,
		"philotes:workerType":       sizeConfig.WorkerType,
		"philotes:workerCount":      fmt.Sprintf("%d", sizeConfig.WorkerCount),
		"philotes:storageSizeGB":    fmt.Sprintf("%d", sizeConfig.StorageSizeGB),
	}

	// Add deployment config overrides
	if cfg.Config != nil {
		if cfg.Config.WorkerCount > 0 {
			configs["philotes:workerCount"] = fmt.Sprintf("%d", cfg.Config.WorkerCount)
		}
		if cfg.Config.StorageSizeGB > 0 {
			configs["philotes:storageSizeGB"] = fmt.Sprintf("%d", cfg.Config.StorageSizeGB)
		}
		if cfg.Config.ChartVersion != "" {
			configs["philotes:chartVersion"] = cfg.Config.ChartVersion
		}
		if cfg.Config.SSHPublicKey != "" {
			// Write SSH public key to a temp file (0o644 permissions for public key)
			sshKeyPath := filepath.Join(os.TempDir(), fmt.Sprintf("philotes-%s.pub", cfg.DeploymentID.String()))
			if err := os.WriteFile(sshKeyPath, []byte(cfg.Config.SSHPublicKey), 0o644); err != nil {
				return nil, fmt.Errorf("failed to write SSH public key: %w", err)
			}
			configs["philotes:sshPublicKeyPath"] = sshKeyPath
			tempFiles = append(tempFiles, sshKeyPath)
		}
	}

	// Apply configuration
	for key, value := range configs {
		if err := stack.SetConfig(ctx, key, auto.ConfigValue{Value: value}); err != nil {
			return tempFiles, fmt.Errorf("failed to set config %s: %w", key, err)
		}
	}

	// Set provider credentials as secrets
	if cfg.Credentials != nil {
		if err := r.setProviderCredentials(ctx, stack, cfg.Provider, cfg.Credentials); err != nil {
			return tempFiles, err
		}
	}

	return tempFiles, nil
}

// setProviderCredentials sets provider-specific credentials as secrets.
func (r *DeploymentRunner) setProviderCredentials(ctx context.Context, stack auto.Stack, provider string, creds *models.ProviderCredentials) error {
	switch provider {
	case "hetzner":
		if creds.HetznerToken != "" {
			if err := stack.SetConfig(ctx, "hcloud:token", auto.ConfigValue{Value: creds.HetznerToken, Secret: true}); err != nil {
				return fmt.Errorf("failed to set hcloud token: %w", err)
			}
		}

	case "scaleway":
		if creds.ScalewayAccessKey != "" {
			if err := stack.SetConfig(ctx, "scaleway:accessKey", auto.ConfigValue{Value: creds.ScalewayAccessKey, Secret: true}); err != nil {
				return fmt.Errorf("failed to set scaleway access key: %w", err)
			}
		}
		if creds.ScalewaySecretKey != "" {
			if err := stack.SetConfig(ctx, "scaleway:secretKey", auto.ConfigValue{Value: creds.ScalewaySecretKey, Secret: true}); err != nil {
				return fmt.Errorf("failed to set scaleway secret key: %w", err)
			}
		}
		if creds.ScalewayProjectID != "" {
			if err := stack.SetConfig(ctx, "scaleway:projectId", auto.ConfigValue{Value: creds.ScalewayProjectID}); err != nil {
				return fmt.Errorf("failed to set scaleway project id: %w", err)
			}
		}

	case "ovh":
		if creds.OVHApplicationKey != "" {
			if err := stack.SetConfig(ctx, "ovh:applicationKey", auto.ConfigValue{Value: creds.OVHApplicationKey, Secret: true}); err != nil {
				return fmt.Errorf("failed to set ovh application key: %w", err)
			}
		}
		if creds.OVHApplicationSecret != "" {
			if err := stack.SetConfig(ctx, "ovh:applicationSecret", auto.ConfigValue{Value: creds.OVHApplicationSecret, Secret: true}); err != nil {
				return fmt.Errorf("failed to set ovh application secret: %w", err)
			}
		}
		if creds.OVHConsumerKey != "" {
			if err := stack.SetConfig(ctx, "ovh:consumerKey", auto.ConfigValue{Value: creds.OVHConsumerKey, Secret: true}); err != nil {
				return fmt.Errorf("failed to set ovh consumer key: %w", err)
			}
		}
		if creds.OVHEndpoint != "" {
			if err := stack.SetConfig(ctx, "ovh:endpoint", auto.ConfigValue{Value: creds.OVHEndpoint}); err != nil {
				return fmt.Errorf("failed to set ovh endpoint: %w", err)
			}
		}
		if creds.OVHServiceName != "" {
			if err := stack.SetConfig(ctx, "ovh:serviceName", auto.ConfigValue{Value: creds.OVHServiceName}); err != nil {
				return fmt.Errorf("failed to set ovh service name: %w", err)
			}
		}

	case "exoscale":
		if creds.ExoscaleAPIKey != "" {
			if err := stack.SetConfig(ctx, "exoscale:key", auto.ConfigValue{Value: creds.ExoscaleAPIKey, Secret: true}); err != nil {
				return fmt.Errorf("failed to set exoscale api key: %w", err)
			}
		}
		if creds.ExoscaleAPISecret != "" {
			if err := stack.SetConfig(ctx, "exoscale:secret", auto.ConfigValue{Value: creds.ExoscaleAPISecret, Secret: true}); err != nil {
				return fmt.Errorf("failed to set exoscale api secret: %w", err)
			}
		}

	case "contabo":
		// Contabo uses environment variables for authentication
		// Set them via environment configuration
		if creds.ContaboClientID != "" {
			if err := stack.SetConfig(ctx, "contabo:clientId", auto.ConfigValue{Value: creds.ContaboClientID, Secret: true}); err != nil {
				return fmt.Errorf("failed to set contabo client id: %w", err)
			}
		}
		if creds.ContaboClientSecret != "" {
			if err := stack.SetConfig(ctx, "contabo:clientSecret", auto.ConfigValue{Value: creds.ContaboClientSecret, Secret: true}); err != nil {
				return fmt.Errorf("failed to set contabo client secret: %w", err)
			}
		}
		if creds.ContaboAPIUser != "" {
			if err := stack.SetConfig(ctx, "contabo:apiUser", auto.ConfigValue{Value: creds.ContaboAPIUser}); err != nil {
				return fmt.Errorf("failed to set contabo api user: %w", err)
			}
		}
		if creds.ContaboAPIPassword != "" {
			if err := stack.SetConfig(ctx, "contabo:apiPassword", auto.ConfigValue{Value: creds.ContaboAPIPassword, Secret: true}); err != nil {
				return fmt.Errorf("failed to set contabo api password: %w", err)
			}
		}
	}

	return nil
}

// processEvent processes Pulumi engine events and forwards them to the log callback.
func (r *DeploymentRunner) processEvent(event events.EngineEvent, logCallback LogCallback) {
	// Handle diagnostic events
	if e := event.DiagnosticEvent; e != nil {
		level := "info"
		switch e.Severity {
		case "warning":
			level = "warn"
		case "error":
			level = "error"
		}
		if e.Message != "" {
			logCallback(level, "provisioning", e.Message)
		}
		return
	}

	// Handle resource pre events
	if e := event.ResourcePreEvent; e != nil {
		if e.Metadata.Type != "" {
			msg := fmt.Sprintf("Creating %s: %s", e.Metadata.Type, e.Metadata.URN)
			logCallback("info", "provisioning", msg)
		}
		return
	}

	// Handle resource outputs events
	if e := event.ResOutputsEvent; e != nil {
		if e.Metadata.Type != "" {
			msg := fmt.Sprintf("Created %s: %s", e.Metadata.Type, e.Metadata.URN)
			logCallback("info", "provisioning", msg)
		}
		return
	}

	// Handle summary events
	if e := event.SummaryEvent; e != nil {
		if e.ResourceChanges != nil {
			var changes []string
			for op, count := range e.ResourceChanges {
				if count > 0 {
					changes = append(changes, fmt.Sprintf("%s: %d", op, count))
				}
			}
			if len(changes) > 0 {
				logCallback("info", "summary", strings.Join(changes, ", "))
			}
		}
		return
	}
}

// resourceTypeToStep maps Pulumi resource types to deployment steps.
var resourceTypeToStep = map[string]string{
	// Network resources
	"hcloud:index/network:Network":           "network",
	"hcloud:index/networkSubnet:NetworkSubnet": "network",
	"hcloud:index/firewall:Firewall":         "network",
	"scaleway:index/vpcPrivateNetwork:VpcPrivateNetwork": "network",
	"exoscale:index/securityGroup:SecurityGroup":         "network",

	// Compute resources
	"hcloud:index/server:Server":       "compute",
	"hcloud:index/sshKey:SshKey":       "compute",
	"scaleway:index/instance:Instance": "compute",
	"exoscale:index/compute:Compute":   "compute",

	// Load balancer
	"hcloud:index/loadBalancer:LoadBalancer":             "compute",
	"hcloud:index/loadBalancerService:LoadBalancerService": "compute",
	"hcloud:index/loadBalancerTarget:LoadBalancerTarget": "compute",

	// Volume/storage
	"hcloud:index/volume:Volume":           "storage",
	"hcloud:index/volumeAttachment:VolumeAttachment": "storage",

	// Kubernetes resources
	"command:remote:Command": "k3s",

	// Helm charts
	"kubernetes:helm.sh/v3:Release": "philotes",
}

// processEventWithTracker processes events and updates the progress tracker.
func (r *DeploymentRunner) processEventWithTracker(event events.EngineEvent, deploymentID uuid.UUID, logCallback LogCallback, tracker *ProgressTracker) {
	// Handle diagnostic events
	if e := event.DiagnosticEvent; e != nil {
		level := "info"
		switch e.Severity {
		case "warning":
			level = "warn"
		case "error":
			level = "error"
		}
		if e.Message != "" {
			logCallback(level, "provisioning", e.Message)
		}
		return
	}

	// Handle resource pre events (resource is being created)
	if e := event.ResourcePreEvent; e != nil {
		if e.Metadata.Type != "" {
			// Determine which step this resource belongs to
			stepID := r.mapResourceTypeToStep(e.Metadata.Type)
			if stepID != "" {
				// Get current progress
				progress := tracker.GetProgress(deploymentID)
				if progress != nil {
					currentStepID := ""
					if progress.CurrentStepIndex < len(progress.Steps) {
						currentStepID = progress.Steps[progress.CurrentStepIndex].ID
					}

					// If this resource belongs to a different step, complete current and start new
					if stepID != currentStepID {
						// Complete current step
						if currentStepID != "" && currentStepID != stepID {
							tracker.CompleteStep(deploymentID, currentStepID)
						}
						// Start new step
						tracker.StartStep(deploymentID, stepID)
					}
				}
			}

			msg := fmt.Sprintf("Creating %s", r.extractResourceName(e.Metadata.URN))
			logCallback("info", stepID, msg)
		}
		return
	}

	// Handle resource outputs events (resource created)
	if e := event.ResOutputsEvent; e != nil {
		if e.Metadata.Type != "" {
			stepID := r.mapResourceTypeToStep(e.Metadata.Type)
			resourceName := r.extractResourceName(e.Metadata.URN)

			// Track created resource
			tracker.AddResource(deploymentID, CreatedResource{
				Type: e.Metadata.Type,
				Name: resourceName,
			})

			msg := fmt.Sprintf("Created %s", resourceName)
			logCallback("info", stepID, msg)
		}
		return
	}

	// Handle summary events
	if e := event.SummaryEvent; e != nil {
		if e.ResourceChanges != nil {
			var changes []string
			for op, count := range e.ResourceChanges {
				if count > 0 {
					changes = append(changes, fmt.Sprintf("%s: %d", op, count))
				}
			}
			if len(changes) > 0 {
				logCallback("info", "summary", strings.Join(changes, ", "))
			}
		}
		return
	}
}

// mapResourceTypeToStep maps a Pulumi resource type to a deployment step ID.
func (r *DeploymentRunner) mapResourceTypeToStep(resourceType string) string {
	if stepID, ok := resourceTypeToStep[resourceType]; ok {
		return stepID
	}

	// Infer from resource type name
	typeLower := strings.ToLower(resourceType)
	switch {
	case strings.Contains(typeLower, "network") || strings.Contains(typeLower, "subnet") || strings.Contains(typeLower, "firewall") || strings.Contains(typeLower, "security"):
		return "network"
	case strings.Contains(typeLower, "server") || strings.Contains(typeLower, "instance") || strings.Contains(typeLower, "compute") || strings.Contains(typeLower, "ssh"):
		return "compute"
	case strings.Contains(typeLower, "loadbalancer") || strings.Contains(typeLower, "lb"):
		return "compute"
	case strings.Contains(typeLower, "volume") || strings.Contains(typeLower, "storage"):
		return "storage"
	case strings.Contains(typeLower, "command"):
		return "k3s"
	case strings.Contains(typeLower, "helm") || strings.Contains(typeLower, "release"):
		return "philotes"
	case strings.Contains(typeLower, "cert") || strings.Contains(typeLower, "certificate"):
		return "ssl"
	default:
		return "provisioning"
	}
}

// extractResourceName extracts a friendly name from a Pulumi URN.
func (r *DeploymentRunner) extractResourceName(urn string) string {
	// URN format: urn:pulumi:stack::project::type::name
	parts := strings.Split(urn, "::")
	if len(parts) >= 4 {
		// Get the last part (resource name)
		name := parts[len(parts)-1]
		// Also get the type for context
		resourceType := parts[len(parts)-2]
		// Extract just the type name (after the last /)
		typeParts := strings.Split(resourceType, "/")
		shortType := typeParts[len(typeParts)-1]
		// Split by : and get last part
		typeNameParts := strings.Split(shortType, ":")
		shortType = typeNameParts[len(typeNameParts)-1]

		return fmt.Sprintf("%s (%s)", name, shortType)
	}
	return urn
}
