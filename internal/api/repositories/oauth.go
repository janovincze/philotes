// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
)

// OAuth state repository errors
var (
	ErrOAuthStateNotFound = errors.New("oauth state not found")
	ErrOAuthStateExpired  = errors.New("oauth state expired")
	ErrCredentialNotFound = errors.New("credential not found")
)

// OAuthRepository handles OAuth state and credential storage.
type OAuthRepository struct {
	db *sql.DB
}

// NewOAuthRepository creates a new OAuthRepository.
func NewOAuthRepository(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// DB returns the underlying database connection for transaction support.
func (r *OAuthRepository) DB() *sql.DB {
	return r.db
}

// --- OAuth State Operations ---

// CreateState stores a new OAuth state for PKCE flow.
func (r *OAuthRepository) CreateState(ctx context.Context, state *models.OAuthState) error {
	query := `
		INSERT INTO philotes.oauth_states (
			id, provider, state, code_verifier, redirect_uri, user_id, session_id, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		state.ID,
		state.Provider,
		state.State,
		state.CodeVerifier,
		state.RedirectURI,
		state.UserID,
		state.SessionID,
		state.ExpiresAt,
		state.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create oauth state: %w", err)
	}

	return nil
}

// GetStateByState retrieves an OAuth state by its state parameter.
func (r *OAuthRepository) GetStateByState(ctx context.Context, state string) (*models.OAuthState, error) {
	query := `
		SELECT id, provider, state, code_verifier, redirect_uri, user_id, session_id, expires_at, created_at
		FROM philotes.oauth_states
		WHERE state = $1
	`

	var oauthState models.OAuthState
	var userID sql.NullString
	var sessionID sql.NullString

	err := r.db.QueryRowContext(ctx, query, state).Scan(
		&oauthState.ID,
		&oauthState.Provider,
		&oauthState.State,
		&oauthState.CodeVerifier,
		&oauthState.RedirectURI,
		&userID,
		&sessionID,
		&oauthState.ExpiresAt,
		&oauthState.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOAuthStateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth state: %w", err)
	}

	if userID.Valid {
		uid, _ := uuid.Parse(userID.String)
		oauthState.UserID = &uid
	}
	if sessionID.Valid {
		oauthState.SessionID = sessionID.String
	}

	// Check if expired
	if time.Now().After(oauthState.ExpiresAt) {
		return nil, ErrOAuthStateExpired
	}

	return &oauthState, nil
}

// DeleteState removes an OAuth state (one-time use).
func (r *OAuthRepository) DeleteState(ctx context.Context, state string) error {
	query := `DELETE FROM philotes.oauth_states WHERE state = $1`
	_, err := r.db.ExecContext(ctx, query, state)
	if err != nil {
		return fmt.Errorf("failed to delete oauth state: %w", err)
	}
	return nil
}

// CleanupExpiredStates removes all expired OAuth states.
func (r *OAuthRepository) CleanupExpiredStates(ctx context.Context) (int64, error) {
	query := `DELETE FROM philotes.oauth_states WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired states: %w", err)
	}
	return result.RowsAffected()
}

// --- Cloud Credential Operations ---

// CreateCredential stores a new cloud credential.
func (r *OAuthRepository) CreateCredential(ctx context.Context, cred *models.CloudCredential) error {
	query := `
		INSERT INTO philotes.cloud_credentials (
			id, deployment_id, user_id, provider, credential_type,
			credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		cred.ID,
		cred.DeploymentID,
		cred.UserID,
		cred.Provider,
		cred.CredentialType,
		cred.CredentialsEncrypted,
		cred.RefreshTokenEncrypted,
		cred.TokenExpiresAt,
		cred.ExpiresAt,
		cred.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	return nil
}

// GetCredentialByID retrieves a credential by ID.
func (r *OAuthRepository) GetCredentialByID(ctx context.Context, id uuid.UUID) (*models.CloudCredential, error) {
	query := `
		SELECT id, deployment_id, user_id, provider, credential_type,
		       credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		FROM philotes.cloud_credentials
		WHERE id = $1
	`

	return r.scanCredential(r.db.QueryRowContext(ctx, query, id))
}

// GetCredentialByProvider retrieves a credential by provider for a user.
func (r *OAuthRepository) GetCredentialByProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.CloudCredential, error) {
	query := `
		SELECT id, deployment_id, user_id, provider, credential_type,
		       credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		FROM philotes.cloud_credentials
		WHERE user_id = $1 AND provider = $2 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanCredential(r.db.QueryRowContext(ctx, query, userID, provider))
}

