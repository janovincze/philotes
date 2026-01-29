package contabo

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateServer creates a Contabo VPS instance.
// Note: Contabo doesn't have a Pulumi provider, so this uses the command
// provider to configure pre-existing VPS instances via SSH.
//
// For production use, VPS instances should be pre-provisioned through the
// Contabo control panel or API, then configured here via cloud-init/SSH.
func (p *Provider) CreateServer(ctx *pulumi.Context, name string, opts provider.ServerOptions) (*provider.ServerResult, error) {
	// Contabo requires manual VPS provisioning through their control panel
	// This implementation expects an existing VPS and configures it via SSH
	
	ctx.Log.Warn("Contabo: VPS instances must be pre-provisioned. This will configure an existing instance.", nil)

	// For now, create a placeholder that documents the limitation
	// In a full implementation, this would:
	// 1. Use Contabo API to provision VPS (if API key provided)
	// 2. Wait for VPS to be ready
	// 3. Configure via SSH

	// Create a remote command to verify SSH connectivity (if IP provided via labels)
	publicIP := ""
	if opts.Labels != nil {
		if ip, ok := opts.Labels["ip"]; ok {
			publicIP = ip
		}
	}

	if publicIP != "" && opts.SSHPublicKey != "" {
		// Verify connectivity and run cloud-init
		_, err := remote.NewCommand(ctx, name+"-init", &remote.CommandArgs{
			Connection: &remote.ConnectionArgs{
				Host:       pulumi.String(publicIP),
				User:       pulumi.String("root"),
				PrivateKey: pulumi.String(""), // Would need private key
			},
			Create: pulumi.String("echo 'Contabo VPS configured'"),
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to configure VPS: %v", err), nil)
		}
	}

	// Return synthetic outputs
	return &provider.ServerResult{
		ServerID:  pulumi.ID(name).ToIDOutput(),
		PublicIP:  pulumi.String(publicIP).ToStringOutput(),
		PrivateIP: pulumi.String("").ToStringOutput(), // Requires VPN setup
		SSHKeyID:  pulumi.ID("manual").ToIDOutput(),
	}, nil
}
