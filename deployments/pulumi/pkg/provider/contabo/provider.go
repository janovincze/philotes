// Package contabo implements the CloudProvider interface for Contabo.
//
// IMPORTANT: Contabo does not have a native Pulumi provider. This implementation
// uses Contabo's REST API directly for basic operations. Some features may be
// limited compared to other providers.
//
// Limitations:
// - No managed load balancer service (requires HAProxy/nginx setup)
// - No managed Kubernetes (K3s must be self-installed)
// - Limited API compared to Hetzner/Scaleway
// - Manual setup may be required for some features
//
// For full Contabo API documentation, see: https://api.contabo.com/
package contabo

import (
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// Provider implements provider.CloudProvider for Contabo.
// This provider uses Contabo's REST API for basic VPS operations.
type Provider struct {
	region string
}

// New creates a new Contabo provider.
func New(region string) *Provider {
	if region == "" {
		region = "EU" // Default to European datacenter
	}
	return &Provider{region: region}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "contabo"
}

// Region returns the provider region.
func (p *Provider) Region() string {
	return p.region
}

// Ensure Provider implements CloudProvider at compile time.
var _ provider.CloudProvider = (*Provider)(nil)
