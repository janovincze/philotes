// Package services provides business logic for API resources.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/janovincze/philotes/internal/api/models"
	"github.com/janovincze/philotes/internal/api/repositories"
	"github.com/janovincze/philotes/internal/config"
)

// Tenant service errors.
var (
	ErrCannotRemoveOwner     = errors.New("cannot remove tenant owner")
	ErrCannotDemoteLastAdmin = errors.New("cannot demote the last admin")
	ErrCannotRemoveLastAdmin = errors.New("cannot remove the last admin")
)

// TenantService provides tenant management business logic.
type TenantService struct {
	tenantRepo *repositories.TenantRepository
	roleRepo   *repositories.TenantRoleRepository
	userRepo   *repositories.UserRepository
	auditRepo  *repositories.AuditRepository
	cfg        *config.MultiTenancyConfig
	logger     *slog.Logger
}

// NewTenantService creates a new TenantService.
func NewTenantService(
	tenantRepo *repositories.TenantRepository,
	roleRepo *repositories.TenantRoleRepository,
	userRepo *repositories.UserRepository,
	auditRepo *repositories.AuditRepository,
	cfg *config.MultiTenancyConfig,
	logger *slog.Logger,
) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
		cfg:        cfg,
		logger:     logger.With("component", "tenant-service"),
	}
}

// --- Tenant Operations ---

// Create creates a new tenant.
func (s *TenantService) Create(ctx context.Context, req *models.CreateTenantRequest, creatorUserID uuid.UUID) (*models.Tenant, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Create tenant with creator as owner in a single transaction
	tenant, _, err := s.tenantRepo.CreateWithOwner(ctx, req.Name, req.Slug, creatorUserID, req.Settings)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantSlugExists) {
			return nil, &ConflictError{Message: "tenant with slug '" + req.Slug + "' already exists"}
		}
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	s.logger.Info("tenant created", "tenant_id", tenant.ID, "slug", tenant.Slug, "owner_id", creatorUserID)

	return tenant, nil
}

// GetByID retrieves a tenant by ID.
func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return tenant, nil
}

// GetBySlug retrieves a tenant by slug.
func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	tenant, err := s.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: slug}
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}
	return tenant, nil
}

// List retrieves all tenants (admin only).
func (s *TenantService) List(ctx context.Context) ([]models.Tenant, error) {
	tenants, err := s.tenantRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	return tenants, nil
}

// ListByUser retrieves all tenants that a user is a member of.
func (s *TenantService) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Tenant, error) {
	tenants, err := s.tenantRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants by user: %w", err)
	}
	return tenants, nil
}

// Update updates a tenant.
func (s *TenantService) Update(ctx context.Context, id uuid.UUID, req *models.UpdateTenantRequest) (*models.Tenant, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	tenant, err := s.tenantRepo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrTenantSlugExists) {
			return nil, &ConflictError{Message: "tenant with slug '" + *req.Slug + "' already exists"}
		}
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	s.logger.Info("tenant updated", "tenant_id", id)

	return tenant, nil
}

// Delete deletes a tenant.
func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.tenantRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return &NotFoundError{Resource: "tenant", ID: id.String()}
		}
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	s.logger.Info("tenant deleted", "tenant_id", id)

	return nil
}

// --- Member Operations ---

// AddMember adds a user as a member of a tenant.
func (s *TenantService) AddMember(ctx context.Context, tenantID uuid.UUID, req *models.AddMemberRequest) (*models.TenantMember, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Verify tenant exists
	_, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: tenantID.String()}
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Verify user exists
	_, err = s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, &NotFoundError{Resource: "user", ID: req.UserID.String()}
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	member, err := s.tenantRepo.AddMember(ctx, tenantID, req.UserID, req.Role, req.CustomPermissions)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberAlreadyExists) {
			return nil, &ConflictError{Message: "user is already a member of this tenant"}
		}
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	s.logger.Info("member added to tenant", "tenant_id", tenantID, "user_id", req.UserID, "role", req.Role)

	return member, nil
}

// GetMember retrieves a specific member of a tenant.
func (s *TenantService) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*models.TenantMember, error) {
	member, err := s.tenantRepo.GetMember(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return nil, &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return member, nil
}

// ListMembers retrieves all members of a tenant.
func (s *TenantService) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]models.TenantMember, error) {
	// Verify tenant exists
	_, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: tenantID.String()}
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	members, err := s.tenantRepo.ListMembers(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	return members, nil
}

// UpdateMember updates a member's role and permissions.
func (s *TenantService) UpdateMember(ctx context.Context, tenantID, userID uuid.UUID, req *models.UpdateMemberRequest) (*models.TenantMember, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Check if demoting the last admin
	if req.Role != nil && *req.Role != models.TenantRoleAdmin {
		currentMember, err := s.tenantRepo.GetMember(ctx, tenantID, userID)
		if err != nil {
			if errors.Is(err, repositories.ErrMemberNotFound) {
				return nil, &NotFoundError{Resource: "member", ID: userID.String()}
			}
			return nil, fmt.Errorf("failed to get member: %w", err)
		}

		if currentMember.Role == models.TenantRoleAdmin {
			// Count admins in tenant
			members, listErr := s.tenantRepo.ListMembers(ctx, tenantID)
			if listErr != nil {
				return nil, fmt.Errorf("failed to list members: %w", listErr)
			}

			adminCount := 0
			for i := range members {
				if members[i].Role == models.TenantRoleAdmin {
					adminCount++
				}
			}

			if adminCount <= 1 {
				return nil, ErrCannotDemoteLastAdmin
			}
		}
	}

	member, err := s.tenantRepo.UpdateMember(ctx, tenantID, userID, req)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return nil, &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	s.logger.Info("member updated", "tenant_id", tenantID, "user_id", userID)

	return member, nil
}

