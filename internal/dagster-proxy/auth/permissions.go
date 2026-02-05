// Package auth provides authentication and authorization for the Dagster proxy.
package auth

import (
	"github.com/google/uuid"
)

// ResourceType represents a Dagster resource type.
type ResourceType string

const (
	ResourceTypeJob      ResourceType = "jobs"
	ResourceTypeAsset    ResourceType = "assets"
	ResourceTypeSchedule ResourceType = "schedules"
	ResourceTypeSensor   ResourceType = "sensors"
	ResourceTypeRun      ResourceType = "runs"
)

// ActionType represents an action that can be performed on a resource.
type ActionType string

const (
	ActionView        ActionType = "view"
	ActionExecute     ActionType = "execute"
	ActionMaterialize ActionType = "materialize"
	ActionControl     ActionType = "control"
	ActionTerminate   ActionType = "terminate"
)

// ResourceRef represents a reference to a Dagster resource.
type ResourceRef struct {
	Type   ResourceType
	Name   string
	Action ActionType
}

// PermissionFormat is the format for Dagster permissions:
// dagster:{resource_type}:{resource_name}:{action}
// Examples:
//   - dagster:jobs:*:view - View all jobs
//   - dagster:jobs:etl_orders:execute - Execute specific job
//   - dagster:assets:warehouse/*:view - View assets in warehouse namespace
//   - dagster:*:*:* - Full admin access

// Pre-built role names
const (
	RoleDagsterViewer   = "dagster-viewer"
	RoleDagsterOperator = "dagster-operator"
	RoleDagsterAdmin    = "dagster-admin"
)

// RolePermissions maps roles to their default permissions.
var RolePermissions = map[string][]string{
	RoleDagsterViewer: {
		"dagster:jobs:*:view",
		"dagster:assets:*:view",
		"dagster:schedules:*:view",
		"dagster:sensors:*:view",
		"dagster:runs:*:view",
	},
	RoleDagsterOperator: {
		// Viewer permissions
		"dagster:jobs:*:view",
		"dagster:assets:*:view",
		"dagster:schedules:*:view",
		"dagster:sensors:*:view",
		"dagster:runs:*:view",
		// Execute permissions
		"dagster:jobs:*:execute",
		"dagster:assets:*:materialize",
	},
	RoleDagsterAdmin: {
		// Full access
		"dagster:*:*:*",
	},
}

// Context holds authentication context for a request.
type Context struct {
	UserID      uuid.UUID
	UserEmail   string
	Role        string
	Permissions []string
	IsAPIKey    bool
	APIKeyID    *uuid.UUID
}

// HasPermission checks if the context has a specific permission.
func (c *Context) HasPermission(permission string) bool {
	if c == nil {
		return false
	}

	for _, p := range c.Permissions {
		if MatchPermission(p, permission) {
			return true
		}
	}
	return false
}

// GetDagsterPermissions returns only Dagster-related permissions.
func (c *Context) GetDagsterPermissions() []string {
	if c == nil {
		return nil
	}

	var dagsterPerms []string
	for _, p := range c.Permissions {
		if len(p) > 8 && p[:8] == "dagster:" {
			dagsterPerms = append(dagsterPerms, p)
		}
	}
	return dagsterPerms
}

// PermissionError represents a permission denied error.
type PermissionError struct {
	Message      string
	ResourceType string
	ResourceName string
	Action       string
}

func (e *PermissionError) Error() string {
	if e.ResourceType != "" {
		return e.Message + ": " + e.ResourceType + "/" + e.ResourceName + " " + e.Action
	}
	return e.Message
}

// BuildPermission constructs a permission string from components.
func BuildPermission(resourceType ResourceType, resourceName string, action ActionType) string {
	return "dagster:" + string(resourceType) + ":" + resourceName + ":" + string(action)
}

// ParsePermission parses a permission string into its components.
// Returns empty strings if the format is invalid.
func ParsePermission(perm string) (prefix, resourceType, resourceName, action string) {
	parts := splitPermission(perm)
	if len(parts) != 4 {
		return "", "", "", ""
	}
	return parts[0], parts[1], parts[2], parts[3]
}

// splitPermission splits a permission string by colons, handling escaped colons.
func splitPermission(perm string) []string {
	var parts []string
	var current []byte

	for i := 0; i < len(perm); i++ {
		if perm[i] == ':' {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, perm[i])
		}
	}
	parts = append(parts, string(current))

	return parts
}
