// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/crypto"
)

// OIDC repository errors.
var (
	ErrOIDCProviderNotFound   = errors.New("oidc provider not found")
	ErrOIDCProviderNameExists = errors.New("oidc provider with this name already exists")
	ErrOIDCStateNotFound      = errors.New("oidc state not found")
	ErrOIDCStateExpired       = errors.New("oidc state expired")
)

// OIDCRepository handles database operations for OIDC providers and states.
type OIDCRepository struct {
	db        *sql.DB
	encryptor *crypto.Encryptor
}

// NewOIDCRepository creates a new OIDCRepository.
func NewOIDCRepository(db *sql.DB, encryptor *crypto.Encryptor) *OIDCRepository {
	return &OIDCRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// DB returns the underlying database connection for transaction support.
func (r *OIDCRepository) DB() *sql.DB {
	return r.db
}

// --- OIDC Provider Operations ---

// CreateProvider creates a new OIDC provider.
func (r *OIDCRepository) CreateProvider(ctx context.Context, req *models.CreateOIDCProviderRequest) (*models.OIDCProvider, error) {
	// Encrypt client secret
	encryptedSecret, err := r.encryptor.EncryptToBytes(req.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	// Marshal role mapping to JSON
	roleMappingJSON, err := json.Marshal(req.RoleMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role mapping: %w", err)
	}

	query := `
		INSERT INTO philotes.oidc_providers (
			name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
			scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
		          scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users, created_at, updated_at
	`

	var provider models.OIDCProvider
	var roleMappingRaw []byte
	err = r.db.QueryRowContext(ctx, query,
		req.Name,
		req.DisplayName,
		req.ProviderType,
		req.IssuerURL,
		req.ClientID,
		encryptedSecret,
		pq.Array(req.Scopes),
		req.GroupsClaim,
		roleMappingJSON,
		req.DefaultRole,
		*req.Enabled,
		*req.AutoCreateUsers,
	).Scan(
		&provider.ID,
		&provider.Name,
		&provider.DisplayName,
		&provider.ProviderType,
		&provider.IssuerURL,
		&provider.ClientID,
		&provider.ClientSecretEncrypted,
		pq.Array(&provider.Scopes),
		&provider.GroupsClaim,
		&roleMappingRaw,
		&provider.DefaultRole,
		&provider.Enabled,
		&provider.AutoCreateUsers,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrOIDCProviderNameExists
		}
		return nil, fmt.Errorf("failed to create oidc provider: %w", err)
	}

	// Unmarshal role mapping
	if err := json.Unmarshal(roleMappingRaw, &provider.RoleMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role mapping: %w", err)
	}

	return &provider, nil
}

// GetProviderByID retrieves an OIDC provider by ID.
func (r *OIDCRepository) GetProviderByID(ctx context.Context, id uuid.UUID) (*models.OIDCProvider, error) {
	query := `
		SELECT id, name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
		       scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users, created_at, updated_at
		FROM philotes.oidc_providers
		WHERE id = $1
	`

	return r.scanProvider(r.db.QueryRowContext(ctx, query, id))
}

// GetProviderByName retrieves an OIDC provider by name.
func (r *OIDCRepository) GetProviderByName(ctx context.Context, name string) (*models.OIDCProvider, error) {
	query := `
		SELECT id, name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
		       scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users, created_at, updated_at
		FROM philotes.oidc_providers
		WHERE name = $1
	`

	return r.scanProvider(r.db.QueryRowContext(ctx, query, name))
}

// ListProviders retrieves all OIDC providers.
func (r *OIDCRepository) ListProviders(ctx context.Context) ([]models.OIDCProvider, error) {
	query := `
		SELECT id, name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
		       scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users, created_at, updated_at
		FROM philotes.oidc_providers
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list oidc providers: %w", err)
	}
	defer rows.Close()

	var providers []models.OIDCProvider
	for rows.Next() {
		provider, err := r.scanProviderFromRows(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, *provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate oidc providers: %w", err)
	}

	return providers, nil
}

// ListEnabledProviders retrieves all enabled OIDC providers.
func (r *OIDCRepository) ListEnabledProviders(ctx context.Context) ([]models.OIDCProvider, error) {
	query := `
		SELECT id, name, display_name, provider_type, issuer_url, client_id, client_secret_encrypted,
		       scopes, groups_claim, role_mapping, default_role, enabled, auto_create_users, created_at, updated_at
		FROM philotes.oidc_providers
		WHERE enabled = true
		ORDER BY display_name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled oidc providers: %w", err)
	}
	defer rows.Close()

	var providers []models.OIDCProvider
	for rows.Next() {
		provider, err := r.scanProviderFromRows(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, *provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate enabled oidc providers: %w", err)
	}

	return providers, nil
}

// UpdateProvider updates an OIDC provider.
func (r *OIDCRepository) UpdateProvider(ctx context.Context, id uuid.UUID, req *models.UpdateOIDCProviderRequest) (*models.OIDCProvider, error) {
	// First check if provider exists
	_, err := r.GetProviderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.oidc_providers SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.DisplayName != nil {
		query += fmt.Sprintf(", display_name = $%d", argIdx)
		args = append(args, *req.DisplayName)
		argIdx++
	}
	if req.IssuerURL != nil {
		query += fmt.Sprintf(", issuer_url = $%d", argIdx)
		args = append(args, *req.IssuerURL)
		argIdx++
	}
	if req.ClientID != nil {
		query += fmt.Sprintf(", client_id = $%d", argIdx)
		args = append(args, *req.ClientID)
		argIdx++
	}
	if req.ClientSecret != nil {
		encryptedSecret, err := r.encryptor.EncryptToBytes(*req.ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
		}
		query += fmt.Sprintf(", client_secret_encrypted = $%d", argIdx)
		args = append(args, encryptedSecret)
		argIdx++
	}
	if req.Scopes != nil {
		query += fmt.Sprintf(", scopes = $%d", argIdx)
		args = append(args, pq.Array(req.Scopes))
		argIdx++
	}
	if req.GroupsClaim != nil {
		query += fmt.Sprintf(", groups_claim = $%d", argIdx)
		args = append(args, *req.GroupsClaim)
		argIdx++
	}
	if req.RoleMapping != nil {
		roleMappingJSON, err := json.Marshal(req.RoleMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal role mapping: %w", err)
		}
		query += fmt.Sprintf(", role_mapping = $%d", argIdx)
		args = append(args, roleMappingJSON)
		argIdx++
	}
	if req.DefaultRole != nil {
		query += fmt.Sprintf(", default_role = $%d", argIdx)
		args = append(args, *req.DefaultRole)
		argIdx++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argIdx)
		args = append(args, *req.Enabled)
		argIdx++
	}
	if req.AutoCreateUsers != nil {
		query += fmt.Sprintf(", auto_create_users = $%d", argIdx)
		args = append(args, *req.AutoCreateUsers)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update oidc provider: %w", err)
	}

	return r.GetProviderByID(ctx, id)
}