// RemoveMember removes a user from a tenant.
func (s *TenantService) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	// Get tenant to check owner
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return &NotFoundError{Resource: "tenant", ID: tenantID.String()}
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Cannot remove the owner
	if tenant.OwnerUserID != nil && *tenant.OwnerUserID == userID {
		return ErrCannotRemoveOwner
	}

	// Check if removing the last admin
	currentMember, err := s.tenantRepo.GetMember(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return fmt.Errorf("failed to get member: %w", err)
	}

	if currentMember.Role == models.TenantRoleAdmin {
		members, listErr := s.tenantRepo.ListMembers(ctx, tenantID)
		if listErr != nil {
			return fmt.Errorf("failed to list members: %w", listErr)
		}

		adminCount := 0
		for i := range members {
			if members[i].Role == models.TenantRoleAdmin {
				adminCount++
			}
		}

		if adminCount <= 1 {
			return ErrCannotRemoveLastAdmin
		}
	}

	err = s.tenantRepo.RemoveMember(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.logger.Info("member removed from tenant", "tenant_id", tenantID, "user_id", userID)

	return nil
}

// IsMember checks if a user is a member of a tenant.
func (s *TenantService) IsMember(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	return s.tenantRepo.IsMember(ctx, tenantID, userID)
}

// GetMemberRole gets a user's role in a tenant.
func (s *TenantService) GetMemberRole(ctx context.Context, tenantID, userID uuid.UUID) (models.TenantRole, error) {
	role, err := s.tenantRepo.GetMemberRole(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return "", &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return "", fmt.Errorf("failed to get member role: %w", err)
	}
	return role, nil
}

// GetMemberPermissions gets a user's effective permissions in a tenant.
func (s *TenantService) GetMemberPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	member, err := s.tenantRepo.GetMember(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrMemberNotFound) {
			return nil, &NotFoundError{Resource: "member", ID: userID.String()}
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	// Get role-based permissions
	permissions := make(map[string]bool)
	if rolePerms, ok := models.TenantRolePermissions[member.Role]; ok {
		for _, p := range rolePerms {
			permissions[p] = true
		}
	}

	// Add custom permissions
	for _, p := range member.CustomPermissions {
		permissions[p] = true
	}

	// Convert to slice
	result := make([]string, 0, len(permissions))
	for p := range permissions {
		result = append(result, p)
	}

	return result, nil
}

// --- Custom Role Operations ---

// CreateCustomRole creates a new custom role in a tenant.
func (s *TenantService) CreateCustomRole(ctx context.Context, tenantID uuid.UUID, req *models.CreateCustomRoleRequest) (*models.TenantCustomRole, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	// Verify tenant exists
	_, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: tenantID.String()}
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	role, err := s.roleRepo.Create(ctx, tenantID, req.Name, req.Description, req.Permissions)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleNameExists) {
			return nil, &ConflictError{Message: "role with name '" + req.Name + "' already exists in this tenant"}
		}
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	s.logger.Info("custom role created", "tenant_id", tenantID, "role_id", role.ID, "name", role.Name)

	return role, nil
}

// GetCustomRole retrieves a custom role by ID.
func (s *TenantService) GetCustomRole(ctx context.Context, id uuid.UUID) (*models.TenantCustomRole, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleNotFound) {
			return nil, &NotFoundError{Resource: "role", ID: id.String()}
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return role, nil
}

// ListCustomRoles retrieves all custom roles for a tenant.
func (s *TenantService) ListCustomRoles(ctx context.Context, tenantID uuid.UUID) ([]models.TenantCustomRole, error) {
	// Verify tenant exists
	_, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repositories.ErrTenantNotFound) {
			return nil, &NotFoundError{Resource: "tenant", ID: tenantID.String()}
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	roles, err := s.roleRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

// UpdateCustomRole updates a custom role.
func (s *TenantService) UpdateCustomRole(ctx context.Context, id uuid.UUID, req *models.UpdateCustomRoleRequest) (*models.TenantCustomRole, error) {
	// Validate request
	if fieldErrors := req.Validate(); len(fieldErrors) > 0 {
		return nil, &ValidationError{Errors: fieldErrors}
	}

	role, err := s.roleRepo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleNotFound) {
			return nil, &NotFoundError{Resource: "role", ID: id.String()}
		}
		if errors.Is(err, repositories.ErrRoleNameExists) {
			nameMsg := "the specified name"
			if req.Name != nil {
				nameMsg = "'" + *req.Name + "'"
			}
			return nil, &ConflictError{Message: "role with name " + nameMsg + " already exists in this tenant"}
		}
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	s.logger.Info("custom role updated", "role_id", id)

	return role, nil
}

// DeleteCustomRole deletes a custom role.
func (s *TenantService) DeleteCustomRole(ctx context.Context, id uuid.UUID) error {
	err := s.roleRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repositories.ErrRoleNotFound) {
			return &NotFoundError{Resource: "role", ID: id.String()}
		}
		return fmt.Errorf("failed to delete role: %w", err)
	}

	s.logger.Info("custom role deleted", "role_id", id)

	return nil
}

// GetDefaultTenant returns the default system tenant.
func (s *TenantService) GetDefaultTenant(ctx context.Context) (*models.Tenant, error) {
	defaultID := models.GetDefaultTenantUUID()
	return s.GetByID(ctx, defaultID)
}
