package ovh

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateLoadBalancer creates a load balancer for OVHcloud.
// Note: OVH Managed Kubernetes provides automatic load balancer provisioning
// through Kubernetes Service type LoadBalancer. The OVH CCM (Cloud Controller
// Manager) handles the integration automatically.
func (p *Provider) CreateLoadBalancer(ctx *pulumi.Context, name string, opts provider.LBOptions) (*provider.LBResult, error) {
	// OVH Managed Kubernetes automatically provisions load balancers
	// when Kubernetes Services of type LoadBalancer are created.
	// The ingress-nginx controller will get an external IP automatically.

	ctx.Log.Info("OVH: Load balancer will be auto-provisioned by OVH Cloud Controller Manager via Kubernetes Service type LoadBalancer", nil)

	// Return synthetic outputs - actual LB IP will come from K8s Service
	return &provider.LBResult{
		LBID:     pulumi.ID(fmt.Sprintf("%s-lb", name)).ToIDOutput(),
		PublicIP: pulumi.String("pending").ToStringOutput(), // Will be assigned by OVH CCM
	}, nil
}
