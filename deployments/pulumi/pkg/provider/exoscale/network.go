package exoscale

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateNetwork creates an Exoscale private network.
func (p *Provider) CreateNetwork(ctx *pulumi.Context, name string, opts provider.NetworkOptions) (*provider.NetworkResult, error) {
	cidr := opts.SubnetCIDR
	if cidr == "" {
		cidr = "10.0.1.0/24"
	}

	// Create private network
	network, err := exoscale.NewPrivateNetwork(ctx, name, &exoscale.PrivateNetworkArgs{
		Zone:      pulumi.String(p.zone),
		Name:      pulumi.String(name),
		StartIp:   pulumi.String("10.0.1.10"),
		EndIp:     pulumi.String("10.0.1.254"),
		Netmask:   pulumi.String("255.255.255.0"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return &provider.NetworkResult{
		NetworkID: network.ID(),
		SubnetID:  network.ID(), // Exoscale doesn't have separate subnet resources
	}, nil
}

// CreateFirewall creates an Exoscale security group.
func (p *Provider) CreateFirewall(ctx *pulumi.Context, name string, rules []provider.FirewallRule) (*provider.FirewallResult, error) {
	// Create security group
	sg, err := exoscale.NewSecurityGroup(ctx, name, &exoscale.SecurityGroupArgs{
		Name:        pulumi.String(name),
		Description: pulumi.String("Philotes cluster security group"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}

	// Add rules
	for i, rule := range rules {
		if rule.Direction != "in" {
			continue // Exoscale security groups are ingress-only
		}

		// Parse port
		startPort := 0
		endPort := 0
		fmt.Sscanf(rule.Port, "%d", &startPort)
		endPort = startPort

		// Check for port range
		if n, _ := fmt.Sscanf(rule.Port, "%d-%d", &startPort, &endPort); n < 2 {
			endPort = startPort
		}

		for _, cidr := range rule.SourceIPs {
			// Skip IPv6 for now
			if len(cidr) > 0 && cidr[0] == ':' {
				continue
			}

			_, err = exoscale.NewSecurityGroupRule(ctx, fmt.Sprintf("%s-rule-%d-%s", name, i, cidr), &exoscale.SecurityGroupRuleArgs{
				SecurityGroupId: sg.ID().ToStringOutput(),
				Type:            pulumi.String("INGRESS"),
				Protocol:        pulumi.String(rule.Protocol),
				StartPort:       pulumi.Int(startPort),
				EndPort:         pulumi.Int(endPort),
				Cidr:            pulumi.String(cidr),
				Description:     pulumi.String(rule.Description),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create rule %d: %w", i, err)
			}
		}
	}

	return &provider.FirewallResult{
		FirewallID: sg.ID(),
	}, nil
}
