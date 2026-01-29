package ovh

import (
	"fmt"

	"github.com/ovh/pulumi-ovh/sdk/go/ovh/cloudproject"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// getServiceName retrieves the OVH project/service name from config.
func getServiceName(ctx *pulumi.Context) (string, error) {
	cfg := pulumiconfig.New(ctx, "ovh")
	serviceName := cfg.Get("serviceName")
	if serviceName == "" {
		return "", fmt.Errorf("ovh:serviceName config is required (your OVH Cloud Project ID)")
	}
	return serviceName, nil
}

// CreateNetwork creates an OVHcloud private network with a subnet.
func (p *Provider) CreateNetwork(ctx *pulumi.Context, name string, opts provider.NetworkOptions) (*provider.NetworkResult, error) {
	serviceName, err := getServiceName(ctx)
	if err != nil {
		return nil, err
	}

	subnetCIDR := opts.SubnetCIDR
	if subnetCIDR == "" {
		subnetCIDR = "10.0.1.0/24"
	}

	// Create private network
	network, err := cloudproject.NewNetworkPrivate(ctx, name, &cloudproject.NetworkPrivateArgs{
		ServiceName: pulumi.String(serviceName),
		Name:        pulumi.String(name),
		Regions:     pulumi.StringArray{pulumi.String(p.region)},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Create subnet
	subnet, err := cloudproject.NewNetworkPrivateSubnet(ctx, name+"-subnet", &cloudproject.NetworkPrivateSubnetArgs{
		ServiceName: pulumi.String(serviceName),
		NetworkId:   network.ID().ToStringOutput(),
		Region:      pulumi.String(p.region),
		Start:       pulumi.String("10.0.1.10"),
		End:         pulumi.String("10.0.1.254"),
		Network:     pulumi.String(subnetCIDR),
		Dhcp:        pulumi.Bool(true),
		NoGateway:   pulumi.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	return &provider.NetworkResult{
		NetworkID: network.ID(),
		SubnetID:  subnet.ID(),
	}, nil
}

// CreateFirewall creates firewall rules for OVHcloud.
// Note: OVH Managed Kubernetes handles network policies internally.
// For K3s deployments, firewall rules would be configured via cloud-init.
// This returns a synthetic ID as OVH doesn't have a separate firewall resource.
func (p *Provider) CreateFirewall(ctx *pulumi.Context, name string, rules []provider.FirewallRule) (*provider.FirewallResult, error) {
	// OVH Managed Kubernetes handles security internally
	// For self-managed K3s, we would configure iptables via cloud-init
	// Return a synthetic ID for interface compatibility
	return &provider.FirewallResult{
		FirewallID: pulumi.ID(name + "-firewall").ToIDOutput(),
	}, nil
}
