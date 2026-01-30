// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/janovincze/philotes/internal/api/models"
)

// User repository errors.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserEmailExists = errors.New("user with this email already exists")
)

// UserRepository handles database operations for users.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// userRow represents a database row for a user.
type userRow struct {
	ID             uuid.UUID
	Email          string
	PasswordHash   sql.NullString
	Name           sql.NullString
	Role           string
	IsActive       bool
	LastLoginAt    sql.NullTime
	OIDCProviderID sql.NullString
	OIDCSubject    sql.NullString
	OIDCGroups     []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// toModel converts a database row to an API model.
func (r *userRow) toModel() *models.User {
	user := &models.User{
		ID:          r.ID,
		Email:       r.Email,
		Name:        r.Name.String,
		Role:        models.UserRole(r.Role),
		IsActive:    r.IsActive,
		OIDCSubject: r.OIDCSubject.String,
		OIDCGroups:  r.OIDCGroups,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.LastLoginAt.Valid {
		user.LastLoginAt = &r.LastLoginAt.Time
	}
	if r.OIDCProviderID.Valid {
		if pid, err := uuid.Parse(r.OIDCProviderID.String); err == nil {
			user.OIDCProviderID = &pid
		}
	}
	return user
}

// Create creates a new user in the database.
func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name string, role models.UserRole) (*models.User, error) {
	query := `
		INSERT INTO philotes.users (email, password_hash, name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, name, role, is_active, last_login_at,
		          oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, email, nullString(passwordHash), nullString(name), role).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserEmailExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, is_active, last_login_at,
		       oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
		FROM philotes.users
		WHERE id = $1
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return row.toModel(), nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, is_active, last_login_at,
		       oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
		FROM philotes.users
		WHERE email = $1
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return row.toModel(), nil
}

// GetByEmailWithPassword retrieves a user by email including the password hash.
func (r *UserRepository) GetByEmailWithPassword(ctx context.Context, email string) (*models.User, string, error) {
	query := `
		SELECT id, email, password_hash, name, role, is_active, last_login_at,
		       oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
		FROM philotes.users
		WHERE email = $1
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrUserNotFound
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	return row.toModel(), row.PasswordHash.String, nil
}

// List retrieves all users.
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, is_active, last_login_at,
		       oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
		FROM philotes.users
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var row userRow
		err := rows.Scan(
			&row.ID,
			&row.Email,
			&row.PasswordHash,
			&row.Name,
			&row.Role,
			&row.IsActive,
			&row.LastLoginAt,
			&row.OIDCProviderID,
			&row.OIDCSubject,
			pq.Array(&row.OIDCGroups),
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// Update updates a user in the database.
func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateUserRequest) (*models.User, error) {
	// First check if user exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.users SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, nullString(*req.Name))
		argIdx++
	}
	if req.Role != nil {
		query += fmt.Sprintf(", role = $%d", argIdx)
		args = append(args, *req.Role)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return r.GetByID(ctx, id)
}

// UpdateLastLogin updates the last login time for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE philotes.users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates a user's password.
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE philotes.users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete deletes a user from the database.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM philotes.users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// HasAdminUser checks if any active admin user exists in the system.
func (r *UserRepository) HasAdminUser(ctx context.Context) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM philotes.users WHERE role = 'admin' AND is_active = true)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check admin existence: %w", err)
	}

	return exists, nil
}

// --- OIDC User Operations ---

// GetByOIDCSubject retrieves a user by OIDC provider ID and subject.
func (r *UserRepository) GetByOIDCSubject(ctx context.Context, providerID uuid.UUID, subject string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, is_active, last_login_at,
		       oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
		FROM philotes.users
		WHERE oidc_provider_id = $1 AND oidc_subject = $2
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, providerID, subject).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by oidc subject: %w", err)
	}

	return row.toModel(), nil
}

// CreateOIDCUser creates a new user from OIDC authentication.
func (r *UserRepository) CreateOIDCUser(ctx context.Context, email, name string, role models.UserRole, providerID uuid.UUID, subject string, groups []string) (*models.User, error) {
	query := `
		INSERT INTO philotes.users (email, name, role, oidc_provider_id, oidc_subject, oidc_groups)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, password_hash, name, role, is_active, last_login_at,
		          oidc_provider_id, oidc_subject, oidc_groups, created_at, updated_at
	`

	var row userRow
	err := r.db.QueryRowContext(ctx, query, email, nullString(name), role, providerID, subject, pq.Array(groups)).Scan(
		&row.ID,
		&row.Email,
		&row.PasswordHash,
		&row.Name,
		&row.Role,
		&row.IsActive,
		&row.LastLoginAt,
		&row.OIDCProviderID,
		&row.OIDCSubject,
		pq.Array(&row.OIDCGroups),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserEmailExists
		}
		return nil, fmt.Errorf("failed to create oidc user: %w", err)
	}

	return row.toModel(), nil
}

// UpdateOIDCInfo updates a user's OIDC-related information.
func (r *UserRepository) UpdateOIDCInfo(ctx context.Context, id uuid.UUID, providerID *uuid.UUID, subject string, groups []string) error {
	query := `
		UPDATE philotes.users
		SET oidc_provider_id = $1, oidc_subject = $2, oidc_groups = $3, updated_at = NOW()
		WHERE id = $4
	`

	var providerIDStr interface{}
	if providerID != nil {
		providerIDStr = *providerID
	}

	result, err := r.db.ExecContext(ctx, query, providerIDStr, nullString(subject), pq.Array(groups), id)
	if err != nil {
		return fmt.Errorf("failed to update oidc info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// LinkOIDCProvider links an existing user to an OIDC provider.
func (r *UserRepository) LinkOIDCProvider(ctx context.Context, userID, providerID uuid.UUID, subject string, groups []string) error {
	return r.UpdateOIDCInfo(ctx, userID, &providerID, subject, groups)
}

// UnlinkOIDCProvider unlinks a user from their OIDC provider.
func (r *UserRepository) UnlinkOIDCProvider(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE philotes.users
		SET oidc_provider_id = NULL, oidc_subject = NULL, oidc_groups = '{}', updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unlink oidc provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateOIDCGroups updates only the OIDC groups for a user.
func (r *UserRepository) UpdateOIDCGroups(ctx context.Context, id uuid.UUID, groups []string) error {
	query := `
		UPDATE philotes.users
		SET oidc_groups = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, pq.Array(groups), id)
	if err != nil {
		return fmt.Errorf("failed to update oidc groups: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
