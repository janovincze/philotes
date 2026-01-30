// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/config"
	"github.com/janovincze/philotes/internal/oidc"
	"github.com/janovincze/philotes/internal/oidc/providers"
)

// OIDC service errors.
var (
	ErrOIDCProviderNotFound   = errors.New("oidc provider not found")
	ErrOIDCProviderDisabled   = errors.New("oidc provider is disabled")
	ErrOIDCStateInvalid       = errors.New("invalid or expired state")
	ErrOIDCCallbackFailed     = errors.New("oidc callback failed")
	ErrOIDCUserCreationFailed = errors.New("failed to create oidc user")
	ErrOIDCDiscoveryFailed    = errors.New("oidc discovery failed")
)

// OIDCService provides OIDC authentication business logic.
type OIDCService struct {
	oidcRepo         *repositories.OIDCRepository
	userRepo         *repositories.UserRepository
	auditRepo        *repositories.AuditRepository
	providerRegistry *providers.Registry
	oidcCfg          *config.OIDCConfig
	authCfg          *config.AuthConfig
	baseURL          string
	logger           *slog.Logger
}

// NewOIDCService creates a new OIDCService.
func NewOIDCService(
	oidcRepo *repositories.OIDCRepository,
	userRepo *repositories.UserRepository,
	auditRepo *repositories.AuditRepository,
	oidcCfg *config.OIDCConfig,
	authCfg *config.AuthConfig,
	baseURL string,
	logger *slog.Logger,
) *OIDCService {
	return &OIDCService{
		oidcRepo:         oidcRepo,
		userRepo:         userRepo,
		auditRepo:        auditRepo,
		providerRegistry: providers.NewRegistry(),
		oidcCfg:          oidcCfg,
		authCfg:          authCfg,
		baseURL:          baseURL,
		logger:           logger.With("component", "oidc-service"),
	}
}

// --- Public Endpoints ---

// ListEnabledProviders returns all enabled OIDC providers.
func (s *OIDCService) ListEnabledProviders(ctx context.Context) (*models.OIDCProvidersResponse, error) {
	providerList, err := s.oidcRepo.ListEnabledProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled providers: %w", err)
	}

	summaries := make([]models.OIDCProviderSummary, len(providerList))
	for i := range providerList {
		summaries[i] = *providerList[i].ToSummary()
	}

	return &models.OIDCProvidersResponse{
		Providers:  summaries,
		TotalCount: len(summaries),
	}, nil
}

