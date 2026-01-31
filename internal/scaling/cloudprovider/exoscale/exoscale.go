// Package exoscale provides Exoscale cloud provider implementation.
package exoscale

import (
	"context"
	"fmt"
	"log/slog"

	egoscale "github.com/exoscale/egoscale/v2"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
)

// Provider implements cloudprovider.NodeProvider for Exoscale.
type Provider struct {
	client *egoscale.Client
	logger *slog.Logger
	config cloudprovider.ProviderConfig
}

// New creates a new Exoscale provider.
func New(apiKey, apiSecret string, logger *slog.Logger, config cloudprovider.ProviderConfig) (*Provider, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, cloudprovider.ErrInvalidCredentials
	}

	client, err := egoscale.NewClient(apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create Exoscale client: %w", err)
	}

	return &Provider{
		client: client,
		logger: logger.With("provider", "exoscale"),
		config: config,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "exoscale"
}

// Regions returns available Exoscale zones.
func (p *Provider) Regions() []string {
	return []string{"ch-gva-2", "ch-dk-2", "de-fra-1", "de-muc-1", "at-vie-1", "bg-sof-1"}
}

// CreateServer creates a new server instance.
func (p *Provider) CreateServer(ctx context.Context, opts cloudprovider.CreateServerOptions) (*cloudprovider.Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	p.logger.Info("creating server",
		"name", opts.Name,
		"region", opts.Region,
		"type", opts.InstanceType,
	)

	// Get instance type
	instanceType, err := p.client.GetInstanceType(ctx, opts.Region, opts.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance type: %w", err)
	}

	// Get template (image)
	templates, err := p.client.ListTemplates(ctx, opts.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var template *egoscale.Template
	for _, t := range templates {
		if t.Name != nil && *t.Name == opts.Image {
			template = t
			break
		}
	}
	if template == nil {
		return nil, fmt.Errorf("template not found: %s", opts.Image)
	}

	// Build labels
	labels := make(map[string]string)
	for k, v := range p.config.DefaultLabels {
		labels[k] = v
	}
	for k, v := range opts.Labels {
		labels[k] = v
	}

	// Create instance
	instance, err := p.client.CreateInstance(ctx, opts.Region, &egoscale.Instance{
		Name:           &opts.Name,
		InstanceTypeID: instanceType.ID,
		TemplateID:     template.ID,
		Labels:         &labels,
		UserData:       &opts.UserData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	p.logger.Info("server created", "id", *instance.ID, "name", *instance.Name)

	return p.toServer(instance, opts.Region), nil
}

// DeleteServer deletes a server instance.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	p.logger.Info("deleting server", "id", serverID)

	// Try each zone
	for _, zone := range p.Regions() {
		err := p.client.DeleteInstance(ctx, zone, &egoscale.Instance{ID: &serverID})
		if err == nil {
			p.logger.Info("server deleted", "id", serverID)
			return nil
		}
	}

	return fmt.Errorf("failed to delete server: server not found in any zone")
}

// GetServer retrieves a server by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*cloudprovider.Server, error) {
	// Try each zone
	for _, zone := range p.Regions() {
		instance, err := p.client.GetInstance(ctx, zone, serverID)
		if err == nil && instance != nil {
			return p.toServer(instance, zone), nil
		}
	}

	return nil, cloudprovider.ErrServerNotFound
}

// ListServers lists servers matching the given labels.
func (p *Provider) ListServers(ctx context.Context, labels map[string]string) ([]cloudprovider.Server, error) {
	var result []cloudprovider.Server

	for _, zone := range p.Regions() {
		instances, err := p.client.ListInstances(ctx, zone)
		if err != nil {
			p.logger.Warn("failed to list instances in zone", "zone", zone, "error", err)
			continue
		}

		for _, inst := range instances {
			// Filter by labels
			var instLabels map[string]string
			if inst.Labels != nil {
				instLabels = *inst.Labels
			}
			if matchesLabels(instLabels, labels) {
				result = append(result, *p.toServer(inst, zone))
			}
		}
	}

	return result, nil
}

// GetInstanceType retrieves an instance type by name.
func (p *Provider) GetInstanceType(ctx context.Context, typeName, region string) (*cloudprovider.InstanceType, error) {
	instanceType, err := p.client.GetInstanceType(ctx, region, typeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance type: %w", err)
	}

	return &cloudprovider.InstanceType{
		Name:        *instanceType.ID,
		CPUCores:    int(*instanceType.CPUs),
		MemoryMB:    int(*instanceType.Memory / (1024 * 1024)), // Convert bytes to MB
		DiskGB:      0,                                         // Exoscale uses separate volumes
		HourlyCost:  0,                                         // Would need pricing API
		SpotSupport: false,
	}, nil
}

// ListInstanceTypes lists available instance types.
func (p *Provider) ListInstanceTypes(ctx context.Context, region string) ([]cloudprovider.InstanceType, error) {
	types, err := p.client.ListInstanceTypes(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance types: %w", err)
	}

	var result []cloudprovider.InstanceType
	for _, t := range types {
		result = append(result, cloudprovider.InstanceType{
			Name:        *t.ID,
			CPUCores:    int(*t.CPUs),
			MemoryMB:    int(*t.Memory / (1024 * 1024)),
			DiskGB:      0,
			HourlyCost:  0,
			SpotSupport: false,
		})
	}

	return result, nil
}

// IsServerReady checks if the server is ready.
func (p *Provider) IsServerReady(ctx context.Context, serverID string) (bool, error) {
	server, err := p.GetServer(ctx, serverID)
	if err != nil {
		return false, err
	}

	return server.Status == cloudprovider.ServerStatusRunning, nil
}

// toServer converts an Exoscale instance to a cloudprovider.Server.
func (p *Provider) toServer(inst *egoscale.Instance, zone string) *cloudprovider.Server {
	var publicIP, privateIP string

	if inst.PublicIPAddress != nil {
		publicIP = inst.PublicIPAddress.String()
	}

	var labels map[string]string
	if inst.Labels != nil {
		labels = *inst.Labels
	}

	var instanceType string
	if inst.InstanceTypeID != nil {
		instanceType = *inst.InstanceTypeID
	}

	return &cloudprovider.Server{
		ID:        *inst.ID,
		Name:      *inst.Name,
		Status:    mapStatus(*inst.State),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		Region:    zone,
		Type:      instanceType,
		Labels:    labels,
		CreatedAt: *inst.CreatedAt,
	}
}

// mapStatus maps Exoscale instance state to cloudprovider status.
func mapStatus(state string) cloudprovider.ServerStatus {
	switch state {
	case "starting":
		return cloudprovider.ServerStatusStarting
	case "running":
		return cloudprovider.ServerStatusRunning
	case "stopping":
		return cloudprovider.ServerStatusStopping
	case "stopped":
		return cloudprovider.ServerStatusStopped
	case "destroying":
		return cloudprovider.ServerStatusDeleting
	default:
		return cloudprovider.ServerStatusUnknown
	}
}

// matchesLabels checks if instance labels contain all required labels.
func matchesLabels(instLabels map[string]string, required map[string]string) bool {
	if instLabels == nil {
		return len(required) == 0
	}

	for k, v := range required {
		if instLabels[k] != v {
			return false
		}
	}
	return true
}
