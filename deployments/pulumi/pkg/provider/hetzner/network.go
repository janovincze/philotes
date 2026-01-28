package hetzner

import (
	"fmt"

	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateNetwork creates a Hetzner Cloud private network with a subnet.
func (p *Provider) CreateNetwork(ctx *pulumi.Context, name string, opts provider.NetworkOptions) (*provider.NetworkResult, error) {
	cidr := opts.CIDRBlock
	if cidr == "" {
		cidr = "10.0.0.0/16"
	}
	subnetCIDR := opts.SubnetCIDR
	if subnetCIDR == "" {
		subnetCIDR = "10.0.1.0/24"
	}

	network, err := hcloud.NewNetwork(ctx, name, &hcloud.NetworkArgs{
		IpRange: pulumi.String(cidr),
		Labels: pulumi.StringMap{
			"managed-by": pulumi.String("pulumi"),
			"project":    pulumi.String("philotes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Map region to Hetzner network zone
	networkZone := regionToNetworkZone(p.region)

	// Convert network ID to int for the subnet
	networkIdInt := network.ID().ToStringOutput().ApplyT(func(id string) int {
		var i int
		fmt.Sscanf(id, "%d", &i)
		return i
	}).(pulumi.IntOutput)

	subnet, err := hcloud.NewNetworkSubnet(ctx, name+"-subnet", &hcloud.NetworkSubnetArgs{
		NetworkId:   networkIdInt,
		Type:        pulumi.String("cloud"),
		NetworkZone: pulumi.String(networkZone),
		IpRange:     pulumi.String(subnetCIDR),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	return &provider.NetworkResult{
		NetworkID: network.ID(),
		SubnetID:  subnet.ID(),
	}, nil
}

// CreateFirewall creates a Hetzner Cloud firewall.
func (p *Provider) CreateFirewall(ctx *pulumi.Context, name string, rules []provider.FirewallRule) (*provider.FirewallResult, error) {
	var hcloudRules hcloud.FirewallRuleArray
	for _, rule := range rules {
		sourceIPs := pulumi.ToStringArray(rule.SourceIPs)
		hcloudRules = append(hcloudRules, &hcloud.FirewallRuleArgs{
			Direction:   pulumi.String(rule.Direction),
			Protocol:    pulumi.String(rule.Protocol),
			Port:        pulumi.String(rule.Port),
			SourceIps:   sourceIPs,
			Description: pulumi.String(rule.Description),
		})
	}

	firewall, err := hcloud.NewFirewall(ctx, name, &hcloud.FirewallArgs{
		Rules: hcloudRules,
		Labels: pulumi.StringMap{
			"managed-by": pulumi.String("pulumi"),
			"project":    pulumi.String("philotes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	return &provider.FirewallResult{
		FirewallID: firewall.ID(),
	}, nil
}

// regionToNetworkZone maps a Hetzner region to a network zone.
func regionToNetworkZone(region string) string {
	switch region {
	case "nbg1", "fsn1":
		return "eu-central"
	case "hel1":
		return "eu-central"
	case "ash":
		return "us-east"
	case "hil":
		return "us-west"
	default:
		return "eu-central"
	}
}
