// Package sshkeys provides SSH key management for Pulumi deployments.
// It supports multiple sources: Pulumi secrets (encrypted), HashiCorp Vault,
// and local files (fallback for development).
package sshkeys

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// SSHKeySource defines where to load SSH keys from.
type SSHKeySource string

const (
	// SourcePulumi loads SSH key from Pulumi encrypted secrets.
	// Key is stored encrypted in Pulumi config and decrypted at runtime.
	SourcePulumi SSHKeySource = "pulumi"

	// SourceVault loads SSH key from HashiCorp Vault.
	// Recommended for production deployments.
	SourceVault SSHKeySource = "vault"

	// SourceFile loads SSH key from a local file.
	// WARNING: Less secure, use only for development.
	SourceFile SSHKeySource = "file"
)

// SSHKeyProvider provides SSH private keys from various sources.
type SSHKeyProvider interface {
	// GetPrivateKey returns the SSH private key as a Pulumi StringOutput.
	// The key is handled securely and not stored in plain text in Pulumi state.
	GetPrivateKey(ctx *pulumi.Context) (pulumi.StringOutput, error)

	// Source returns the source type of this provider.
	Source() SSHKeySource
}

// Options configures the SSH key provider.
type Options struct {
	// FilePath is the path to the SSH private key file (for SourceFile).
	FilePath string

	// VaultAddress is the Vault server address (for SourceVault).
	VaultAddress string

	// VaultSecretPath is the path to the SSH key secret in Vault (for SourceVault).
	// Default: "secret/data/philotes/ssh-key"
	VaultSecretPath string

	// VaultAuthMethod is the Vault authentication method (for SourceVault).
	// Supported: "token", "kubernetes"
	VaultAuthMethod string

	// VaultToken is the Vault token (for token auth).
	VaultToken string

	// VaultRole is the Vault role for Kubernetes auth.
	VaultRole string
}

// NewSSHKeyProvider creates an SSH key provider based on the specified source.
func NewSSHKeyProvider(source SSHKeySource, opts Options) (SSHKeyProvider, error) {
	switch source {
	case SourcePulumi:
		return NewPulumiProvider(), nil

	case SourceVault:
		if opts.VaultAddress == "" {
			return nil, fmt.Errorf("vault address is required for vault source")
		}
		secretPath := opts.VaultSecretPath
		if secretPath == "" {
			secretPath = "secret/data/philotes/ssh-key"
		}
		return NewVaultProvider(opts.VaultAddress, secretPath, opts.VaultAuthMethod, opts.VaultToken, opts.VaultRole), nil

	case SourceFile:
		if opts.FilePath == "" {
			return nil, fmt.Errorf("file path is required for file source")
		}
		return NewFileProvider(opts.FilePath), nil

	default:
		return nil, fmt.Errorf("unsupported SSH key source: %s (supported: pulumi, vault, file)", source)
	}
}

// ParseSSHKeySource parses a string into an SSHKeySource.
func ParseSSHKeySource(s string) (SSHKeySource, error) {
	switch s {
	case "pulumi", "":
		return SourcePulumi, nil
	case "vault":
		return SourceVault, nil
	case "file":
		return SourceFile, nil
	default:
		return "", fmt.Errorf("invalid SSH key source: %s (valid: pulumi, vault, file)", s)
	}
}
