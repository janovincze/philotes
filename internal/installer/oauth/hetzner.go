// Package oauth provides OAuth 2.0 configuration for cloud providers.
package oauth

import (
	"strings"
)

// HetznerProvider implements OAuth for Hetzner Cloud.
type HetznerProvider struct {
	clientID     string
	clientSecret string
	enabled      bool
}

// HetznerConfig holds Hetzner OAuth configuration.
type HetznerConfig struct {
	ClientID     string
	ClientSecret string
	Enabled      bool
}

// NewHetznerProvider creates a new Hetzner OAuth provider.
func NewHetznerProvider(cfg HetznerConfig) *HetznerProvider {
	return &HetznerProvider{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		enabled:      cfg.Enabled,
	}
}

// Name returns the provider identifier.
func (p *HetznerProvider) Name() string {
	return "hetzner"
}

// DisplayName returns the human-readable provider name.
func (p *HetznerProvider) DisplayName() string {
	return "Hetzner Cloud"
}

// AuthorizationURL builds the OAuth authorization URL with PKCE.
func (p *HetznerProvider) AuthorizationURL(state, codeChallenge, redirectURI string) string {
	return BuildAuthURL(hetznerAuthURL, map[string]string{
		"response_type":         "code",
		"client_id":             p.clientID,
		"redirect_uri":          redirectURI,
		"scope":                 strings.Join(p.Scopes(), " "),
		"state":                 state,
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
	})
}

// TokenURL returns the token endpoint URL.
func (p *HetznerProvider) TokenURL() string {
	return hetznerTokenURL
}

// Scopes returns the required OAuth scopes for Hetzner.
// These scopes provide full access to manage cloud resources.
func (p *HetznerProvider) Scopes() []string {
	return []string{
		"read",  // Read access to account and resources
		"write", // Write access to create/modify resources
	}
}

// ClientID returns the OAuth client ID.
func (p *HetznerProvider) ClientID() string {
	return p.clientID
}

// ClientSecret returns the OAuth client secret.
func (p *HetznerProvider) ClientSecret() string {
	return p.clientSecret
}

// IsEnabled returns whether Hetzner OAuth is enabled.
func (p *HetznerProvider) IsEnabled() bool {
	return p.enabled && p.clientID != ""
}

// Hetzner OAuth endpoints.
const (
	hetznerAuthURL  = "https://console.hetzner.cloud/oauth/authorize"
	hetznerTokenURL = "https://console.hetzner.cloud/oauth/token"
)
