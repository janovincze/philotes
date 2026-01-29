// Package ovh implements the CloudProvider interface for OVHcloud.
// Note: OVH's Pulumi provider focuses on Managed Kubernetes (OKE) rather than
// raw VM instances. This provider uses OVH Managed Kubernetes for cluster
// deployment, which simplifies infrastructure management.
package ovh

import (
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// Provider implements provider.CloudProvider for OVHcloud.
// OVH uses Managed Kubernetes, so some CloudProvider methods
// create synthetic outputs while the actual cluster is managed.
type Provider struct {
	region string
}

// New creates a new OVHcloud provider.
func New(region string) *Provider {
	if region == "" {
		region = "GRA7" // Gravelines, France (default)
	}
	return &Provider{region: region}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ovh"
}

// Region returns the provider region.
func (p *Provider) Region() string {
	return p.region
}

// Ensure Provider implements CloudProvider at compile time.
var _ provider.CloudProvider = (*Provider)(nil)
