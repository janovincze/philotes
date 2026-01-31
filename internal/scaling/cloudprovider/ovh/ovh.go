// Package ovh provides OVHcloud provider implementation.
package ovh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ovh/go-ovh/ovh"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
)

// Provider implements cloudprovider.NodeProvider for OVHcloud.
type Provider struct {
	client  *ovh.Client
	logger  *slog.Logger
	config  cloudprovider.ProviderConfig
	service string
}

// New creates a new OVHcloud provider.
func New(appKey, appSecret, consumerKey, endpoint, serviceName string, logger *slog.Logger, config cloudprovider.ProviderConfig) (*Provider, error) {
	if appKey == "" || appSecret == "" || consumerKey == "" {
		return nil, cloudprovider.ErrInvalidCredentials
	}

	client, err := ovh.NewClient(endpoint, appKey, appSecret, consumerKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create OVH client: %w", err)
	}

	return &Provider{
		client:  client,
		logger:  logger.With("provider", "ovh"),
		config:  config,
		service: serviceName,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ovh"
}

// Regions returns available OVH regions.
func (p *Provider) Regions() []string {
	return []string{"GRA11", "SBG5", "DE1", "UK1", "WAW1", "BHS5"}
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

	// OVH Public Cloud instance creation
	// POST /cloud/project/{serviceName}/instance
	type createRequest struct {
		FlavorID       string `json:"flavorId"`
		ImageID        string `json:"imageId"`
		Name           string `json:"name"`
		Region         string `json:"region"`
		SSHKeyID       string `json:"sshKeyId,omitempty"`
		UserData       string `json:"userData,omitempty"`
	}

	type instanceResponse struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Region    string `json:"region"`
		IPAddresses []struct {
			IP      string `json:"ip"`
			Type    string `json:"type"`
			Version int    `json:"version"`
		} `json:"ipAddresses"`
		Created string `json:"created"`
	}

	req := createRequest{
		FlavorID: opts.InstanceType,
		ImageID:  opts.Image,
		Name:     opts.Name,
		Region:   opts.Region,
		UserData: opts.UserData,
	}

	if len(opts.SSHKeyIDs) > 0 {
		req.SSHKeyID = opts.SSHKeyIDs[0]
	}

	var resp instanceResponse
	err := p.client.Post(fmt.Sprintf("/cloud/project/%s/instance", p.service), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	p.logger.Info("server created", "id", resp.ID, "name", resp.Name)

	var publicIP, privateIP string
	for _, ip := range resp.IPAddresses {
		if ip.Type == "public" && ip.Version == 4 {
			publicIP = ip.IP
		} else if ip.Type == "private" && ip.Version == 4 {
			privateIP = ip.IP
		}
	}

	return &cloudprovider.Server{
		ID:        resp.ID,
		Name:      resp.Name,
		Status:    mapStatus(resp.Status),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		Region:    resp.Region,
		Type:      opts.InstanceType,
		Labels:    opts.Labels,
	}, nil
}

// DeleteServer deletes a server instance.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	p.logger.Info("deleting server", "id", serverID)

	err := p.client.Delete(fmt.Sprintf("/cloud/project/%s/instance/%s", p.service, serverID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	p.logger.Info("server deleted", "id", serverID)
	return nil
}

// GetServer retrieves a server by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*cloudprovider.Server, error) {
	type instanceResponse struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Region    string `json:"region"`
		Flavor    struct {
			ID string `json:"id"`
		} `json:"flavor"`
		IPAddresses []struct {
			IP      string `json:"ip"`
			Type    string `json:"type"`
			Version int    `json:"version"`
		} `json:"ipAddresses"`
		Created string `json:"created"`
	}

	var resp instanceResponse
	err := p.client.Get(fmt.Sprintf("/cloud/project/%s/instance/%s", p.service, serverID), &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	var publicIP, privateIP string
	for _, ip := range resp.IPAddresses {
		if ip.Type == "public" && ip.Version == 4 {
			publicIP = ip.IP
		} else if ip.Type == "private" && ip.Version == 4 {
			privateIP = ip.IP
		}
	}

	return &cloudprovider.Server{
		ID:        resp.ID,
		Name:      resp.Name,
		Status:    mapStatus(resp.Status),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		Region:    resp.Region,
		Type:      resp.Flavor.ID,
	}, nil
}

// ListServers lists servers matching the given labels.
func (p *Provider) ListServers(ctx context.Context, labels map[string]string) ([]cloudprovider.Server, error) {
	type instanceResponse struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		Region    string `json:"region"`
		Flavor    struct {
			ID string `json:"id"`
		} `json:"flavor"`
		IPAddresses []struct {
			IP      string `json:"ip"`
			Type    string `json:"type"`
			Version int    `json:"version"`
		} `json:"ipAddresses"`
	}

	var instances []instanceResponse
	err := p.client.Get(fmt.Sprintf("/cloud/project/%s/instance", p.service), &instances)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}

	var result []cloudprovider.Server
	for _, inst := range instances {
		var publicIP, privateIP string
		for _, ip := range inst.IPAddresses {
			if ip.Type == "public" && ip.Version == 4 {
				publicIP = ip.IP
			} else if ip.Type == "private" && ip.Version == 4 {
				privateIP = ip.IP
			}
		}

		result = append(result, cloudprovider.Server{
			ID:        inst.ID,
			Name:      inst.Name,
			Status:    mapStatus(inst.Status),
			PublicIP:  publicIP,
			PrivateIP: privateIP,
			Region:    inst.Region,
			Type:      inst.Flavor.ID,
		})
	}

	return result, nil
}

// GetInstanceType retrieves an instance type by name.
func (p *Provider) GetInstanceType(ctx context.Context, typeName string, region string) (*cloudprovider.InstanceType, error) {
	types, err := p.ListInstanceTypes(ctx, region)
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		if t.Name == typeName {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("instance type not found: %s", typeName)
}

// ListInstanceTypes lists available instance types.
func (p *Provider) ListInstanceTypes(ctx context.Context, region string) ([]cloudprovider.InstanceType, error) {
	type flavorResponse struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		VCPUs      int     `json:"vcpus"`
		RAM        int     `json:"ram"`
		Disk       int     `json:"disk"`
		Region     string  `json:"region"`
		PlanCodes  struct {
			Hourly string `json:"hourly"`
		} `json:"planCodes"`
	}

	url := fmt.Sprintf("/cloud/project/%s/flavor", p.service)
	if region != "" {
		url += "?region=" + region
	}

	var flavors []flavorResponse
	err := p.client.Get(url, &flavors)
	if err != nil {
		return nil, fmt.Errorf("failed to list flavors: %w", err)
	}

	var result []cloudprovider.InstanceType
	for _, f := range flavors {
		result = append(result, cloudprovider.InstanceType{
			Name:        f.ID,
			CPUCores:    f.VCPUs,
			MemoryMB:    f.RAM,
			DiskGB:      f.Disk,
			HourlyCost:  0, // Would need to query pricing API
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

// mapStatus maps OVH instance status to cloudprovider status.
func mapStatus(status string) cloudprovider.ServerStatus {
	switch status {
	case "BUILD", "BUILDING":
		return cloudprovider.ServerStatusCreating
	case "ACTIVE":
		return cloudprovider.ServerStatusRunning
	case "SHUTOFF", "STOPPED":
		return cloudprovider.ServerStatusStopped
	case "DELETED", "DELETING":
		return cloudprovider.ServerStatusDeleting
	case "ERROR":
		return cloudprovider.ServerStatusError
	default:
		return cloudprovider.ServerStatusUnknown
	}
}
