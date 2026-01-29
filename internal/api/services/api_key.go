// Package services provides business logic for API resources.
package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/config"
)

// API key service errors.
var (
	ErrAPIKeyExpired  = errors.New("api key has expired")
	ErrAPIKeyInactive = errors.New("api key is inactive")
)

// APIKeyService provides API key management business logic.
type APIKeyService struct {
	apiKeyRepo *repositories.APIKeyRepository
	userRepo   *repositories.UserRepository
	auditRepo  *repositories.AuditRepository
	cfg        *config.AuthConfig
	logger     *slog.Logger
}

// NewAPIKeyService creates a new APIKeyService.
func NewAPIKeyService(
	apiKeyRepo *repositories.APIKeyRepository,
	userRepo *repositories.UserRepository,
	auditRepo *repositories.AuditRepository,
	cfg *config.AuthConfig,
	logger *slog.Logger,
) *APIKeyService {
	return &APIKeyService{
		apiKeyRepo: apiKeyRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
		cfg:        cfg,
		logger:     logger.With("component", "api-key-service"),
	}
}

// Create creates a new API key for a user.
func (s *APIKeyService) Create(ctx context.Context, userID uuid.UUID, req *models.CreateAPIKeyRequest, ipAddress, userAgent string) (*models.CreateAPIKeyResponse, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, &NotFoundError{Resource: "user", ID: userID.String()}
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get user's allowed permissions based on role
	userPermissions := models.RolePermissions[user.Role]
	userPermSet := make(map[string]bool)
	for _, p := range userPermissions {
		userPermSet[p] = true
	}

	// Determine and validate permissions
	permissions := req.Permissions
	if len(permissions) == 0 {
		// Default to user's role permissions
		permissions = userPermissions
	} else {
		// Validate requested permissions are subset of user's permissions
		var invalidPerms []string
		for _, p := range permissions {
			if !userPermSet[p] {
				invalidPerms = append(invalidPerms, p)
			}
		}
		if len(invalidPerms) > 0 {
			return nil, &ValidationError{Errors: []models.FieldError{
				{Field: "permissions", Message: fmt.Sprintf("cannot grant permissions you don't have: %v", invalidPerms)},
			}}
		}
	}

	// Generate API key
	plaintextKey, keyPrefix, keyHash, err := s.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// Create API key in database
	apiKey, err := s.apiKeyRepo.Create(ctx, userID, req.Name, keyPrefix, keyHash, permissions, req.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &userID, &apiKey.ID, models.AuditActionAPIKeyCreated, ipAddress, userAgent, map[string]interface{}{
		"key_name": req.Name,
	})

	s.logger.Info("api key created", "api_key_id", apiKey.ID, "user_id", userID, "name", req.Name)

	return &models.CreateAPIKeyResponse{
		APIKey: apiKey,
		Key:    plaintextKey,
	}, nil
}

// Validate validates an API key and returns the associated user and API key.
func (s *APIKeyService) Validate(ctx context.Context, plaintextKey string) (*models.User, *models.APIKey, error) {
	// Hash the key
	keyHash := s.hashKey(plaintextKey)

	// Look up API key by hash
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return nil, nil, repositories.ErrAPIKeyNotFound
		}
		return nil, nil, fmt.Errorf("failed to get api key: %w", err)
	}

	// Check if key is active
	if !apiKey.IsActive {
		return nil, nil, ErrAPIKeyInactive
	}

	// Check if key is expired
	if s.apiKeyRepo.IsExpired(apiKey) {
		return nil, nil, ErrAPIKeyExpired
	}

	// Get associated user
	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Update last used timestamp asynchronously
	go func() {
		if err := s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID); err != nil {
			s.logger.Warn("failed to update api key last used", "api_key_id", apiKey.ID, "error", err)
		}
	}()

	return user, apiKey, nil
}

