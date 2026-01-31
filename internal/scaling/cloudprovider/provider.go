// Package cloudprovider provides abstractions for cloud provider node management.
package cloudprovider

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Common errors for cloud provider operations.
var (
	ErrServerNotFound     = errors.New("server not found")
	ErrQuotaExceeded      = errors.New("quota exceeded")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRateLimited        = errors.New("rate limited")
	ErrServerCreating     = errors.New("server still creating")
)

// ServerStatus represents the status of a cloud server.
type ServerStatus string

const (
	ServerStatusCreating   ServerStatus = "creating"
	ServerStatusStarting   ServerStatus = "starting"
	ServerStatusRunning    ServerStatus = "running"
	ServerStatusStopping   ServerStatus = "stopping"
	ServerStatusStopped    ServerStatus = "stopped"
	ServerStatusDeleting   ServerStatus = "deleting"
	ServerStatusDeleted    ServerStatus = "deleted"
	ServerStatusError      ServerStatus = "error"
	ServerStatusUnknown    ServerStatus = "unknown"
)

// IsRunning returns true if the server is running.
func (s ServerStatus) IsRunning() bool {
	return s == ServerStatusRunning
}

// IsTerminated returns true if the server is terminated.
func (s ServerStatus) IsTerminated() bool {
	return s == ServerStatusDeleted || s == ServerStatusError
}

// Server represents a cloud provider server instance.
type Server struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Status     ServerStatus      `json:"status"`
	PublicIP   string            `json:"public_ip,omitempty"`
	PrivateIP  string            `json:"private_ip,omitempty"`
	Region     string            `json:"region"`
	Type       string            `json:"type"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// InstanceType represents a cloud provider instance type.
type InstanceType struct {
	Name        string  `json:"name"`
	CPUCores    int     `json:"cpu_cores"`
	MemoryMB    int     `json:"memory_mb"`
	DiskGB      int     `json:"disk_gb"`
	HourlyCost  float64 `json:"hourly_cost"`
	SpotSupport bool    `json:"spot_support"`
}

// CreateServerOptions contains options for creating a server.
type CreateServerOptions struct {
	Name         string
	Region       string
	InstanceType string
	Image        string
	SSHKeyIDs    []string
	UserData     string
	Labels       map[string]string
	NetworkID    string
	FirewallID   string
	UseSpot      bool
}

// Validate validates the server creation options.
func (o *CreateServerOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}
	if o.Region == "" {
		return fmt.Errorf("region is required")
	}
	if o.InstanceType == "" {
		return fmt.Errorf("instance_type is required")
	}
	if o.Image == "" {
		return fmt.Errorf("image is required")
	}
	return nil
}

// NodeProvider defines the interface for cloud provider node management.
type NodeProvider interface {
	// Server management
	CreateServer(ctx context.Context, opts CreateServerOptions) (*Server, error)
	DeleteServer(ctx context.Context, serverID string) error
	GetServer(ctx context.Context, serverID string) (*Server, error)
	ListServers(ctx context.Context, labels map[string]string) ([]Server, error)

	// Instance types
	GetInstanceType(ctx context.Context, typeName string, region string) (*InstanceType, error)
	ListInstanceTypes(ctx context.Context, region string) ([]InstanceType, error)

	// Health
	IsServerReady(ctx context.Context, serverID string) (bool, error)

	// Provider info
	Name() string
	Regions() []string
}

// ProviderConfig holds common configuration for cloud providers.
type ProviderConfig struct {
	// Provider-specific credentials (varies by provider)
	Credentials map[string]string

	// Default labels to apply to all servers
	DefaultLabels map[string]string

	// Timeout for API operations
	Timeout time.Duration

	// Retry configuration
	MaxRetries    int
	RetryInterval time.Duration
}

// DefaultConfig returns a default provider configuration.
func DefaultConfig() ProviderConfig {
	return ProviderConfig{
		Credentials:   make(map[string]string),
		DefaultLabels: make(map[string]string),
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		RetryInterval: 5 * time.Second,
	}
}

// Registry holds registered cloud providers.
type Registry struct {
	providers map[string]NodeProvider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]NodeProvider),
	}
}

// Register registers a cloud provider.
func (r *Registry) Register(provider NodeProvider) {
	r.providers[provider.Name()] = provider
}

// Get returns a registered provider by name.
func (r *Registry) Get(name string) (NodeProvider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// UserDataTemplate is a helper for generating cloud-init user data.
type UserDataTemplate struct {
	K3sVersion      string
	K3sToken        string
	ControlPlaneIP  string
	NodeLabels      map[string]string
	NodeTaints      []string
	IsControlPlane  bool
}

// GenerateK3sWorkerUserData generates cloud-init user data for a K3s worker node.
func (t *UserDataTemplate) GenerateK3sWorkerUserData() string {
	labelsArg := ""
	for k, v := range t.NodeLabels {
		if labelsArg != "" {
			labelsArg += ","
		}
		labelsArg += fmt.Sprintf("%s=%s", k, v)
	}

	taintsArg := ""
	for _, taint := range t.NodeTaints {
		if taintsArg != "" {
			taintsArg += ","
		}
		taintsArg += taint
	}

	userData := fmt.Sprintf(`#!/bin/bash
set -e

# Install K3s agent
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="%s" K3S_URL="https://%s:6443" K3S_TOKEN="%s" sh -s - agent`,
		t.K3sVersion, t.ControlPlaneIP, t.K3sToken)

	if labelsArg != "" {
		userData += fmt.Sprintf(` --node-label="%s"`, labelsArg)
	}
	if taintsArg != "" {
		userData += fmt.Sprintf(` --node-taint="%s"`, taintsArg)
	}

	return userData
}
