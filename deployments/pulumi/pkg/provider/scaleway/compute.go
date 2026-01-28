package scaleway

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateServer creates a Scaleway instance with cloud-init.
func (p *Provider) CreateServer(ctx *pulumi.Context, name string, opts provider.ServerOptions) (*provider.ServerResult, error) {
	image := opts.Image
	if image == "" {
		image = "ubuntu_jammy"
	}

	zone := regionToZone(p.region)

	// Create SSH key
	sshKey, err := scaleway.NewIamSshKey(ctx, name+"-key", &scaleway.IamSshKeyArgs{
		Name:      pulumi.String(name + "-key"),
		PublicKey: pulumi.String(opts.SSHPublicKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	// Build server arguments
	serverArgs := &scaleway.InstanceServerArgs{
		Name:  pulumi.String(name),
		Type:  pulumi.String(opts.ServerType),
		Image: pulumi.String(image),
		Zone:  pulumi.String(zone),
		Tags:  pulumi.ToStringArray(labelsToTags(opts.Labels)),
	}

	// Add cloud-init user data if provided
	if opts.UserData != nil {
		serverArgs.CloudInit = opts.UserData.ToStringOutput()
	}

	// Attach security group (firewall) if specified
	if opts.FirewallID != (pulumi.IDOutput{}) {
		serverArgs.SecurityGroupId = opts.FirewallID.ToStringOutput()
	}

	server, err := scaleway.NewInstanceServer(ctx, name, serverArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Attach to private network if specified
	var privateIP pulumi.StringOutput
	if opts.NetworkID != (pulumi.IDOutput{}) {
		nic, nicErr := scaleway.NewInstancePrivateNic(ctx, name+"-nic", &scaleway.InstancePrivateNicArgs{
			ServerId:         server.ID().ToStringOutput(),
			PrivateNetworkId: opts.NetworkID.ToStringOutput(),
		})
		if nicErr != nil {
			return nil, fmt.Errorf("failed to attach server to private network: %w", nicErr)
		}
		privateIP = nic.ID().ToStringOutput()
	}

	return &provider.ServerResult{
		ServerID:  server.ID(),
		PublicIP:  server.PublicIp,
		PrivateIP: privateIP,
		SSHKeyID:  sshKey.ID(),
	}, nil
}

// labelsToTags converts a map of labels to Scaleway tags (key=value format).
func labelsToTags(labels map[string]string) []string {
	tags := make([]string, 0, len(labels))
	for k, v := range labels {
		tags = append(tags, fmt.Sprintf("%s=%s", k, v))
	}
	return tags
}
