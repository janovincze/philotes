// Package config provides configuration loading from Pulumi stack config.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/sshkeys"
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
	// SSHPrivateKey is the SSH private key as a Pulumi StringOutput.
	// Loaded from the configured source (pulumi secrets, vault, or file).
	SSHPrivateKey pulumi.StringOutput
	// SSHKeySource indicates where the SSH private key is loaded from.
	SSHKeySource sshkeys.SSHKeySource

	// SSHPrivateKeyPath is the path to the SSH private key file.
	// Deprecated: Use SSHPrivateKey instead. Kept for backward compatibility
	// when SSHKeySource is "file".
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

// OVHDefaults returns default values for OVHcloud.
func OVHDefaults() map[string]string {
	return map[string]string{
		"region":           "GRA7",
		"controlPlaneType": "d2-4",
		"workerType":       "d2-8",
	}
}

// ExoscaleDefaults returns default values for Exoscale.
func ExoscaleDefaults() map[string]string {
	return map[string]string{
		"region":           "de-fra-1",
		"controlPlaneType": "standard.medium",
		"workerType":       "standard.large",
	}
}

// ContaboDefaults returns default values for Contabo.
func ContaboDefaults() map[string]string {
	return map[string]string{
		"region":           "EU",
		"controlPlaneType": "VPS-S",  // 4 vCPU, 8GB RAM
		"workerType":       "VPS-M",  // 6 vCPU, 16GB RAM
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
	case "ovh":
		defaults = OVHDefaults()
	case "exoscale":
		defaults = ExoscaleDefaults()
	case "contabo":
		defaults = ContaboDefaults()
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: hetzner, scaleway, ovh, exoscale, contabo)", provider)
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

	// Load SSH private key from configured source
	sshKeySourceStr := cfg.Get("sshKeySource")
	if sshKeySourceStr == "" {
		sshKeySourceStr = "file" // Default to file for backward compatibility
	}

	sshKeySource, err := sshkeys.ParseSSHKeySource(sshKeySourceStr)
	if err != nil {
		return nil, fmt.Errorf("invalid sshKeySource: %w", err)
	}

	// Build options for the SSH key provider
	sshKeyOpts := sshkeys.Options{
		FilePath:        sshPrivateKeyPath,
		VaultAddress:    cfg.Get("vaultAddress"),
		VaultSecretPath: cfg.Get("vaultSecretPath"),
		VaultAuthMethod: cfg.Get("vaultAuthMethod"),
		VaultToken:      cfg.Get("vaultToken"),
		VaultRole:       cfg.Get("vaultRole"),
	}

	// Create SSH key provider
	sshKeyProvider, err := sshkeys.NewSSHKeyProvider(sshKeySource, sshKeyOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key provider: %w", err)
	}

	// Get the SSH private key
	sshPrivateKey, err := sshKeyProvider.GetPrivateKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load SSH private key: %w", err)
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
		SSHPrivateKey:     sshPrivateKey,
		SSHKeySource:      sshKeySource,
		SSHPrivateKeyPath: sshPrivateKeyPath, // Kept for backward compatibility
	}, nil
}

// ResourceName returns a prefixed resource name for the environment.
func (c *Config) ResourceName(component string) string {
	return fmt.Sprintf("philotes-%s-%s", c.Environment, component)
}
