// Package providers defines OIDC provider configurations.
package providers

import (
	"github.com/janovincze/philotes/internal/api/models"
)

// ProviderConfig defines provider-specific configuration.
type ProviderConfig interface {
	// Type returns the provider type.
	Type() models.OIDCProviderType

	// DefaultIssuerURL returns the default issuer URL for the provider.
	DefaultIssuerURL() string

	// DefaultScopes returns the default scopes for the provider.
	DefaultScopes() []string

	// DefaultGroupsClaim returns the default groups claim name.
	DefaultGroupsClaim() string

	// SupportsDiscovery returns whether the provider supports OIDC discovery.
	SupportsDiscovery() bool
}

// Registry holds all provider configurations.
type Registry struct {
	configs map[models.OIDCProviderType]ProviderConfig
}

// NewRegistry creates a new provider registry with all supported providers.
func NewRegistry() *Registry {
	r := &Registry{
		configs: make(map[models.OIDCProviderType]ProviderConfig),
	}

	// Register all supported providers
	r.Register(&GoogleConfig{})
	r.Register(&OktaConfig{})
	r.Register(&AzureADConfig{})
	r.Register(&Auth0Config{})
	r.Register(&GenericConfig{})

	return r
}

// Register adds a provider configuration to the registry.
func (r *Registry) Register(config ProviderConfig) {
	r.configs[config.Type()] = config
}

// Get returns the configuration for a provider type.
func (r *Registry) Get(providerType models.OIDCProviderType) (ProviderConfig, bool) {
	config, ok := r.configs[providerType]
	return config, ok
}

// List returns all registered provider configurations.
func (r *Registry) List() []ProviderConfig {
	configs := make([]ProviderConfig, 0, len(r.configs))
	for _, config := range r.configs {
		configs = append(configs, config)
	}
	return configs
}

// ApplyDefaults applies provider-specific defaults to a create request.
func (r *Registry) ApplyDefaults(req *models.CreateOIDCProviderRequest) {
	config, ok := r.Get(req.ProviderType)
	if !ok {
		return
	}

	if len(req.Scopes) == 0 {
		req.Scopes = config.DefaultScopes()
	}

	if req.GroupsClaim == "" {
		req.GroupsClaim = config.DefaultGroupsClaim()
	}
}
