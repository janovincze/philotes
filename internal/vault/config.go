// Package vault provides HashiCorp Vault integration for secrets management.
package vault

import "time"

// Config holds Vault client configuration.
type Config struct {
	// Enabled enables Vault integration
	Enabled bool

	// Address is the Vault server URL
	Address string

	// Namespace is the Vault namespace (Enterprise feature)
	Namespace string

	// AuthMethod is the authentication method ("kubernetes" or "token")
	AuthMethod string

	// Role is the Vault role for Kubernetes authentication
	Role string

	// TokenPath is the path to the Kubernetes service account token
	TokenPath string

	// Token is a static Vault token (for development/testing)
	Token string

	// TLSSkipVerify skips TLS certificate verification
	TLSSkipVerify bool

	// CACert is the path to a CA certificate file
	CACert string

	// SecretMountPath is the mount path for the KV secrets engine
	SecretMountPath string

	// TokenRenewalInterval is how often to renew the Vault token
	TokenRenewalInterval time.Duration

	// SecretRefreshInterval is how often to refresh cached secrets
	SecretRefreshInterval time.Duration

	// FallbackToEnv enables fallback to environment variables if Vault is unavailable
	FallbackToEnv bool

	// SecretPaths contains the Vault paths for each secret type
	SecretPaths SecretPaths
}

// SecretPaths defines the Vault paths for different secret types.
type SecretPaths struct {
	// DatabaseBuffer is the path to buffer database credentials
	DatabaseBuffer string

	// DatabaseSource is the path to source database credentials
	DatabaseSource string

	// StorageMinio is the path to MinIO/S3 credentials
	StorageMinio string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:               false,
		Address:               "",
		Namespace:             "",
		AuthMethod:            AuthMethodKubernetes,
		Role:                  "philotes",
		TokenPath:             DefaultTokenPath,
		Token:                 "",
		TLSSkipVerify:         false,
		CACert:                "",
		SecretMountPath:       "secret",
		TokenRenewalInterval:  time.Hour,
		SecretRefreshInterval: 5 * time.Minute,
		FallbackToEnv:         true,
		SecretPaths: SecretPaths{
			DatabaseBuffer: "philotes/database/buffer",
			DatabaseSource: "philotes/database/source",
			StorageMinio:   "philotes/storage/minio",
		},
	}
}

// Authentication method constants.
const (
	// AuthMethodKubernetes uses Kubernetes service account authentication
	AuthMethodKubernetes = "kubernetes"

	// AuthMethodToken uses a static Vault token
	AuthMethodToken = "token"
)

// DefaultTokenPath is the default path to the Kubernetes service account token.
const DefaultTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// Secret key constants for Vault KV secrets.
const (
	// SecretKeyPassword is the key for password fields
	SecretKeyPassword = "password"

	// SecretKeyAccessKey is the key for access key fields
	SecretKeyAccessKey = "access_key"

	// SecretKeySecretKey is the key for secret key fields
	SecretKeySecretKey = "secret_key"

	// SecretKeyUsername is the key for username fields
	SecretKeyUsername = "username"
)
