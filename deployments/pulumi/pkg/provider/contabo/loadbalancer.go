package contabo

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// CreateLoadBalancer creates a load balancer for Contabo.
// Note: Contabo does not offer a managed load balancer service.
// Load balancing must be implemented using:
// - HAProxy on a dedicated VPS
// - nginx reverse proxy
// - Kubernetes ingress controller (for K8s deployments)
//
// For Philotes deployments, the ingress-nginx controller provides
// load balancing functionality within the K3s cluster.
func (p *Provider) CreateLoadBalancer(ctx *pulumi.Context, name string, opts provider.LBOptions) (*provider.LBResult, error) {
	ctx.Log.Warn("Contabo: No managed load balancer. Using Kubernetes ingress-nginx controller.", nil)
	ctx.Log.Info("Load balancing will be handled by the K3s ingress controller on the control plane node.", nil)

	// Return synthetic outputs - LB functionality provided by ingress controller
	return &provider.LBResult{
		LBID:     pulumi.ID(name + "-ingress").ToIDOutput(),
		PublicIP: pulumi.String("").ToStringOutput(), // Will use control plane node IP
	}, nil
}
