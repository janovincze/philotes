// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// Auth0Config provides Auth0 OIDC configuration.
type Auth0Config struct{}

// Type returns the provider type.
func (c *Auth0Config) Type() models.OIDCProviderType {
	return models.OIDCProviderTypeAuth0
}

// DefaultIssuerURL returns an empty string as Auth0 requires a tenant-specific URL.
// Pattern: https://your-domain.auth0.com/ or https://your-domain.us.auth0.com/
func (c *Auth0Config) DefaultIssuerURL() string {
	return ""
}

// DefaultScopes returns the default scopes for Auth0.
func (c *Auth0Config) DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
	}
}

// DefaultGroupsClaim returns the default groups claim for Auth0.
// Auth0 requires custom rules/actions to include groups in the ID token.
// Common claim names: "groups", "https://your-domain/groups", or custom namespace.
func (c *Auth0Config) DefaultGroupsClaim() string {
	return "groups"
}

// SupportsDiscovery returns true as Auth0 supports OIDC discovery.
func (c *Auth0Config) SupportsDiscovery() bool {
	return true
}
