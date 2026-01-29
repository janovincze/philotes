package exoscale

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateLoadBalancer creates an Exoscale Network Load Balancer (NLB).
func (p *Provider) CreateLoadBalancer(ctx *pulumi.Context, name string, opts provider.LBOptions) (*provider.LBResult, error) {
	zone := opts.Region
	if zone == "" {
		zone = p.zone
	}

	// Create NLB
	nlb, err := exoscale.NewNlb(ctx, name, &exoscale.NlbArgs{
		Zone:        pulumi.String(zone),
		Name:        pulumi.String(name),
		Description: pulumi.String("Philotes cluster load balancer"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create NLB: %w", err)
	}

	// Create services (port mappings)
	for i, port := range opts.Ports {
		service, err := exoscale.NewNlbService(ctx, fmt.Sprintf("%s-svc-%d", name, i), &exoscale.NlbServiceArgs{
			Zone:           pulumi.String(zone),
			NlbId:          nlb.ID().ToStringOutput(),
			Name:           pulumi.String(fmt.Sprintf("%s-port-%d", name, port.ListenPort)),
			Port:           pulumi.Int(port.ListenPort),
			TargetPort:     pulumi.Int(port.TargetPort),
			Protocol:       pulumi.String(port.Protocol),
			Strategy:       pulumi.String("round-robin"),
			InstancePoolId: pulumi.String(""), // Will be set if using instance pools
			Healthchecks: exoscale.NlbServiceHealthcheckArray{
				&exoscale.NlbServiceHealthcheckArgs{
					Port:     pulumi.Int(port.TargetPort),
					Mode:     pulumi.String("tcp"),
					Interval: pulumi.Int(10),
					Timeout:  pulumi.Int(5),
					Retries:  pulumi.Int(3),
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create service %d: %w", i, err)
		}
		_ = service // Use the service resource
	}

	return &provider.LBResult{
		LBID:     nlb.ID(),
		PublicIP: nlb.IpAddress,
	}, nil
}
