// Package oidc provides OIDC authentication functionality.
package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/installer/oauth"
)

// Client wraps OIDC operations for a provider.
type Client struct {
	httpClient *http.Client
	issuerURL  string
	clientID   string
	scopes     []string
	config     *DiscoveryConfig
}

// DiscoveryConfig holds OIDC discovery document configuration.
type DiscoveryConfig struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
}

// TokenResponse represents the token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope,omitempty"`
}

// IDTokenClaims represents standard OIDC ID token claims.
type IDTokenClaims struct {
	Issuer        string   `json:"iss"`
	Subject       string   `json:"sub"`
	Audience      Audience `json:"aud"`
	ExpiresAt     int64    `json:"exp"`
	IssuedAt      int64    `json:"iat"`
	Nonce         string   `json:"nonce,omitempty"`
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	Name          string   `json:"name,omitempty"`
	GivenName     string   `json:"given_name,omitempty"`
	FamilyName    string   `json:"family_name,omitempty"`
	Picture       string   `json:"picture,omitempty"`
	Groups        []string `json:"groups,omitempty"`
}

// Audience can be a string or array of strings.
type Audience []string

// UnmarshalJSON handles both string and []string for audience.
func (a *Audience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = []string{single}
		return nil
	}
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return err
	}
	*a = multiple
	return nil
}

// NewClient creates a new OIDC client.
func NewClient(issuerURL, clientID string, scopes []string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		issuerURL:  strings.TrimSuffix(issuerURL, "/"),
		clientID:   clientID,
		scopes:     scopes,
	}
}

// Discover fetches the OIDC discovery document.
func (c *Client) Discover(ctx context.Context) (*DiscoveryConfig, error) {
	if c.config != nil {
		return c.config, nil
	}

	discoveryURL := c.issuerURL + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, fmt.Errorf("discovery endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var config DiscoveryConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode discovery document: %w", err)
	}

	c.config = &config
	return &config, nil
}

// AuthorizationURL builds the authorization URL with PKCE.
func (c *Client) AuthorizationURL(ctx context.Context, state, nonce, codeChallenge, redirectURI string) (string, error) {
	config, err := c.Discover(ctx)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"client_id":             {c.clientID},
		"response_type":         {"code"},
		"scope":                 {strings.Join(c.scopes, " ")},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	return config.AuthorizationEndpoint + "?" + params.Encode(), nil
}

// Exchange exchanges an authorization code for tokens.
func (c *Client) Exchange(ctx context.Context, code, codeVerifier, redirectURI, clientSecret string) (*TokenResponse, error) {
	config, err := c.Discover(ctx)
	if err != nil {
		return nil, err
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.clientID},
		"code_verifier": {codeVerifier},
	}

	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// ParseIDToken parses and validates an ID token (without signature verification for now).
// In production, you should verify the signature using JWKS.
func (c *Client) ParseIDToken(idToken, nonce string) (*IDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid ID token format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ID token payload: %w", err)
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	// Validate issuer
	if claims.Issuer != c.issuerURL {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", c.issuerURL, claims.Issuer)
	}

	// Validate audience
	validAudience := false
	for _, aud := range claims.Audience {
		if aud == c.clientID {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, fmt.Errorf("invalid audience: %v does not contain %s", claims.Audience, c.clientID)
	}

	// Validate expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("ID token has expired")
	}

	// Validate nonce
	if nonce != "" && claims.Nonce != nonce {
		return nil, fmt.Errorf("invalid nonce")
	}

	return &claims, nil
}

// GetUserInfo fetches user info from the userinfo endpoint.
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*models.OIDCUserInfo, error) {
	config, err := c.Discover(ctx)
	if err != nil {
		return nil, err
	}

	if config.UserInfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint not available")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.UserInfoEndpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read for error message
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var userInfo models.OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &userInfo, nil
}

// ClaimsToUserInfo converts ID token claims to OIDCUserInfo.
func ClaimsToUserInfo(claims *IDTokenClaims, groupsClaim string) *models.OIDCUserInfo {
	return &models.OIDCUserInfo{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		GivenName:     claims.GivenName,
		FamilyName:    claims.FamilyName,
		Picture:       claims.Picture,
		Groups:        claims.Groups,
	}
}

// GenerateState generates a cryptographically secure state parameter.
func GenerateState() (string, error) {
	return oauth.GenerateState()
}

// GenerateNonce generates a cryptographically secure nonce.
func GenerateNonce() (string, error) {
	return oauth.GenerateState() // Same format as state
}

// GenerateCodeVerifier generates a PKCE code verifier.
func GenerateCodeVerifier() (string, error) {
	return oauth.GenerateCodeVerifier()
}

// GenerateCodeChallenge generates a PKCE code challenge from a verifier.
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
