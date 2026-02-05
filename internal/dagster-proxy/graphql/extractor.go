// Package graphql provides GraphQL parsing and transformation for the Dagster proxy.
package graphql

import (
	"strings"

	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

// ExtractResources extracts resource references from a parsed GraphQL operation.
// For mutations, it returns the specific resources being modified.
// For queries, it returns the resource types being queried.
func ExtractResources(op *ParsedOperation) []auth.ResourceRef {
	if op == nil {
		return nil
	}

	var resources []auth.ResourceRef

	switch op.Type {
	case OperationMutation:
		resources = extractMutationResources(op)
	case OperationQuery:
		resources = extractQueryResources(op)
	case OperationSubscription:
		resources = extractSubscriptionResources(op)
	}

	return resources
}

// extractMutationResources extracts resources from mutation operations.
func extractMutationResources(op *ParsedOperation) []auth.ResourceRef {
	var resources []auth.ResourceRef

	for _, field := range op.Fields {
		// Skip introspection fields
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		mutOp, ok := GetMutationOperation(field.Name)
		if !ok {
			// Unknown mutation - require admin access
			resources = append(resources, auth.ResourceRef{
				Type:   auth.ResourceTypeJob, // Default to job
				Name:   "*",
				Action: auth.ActionExecute,
			})
			continue
		}

		// Extract resource name from arguments
		resourceName := extractResourceName(field, mutOp, op.Variables)

		resources = append(resources, auth.ResourceRef{
			Type:   mutOp.ResourceType,
			Name:   resourceName,
			Action: mutOp.Action,
		})
	}

	return resources
}

// extractQueryResources extracts resources from query operations.
func extractQueryResources(op *ParsedOperation) []auth.ResourceRef {
	var resources []auth.ResourceRef

	for _, field := range op.Fields {
		// Skip introspection fields
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		queryOp, ok := GetQueryOperation(field.Name)
		if !ok {
			// Unknown query - we'll allow it but log it
			continue
		}

		// For single-resource queries, extract the resource name
		resourceName := "*"
		if queryOp.ResourceField != "" {
			if name := field.GetStringArgument(queryOp.ResourceField); name != "" {
				resourceName = name
			}
		}

		resources = append(resources, auth.ResourceRef{
			Type:   queryOp.ResourceType,
			Name:   resourceName,
			Action: queryOp.Action,
		})
	}

	return resources
}

// extractSubscriptionResources extracts resources from subscription operations.
func extractSubscriptionResources(op *ParsedOperation) []auth.ResourceRef {
	// Subscriptions are treated similar to queries
	return extractQueryResources(op)
}

// extractResourceName extracts the resource name from a field's arguments.
func extractResourceName(field FieldSelection, mutOp DagsterOperation, variables map[string]interface{}) string {
	if mutOp.ResourceField == "" {
		return "*"
	}

	// Handle nested paths like "executionParams.selector.pipelineName"
	parts := strings.Split(mutOp.ResourceField, ".")

	// Start with the field arguments
	var current interface{} = field.Arguments

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			// Check variables
			if variables != nil {
				if nested := GetNestedValue(variables, mutOp.ResourceField); nested != nil {
					if s, ok := nested.(string); ok {
						return s
					}
				}
			}
			return "*"
		}
	}

	// Convert final value to string
	switch v := current.(type) {
	case string:
		return v
	case []interface{}:
		// Multiple resources (e.g., assetKeys) - return wildcard for now
		// In a more sophisticated implementation, we'd check each
		return "*"
	case map[string]interface{}:
		// Asset key object - try to extract path
		if path, ok := v["path"]; ok {
			if pathSlice, ok := path.([]interface{}); ok {
				var parts []string
				for _, p := range pathSlice {
					if s, ok := p.(string); ok {
						parts = append(parts, s)
					}
				}
				return strings.Join(parts, "/")
			}
		}
		return "*"
	default:
		return "*"
	}
}

// ExtractJobNames extracts job names from a jobs query response.
func ExtractJobNames(jobs []map[string]interface{}) []string {
	var names []string
	for _, job := range jobs {
		if name, ok := job["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// ExtractAssetKeys extracts asset keys from an assets query response.
func ExtractAssetKeys(assets []map[string]interface{}) []string {
	var keys []string
	for _, asset := range assets {
		key := extractAssetKeyString(asset)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// extractAssetKeyString extracts the asset key string from an asset object.
func extractAssetKeyString(asset map[string]interface{}) string {
	// Asset key can be in different formats
	if key, ok := asset["assetKey"].(map[string]interface{}); ok {
		if path, ok := key["path"].([]interface{}); ok {
			var parts []string
			for _, p := range path {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			return strings.Join(parts, "/")
		}
	}

	// Try direct key field
	if key, ok := asset["key"].(map[string]interface{}); ok {
		if path, ok := key["path"].([]interface{}); ok {
			var parts []string
			for _, p := range path {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			return strings.Join(parts, "/")
		}
	}

	// Try id field
	if id, ok := asset["id"].(string); ok {
		return id
	}

	return ""
}

// ExtractScheduleNames extracts schedule names from a schedules query response.
func ExtractScheduleNames(schedules []map[string]interface{}) []string {
	var names []string
	for _, schedule := range schedules {
		if name, ok := schedule["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// ExtractSensorNames extracts sensor names from a sensors query response.
func ExtractSensorNames(sensors []map[string]interface{}) []string {
	var names []string
	for _, sensor := range sensors {
		if name, ok := sensor["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// GetResourceIdentifier returns the identifying field for a resource type.
func GetResourceIdentifier(resourceType auth.ResourceType) string {
	switch resourceType {
	case auth.ResourceTypeJob:
		return "name"
	case auth.ResourceTypeAsset:
		return "assetKey"
	case auth.ResourceTypeSchedule:
		return "name"
	case auth.ResourceTypeSensor:
		return "name"
	case auth.ResourceTypeRun:
		return "runId"
	default:
		return "name"
	}
}
