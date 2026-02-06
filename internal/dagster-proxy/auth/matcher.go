// Package auth provides authentication and authorization for the Dagster proxy.
package auth

import (
	"strings"
)

// MatchPermission checks if a pattern matches a required permission.
// Pattern can include wildcards:
//   - * matches any single segment
//   - ** matches any number of segments (not implemented yet)
//
// Examples:
//
//	MatchPermission("dagster:jobs:*:view", "dagster:jobs:orders:view") -> true
//	MatchPermission("dagster:jobs:etl_*:execute", "dagster:jobs:etl_orders:execute") -> true
//	MatchPermission("dagster:*:*:*", "dagster:jobs:orders:execute") -> true
func MatchPermission(pattern, required string) bool {
	patternParts := splitPermission(pattern)
	requiredParts := splitPermission(required)

	// Must have same number of parts
	if len(patternParts) != len(requiredParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if !matchSegment(patternPart, requiredParts[i]) {
			return false
		}
	}

	return true
}

// matchSegment checks if a pattern segment matches a required segment.
func matchSegment(pattern, required string) bool {
	// Exact wildcard matches everything
	if pattern == "*" {
		return true
	}

	// No wildcards - exact match required
	if !strings.Contains(pattern, "*") {
		return pattern == required
	}

	// Handle prefix wildcards: "etl_*" matches "etl_orders"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(required, prefix)
	}

	// Handle suffix wildcards: "*_orders" matches "etl_orders"
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(required, suffix)
	}

	// Handle middle wildcards: "etl_*_daily" matches "etl_orders_daily"
	parts := strings.SplitN(pattern, "*", 2)
	if len(parts) == 2 {
		return strings.HasPrefix(required, parts[0]) && strings.HasSuffix(required, parts[1])
	}

	return pattern == required
}

// MatchResourcePermission checks if user permissions allow access to a resource.
func MatchResourcePermission(permissions []string, resourceType ResourceType, resourceName string, action ActionType) bool {
	required := BuildPermission(resourceType, resourceName, action)

	for _, perm := range permissions {
		if MatchPermission(perm, required) {
			return true
		}
	}

	return false
}

// FilterResourcesByPermission filters a list of resources based on permissions.
// Returns only resources the user is allowed to view.
func FilterResourcesByPermission(
	permissions []string,
	resources []string,
	resourceType ResourceType,
	action ActionType,
) []string {
	var allowed []string

	for _, resourceName := range resources {
		if MatchResourcePermission(permissions, resourceType, resourceName, action) {
			allowed = append(allowed, resourceName)
		}
	}

	return allowed
}

// CanViewResource checks if the user can view a specific resource.
func CanViewResource(permissions []string, resourceType ResourceType, resourceName string) bool {
	return MatchResourcePermission(permissions, resourceType, resourceName, ActionView)
}

// CanExecuteJob checks if the user can execute a specific job.
func CanExecuteJob(permissions []string, jobName string) bool {
	return MatchResourcePermission(permissions, ResourceTypeJob, jobName, ActionExecute)
}

// CanMaterializeAsset checks if the user can materialize a specific asset.
func CanMaterializeAsset(permissions []string, assetKey string) bool {
	return MatchResourcePermission(permissions, ResourceTypeAsset, assetKey, ActionMaterialize)
}

// CanControlSchedule checks if the user can control a specific schedule.
func CanControlSchedule(permissions []string, scheduleName string) bool {
	return MatchResourcePermission(permissions, ResourceTypeSchedule, scheduleName, ActionControl)
}

// CanControlSensor checks if the user can control a specific sensor.
func CanControlSensor(permissions []string, sensorName string) bool {
	return MatchResourcePermission(permissions, ResourceTypeSensor, sensorName, ActionControl)
}

// CanTerminateRun checks if the user can terminate runs.
func CanTerminateRun(permissions []string) bool {
	// Run termination is a global permission, not per-run
	return MatchResourcePermission(permissions, ResourceTypeRun, "*", ActionTerminate)
}

// ExpandRoleToPermissions expands a role name to its permissions.
func ExpandRoleToPermissions(roleName string) []string {
	if perms, ok := RolePermissions[roleName]; ok {
		return perms
	}
	return nil
}

// MergePermissions merges multiple permission sets, removing duplicates.
func MergePermissions(permSets ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, perms := range permSets {
		for _, perm := range perms {
			if !seen[perm] {
				seen[perm] = true
				result = append(result, perm)
			}
		}
	}

	return result
}