// GetCredentialByDeployment retrieves a credential for a deployment.
func (r *OAuthRepository) GetCredentialByDeployment(ctx context.Context, deploymentID uuid.UUID, provider string) (*models.CloudCredential, error) {
	query := `
		SELECT id, deployment_id, user_id, provider, credential_type,
		       credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		FROM philotes.cloud_credentials
		WHERE deployment_id = $1 AND provider = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanCredential(r.db.QueryRowContext(ctx, query, deploymentID, provider))
}

// ListCredentialsByUser lists all non-expired credentials for a user.
func (r *OAuthRepository) ListCredentialsByUser(ctx context.Context, userID uuid.UUID) ([]models.CloudCredential, error) {
	query := `
		SELECT id, deployment_id, user_id, provider, credential_type,
		       credentials_encrypted, refresh_token_encrypted, token_expires_at, expires_at, created_at
		FROM philotes.cloud_credentials
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var credentials []models.CloudCredential
	for rows.Next() {
		cred, err := r.scanCredentialFromRows(rows)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *cred)
	}

	return credentials, rows.Err()
}

// UpdateCredential updates a credential's token data.
func (r *OAuthRepository) UpdateCredential(ctx context.Context, cred *models.CloudCredential) error {
	query := `
		UPDATE philotes.cloud_credentials
		SET credentials_encrypted = $1,
		    refresh_token_encrypted = $2,
		    token_expires_at = $3,
		    expires_at = $4
		WHERE id = $5
	`

	result, err := r.db.ExecContext(ctx, query,
		cred.CredentialsEncrypted,
		cred.RefreshTokenEncrypted,
		cred.TokenExpiresAt,
		cred.ExpiresAt,
		cred.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCredentialNotFound
	}

	return nil
}

// DeleteCredential removes a credential.
func (r *OAuthRepository) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.cloud_credentials WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrCredentialNotFound
	}

	return nil
}

// DeleteCredentialByProvider removes credentials for a provider and user.
func (r *OAuthRepository) DeleteCredentialByProvider(ctx context.Context, userID uuid.UUID, provider string) error {
	query := `DELETE FROM philotes.cloud_credentials WHERE user_id = $1 AND provider = $2`
	_, err := r.db.ExecContext(ctx, query, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	return nil
}

// CleanupExpiredCredentials removes all expired credentials.
func (r *OAuthRepository) CleanupExpiredCredentials(ctx context.Context) (int64, error) {
	query := `DELETE FROM philotes.cloud_credentials WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired credentials: %w", err)
	}
	return result.RowsAffected()
}

// --- Helper Functions ---

func (r *OAuthRepository) scanCredential(row *sql.Row) (*models.CloudCredential, error) {
	var cred models.CloudCredential
	var deploymentID sql.NullString
	var userID sql.NullString
	var tokenExpiresAt sql.NullTime
	var refreshToken []byte

	err := row.Scan(
		&cred.ID,
		&deploymentID,
		&userID,
		&cred.Provider,
		&cred.CredentialType,
		&cred.CredentialsEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&cred.ExpiresAt,
		&cred.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan credential: %w", err)
	}

	if deploymentID.Valid {
		did, _ := uuid.Parse(deploymentID.String)
		cred.DeploymentID = &did
	}
	if userID.Valid {
		uid, _ := uuid.Parse(userID.String)
		cred.UserID = &uid
	}
	if tokenExpiresAt.Valid {
		cred.TokenExpiresAt = &tokenExpiresAt.Time
	}
	cred.RefreshTokenEncrypted = refreshToken

	return &cred, nil
}

func (r *OAuthRepository) scanCredentialFromRows(rows *sql.Rows) (*models.CloudCredential, error) {
	var cred models.CloudCredential
	var deploymentID sql.NullString
	var userID sql.NullString
	var tokenExpiresAt sql.NullTime
	var refreshToken []byte

	err := rows.Scan(
		&cred.ID,
		&deploymentID,
		&userID,
		&cred.Provider,
		&cred.CredentialType,
		&cred.CredentialsEncrypted,
		&refreshToken,
		&tokenExpiresAt,
		&cred.ExpiresAt,
		&cred.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan credential: %w", err)
	}

	if deploymentID.Valid {
		did, _ := uuid.Parse(deploymentID.String)
		cred.DeploymentID = &did
	}
	if userID.Valid {
		uid, _ := uuid.Parse(userID.String)
		cred.UserID = &uid
	}
	if tokenExpiresAt.Valid {
		cred.TokenExpiresAt = &tokenExpiresAt.Time
	}
	cred.RefreshTokenEncrypted = refreshToken

	return &cred, nil
}
