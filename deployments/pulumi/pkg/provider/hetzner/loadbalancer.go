package hetzner

import (
	"fmt"

	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateLoadBalancer creates a Hetzner Cloud load balancer.
func (p *Provider) CreateLoadBalancer(ctx *pulumi.Context, name string, opts provider.LBOptions) (*provider.LBResult, error) {
	region := opts.Region
	if region == "" {
		region = p.region
	}

	lb, err := hcloud.NewLoadBalancer(ctx, name, &hcloud.LoadBalancerArgs{
		Name:             pulumi.String(name),
		LoadBalancerType: pulumi.String("lb11"),
		Location:         pulumi.String(region),
		Labels: pulumi.StringMap{
			"managed-by": pulumi.String("pulumi"),
			"project":    pulumi.String("philotes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Attach to network
	if opts.NetworkID != (pulumi.IDOutput{}) {
		_, err = hcloud.NewLoadBalancerNetwork(ctx, name+"-net", &hcloud.LoadBalancerNetworkArgs{
			LoadBalancerId: lb.ID().ToStringOutput().ApplyT(func(id string) int {
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
			return nil, fmt.Errorf("failed to attach load balancer to network: %w", err)
		}
	}

	// Add server targets
	for i, serverID := range opts.TargetServerIDs {
		_, err = hcloud.NewLoadBalancerTarget(ctx, fmt.Sprintf("%s-target-%d", name, i), &hcloud.LoadBalancerTargetArgs{
			Type: pulumi.String("server"),
			LoadBalancerId: lb.ID().ToStringOutput().ApplyT(func(id string) int {
				var sid int
				fmt.Sscanf(id, "%d", &sid)
				return sid
			}).(pulumi.IntOutput),
			ServerId: serverID.ToStringOutput().ApplyT(func(id string) int {
				var sid int
				fmt.Sscanf(id, "%d", &sid)
				return sid
			}).(pulumi.IntOutput),
			UsePrivateIp: pulumi.Bool(true),
		}, pulumi.DependsOn([]pulumi.Resource{}))
		if err != nil {
			return nil, fmt.Errorf("failed to add target %d: %w", i, err)
		}
	}

	// Add services (port mappings)
	for i, port := range opts.Ports {
		_, err = hcloud.NewLoadBalancerService(ctx, fmt.Sprintf("%s-svc-%d", name, i), &hcloud.LoadBalancerServiceArgs{
			LoadBalancerId: lb.ID().ToStringOutput().ApplyT(func(id string) string {
				return id
			}).(pulumi.StringOutput),
			Protocol:        pulumi.String(port.Protocol),
			ListenPort:      pulumi.Int(port.ListenPort),
			DestinationPort: pulumi.Int(port.TargetPort),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add service %d: %w", i, err)
		}
	}

	return &provider.LBResult{
		LBID:     lb.ID(),
		PublicIP: lb.Ipv4,
	}, nil
}