// DeleteProvider deletes an OIDC provider.
func (r *OIDCRepository) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.oidc_providers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete oidc provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOIDCProviderNotFound
	}

	return nil
}

// GetProviderClientSecret retrieves and decrypts the client secret for a provider.
func (r *OIDCRepository) GetProviderClientSecret(ctx context.Context, id uuid.UUID) (string, error) {
	provider, err := r.GetProviderByID(ctx, id)
	if err != nil {
		return "", err
	}

	secret, err := r.encryptor.DecryptFromBytes(provider.ClientSecretEncrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	return secret, nil
}

// --- OIDC State Operations ---

// CreateState creates a new OIDC state.
func (r *OIDCRepository) CreateState(ctx context.Context, state *models.OIDCState) error {
	query := `
		INSERT INTO philotes.oidc_states (
			id, state, nonce, code_verifier, provider_id, redirect_uri, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		state.ID,
		state.State,
		state.Nonce,
		state.CodeVerifier,
		state.ProviderID,
		state.RedirectURI,
		state.ExpiresAt,
		state.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create oidc state: %w", err)
	}

	return nil
}

// GetState retrieves an OIDC state by state parameter.
func (r *OIDCRepository) GetState(ctx context.Context, state string) (*models.OIDCState, error) {
	query := `
		SELECT id, state, nonce, code_verifier, provider_id, redirect_uri, created_at, expires_at
		FROM philotes.oidc_states
		WHERE state = $1
	`

	var oidcState models.OIDCState
	err := r.db.QueryRowContext(ctx, query, state).Scan(
		&oidcState.ID,
		&oidcState.State,
		&oidcState.Nonce,
		&oidcState.CodeVerifier,
		&oidcState.ProviderID,
		&oidcState.RedirectURI,
		&oidcState.CreatedAt,
		&oidcState.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOIDCStateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get oidc state: %w", err)
	}

	// Check if expired
	if time.Now().After(oidcState.ExpiresAt) {
		return nil, ErrOIDCStateExpired
	}

	return &oidcState, nil
}

// DeleteState removes an OIDC state (one-time use).
func (r *OIDCRepository) DeleteState(ctx context.Context, state string) error {
	query := `DELETE FROM philotes.oidc_states WHERE state = $1`
	_, err := r.db.ExecContext(ctx, query, state)
	if err != nil {
		return fmt.Errorf("failed to delete oidc state: %w", err)
	}
	return nil
}

// CleanupExpiredStates removes all expired OIDC states.
func (r *OIDCRepository) CleanupExpiredStates(ctx context.Context) (int64, error) {
	query := `DELETE FROM philotes.oidc_states WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired oidc states: %w", err)
	}
	return result.RowsAffected()
}

// --- Helper Functions ---

func (r *OIDCRepository) scanProvider(row *sql.Row) (*models.OIDCProvider, error) {
	var provider models.OIDCProvider
	var roleMappingRaw []byte

	err := row.Scan(
		&provider.ID,
		&provider.Name,
		&provider.DisplayName,
		&provider.ProviderType,
		&provider.IssuerURL,
		&provider.ClientID,
		&provider.ClientSecretEncrypted,
		pq.Array(&provider.Scopes),
		&provider.GroupsClaim,
		&roleMappingRaw,
		&provider.DefaultRole,
		&provider.Enabled,
		&provider.AutoCreateUsers,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOIDCProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan oidc provider: %w", err)
	}

	// Unmarshal role mapping
	if err := json.Unmarshal(roleMappingRaw, &provider.RoleMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role mapping: %w", err)
	}

	return &provider, nil
}

func (r *OIDCRepository) scanProviderFromRows(rows *sql.Rows) (*models.OIDCProvider, error) {
	var provider models.OIDCProvider
	var roleMappingRaw []byte

	err := rows.Scan(
		&provider.ID,
		&provider.Name,
		&provider.DisplayName,
		&provider.ProviderType,
		&provider.IssuerURL,
		&provider.ClientID,
		&provider.ClientSecretEncrypted,
		pq.Array(&provider.Scopes),
		&provider.GroupsClaim,
		&roleMappingRaw,
		&provider.DefaultRole,
		&provider.Enabled,
		&provider.AutoCreateUsers,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan oidc provider: %w", err)
	}

	// Unmarshal role mapping
	if err := json.Unmarshal(roleMappingRaw, &provider.RoleMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role mapping: %w", err)
	}

	return &provider, nil
}
