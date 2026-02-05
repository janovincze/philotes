// Package graphql provides GraphQL parsing and transformation for the Dagster proxy.
package graphql

import (
	"encoding/json"

	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

// TransformResponse filters a GraphQL response based on user permissions.
func TransformResponse(
	body []byte,
	op *ParsedOperation,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) ([]byte, error) {
	if authCtx == nil {
		return body, nil
	}

	// Parse response JSON
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return body, nil // Return original on parse error
	}

	// Get data field
	data, ok := response["data"].(map[string]interface{})
	if !ok || data == nil {
		return body, nil
	}

	// Check if user is admin - no filtering needed
	if checker.IsAdmin(authCtx) {
		return body, nil
	}

	// Filter each query field
	modified := false
	for _, field := range op.Fields {
		queryOp, ok := GetQueryOperation(field.Name)
		if !ok {
			continue
		}

		// Get the response field
		responseField := queryOp.ResponseField
		if responseField == "" {
			responseField = field.Name
		}

		// Handle aliased fields
		fieldKey := field.Name
		if field.Alias != "" {
			fieldKey = field.Alias
		}

		fieldData, exists := data[fieldKey]
		if !exists {
			continue
		}

		// Filter the response based on resource type
		filtered := filterResponseField(fieldData, queryOp.ResourceType, authCtx, checker)
		if filtered != nil {
			data[fieldKey] = filtered
			modified = true
		}
	}

	if !modified {
		return body, nil
	}

	// Re-encode response
	response["data"] = data
	return json.Marshal(response)
}

// filterResponseField filters a response field based on resource type and permissions.
func filterResponseField(
	fieldData interface{},
	resourceType auth.ResourceType,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) interface{} {
	// Handle different response structures
	switch data := fieldData.(type) {
	case map[string]interface{}:
		return filterMapResponse(data, resourceType, authCtx, checker)
	case []interface{}:
		return filterArrayResponse(data, resourceType, authCtx, checker)
	default:
		return nil
	}
}

// filterMapResponse filters a map response (e.g., jobsOrError, assetsOrError).
func filterMapResponse(
	data map[string]interface{},
	resourceType auth.ResourceType,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) map[string]interface{} {
	// Check __typename to determine structure
	typename, _ := data["__typename"].(string)

	switch typename {
	case "Jobs", "Pipelines":
		// Has "results" array
		if results, ok := data["results"].([]interface{}); ok {
			data["results"] = filterResourceArray(results, resourceType, authCtx, checker)
		}
		return data

	case "Assets", "AssetConnection":
		// Has "nodes" array
		if nodes, ok := data["nodes"].([]interface{}); ok {
			data["nodes"] = filterResourceArray(nodes, resourceType, authCtx, checker)
		}
		return data

	case "Schedules", "ScheduleConnection":
		if results, ok := data["results"].([]interface{}); ok {
			data["results"] = filterResourceArray(results, resourceType, authCtx, checker)
		}
		return data

	case "Sensors", "SensorConnection":
		if results, ok := data["results"].([]interface{}); ok {
			data["results"] = filterResourceArray(results, resourceType, authCtx, checker)
		}
		return data

	case "Runs", "RunsOrError":
		if results, ok := data["results"].([]interface{}); ok {
			data["results"] = filterRunsArray(results, authCtx, checker)
		}
		return data

	default:
		// Try common array field names
		for _, arrayField := range []string{"results", "nodes", "items"} {
			if arr, ok := data[arrayField].([]interface{}); ok {
				data[arrayField] = filterResourceArray(arr, resourceType, authCtx, checker)
				return data
			}
		}

		// Single resource - check permission
		if !hasViewPermission(data, resourceType, authCtx, checker) {
			// Return error-like response
			return map[string]interface{}{
				"__typename": "UnauthorizedError",
				"message":    "Insufficient permissions to view this resource",
			}
		}
		return data
	}
}

// filterArrayResponse filters an array response.
func filterArrayResponse(
	data []interface{},
	resourceType auth.ResourceType,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) []interface{} {
	return filterResourceArray(data, resourceType, authCtx, checker)
}

// filterResourceArray filters an array of resources.
func filterResourceArray(
	resources []interface{},
	resourceType auth.ResourceType,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) []interface{} {
	canView := checker.CanViewResources(authCtx, resourceType)

	var filtered []interface{}
	for _, item := range resources {
		if resourceMap, ok := item.(map[string]interface{}); ok {
			name := getResourceName(resourceMap, resourceType)
			if canView(name) {
				filtered = append(filtered, item)
			}
		}
	}

	return filtered
}

// filterRunsArray filters runs based on the job they belong to.
func filterRunsArray(
	runs []interface{},
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) []interface{} {
	canViewJob := checker.CanViewResources(authCtx, auth.ResourceTypeJob)

	var filtered []interface{}
	for _, item := range runs {
		if runMap, ok := item.(map[string]interface{}); ok {
			// Get the pipeline/job name from the run
			pipelineName := getRunPipelineName(runMap)
			if pipelineName == "" || canViewJob(pipelineName) {
				filtered = append(filtered, item)
			}
		}
	}

	return filtered
}

// hasViewPermission checks if the user can view a specific resource.
func hasViewPermission(
	resource map[string]interface{},
	resourceType auth.ResourceType,
	authCtx *auth.Context,
	checker *auth.PermissionChecker,
) bool {
	name := getResourceName(resource, resourceType)
	return checker.HasPermission(authCtx, auth.ResourceRef{
		Type:   resourceType,
		Name:   name,
		Action: auth.ActionView,
	})
}

// getResourceName extracts the name/identifier from a resource map.
func getResourceName(resource map[string]interface{}, resourceType auth.ResourceType) string {
	switch resourceType {
	case auth.ResourceTypeJob:
		if name, ok := resource["name"].(string); ok {
			return name
		}
		if name, ok := resource["pipelineName"].(string); ok {
			return name
		}

	case auth.ResourceTypeAsset:
		if key, ok := resource["assetKey"].(map[string]interface{}); ok {
			if path, ok := key["path"].([]interface{}); ok {
				return assetPathToString(path)
			}
		}
		if key, ok := resource["key"].(map[string]interface{}); ok {
			if path, ok := key["path"].([]interface{}); ok {
				return assetPathToString(path)
			}
		}
		if id, ok := resource["id"].(string); ok {
			return id
		}

	case auth.ResourceTypeSchedule, auth.ResourceTypeSensor:
		if name, ok := resource["name"].(string); ok {
			return name
		}

	case auth.ResourceTypeRun:
		if runId, ok := resource["runId"].(string); ok {
			return runId
		}
		if id, ok := resource["id"].(string); ok {
			return id
		}
	}

	return "*"
}

// assetPathToString converts an asset path array to a string.
func assetPathToString(path []interface{}) string {
	var parts []string
	for _, p := range path {
		if s, ok := p.(string); ok {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "*"
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "/" + parts[i]
	}
	return result
}

// getRunPipelineName extracts the pipeline/job name from a run.
func getRunPipelineName(run map[string]interface{}) string {
	// Try pipelineName
	if name, ok := run["pipelineName"].(string); ok {
		return name
	}

	// Try pipeline.name
	if pipeline, ok := run["pipeline"].(map[string]interface{}); ok {
		if name, ok := pipeline["name"].(string); ok {
			return name
		}
	}

	// Try job field
	if job, ok := run["job"].(map[string]interface{}); ok {
		if name, ok := job["name"].(string); ok {
			return name
		}
	}

	return ""
}
