// Package config provides configuration loading and management for Philotes services.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for Philotes services.
type Config struct {
	// Version is the application version
	Version string

	// Environment is the deployment environment (development, staging, production)
	Environment string

	// API configuration
	API APIConfig

	// Database configuration for the buffer/metadata database
	Database DatabaseConfig

	// CDC configuration
	CDC CDCConfig

	// Iceberg configuration
	Iceberg IcebergConfig

	// MinIO/S3 configuration
	Storage StorageConfig

	// Metrics configuration
	Metrics MetricsConfig

	// Alerting configuration
	Alerting AlertingConfig

	// Scaling configuration
	Scaling ScalingConfig

	// NodeScaling configuration for infrastructure node auto-scaling
	NodeScaling NodeScalingConfig

	// Auth configuration
	Auth AuthConfig

	// Vault configuration for secrets management
	Vault VaultConfig

	// OAuth configuration for cloud providers
	OAuth OAuthConfig

	// OIDC configuration for SSO authentication
	OIDC OIDCConfig

	// Trino configuration for SQL query layer
	Trino TrinoConfig
}

// TrinoConfig holds Trino query engine configuration.
type TrinoConfig struct {
	// Enabled enables the Trino query layer
	Enabled bool

	// URL is the Trino coordinator URL
	URL string

	// Username for Trino authentication
	Username string

	// Password for Trino authentication
	Password string

	// Catalog is the default Iceberg catalog name in Trino
	Catalog string

	// Schema is the default schema name
	Schema string

	// QueryTimeout is the maximum time for queries
	QueryTimeout time.Duration

	// HealthCheckInterval is how often to check Trino health
	HealthCheckInterval time.Duration
}

// OAuthConfig holds OAuth configuration for cloud providers.
type OAuthConfig struct {
	// EncryptionKey is the base64-encoded 32-byte key for encrypting tokens.
	// Generate with: openssl rand -base64 32
	EncryptionKey string

	// BaseURL is the base URL of the Philotes API (for OAuth callbacks).
	// Example: https://philotes.example.com
	BaseURL string

	// AllowedRedirectHosts is a list of allowed hosts for OAuth redirects.
	// If empty, only the host from BaseURL is allowed.
	// Example: ["localhost:3000", "philotes.example.com"]
	AllowedRedirectHosts []string

	// Hetzner OAuth configuration
	Hetzner HetznerOAuthConfig

	// OVH OAuth configuration
	OVH OVHOAuthConfig
}

// HetznerOAuthConfig holds Hetzner Cloud OAuth settings.
type HetznerOAuthConfig struct {
	// ClientID is the OAuth application client ID
	ClientID string
	// ClientSecret is the OAuth application client secret
	ClientSecret string
	// Enabled indicates if Hetzner OAuth is configured
	Enabled bool
}

// OVHOAuthConfig holds OVHcloud OAuth settings.
type OVHOAuthConfig struct {
	// ClientID is the OAuth application client ID
	ClientID string
	// ClientSecret is the OAuth application client secret
	ClientSecret string
	// Enabled indicates if OVH OAuth is configured
	Enabled bool
}

// OIDCConfig holds OIDC/SSO configuration.
type OIDCConfig struct {
	// Enabled enables OIDC authentication
	Enabled bool

	// AllowLocalLogin allows local username/password login when OIDC is enabled
	AllowLocalLogin bool

	// AutoCreateUsers automatically creates users on first OIDC login
	AutoCreateUsers bool

	// DefaultRole is the default role for auto-created users
	DefaultRole string

	// EncryptionKey is the base64-encoded 32-byte key for encrypting OIDC secrets.
	// Generate with: openssl rand -base64 32
	// If empty, falls back to OAuth.EncryptionKey
	EncryptionKey string

	// StateExpiration is how long OIDC states are valid
	StateExpiration time.Duration
}

// VaultConfig holds HashiCorp Vault configuration.
type VaultConfig struct {
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
	SecretPaths VaultSecretPaths
}

