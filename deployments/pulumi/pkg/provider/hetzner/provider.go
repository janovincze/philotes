// Package hetzner implements the CloudProvider interface for Hetzner Cloud.
package hetzner

import (
	"github.com/janovincze/philotes/deployments/pulumi/pkg/provider"
)

// Provider implements provider.CloudProvider for Hetzner Cloud.
type Provider struct {
	region string
}

// New creates a new Hetzner Cloud provider.
func New(region string) *Provider {
	if region == "" {
		region = "nbg1"
	}
	return &Provider{region: region}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "hetzner"
}

// Ensure Provider implements CloudProvider at compile time.
var _ provider.CloudProvider = (*Provider)(nil)
