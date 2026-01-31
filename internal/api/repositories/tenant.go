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
)

// Tenant repository errors.
var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantSlugExists    = errors.New("tenant with this slug already exists")
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member already exists in tenant")
	ErrRoleNotFound        = errors.New("role not found")
	ErrRoleNameExists      = errors.New("role with this name already exists")
)

// TenantRepository handles database operations for tenants.
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// tenantRow represents a database row for a tenant.
type tenantRow struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	OwnerUserID sql.NullString
	IsActive    bool
	Settings    []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// toModel converts a database row to an API model.
func (r *tenantRow) toModel() *models.Tenant {
	tenant := &models.Tenant{
		ID:        r.ID,
		Name:      r.Name,
		Slug:      r.Slug,
		IsActive:  r.IsActive,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	if r.OwnerUserID.Valid {
		if ownerID, err := uuid.Parse(r.OwnerUserID.String); err == nil {
			tenant.OwnerUserID = &ownerID
		}
	}

	if len(r.Settings) > 0 {
		var settings map[string]interface{}
		if err := json.Unmarshal(r.Settings, &settings); err == nil {
			tenant.Settings = settings
		}
	}

	return tenant
}

// memberRow represents a database row for a tenant member.
type memberRow struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	UserID            uuid.UUID
	Role              string
	CustomPermissions []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// toModel converts a database row to an API model.
func (r *memberRow) toModel() *models.TenantMember {
	return &models.TenantMember{
		ID:                r.ID,
		TenantID:          r.TenantID,
		UserID:            r.UserID,
		Role:              models.TenantRole(r.Role),
		CustomPermissions: r.CustomPermissions,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

// Create creates a new tenant in the database.
func (r *TenantRepository) Create(ctx context.Context, name, slug string, ownerUserID *uuid.UUID, settings map[string]interface{}) (*models.Tenant, error) {
	var settingsJSON []byte
	var err error
	if settings != nil {
		settingsJSON, err = json.Marshal(settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", err)
		}
	} else {
		settingsJSON = []byte("{}")
	}

	var ownerID interface{}
	if ownerUserID != nil {
		ownerID = *ownerUserID
	}

	query := `
		INSERT INTO philotes.tenants (name, slug, owner_user_id, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, owner_user_id, is_active, settings, created_at, updated_at
	`

	var row tenantRow
	err = r.db.QueryRowContext(ctx, query, name, slug, ownerID, settingsJSON).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.OwnerUserID,
		&row.IsActive,
		&row.Settings,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTenantSlugExists
		}
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a tenant by ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	query := `
		SELECT id, name, slug, owner_user_id, is_active, settings, created_at, updated_at
		FROM philotes.tenants
		WHERE id = $1
	`

	var row tenantRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.OwnerUserID,
		&row.IsActive,
		&row.Settings,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return row.toModel(), nil
}

// GetBySlug retrieves a tenant by slug.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	query := `
		SELECT id, name, slug, owner_user_id, is_active, settings, created_at, updated_at
		FROM philotes.tenants
		WHERE slug = $1
	`

	var row tenantRow
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&row.ID,
		&row.Name,
		&row.Slug,
		&row.OwnerUserID,
		&row.IsActive,
		&row.Settings,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	return row.toModel(), nil
}

// List retrieves all tenants.
func (r *TenantRepository) List(ctx context.Context) ([]models.Tenant, error) {
	query := `
		SELECT id, name, slug, owner_user_id, is_active, settings, created_at, updated_at
		FROM philotes.tenants
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var row tenantRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Slug,
			&row.OwnerUserID,
			&row.IsActive,
			&row.Settings,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant row: %w", err)
		}
		tenants = append(tenants, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tenants: %w", err)
	}

	return tenants, nil
}

// ListByUser retrieves all tenants that a user is a member of.
func (r *TenantRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Tenant, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.owner_user_id, t.is_active, t.settings, t.created_at, t.updated_at
		FROM philotes.tenants t
		INNER JOIN philotes.tenant_members tm ON t.id = tm.tenant_id
		WHERE tm.user_id = $1 AND t.is_active = true
		ORDER BY t.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants by user: %w", err)
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var row tenantRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Slug,
			&row.OwnerUserID,
			&row.IsActive,
			&row.Settings,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant row: %w", err)
		}
		tenants = append(tenants, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tenants: %w", err)
	}

	return tenants, nil
}

// Update updates a tenant in the database.
func (r *TenantRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	// First check if tenant exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.tenants SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Slug != nil {
		query += fmt.Sprintf(", slug = $%d", argIdx)
		args = append(args, *req.Slug)
		argIdx++
	}
	if req.IsActive != nil {
		query += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Settings != nil {
		settingsJSON, marshalErr := json.Marshal(*req.Settings)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal settings: %w", marshalErr)
		}
		query += fmt.Sprintf(", settings = $%d", argIdx)
		args = append(args, settingsJSON)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTenantSlugExists
		}
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Delete deletes a tenant from the database.
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.tenants WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// --- Member Operations ---

// AddMember adds a user as a member of a tenant.
func (r *TenantRepository) AddMember(ctx context.Context, tenantID, userID uuid.UUID, role models.TenantRole, customPermissions []string) (*models.TenantMember, error) {
	if customPermissions == nil {
		customPermissions = []string{}
	}

	query := `
		INSERT INTO philotes.tenant_members (tenant_id, user_id, role, custom_permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, user_id, role, custom_permissions, created_at, updated_at
	`

	var row memberRow
	err := r.db.QueryRowContext(ctx, query, tenantID, userID, role, pq.Array(customPermissions)).Scan(
		&row.ID,
		&row.TenantID,
		&row.UserID,
		&row.Role,
		pq.Array(&row.CustomPermissions),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrMemberAlreadyExists
		}
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return row.toModel(), nil
}

