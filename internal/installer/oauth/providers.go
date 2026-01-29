// Package oauth provides OAuth 2.0 configuration for cloud providers.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
)

// Provider represents an OAuth provider configuration.
type Provider interface {
	// Name returns the provider identifier (e.g., "hetzner", "ovh").
	Name() string

	// DisplayName returns the human-readable provider name.
	DisplayName() string

	// AuthorizationURL builds the OAuth authorization URL with PKCE.
	AuthorizationURL(state, codeChallenge, redirectURI string) string

	// TokenURL returns the token endpoint URL.
	TokenURL() string

	// Scopes returns the required OAuth scopes.
	Scopes() []string

	// ClientID returns the OAuth client ID.
	ClientID() string

	// ClientSecret returns the OAuth client secret (empty for public clients).
	ClientSecret() string

	// IsEnabled returns whether this provider's OAuth is enabled.
	IsEnabled() bool
}

// ProviderRegistry holds all registered OAuth providers.
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *ProviderRegistry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered providers.
func (r *ProviderRegistry) List() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListEnabled returns all enabled OAuth providers.
func (r *ProviderRegistry) ListEnabled() []Provider {
	providers := make([]Provider, 0)
	for _, p := range r.providers {
		if p.IsEnabled() {
			providers = append(providers, p)
		}
	}
	return providers
}

// --- PKCE Utilities ---

// GenerateState generates a cryptographically secure random state parameter.
func GenerateState() (string, error) {
	return generateRandomString(32)
}

// GenerateCodeVerifier generates a PKCE code verifier.
// Returns a 43-128 character URL-safe string.
func GenerateCodeVerifier() (string, error) {
	return generateRandomString(64)
}

// GenerateCodeChallenge generates a PKCE code challenge from a verifier.
// Uses the S256 method (SHA-256 hash, base64url encoded).
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// BuildAuthURL is a helper to construct OAuth authorization URLs.
func BuildAuthURL(baseURL string, params map[string]string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// SupportedProviders returns the list of provider IDs that support OAuth.
func SupportedProviders() []string {
	return []string{"hetzner", "ovh"}
}

// IsOAuthSupported returns whether OAuth is supported for the given provider.
func IsOAuthSupported(providerID string) bool {
	for _, p := range SupportedProviders() {
		if p == providerID {
			return true
		}
	}
	return false
}
