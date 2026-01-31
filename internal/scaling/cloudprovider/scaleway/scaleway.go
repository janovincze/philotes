// Package scaleway provides Scaleway cloud provider implementation.
package scaleway

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
)

// Provider implements cloudprovider.NodeProvider for Scaleway.
type Provider struct {
	client      *scw.Client
	instanceAPI *instance.API
	logger      *slog.Logger
	config      cloudprovider.ProviderConfig
	projectID   string
}

// New creates a new Scaleway provider.
func New(accessKey, secretKey, projectID string, logger *slog.Logger, config cloudprovider.ProviderConfig) (*Provider, error) {
	if accessKey == "" || secretKey == "" {
		return nil, cloudprovider.ErrInvalidCredentials
	}

	client, err := scw.NewClient(
		scw.WithAuth(accessKey, secretKey),
		scw.WithDefaultProjectID(projectID),
		scw.WithUserAgent("philotes/1.0.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scaleway client: %w", err)
	}

	instanceAPI := instance.NewAPI(client)

	return &Provider{
		client:      client,
		instanceAPI: instanceAPI,
		logger:      logger.With("provider", "scaleway"),
		config:      config,
		projectID:   projectID,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "scaleway"
}

// Regions returns available Scaleway regions.
func (p *Provider) Regions() []string {
	return []string{"fr-par-1", "fr-par-2", "fr-par-3", "nl-ams-1", "nl-ams-2", "pl-waw-1", "pl-waw-2"}
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

	zone, err := scw.ParseZone(opts.Region)
	if err != nil {
		return nil, fmt.Errorf("invalid zone: %w", err)
	}

	// Build tags from labels
	tags := make([]string, 0, len(p.config.DefaultLabels)+len(opts.Labels))
	for k, v := range p.config.DefaultLabels {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range opts.Labels {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}

	// Create server request
	image := opts.Image
	createReq := &instance.CreateServerRequest{
		Zone:           zone,
		Name:           opts.Name,
		CommercialType: opts.InstanceType,
		Image:          &image,
		Tags:           tags,
		Project:        &p.projectID,
	}

	// Create the server
	resp, err := p.instanceAPI.CreateServer(createReq)
	if err != nil {
		if isQuotaError(err) {
			return nil, cloudprovider.ErrQuotaExceeded
		}
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	server := resp.Server

	// Set user data if provided
	if opts.UserData != "" {
		setErr := p.instanceAPI.SetServerUserData(&instance.SetServerUserDataRequest{
			Zone:     zone,
			ServerID: server.ID,
			Key:      "cloud-init",
			Content:  strings.NewReader(opts.UserData),
		})
		if setErr != nil {
			p.logger.Warn("failed to set user data", "error", setErr)
		}
	}

	// Power on the server
	_, err = p.instanceAPI.ServerAction(&instance.ServerActionRequest{
		Zone:     zone,
		ServerID: server.ID,
		Action:   instance.ServerActionPoweron,
	})
	if err != nil {
		p.logger.Warn("failed to power on server", "error", err)
	}

	p.logger.Info("server created",
		"id", server.ID,
		"name", server.Name,
		"state", server.State,
	)

	return p.toServer(server, zone), nil
}

// DeleteServer deletes a server instance.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	p.logger.Info("deleting server", "id", serverID)

	// Parse zone from server ID (format: zone/id)
	zone, id, err := parseServerID(serverID)
	if err != nil {
		return err
	}

	// First power off the server (best-effort, ignore errors)
	//nolint:errcheck // intentional: power off before delete is best-effort
	p.instanceAPI.ServerAction(&instance.ServerActionRequest{
		Zone:     zone,
		ServerID: id,
		Action:   instance.ServerActionPoweroff,
	})

	// Wait a bit for power off
	time.Sleep(5 * time.Second)

	// Delete the server
	err = p.instanceAPI.DeleteServer(&instance.DeleteServerRequest{
		Zone:     zone,
		ServerID: id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	p.logger.Info("server deleted", "id", serverID)
	return nil
}

// GetServer retrieves a server by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*cloudprovider.Server, error) {
	zone, id, err := parseServerID(serverID)
	if err != nil {
		return nil, err
	}

	resp, err := p.instanceAPI.GetServer(&instance.GetServerRequest{
		Zone:     zone,
		ServerID: id,
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, cloudprovider.ErrServerNotFound
		}
		return nil, fmt.Errorf("failed to get server: %w", err)
	}

	return p.toServer(resp.Server, zone), nil
}

// ListServers lists servers matching the given labels.
func (p *Provider) ListServers(ctx context.Context, labels map[string]string) ([]cloudprovider.Server, error) {
	// Build tags filter
	tags := make([]string, 0, len(labels))
	for k, v := range labels {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}

	var result []cloudprovider.Server

	// List servers in all zones
	for _, regionStr := range p.Regions() {
		zone, zoneErr := scw.ParseZone(regionStr)
		if zoneErr != nil {
			continue
		}

		resp, err := p.instanceAPI.ListServers(&instance.ListServersRequest{
			Zone:    zone,
			Tags:    tags,
			Project: &p.projectID,
		})
		if err != nil {
			p.logger.Warn("failed to list servers in zone", "zone", zone, "error", err)
			continue
		}

		for _, s := range resp.Servers {
			result = append(result, *p.toServer(s, zone))
		}
	}

	return result, nil
}

// GetInstanceType retrieves an instance type by name.
func (p *Provider) GetInstanceType(ctx context.Context, typeName, region string) (*cloudprovider.InstanceType, error) {
	// Use cached instance types since the Scaleway SDK API for server types is complex
	types := getKnownInstanceTypes()
	for _, t := range types {
		if t.Name == typeName {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("instance type not found: %s", typeName)
}

// ListInstanceTypes lists available instance types.
func (p *Provider) ListInstanceTypes(ctx context.Context, region string) ([]cloudprovider.InstanceType, error) {
	// Return known instance types - Scaleway SDK doesn't have a simple ListServerTypes
	return getKnownInstanceTypes(), nil
}

// getKnownInstanceTypes returns common Scaleway instance types.
func getKnownInstanceTypes() []cloudprovider.InstanceType {
	return []cloudprovider.InstanceType{
		{Name: "DEV1-S", CPUCores: 2, MemoryMB: 2048, DiskGB: 20, HourlyCost: 0.0099, SpotSupport: false},
		{Name: "DEV1-M", CPUCores: 3, MemoryMB: 4096, DiskGB: 40, HourlyCost: 0.0198, SpotSupport: false},
		{Name: "DEV1-L", CPUCores: 4, MemoryMB: 8192, DiskGB: 80, HourlyCost: 0.0396, SpotSupport: false},
		{Name: "DEV1-XL", CPUCores: 4, MemoryMB: 12288, DiskGB: 120, HourlyCost: 0.0594, SpotSupport: false},
		{Name: "GP1-XS", CPUCores: 4, MemoryMB: 16384, DiskGB: 150, HourlyCost: 0.086, SpotSupport: false},
		{Name: "GP1-S", CPUCores: 8, MemoryMB: 32768, DiskGB: 300, HourlyCost: 0.172, SpotSupport: false},
		{Name: "GP1-M", CPUCores: 16, MemoryMB: 65536, DiskGB: 600, HourlyCost: 0.344, SpotSupport: false},
		{Name: "GP1-L", CPUCores: 32, MemoryMB: 131072, DiskGB: 600, HourlyCost: 0.688, SpotSupport: false},
		{Name: "GP1-XL", CPUCores: 48, MemoryMB: 262144, DiskGB: 600, HourlyCost: 1.032, SpotSupport: false},
	}
}

// IsServerReady checks if the server is ready.
func (p *Provider) IsServerReady(ctx context.Context, serverID string) (bool, error) {
	zone, id, err := parseServerID(serverID)
	if err != nil {
		return false, err
	}

	resp, err := p.instanceAPI.GetServer(&instance.GetServerRequest{
		Zone:     zone,
		ServerID: id,
	})
	if err != nil {
		return false, fmt.Errorf("failed to get server: %w", err)
	}

	return resp.Server.State == instance.ServerStateRunning, nil
}

// toServer converts a Scaleway server to a cloudprovider.Server.
func (p *Provider) toServer(s *instance.Server, zone scw.Zone) *cloudprovider.Server {
	var publicIP, privateIP string

	if s.PublicIP != nil {
		publicIP = s.PublicIP.Address.String()
	}
	if s.PrivateIP != nil {
		privateIP = *s.PrivateIP
	}

	// Parse labels from tags
	labels := make(map[string]string)
	for _, tag := range s.Tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return &cloudprovider.Server{
		ID:        fmt.Sprintf("%s/%s", zone, s.ID),
		Name:      s.Name,
		Status:    mapStatus(s.State),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		Region:    zone.String(),
		Type:      s.CommercialType,
		Labels:    labels,
		CreatedAt: *s.CreationDate,
	}
}

// mapStatus maps Scaleway server state to cloudprovider status.
func mapStatus(state instance.ServerState) cloudprovider.ServerStatus {
	switch state {
	case instance.ServerStateStarting:
		return cloudprovider.ServerStatusStarting
	case instance.ServerStateRunning:
		return cloudprovider.ServerStatusRunning
	case instance.ServerStateStopping:
		return cloudprovider.ServerStatusStopping
	case instance.ServerStateStopped:
		return cloudprovider.ServerStatusStopped
	case instance.ServerStateStoppedInPlace:
		return cloudprovider.ServerStatusStopped
	default:
		return cloudprovider.ServerStatusUnknown
	}
}

// parseServerID parses a server ID in the format "zone/id".
func parseServerID(serverID string) (scw.Zone, string, error) {
	parts := strings.SplitN(serverID, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid server ID format: %s (expected zone/id)", serverID)
	}

	zone, err := scw.ParseZone(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid zone in server ID: %w", err)
	}

	return zone, parts[1], nil
}

// isQuotaError checks if the error is a quota exceeded error.
func isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota") || strings.Contains(errStr, "limit")
}

// isNotFoundError checks if the error is a not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "404")
}
