package sshkeys

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// PulumiProvider loads SSH keys from Pulumi encrypted secrets.
// This is the recommended approach for most deployments as the key
// is stored encrypted in Pulumi config and decrypted only at runtime.
//
// To set the SSH key:
//
//	pulumi config set --secret philotes:sshPrivateKey "$(cat ~/.ssh/id_rsa)"
type PulumiProvider struct{}

// NewPulumiProvider creates a new Pulumi secrets-based SSH key provider.
func NewPulumiProvider() *PulumiProvider {
	return &PulumiProvider{}
}

// GetPrivateKey retrieves the SSH private key from Pulumi encrypted config.
// The key must be set using: pulumi config set --secret philotes:sshPrivateKey "..."
func (p *PulumiProvider) GetPrivateKey(ctx *pulumi.Context) (pulumi.StringOutput, error) {
	cfg := pulumiconfig.New(ctx, "philotes")

	// First check if the key exists in config (non-secret check)
	if cfg.Get("sshPrivateKey") == "" {
		return pulumi.StringOutput{}, fmt.Errorf(
			"SSH private key not found in Pulumi config. Set it using:\n" +
				"  pulumi config set --secret philotes:sshPrivateKey \"$(cat ~/.ssh/id_rsa)\"",
		)
	}

	// Get the secret value - RequireSecret panics if not found, but we checked above
	sshKey := cfg.RequireSecret("sshPrivateKey")

	ctx.Log.Info("SSH key loaded from Pulumi encrypted secrets", nil)

	return sshKey, nil
}

// Source returns the source type.
func (p *PulumiProvider) Source() SSHKeySource {
	return SourcePulumi
}
