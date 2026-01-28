// Package config provides configuration loading from Pulumi stack config.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Config holds all configuration for a Philotes deployment.
type Config struct {
	// Provider is the cloud provider name (hetzner, scaleway).
	Provider string
	// Region is the cloud provider region.
	Region string
	// Environment is the deployment environment (dev, staging, production).
	Environment string
	// ControlPlaneType is the server type for the control plane node.
	ControlPlaneType string
	// WorkerType is the server type for worker nodes.
	WorkerType string
	// WorkerCount is the number of worker nodes.
	WorkerCount int
	// StorageSizeGB is the block storage size in GB.
	StorageSizeGB int
	// SSHPublicKey is the contents of the SSH public key.
	SSHPublicKey string
	// SSHPrivateKeyPath is the path to the SSH private key file.
	SSHPrivateKeyPath string
}

// HetznerDefaults returns default values for Hetzner Cloud.
func HetznerDefaults() map[string]string {
	return map[string]string{
		"region":           "nbg1",
		"controlPlaneType": "cx22",
		"workerType":       "cx32",
	}
}

// ScalewayDefaults returns default values for Scaleway.
func ScalewayDefaults() map[string]string {
	return map[string]string{
		"region":           "fr-par-1",
		"controlPlaneType": "DEV1-M",
		"workerType":       "DEV1-L",
	}
}

// LoadConfig loads configuration from the Pulumi stack.
func LoadConfig(ctx *pulumi.Context) (*Config, error) {
	cfg := pulumiconfig.New(ctx, "philotes")

	provider := cfg.Get("provider")
	if provider == "" {
		provider = "hetzner"
	}

	// Select defaults based on provider
	var defaults map[string]string
	switch provider {
	case "hetzner":
		defaults = HetznerDefaults()
	case "scaleway":
		defaults = ScalewayDefaults()
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: hetzner, scaleway)", provider)
	}

	region := cfg.Get("region")
	if region == "" {
		region = defaults["region"]
	}

	environment := cfg.Get("environment")
	if environment == "" {
		environment = "dev"
	}

	controlPlaneType := cfg.Get("controlPlaneType")
	if controlPlaneType == "" {
		controlPlaneType = defaults["controlPlaneType"]
	}

	workerType := cfg.Get("workerType")
	if workerType == "" {
		workerType = defaults["workerType"]
	}

	workerCountStr := cfg.Get("workerCount")
	workerCount := 2
	if workerCountStr != "" {
		var err error
		workerCount, err = strconv.Atoi(workerCountStr)
		if err != nil {
			return nil, fmt.Errorf("invalid workerCount: %w", err)
		}
	}

	storageSizeStr := cfg.Get("storageSizeGB")
	storageSizeGB := 50
	if storageSizeStr != "" {
		var err error
		storageSizeGB, err = strconv.Atoi(storageSizeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid storageSizeGB: %w", err)
		}
	}

	sshKeyPath := cfg.Get("sshPublicKeyPath")
	if sshKeyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		sshKeyPath = filepath.Join(home, ".ssh", "id_rsa.pub")
	}

	sshPublicKey, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH public key from %s: %w", sshKeyPath, err)
	}

	// Derive private key path from public key path (remove .pub suffix)
	sshPrivateKeyPath := cfg.Get("sshPrivateKeyPath")
	if sshPrivateKeyPath == "" {
		if len(sshKeyPath) > 4 && sshKeyPath[len(sshKeyPath)-4:] == ".pub" {
			sshPrivateKeyPath = sshKeyPath[:len(sshKeyPath)-4]
		} else {
			home, _ := os.UserHomeDir()
			sshPrivateKeyPath = filepath.Join(home, ".ssh", "id_rsa")
		}
	}

	return &Config{
		Provider:          provider,
		Region:            region,
		Environment:       environment,
		ControlPlaneType:  controlPlaneType,
		WorkerType:        workerType,
		WorkerCount:       workerCount,
		StorageSizeGB:     storageSizeGB,
		SSHPublicKey:      string(sshPublicKey),
		SSHPrivateKeyPath: sshPrivateKeyPath,
	}, nil
}

// ResourceName returns a prefixed resource name for the environment.
func (c *Config) ResourceName(component string) string {
	return fmt.Sprintf("philotes-%s-%s", c.Environment, component)
}
