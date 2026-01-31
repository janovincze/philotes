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

// TenantRoleRepository handles database operations for custom tenant roles.
type TenantRoleRepository struct {
	db *sql.DB
}

// NewTenantRoleRepository creates a new TenantRoleRepository.
func NewTenantRoleRepository(db *sql.DB) *TenantRoleRepository {
	return &TenantRoleRepository{db: db}
}

// roleRow represents a database row for a tenant custom role.
type roleRow struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description sql.NullString
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// toModel converts a database row to an API model.
func (r *roleRow) toModel() *models.TenantCustomRole {
	return &models.TenantCustomRole{
		ID:          r.ID,
		TenantID:    r.TenantID,
		Name:        r.Name,
		Description: r.Description.String,
		Permissions: r.Permissions,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// Create creates a new custom role in the database.
func (r *TenantRoleRepository) Create(ctx context.Context, tenantID uuid.UUID, name, description string, permissions []string) (*models.TenantCustomRole, error) {
	if permissions == nil {
		permissions = []string{}
	}

	query := `
		INSERT INTO philotes.tenant_roles (tenant_id, name, description, permissions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, name, description, permissions, created_at, updated_at
	`

	var row roleRow
	err := r.db.QueryRowContext(ctx, query, tenantID, name, nullString(description), pq.Array(permissions)).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.Description,
		pq.Array(&row.Permissions),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrRoleNameExists
		}
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return row.toModel(), nil
}

// GetByID retrieves a custom role by ID.
func (r *TenantRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TenantCustomRole, error) {
	query := `
		SELECT id, tenant_id, name, description, permissions, created_at, updated_at
		FROM philotes.tenant_roles
		WHERE id = $1
	`

	var row roleRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.Description,
		pq.Array(&row.Permissions),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return row.toModel(), nil
}

// GetByName retrieves a custom role by tenant ID and name.
func (r *TenantRoleRepository) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.TenantCustomRole, error) {
	query := `
		SELECT id, tenant_id, name, description, permissions, created_at, updated_at
		FROM philotes.tenant_roles
		WHERE tenant_id = $1 AND name = $2
	`

	var row roleRow
	err := r.db.QueryRowContext(ctx, query, tenantID, name).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.Description,
		pq.Array(&row.Permissions),
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return row.toModel(), nil
}

// ListByTenant retrieves all custom roles for a tenant.
func (r *TenantRoleRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.TenantCustomRole, error) {
	query := `
		SELECT id, tenant_id, name, description, permissions, created_at, updated_at
		FROM philotes.tenant_roles
		WHERE tenant_id = $1
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []models.TenantCustomRole
	for rows.Next() {
		var row roleRow
		err := rows.Scan(
			&row.ID,
			&row.TenantID,
			&row.Name,
			&row.Description,
			pq.Array(&row.Permissions),
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, *row.toModel())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate roles: %w", err)
	}

	return roles, nil
}

// Update updates a custom role in the database.
func (r *TenantRoleRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateCustomRoleRequest) (*models.TenantCustomRole, error) {
	// Verify role exists before updating (returns ErrRoleNotFound if not)
	if _, err := r.GetByID(ctx, id); err != nil {
		return nil, err
	}

	// Build dynamic update query
	query := `UPDATE philotes.tenant_roles SET updated_at = NOW()`
	args := []any{}
	argIdx := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, nullString(*req.Description))
		argIdx++
	}
	if req.Permissions != nil {
		query += fmt.Sprintf(", permissions = $%d", argIdx)
		args = append(args, pq.Array(*req.Permissions))
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrRoleNameExists
		}
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Delete deletes a custom role from the database.
func (r *TenantRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM philotes.tenant_roles WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrRoleNotFound
	}

	return nil
}

// ExistsByName checks if a role with the given name exists in the tenant.
func (r *TenantRoleRepository) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM philotes.tenant_roles WHERE tenant_id = $1 AND name = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check role existence: %w", err)
	}

	return exists, nil
}