// VaultSecretPaths defines the Vault paths for different secret types.
type VaultSecretPaths struct {
	// DatabaseBuffer is the path to buffer database credentials
	DatabaseBuffer string

	// DatabaseSource is the path to source database credentials
	DatabaseSource string

	// StorageMinio is the path to MinIO/S3 credentials
	StorageMinio string
}

// APIConfig holds API server configuration.
type APIConfig struct {
	// ListenAddr is the address to listen on (e.g., ":8080")
	ListenAddr string

	// BaseURL is the external base URL of the API
	BaseURL string

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration

	// CORSOrigins is a list of allowed CORS origins (use "*" for all)
	CORSOrigins []string

	// RateLimitRPS is the rate limit in requests per second
	RateLimitRPS float64

	// RateLimitBurst is the maximum burst size for rate limiting
	RateLimitBurst int
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	// Host is the database host
	Host string

	// Port is the database port
	Port int

	// Name is the database name
	Name string

	// User is the database user
	User string

	// Password is the database password
	Password string

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full)
	SSLMode string

	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
}

// DSN returns the database connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

// CDCConfig holds CDC pipeline configuration.
type CDCConfig struct {
	// BufferSize is the event buffer size
	BufferSize int

	// BatchSize is the batch size for flushing events
	BatchSize int

	// FlushInterval is the interval for flushing events
	FlushInterval time.Duration

	// Source is the source PostgreSQL database configuration
	Source SourceConfig

	// Replication holds replication slot and publication settings
	Replication ReplicationConfig

	// Checkpoint holds checkpointing configuration
	Checkpoint CheckpointConfig

	// Buffer holds buffer database configuration
	Buffer BufferConfig

	// Retry holds retry policy configuration
	Retry RetryConfig

	// DeadLetter holds dead-letter queue configuration
	DeadLetter DeadLetterConfig

	// Health holds health check configuration
	Health HealthConfig

	// Backpressure holds backpressure configuration
	Backpressure BackpressureConfig
}

// RetryConfig holds retry policy configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int

	// InitialInterval is the initial backoff interval
	InitialInterval time.Duration

	// MaxInterval is the maximum backoff interval
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier
	Multiplier float64
}

// DeadLetterConfig holds dead-letter queue configuration.
type DeadLetterConfig struct {
	// Enabled enables the dead-letter queue
	Enabled bool

	// Retention is how long to keep dead-letter events
	Retention time.Duration
}

// HealthConfig holds health check configuration.
type HealthConfig struct {
	// Enabled enables health check endpoints
	Enabled bool

	// ListenAddr is the address for health check endpoints
	ListenAddr string

	// ReadinessTimeout is how long to wait for readiness checks
	ReadinessTimeout time.Duration
}

// BackpressureConfig holds backpressure configuration.
type BackpressureConfig struct {
	// Enabled enables backpressure handling
	Enabled bool

	// HighWatermark is the buffer size threshold to trigger pause
	HighWatermark int

	// LowWatermark is the buffer size threshold to resume processing
	LowWatermark int

	// CheckInterval is how often to check buffer size
	CheckInterval time.Duration
}

// BufferConfig holds buffer database configuration.
type BufferConfig struct {
	// Enabled enables event buffering
	Enabled bool

	// Retention is how long to keep processed events before cleanup
	Retention time.Duration

	// CleanupInterval is how often to run the cleanup job
	CleanupInterval time.Duration
}

// SourceConfig holds the source PostgreSQL database configuration.
type SourceConfig struct {
	// Host is the source database host
	Host string

	// Port is the source database port
	Port int

	// Database is the source database name
	Database string

	// User is the source database user
	User string

	// Password is the source database password
	Password string

	// SSLMode is the SSL mode for the source connection
	SSLMode string
}

// DSN returns the source database connection string.
func (s SourceConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		s.Host, s.Port, s.Database, s.User, s.Password, s.SSLMode,
	)
}

