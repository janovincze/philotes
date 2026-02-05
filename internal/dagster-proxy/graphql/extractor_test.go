package graphql

import (
	"testing"

	"github.com/janovincze/philotes/internal/dagster-proxy/auth"
)

func TestExtractResources_Query(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name         string
		query        string
		wantType     auth.ResourceType
		wantAction   auth.ActionType
		wantResCount int
	}{
		{
			name:         "jobs query",
			query:        `{"query": "{ jobsOrError { results { name } } }"}`,
			wantType:     auth.ResourceTypeJob,
			wantAction:   auth.ActionView,
			wantResCount: 1,
		},
		{
			name:         "assets query",
			query:        `{"query": "{ assetNodes { id } }"}`,
			wantType:     auth.ResourceTypeAsset,
			wantAction:   auth.ActionView,
			wantResCount: 1,
		},
		{
			name:         "schedules query",
			query:        `{"query": "{ schedulesOrError { results { name } } }"}`,
			wantType:     auth.ResourceTypeSchedule,
			wantAction:   auth.ActionView,
			wantResCount: 1,
		},
		{
			name:         "sensors query",
			query:        `{"query": "{ sensorsOrError { results { name } } }"}`,
			wantType:     auth.ResourceTypeSensor,
			wantAction:   auth.ActionView,
			wantResCount: 1,
		},
		{
			name:         "multiple queries",
			query:        `{"query": "{ jobsOrError { results { name } } assetNodes { id } }"}`,
			wantResCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse([]byte(tt.query))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			resources := ExtractResources(parsed)

			if len(resources) != tt.wantResCount {
				t.Fatalf("ExtractResources() returned %d resources, want %d",
					len(resources), tt.wantResCount)
			}

			if tt.wantResCount > 0 && tt.wantType != "" {
				if resources[0].Type != tt.wantType {
					t.Errorf("Resource.Type = %v, want %v", resources[0].Type, tt.wantType)
				}
				if resources[0].Action != tt.wantAction {
					t.Errorf("Resource.Action = %v, want %v", resources[0].Action, tt.wantAction)
				}
			}
		})
	}
}

func TestExtractResources_Mutation(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name       string
		query      string
		wantType   auth.ResourceType
		wantAction auth.ActionType
	}{
		{
			name:       "launch run mutation",
			query:      `{"query": "mutation { launchRun(executionParams: {}) { __typename } }"}`,
			wantType:   auth.ResourceTypeJob,
			wantAction: auth.ActionExecute,
		},
		{
			name:       "terminate run mutation",
			query:      `{"query": "mutation { terminateRun(runId: \"abc123\") { __typename } }"}`,
			wantType:   auth.ResourceTypeRun,
			wantAction: auth.ActionTerminate,
		},
		{
			name:       "start schedule mutation",
			query:      `{"query": "mutation { startSchedule(scheduleSelector: {}) { __typename } }"}`,
			wantType:   auth.ResourceTypeSchedule,
			wantAction: auth.ActionControl,
		},
		{
			name:       "stop sensor mutation",
			query:      `{"query": "mutation { stopSensor(sensorOriginId: \"abc\") { __typename } }"}`,
			wantType:   auth.ResourceTypeSensor,
			wantAction: auth.ActionControl,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse([]byte(tt.query))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			resources := ExtractResources(parsed)

			if len(resources) != 1 {
				t.Fatalf("ExtractResources() returned %d resources, want 1", len(resources))
			}

			if resources[0].Type != tt.wantType {
				t.Errorf("Resource.Type = %v, want %v", resources[0].Type, tt.wantType)
			}
			if resources[0].Action != tt.wantAction {
				t.Errorf("Resource.Action = %v, want %v", resources[0].Action, tt.wantAction)
			}
		})
	}
}

func TestExtractJobNames(t *testing.T) {
	jobs := []map[string]interface{}{
		{"name": "orders_etl", "description": "ETL for orders"},
		{"name": "customers_sync", "description": "Customer sync"},
		{"name": "reports_daily"},
	}

	names := ExtractJobNames(jobs)

	if len(names) != 3 {
		t.Fatalf("ExtractJobNames() returned %d names, want 3", len(names))
	}

	expected := []string{"orders_etl", "customers_sync", "reports_daily"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %v, want %v", i, name, expected[i])
		}
	}
}

func TestExtractAssetKeys(t *testing.T) {
	assets := []map[string]interface{}{
		{
			"assetKey": map[string]interface{}{
				"path": []interface{}{"warehouse", "customers"},
			},
		},
		{
			"key": map[string]interface{}{
				"path": []interface{}{"raw", "orders"},
			},
		},
		{
			"id": "simple_asset",
		},
	}

	keys := ExtractAssetKeys(assets)

	if len(keys) != 3 {
		t.Fatalf("ExtractAssetKeys() returned %d keys, want 3", len(keys))
	}

	expected := []string{"warehouse/customers", "raw/orders", "simple_asset"}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] = %v, want %v", i, key, expected[i])
		}
	}
}

func TestExtractScheduleNames(t *testing.T) {
	schedules := []map[string]interface{}{
		{"name": "daily_sync"},
		{"name": "hourly_check"},
	}

	names := ExtractScheduleNames(schedules)

	if len(names) != 2 {
		t.Fatalf("ExtractScheduleNames() returned %d names, want 2", len(names))
	}

	if names[0] != "daily_sync" || names[1] != "hourly_check" {
		t.Errorf("Unexpected schedule names: %v", names)
	}
}

func TestExtractSensorNames(t *testing.T) {
	sensors := []map[string]interface{}{
		{"name": "file_watcher"},
		{"name": "queue_sensor"},
	}

	names := ExtractSensorNames(sensors)

	if len(names) != 2 {
		t.Fatalf("ExtractSensorNames() returned %d names, want 2", len(names))
	}

	if names[0] != "file_watcher" || names[1] != "queue_sensor" {
		t.Errorf("Unexpected sensor names: %v", names)
	}
}

func TestGetResourceIdentifier(t *testing.T) {
	tests := []struct {
		resourceType auth.ResourceType
		want         string
	}{
		{auth.ResourceTypeJob, "name"},
		{auth.ResourceTypeAsset, "assetKey"},
		{auth.ResourceTypeSchedule, "name"},
		{auth.ResourceTypeSensor, "name"},
		{auth.ResourceTypeRun, "runId"},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			got := GetResourceIdentifier(tt.resourceType)
			if got != tt.want {
				t.Errorf("GetResourceIdentifier(%v) = %v, want %v",
					tt.resourceType, got, tt.want)
			}
		})
	}
}
