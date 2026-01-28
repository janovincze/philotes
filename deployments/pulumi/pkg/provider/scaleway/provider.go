// Package scaleway implements the CloudProvider interface for Scaleway.
package scaleway

import (
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// Provider implements provider.CloudProvider for Scaleway.
type Provider struct {
	region string
}

// New creates a new Scaleway provider.
func New(region string) *Provider {
	if region == "" {
		region = "fr-par-1"
	}
	return &Provider{region: region}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "scaleway"
}

// Ensure Provider implements CloudProvider at compile time.
var _ provider.CloudProvider = (*Provider)(nil)
