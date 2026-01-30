// Package installer provides error handling with troubleshooting suggestions.
package installer

import (
	"strings"
)

// StepError contains error details with troubleshooting suggestions.
type StepError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
	// Details provides additional context about the error.
	Details string `json:"details,omitempty"`
	// Suggestions are troubleshooting steps the user can try.
	Suggestions []string `json:"suggestions"`
	// Retryable indicates if the operation can be retried.
	Retryable bool `json:"retryable"`
	// DocsURL links to relevant documentation.
	DocsURL string `json:"docs_url,omitempty"`
}

// ErrorPattern defines a pattern to match and its corresponding error info.
type ErrorPattern struct {
	// Patterns are substrings to match in error messages.
	Patterns []string
	// Error is the structured error to return.
	Error StepError
}

// errorPatterns maps error patterns to helpful suggestions.
var errorPatterns = []ErrorPattern{
	// Authentication errors
	{
		Patterns: []string{"unauthorized", "401", "invalid token", "authentication failed", "invalid api key"},
		Error: StepError{
			Code:    "AUTH_FAILED",
			Message: "Cloud provider authentication failed",
			Suggestions: []string{
				"Verify your API token is correct and has not expired",
				"Check that your API token has the required permissions",
				"If using OAuth, try re-authorizing the connection",
				"Ensure the API key was copied completely without extra spaces",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/authentication",
		},
	},
	// Quota/limit errors
	{
		Patterns: []string{"quota exceeded", "limit reached", "resource limit", "insufficient quota", "maximum number"},
		Error: StepError{
			Code:    "QUOTA_EXCEEDED",
			Message: "Cloud provider quota or resource limit exceeded",
			Suggestions: []string{
				"Request a quota increase from your cloud provider",
				"Try a smaller deployment size",
				"Check if you have other resources consuming quota",
				"Delete unused resources in your account",
			},
			Retryable: false,
			DocsURL:   "https://docs.philotes.io/troubleshooting/quotas",
		},
	},
	// Timeout errors
	{
		Patterns: []string{"timeout", "timed out", "deadline exceeded", "context deadline"},
		Error: StepError{
			Code:    "TIMEOUT",
			Message: "Operation timed out",
			Suggestions: []string{
				"The cloud provider may be experiencing delays",
				"Check your internet connection",
				"Try again in a few minutes",
				"Consider using a region closer to your location",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/timeouts",
		},
	},
	// Network/connectivity errors
	{
		Patterns: []string{"connection refused", "network unreachable", "no route to host", "connection reset", "dns"},
		Error: StepError{
			Code:    "NETWORK_ERROR",
			Message: "Network connectivity issue",
			Suggestions: []string{
				"Check your internet connection",
				"Verify the cloud provider's API endpoint is accessible",
				"Check if a firewall is blocking outbound connections",
				"Try using a different network or VPN",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/network",
		},
	},
	// Permission/access errors
	{
		Patterns: []string{"forbidden", "403", "permission denied", "access denied", "not authorized"},
		Error: StepError{
			Code:    "PERMISSION_DENIED",
			Message: "Insufficient permissions",
			Suggestions: []string{
				"Verify your API token has the required permissions",
				"Check if your account has access to the requested region",
				"Ensure you have permission to create the requested resource types",
				"Contact your cloud provider to enable the required APIs",
			},
			Retryable: false,
			DocsURL:   "https://docs.philotes.io/troubleshooting/permissions",
		},
	},
	// Resource not found
	{
		Patterns: []string{"not found", "404", "does not exist", "no such"},
		Error: StepError{
			Code:    "RESOURCE_NOT_FOUND",
			Message: "Required resource not found",
			Suggestions: []string{
				"Verify the region is correct and available",
				"Check if the requested instance type exists in the selected region",
				"Ensure prerequisite resources have been created",
			},
			Retryable: false,
			DocsURL:   "https://docs.philotes.io/troubleshooting/resources",
		},
	},
	// Resource conflict/already exists
	{
		Patterns: []string{"already exists", "conflict", "duplicate", "name taken"},
		Error: StepError{
			Code:    "RESOURCE_CONFLICT",
			Message: "Resource already exists",
			Suggestions: []string{
				"Use a different name for your deployment",
				"Delete the existing resource if it's no longer needed",
				"Check if a previous deployment with this name exists",
			},
			Retryable: false,
			DocsURL:   "https://docs.philotes.io/troubleshooting/conflicts",
		},
	},
	// Rate limiting
	{
		Patterns: []string{"rate limit", "too many requests", "429", "throttl"},
		Error: StepError{
			Code:    "RATE_LIMITED",
			Message: "Request rate limit exceeded",
			Suggestions: []string{
				"Wait a few minutes before retrying",
				"The cloud provider is rate limiting requests",
				"Consider spacing out deployment requests",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/rate-limits",
		},
	},
	// Server/infrastructure errors
	{
		Patterns: []string{"server error", "500", "internal error", "service unavailable", "503"},
		Error: StepError{
			Code:    "PROVIDER_ERROR",
			Message: "Cloud provider internal error",
			Suggestions: []string{
				"The cloud provider is experiencing issues",
				"Check the provider's status page for outages",
				"Try again in a few minutes",
				"Consider trying a different region",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/provider-errors",
		},
	},
	// SSH/connection errors for K3s setup
	{
		Patterns: []string{"ssh", "connection closed", "host key", "known_hosts"},
		Error: StepError{
			Code:    "SSH_ERROR",
			Message: "SSH connection to server failed",
			Suggestions: []string{
				"The server may still be initializing",
				"Check if the SSH key was correctly configured",
				"Verify the server's firewall allows SSH connections",
				"Wait a moment and retry the deployment",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/ssh",
		},
	},
	// Kubernetes/K3s errors
	{
		Patterns: []string{"kubernetes", "k3s", "kubectl", "pod", "deployment", "helm"},
		Error: StepError{
			Code:    "K8S_ERROR",
			Message: "Kubernetes deployment error",
			Suggestions: []string{
				"Check if the cluster has sufficient resources",
				"Verify the Kubernetes nodes are healthy",
				"Check for any failed pods or deployments",
				"Review the Helm chart values for any issues",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/kubernetes",
		},
	},
	// Storage/volume errors
	{
		Patterns: []string{"volume", "storage", "disk", "pvc", "persistent"},
		Error: StepError{
			Code:    "STORAGE_ERROR",
			Message: "Storage provisioning error",
			Suggestions: []string{
				"Check if storage is available in the selected region",
				"Verify you have quota for additional storage",
				"Try reducing the storage size",
				"Check for any existing claims on the storage",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/storage",
		},
	},
	// SSL/TLS certificate errors
	{
		Patterns: []string{"certificate", "ssl", "tls", "cert-manager", "acme", "let's encrypt"},
		Error: StepError{
			Code:    "SSL_ERROR",
			Message: "SSL/TLS certificate error",
			Suggestions: []string{
				"Verify your domain DNS is correctly configured",
				"Check if the domain points to the load balancer IP",
				"Ensure ports 80 and 443 are accessible for ACME challenges",
				"Wait for DNS propagation if recently updated",
			},
			Retryable: true,
			DocsURL:   "https://docs.philotes.io/troubleshooting/ssl",
		},
	},
}

// stepSpecificErrors provides additional context based on the deployment step.
var stepSpecificErrors = map[string][]string{
	"auth": {
		"Ensure you've completed the OAuth flow or entered valid API credentials",
		"Check that your cloud provider account is active and in good standing",
	},
	"network": {
		"Verify your cloud provider account can create VPCs and subnets",
		"Check if there are any existing networks with conflicting CIDR ranges",
	},
	"compute": {
		"Verify the selected instance types are available in your region",
		"Check if you have sufficient quota for the number of instances",
	},
	"k3s": {
		"The K3s installation requires servers to be fully provisioned first",
		"Check if the servers have internet access for downloading K3s",
	},
	"storage": {
		"MinIO requires block storage to be available",
		"Verify the storage class is supported in your cluster",
	},
	"catalog": {
		"Lakekeeper requires MinIO to be running",
		"Check if the Postgres database is accessible",
	},
	"philotes": {
		"Ensure all prerequisite services (MinIO, Lakekeeper) are running",
		"Check if the Helm chart version is compatible",
	},
	"health": {
		"Some services may need additional time to initialize",
		"Check pod logs for any startup errors",
	},
	"ssl": {
		"SSL certificate issuance requires valid DNS configuration",
		"Let's Encrypt rate limits may apply if certificates were recently issued",
	},
}

// GetErrorSuggestion analyzes an error and returns helpful suggestions.
func GetErrorSuggestion(err error, step string) *StepError {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())

	// Find matching error pattern
	for _, pattern := range errorPatterns {
		for _, p := range pattern.Patterns {
			if strings.Contains(errStr, p) {
				// Clone the error to avoid modifying the original
				result := pattern.Error
				result.Details = err.Error()

				// Add step-specific suggestions
				if stepSuggestions, ok := stepSpecificErrors[step]; ok {
					result.Suggestions = append(result.Suggestions, stepSuggestions...)
				}

				return &result
			}
		}
	}

	// Return a generic error if no pattern matches
	genericError := &StepError{
		Code:    "UNKNOWN_ERROR",
		Message: "An unexpected error occurred",
		Details: err.Error(),
		Suggestions: []string{
			"Check the deployment logs for more details",
			"Try canceling and restarting the deployment",
			"If the issue persists, contact support",
		},
		Retryable: true,
		DocsURL:   "https://docs.philotes.io/troubleshooting",
	}

	// Add step-specific suggestions
	if stepSuggestions, ok := stepSpecificErrors[step]; ok {
		genericError.Suggestions = append(genericError.Suggestions, stepSuggestions...)
	}

	return genericError
}

// IsRetryableError checks if an error is retryable based on its pattern.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	for _, pattern := range errorPatterns {
		for _, p := range pattern.Patterns {
			if strings.Contains(errStr, p) {
				return pattern.Error.Retryable
			}
		}
	}

	// Default to retryable for unknown errors
	return true
}
