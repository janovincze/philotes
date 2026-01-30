package installer

import (
	"errors"
	"testing"
)

func TestGetErrorSuggestion(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		step         string
		expectedCode string
		retryable    bool
	}{
		{
			name:         "authentication error",
			err:          errors.New("authentication failed: invalid api key"),
			step:         "auth",
			expectedCode: "AUTH_FAILED",
			retryable:    true,
		},
		{
			name:         "unauthorized 401",
			err:          errors.New("request failed with status 401"),
			step:         "auth",
			expectedCode: "AUTH_FAILED",
			retryable:    true,
		},
		{
			name:         "quota exceeded",
			err:          errors.New("quota exceeded for instance creation"),
			step:         "compute",
			expectedCode: "QUOTA_EXCEEDED",
			retryable:    false,
		},
		{
			name:         "resource limit",
			err:          errors.New("resource limit reached"),
			step:         "network",
			expectedCode: "QUOTA_EXCEEDED",
			retryable:    false,
		},
		{
			name:         "timeout error",
			err:          errors.New("operation timed out"),
			step:         "compute",
			expectedCode: "TIMEOUT",
			retryable:    true,
		},
		{
			name:         "context deadline",
			err:          errors.New("context deadline exceeded"),
			step:         "k3s",
			expectedCode: "TIMEOUT",
			retryable:    true,
		},
		{
			name:         "network error",
			err:          errors.New("connection refused"),
			step:         "auth",
			expectedCode: "NETWORK_ERROR",
			retryable:    true,
		},
		{
			name:         "DNS error",
			err:          errors.New("dns lookup failed"),
			step:         "auth",
			expectedCode: "NETWORK_ERROR",
			retryable:    true,
		},
		{
			name:         "permission denied",
			err:          errors.New("forbidden: permission denied"),
			step:         "compute",
			expectedCode: "PERMISSION_DENIED",
			retryable:    false,
		},
		{
			name:         "403 error",
			err:          errors.New("request failed with status 403"),
			step:         "storage",
			expectedCode: "PERMISSION_DENIED",
			retryable:    false,
		},
		{
			name:         "resource not found",
			err:          errors.New("resource not found"),
			step:         "compute",
			expectedCode: "RESOURCE_NOT_FOUND",
			retryable:    false,
		},
		{
			name:         "404 error",
			err:          errors.New("request failed with status 404"),
			step:         "network",
			expectedCode: "RESOURCE_NOT_FOUND",
			retryable:    false,
		},
		{
			name:         "resource conflict",
			err:          errors.New("resource already exists"),
			step:         "network",
			expectedCode: "RESOURCE_CONFLICT",
			retryable:    false,
		},
		{
			name:         "duplicate name",
			err:          errors.New("name taken: myresource"),
			step:         "compute",
			expectedCode: "RESOURCE_CONFLICT",
			retryable:    false,
		},
		{
			name:         "rate limit",
			err:          errors.New("rate limit exceeded"),
			step:         "auth",
			expectedCode: "RATE_LIMITED",
			retryable:    true,
		},
		{
			name:         "429 error",
			err:          errors.New("request failed with status 429"),
			step:         "compute",
			expectedCode: "RATE_LIMITED",
			retryable:    true,
		},
		{
			name:         "throttled",
			err:          errors.New("request was throttled"),
			step:         "network",
			expectedCode: "RATE_LIMITED",
			retryable:    true,
		},
		{
			name:         "server error",
			err:          errors.New("internal server error"),
			step:         "compute",
			expectedCode: "PROVIDER_ERROR",
			retryable:    true,
		},
		{
			name:         "503 error",
			err:          errors.New("service unavailable (503)"),
			step:         "storage",
			expectedCode: "PROVIDER_ERROR",
			retryable:    true,
		},
		{
			name:         "SSH error",
			err:          errors.New("ssh connection failed"),
			step:         "k3s",
			expectedCode: "SSH_ERROR",
			retryable:    true,
		},
		{
			name:         "host key error",
			err:          errors.New("host key verification failed"),
			step:         "k3s",
			expectedCode: "SSH_ERROR",
			retryable:    true,
		},
		{
			name:         "kubernetes error",
			err:          errors.New("kubectl apply failed"),
			step:         "philotes",
			expectedCode: "K8S_ERROR",
			retryable:    true,
		},
		{
			name:         "helm error",
			err:          errors.New("helm install failed"),
			step:         "catalog",
			expectedCode: "K8S_ERROR",
			retryable:    true,
		},
		{
			name:         "storage error",
			err:          errors.New("volume creation failed"),
			step:         "storage",
			expectedCode: "STORAGE_ERROR",
			retryable:    true,
		},
		{
			name:         "pvc error",
			err:          errors.New("pvc binding failed"),
			step:         "storage",
			expectedCode: "STORAGE_ERROR",
			retryable:    true,
		},
		{
			name:         "SSL error",
			err:          errors.New("certificate issuance failed"),
			step:         "ssl",
			expectedCode: "SSL_ERROR",
			retryable:    true,
		},
		{
			name:         "ACME error",
			err:          errors.New("acme challenge failed"),
			step:         "ssl",
			expectedCode: "SSL_ERROR",
			retryable:    true,
		},
		{
			name:         "unknown error",
			err:          errors.New("something completely unexpected"),
			step:         "compute",
			expectedCode: "UNKNOWN_ERROR",
			retryable:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetErrorSuggestion(tt.err, tt.step)

			if result == nil {
				t.Fatal("GetErrorSuggestion() returned nil")
			}

			if result.Code != tt.expectedCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.expectedCode)
			}

			if result.Retryable != tt.retryable {
				t.Errorf("Retryable = %v, want %v", result.Retryable, tt.retryable)
			}

			// Should have error details
			if result.Details == "" {
				t.Error("Details should not be empty")
			}

			// Should have suggestions
			if len(result.Suggestions) == 0 {
				t.Error("Suggestions should not be empty")
			}

			// Should have a message
			if result.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestGetErrorSuggestion_Nil(t *testing.T) {
	result := GetErrorSuggestion(nil, "auth")
	if result != nil {
		t.Error("GetErrorSuggestion(nil) should return nil")
	}
}

