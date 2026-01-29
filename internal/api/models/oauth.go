// Package models provides API request and response types.
package models

import (
	"time"

	"github.com/google/uuid"
)

// CredentialType represents the type of credential (oauth or manual).
type CredentialType string

const (
	// CredentialTypeOAuth indicates OAuth-based credentials.
	CredentialTypeOAuth CredentialType = "oauth"
	// CredentialTypeManual indicates manually entered API credentials.
	CredentialTypeManual CredentialType = "manual"
)

// OAuthState represents a temporary OAuth state for PKCE flow.
type OAuthState struct {
	ID           uuid.UUID  `json:"id"`
	Provider     string     `json:"provider"`
	State        string     `json:"state"`
	CodeVerifier string     `json:"-"` // Never expose code verifier
	RedirectURI  string     `json:"redirect_uri"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	SessionID    string     `json:"session_id,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CloudCredential represents stored cloud provider credentials.
type CloudCredential struct {
	ID                    uuid.UUID      `json:"id"`
	DeploymentID          *uuid.UUID     `json:"deployment_id,omitempty"`
	UserID                *uuid.UUID     `json:"user_id,omitempty"`
	Provider              string         `json:"provider"`
	CredentialType        CredentialType `json:"credential_type"`
	CredentialsEncrypted  []byte         `json:"-"` // Never expose encrypted data
	RefreshTokenEncrypted []byte         `json:"-"`
	TokenExpiresAt        *time.Time     `json:"token_expires_at,omitempty"`
	ExpiresAt             time.Time      `json:"expires_at"`
	CreatedAt             time.Time      `json:"created_at"`
}

// OAuthToken represents the token response from OAuth providers.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// --- Request Types ---

// OAuthAuthorizeRequest represents a request to start OAuth flow.
type OAuthAuthorizeRequest struct {
	RedirectURI string `json:"redirect_uri" binding:"required"`
	SessionID   string `json:"session_id,omitempty"`
}

// StoreCredentialRequest represents a request to store manual credentials.
type StoreCredentialRequest struct {
	Provider     string               `json:"provider" binding:"required"`
	Credentials  *ProviderCredentials `json:"credentials" binding:"required"`
	DeploymentID *uuid.UUID           `json:"deployment_id,omitempty"`
	ExpiresIn    int                  `json:"expires_in,omitempty"` // Seconds until expiration, default 24h
}

// --- Response Types ---

// OAuthAuthorizeResponse contains the authorization URL for OAuth.
type OAuthAuthorizeResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
	Provider         string `json:"provider"`
}

// OAuthCallbackResponse contains the result of OAuth callback.
type OAuthCallbackResponse struct {
	Success      bool      `json:"success"`
	Provider     string    `json:"provider"`
	CredentialID uuid.UUID `json:"credential_id,omitempty"`
	Error        string    `json:"error,omitempty"`
	RedirectURI  string    `json:"redirect_uri,omitempty"`
}

// CredentialSummary represents a stored credential without sensitive data.
type CredentialSummary struct {
	ID             uuid.UUID      `json:"id"`
	Provider       string         `json:"provider"`
	CredentialType CredentialType `json:"credential_type"`
	TokenExpiresAt *time.Time     `json:"token_expires_at,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// CredentialListResponse contains a list of stored credentials.
type CredentialListResponse struct {
	Credentials []CredentialSummary `json:"credentials"`
	TotalCount  int                 `json:"total_count"`
}

// StoreCredentialResponse contains the result of storing credentials.
type StoreCredentialResponse struct {
	CredentialID uuid.UUID `json:"credential_id"`
	Provider     string    `json:"provider"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// OAuthProviderInfo contains OAuth configuration for a provider.
type OAuthProviderInfo struct {
	Provider    string   `json:"provider"`
	Name        string   `json:"name"`
	OAuthURL    string   `json:"oauth_url"`
	Scopes      []string `json:"scopes"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description,omitempty"`
}

// OAuthProvidersResponse contains OAuth info for all providers.
type OAuthProvidersResponse struct {
	Providers []OAuthProviderInfo `json:"providers"`
}

// --- Validation ---

// Validate validates the OAuth authorize request.
func (r *OAuthAuthorizeRequest) Validate() []FieldError {
	var errors []FieldError
	if r.RedirectURI == "" {
		errors = append(errors, FieldError{Field: "redirect_uri", Message: "redirect_uri is required"})
	}
	return errors
}

// Validate validates the store credential request.
func (r *StoreCredentialRequest) Validate() []FieldError {
	var errors []FieldError

	validProviders := map[string]bool{
		"hetzner": true, "scaleway": true, "ovh": true, "exoscale": true, "contabo": true,
	}
	if !validProviders[r.Provider] {
		errors = append(errors, FieldError{Field: "provider", Message: "invalid provider"})
	}

	if r.Credentials == nil {
		errors = append(errors, FieldError{Field: "credentials", Message: "credentials are required"})
	}

	return errors
}
