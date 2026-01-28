package alerting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEvaluator(t *testing.T) {
	tests := []struct {
		name          string
		prometheusURL string
		wantURL       string
	}{
		{
			name:          "basic URL",
			prometheusURL: "http://localhost:9090",
			wantURL:       "http://localhost:9090",
		},
		{
			name:          "URL with trailing slash",
			prometheusURL: "http://localhost:9090/",
			wantURL:       "http://localhost:9090",
		},
		{
			name:          "URL with multiple trailing slashes",
			prometheusURL: "http://prometheus.default.svc:9090///",
			wantURL:       "http://prometheus.default.svc:9090//",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEvaluator(tt.prometheusURL, nil)
			if e == nil {
				t.Fatal("NewEvaluator returned nil")
			}
			if e.prometheusURL != tt.wantURL {
				t.Errorf("prometheusURL = %q, want %q", e.prometheusURL, tt.wantURL)
			}
			if e.httpClient == nil {
				t.Error("httpClient should not be nil")
			}
			if e.logger == nil {
				t.Error("logger should not be nil")
			}
		})
	}
}

func TestEvaluator_buildQuery(t *testing.T) {
	e := NewEvaluator("http://localhost:9090", nil)

	tests := []struct {
		name       string
		metricName string
		labels     map[string]string
		want       string
	}{
		{
			name:       "metric only",
			metricName: "up",
			labels:     nil,
			want:       "up",
		},
		{
			name:       "metric with empty labels",
			metricName: "up",
			labels:     map[string]string{},
			want:       "up",
		},
		{
			name:       "metric with single label",
			metricName: "philotes_cdc_events_total",
			labels:     map[string]string{"source": "db1"},
			want:       `philotes_cdc_events_total{source="db1"}`,
		},
		{
			name:       "metric with label containing quotes",
			metricName: "test_metric",
			labels:     map[string]string{"label": `value"with"quotes`},
			want:       `test_metric{label="value\"with\"quotes"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.buildQuery(tt.metricName, tt.labels)
			if got != tt.want {
				t.Errorf("buildQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvaluator_parseResult(t *testing.T) {
	e := NewEvaluator("http://localhost:9090", nil)

	tests := []struct {
		name    string
		result  prometheusResult
		want    MetricValue
		wantErr bool
	}{
		{
			name: "valid result",
			result: prometheusResult{
				Metric: map[string]string{"source": "db1"},
				Value:  []interface{}{float64(1704067200), "42.5"},
			},
			want: MetricValue{
				Labels: map[string]string{"source": "db1"},
				Value:  42.5,
				Time:   time.Unix(1704067200, 0),
			},
			wantErr: false,
		},
		{
			name: "integer value as string",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{float64(1704067200), "100"},
			},
			want: MetricValue{
				Labels: map[string]string{},
				Value:  100,
				Time:   time.Unix(1704067200, 0),
			},
			wantErr: false,
		},
		{
			name: "empty value array",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{},
			},
			wantErr: true,
		},
		{
			name: "single value in array",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{float64(1704067200)},
			},
			wantErr: true,
		},
		{
			name: "invalid timestamp type",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{"invalid", "42.5"},
			},
			wantErr: true,
		},
		{
			name: "invalid value type",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{float64(1704067200), 42.5},
			},
			wantErr: true,
		},
		{
			name: "invalid value string",
			result: prometheusResult{
				Metric: map[string]string{},
				Value:  []interface{}{float64(1704067200), "not_a_number"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.parseResult(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Value != tt.want.Value {
					t.Errorf("Value = %v, want %v", got.Value, tt.want.Value)
				}
				if !got.Time.Equal(tt.want.Time) {
					t.Errorf("Time = %v, want %v", got.Time, tt.want.Time)
				}
				for k, v := range tt.want.Labels {
					if got.Labels[k] != v {
						t.Errorf("Labels[%q] = %q, want %q", k, got.Labels[k], v)
					}
				}
			}
		})
	}
}

func TestEvaluator_Evaluate(t *testing.T) {
	tests := []struct {
		name       string
		response   prometheusResponse
		rule       AlertRule
		wantCount  int
		wantFiring []bool
		wantErr    bool
	}{
		{
			name: "single result firing",
			response: prometheusResponse{
				Status: "success",
				Data: struct {
					ResultType string             `json:"resultType"`
					Result     []prometheusResult `json:"result"`
				}{
					ResultType: "vector",
					Result: []prometheusResult{
						{
							Metric: map[string]string{"source": "db1"},
							Value:  []interface{}{float64(1704067200), "100"},
						},
					},
				},
			},
			rule: AlertRule{
				ID:         uuid.New(),
				MetricName: "test_metric",
				Operator:   OpGreaterThan,
				Threshold:  50,
			},
			wantCount:  1,
			wantFiring: []bool{true},
			wantErr:    false,
		},
		{
			name: "single result not firing",
			response: prometheusResponse{
				Status: "success",
				Data: struct {
					ResultType string             `json:"resultType"`
					Result     []prometheusResult `json:"result"`
				}{
					ResultType: "vector",
					Result: []prometheusResult{
						{
							Metric: map[string]string{"source": "db1"},
							Value:  []interface{}{float64(1704067200), "30"},
						},
					},
				},
			},
			rule: AlertRule{
				ID:         uuid.New(),
				MetricName: "test_metric",
				Operator:   OpGreaterThan,
				Threshold:  50,
			},
			wantCount:  1,
			wantFiring: []bool{false},
			wantErr:    false,
		},
		{
			name: "multiple results mixed",
			response: prometheusResponse{
				Status: "success",
				Data: struct {
					ResultType string             `json:"resultType"`
					Result     []prometheusResult `json:"result"`
				}{
					ResultType: "vector",
					Result: []prometheusResult{
						{
							Metric: map[string]string{"source": "db1"},
							Value:  []interface{}{float64(1704067200), "100"},
						},
						{
							Metric: map[string]string{"source": "db2"},
							Value:  []interface{}{float64(1704067200), "30"},
						},
					},
				},
			},
			rule: AlertRule{
				ID:         uuid.New(),
				MetricName: "test_metric",
				Operator:   OpGreaterThan,
				Threshold:  50,
			},
			wantCount:  2,
			wantFiring: []bool{true, false},
			wantErr:    false,
		},
		{
			name: "no results",
			response: prometheusResponse{
				Status: "success",
				Data: struct {
					ResultType string             `json:"resultType"`
					Result     []prometheusResult `json:"result"`
				}{
					ResultType: "vector",
					Result:     []prometheusResult{},
				},
			},
			rule: AlertRule{
				ID:         uuid.New(),
				MetricName: "test_metric",
				Operator:   OpGreaterThan,
				Threshold:  50,
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "prometheus error",
			response: prometheusResponse{
				Status:    "error",
				ErrorType: "bad_data",
				Error:     "invalid query",
			},
			rule: AlertRule{
				ID:         uuid.New(),
				MetricName: "test_metric",
				Operator:   OpGreaterThan,
				Threshold:  50,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response) //nolint:errcheck
			}))
			defer server.Close()

			e := NewEvaluator(server.URL, nil)

			results, err := e.Evaluate(context.Background(), tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) != tt.wantCount {
					t.Errorf("Evaluate() returned %d results, want %d", len(results), tt.wantCount)
				}

				for i, want := range tt.wantFiring {
					if results[i].ShouldFire != want {
						t.Errorf("results[%d].ShouldFire = %v, want %v", i, results[i].ShouldFire, want)
					}
				}
			}
		})
	}
}

func TestEvaluator_Evaluate_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error")) //nolint:errcheck
	}))
	defer server.Close()

	e := NewEvaluator(server.URL, nil)

	rule := AlertRule{
		ID:         uuid.New(),
		MetricName: "test_metric",
		Operator:   OpGreaterThan,
		Threshold:  50,
	}

	_, err := e.Evaluate(context.Background(), rule)
	if err == nil {
		t.Error("Evaluate() should return error on HTTP 500")
	}
}

func TestEvaluator_Evaluate_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json")) //nolint:errcheck
	}))
	defer server.Close()

	e := NewEvaluator(server.URL, nil)

	rule := AlertRule{
		ID:         uuid.New(),
		MetricName: "test_metric",
		Operator:   OpGreaterThan,
		Threshold:  50,
	}

	_, err := e.Evaluate(context.Background(), rule)
	if err == nil {
		t.Error("Evaluate() should return error on invalid JSON")
	}
}

func TestEvaluator_Evaluate_LabelMerging(t *testing.T) {
	response := prometheusResponse{
		Status: "success",
		Data: struct {
			ResultType string             `json:"resultType"`
			Result     []prometheusResult `json:"result"`
		}{
			ResultType: "vector",
			Result: []prometheusResult{
				{
					Metric: map[string]string{"source": "db1", "instance": "localhost:9090"},
					Value:  []interface{}{float64(1704067200), "100"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response) //nolint:errcheck
	}))
	defer server.Close()

	e := NewEvaluator(server.URL, nil)

	rule := AlertRule{
		ID:         uuid.New(),
		MetricName: "test_metric",
		Operator:   OpGreaterThan,
		Threshold:  50,
		Labels:     map[string]string{"env": "prod", "source": "override-me"},
	}

	results, err := e.Evaluate(context.Background(), rule)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Metric labels should take precedence
	if results[0].Labels["source"] != "db1" {
		t.Errorf("source label should be from metric (db1), got %q", results[0].Labels["source"])
	}

	// Rule labels should be included
	if results[0].Labels["env"] != "prod" {
		t.Errorf("env label should be from rule (prod), got %q", results[0].Labels["env"])
	}

	// New metric labels should be included
	if results[0].Labels["instance"] != "localhost:9090" {
		t.Errorf("instance label should be from metric, got %q", results[0].Labels["instance"])
	}
}

func TestEvaluator_Evaluate_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prometheusResponse{Status: "success"})
	}))
	defer server.Close()

	e := NewEvaluator(server.URL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	rule := AlertRule{
		ID:         uuid.New(),
		MetricName: "test_metric",
		Operator:   OpGreaterThan,
		Threshold:  50,
	}

	_, err := e.Evaluate(ctx, rule)
	if err == nil {
		t.Error("Evaluate() should return error on cancelled context")
	}
}

func TestEvaluator_SetHTTPClient(t *testing.T) {
	e := NewEvaluator("http://localhost:9090", nil)

	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	e.SetHTTPClient(customClient)

	if e.httpClient != customClient {
		t.Error("SetHTTPClient did not set the custom client")
	}
}