// URL returns the source database connection URL.
func (s SourceConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		s.User, s.Password, s.Host, s.Port, s.Database, s.SSLMode,
	)
}

// ReplicationConfig holds replication slot and publication settings.
type ReplicationConfig struct {
	// SlotName is the name of the replication slot
	SlotName string

	// PublicationName is the name of the publication to subscribe to
	PublicationName string

	// Tables is a list of tables to replicate (empty means all tables in publication)
	Tables []string
}

// CheckpointConfig holds checkpointing configuration.
type CheckpointConfig struct {
	// Enabled enables checkpointing
	Enabled bool

	// Interval is the interval between checkpoints
	Interval time.Duration
}

// IcebergConfig holds Apache Iceberg configuration.
type IcebergConfig struct {
	// CatalogURL is the Lakekeeper REST catalog URL
	CatalogURL string

	// Warehouse is the warehouse name
	Warehouse string
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	// Endpoint is the S3/MinIO endpoint
	Endpoint string

	// AccessKey is the access key
	AccessKey string

	// SecretKey is the secret key
	SecretKey string

	// Bucket is the default bucket name
	Bucket string

	// UseSSL enables SSL for the connection
	UseSSL bool
}

// MetricsConfig holds metrics/observability configuration.
type MetricsConfig struct {
	// Enabled enables metrics collection
	Enabled bool

	// ListenAddr is the address for the metrics endpoint
	ListenAddr string
}

// AlertingConfig holds alerting framework configuration.
type AlertingConfig struct {
	// Enabled enables the alerting framework
	Enabled bool

	// EvaluationInterval is the interval between rule evaluations
	EvaluationInterval time.Duration

	// NotificationTimeout is the timeout for sending notifications
	NotificationTimeout time.Duration

	// PrometheusURL is the URL of the Prometheus server to query metrics from
	PrometheusURL string

	// RetentionDays is the number of days to retain alert history
	RetentionDays int
}

// ScalingConfig holds scaling engine configuration.
type ScalingConfig struct {
	// Enabled enables the scaling engine
	Enabled bool

	// EvaluationInterval is the interval between policy evaluations
	EvaluationInterval time.Duration

	// PrometheusURL is the URL of the Prometheus server to query metrics from
	PrometheusURL string

	// DefaultCooldownSeconds is the default cooldown period between scaling actions
	DefaultCooldownSeconds int

	// DryRun enables dry-run mode where scaling actions are logged but not executed
	DryRun bool

	// ScaleToZero holds scale-to-zero specific configuration
	ScaleToZero ScaleToZeroConfig
}

// ScaleToZeroConfig holds scale-to-zero specific configuration.
type ScaleToZeroConfig struct {
	// DefaultIdleThreshold is the default idle duration before scaling to zero
	DefaultIdleThreshold time.Duration

	// DefaultKeepAliveWindow is the grace period to prevent flapping
	DefaultKeepAliveWindow time.Duration

	// ColdStartTimeout is the maximum time to wait for a cold start
	ColdStartTimeout time.Duration

	// IdleCheckInterval is how often to check idle state
	IdleCheckInterval time.Duration

	// EnableCostTracking enables cost savings tracking
	EnableCostTracking bool
}

