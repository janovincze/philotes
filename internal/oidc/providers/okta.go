// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// OktaConfig provides Okta OIDC configuration.
type OktaConfig struct{}

// Type returns the provider type.
func (c *OktaConfig) Type() models.OIDCProviderType {
	return models.OIDCProviderTypeOkta
}

// DefaultIssuerURL returns an empty string as Okta requires a tenant-specific URL.
// Example: https://your-domain.okta.com or https://your-domain.oktapreview.com
func (c *OktaConfig) DefaultIssuerURL() string {
	return ""
}

// DefaultScopes returns the default scopes for Okta.
func (c *OktaConfig) DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
		"groups",
	}
}

// DefaultGroupsClaim returns the default groups claim for Okta.
func (c *OktaConfig) DefaultGroupsClaim() string {
	return "groups"
}

// SupportsDiscovery returns true as Okta supports OIDC discovery.
func (c *OktaConfig) SupportsDiscovery() bool {
	return true
}