// StartAuthorization initiates the OIDC authorization flow.
func (s *OIDCService) StartAuthorization(ctx context.Context, providerName, redirectURI string) (*models.OIDCAuthorizeResponse, error) {
	// Get provider
	provider, err := s.oidcRepo.GetProviderByName(ctx, providerName)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNotFound) {
			return nil, ErrOIDCProviderNotFound
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	if !provider.Enabled {
		return nil, ErrOIDCProviderDisabled
	}

	// Validate redirect URI
	if err := s.validateRedirectURI(redirectURI); err != nil {
		return nil, fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Generate PKCE values
	state, err := oidc.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := oidc.GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	codeVerifier, err := oidc.GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := oidc.GenerateCodeChallenge(codeVerifier)

	// Store state in database
	oidcState := &models.OIDCState{
		ID:           uuid.New(),
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ProviderID:   provider.ID,
		RedirectURI:  redirectURI,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(s.oidcCfg.StateExpiration),
	}

	if err := s.oidcRepo.CreateState(ctx, oidcState); err != nil {
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	// Build callback URL
	callbackURL := fmt.Sprintf("%s/api/v1/auth/oidc/callback", s.baseURL)

	// Create OIDC client and get authorization URL
	client := oidc.NewClient(provider.IssuerURL, provider.ClientID, provider.Scopes)
	authURL, err := client.AuthorizationURL(ctx, state, nonce, codeChallenge, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build authorization URL: %w", err)
	}

	s.logger.Info("starting OIDC authorization",
		"provider", providerName,
		"state", state[:8]+"...",
	)

	return &models.OIDCAuthorizeResponse{
		AuthorizationURL: authURL,
		State:            state,
		Provider:         providerName,
	}, nil
}

// HandleCallback processes the OIDC callback and returns a JWT token.
func (s *OIDCService) HandleCallback(ctx context.Context, code, state, ipAddress, userAgent string) (*models.OIDCCallbackResponse, error) {
	// Retrieve and validate state
	oidcState, err := s.oidcRepo.GetState(ctx, state)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCStateNotFound) || errors.Is(err, repositories.ErrOIDCStateExpired) {
			return &models.OIDCCallbackResponse{
				Success: false,
				Error:   "invalid or expired state",
			}, nil
		}
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	// Delete state immediately (one-time use)
	if err := s.oidcRepo.DeleteState(ctx, state); err != nil {
		s.logger.Warn("failed to delete state", "error", err)
	}

	// Get provider
	provider, err := s.oidcRepo.GetProviderByID(ctx, oidcState.ProviderID)
	if err != nil {
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "provider not found",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Get client secret
	clientSecret, err := s.oidcRepo.GetProviderClientSecret(ctx, provider.ID)
	if err != nil {
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "failed to get client secret",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Build callback URL (same as in StartAuthorization)
	callbackURL := fmt.Sprintf("%s/api/v1/auth/oidc/callback", s.baseURL)

	// Exchange code for tokens
	client := oidc.NewClient(provider.IssuerURL, provider.ClientID, provider.Scopes)
	tokenResp, err := client.Exchange(ctx, code, oidcState.CodeVerifier, callbackURL, clientSecret)
	if err != nil {
		s.logger.Error("token exchange failed", "error", err, "provider", provider.Name)
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "token exchange failed",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Parse and validate ID token
	claims, err := client.ParseIDToken(tokenResp.IDToken, oidcState.Nonce)
	if err != nil {
		s.logger.Error("ID token validation failed", "error", err, "provider", provider.Name)
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "ID token validation failed",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Convert claims to user info
	userInfo := oidc.ClaimsToUserInfo(claims, provider.GroupsClaim)

	// Try to fetch additional user info if available
	if extendedInfo, err := client.GetUserInfo(ctx, tokenResp.AccessToken); err == nil {
		if userInfo.Email == "" && extendedInfo.Email != "" {
			userInfo.Email = extendedInfo.Email
			userInfo.EmailVerified = extendedInfo.EmailVerified
		}
		if userInfo.Name == "" && extendedInfo.Name != "" {
			userInfo.Name = extendedInfo.Name
		}
		if len(userInfo.Groups) == 0 && len(extendedInfo.Groups) > 0 {
			userInfo.Groups = extendedInfo.Groups
		}
	}

	// Provision or update user
	user, err := s.provisionUser(ctx, provider, userInfo)
	if err != nil {
		s.logger.Error("user provisioning failed", "error", err, "provider", provider.Name, "subject", claims.Subject)
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "user provisioning failed",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Generate JWT token
	expiresAt := time.Now().Add(s.authCfg.JWTExpiration)
	token, err := s.generateJWT(user, expiresAt)
	if err != nil {
		return &models.OIDCCallbackResponse{
			Success:     false,
			Error:       "failed to generate token",
			RedirectURI: oidcState.RedirectURI,
		}, nil
	}

	// Log successful login
	s.logAuditEvent(ctx, &user.ID, nil, models.AuditActionLogin, ipAddress, userAgent, map[string]interface{}{
		"method":   "oidc",
		"provider": provider.Name,
	})

	s.logger.Info("OIDC login successful",
		"user_id", user.ID,
		"email", user.Email,
		"provider", provider.Name,
	)

	return &models.OIDCCallbackResponse{
		Success:     true,
		Token:       token,
		ExpiresAt:   expiresAt,
		User:        user,
		RedirectURI: oidcState.RedirectURI,
	}, nil
}

// --- Admin Endpoints ---

// CreateProvider creates a new OIDC provider.
func (s *OIDCService) CreateProvider(ctx context.Context, req *models.CreateOIDCProviderRequest) (*models.OIDCProvider, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Apply provider-specific defaults
	s.providerRegistry.ApplyDefaults(req)
	req.ApplyDefaults()

	// Create provider
	provider, err := s.oidcRepo.CreateProvider(ctx, req)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNameExists) {
			return nil, &ConflictError{Message: "provider with this name already exists"}
		}
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	s.logger.Info("OIDC provider created",
		"id", provider.ID,
		"name", provider.Name,
		"type", provider.ProviderType,
	)

	return provider, nil
}

// GetProvider retrieves an OIDC provider by ID.
func (s *OIDCService) GetProvider(ctx context.Context, id uuid.UUID) (*models.OIDCProvider, error) {
	provider, err := s.oidcRepo.GetProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNotFound) {
			return nil, &NotFoundError{Resource: "oidc_provider", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	return provider, nil
}

// ListProviders lists all OIDC providers.
func (s *OIDCService) ListProviders(ctx context.Context) (*models.OIDCProvidersResponse, error) {
	providerList, err := s.oidcRepo.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	summaries := make([]models.OIDCProviderSummary, len(providerList))
	for i := range providerList {
		summaries[i] = *providerList[i].ToSummary()
	}

	return &models.OIDCProvidersResponse{
		Providers:  summaries,
		TotalCount: len(summaries),
	}, nil
}

// UpdateProvider updates an OIDC provider.
func (s *OIDCService) UpdateProvider(ctx context.Context, id uuid.UUID, req *models.UpdateOIDCProviderRequest) (*models.OIDCProvider, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	provider, err := s.oidcRepo.UpdateProvider(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNotFound) {
			return nil, &NotFoundError{Resource: "oidc_provider", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	s.logger.Info("OIDC provider updated", "id", provider.ID, "name", provider.Name)

	return provider, nil
}

// DeleteProvider deletes an OIDC provider.
func (s *OIDCService) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	err := s.oidcRepo.DeleteProvider(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNotFound) {
			return &NotFoundError{Resource: "oidc_provider", ID: id.String()}
		}
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	s.logger.Info("OIDC provider deleted", "id", id)

	return nil
}

// TestProvider tests an OIDC provider's configuration.
func (s *OIDCService) TestProvider(ctx context.Context, id uuid.UUID) error {
	provider, err := s.oidcRepo.GetProviderByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrOIDCProviderNotFound) {
			return &NotFoundError{Resource: "oidc_provider", ID: id.String()}
		}
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Test OIDC discovery
	client := oidc.NewClient(provider.IssuerURL, provider.ClientID, provider.Scopes)
	config, err := client.Discover(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	s.logger.Info("OIDC provider test successful",
		"id", provider.ID,
		"name", provider.Name,
		"issuer", config.Issuer,
	)

	return nil
}

// CleanupExpiredStates removes expired OIDC states.
func (s *OIDCService) CleanupExpiredStates(ctx context.Context) (int64, error) {
	count, err := s.oidcRepo.CleanupExpiredStates(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired states: %w", err)
	}
	if count > 0 {
		s.logger.Info("cleaned up expired OIDC states", "count", count)
	}
	return count, nil
}

// --- Helper Methods ---

// provisionUser creates or updates a user based on OIDC claims.
func (s *OIDCService) provisionUser(ctx context.Context, provider *models.OIDCProvider, userInfo *models.OIDCUserInfo) (*models.User, error) {
	// Try to find existing user by OIDC subject
	existingUser, err := s.userRepo.GetByOIDCSubject(ctx, provider.ID, userInfo.Subject)
	if err == nil {
		// Update existing user's OIDC groups
		groups := userInfo.Groups
		if err := s.userRepo.UpdateOIDCGroups(ctx, existingUser.ID, groups); err != nil {
			s.logger.Warn("failed to update OIDC groups", "user_id", existingUser.ID, "error", err)
		}

		// Update role if group mapping changed
		if newRole := s.mapGroupsToRole(provider, groups); newRole != existingUser.Role {
			if _, err := s.userRepo.Update(ctx, existingUser.ID, &models.UpdateUserRequest{Role: &newRole}); err != nil {
				s.logger.Warn("failed to update role from groups", "user_id", existingUser.ID, "error", err)
			}
			existingUser.Role = newRole
		}

		// Update last login
		if err := s.userRepo.UpdateLastLogin(ctx, existingUser.ID); err != nil {
			s.logger.Warn("failed to update last login", "user_id", existingUser.ID, "error", err)
		}

		return existingUser, nil
	}

	// Try to find user by email and link OIDC
	if userInfo.Email != "" {
		if existingUser, err := s.userRepo.GetByEmail(ctx, userInfo.Email); err == nil {
			// Link OIDC to existing user
			if err := s.userRepo.LinkOIDCProvider(ctx, existingUser.ID, provider.ID, userInfo.Subject, userInfo.Groups); err != nil {
				return nil, fmt.Errorf("failed to link OIDC provider: %w", err)
			}
			s.logger.Info("linked OIDC to existing user", "user_id", existingUser.ID, "provider", provider.Name)

			if err := s.userRepo.UpdateLastLogin(ctx, existingUser.ID); err != nil {
				s.logger.Warn("failed to update last login", "user_id", existingUser.ID, "error", err)
			}

			return existingUser, nil
		}
	}

	// Create new user if auto-create is enabled
	if !provider.AutoCreateUsers {
		return nil, fmt.Errorf("user not found and auto-create is disabled")
	}

	// Determine role from groups
	role := s.mapGroupsToRole(provider, userInfo.Groups)

	// Determine name
	name := userInfo.Name
	if name == "" {
		if userInfo.GivenName != "" || userInfo.FamilyName != "" {
			name = userInfo.GivenName + " " + userInfo.FamilyName
		} else {
			name = userInfo.Email
		}
	}

	// Create OIDC user
	user, err := s.userRepo.CreateOIDCUser(ctx, userInfo.Email, name, role, provider.ID, userInfo.Subject, userInfo.Groups)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC user: %w", err)
	}

	s.logger.Info("created OIDC user",
		"user_id", user.ID,
		"email", user.Email,
		"provider", provider.Name,
		"role", role,
	)

	return user, nil
}

// mapGroupsToRole maps IdP groups to a Philotes role.
func (s *OIDCService) mapGroupsToRole(provider *models.OIDCProvider, groups []string) models.UserRole {
	// Check role mapping
	for _, group := range groups {
		if role, ok := provider.RoleMapping[group]; ok {
			return role
		}
	}
	return provider.DefaultRole
}

// validateRedirectURI validates that the redirect URI is allowed.
func (s *OIDCService) validateRedirectURI(redirectURI string) error {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("redirect URI must use http or https scheme")
	}

	if parsed.Host == "" {
		return fmt.Errorf("redirect URI must have a host")
	}

	// Allow localhost for development
	if parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" {
		return nil
	}

	// Allow same host as base URL
	if baseURL, err := url.Parse(s.baseURL); err == nil && parsed.Host == baseURL.Host {
		return nil
	}

	return nil
}

// generateJWT generates a JWT token for a user.
func (s *OIDCService) generateJWT(user *models.User, expiresAt time.Time) (string, error) {
	permissions := models.RolePermissions[user.Role]

	claims := &models.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "philotes",
		},
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.authCfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// logAuditEvent logs an audit event asynchronously.
func (s *OIDCService) logAuditEvent(ctx context.Context, userID, apiKeyID *uuid.UUID, action, ipAddress, userAgent string, details map[string]interface{}) {
	if s.auditRepo == nil {
		return
	}

	log := &models.AuditLog{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   details,
	}

	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.auditRepo.Create(auditCtx, log); err != nil {
			s.logger.Warn("failed to create audit log", "action", action, "error", err)
		}
	}()
}
