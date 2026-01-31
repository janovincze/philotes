// Package contabo provides Contabo cloud provider implementation.
package contabo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/janovincze/philotes/internal/scaling/cloudprovider"
)

const baseURL = "https://api.contabo.com/v1"

// Provider implements cloudprovider.NodeProvider for Contabo.
type Provider struct {
	httpClient *http.Client
	logger     *slog.Logger
	config     cloudprovider.ProviderConfig
	token      string
}

// AuthResponse represents the OAuth token response.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// New creates a new Contabo provider.
func New(clientID, clientSecret, apiUser, apiPassword string, logger *slog.Logger, config cloudprovider.ProviderConfig) (*Provider, error) {
	if clientID == "" || clientSecret == "" || apiUser == "" || apiPassword == "" {
		return nil, cloudprovider.ErrInvalidCredentials
	}

	p := &Provider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger.With("provider", "contabo"),
		config:     config,
	}

	// Authenticate and get token
	token, err := p.authenticate(clientID, clientSecret, apiUser, apiPassword)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	p.token = token

	return p, nil
}

// authenticate gets an OAuth token from Contabo.
func (p *Provider) authenticate(clientID, clientSecret, apiUser, apiPassword string) (string, error) {
	data := fmt.Sprintf("client_id=%s&client_secret=%s&username=%s&password=%s&grant_type=password",
		clientID, clientSecret, apiUser, apiPassword)

	req, err := http.NewRequest(http.MethodPost, "https://auth.contabo.com/auth/realms/contabo/protocol/openid-connect/token",
		bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed: %s", string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	return authResp.AccessToken, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "contabo"
}

// Regions returns available Contabo regions.
func (p *Provider) Regions() []string {
	return []string{"EU", "US-central", "US-east", "US-west", "SIN", "AUS", "JPN", "UK"}
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

	type createRequest struct {
		ImageID     string `json:"imageId"`
		ProductID   string `json:"productId"`
		Region      string `json:"region"`
		DisplayName string `json:"displayName"`
		UserData    string `json:"userData,omitempty"`
	}

	reqBody := createRequest{
		ImageID:     opts.Image,
		ProductID:   opts.InstanceType,
		Region:      opts.Region,
		DisplayName: opts.Name,
		UserData:    opts.UserData,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/compute/instances", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-request-id", fmt.Sprintf("philotes-%d", time.Now().UnixNano()))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create server: %s", string(respBody))
	}

	type createResponse struct {
		Data []struct {
			InstanceID  int64  `json:"instanceId"`
			DisplayName string `json:"displayName"`
			Status      string `json:"status"`
			IPConfig    struct {
				V4 struct {
					IP string `json:"ip"`
				} `json:"v4"`
			} `json:"ipConfig"`
			Region    string `json:"region"`
			ProductID string `json:"productId"`
		} `json:"data"`
	}

	var createResp createResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(createResp.Data) == 0 {
		return nil, fmt.Errorf("no instance returned in response")
	}

	inst := createResp.Data[0]
	p.logger.Info("server created", "id", inst.InstanceID, "name", inst.DisplayName)

	return &cloudprovider.Server{
		ID:       fmt.Sprintf("%d", inst.InstanceID),
		Name:     inst.DisplayName,
		Status:   mapStatus(inst.Status),
		PublicIP: inst.IPConfig.V4.IP,
		Region:   inst.Region,
		Type:     inst.ProductID,
		Labels:   opts.Labels,
	}, nil
}

// DeleteServer deletes a server instance.
func (p *Provider) DeleteServer(ctx context.Context, serverID string) error {
	p.logger.Info("deleting server", "id", serverID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL+"/compute/instances/"+serverID, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("x-request-id", fmt.Sprintf("philotes-%d", time.Now().UnixNano()))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete server: %s", string(body))
	}

	p.logger.Info("server deleted", "id", serverID)
	return nil
}