// List lists all API keys for a user.
func (s *APIKeyService) List(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	keys, err := s.apiKeyRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	if keys == nil {
		keys = []models.APIKey{}
	}
	return keys, nil
}

// Get retrieves an API key by ID.
func (s *APIKeyService) Get(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return nil, &NotFoundError{Resource: "api_key", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return apiKey, nil
}

// Revoke deactivates an API key.
func (s *APIKeyService) Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID, userRole models.UserRole, ipAddress, userAgent string) error {
	// Get the API key first to verify ownership
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return &NotFoundError{Resource: "api_key", ID: id.String()}
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	// Verify ownership (admins can revoke any key)
	if apiKey.UserID != userID && userRole != models.RoleAdmin {
		return &NotFoundError{Resource: "api_key", ID: id.String()}
	}

	// Revoke the key
	if err := s.apiKeyRepo.Revoke(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return &NotFoundError{Resource: "api_key", ID: id.String()}
		}
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &userID, &id, models.AuditActionAPIKeyRevoked, ipAddress, userAgent, map[string]interface{}{
		"key_name": apiKey.Name,
	})

	s.logger.Info("api key revoked", "api_key_id", id, "user_id", userID)

	return nil
}

// Delete permanently deletes an API key.
func (s *APIKeyService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, userRole models.UserRole, ipAddress, userAgent string) error {
	// Get the API key first to verify ownership
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return &NotFoundError{Resource: "api_key", ID: id.String()}
		}
		return fmt.Errorf("failed to get api key: %w", err)
	}

	// Verify ownership (admins can delete any key)
	if apiKey.UserID != userID && userRole != models.RoleAdmin {
		return &NotFoundError{Resource: "api_key", ID: id.String()}
	}

	// Delete the key
	if err := s.apiKeyRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repositories.ErrAPIKeyNotFound) {
			return &NotFoundError{Resource: "api_key", ID: id.String()}
		}
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &userID, &id, models.AuditActionAPIKeyRevoked, ipAddress, userAgent, map[string]interface{}{
		"key_name": apiKey.Name,
		"deleted":  true,
	})

	s.logger.Info("api key deleted", "api_key_id", id, "user_id", userID)

	return nil
}

// generateAPIKey generates a new API key.
// Returns: plaintext key, key prefix (first 8 chars), key hash, error
func (s *APIKeyService) generateAPIKey() (string, string, string, error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as hex
	randomPart := hex.EncodeToString(randomBytes)

	// Build full key with prefix
	// Format: pk_live_[64 hex chars]
	plaintextKey := fmt.Sprintf("%slive_%s", s.cfg.APIKeyPrefix, randomPart)

	// Extract prefix for storage (first 8 chars)
	keyPrefix := plaintextKey[:8]

	// Hash the full key
	keyHash := s.hashKey(plaintextKey)

	return plaintextKey, keyPrefix, keyHash, nil
}

// hashKey hashes an API key using SHA256.
func (s *APIKeyService) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// logAuditEvent logs an audit event asynchronously.
func (s *APIKeyService) logAuditEvent(ctx context.Context, userID, apiKeyID *uuid.UUID, action, ipAddress, userAgent string, details map[string]interface{}) {
	log := &models.AuditLog{
		UserID:       userID,
		APIKeyID:     apiKeyID,
		Action:       action,
		ResourceType: "api_key",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Details:      details,
	}
	if apiKeyID != nil {
		log.ResourceID = apiKeyID
	}

	// Log asynchronously with timeout to prevent goroutine leaks
	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.auditRepo.Create(auditCtx, log); err != nil {
			s.logger.Warn("failed to create audit log", "action", action, "error", err)
		}
	}()
}

// CleanupExpiredKeys removes expired API keys.
func (s *APIKeyService) CleanupExpiredKeys(ctx context.Context) (int, error) {
	// This could be implemented as a periodic cleanup job
	// For now, we just rely on the expiration check during validation
	return 0, nil
}
