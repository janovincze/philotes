package hetzner

import (
	"fmt"

	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateServer creates a Hetzner Cloud server with cloud-init.
func (p *Provider) CreateServer(ctx *pulumi.Context, name string, opts provider.ServerOptions) (*provider.ServerResult, error) {
	image := opts.Image
	if image == "" {
		image = "ubuntu-24.04"
	}

	region := opts.Region
	if region == "" {
		region = p.region
	}

	// Create or reuse SSH key
	sshKey, err := hcloud.NewSshKey(ctx, name+"-key", &hcloud.SshKeyArgs{
		Name:      pulumi.String(name + "-key"),
		PublicKey: pulumi.String(opts.SSHPublicKey),
		Labels: pulumi.StringMap{
			"managed-by": pulumi.String("pulumi"),
			"project":    pulumi.String("philotes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	// Build server arguments
	serverArgs := &hcloud.ServerArgs{
		Name:       pulumi.String(name),
		ServerType: pulumi.String(opts.ServerType),
		Image:      pulumi.String(image),
		Location:   pulumi.String(region),
		SshKeys:    pulumi.StringArray{sshKey.ID().ToStringOutput()},
		Labels:     pulumi.ToStringMap(opts.Labels),
	}

	// Add cloud-init user data if provided
	if opts.UserData != nil {
		serverArgs.UserData = opts.UserData.ToStringOutput()
	}

	// Attach to firewall if specified
	if opts.FirewallID != (pulumi.IDOutput{}) {
		serverArgs.FirewallIds = pulumi.IntArray{
			opts.FirewallID.ToStringOutput().ApplyT(func(id string) int {
				var i int
				fmt.Sscanf(id, "%d", &i)
				return i
			}).(pulumi.IntOutput),
		}
	}

	server, err := hcloud.NewServer(ctx, name, serverArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Attach to private network if specified
	var privateIP pulumi.StringOutput
	if opts.NetworkID != (pulumi.IDOutput{}) {
		attachment, err := hcloud.NewServerNetwork(ctx, name+"-net", &hcloud.ServerNetworkArgs{
			ServerId: server.ID().ToStringOutput().ApplyT(func(id string) int {
				var i int
				fmt.Sscanf(id, "%d", &i)
				return i
			}).(pulumi.IntOutput),
			NetworkId: opts.NetworkID.ToStringOutput().ApplyT(func(id string) int {
				var i int
				fmt.Sscanf(id, "%d", &i)
				return i
			}).(pulumi.IntOutput),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to attach server to network: %w", err)
		}
		privateIP = attachment.Ip
	}

	return &provider.ServerResult{
		ServerID:  server.ID(),
		PublicIP:  server.Ipv4Address,
		PrivateIP: privateIP,
		SSHKeyID:  sshKey.ID(),
	}, nil
}