func TestGetErrorSuggestion_StepSpecificSuggestions(t *testing.T) {
	steps := []string{"auth", "network", "compute", "k3s", "storage", "catalog", "philotes", "health", "ssl"}

	err := errors.New("something went wrong")

	for _, step := range steps {
		t.Run(step, func(t *testing.T) {
			result := GetErrorSuggestion(err, step)

			// Step-specific suggestions should be added
			// The unknown error case gets step-specific suggestions appended
			found := false
			for _, suggestion := range result.Suggestions {
				// Check if any step-specific suggestion was added
				if stepSuggestions, ok := stepSpecificErrors[step]; ok {
					for _, expected := range stepSuggestions {
						if suggestion == expected {
							found = true
							break
						}
					}
				}
			}

			if !found && len(stepSpecificErrors[step]) > 0 {
				t.Errorf("step-specific suggestions not found for step %s", step)
			}
		})
	}
}

func TestGetErrorSuggestion_CaseInsensitive(t *testing.T) {
	tests := []struct {
		err          error
		expectedCode string
	}{
		{errors.New("UNAUTHORIZED"), "AUTH_FAILED"},
		{errors.New("UnAuThOrIzEd"), "AUTH_FAILED"},
		{errors.New("TIMEOUT"), "TIMEOUT"},
		{errors.New("TimeOut"), "TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			result := GetErrorSuggestion(tt.err, "auth")
			if result.Code != tt.expectedCode {
				t.Errorf("Code = %s, want %s", result.Code, tt.expectedCode)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"auth error", errors.New("unauthorized"), true},
		{"quota error", errors.New("quota exceeded"), false},
		{"timeout error", errors.New("timed out"), true},
		{"network error", errors.New("connection refused"), true},
		{"permission error", errors.New("forbidden"), false},
		{"not found error", errors.New("not found"), false},
		{"conflict error", errors.New("already exists"), false},
		{"rate limit error", errors.New("rate limit"), true},
		{"server error", errors.New("500 internal error"), true},
		{"ssh error", errors.New("ssh failed"), true},
		{"k8s error", errors.New("kubernetes error"), true},
		{"storage error", errors.New("volume error"), true},
		{"ssl error", errors.New("certificate error"), true},
		{"unknown error", errors.New("something random"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestErrorPatterns_Coverage(t *testing.T) {
	// Ensure all error patterns have required fields
	for i, pattern := range errorPatterns {
		if len(pattern.Patterns) == 0 {
			t.Errorf("error pattern %d has no patterns", i)
		}

		if pattern.Error.Code == "" {
			t.Errorf("error pattern %d has empty code", i)
		}

		if pattern.Error.Message == "" {
			t.Errorf("error pattern %d has empty message", i)
		}

		if len(pattern.Error.Suggestions) == 0 {
			t.Errorf("error pattern %d has no suggestions", i)
		}
	}
}

func TestStepSpecificErrors_Coverage(t *testing.T) {
	// Ensure all deployment steps have specific suggestions
	expectedSteps := []string{"auth", "network", "compute", "k3s", "storage", "catalog", "philotes", "health", "ssl"}

	for _, step := range expectedSteps {
		if suggestions, ok := stepSpecificErrors[step]; !ok {
			t.Errorf("missing step-specific errors for step: %s", step)
		} else if len(suggestions) == 0 {
			t.Errorf("empty suggestions for step: %s", step)
		}
	}
}
