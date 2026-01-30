// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// GoogleConfig provides Google Workspace OIDC configuration.
type GoogleConfig struct{}

// Type returns the provider type.
func (c *GoogleConfig) Type() models.OIDCProviderType {
	return models.OIDCProviderTypeGoogle
}

// DefaultIssuerURL returns Google's OIDC issuer URL.
func (c *GoogleConfig) DefaultIssuerURL() string {
	return "https://accounts.google.com"
}

// DefaultScopes returns the default scopes for Google.
func (c *GoogleConfig) DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
	}
}

// DefaultGroupsClaim returns the default groups claim for Google.
// Note: Google Workspace uses a custom claim for groups.
func (c *GoogleConfig) DefaultGroupsClaim() string {
	return "groups"
}

// SupportsDiscovery returns true as Google supports OIDC discovery.
func (c *GoogleConfig) SupportsDiscovery() bool {
	return true
}
