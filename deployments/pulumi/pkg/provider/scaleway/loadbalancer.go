package scaleway

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateLoadBalancer creates a Scaleway load balancer.
func (p *Provider) CreateLoadBalancer(ctx *pulumi.Context, name string, opts provider.LBOptions) (*provider.LBResult, error) {
	zone := regionToZone(p.region)

	lbArgs := &scaleway.LoadbalancerArgs{
		Name: pulumi.String(name),
		Type: pulumi.String("LB-S"),
		Tags: pulumi.ToStringArray([]string{"managed-by=pulumi", "project=philotes"}),
	}

	// Attach to private network if specified
	if opts.NetworkID != (pulumi.IDOutput{}) {
		lbArgs.PrivateNetworks = scaleway.LoadbalancerPrivateNetworkArray{
			&scaleway.LoadbalancerPrivateNetworkArgs{
				PrivateNetworkId: opts.NetworkID.ToStringOutput(),
				Zone:             pulumi.String(zone),
			},
		}
	}

	lb, err := scaleway.NewLoadbalancer(ctx, name, lbArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Create backend and frontend for each port mapping
	for i, port := range opts.Ports {
		backend, backendErr := scaleway.NewLoadbalancerBackend(ctx, fmt.Sprintf("%s-backend-%d", name, i), &scaleway.LoadbalancerBackendArgs{
			LbId:            lb.ID().ToStringOutput(),
			Name:            pulumi.String(fmt.Sprintf("%s-backend-%d", name, i)),
			ForwardProtocol: pulumi.String(mapLBProtocol(port.Protocol)),
			ForwardPort:     pulumi.Int(port.TargetPort),
		})
		if backendErr != nil {
			return nil, fmt.Errorf("failed to create backend %d: %w", i, backendErr)
		}

		_, frontendErr := scaleway.NewLoadbalancerFrontend(ctx, fmt.Sprintf("%s-frontend-%d", name, i), &scaleway.LoadbalancerFrontendArgs{
			LbId:        lb.ID().ToStringOutput(),
			BackendId:   backend.ID().ToStringOutput(),
			Name:        pulumi.String(fmt.Sprintf("%s-frontend-%d", name, i)),
			InboundPort: pulumi.Int(port.ListenPort),
		})
		if frontendErr != nil {
			return nil, fmt.Errorf("failed to create frontend %d: %w", i, frontendErr)
		}
	}

	return &provider.LBResult{
		LBID:     lb.ID(),
		PublicIP: lb.IpAddress,
	}, nil
}

// mapLBProtocol maps generic protocol names to Scaleway LB protocol names.
func mapLBProtocol(protocol string) string {
	switch protocol {
	case "tcp":
		return "tcp"
	case "http":
		return "http"
	default:
		return "tcp"
	}
}