// GetServer retrieves a server by ID.
func (p *Provider) GetServer(ctx context.Context, serverID string) (*cloudprovider.Server, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/compute/instances/"+serverID, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("x-request-id", fmt.Sprintf("philotes-%d", time.Now().UnixNano()))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, cloudprovider.ErrServerNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get server: %s", string(body))
	}

	type getResponse struct {
		Data []struct {
			InstanceID  int64  `json:"instanceId"`
			DisplayName string `json:"displayName"`
			Status      string `json:"status"`
			IPConfig    struct {
				V4 struct {
					IP string `json:"ip"`
				} `json:"v4"`
			} `json:"ipConfig"`
			Region    string `json:"region"`
			ProductID string `json:"productId"`
			CreatedAt string `json:"createdDate"`
		} `json:"data"`
	}

	var getResp getResponse
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(getResp.Data) == 0 {
		return nil, cloudprovider.ErrServerNotFound
	}

	inst := getResp.Data[0]
	return &cloudprovider.Server{
		ID:       fmt.Sprintf("%d", inst.InstanceID),
		Name:     inst.DisplayName,
		Status:   mapStatus(inst.Status),
		PublicIP: inst.IPConfig.V4.IP,
		Region:   inst.Region,
		Type:     inst.ProductID,
	}, nil
}

// ListServers lists servers matching the given labels.
func (p *Provider) ListServers(ctx context.Context, labels map[string]string) ([]cloudprovider.Server, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/compute/instances", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("x-request-id", fmt.Sprintf("philotes-%d", time.Now().UnixNano()))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list servers: %s", string(body))
	}

	type listResponse struct {
		Data []struct {
			InstanceID  int64  `json:"instanceId"`
			DisplayName string `json:"displayName"`
			Status      string `json:"status"`
			IPConfig    struct {
				V4 struct {
					IP string `json:"ip"`
				} `json:"v4"`
			} `json:"ipConfig"`
			Region    string `json:"region"`
			ProductID string `json:"productId"`
		} `json:"data"`
	}

	var listResp listResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Contabo doesn't support labels, so we return all servers
	// Filtering by name pattern could be done here if needed
	var result []cloudprovider.Server
	for _, inst := range listResp.Data {
		result = append(result, cloudprovider.Server{
			ID:       fmt.Sprintf("%d", inst.InstanceID),
			Name:     inst.DisplayName,
			Status:   mapStatus(inst.Status),
			PublicIP: inst.IPConfig.V4.IP,
			Region:   inst.Region,
			Type:     inst.ProductID,
		})
	}

	return result, nil
}

// GetInstanceType retrieves an instance type by name.
func (p *Provider) GetInstanceType(ctx context.Context, typeName string, region string) (*cloudprovider.InstanceType, error) {
	// Contabo product types are fixed, return hardcoded values
	types := getContaboInstanceTypes()
	for _, t := range types {
		if t.Name == typeName {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("instance type not found: %s", typeName)
}

// ListInstanceTypes lists available instance types.
func (p *Provider) ListInstanceTypes(ctx context.Context, region string) ([]cloudprovider.InstanceType, error) {
	return getContaboInstanceTypes(), nil
}

// IsServerReady checks if the server is ready.
func (p *Provider) IsServerReady(ctx context.Context, serverID string) (bool, error) {
	server, err := p.GetServer(ctx, serverID)
	if err != nil {
		return false, err
	}

	return server.Status == cloudprovider.ServerStatusRunning, nil
}

// mapStatus maps Contabo instance status to cloudprovider status.
func mapStatus(status string) cloudprovider.ServerStatus {
	switch status {
	case "provisioning", "installing":
		return cloudprovider.ServerStatusCreating
	case "running":
		return cloudprovider.ServerStatusRunning
	case "stopped":
		return cloudprovider.ServerStatusStopped
	case "error":
		return cloudprovider.ServerStatusError
	default:
		return cloudprovider.ServerStatusUnknown
	}
}

// getContaboInstanceTypes returns hardcoded Contabo instance types.
func getContaboInstanceTypes() []cloudprovider.InstanceType {
	return []cloudprovider.InstanceType{
		{Name: "V1", CPUCores: 4, MemoryMB: 8192, DiskGB: 50, HourlyCost: 0.0069, SpotSupport: false},
		{Name: "V2", CPUCores: 6, MemoryMB: 16384, DiskGB: 100, HourlyCost: 0.0125, SpotSupport: false},
		{Name: "V3", CPUCores: 8, MemoryMB: 30720, DiskGB: 200, HourlyCost: 0.0194, SpotSupport: false},
		{Name: "V4", CPUCores: 12, MemoryMB: 49152, DiskGB: 400, HourlyCost: 0.0278, SpotSupport: false},
		{Name: "V5", CPUCores: 16, MemoryMB: 65536, DiskGB: 600, HourlyCost: 0.0403, SpotSupport: false},
	}
}
