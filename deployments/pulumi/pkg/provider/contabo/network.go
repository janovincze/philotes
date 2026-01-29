package contabo

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateNetwork creates a private network for Contabo.
// Note: Contabo VPS instances are on shared networking by default.
// Private networking requires manual VPN/WireGuard setup via cloud-init.
func (p *Provider) CreateNetwork(ctx *pulumi.Context, name string, opts provider.NetworkOptions) (*provider.NetworkResult, error) {
	ctx.Log.Warn("Contabo: Private networking requires manual VPN/WireGuard configuration via cloud-init", nil)

	// Return synthetic IDs - actual network setup is done via cloud-init
	return &provider.NetworkResult{
		NetworkID: pulumi.ID(name + "-network").ToIDOutput(),
		SubnetID:  pulumi.ID(name + "-subnet").ToIDOutput(),
	}, nil
}

// CreateFirewall creates firewall rules for Contabo.
// Note: Contabo firewall rules are configured via cloud-init iptables rules.
func (p *Provider) CreateFirewall(ctx *pulumi.Context, name string, rules []provider.FirewallRule) (*provider.FirewallResult, error) {
	ctx.Log.Info("Contabo: Firewall rules will be configured via cloud-init iptables", nil)

	// Return synthetic ID - actual firewall is configured via cloud-init
	return &provider.FirewallResult{
		FirewallID: pulumi.ID(name + "-firewall").ToIDOutput(),
	}, nil
}
