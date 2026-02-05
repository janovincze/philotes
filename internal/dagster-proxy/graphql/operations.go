// Package graphql provides GraphQL parsing and transformation for the Dagster proxy.
package graphql

import (
	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

// OperationType represents a GraphQL operation type.
type OperationType string

const (
	OperationQuery        OperationType = "query"
	OperationMutation     OperationType = "mutation"
	OperationSubscription OperationType = "subscription"
)

// DagsterOperation represents a known Dagster GraphQL operation.
type DagsterOperation struct {
	Name         string
	Type         OperationType
	ResourceType auth.ResourceType
	Action       auth.ActionType
	// ResourceField is the field name in the request that contains the resource identifier
	ResourceField string
	// ResponseField is the field name in the response that contains the results
	ResponseField string
}

// KnownQueries maps Dagster query field names to their resource types.
var KnownQueries = map[string]DagsterOperation{
	// Jobs/Pipelines
	"pipelinesOrError": {
		Name:          "pipelinesOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionView,
		ResponseField: "pipelinesOrError",
	},
	"jobsOrError": {
		Name:          "jobsOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionView,
		ResponseField: "jobsOrError",
	},
	"pipelineOrError": {
		Name:          "pipelineOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionView,
		ResourceField: "pipelineName",
		ResponseField: "pipelineOrError",
	},
	"jobOrError": {
		Name:          "jobOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionView,
		ResourceField: "jobName",
		ResponseField: "jobOrError",
	},

	// Assets
	"assetNodes": {
		Name:          "assetNodes",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionView,
		ResponseField: "assetNodes",
	},
	"assetsOrError": {
		Name:          "assetsOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionView,
		ResponseField: "assetsOrError",
	},
	"assetOrError": {
		Name:          "assetOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionView,
		ResourceField: "assetKey",
		ResponseField: "assetOrError",
	},
	"assetNodeOrError": {
		Name:          "assetNodeOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionView,
		ResourceField: "assetKey",
		ResponseField: "assetNodeOrError",
	},

	// Schedules
	"schedulesOrError": {
		Name:          "schedulesOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeSchedule,
		Action:        auth.ActionView,
		ResponseField: "schedulesOrError",
	},
	"scheduleOrError": {
		Name:          "scheduleOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeSchedule,
		Action:        auth.ActionView,
		ResourceField: "scheduleName",
		ResponseField: "scheduleOrError",
	},

	// Sensors
	"sensorsOrError": {
		Name:          "sensorsOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionView,
		ResponseField: "sensorsOrError",
	},
	"sensorOrError": {
		Name:          "sensorOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionView,
		ResourceField: "sensorName",
		ResponseField: "sensorOrError",
	},

	// Runs
	"runsOrError": {
		Name:          "runsOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionView,
		ResponseField: "runsOrError",
	},
	"runOrError": {
		Name:          "runOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionView,
		ResourceField: "runId",
		ResponseField: "runOrError",
	},
	"pipelineRunOrError": {
		Name:          "pipelineRunOrError",
		Type:          OperationQuery,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionView,
		ResourceField: "runId",
		ResponseField: "pipelineRunOrError",
	},
}

// KnownMutations maps Dagster mutation field names to their resource types and actions.
var KnownMutations = map[string]DagsterOperation{
	// Job execution
	"launchRun": {
		Name:          "launchRun",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionExecute,
		ResourceField: "executionParams.selector.pipelineName",
	},
	"launchPipelineExecution": {
		Name:          "launchPipelineExecution",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionExecute,
		ResourceField: "executionParams.selector.pipelineName",
	},
	"launchBackfill": {
		Name:          "launchBackfill",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeJob,
		Action:        auth.ActionExecute,
		ResourceField: "backfillParams.selector.pipelineName",
	},

	// Asset materialization
	"launchPartitionBackfill": {
		Name:          "launchPartitionBackfill",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionMaterialize,
		ResourceField: "backfillParams.assetSelection",
	},
	"reportRunlessAssetEvents": {
		Name:          "reportRunlessAssetEvents",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionMaterialize,
		ResourceField: "eventParams.assetKey",
	},
	"wipeAssets": {
		Name:          "wipeAssets",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeAsset,
		Action:        auth.ActionMaterialize,
		ResourceField: "assetKeys",
	},

	// Schedule control
	"startSchedule": {
		Name:          "startSchedule",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSchedule,
		Action:        auth.ActionControl,
		ResourceField: "scheduleSelector.scheduleName",
	},
	"stopSchedule": {
		Name:          "stopSchedule",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSchedule,
		Action:        auth.ActionControl,
		ResourceField: "scheduleOriginId",
	},
	"resetSchedule": {
		Name:          "resetSchedule",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSchedule,
		Action:        auth.ActionControl,
		ResourceField: "scheduleSelector.scheduleName",
	},

	// Sensor control
	"startSensor": {
		Name:          "startSensor",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionControl,
		ResourceField: "sensorSelector.sensorName",
	},
	"stopSensor": {
		Name:          "stopSensor",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionControl,
		ResourceField: "sensorOriginId",
	},
	"resetSensor": {
		Name:          "resetSensor",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionControl,
		ResourceField: "sensorSelector.sensorName",
	},
	"setSensorCursor": {
		Name:          "setSensorCursor",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeSensor,
		Action:        auth.ActionControl,
		ResourceField: "sensorSelector.sensorName",
	},

	// Run termination
	"terminateRun": {
		Name:          "terminateRun",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionTerminate,
		ResourceField: "runId",
	},
	"terminateRuns": {
		Name:          "terminateRuns",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionTerminate,
		ResourceField: "runIds",
	},
	"deletePipelineRun": {
		Name:          "deletePipelineRun",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionTerminate,
		ResourceField: "runId",
	},
	"deleteRun": {
		Name:          "deleteRun",
		Type:          OperationMutation,
		ResourceType:  auth.ResourceTypeRun,
		Action:        auth.ActionTerminate,
		ResourceField: "runId",
	},
}

// GetQueryOperation returns the operation info for a query field.
func GetQueryOperation(fieldName string) (DagsterOperation, bool) {
	op, ok := KnownQueries[fieldName]
	return op, ok
}

// GetMutationOperation returns the operation info for a mutation field.
func GetMutationOperation(fieldName string) (DagsterOperation, bool) {
	op, ok := KnownMutations[fieldName]
	return op, ok
}

// IsResourceQuery checks if a field name is a known resource query.
func IsResourceQuery(fieldName string) bool {
	_, ok := KnownQueries[fieldName]
	return ok
}

// IsResourceMutation checks if a field name is a known resource mutation.
func IsResourceMutation(fieldName string) bool {
	_, ok := KnownMutations[fieldName]
	return ok
}