// NodeScalingConfig holds node auto-scaling configuration.
type NodeScalingConfig struct {
	// Enabled enables node auto-scaling
	Enabled bool

	// KubeconfigPath is the path to the kubeconfig file (leave empty for in-cluster config)
	KubeconfigPath string

	// NodeJoinTimeout is the timeout for waiting for a node to join the cluster
	NodeJoinTimeout time.Duration

	// NodeDrainTimeout is the timeout for draining a node
	NodeDrainTimeout time.Duration

	// NodeDrainGracePeriod is the grace period for pod eviction during drain
	NodeDrainGracePeriod time.Duration

	// DefaultMinNodes is the default minimum nodes for new pools
	DefaultMinNodes int

	// DefaultMaxNodes is the default maximum nodes for new pools
	DefaultMaxNodes int

	// DefaultImage is the default OS image for new nodes
	DefaultImage string

	// Hetzner cloud provider configuration
	Hetzner HetznerProviderConfig

	// Scaleway cloud provider configuration
	Scaleway ScalewayProviderConfig

	// OVH cloud provider configuration
	OVH OVHProviderConfig

	// Exoscale cloud provider configuration
	Exoscale ExoscaleProviderConfig

	// Contabo cloud provider configuration
	Contabo ContaboProviderConfig
}

// HetznerProviderConfig holds Hetzner Cloud provider configuration.
type HetznerProviderConfig struct {
	// Token is the Hetzner Cloud API token
	Token string
}

// ScalewayProviderConfig holds Scaleway provider configuration.
type ScalewayProviderConfig struct {
	// AccessKey is the Scaleway access key
	AccessKey string
	// SecretKey is the Scaleway secret key
	SecretKey string
	// OrganizationID is the Scaleway organization ID
	OrganizationID string
	// ProjectID is the default Scaleway project ID
	ProjectID string
}

// OVHProviderConfig holds OVHcloud provider configuration.
type OVHProviderConfig struct {
	// Endpoint is the OVH API endpoint (ovh-eu, ovh-us, ovh-ca)
	Endpoint string
	// ApplicationKey is the OVH application key
	ApplicationKey string
	// ApplicationSecret is the OVH application secret
	ApplicationSecret string
	// ConsumerKey is the OVH consumer key
	ConsumerKey string
	// ServiceName is the OVH cloud project service name
	ServiceName string
}

// ExoscaleProviderConfig holds Exoscale provider configuration.
type ExoscaleProviderConfig struct {
	// APIKey is the Exoscale API key
	APIKey string
	// APISecret is the Exoscale API secret
	APISecret string
}

