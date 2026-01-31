// Package hetzner provides Hetzner Cloud provider implementation.
package hetzner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
)

// Provider implements cloudprovider.NodeProvider for Hetzner Cloud.
type Provider struct {
	client *hcloud.Client
	logger *slog.Logger
	config cloudprovider.ProviderConfig
}

// New creates a new Hetzner Cloud provider.
func New(token string, logger *slog.Logger, config cloudprovider.ProviderConfig) (*Provider, error) {
	if token == "" {
		return nil, cloudprovider.ErrInvalidCredentials
	}

	client := hcloud.NewClient(
		hcloud.WithToken(token),
		hcloud.WithApplication("philotes", "1.0.0"),
	)

	return &Provider{
		client: client,
		logger: logger.With("provider", "hetzner"),
		config: config,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "hetzner"
}

// Regions returns available Hetzner regions.
func (p *Provider) Regions() []string {
	return []string{"nbg1", "fsn1", "hel1", "ash", "hil"}
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

	// Get server type
	serverType, _, err := p.client.ServerType.GetByName(ctx, opts.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %w", err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", opts.InstanceType)
	}

	// Get location
	location, _, err := p.client.Location.GetByName(ctx, opts.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}
	if location == nil {
		return nil, fmt.Errorf("location not found: %s", opts.Region)
	}

	// Get image
	image, _, err := p.client.Image.GetByNameAndArchitecture(ctx, opts.Image, hcloud.ArchitectureX86)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return nil, fmt.Errorf("image not found: %s", opts.Image)
	}

	// Build labels
	labels := make(map[string]string)
	for k, v := range p.config.DefaultLabels {
		labels[k] = v
	}
	for k, v := range opts.Labels {
		labels[k] = v
	}

	// Build create options
	createOpts := hcloud.ServerCreateOpts{
		Name:       opts.Name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		Labels:     labels,
		UserData:   opts.UserData,
	}

	// Add SSH keys if specified
	if len(opts.SSHKeyIDs) > 0 {
		for _, keyID := range opts.SSHKeyIDs {
			key, _, keyErr := p.client.SSHKey.GetByName(ctx, keyID)
			if keyErr != nil {
				p.logger.Warn("failed to get SSH key", "key", keyID, "error", keyErr)
				continue
			}
			if key != nil {
				createOpts.SSHKeys = append(createOpts.SSHKeys, key)
			}
		}
	}

	// Add network if specified
	if opts.NetworkID != "" {
		network, _, netErr := p.client.Network.GetByName(ctx, opts.NetworkID)
		if netErr == nil && network != nil {
			createOpts.Networks = []*hcloud.Network{network}
		}
	}

	// Add firewall if specified
	if opts.FirewallID != "" {
		firewall, _, fwErr := p.client.Firewall.GetByName(ctx, opts.FirewallID)
		if fwErr == nil && firewall != nil {
			createOpts.Firewalls = []*hcloud.ServerCreateFirewall{
				{Firewall: *firewall},
			}
		}
	}

	// Create the server
	result, _, err := p.client.Server.Create(ctx, createOpts)
	if err != nil {
		if isQuotaError(err) {
			return nil, cloudprovider.ErrQuotaExceeded
		}
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	server := result.Server
	p.logger.Info("server created",
		"id", server.ID,
		"name", server.Name,
		"status", server.Status,
	)

	return p.toServer(server), nil
}

// DeleteServer deletes a server instance.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	p.logger.Info("deleting server", "id", serverID)

	server, err := p.getServerByID(ctx, serverID)
	if err != nil {
		return err
	}

	_, _, err = p.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	p.logger.Info("server deleted", "id", serverID)
	return nil
}

// GetServer retrieves a server by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*cloudprovider.Server, error) {
	server, err := p.getServerByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return p.toServer(server), nil
}

// ListServers lists servers matching the given labels.
func (p *Provider) ListServers(ctx context.Context, labels map[string]string) ([]cloudprovider.Server, error) {
	// Build label selector
	var labelSelector string
	for k, v := range labels {
		if labelSelector != "" {
			labelSelector += ","
		}
		labelSelector += fmt.Sprintf("%s=%s", k, v)
	}

	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector,
		},
	}

	servers, err := p.client.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	result := make([]cloudprovider.Server, 0, len(servers))
	for _, s := range servers {
		result = append(result, *p.toServer(s))
	}

	return result, nil
}