// GetMember retrieves a specific member of a tenant.
func (r *TenantRepository) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error) {
	query := `
		SELECT id, tenant_id, user_id, role, custom_permissions, created_at, updated_at
		FROM philotes.tenant_members
		WHERE tenant_id = $1 AND user_id = $2
	`

	var row memberRow
	err := r.db.QueryRowContext(ctx, query, tenantID, userID).Scan(
		&row.ID,
		&row.TenantID,
		&row.UserID,
		&row.Role,
		pq.Array(&row.CustomPermissions),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return row.toModel(), nil
}

// ListMembers retrieves all members of a tenant.
func (r *TenantRepository) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantMember, error) {
	query := `
		SELECT tm.id, tm.tenant_id, tm.user_id, tm.role, tm.custom_permissions, tm.created_at, tm.updated_at,
		       u.id, u.email, u.name, u.role, u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM philotes.tenant_members tm
		INNER JOIN philotes.users u ON tm.user_id = u.id
		WHERE tm.tenant_id = $1
		ORDER BY tm.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []models.TenantMember
	for rows.Next() {
		var row memberRow
		var user models.User
		var userName sql.NullString
		var userLastLogin sql.NullTime

		err := rows.Scan(
			&row.ID,
			&row.TenantID,
			&row.UserID,
			&row.Role,
			pq.Array(&row.CustomPermissions),
			&row.CreatedAt,
			&row.UpdatedAt,
			&user.ID,
			&user.Email,
			&userName,
			&user.Role,
			&user.IsActive,
			&userLastLogin,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member row: %w", err)
		}

		user.Name = userName.String
		if userLastLogin.Valid {
			user.LastLoginAt = &userLastLogin.Time
		}

		member := row.toModel()
		member.User = &user
		members = append(members, *member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate members: %w", err)
	}

	return members, nil
}

// UpdateMember updates a member's role and permissions.
func (r *TenantRepository) UpdateMember(ctx context.Context, tenantID, userID uuid.UUID, req *models.UpdateMemberRequest) (*models.TenantMember, error) {
	// First check if member exists
	_, err := r.GetMember(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.tenant_members SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Role != nil {
		query += fmt.Sprintf(", role = $%d", argIdx)
		args = append(args, *req.Role)
		argIdx++
	}
	if req.CustomPermissions != nil {
		query += fmt.Sprintf(", custom_permissions = $%d", argIdx)
		args = append(args, pq.Array(*req.CustomPermissions))
		argIdx++
	}

	query += fmt.Sprintf(" WHERE tenant_id = $%d AND user_id = $%d", argIdx, argIdx+1)
	args = append(args, tenantID, userID)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	return r.GetMember(ctx, tenantID, userID)
}

// RemoveMember removes a user from a tenant.
func (r *TenantRepository) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	query := `DELETE FROM philotes.tenant_members WHERE tenant_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}

// IsMember checks if a user is a member of a tenant.
func (r *TenantRepository) IsMember(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM philotes.tenant_members WHERE tenant_id = $1 AND user_id = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return exists, nil
}

// GetMemberRole gets a user's role in a tenant.
func (r *TenantRepository) GetMemberRole(ctx context.Context, tenantID, userID uuid.UUID) (models.TenantRole, error) {
	query := `SELECT role FROM philotes.tenant_members WHERE tenant_id = $1 AND user_id = $2`

	var role string
	err := r.db.QueryRowContext(ctx, query, tenantID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrMemberNotFound
		}
		return "", fmt.Errorf("failed to get member role: %w", err)
	}

	return models.TenantRole(role), nil
}

// CreateWithOwner creates a new tenant and adds the owner as an admin member in a single transaction.
func (r *TenantRepository) CreateWithOwner(ctx context.Context, name, slug string, ownerUserID uuid.UUID, settings map[string]interface{}) (*models.Tenant, *models.TenantMember, error) {
	var settingsJSON []byte
	var err error
	if settings != nil {
		settingsJSON, err = json.Marshal(settings)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal settings: %w", err)
		}
	} else {
		settingsJSON = []byte("{}")
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			// nolint:errcheck // Rollback error is intentionally ignored on failure path
			_ = tx.Rollback()
		}
	}()

	// Create tenant
	tenantQuery := `
		INSERT INTO philotes.tenants (name, slug, owner_user_id, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, slug, owner_user_id, is_active, settings, created_at, updated_at
	`

	var tenantRowData tenantRow
	err = tx.QueryRowContext(ctx, tenantQuery, name, slug, ownerUserID, settingsJSON).Scan(
		&tenantRowData.ID,
		&tenantRowData.Name,
		&tenantRowData.Slug,
		&tenantRowData.OwnerUserID,
		&tenantRowData.IsActive,
		&tenantRowData.Settings,
		&tenantRowData.CreatedAt,
		&tenantRowData.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, nil, ErrTenantSlugExists
		}
		return nil, nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Add owner as admin member
	memberQuery := `
		INSERT INTO philotes.tenant_members (tenant_id, user_id, role, custom_permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, user_id, role, custom_permissions, created_at, updated_at
	`

	var memberRowData memberRow
	err = tx.QueryRowContext(ctx, memberQuery, tenantRowData.ID, ownerUserID, models.TenantRoleAdmin, pq.Array([]string{})).Scan(
		&memberRowData.ID,
		&memberRowData.TenantID,
		&memberRowData.UserID,
		&memberRowData.Role,
		pq.Array(&memberRowData.CustomPermissions),
		&memberRowData.CreatedAt,
		&memberRowData.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add owner as member: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return tenantRowData.toModel(), memberRowData.toModel(), nil
}
