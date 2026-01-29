// Package oauth provides OAuth 2.0 configuration for cloud providers.
package oauth

import (
	"strings"
)

// OVHProvider implements OAuth for OVHcloud.
type OVHProvider struct {
	clientID     string
	clientSecret string
	enabled      bool
	region       string // eu, ca, us
}

// OVHConfig holds OVH OAuth configuration.
type OVHConfig struct {
	ClientID     string
	ClientSecret string
	Enabled      bool
	Region       string // Defaults to "eu" if empty
}

// NewOVHProvider creates a new OVH OAuth provider.
func NewOVHProvider(cfg OVHConfig) *OVHProvider {
	region := cfg.Region
	if region == "" {
		region = "eu"
	}
	return &OVHProvider{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		enabled:      cfg.Enabled,
		region:       region,
	}
}

// Name returns the provider identifier.
func (p *OVHProvider) Name() string {
	return "ovh"
}

// DisplayName returns the human-readable provider name.
func (p *OVHProvider) DisplayName() string {
	return "OVHcloud"
}

// AuthorizationURL builds the OAuth authorization URL with PKCE.
func (p *OVHProvider) AuthorizationURL(state, codeChallenge, redirectURI string) string {
	return BuildAuthURL(p.authURL(), map[string]string{
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
func (p *OVHProvider) TokenURL() string {
	return p.tokenURL()
}

// Scopes returns the required OAuth scopes for OVH.
// These scopes provide access to manage cloud resources.
func (p *OVHProvider) Scopes() []string {
	return []string{
		"all", // Full access to manage cloud resources
	}
}

// ClientID returns the OAuth client ID.
func (p *OVHProvider) ClientID() string {
	return p.clientID
}

// ClientSecret returns the OAuth client secret.
func (p *OVHProvider) ClientSecret() string {
	return p.clientSecret
}

// IsEnabled returns whether OVH OAuth is enabled.
func (p *OVHProvider) IsEnabled() bool {
	return p.enabled && p.clientID != ""
}

// authURL returns the authorization URL based on region.
func (p *OVHProvider) authURL() string {
	switch p.region {
	case "ca":
		return ovhAuthURLCA
	case "us":
		return ovhAuthURLUS
	default:
		return ovhAuthURLEU
	}
}

// tokenURL returns the token URL based on region.
func (p *OVHProvider) tokenURL() string {
	switch p.region {
	case "ca":
		return ovhTokenURLCA
	case "us":
		return ovhTokenURLUS
	default:
		return ovhTokenURLEU
	}
}

// OVH OAuth endpoints by region.
const (
	// EU region (Europe)
	ovhAuthURLEU  = "https://www.ovh.com/auth/oauth2/authorize"
	ovhTokenURLEU = "https://www.ovh.com/auth/oauth2/token"

	// CA region (Canada)
	ovhAuthURLCA  = "https://ca.ovh.com/auth/oauth2/authorize"
	ovhTokenURLCA = "https://ca.ovh.com/auth/oauth2/token"

	// US region (United States)
	ovhAuthURLUS  = "https://us.ovhcloud.com/auth/oauth2/authorize"
	ovhTokenURLUS = "https://us.ovhcloud.com/auth/oauth2/token"
)
