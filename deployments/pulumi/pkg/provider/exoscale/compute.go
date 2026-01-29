package exoscale

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateServer creates an Exoscale compute instance.
func (p *Provider) CreateServer(ctx *pulumi.Context, name string, opts provider.ServerOptions) (*provider.ServerResult, error) {
	zone := opts.Region
	if zone == "" {
		zone = p.zone
	}

	instanceType := opts.ServerType
	if instanceType == "" {
		instanceType = "standard.medium" // 2 vCPU, 4GB RAM
	}

	// Create SSH key
	sshKey, err := exoscale.NewSshKey(ctx, name+"-key", &exoscale.SshKeyArgs{
		Name:      pulumi.String(name + "-key"),
		PublicKey: pulumi.String(opts.SSHPublicKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH key: %w", err)
	}

	// Build instance arguments
	instanceArgs := &exoscale.ComputeInstanceArgs{
		Zone:         pulumi.String(zone),
		Name:         pulumi.String(name),
		TemplateId:   pulumi.String(getUbuntuTemplateID(zone)),
		Type:         pulumi.String(instanceType),
		SshKey:       sshKey.Name,
		DiskSize:     pulumi.Int(50), // 50GB disk
	}

	// Add cloud-init user data if provided
	if opts.UserData != nil {
		instanceArgs.UserData = opts.UserData.ToStringOutput()
	}

	// Attach to security group if specified
	if opts.FirewallID != (pulumi.IDOutput{}) {
		instanceArgs.SecurityGroupIds = pulumi.StringArray{opts.FirewallID.ToStringOutput()}
	}

	// Attach to private network if specified
	if opts.NetworkID != (pulumi.IDOutput{}) {
		instanceArgs.NetworkInterfaces = exoscale.ComputeInstanceNetworkInterfaceArray{
			&exoscale.ComputeInstanceNetworkInterfaceArgs{
				NetworkId: opts.NetworkID.ToStringOutput(),
			},
		}
	}

	instance, err := exoscale.NewComputeInstance(ctx, name, instanceArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Get IPs
	publicIP := instance.PublicIpAddress
	privateIP := instance.NetworkInterfaces.ApplyT(func(nics []exoscale.ComputeInstanceNetworkInterface) string {
		if len(nics) > 0 && nics[0].IpAddress != nil {
			return *nics[0].IpAddress
		}
		return ""
	}).(pulumi.StringOutput)

	return &provider.ServerResult{
		ServerID:  instance.ID(),
		PublicIP:  publicIP,
		PrivateIP: privateIP,
		SSHKeyID:  sshKey.ID(),
	}, nil
}

// getUbuntuTemplateID returns the Ubuntu 24.04 template ID for a given zone.
// These are static IDs from Exoscale's template catalog.
func getUbuntuTemplateID(zone string) string {
	// Ubuntu 24.04 LTS template IDs per zone (these may need updating)
	// Use 'exo compute instance-template list' to get current IDs
	templates := map[string]string{
		"de-fra-1": "ubuntu-24.04-lts",
		"de-muc-1": "ubuntu-24.04-lts",
		"at-vie-1": "ubuntu-24.04-lts",
		"at-vie-2": "ubuntu-24.04-lts",
		"ch-gva-2": "ubuntu-24.04-lts",
		"ch-dk-2":  "ubuntu-24.04-lts",
		"bg-sof-1": "ubuntu-24.04-lts",
	}
	if id, ok := templates[zone]; ok {
		return id
	}
	return "ubuntu-24.04-lts" // Default template name
}
