package graphql

import (
	"testing"
)

func TestParseRequest(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		body     string
		wantType OperationType
		wantName string
		wantErr  bool
	}{
		{
			name:     "simple query",
			body:     `{"query": "query GetJobs { jobsOrError { __typename } }"}`,
			wantType: OperationQuery,
			wantName: "GetJobs",
			wantErr:  false,
		},
		{
			name:     "mutation",
			body:     `{"query": "mutation LaunchRun { launchRun(executionParams: {}) { __typename } }"}`,
			wantType: OperationMutation,
			wantName: "LaunchRun",
			wantErr:  false,
		},
		{
			name:     "anonymous query",
			body:     `{"query": "{ jobsOrError { __typename } }"}`,
			wantType: OperationQuery,
			wantName: "",
			wantErr:  false,
		},
		{
			name:     "with operation name",
			body:     `{"query": "query A { a } query B { b }", "operationName": "B"}`,
			wantType: OperationQuery,
			wantName: "B",
			wantErr:  false,
		},
		{
			name:    "empty query",
			body:    `{"query": ""}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			body:    `not json`,
			wantErr: true,
		},
		{
			name:    "invalid GraphQL",
			body:    `{"query": "not valid graphql }"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if parsed.Type != tt.wantType {
				t.Errorf("Parse().Type = %v, want %v", parsed.Type, tt.wantType)
			}
			if parsed.Name != tt.wantName {
				t.Errorf("Parse().Name = %v, want %v", parsed.Name, tt.wantName)
			}
		})
	}
}

func TestParseWithVariables(t *testing.T) {
	parser := NewParser()

	body := `{
		"query": "query GetJob($name: String!) { jobOrError(jobName: $name) { __typename } }",
		"variables": {"name": "orders_etl"}
	}`

	parsed, err := parser.Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Variables == nil {
		t.Error("Parse().Variables is nil")
		return
	}

	if name, ok := parsed.Variables["name"]; !ok || name != "orders_etl" {
		t.Errorf("Parse().Variables[\"name\"] = %v, want \"orders_etl\"", name)
	}
}

func TestParseFields(t *testing.T) {
	parser := NewParser()

	body := `{
		"query": "query { jobsOrError { results { name } } }"
	}`

	parsed, err := parser.Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Fields) != 1 {
		t.Fatalf("Parse().Fields has %d fields, want 1", len(parsed.Fields))
	}

	field := parsed.Fields[0]
	if field.Name != "jobsOrError" {
		t.Errorf("Field.Name = %v, want \"jobsOrError\"", field.Name)
	}

	if len(field.Fields) != 1 {
		t.Fatalf("Field.Fields has %d sub-fields, want 1", len(field.Fields))
	}

	if field.Fields[0].Name != "results" {
		t.Errorf("Sub-field.Name = %v, want \"results\"", field.Fields[0].Name)
	}
}

func TestParseFieldsWithArguments(t *testing.T) {
	parser := NewParser()

	body := `{
		"query": "query { jobOrError(jobName: \"orders_etl\") { __typename } }"
	}`

	parsed, err := parser.Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(parsed.Fields) != 1 {
		t.Fatalf("Parse().Fields has %d fields, want 1", len(parsed.Fields))
	}

	field := parsed.Fields[0]
	if field.Arguments == nil {
		t.Error("Field.Arguments is nil")
		return
	}

	jobName, ok := field.Arguments["jobName"]
	if !ok {
		t.Error("Field.Arguments missing \"jobName\"")
		return
	}

	if jobName != "orders_etl" {
		t.Errorf("Field.Arguments[\"jobName\"] = %v, want \"orders_etl\"", jobName)
	}
}

func TestParsedOperationMethods(t *testing.T) {
	parser := NewParser()

	body := `{
		"query": "query { jobsOrError { results { name } } assetNodes { id } }"
	}`

	parsed, err := parser.Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Test GetTopLevelFields
	fields := parsed.GetTopLevelFields()
	if len(fields) != 2 {
		t.Errorf("GetTopLevelFields() returned %d fields, want 2", len(fields))
	}

	// Test HasField
	if !parsed.HasField("jobsOrError") {
		t.Error("HasField(\"jobsOrError\") = false, want true")
	}
	if !parsed.HasField("assetNodes") {
		t.Error("HasField(\"assetNodes\") = false, want true")
	}
	if parsed.HasField("nonExistent") {
		t.Error("HasField(\"nonExistent\") = true, want false")
	}

	// Test GetField
	field := parsed.GetField("jobsOrError")
	if field == nil {
		t.Error("GetField(\"jobsOrError\") = nil, want non-nil")
	}
	if field != nil && field.Name != "jobsOrError" {
		t.Errorf("GetField().Name = %v, want \"jobsOrError\"", field.Name)
	}
}

func TestGetNestedValue(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		path string
		want interface{}
	}{
		{
			name: "simple path",
			data: map[string]interface{}{"name": "orders"},
			path: "name",
			want: "orders",
		},
		{
			name: "nested path",
			data: map[string]interface{}{
				"selector": map[string]interface{}{
					"pipelineName": "etl_orders",
				},
			},
			path: "selector.pipelineName",
			want: "etl_orders",
		},
		{
			name: "deep nested path",
			data: map[string]interface{}{
				"executionParams": map[string]interface{}{
					"selector": map[string]interface{}{
						"pipelineName": "orders_etl",
					},
				},
			},
			path: "executionParams.selector.pipelineName",
			want: "orders_etl",
		},
		{
			name: "missing path",
			data: map[string]interface{}{"name": "orders"},
			path: "missing.path",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNestedValue(tt.data, tt.path)
			if got != tt.want {
				t.Errorf("GetNestedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark parsing performance
func BenchmarkParse(b *testing.B) {
	parser := NewParser()
	body := []byte(`{
		"query": "query GetJobsAndAssets { jobsOrError { results { name description } } assetNodes { id assetKey { path } } }",
		"operationName": "GetJobsAndAssets"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parser.Parse(body); err != nil {
			b.Fatalf("Parse() error = %v", err)
		}
	}
}
