// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/config"
)

// Auth service errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrAdminAlreadyExists = errors.New("admin user already exists")
)

// AuthService provides authentication business logic.
type AuthService struct {
	userRepo  *repositories.UserRepository
	auditRepo *repositories.AuditRepository
	cfg       *config.AuthConfig
	logger    *slog.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo *repositories.UserRepository,
	auditRepo *repositories.AuditRepository,
	cfg *config.AuthConfig,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		auditRepo: auditRepo,
		cfg:       cfg,
		logger:    logger.With("component", "auth-service"),
	}
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, ipAddress, userAgent string) (*models.LoginResponse, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Get user by email
	user, passwordHash, err := s.userRepo.GetByEmailWithPassword(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			s.logAuditEvent(ctx, nil, nil, models.AuditActionLoginFailed, ipAddress, userAgent, map[string]interface{}{
				"email":  req.Email,
				"reason": "user not found",
			})
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		s.logAuditEvent(ctx, &user.ID, nil, models.AuditActionLoginFailed, ipAddress, userAgent, map[string]interface{}{
			"reason": "user inactive",
		})
		return nil, ErrUserInactive
	}

	// Verify password
	if pwdErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); pwdErr != nil {
		s.logAuditEvent(ctx, &user.ID, nil, models.AuditActionLoginFailed, ipAddress, userAgent, map[string]interface{}{
			"reason": "invalid password",
		})
		return nil, ErrInvalidCredentials
	}

	// Generate JWT token
	expiresAt := time.Now().Add(s.cfg.JWTExpiration)
	token, err := s.generateJWT(user, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("failed to update last login", "user_id", user.ID, "error", err)
	}

	// Log successful login
	s.logAuditEvent(ctx, &user.ID, nil, models.AuditActionLogin, ipAddress, userAgent, nil)

	s.logger.Info("user logged in", "user_id", user.ID, "email", user.Email)

	return &models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// ValidateJWT validates a JWT token and returns the claims.
func (s *AuthService) ValidateJWT(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by ID.
func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, &NotFoundError{Resource: "user", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// CreateUser creates a new user.
func (s *AuthService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Apply defaults
	req.ApplyDefaults()

	// Hash password
	passwordHash, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.Create(ctx, req.Email, passwordHash, req.Name, req.Role)
	if err != nil {
		if errors.Is(err, repositories.ErrUserEmailExists) {
			return nil, &ConflictError{Message: "user with this email already exists"}
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user created", "user_id", user.ID, "email", user.Email, "role", user.Role)

	return user, nil
}

// ListUsers lists all users.
func (s *AuthService) ListUsers(ctx context.Context) ([]models.User, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

// UpdateUser updates a user.
func (s *AuthService) UpdateUser(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := s.userRepo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, &NotFoundError{Resource: "user", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("user updated", "user_id", user.ID, "email", user.Email)

	return user, nil
}

// DeleteUser deletes a user.
func (s *AuthService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := s.userRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return &NotFoundError{Resource: "user", ID: id.String()}
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info("user deleted", "user_id", id)

	return nil
}

// HashPassword hashes a password using bcrypt.
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BCryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// BootstrapAdmin creates the initial admin user if configured and doesn't exist.
func (s *AuthService) BootstrapAdmin(ctx context.Context) error {
	if s.cfg.AdminEmail == "" || s.cfg.AdminPassword == "" {
		s.logger.Debug("no bootstrap admin configured")
		return nil
	}

	// Check if admin already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, s.cfg.AdminEmail)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}

	if exists {
		s.logger.Debug("bootstrap admin already exists", "email", s.cfg.AdminEmail)
		return nil
	}

	// Create admin user
	req := &models.CreateUserRequest{
		Email:    s.cfg.AdminEmail,
		Password: s.cfg.AdminPassword,
		Name:     "Admin",
		Role:     models.RoleAdmin,
	}

	user, err := s.CreateUser(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create bootstrap admin: %w", err)
	}

	s.logger.Info("bootstrap admin created", "user_id", user.ID, "email", user.Email)

	return nil
}

// generateJWT generates a JWT token for a user.
func (s *AuthService) generateJWT(user *models.User, expiresAt time.Time) (string, error) {
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
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// RegisterFirstAdmin registers the first admin user during onboarding.
// This is only allowed if no admin user exists yet.
func (s *AuthService) RegisterFirstAdmin(ctx context.Context, req *models.RegisterRequest, ipAddress, userAgent string) (*models.RegisterResponse, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Check if admin already exists
	hasAdmin, err := s.userRepo.HasAdminUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin existence: %w", err)
	}
	if hasAdmin {
		return nil, ErrAdminAlreadyExists
	}

	// Create user request
	createReq := &models.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     models.RoleAdmin,
	}

	// Create the admin user
	user, err := s.CreateUser(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	expiresAt := time.Now().Add(s.cfg.JWTExpiration)
	token, err := s.generateJWT(user, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Log audit event
	s.logAuditEvent(ctx, &user.ID, nil, models.AuditActionUserCreated, ipAddress, userAgent, map[string]interface{}{
		"role":   "admin",
		"reason": "first_admin_registration",
	})

	s.logger.Info("first admin registered", "user_id", user.ID, "email", user.Email)

	return &models.RegisterResponse{
		User:  user,
		Token: token,
	}, nil
}

// logAuditEvent logs an audit event asynchronously.
func (s *AuthService) logAuditEvent(ctx context.Context, userID, apiKeyID *uuid.UUID, action, ipAddress, userAgent string, details map[string]interface{}) {
	log := &models.AuditLog{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   details,
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
