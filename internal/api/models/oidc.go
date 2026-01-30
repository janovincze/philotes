// Package models provides API request and response types.
package models

import (
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OIDCProviderType represents the type of OIDC provider.
type OIDCProviderType string

const (
	// OIDCProviderTypeGoogle represents Google as OIDC provider.
	OIDCProviderTypeGoogle OIDCProviderType = "google"
	// OIDCProviderTypeOkta represents Okta as OIDC provider.
	OIDCProviderTypeOkta OIDCProviderType = "okta"
	// OIDCProviderTypeAzureAD represents Azure AD as OIDC provider.
	OIDCProviderTypeAzureAD OIDCProviderType = "azure_ad"
	// OIDCProviderTypeAuth0 represents Auth0 as OIDC provider.
	OIDCProviderTypeAuth0 OIDCProviderType = "auth0"
	// OIDCProviderTypeGeneric represents a generic OIDC provider.
	OIDCProviderTypeGeneric OIDCProviderType = "generic"
)

// ValidOIDCProviderTypes contains all valid OIDC provider types.
var ValidOIDCProviderTypes = map[OIDCProviderType]bool{
	OIDCProviderTypeGoogle:  true,
	OIDCProviderTypeOkta:    true,
	OIDCProviderTypeAzureAD: true,
	OIDCProviderTypeAuth0:   true,
	OIDCProviderTypeGeneric: true,
}

// OIDCProvider represents an OIDC identity provider configuration.
type OIDCProvider struct {
	ID                    uuid.UUID           `json:"id"`
	Name                  string              `json:"name"`
	DisplayName           string              `json:"display_name"`
	ProviderType          OIDCProviderType    `json:"provider_type"`
	IssuerURL             string              `json:"issuer_url"`
	ClientID              string              `json:"client_id"`
	ClientSecretEncrypted []byte              `json:"-"` // Never expose encrypted secret
	Scopes                []string            `json:"scopes"`
	GroupsClaim           string              `json:"groups_claim"`
	RoleMapping           map[string]UserRole `json:"role_mapping"`
	DefaultRole           UserRole            `json:"default_role"`
	Enabled               bool                `json:"enabled"`
	AutoCreateUsers       bool                `json:"auto_create_users"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

// OIDCState represents a temporary OIDC state for authorization flow.
type OIDCState struct {
	ID           uuid.UUID `json:"id"`
	State        string    `json:"state"`
	Nonce        string    `json:"-"` // Never expose nonce
	CodeVerifier string    `json:"-"` // Never expose code verifier
	ProviderID   uuid.UUID `json:"provider_id"`
	RedirectURI  string    `json:"redirect_uri"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// --- Request Types ---

// CreateOIDCProviderRequest represents a request to create an OIDC provider.
type CreateOIDCProviderRequest struct {
	Name            string              `json:"name" binding:"required"`
	DisplayName     string              `json:"display_name" binding:"required"`
	ProviderType    OIDCProviderType    `json:"provider_type" binding:"required"`
	IssuerURL       string              `json:"issuer_url" binding:"required"`
	ClientID        string              `json:"client_id" binding:"required"`
	ClientSecret    string              `json:"client_secret" binding:"required"`
	Scopes          []string            `json:"scopes,omitempty"`
	GroupsClaim     string              `json:"groups_claim,omitempty"`
	RoleMapping     map[string]UserRole `json:"role_mapping,omitempty"`
	DefaultRole     UserRole            `json:"default_role,omitempty"`
	Enabled         *bool               `json:"enabled,omitempty"`
	AutoCreateUsers *bool               `json:"auto_create_users,omitempty"`
}

// Validate validates the create OIDC provider request.
func (r *CreateOIDCProviderRequest) Validate() []FieldError {
	var errors []FieldError

	// Validate name
	switch {
	case r.Name == "":
		errors = append(errors, FieldError{Field: "name", Message: "name is required"})
	case len(r.Name) > 100:
		errors = append(errors, FieldError{Field: "name", Message: "name must be at most 100 characters"})
	case !isValidProviderName(r.Name):
		errors = append(errors, FieldError{Field: "name", Message: "name must contain only lowercase letters, numbers, and hyphens"})
	}

	// Validate display name
	if r.DisplayName == "" {
		errors = append(errors, FieldError{Field: "display_name", Message: "display_name is required"})
	} else if len(r.DisplayName) > 255 {
		errors = append(errors, FieldError{Field: "display_name", Message: "display_name must be at most 255 characters"})
	}

	// Validate provider type
	if !ValidOIDCProviderTypes[r.ProviderType] {
		errors = append(errors, FieldError{Field: "provider_type", Message: "invalid provider_type"})
	}

	// Validate issuer URL
	if r.IssuerURL == "" {
		errors = append(errors, FieldError{Field: "issuer_url", Message: "issuer_url is required"})
	} else if _, err := url.ParseRequestURI(r.IssuerURL); err != nil {
		errors = append(errors, FieldError{Field: "issuer_url", Message: "issuer_url must be a valid URL"})
	}

	// Validate client ID
	if r.ClientID == "" {
		errors = append(errors, FieldError{Field: "client_id", Message: "client_id is required"})
	}

	// Validate client secret
	if r.ClientSecret == "" {
		errors = append(errors, FieldError{Field: "client_secret", Message: "client_secret is required"})
	}

	// Validate default role if provided
	if r.DefaultRole != "" && r.DefaultRole != RoleAdmin && r.DefaultRole != RoleOperator && r.DefaultRole != RoleViewer {
		errors = append(errors, FieldError{Field: "default_role", Message: "invalid default_role"})
	}

	// Validate role mapping values
	for group, role := range r.RoleMapping {
		if role != RoleAdmin && role != RoleOperator && role != RoleViewer {
			errors = append(errors, FieldError{Field: "role_mapping", Message: "invalid role for group: " + group})
		}
	}

	return errors
}

// ApplyDefaults applies default values to the request.
func (r *CreateOIDCProviderRequest) ApplyDefaults() {
	if len(r.Scopes) == 0 {
		r.Scopes = []string{"openid", "profile", "email"}
	}
	if r.GroupsClaim == "" {
		r.GroupsClaim = "groups"
	}
	if r.RoleMapping == nil {
		r.RoleMapping = make(map[string]UserRole)
	}
	if r.DefaultRole == "" {
		r.DefaultRole = RoleViewer
	}
	if r.Enabled == nil {
		enabled := true
		r.Enabled = &enabled
	}
	if r.AutoCreateUsers == nil {
		autoCreate := true
		r.AutoCreateUsers = &autoCreate
	}
}

// UpdateOIDCProviderRequest represents a request to update an OIDC provider.
type UpdateOIDCProviderRequest struct {
	DisplayName     *string             `json:"display_name,omitempty"`
	IssuerURL       *string             `json:"issuer_url,omitempty"`
	ClientID        *string             `json:"client_id,omitempty"`
	ClientSecret    *string             `json:"client_secret,omitempty"`
	Scopes          []string            `json:"scopes,omitempty"`
	GroupsClaim     *string             `json:"groups_claim,omitempty"`
	RoleMapping     map[string]UserRole `json:"role_mapping,omitempty"`
	DefaultRole     *UserRole           `json:"default_role,omitempty"`
	Enabled         *bool               `json:"enabled,omitempty"`
	AutoCreateUsers *bool               `json:"auto_create_users,omitempty"`
}

// Validate validates the update OIDC provider request.
func (r *UpdateOIDCProviderRequest) Validate() []FieldError {
	var errors []FieldError

	// Validate display name if provided
	if r.DisplayName != nil && len(*r.DisplayName) > 255 {
		errors = append(errors, FieldError{Field: "display_name", Message: "display_name must be at most 255 characters"})
	}

	// Validate issuer URL if provided
	if r.IssuerURL != nil {
		if _, err := url.ParseRequestURI(*r.IssuerURL); err != nil {
			errors = append(errors, FieldError{Field: "issuer_url", Message: "issuer_url must be a valid URL"})
		}
	}

	// Validate default role if provided
	if r.DefaultRole != nil && *r.DefaultRole != RoleAdmin && *r.DefaultRole != RoleOperator && *r.DefaultRole != RoleViewer {
		errors = append(errors, FieldError{Field: "default_role", Message: "invalid default_role"})
	}

	// Validate role mapping values
	for group, role := range r.RoleMapping {
		if role != RoleAdmin && role != RoleOperator && role != RoleViewer {
			errors = append(errors, FieldError{Field: "role_mapping", Message: "invalid role for group: " + group})
		}
	}

	return errors
}

// OIDCAuthorizeRequest represents a request to start OIDC authorization flow.
type OIDCAuthorizeRequest struct {
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

// Validate validates the OIDC authorize request.
func (r *OIDCAuthorizeRequest) Validate() []FieldError {
	var errors []FieldError
	if r.RedirectURI == "" {
		errors = append(errors, FieldError{Field: "redirect_uri", Message: "redirect_uri is required"})
	} else if _, err := url.ParseRequestURI(r.RedirectURI); err != nil {
		errors = append(errors, FieldError{Field: "redirect_uri", Message: "redirect_uri must be a valid URL"})
	}
	return errors
}

// OIDCCallbackRequest represents the callback from an OIDC provider.
type OIDCCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
	Error string `form:"error,omitempty"`
}

// Validate validates the OIDC callback request.
func (r *OIDCCallbackRequest) Validate() []FieldError {
	var errors []FieldError
	if r.Error != "" {
		errors = append(errors, FieldError{Field: "error", Message: r.Error})
		return errors
	}
	if r.Code == "" {
		errors = append(errors, FieldError{Field: "code", Message: "code is required"})
	}
	if r.State == "" {
		errors = append(errors, FieldError{Field: "state", Message: "state is required"})
	}
	return errors
}

// --- Response Types ---

// OIDCProviderResponse wraps an OIDC provider for API responses.
type OIDCProviderResponse struct {
	Provider *OIDCProviderSummary `json:"provider"`
}

// OIDCProviderSummary represents an OIDC provider without sensitive data.
type OIDCProviderSummary struct {
	ID              uuid.UUID           `json:"id"`
	Name            string              `json:"name"`
	DisplayName     string              `json:"display_name"`
	ProviderType    OIDCProviderType    `json:"provider_type"`
	IssuerURL       string              `json:"issuer_url"`
	ClientID        string              `json:"client_id"`
	Scopes          []string            `json:"scopes"`
	GroupsClaim     string              `json:"groups_claim"`
	RoleMapping     map[string]UserRole `json:"role_mapping"`
	DefaultRole     UserRole            `json:"default_role"`
	Enabled         bool                `json:"enabled"`
	AutoCreateUsers bool                `json:"auto_create_users"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// ToSummary converts an OIDCProvider to OIDCProviderSummary.
func (p *OIDCProvider) ToSummary() *OIDCProviderSummary {
	return &OIDCProviderSummary{
		ID:              p.ID,
		Name:            p.Name,
		DisplayName:     p.DisplayName,
		ProviderType:    p.ProviderType,
		IssuerURL:       p.IssuerURL,
		ClientID:        p.ClientID,
		Scopes:          p.Scopes,
		GroupsClaim:     p.GroupsClaim,
		RoleMapping:     p.RoleMapping,
		DefaultRole:     p.DefaultRole,
		Enabled:         p.Enabled,
		AutoCreateUsers: p.AutoCreateUsers,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// OIDCProvidersResponse contains a list of OIDC providers.
type OIDCProvidersResponse struct {
	Providers  []OIDCProviderSummary `json:"providers"`
	TotalCount int                   `json:"total_count"`
}

// OIDCAuthorizeResponse contains the authorization URL for OIDC flow.
type OIDCAuthorizeResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
	Provider         string `json:"provider"`
}

// OIDCCallbackResponse contains the result of OIDC callback.
type OIDCCallbackResponse struct {
	Success     bool      `json:"success"`
	Token       string    `json:"token,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	User        *User     `json:"user,omitempty"`
	RedirectURI string    `json:"redirect_uri,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// OIDCUserInfo represents user information from OIDC claims.
type OIDCUserInfo struct {
	Subject       string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	GivenName     string   `json:"given_name"`
	FamilyName    string   `json:"family_name"`
	Picture       string   `json:"picture"`
	Groups        []string `json:"groups"`
}

// --- Helper Functions ---

// isValidProviderName checks if the provider name contains only valid characters.
func isValidProviderName(name string) bool {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	// Must not start or end with hyphen
	return !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-")
}
