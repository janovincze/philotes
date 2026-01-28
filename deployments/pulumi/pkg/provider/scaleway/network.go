package scaleway

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateNetwork creates a Scaleway VPC private network with a subnet.
func (p *Provider) CreateNetwork(ctx *pulumi.Context, name string, opts provider.NetworkOptions) (*provider.NetworkResult, error) {
	zone := regionToZone(p.region)

	vpc, err := scaleway.NewVpcPrivateNetwork(ctx, name, &scaleway.VpcPrivateNetworkArgs{
		Name: pulumi.String(name),
		Tags: pulumi.ToStringArray([]string{"managed-by:pulumi", "project:philotes"}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC private network: %w", err)
	}

	subnetCIDR := opts.SubnetCIDR
	if subnetCIDR == "" {
		subnetCIDR = "10.0.1.0/24"
	}

	subnet, err := scaleway.NewVpcPublicGatewayDhcp(ctx, name+"-dhcp", &scaleway.VpcPublicGatewayDhcpArgs{
		Subnet: pulumi.String(subnetCIDR),
		Zone:   pulumi.String(zone),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DHCP config: %w", err)
	}

	return &provider.NetworkResult{
		NetworkID: vpc.ID(),
		SubnetID:  subnet.ID(),
	}, nil
}

// CreateFirewall creates Scaleway security group rules.
func (p *Provider) CreateFirewall(ctx *pulumi.Context, name string, rules []provider.FirewallRule) (*provider.FirewallResult, error) {
	sg, err := scaleway.NewInstanceSecurityGroup(ctx, name, &scaleway.InstanceSecurityGroupArgs{
		Name:                  pulumi.String(name),
		InboundDefaultPolicy:  pulumi.String("drop"),
		OutboundDefaultPolicy: pulumi.String("accept"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	for i, rule := range rules {
		if rule.Direction != "in" {
			continue
		}
		_, ruleErr := scaleway.NewInstanceSecurityGroupRules(ctx, fmt.Sprintf("%s-rule-%d", name, i), &scaleway.InstanceSecurityGroupRulesArgs{
			SecurityGroupId: sg.ID().ToStringOutput(),
			InboundRules: scaleway.InstanceSecurityGroupRulesInboundRuleArray{
				&scaleway.InstanceSecurityGroupRulesInboundRuleArgs{
					Action:   pulumi.String("accept"),
					Protocol: pulumi.String(mapProtocol(rule.Protocol)),
					Port:     pulumi.Int(parsePort(rule.Port)),
					Ip:       pulumi.String(firstPublicCIDR(rule.SourceIPs)),
				},
			},
		})
		if ruleErr != nil {
			return nil, fmt.Errorf("failed to create security group rule %d: %w", i, ruleErr)
		}
	}

	return &provider.FirewallResult{
		FirewallID: sg.ID(),
	}, nil
}

// regionToZone maps a Scaleway region to a zone.
func regionToZone(region string) string {
	// Scaleway regions are like "fr-par-1", zones are the same
	return region
}

// mapProtocol maps generic protocol names to Scaleway protocol names.
func mapProtocol(protocol string) string {
	switch protocol {
	case "tcp":
		return "TCP"
	case "udp":
		return "UDP"
	case "icmp":
		return "ICMP"
	default:
		return "TCP"
	}
}

// parsePort extracts a port number from a port string.
func parsePort(port string) int {
	var p int
	fmt.Sscanf(port, "%d", &p)
	return p
}

// firstPublicCIDR returns the first IPv4 CIDR from the list.
func firstPublicCIDR(cidrs []string) string {
	for _, cidr := range cidrs {
		if cidr != "::/0" {
			return cidr
		}
	}
	return "0.0.0.0/0"
}
