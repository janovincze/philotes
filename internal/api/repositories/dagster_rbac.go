// Package repositories provides data access layer for API resources.
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/janovincze/philotes/internal/api/models"
)

// Dagster RBAC repository errors.
var (
	ErrDagsterRoleNotFound       = errors.New("dagster role assignment not found")
	ErrDagsterPermissionNotFound = errors.New("dagster permission not found")
	ErrDagsterRoleExists         = errors.New("dagster role assignment already exists")
	ErrDagsterPermissionExists   = errors.New("dagster permission already exists")
)

// DagsterRBACRepository handles database operations for Dagster RBAC.
type DagsterRBACRepository struct {
	db *sql.DB
}

// NewDagsterRBACRepository creates a new DagsterRBACRepository.
func NewDagsterRBACRepository(db *sql.DB) *DagsterRBACRepository {
	return &DagsterRBACRepository{db: db}
}

// GetUserRoles returns all Dagster role assignments for a user.
func (r *DagsterRBACRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.DagsterRoleAssignment, error) {
	query := `
		SELECT id, user_id, role, resource_pattern, created_at, updated_at
		FROM dagster_role_assignments
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.DagsterRoleAssignment
	for rows.Next() {
		var role models.DagsterRoleAssignment
		var resourcePattern sql.NullString

		if err := rows.Scan(
			&role.ID,
			&role.UserID,
			&role.Role,
			&resourcePattern,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if resourcePattern.Valid {
			role.ResourcePattern = resourcePattern.String
		}

		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetUserPermissions returns all custom Dagster permissions for a user.
func (r *DagsterRBACRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.DagsterPermission, error) {
	query := `
		SELECT id, user_id, permission, created_at
		FROM dagster_permissions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.DagsterPermission
	for rows.Next() {
		var perm models.DagsterPermission
		if err := rows.Scan(
			&perm.ID,
			&perm.UserID,
			&perm.Permission,
			&perm.CreatedAt,
		); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	return permissions, rows.Err()
}

// AssignRole assigns a Dagster role to a user.
func (r *DagsterRBACRepository) AssignRole(ctx context.Context, userID uuid.UUID, role string, resourcePattern string) (*models.DagsterRoleAssignment, error) {
	query := `
		INSERT INTO dagster_role_assignments (user_id, role, resource_pattern)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (user_id, role, COALESCE(resource_pattern, '')) DO UPDATE
		SET updated_at = NOW()
		RETURNING id, user_id, role, resource_pattern, created_at, updated_at
	`

	var assignment models.DagsterRoleAssignment
	var resourcePatternNull sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID, role, resourcePattern).Scan(
		&assignment.ID,
		&assignment.UserID,
		&assignment.Role,
		&resourcePatternNull,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if resourcePatternNull.Valid {
		assignment.ResourcePattern = resourcePatternNull.String
	}

	return &assignment, nil
}

// RemoveRole removes a Dagster role assignment.
func (r *DagsterRBACRepository) RemoveRole(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dagster_role_assignments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrDagsterRoleNotFound
	}

	return nil
}

// AddPermission adds a custom Dagster permission for a user.
func (r *DagsterRBACRepository) AddPermission(ctx context.Context, userID uuid.UUID, permission string) (*models.DagsterPermission, error) {
	query := `
		INSERT INTO dagster_permissions (user_id, permission)
		VALUES ($1, $2)
		ON CONFLICT (user_id, permission) DO NOTHING
		RETURNING id, user_id, permission, created_at
	`

	var perm models.DagsterPermission
	err := r.db.QueryRowContext(ctx, query, userID, permission).Scan(
		&perm.ID,
		&perm.UserID,
		&perm.Permission,
		&perm.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDagsterPermissionExists
		}
		return nil, err
	}

	return &perm, nil
}

// RemovePermission removes a custom Dagster permission.
func (r *DagsterRBACRepository) RemovePermission(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dagster_permissions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrDagsterPermissionNotFound
	}

	return nil
}

// GetAllEffectivePermissions returns all effective Dagster permissions for a user.
// This includes permissions from assigned roles and custom permissions.
func (r *DagsterRBACRepository) GetAllEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// Get roles and expand them to permissions
	roles, err := r.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	permSet := make(map[string]bool)

	// Expand roles to permissions
	for _, role := range roles {
		rolePerms := expandRolePermissions(role.Role)
		for _, p := range rolePerms {
			// Apply resource pattern if specified
			if role.ResourcePattern != "" {
				// For now, we just add the role permissions as-is
				// Resource patterns are for scoped roles (future enhancement)
			}
			permSet[p] = true
		}
	}

	// Get custom permissions
	customPerms, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, p := range customPerms {
		permSet[p.Permission] = true
	}

	// Convert to slice
	permissions := make([]string, 0, len(permSet))
	for p := range permSet {
		permissions = append(permissions, p)
	}

	return permissions, nil
}

// ListAllRoleAssignments returns all Dagster role assignments (for admin view).
func (r *DagsterRBACRepository) ListAllRoleAssignments(ctx context.Context, limit, offset int) ([]models.DagsterRoleAssignmentWithUser, int, error) {
	countQuery := `SELECT COUNT(*) FROM dagster_role_assignments`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			dra.id, dra.user_id, dra.role, dra.resource_pattern, dra.created_at, dra.updated_at,
			u.email, u.name
		FROM dagster_role_assignments dra
		JOIN users u ON u.id = dra.user_id
		ORDER BY dra.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var assignments []models.DagsterRoleAssignmentWithUser
	for rows.Next() {
		var a models.DagsterRoleAssignmentWithUser
		var resourcePattern sql.NullString
		var userName sql.NullString

		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.Role,
			&resourcePattern,
			&a.CreatedAt,
			&a.UpdatedAt,
			&a.UserEmail,
			&userName,
		); err != nil {
			return nil, 0, err
		}

		if resourcePattern.Valid {
			a.ResourcePattern = resourcePattern.String
		}
		if userName.Valid {
			a.UserName = userName.String
		}

		assignments = append(assignments, a)
	}

	return assignments, total, rows.Err()
}

// BulkAssignRole assigns a role to multiple users.
func (r *DagsterRBACRepository) BulkAssignRole(ctx context.Context, userIDs []uuid.UUID, role string) error {
	query := `
		INSERT INTO dagster_role_assignments (user_id, role)
		SELECT unnest($1::uuid[]), $2
		ON CONFLICT (user_id, role, COALESCE(resource_pattern, '')) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, pq.Array(userIDs), role)
	return err
}

// expandRolePermissions returns the permissions for a given role.
func expandRolePermissions(role string) []string {
	rolePermissions := map[string][]string{
		"dagster-viewer": {
			"dagster:jobs:*:view",
			"dagster:assets:*:view",
			"dagster:schedules:*:view",
			"dagster:sensors:*:view",
			"dagster:runs:*:view",
		},
		"dagster-operator": {
			"dagster:jobs:*:view",
			"dagster:jobs:*:execute",
			"dagster:assets:*:view",
			"dagster:assets:*:materialize",
			"dagster:schedules:*:view",
			"dagster:sensors:*:view",
			"dagster:runs:*:view",
		},
		"dagster-admin": {
			"dagster:*:*:*",
		},
	}

	if perms, ok := rolePermissions[role]; ok {
		return perms
	}
	return nil
}
