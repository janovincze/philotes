// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// AzureADConfig provides Azure AD / Entra ID OIDC configuration.
type AzureADConfig struct{}

// Type returns the provider type.
func (c *AzureADConfig) Type() models.OIDCProviderType {
	return models.OIDCProviderTypeAzureAD
}

// DefaultIssuerURL returns an empty string as Azure AD requires a tenant-specific URL.
// Common patterns:
// - Single tenant: https://login.microsoftonline.com/{tenant-id}/v2.0
// - Multi-tenant: https://login.microsoftonline.com/common/v2.0
// - Organizations: https://login.microsoftonline.com/organizations/v2.0
func (c *AzureADConfig) DefaultIssuerURL() string {
	return ""
}

// DefaultScopes returns the default scopes for Azure AD.
func (c *AzureADConfig) DefaultScopes() []string {
	return []string{
		"openid",
		"profile",
		"email",
	}
}

// DefaultGroupsClaim returns the default groups claim for Azure AD.
// Azure AD uses "groups" for security group IDs by default.
// For group names, you may need to configure optional claims.
func (c *AzureADConfig) DefaultGroupsClaim() string {
	return "groups"
}

// SupportsDiscovery returns true as Azure AD supports OIDC discovery.
func (c *AzureADConfig) SupportsDiscovery() bool {
	return true
}
