package auth

import (
	"testing"
)

func TestMatchPermission(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		required string
		want     bool
	}{
		// Exact matches
		{
			name:     "exact match",
			pattern:  "dagster:jobs:orders:view",
			required: "dagster:jobs:orders:view",
			want:     true,
		},
		{
			name:     "no match - different resource",
			pattern:  "dagster:jobs:orders:view",
			required: "dagster:jobs:customers:view",
			want:     false,
		},
		{
			name:     "no match - different action",
			pattern:  "dagster:jobs:orders:view",
			required: "dagster:jobs:orders:execute",
			want:     false,
		},

		// Wildcard matches - single segment
		{
			name:     "wildcard resource name",
			pattern:  "dagster:jobs:*:view",
			required: "dagster:jobs:orders:view",
			want:     true,
		},
		{
			name:     "wildcard action",
			pattern:  "dagster:jobs:orders:*",
			required: "dagster:jobs:orders:execute",
			want:     true,
		},
		{
			name:     "full wildcard",
			pattern:  "dagster:*:*:*",
			required: "dagster:jobs:orders:execute",
			want:     true,
		},

		// Prefix wildcards
		{
			name:     "prefix wildcard match",
			pattern:  "dagster:jobs:etl_*:execute",
			required: "dagster:jobs:etl_orders:execute",
			want:     true,
		},
		{
			name:     "prefix wildcard no match",
			pattern:  "dagster:jobs:etl_*:execute",
			required: "dagster:jobs:ml_orders:execute",
			want:     false,
		},

		// Suffix wildcards
		{
			name:     "suffix wildcard match",
			pattern:  "dagster:jobs:*_daily:execute",
			required: "dagster:jobs:orders_daily:execute",
			want:     true,
		},
		{
			name:     "suffix wildcard no match",
			pattern:  "dagster:jobs:*_daily:execute",
			required: "dagster:jobs:orders_hourly:execute",
			want:     false,
		},

		// Admin wildcard
		{
			name:     "admin full access",
			pattern:  "dagster:*:*:*",
			required: "dagster:schedules:monthly_report:control",
			want:     true,
		},

		// Different segment counts (should not match)
		{
			name:     "different segment counts",
			pattern:  "dagster:jobs:*",
			required: "dagster:jobs:orders:view",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchPermission(tt.pattern, tt.required)
			if got != tt.want {
				t.Errorf("MatchPermission(%q, %q) = %v, want %v",
					tt.pattern, tt.required, got, tt.want)
			}
		})
	}
}

func TestMatchResourcePermission(t *testing.T) {
	tests := []struct {
		name         string
		permissions  []string
		resourceType ResourceType
		resourceName string
		action       ActionType
		want         bool
	}{
		{
			name: "viewer can view jobs",
			permissions: []string{
				"dagster:jobs:*:view",
				"dagster:assets:*:view",
			},
			resourceType: ResourceTypeJob,
			resourceName: "etl_orders",
			action:       ActionView,
			want:         true,
		},
		{
			name: "viewer cannot execute jobs",
			permissions: []string{
				"dagster:jobs:*:view",
				"dagster:assets:*:view",
			},
			resourceType: ResourceTypeJob,
			resourceName: "etl_orders",
			action:       ActionExecute,
			want:         false,
		},
		{
			name: "operator can execute jobs",
			permissions: []string{
				"dagster:jobs:*:view",
				"dagster:jobs:*:execute",
			},
			resourceType: ResourceTypeJob,
			resourceName: "etl_orders",
			action:       ActionExecute,
			want:         true,
		},
		{
			name: "admin has full access",
			permissions: []string{
				"dagster:*:*:*",
			},
			resourceType: ResourceTypeSchedule,
			resourceName: "monthly_report",
			action:       ActionControl,
			want:         true,
		},
		{
			name: "scoped permission - allowed prefix",
			permissions: []string{
				"dagster:jobs:etl_*:execute",
			},
			resourceType: ResourceTypeJob,
			resourceName: "etl_customers",
			action:       ActionExecute,
			want:         true,
		},
		{
			name: "scoped permission - disallowed prefix",
			permissions: []string{
				"dagster:jobs:etl_*:execute",
			},
			resourceType: ResourceTypeJob,
			resourceName: "ml_training",
			action:       ActionExecute,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchResourcePermission(tt.permissions, tt.resourceType, tt.resourceName, tt.action)
			if got != tt.want {
				t.Errorf("MatchResourcePermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPermission(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		resourceName string
		action       ActionType
		want         string
	}{
		{
			resourceType: ResourceTypeJob,
			resourceName: "orders_etl",
			action:       ActionExecute,
			want:         "dagster:jobs:orders_etl:execute",
		},
		{
			resourceType: ResourceTypeAsset,
			resourceName: "warehouse/customers",
			action:       ActionMaterialize,
			want:         "dagster:assets:warehouse/customers:materialize",
		},
		{
			resourceType: ResourceTypeSchedule,
			resourceName: "daily_sync",
			action:       ActionControl,
			want:         "dagster:schedules:daily_sync:control",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := BuildPermission(tt.resourceType, tt.resourceName, tt.action)
			if got != tt.want {
				t.Errorf("BuildPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandRoleToPermissions(t *testing.T) {
	tests := []struct {
		roleName     string
		wantCount    int
		wantContains string
	}{
		{
			roleName:     RoleDagsterViewer,
			wantCount:    5,
			wantContains: "dagster:jobs:*:view",
		},
		{
			roleName:     RoleDagsterOperator,
			wantCount:    7,
			wantContains: "dagster:jobs:*:execute",
		},
		{
			roleName:     RoleDagsterAdmin,
			wantCount:    1,
			wantContains: "dagster:*:*:*",
		},
		{
			roleName:  "unknown-role",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.roleName, func(t *testing.T) {
			perms := ExpandRoleToPermissions(tt.roleName)
			if len(perms) != tt.wantCount {
				t.Errorf("ExpandRoleToPermissions(%q) returned %d perms, want %d",
					tt.roleName, len(perms), tt.wantCount)
			}
			if tt.wantContains != "" {
				found := false
				for _, p := range perms {
					if p == tt.wantContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExpandRoleToPermissions(%q) missing %q", tt.roleName, tt.wantContains)
				}
			}
		})
	}
}
