// Package exoscale implements the CloudProvider interface for Exoscale.
package exoscale

import (
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// Provider implements provider.CloudProvider for Exoscale.
type Provider struct {
	zone string
}

// New creates a new Exoscale provider.
func New(zone string) *Provider {
	if zone == "" {
		zone = "de-fra-1" // Frankfurt, Germany (default)
	}
	return &Provider{zone: zone}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "exoscale"
}

// Zone returns the provider zone.
func (p *Provider) Zone() string {
	return p.zone
}

// Ensure Provider implements CloudProvider at compile time.
var _ provider.CloudProvider = (*Provider)(nil)
