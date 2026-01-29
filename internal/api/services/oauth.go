// Package services provides business logic for API resources.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/config"
	"github.com/janovincze/philotes/internal/crypto"
	"github.com/janovincze/philotes/internal/installer/oauth"
)

// OAuthService handles OAuth flows for cloud providers.
type OAuthService struct {
	repo       *repositories.OAuthRepository
	encryptor  *crypto.Encryptor
	config     config.OAuthConfig
	registry   *oauth.ProviderRegistry
	httpClient *http.Client
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	repo *repositories.OAuthRepository,
	cfg config.OAuthConfig,
) (*OAuthService, error) {
	// Check if any OAuth provider is enabled
	anyProviderEnabled := cfg.Hetzner.Enabled || cfg.OVH.Enabled

	// Fail fast: if OAuth is enabled, encryption key must be configured
	if anyProviderEnabled && cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("OAuth encryption key is required when OAuth providers are enabled")
	}

	// Create encryptor if encryption key is configured
	var encryptor *crypto.Encryptor
	if cfg.EncryptionKey != "" {
		var err error
		encryptor, err = crypto.NewEncryptorFromString(cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
	}

	// Build provider registry
	registry := oauth.NewProviderRegistry()

	// Register Hetzner if configured
	if cfg.Hetzner.ClientID != "" {
		registry.Register(oauth.NewHetznerProvider(oauth.HetznerConfig{
			ClientID:     cfg.Hetzner.ClientID,
			ClientSecret: cfg.Hetzner.ClientSecret,
			Enabled:      cfg.Hetzner.Enabled,
		}))
	}

	// Register OVH if configured
	if cfg.OVH.ClientID != "" {
		registry.Register(oauth.NewOVHProvider(oauth.OVHConfig{
			ClientID:     cfg.OVH.ClientID,
			ClientSecret: cfg.OVH.ClientSecret,
			Enabled:      cfg.OVH.Enabled,
		}))
	}

	return &OAuthService{
		repo:      repo,
		encryptor: encryptor,
		config:    cfg,
		registry:  registry,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// validateRedirectURI validates that the redirect URI is allowed to prevent open redirect attacks.
func (s *OAuthService) validateRedirectURI(redirectURI string) error {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	// Must be an absolute URL with http or https scheme
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("redirect URI must use http or https scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("redirect URI must have a host")
	}

	// Build list of allowed hosts
	allowedHosts := make(map[string]bool)

	// Add explicitly configured allowed hosts
	for _, host := range s.config.AllowedRedirectHosts {
		allowedHosts[host] = true
	}

	// Add host from BaseURL if configured
	if s.config.BaseURL != "" {
		if baseURL, err := url.Parse(s.config.BaseURL); err == nil && baseURL.Host != "" {
			allowedHosts[baseURL.Host] = true
		}
	}

	// Always allow localhost for development
	allowedHosts["localhost"] = true
	allowedHosts["localhost:3000"] = true
	allowedHosts["127.0.0.1"] = true
	allowedHosts["127.0.0.1:3000"] = true

	// Check if the redirect host is allowed
	if !allowedHosts[parsed.Host] {
		return fmt.Errorf("host %q is not in the allowed redirect hosts", parsed.Host)
	}

	return nil
}

// StartAuthorization initiates the OAuth flow for a provider.
func (s *OAuthService) StartAuthorization(
	ctx context.Context,
	providerID string,
	redirectURI string,
	userID *uuid.UUID,
	sessionID string,
) (*models.OAuthAuthorizeResponse, error) {
	// Validate redirect URI to prevent open redirect attacks
	if err := s.validateRedirectURI(redirectURI); err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Get provider
	provider, ok := s.registry.Get(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not configured for OAuth", providerID)
	}

	if !provider.IsEnabled() {
		return nil, fmt.Errorf("OAuth is not enabled for provider %s", providerID)
	}

	// Generate PKCE values
	state, err := oauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := oauth.GenerateCodeChallenge(codeVerifier)

	// Build callback URL
	callbackURL := fmt.Sprintf("%s/api/v1/installer/oauth/%s/callback", s.config.BaseURL, providerID)

	// Store state
	oauthState := &models.OAuthState{
		ID:           uuid.New(),
		Provider:     providerID,
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		UserID:       userID,
		SessionID:    sessionID,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateState(ctx, oauthState); err != nil {
		return nil, fmt.Errorf("failed to store oauth state: %w", err)
	}

	// Build authorization URL
	authURL := provider.AuthorizationURL(state, codeChallenge, callbackURL)

	return &models.OAuthAuthorizeResponse{
		AuthorizationURL: authURL,
		State:            state,
		Provider:         providerID,
	}, nil
}

// HandleCallback processes the OAuth callback and exchanges code for tokens.
func (s *OAuthService) HandleCallback(
	ctx context.Context,
	providerID string,
	code string,
	state string,
) (*models.OAuthCallbackResponse, error) {
	// Retrieve and validate state
	oauthState, err := s.repo.GetStateByState(ctx, state)
	if err != nil {
		return &models.OAuthCallbackResponse{
			Success: false,
			Error:   "invalid or expired state",
		}, nil
	}

	// Verify provider matches
	if oauthState.Provider != providerID {
		return &models.OAuthCallbackResponse{
			Success: false,
			Error:   "provider mismatch",
		}, nil
	}

	// Get provider
	provider, ok := s.registry.Get(providerID)
	if !ok {
		return &models.OAuthCallbackResponse{
			Success: false,
			Error:   "provider not configured",
		}, nil
	}

	// Build callback URL (same as in StartAuthorization)
	callbackURL := fmt.Sprintf("%s/api/v1/installer/oauth/%s/callback", s.config.BaseURL, providerID)

	// Exchange code for tokens BEFORE deleting state
	// If this fails, the state is still available for retry (until it expires)
	token, err := s.exchangeCode(ctx, provider, code, oauthState.CodeVerifier, callbackURL)
	if err != nil {
		return &models.OAuthCallbackResponse{
			Success:     false,
			Error:       fmt.Sprintf("token exchange failed: %v", err),
			RedirectURI: oauthState.RedirectURI,
		}, nil
	}

	// Use a transaction to atomically delete state and store credential
	// This ensures we don't lose the credential if state deletion succeeds but storage fails
	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return &models.OAuthCallbackResponse{
			Success:     false,
			Error:       "failed to start transaction",
			RedirectURI: oauthState.RedirectURI,
		}, nil
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Delete state within transaction
	if _, err = tx.ExecContext(ctx, `DELETE FROM philotes.oauth_states WHERE state = $1`, state); err != nil {
		return &models.OAuthCallbackResponse{
			Success:     false,
			Error:       "failed to delete oauth state",
			RedirectURI: oauthState.RedirectURI,
		}, nil
	}

	// Store credential within transaction
	credID, err := s.storeOAuthCredentialTx(ctx, tx, providerID, token, oauthState.UserID)
	if err != nil {
		return &models.OAuthCallbackResponse{
			Success:     false,
			Error:       fmt.Sprintf("failed to store credential: %v", err),
			RedirectURI: oauthState.RedirectURI,
		}, nil
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return &models.OAuthCallbackResponse{
			Success:     false,
			Error:       "failed to commit transaction",
			RedirectURI: oauthState.RedirectURI,
		}, nil
	}

	return &models.OAuthCallbackResponse{
		Success:      true,
		Provider:     providerID,
		CredentialID: credID,
		RedirectURI:  oauthState.RedirectURI,
	}, nil
}

// exchangeCode exchanges an authorization code for tokens.
func (s *OAuthService) exchangeCode(
	ctx context.Context,
	provider oauth.Provider,
	code string,
	codeVerifier string,
	redirectURI string,
) (*models.OAuthToken, error) {
	// Build token request
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
		"client_id":     {provider.ClientID()},
	}

	// Add client secret if available (confidential client)
	if provider.ClientSecret() != "" {
		data.Set("client_secret", provider.ClientSecret())
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.TokenURL(),
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
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

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token := &models.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		RefreshToken: tokenResp.RefreshToken,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}

// storeOAuthCredentialTx encrypts and stores OAuth tokens within a transaction.
func (s *OAuthService) storeOAuthCredentialTx(
	ctx context.Context,
	tx *sql.Tx,
	providerID string,
	token *models.OAuthToken,
	userID *uuid.UUID,
) (uuid.UUID, error) {
	if s.encryptor == nil {
		return uuid.Nil, fmt.Errorf("encryption not configured")
	}

	// Encrypt access token
	credentialsEncrypted, err := s.encryptor.EncryptToBytes(token.AccessToken)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Encrypt refresh token if present
	var refreshTokenEncrypted []byte
	if token.RefreshToken != "" {
		refreshTokenEncrypted, err = s.encryptor.EncryptToBytes(token.RefreshToken)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	// Set expiration (credential valid for 30 days)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	var tokenExpiresAt *time.Time
	if !token.ExpiresAt.IsZero() {
		tokenExpiresAt = &token.ExpiresAt
	}

	credID := uuid.New()

	query := `
		INSERT INTO philotes.cloud_credentials (
			id, deployment_id, user_id, provider, credential_type,
			credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.ExecContext(ctx, query,
		credID,
		nil, // deployment_id
		userID,
		providerID,
		models.CredentialTypeOAuth,
		credentialsEncrypted,
		refreshTokenEncrypted,
		tokenExpiresAt,
		expiresAt,
		time.Now(),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return credID, nil
}

// StoreManualCredential stores manually entered API credentials.
func (s *OAuthService) StoreManualCredential(
	ctx context.Context,
	req *models.StoreCredentialRequest,
	userID *uuid.UUID,
) (*models.StoreCredentialResponse, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("encryption not configured")
	}

	// Serialize credentials to JSON
	credJSON, err := json.Marshal(req.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize credentials: %w", err)
	}

	// Encrypt credentials
	credentialsEncrypted, err := s.encryptor.Encrypt(credJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Set expiration (default 24 hours)
	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 24 * 60 * 60 // 24 hours
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	cred := &models.CloudCredential{
		ID:                   uuid.New(),
		DeploymentID:         req.DeploymentID,
		UserID:               userID,
		Provider:             req.Provider,
		CredentialType:       models.CredentialTypeManual,
		CredentialsEncrypted: credentialsEncrypted,
		ExpiresAt:            expiresAt,
		CreatedAt:            time.Now(),
	}

	if err := s.repo.CreateCredential(ctx, cred); err != nil {
		return nil, fmt.Errorf("failed to store credential: %w", err)
	}

	return &models.StoreCredentialResponse{
		CredentialID: cred.ID,
		Provider:     cred.Provider,
		ExpiresAt:    cred.ExpiresAt,
	}, nil
}

// GetCredential retrieves and decrypts a credential.
func (s *OAuthService) GetCredential(
	ctx context.Context,
	credID uuid.UUID,
) (*models.ProviderCredentials, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("encryption not configured")
	}

	cred, err := s.repo.GetCredentialByID(ctx, credID)
	if err != nil {
		return nil, err
	}

	return s.decryptCredential(cred)
}

// GetCredentialByProvider retrieves a credential by provider for a user.
func (s *OAuthService) GetCredentialByProvider(
	ctx context.Context,
	userID uuid.UUID,
	provider string,
) (*models.ProviderCredentials, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("encryption not configured")
	}

	cred, err := s.repo.GetCredentialByProvider(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	return s.decryptCredential(cred)
}

// decryptCredential decrypts a stored credential.
func (s *OAuthService) decryptCredential(cred *models.CloudCredential) (*models.ProviderCredentials, error) {
	decrypted, err := s.encryptor.Decrypt(cred.CredentialsEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// For OAuth credentials, the decrypted value is the access token
	if cred.CredentialType == models.CredentialTypeOAuth {
		return s.buildOAuthCredentials(cred.Provider, string(decrypted))
	}

	// For manual credentials, parse JSON
	var credentials models.ProviderCredentials
	if err := json.Unmarshal(decrypted, &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &credentials, nil
}

// buildOAuthCredentials builds provider credentials from an OAuth token.
func (s *OAuthService) buildOAuthCredentials(provider, accessToken string) (*models.ProviderCredentials, error) {
	switch provider {
	case "hetzner":
		return &models.ProviderCredentials{
			HetznerToken: accessToken,
		}, nil
	case "ovh":
		// OVH OAuth returns an access token that can be used directly
		// Note: For full API access, OVH may still need application key/secret
		return &models.ProviderCredentials{
			OVHConsumerKey: accessToken,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

// ListCredentials lists all credentials for a user.
func (s *OAuthService) ListCredentials(
	ctx context.Context,
	userID uuid.UUID,
) (*models.CredentialListResponse, error) {
	credentials, err := s.repo.ListCredentialsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.CredentialSummary, len(credentials))
	for i, cred := range credentials {
		summaries[i] = models.CredentialSummary{
			ID:             cred.ID,
			Provider:       cred.Provider,
			CredentialType: cred.CredentialType,
			TokenExpiresAt: cred.TokenExpiresAt,
			ExpiresAt:      cred.ExpiresAt,
			CreatedAt:      cred.CreatedAt,
		}
	}

	return &models.CredentialListResponse{
		Credentials: summaries,
		TotalCount:  len(summaries),
	}, nil
}

// DeleteCredential deletes a credential.
func (s *OAuthService) DeleteCredential(
	ctx context.Context,
	credID uuid.UUID,
) error {
	return s.repo.DeleteCredential(ctx, credID)
}

// DeleteCredentialByProvider deletes credentials for a provider.
func (s *OAuthService) DeleteCredentialByProvider(
	ctx context.Context,
	userID uuid.UUID,
	provider string,
) error {
	return s.repo.DeleteCredentialByProvider(ctx, userID, provider)
}

// GetOAuthProviders returns information about available OAuth providers.
func (s *OAuthService) GetOAuthProviders() *models.OAuthProvidersResponse {
	providers := s.registry.List()
	infos := make([]models.OAuthProviderInfo, 0, len(providers))

	for _, p := range providers {
		infos = append(infos, models.OAuthProviderInfo{
			Provider:    p.Name(),
			Name:        p.DisplayName(),
			Scopes:      p.Scopes(),
			Enabled:     p.IsEnabled(),
			Description: fmt.Sprintf("Connect with %s using OAuth", p.DisplayName()),
		})
	}

	return &models.OAuthProvidersResponse{
		Providers: infos,
	}
}

// RefreshToken refreshes an OAuth token if it's about to expire.
func (s *OAuthService) RefreshToken(
	ctx context.Context,
	credID uuid.UUID,
) error {
	cred, err := s.repo.GetCredentialByID(ctx, credID)
	if err != nil {
		return err
	}

	if cred.CredentialType != models.CredentialTypeOAuth {
		return fmt.Errorf("credential is not OAuth type")
	}

	if cred.RefreshTokenEncrypted == nil {
		return fmt.Errorf("no refresh token available")
	}

	// Decrypt refresh token
	refreshToken, err := s.encryptor.DecryptFromBytes(cred.RefreshTokenEncrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	// Get provider
	provider, ok := s.registry.Get(cred.Provider)
	if !ok {
		return fmt.Errorf("provider %s not configured", cred.Provider)
	}

	// Request new token
	token, err := s.refreshToken(ctx, provider, refreshToken)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// Update stored credential
	newCredEncrypted, err := s.encryptor.EncryptToBytes(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt new token: %w", err)
	}

	cred.CredentialsEncrypted = newCredEncrypted
	if !token.ExpiresAt.IsZero() {
		cred.TokenExpiresAt = &token.ExpiresAt
	}

	// Update refresh token if a new one was provided
	if token.RefreshToken != "" {
		newRefreshEncrypted, err := s.encryptor.EncryptToBytes(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to encrypt new refresh token: %w", err)
		}
		cred.RefreshTokenEncrypted = newRefreshEncrypted
	}

	return s.repo.UpdateCredential(ctx, cred)
}

// refreshToken performs the OAuth token refresh.
func (s *OAuthService) refreshToken(
	ctx context.Context,
	provider oauth.Provider,
	refreshToken string,
) (*models.OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {provider.ClientID()},
	}

	if provider.ClientSecret() != "" {
		data.Set("client_secret", provider.ClientSecret())
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		provider.TokenURL(),
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	token := &models.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		RefreshToken: tokenResp.RefreshToken,
		Scope:        tokenResp.Scope,
	}

	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}

// CleanupExpired removes expired OAuth states and credentials.
func (s *OAuthService) CleanupExpired(ctx context.Context) (states int64, creds int64, err error) {
	states, err = s.repo.CleanupExpiredStates(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup states: %w", err)
	}

	creds, err = s.repo.CleanupExpiredCredentials(ctx)
	if err != nil {
		return states, 0, fmt.Errorf("failed to cleanup credentials: %w", err)
	}

	return states, creds, nil
}