// GetInstanceType retrieves an instance type by name.
func (p *Provider) GetInstanceType(ctx context.Context, typeName, region string) (*cloudprovider.InstanceType, error) {
	serverType, _, err := p.client.ServerType.GetByName(ctx, typeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %w", err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type not found: %s", typeName)
	}

	// Find pricing for the specified location
	var hourlyCost float64
	for i := range serverType.Pricings {
		if serverType.Pricings[i].Location.Name == region {
			// Parse the hourly price (Hetzner returns it as a string)
			_, _ = fmt.Sscanf(serverType.Pricings[i].Hourly.Gross, "%f", &hourlyCost) //nolint:errcheck // best-effort parse
			break
		}
	}

	return &cloudprovider.InstanceType{
		Name:        serverType.Name,
		CPUCores:    serverType.Cores,
		MemoryMB:    int(serverType.Memory * 1024), // Convert GB to MB
		DiskGB:      serverType.Disk,
		HourlyCost:  hourlyCost,
		SpotSupport: false, // Hetzner doesn't have spot instances
	}, nil
}

// ListInstanceTypes lists available instance types.
func (p *Provider) ListInstanceTypes(ctx context.Context, region string) ([]cloudprovider.InstanceType, error) {
	serverTypes, err := p.client.ServerType.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list server types: %w", err)
	}

	result := make([]cloudprovider.InstanceType, 0, len(serverTypes))
	for _, st := range serverTypes {
		// Find pricing for the specified location
		var hourlyCost float64
		for i := range st.Pricings {
			if region == "" || st.Pricings[i].Location.Name == region {
				_, _ = fmt.Sscanf(st.Pricings[i].Hourly.Gross, "%f", &hourlyCost) //nolint:errcheck // best-effort parse
				break
			}
		}

		result = append(result, cloudprovider.InstanceType{
			Name:        st.Name,
			CPUCores:    st.Cores,
			MemoryMB:    int(st.Memory * 1024),
			DiskGB:      st.Disk,
			HourlyCost:  hourlyCost,
			SpotSupport: false,
		})
	}

	return result, nil
}

// IsServerReady checks if the server is ready.
func (p *Provider) IsServerReady(ctx context.Context, serverID string) (bool, error) {
	server, err := p.getServerByID(ctx, serverID)
	if err != nil {
		return false, err
	}

	return server.Status == hcloud.ServerStatusRunning, nil
}

// getServerByID retrieves a server by ID (internal helper).
func (p *Provider) getServerByID(ctx context.Context, serverID string) (*hcloud.Server, error) {
	// Parse server ID as int64
	var id int64
	_, err := fmt.Sscanf(serverID, "%d", &id)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID: %s", serverID)
	}

	server, _, err := p.client.Server.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		return nil, cloudprovider.ErrServerNotFound
	}

	return server, nil
}

// toServer converts a Hetzner server to a cloudprovider.Server.
func (p *Provider) toServer(s *hcloud.Server) *cloudprovider.Server {
	var publicIP, privateIP string

	if s.PublicNet.IPv4.IP != nil {
		publicIP = s.PublicNet.IPv4.IP.String()
	}

	// Get first private network IP
	if len(s.PrivateNet) > 0 {
		privateIP = s.PrivateNet[0].IP.String()
	}

	return &cloudprovider.Server{
		ID:        fmt.Sprintf("%d", s.ID),
		Name:      s.Name,
		Status:    mapStatus(s.Status),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		Region:    s.Datacenter.Location.Name,
		Type:      s.ServerType.Name,
		Labels:    s.Labels,
		CreatedAt: s.Created,
	}
}

// mapStatus maps Hetzner server status to cloudprovider status.
func mapStatus(status hcloud.ServerStatus) cloudprovider.ServerStatus {
	switch status {
	case hcloud.ServerStatusInitializing:
		return cloudprovider.ServerStatusCreating
	case hcloud.ServerStatusStarting:
		return cloudprovider.ServerStatusStarting
	case hcloud.ServerStatusRunning:
		return cloudprovider.ServerStatusRunning
	case hcloud.ServerStatusStopping:
		return cloudprovider.ServerStatusStopping
	case hcloud.ServerStatusOff:
		return cloudprovider.ServerStatusStopped
	case hcloud.ServerStatusDeleting:
		return cloudprovider.ServerStatusDeleting
	default:
		return cloudprovider.ServerStatusUnknown
	}
}

// isQuotaError checks if the error is a quota exceeded error.
func isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota") || strings.Contains(errStr, "limit")
}

// WaitForServerReady waits for a server to become ready.
func (p *Provider) WaitForServerReady(ctx context.Context, serverID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ready, err := p.IsServerReady(ctx, serverID)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue polling
		}
	}

	return fmt.Errorf("timeout waiting for server %s to become ready", serverID)
}
