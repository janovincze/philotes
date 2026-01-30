// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// GenericConfig provides configuration for any OIDC-compliant provider.
type GenericConfig struct{}

// Type returns the provider type.
func (c *GenericConfig) Type() models.OIDCProviderType {
	return models.OIDCProviderTypeGeneric
}

// DefaultIssuerURL returns an empty string as generic providers require configuration.
func (c *GenericConfig) DefaultIssuerURL() string {
	return ""
}

// DefaultScopes returns the standard OIDC scopes.
func (c *GenericConfig) DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
	}
}

// DefaultGroupsClaim returns the default groups claim.
func (c *GenericConfig) DefaultGroupsClaim() string {
	return "groups"
}

// SupportsDiscovery returns true as most OIDC providers support discovery.
func (c *GenericConfig) SupportsDiscovery() bool {
	return true
}