// ContaboProviderConfig holds Contabo provider configuration.
type ContaboProviderConfig struct {
	// ClientID is the Contabo OAuth client ID
	ClientID string
	// ClientSecret is the Contabo OAuth client secret
	ClientSecret string
	// Username is the Contabo account username
	Username string
	// Password is the Contabo account password
	Password string
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// Enabled enables authentication (disabled by default for development)
	Enabled bool

	// JWTSecret is the secret key for signing JWT tokens (min 32 chars)
	JWTSecret string

	// JWTExpiration is the JWT token expiration duration
	JWTExpiration time.Duration

	// APIKeyPrefix is the prefix for generated API keys
	APIKeyPrefix string

	// BCryptCost is the cost factor for bcrypt password hashing
	BCryptCost int

	// AdminEmail is the bootstrap admin user email (created on startup)
	AdminEmail string

	// AdminPassword is the bootstrap admin user password
	AdminPassword string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Version:     getEnv("PHILOTES_VERSION", "0.1.0"),
		Environment: getEnv("PHILOTES_ENV", "development"),

		API: APIConfig{
			ListenAddr:     getEnv("PHILOTES_API_LISTEN_ADDR", ":8080"),
			BaseURL:        getEnv("PHILOTES_API_BASE_URL", "http://localhost:8080"),
			ReadTimeout:    getDurationEnv("PHILOTES_API_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:   getDurationEnv("PHILOTES_API_WRITE_TIMEOUT", 15*time.Second),
			CORSOrigins:    getSliceEnv("PHILOTES_API_CORS_ORIGINS", []string{"*"}),
			RateLimitRPS:   getFloatEnv("PHILOTES_API_RATE_LIMIT_RPS", 100),
			RateLimitBurst: getIntEnv("PHILOTES_API_RATE_LIMIT_BURST", 200),
		},

		Database: DatabaseConfig{
			Host:         getEnv("PHILOTES_DB_HOST", "localhost"),
			Port:         getIntEnv("PHILOTES_DB_PORT", 5432),
			Name:         getEnv("PHILOTES_DB_NAME", "philotes"),
			User:         getEnv("PHILOTES_DB_USER", "philotes"),
			Password:     getEnv("PHILOTES_DB_PASSWORD", "philotes"),
			SSLMode:      getEnv("PHILOTES_DB_SSLMODE", "disable"),
			MaxOpenConns: getIntEnv("PHILOTES_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getIntEnv("PHILOTES_DB_MAX_IDLE_CONNS", 5),
		},

		CDC: CDCConfig{
			BufferSize:    getIntEnv("PHILOTES_CDC_BUFFER_SIZE", 10000),
			BatchSize:     getIntEnv("PHILOTES_CDC_BATCH_SIZE", 1000),
			FlushInterval: getDurationEnv("PHILOTES_CDC_FLUSH_INTERVAL", 5*time.Second),
			Source: SourceConfig{
				Host:     getEnv("PHILOTES_CDC_SOURCE_HOST", "localhost"),
				Port:     getIntEnv("PHILOTES_CDC_SOURCE_PORT", 5433),
				Database: getEnv("PHILOTES_CDC_SOURCE_DATABASE", "source"),
				User:     getEnv("PHILOTES_CDC_SOURCE_USER", "source"),
				Password: getEnv("PHILOTES_CDC_SOURCE_PASSWORD", "source"),
				SSLMode:  getEnv("PHILOTES_CDC_SOURCE_SSLMODE", "disable"),
			},
			Replication: ReplicationConfig{
				SlotName:        getEnv("PHILOTES_CDC_REPLICATION_SLOT", "philotes_cdc"),
				PublicationName: getEnv("PHILOTES_CDC_PUBLICATION", "philotes_pub"),
				Tables:          getSliceEnv("PHILOTES_CDC_TABLES", nil),
			},
			Checkpoint: CheckpointConfig{
				Enabled:  getBoolEnv("PHILOTES_CDC_CHECKPOINT_ENABLED", true),
				Interval: getDurationEnv("PHILOTES_CDC_CHECKPOINT_INTERVAL", 10*time.Second),
			},
			Buffer: BufferConfig{
				Enabled:         getBoolEnv("PHILOTES_BUFFER_ENABLED", true),
				Retention:       getDurationEnv("PHILOTES_BUFFER_RETENTION", 168*time.Hour), // 7 days
				CleanupInterval: getDurationEnv("PHILOTES_BUFFER_CLEANUP_INTERVAL", time.Hour),
			},
			Retry: RetryConfig{
				MaxAttempts:     getIntEnv("PHILOTES_RETRY_MAX_ATTEMPTS", 3),
				InitialInterval: getDurationEnv("PHILOTES_RETRY_INITIAL_INTERVAL", time.Second),
				MaxInterval:     getDurationEnv("PHILOTES_RETRY_MAX_INTERVAL", 30*time.Second),
				Multiplier:      getFloatEnv("PHILOTES_RETRY_MULTIPLIER", 2.0),
			},
			DeadLetter: DeadLetterConfig{
				Enabled:   getBoolEnv("PHILOTES_DLQ_ENABLED", true),
				Retention: getDurationEnv("PHILOTES_DLQ_RETENTION", 168*time.Hour), // 7 days
			},
			Health: HealthConfig{
				Enabled:          getBoolEnv("PHILOTES_HEALTH_ENABLED", true),
				ListenAddr:       getEnv("PHILOTES_HEALTH_LISTEN_ADDR", ":8081"),
				ReadinessTimeout: getDurationEnv("PHILOTES_HEALTH_READINESS_TIMEOUT", 5*time.Second),
			},
			Backpressure: BackpressureConfig{
				Enabled:       getBoolEnv("PHILOTES_BACKPRESSURE_ENABLED", true),
				HighWatermark: getIntEnv("PHILOTES_BACKPRESSURE_HIGH_WATERMARK", 8000),
				LowWatermark:  getIntEnv("PHILOTES_BACKPRESSURE_LOW_WATERMARK", 5000),
				CheckInterval: getDurationEnv("PHILOTES_BACKPRESSURE_CHECK_INTERVAL", time.Second),
			},
		},

		Iceberg: IcebergConfig{
			CatalogURL: getEnv("PHILOTES_ICEBERG_CATALOG_URL", "http://localhost:8181"),
			Warehouse:  getEnv("PHILOTES_ICEBERG_WAREHOUSE", "philotes"),
		},

		Storage: StorageConfig{
			Endpoint:  getEnv("PHILOTES_STORAGE_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("PHILOTES_STORAGE_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("PHILOTES_STORAGE_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("PHILOTES_STORAGE_BUCKET", "philotes"),
			UseSSL:    getBoolEnv("PHILOTES_STORAGE_USE_SSL", false),
		},

		Metrics: MetricsConfig{
			Enabled:    getBoolEnv("PHILOTES_METRICS_ENABLED", true),
			ListenAddr: getEnv("PHILOTES_METRICS_LISTEN_ADDR", ":9090"),
		},

		Alerting: AlertingConfig{
			Enabled:             getBoolEnv("PHILOTES_ALERTING_ENABLED", true),
			EvaluationInterval:  getDurationEnv("PHILOTES_ALERTING_EVALUATION_INTERVAL", 30*time.Second),
			NotificationTimeout: getDurationEnv("PHILOTES_ALERTING_NOTIFICATION_TIMEOUT", 10*time.Second),
			PrometheusURL:       getEnv("PHILOTES_PROMETHEUS_URL", "http://localhost:9090"),
			RetentionDays:       getIntEnv("PHILOTES_ALERTING_RETENTION_DAYS", 30),
		},

		Scaling: ScalingConfig{
			Enabled:                getBoolEnv("PHILOTES_SCALING_ENABLED", true),
			EvaluationInterval:     getDurationEnv("PHILOTES_SCALING_EVALUATION_INTERVAL", 30*time.Second),
			PrometheusURL:          getEnv("PHILOTES_PROMETHEUS_URL", "http://localhost:9090"),
			DefaultCooldownSeconds: getIntEnv("PHILOTES_SCALING_DEFAULT_COOLDOWN", 300),
			DryRun:                 getBoolEnv("PHILOTES_SCALING_DRY_RUN", false),
			ScaleToZero: ScaleToZeroConfig{
				DefaultIdleThreshold:   getDurationEnv("PHILOTES_SCALE_TO_ZERO_IDLE_THRESHOLD", 30*time.Minute),
				DefaultKeepAliveWindow: getDurationEnv("PHILOTES_SCALE_TO_ZERO_KEEP_ALIVE", 5*time.Minute),
				ColdStartTimeout:       getDurationEnv("PHILOTES_SCALE_TO_ZERO_COLD_START_TIMEOUT", 2*time.Minute),
				IdleCheckInterval:      getDurationEnv("PHILOTES_SCALE_TO_ZERO_CHECK_INTERVAL", 1*time.Minute),
				EnableCostTracking:     getBoolEnv("PHILOTES_SCALE_TO_ZERO_COST_TRACKING", true),
			},
		},

		NodeScaling: NodeScalingConfig{
			Enabled:              getBoolEnv("PHILOTES_NODE_SCALING_ENABLED", false),
			KubeconfigPath:       getEnv("PHILOTES_KUBECONFIG", ""),
			NodeJoinTimeout:      getDurationEnv("PHILOTES_NODE_JOIN_TIMEOUT", 10*time.Minute),
			NodeDrainTimeout:     getDurationEnv("PHILOTES_NODE_DRAIN_TIMEOUT", 5*time.Minute),
			NodeDrainGracePeriod: getDurationEnv("PHILOTES_NODE_DRAIN_GRACE_PERIOD", 30*time.Second),
			DefaultMinNodes:      getIntEnv("PHILOTES_NODE_DEFAULT_MIN", 1),
			DefaultMaxNodes:      getIntEnv("PHILOTES_NODE_DEFAULT_MAX", 10),
			DefaultImage:         getEnv("PHILOTES_NODE_DEFAULT_IMAGE", "ubuntu-24.04"),
			Hetzner: HetznerProviderConfig{
				Token: getEnv("PHILOTES_HETZNER_TOKEN", ""),
			},
			Scaleway: ScalewayProviderConfig{
				AccessKey:      getEnv("PHILOTES_SCALEWAY_ACCESS_KEY", ""),
				SecretKey:      getEnv("PHILOTES_SCALEWAY_SECRET_KEY", ""),
				OrganizationID: getEnv("PHILOTES_SCALEWAY_ORGANIZATION_ID", ""),
				ProjectID:      getEnv("PHILOTES_SCALEWAY_PROJECT_ID", ""),
			},
			OVH: OVHProviderConfig{
				Endpoint:          getEnv("PHILOTES_OVH_ENDPOINT", "ovh-eu"),
				ApplicationKey:    getEnv("PHILOTES_OVH_APPLICATION_KEY", ""),
				ApplicationSecret: getEnv("PHILOTES_OVH_APPLICATION_SECRET", ""),
				ConsumerKey:       getEnv("PHILOTES_OVH_CONSUMER_KEY", ""),
				ServiceName:       getEnv("PHILOTES_OVH_SERVICE_NAME", ""),
			},
			Exoscale: ExoscaleProviderConfig{
				APIKey:    getEnv("PHILOTES_EXOSCALE_API_KEY", ""),
				APISecret: getEnv("PHILOTES_EXOSCALE_API_SECRET", ""),
			},
			Contabo: ContaboProviderConfig{
				ClientID:     getEnv("PHILOTES_CONTABO_CLIENT_ID", ""),
				ClientSecret: getEnv("PHILOTES_CONTABO_CLIENT_SECRET", ""),
				Username:     getEnv("PHILOTES_CONTABO_USERNAME", ""),
				Password:     getEnv("PHILOTES_CONTABO_PASSWORD", ""),
			},
		},

		Auth: AuthConfig{
			Enabled:       getBoolEnv("PHILOTES_AUTH_ENABLED", false),
			JWTSecret:     getEnv("PHILOTES_AUTH_JWT_SECRET", ""),
			JWTExpiration: getDurationEnv("PHILOTES_AUTH_JWT_EXPIRATION", 24*time.Hour),
			APIKeyPrefix:  getEnv("PHILOTES_AUTH_API_KEY_PREFIX", "pk_"),
			BCryptCost:    getIntEnv("PHILOTES_AUTH_BCRYPT_COST", 12),
			AdminEmail:    getEnv("PHILOTES_AUTH_ADMIN_EMAIL", ""),
			AdminPassword: getEnv("PHILOTES_AUTH_ADMIN_PASSWORD", ""),
		},

		Vault: VaultConfig{
			Enabled:               getBoolEnv("PHILOTES_VAULT_ENABLED", false),
			Address:               getEnv("PHILOTES_VAULT_ADDRESS", ""),
			Namespace:             getEnv("PHILOTES_VAULT_NAMESPACE", ""),
			AuthMethod:            getEnv("PHILOTES_VAULT_AUTH_METHOD", "kubernetes"),
			Role:                  getEnv("PHILOTES_VAULT_ROLE", "philotes"),
			TokenPath:             getEnv("PHILOTES_VAULT_TOKEN_PATH", "/var/run/secrets/kubernetes.io/serviceaccount/token"),
			Token:                 getEnv("PHILOTES_VAULT_TOKEN", ""),
			TLSSkipVerify:         getBoolEnv("PHILOTES_VAULT_TLS_SKIP_VERIFY", false),
			CACert:                getEnv("PHILOTES_VAULT_CA_CERT", ""),
			SecretMountPath:       getEnv("PHILOTES_VAULT_SECRET_MOUNT_PATH", "secret"),
			TokenRenewalInterval:  getDurationEnv("PHILOTES_VAULT_TOKEN_RENEWAL_INTERVAL", time.Hour),
			SecretRefreshInterval: getDurationEnv("PHILOTES_VAULT_SECRET_REFRESH_INTERVAL", 5*time.Minute),
			FallbackToEnv:         getBoolEnv("PHILOTES_VAULT_FALLBACK_TO_ENV", true),
			SecretPaths: VaultSecretPaths{
				DatabaseBuffer: getEnv("PHILOTES_VAULT_SECRET_PATH_DATABASE_BUFFER", "philotes/database/buffer"),
				DatabaseSource: getEnv("PHILOTES_VAULT_SECRET_PATH_DATABASE_SOURCE", "philotes/database/source"),
				StorageMinio:   getEnv("PHILOTES_VAULT_SECRET_PATH_STORAGE_MINIO", "philotes/storage/minio"),
			},
		},

		OAuth: OAuthConfig{
			EncryptionKey: getEnv("PHILOTES_OAUTH_ENCRYPTION_KEY", ""),
			BaseURL:       getEnv("PHILOTES_OAUTH_BASE_URL", getEnv("PHILOTES_API_BASE_URL", "http://localhost:8080")),
			Hetzner: HetznerOAuthConfig{
				ClientID:     getEnv("PHILOTES_OAUTH_HETZNER_CLIENT_ID", ""),
				ClientSecret: getEnv("PHILOTES_OAUTH_HETZNER_CLIENT_SECRET", ""),
				Enabled:      getEnv("PHILOTES_OAUTH_HETZNER_CLIENT_ID", "") != "",
			},
			OVH: OVHOAuthConfig{
				ClientID:     getEnv("PHILOTES_OAUTH_OVH_CLIENT_ID", ""),
				ClientSecret: getEnv("PHILOTES_OAUTH_OVH_CLIENT_SECRET", ""),
				Enabled:      getEnv("PHILOTES_OAUTH_OVH_CLIENT_ID", "") != "",
			},
		},

		OIDC: OIDCConfig{
			Enabled:         getBoolEnv("PHILOTES_OIDC_ENABLED", false),
			AllowLocalLogin: getBoolEnv("PHILOTES_OIDC_ALLOW_LOCAL_LOGIN", true),
			AutoCreateUsers: getBoolEnv("PHILOTES_OIDC_AUTO_CREATE_USERS", true),
			DefaultRole:     getEnv("PHILOTES_OIDC_DEFAULT_ROLE", "viewer"),
			EncryptionKey:   getEnv("PHILOTES_OIDC_ENCRYPTION_KEY", ""),
			StateExpiration: getDurationEnv("PHILOTES_OIDC_STATE_EXPIRATION", 10*time.Minute),
		},

		Trino: TrinoConfig{
			Enabled:             getBoolEnv("PHILOTES_TRINO_ENABLED", false),
			URL:                 getEnv("PHILOTES_TRINO_URL", "http://localhost:8085"),
			Username:            getEnv("PHILOTES_TRINO_USERNAME", ""),
			Password:            getEnv("PHILOTES_TRINO_PASSWORD", ""),
			Catalog:             getEnv("PHILOTES_TRINO_CATALOG", "iceberg"),
			Schema:              getEnv("PHILOTES_TRINO_SCHEMA", "philotes"),
			QueryTimeout:        getDurationEnv("PHILOTES_TRINO_QUERY_TIMEOUT", 5*time.Minute),
			HealthCheckInterval: getDurationEnv("PHILOTES_TRINO_HEALTH_CHECK_INTERVAL", 30*time.Second),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getFloatEnv(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		var result []string
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
