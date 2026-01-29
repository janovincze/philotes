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

// API key repository errors.
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
)

// APIKeyRepository handles database operations for API keys.
type APIKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new APIKeyRepository.
func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// apiKeyRow represents a database row for an API key.
type apiKeyRow struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Name        string
	KeyPrefix   string
	KeyHash     string
	Permissions []string
	LastUsedAt  sql.NullTime
	ExpiresAt   sql.NullTime
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// toModel converts a database row to an API model.
func (r *apiKeyRow) toModel() *models.APIKey {
	key := &models.APIKey{
		ID:          r.ID,
		UserID:      r.UserID,
		Name:        r.Name,
		KeyPrefix:   r.KeyPrefix,
		Permissions: r.Permissions,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.LastUsedAt.Valid {
		key.LastUsedAt = &r.LastUsedAt.Time
	}
	if r.ExpiresAt.Valid {
		key.ExpiresAt = &r.ExpiresAt.Time
	}
	return key
}

// Create creates a new API key in the database.
func (r *APIKeyRepository) Create(ctx context.Context, userID uuid.UUID, name, keyPrefix, keyHash string, permissions []string, expiresAt *time.Time) (*models.APIKey, error) {
	query := `
		INSERT INTO philotes.api_keys (user_id, name, key_prefix, key_hash, permissions, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, key_prefix, key_hash, permissions, last_used_at, expires_at, is_active, created_at, updated_at
	`

	var row apiKeyRow
	var expiresAtArg sql.NullTime
	if expiresAt != nil {
		expiresAtArg = sql.NullTime{Time: *expiresAt, Valid: true}
	}

	err := r.db.QueryRowContext(ctx, query, userID, name, keyPrefix, keyHash, pq.Array(permissions), expiresAtArg).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.KeyPrefix,
		&row.KeyHash,
		pq.Array(&row.Permissions),
		&row.LastUsedAt,
		&row.ExpiresAt,
		&row.IsActive,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves an API key by ID.
func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, permissions, last_used_at, expires_at, is_active, created_at, updated_at
		FROM philotes.api_keys
		WHERE id = $1
	`

	var row apiKeyRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.KeyPrefix,
		&row.KeyHash,
		pq.Array(&row.Permissions),
		&row.LastUsedAt,
		&row.ExpiresAt,
		&row.IsActive,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return row.toModel(), nil
}

// GetByHash retrieves an API key by its hash.
func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, permissions, last_used_at, expires_at, is_active, created_at, updated_at
		FROM philotes.api_keys
		WHERE key_hash = $1 AND is_active = true
	`

	var row apiKeyRow
	err := r.db.QueryRowContext(ctx, query, keyHash).Scan(
		&row.ID,
		&row.UserID,
		&row.Name,
		&row.KeyPrefix,
		&row.KeyHash,
		pq.Array(&row.Permissions),
		&row.LastUsedAt,
		&row.ExpiresAt,
		&row.IsActive,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return row.toModel(), nil
}

// ListByUserID retrieves all API keys for a user.
func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, permissions, last_used_at, expires_at, is_active, created_at, updated_at
		FROM philotes.api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var row apiKeyRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.Name,
			&row.KeyPrefix,
			&row.KeyHash,
			pq.Array(&row.Permissions),
			&row.LastUsedAt,
			&row.ExpiresAt,
			&row.IsActive,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key row: %w", err)
		}
		keys = append(keys, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate api keys: %w", err)
	}

	return keys, nil
}

// UpdateLastUsed updates the last used timestamp for an API key.
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE philotes.api_keys
		SET last_used_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// Revoke deactivates an API key.
func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE philotes.api_keys
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// Delete permanently deletes an API key.
func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.api_keys WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// IsExpired checks if an API key is expired.
func (r *APIKeyRepository) IsExpired(key *models.APIKey) bool {
	if key.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*key.ExpiresAt)
}
