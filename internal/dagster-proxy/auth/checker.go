// Package auth provides authentication and authorization for the Dagster proxy.
package auth

import (
	"log/slog"
)

// PermissionChecker checks permissions for Dagster resources.
type PermissionChecker struct {
	logger *slog.Logger
}

// NewPermissionChecker creates a new permission checker.
func NewPermissionChecker(logger *slog.Logger) *PermissionChecker {
	return &PermissionChecker{
		logger: logger,
	}
}

// HasPermission checks if the auth context has permission for a resource.
func (c *PermissionChecker) HasPermission(authCtx *Context, resource ResourceRef) bool {
	if authCtx == nil {
		return false
	}

	// Get all Dagster permissions
	permissions := authCtx.GetDagsterPermissions()

	// Check if any permission grants access
	result := MatchResourcePermission(
		permissions,
		resource.Type,
		resource.Name,
		resource.Action,
	)

	if c.logger != nil {
		c.logger.Debug("permission check",
			"user_id", authCtx.UserID,
			"resource_type", resource.Type,
			"resource_name", resource.Name,
			"action", resource.Action,
			"allowed", result,
		)
	}

	return result
}

// CanViewResources returns a function that checks if a resource is viewable.
// This is useful for filtering large lists of resources.
func (c *PermissionChecker) CanViewResources(authCtx *Context, resourceType ResourceType) func(name string) bool {
	if authCtx == nil {
		return func(name string) bool { return false }
	}

	permissions := authCtx.GetDagsterPermissions()

	// Check if user has wildcard view access
	hasWildcardAccess := MatchResourcePermission(permissions, resourceType, "*", ActionView)
	if hasWildcardAccess {
		return func(name string) bool { return true }
	}

	// Return a function that checks each resource individually
	return func(name string) bool {
		return MatchResourcePermission(permissions, resourceType, name, ActionView)
	}
}

// CheckMutation checks if a mutation is allowed based on the operation.
func (c *PermissionChecker) CheckMutation(authCtx *Context, operationName string, resourceType ResourceType, resourceName string) error {
	if authCtx == nil {
		return &PermissionError{Message: "Authentication required"}
	}

	// Determine action based on operation
	action := c.operationToAction(operationName, resourceType)

	// Check permission
	if !c.HasPermission(authCtx, ResourceRef{
		Type:   resourceType,
		Name:   resourceName,
		Action: action,
	}) {
		return &PermissionError{
			Message:      "Insufficient permissions",
			ResourceType: string(resourceType),
			ResourceName: resourceName,
			Action:       string(action),
		}
	}

	return nil
}

// operationToAction maps GraphQL operation names to action types.
func (c *PermissionChecker) operationToAction(operationName string, resourceType ResourceType) ActionType {
	switch operationName {
	// Job execution
	case "launchRun", "launchPipelineExecution", "launchBackfill", "LaunchRunMutation":
		return ActionExecute

	// Asset materialization
	case "launchPartitionBackfill", "LaunchAssetBackfill":
		if resourceType == ResourceTypeAsset {
			return ActionMaterialize
		}
		return ActionExecute

	// Schedule/sensor control
	case "startSchedule", "stopSchedule", "StartScheduleMutation", "StopScheduleMutation":
		return ActionControl
	case "startSensor", "stopSensor", "StartSensorMutation", "StopSensorMutation":
		return ActionControl
	case "resetSensor", "ResetSensorMutation":
		return ActionControl

	// Run termination
	case "terminateRun", "terminateRuns", "deletePipelineRun", "TerminateRunMutation":
		return ActionTerminate

	// Asset operations
	case "wipeAssets", "reportRunlessAssetEvents":
		return ActionMaterialize

	default:
		// Default to execute for unknown mutations
		return ActionExecute
	}
}

// GetAllowedResources returns all resources of a type that the user can view.
// This is used for response filtering.
func (c *PermissionChecker) GetAllowedResources(authCtx *Context, resourceType ResourceType, allResources []string) []string {
	if authCtx == nil {
		return nil
	}

	permissions := authCtx.GetDagsterPermissions()

	// Check for wildcard access
	if MatchResourcePermission(permissions, resourceType, "*", ActionView) {
		return allResources
	}

	// Filter to allowed resources
	return FilterResourcesByPermission(permissions, allResources, resourceType, ActionView)
}

// IsAdmin checks if the auth context has admin permissions.
func (c *PermissionChecker) IsAdmin(authCtx *Context) bool {
	if authCtx == nil {
		return false
	}

	for _, perm := range authCtx.Permissions {
		// Check for full admin wildcard
		if perm == "dagster:*:*:*" {
			return true
		}
	}

	return false
}

// GetEffectivePermissions returns the effective permissions for an auth context,
// including expanded role permissions.
func (c *PermissionChecker) GetEffectivePermissions(authCtx *Context) []string {
	if authCtx == nil {
		return nil
	}

	// Start with explicit permissions
	permissions := authCtx.GetDagsterPermissions()

	// Add role-based permissions if a Dagster role is assigned
	rolePermSets := [][]string{permissions}

	// Check for Dagster roles in the user's permissions
	for _, perm := range authCtx.Permissions {
		if rolePerms := ExpandRoleToPermissions(perm); rolePerms != nil {
			rolePermSets = append(rolePermSets, rolePerms)
		}
	}

	return MergePermissions(rolePermSets...)
}
