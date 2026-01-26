# Implementation Plan - Issue #17: HashiCorp Vault Integration

## Overview

Implement HashiCorp Vault integration for secrets management in Philotes services. The implementation uses a hybrid approach: direct Vault SDK integration with Kubernetes authentication, plus optional Vault Agent sidecar support for environments that prefer that pattern.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Philotes Service                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   Config    │───>│ VaultClient │───>│  Secret Provider    │  │
│  │   Loader    │    │             │    │  (Vault or Env)     │  │
│  └─────────────┘    └──────┬──────┘    └─────────────────────┘  │
│                            │                                     │
│                   ┌────────▼────────┐                           │
│                   │ K8s Auth        │                           │
│                   │ (SA Token)      │                           │
│                   └────────┬────────┘                           │
└────────────────────────────┼────────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  HashiCorp      │
                    │  Vault Server   │
                    └─────────────────┘
```

## Phase 1: Vault Client Package

### 1.1 Create `internal/vault/config.go`

```go
type Config struct {
    Enabled              bool
    Address              string        // Vault server URL
    Namespace            string        // Vault namespace (Enterprise)
    AuthMethod           string        // "kubernetes" or "token"
    Role                 string        // K8s auth role name
    TokenPath            string        // Path to K8s SA token
    Token                string        // Static token (dev only)
    TLSSkipVerify        bool          // Skip TLS verification
    CACert               string        // CA certificate path
    SecretMountPath      string        // KV secrets mount path
    TokenRenewalInterval time.Duration
    SecretRefreshInterval time.Duration
}

type SecretPaths struct {
    DatabaseBuffer  string // secret/data/philotes/database/buffer
    DatabaseSource  string // secret/data/philotes/database/source
    StorageMinio    string // secret/data/philotes/storage/minio
}
```

### 1.2 Create `internal/vault/client.go`

```go
type Client struct {
    config     *Config
    vaultAPI   *vault.Client
    logger     *slog.Logger
    mu         sync.RWMutex
    secrets    map[string]*CachedSecret
}

func NewClient(cfg *Config, logger *slog.Logger) (*Client, error)
func (c *Client) Authenticate(ctx context.Context) error
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error)
func (c *Client) GetSecretString(ctx context.Context, path, key string) (string, error)
func (c *Client) StartRenewal(ctx context.Context) error
func (c *Client) Close() error
```

### 1.3 Create `internal/vault/auth.go`

```go
func (c *Client) authenticateKubernetes(ctx context.Context) error
func (c *Client) authenticateToken(ctx context.Context) error
func (c *Client) renewToken(ctx context.Context) error
```

### 1.4 Create `internal/vault/secrets.go`

```go
type CachedSecret struct {
    Data      map[string]interface{}
    ExpiresAt time.Time
    LeaseID   string
}

type SecretProvider interface {
    GetDatabasePassword(ctx context.Context) (string, error)
    GetSourcePassword(ctx context.Context) (string, error)
    GetStorageCredentials(ctx context.Context) (accessKey, secretKey string, err error)
    Refresh(ctx context.Context) error
}

// VaultSecretProvider implements SecretProvider using Vault
type VaultSecretProvider struct { ... }

// EnvSecretProvider implements SecretProvider using environment variables (fallback)
type EnvSecretProvider struct { ... }
```

## Phase 2: Configuration Integration

### 2.1 Update `internal/config/config.go`

Add Vault configuration section:

```go
type Config struct {
    // ... existing fields ...

    // Vault configuration
    Vault VaultConfig
}

type VaultConfig struct {
    Enabled               bool
    Address               string
    Namespace             string
    AuthMethod            string
    Role                  string
    TokenPath             string
    Token                 string
    TLSSkipVerify         bool
    CACert                string
    SecretMountPath       string
    TokenRenewalInterval  time.Duration
    SecretRefreshInterval time.Duration
    FallbackToEnv         bool
    SecretPaths           VaultSecretPaths
}

type VaultSecretPaths struct {
    DatabaseBuffer string
    DatabaseSource string
    StorageMinio   string
}
```

Environment variables:
- `PHILOTES_VAULT_ENABLED` (default: false)
- `PHILOTES_VAULT_ADDRESS` (default: "")
- `PHILOTES_VAULT_NAMESPACE` (default: "")
- `PHILOTES_VAULT_AUTH_METHOD` (default: "kubernetes")
- `PHILOTES_VAULT_ROLE` (default: "philotes")
- `PHILOTES_VAULT_TOKEN_PATH` (default: "/var/run/secrets/kubernetes.io/serviceaccount/token")
- `PHILOTES_VAULT_TLS_SKIP_VERIFY` (default: false)
- `PHILOTES_VAULT_CA_CERT` (default: "")
- `PHILOTES_VAULT_SECRET_MOUNT_PATH` (default: "secret")
- `PHILOTES_VAULT_TOKEN_RENEWAL_INTERVAL` (default: "1h")
- `PHILOTES_VAULT_SECRET_REFRESH_INTERVAL` (default: "5m")
- `PHILOTES_VAULT_FALLBACK_TO_ENV` (default: true)
- `PHILOTES_VAULT_SECRET_PATH_DATABASE_BUFFER` (default: "philotes/database/buffer")
- `PHILOTES_VAULT_SECRET_PATH_DATABASE_SOURCE` (default: "philotes/database/source")
- `PHILOTES_VAULT_SECRET_PATH_STORAGE_MINIO` (default: "philotes/storage/minio")

## Phase 3: Service Integration

### 3.1 Update `cmd/philotes-api/main.go`

```go
func main() {
    // ... existing setup ...

    cfg, err := config.Load()

    // Initialize secret provider
    var secretProvider vault.SecretProvider
    if cfg.Vault.Enabled {
        vaultClient, err := vault.NewClient(&cfg.Vault, logger)
        if err != nil {
            if cfg.Vault.FallbackToEnv {
                logger.Warn("vault unavailable, falling back to env", "error", err)
                secretProvider = vault.NewEnvSecretProvider()
            } else {
                logger.Error("failed to create vault client", "error", err)
                os.Exit(1)
            }
        } else {
            secretProvider = vault.NewVaultSecretProvider(vaultClient, cfg.Vault.SecretPaths)
            defer vaultClient.Close()
        }
    } else {
        secretProvider = vault.NewEnvSecretProvider()
    }

    // Get database password from provider
    dbPassword, err := secretProvider.GetDatabasePassword(ctx)
    cfg.Database.Password = dbPassword

    // ... continue with existing setup ...
}
```

### 3.2 Update `cmd/philotes-worker/main.go`

Similar pattern plus background refresh goroutine:

```go
// Start secret refresh loop
if vaultProvider, ok := secretProvider.(*vault.VaultSecretProvider); ok {
    go vaultProvider.StartRefreshLoop(ctx, cfg.Vault.SecretRefreshInterval)
}
```

## Phase 4: Helm Chart Updates

### 4.1 Update values.yaml files

Add to `charts/philotes-api/values.yaml` and `charts/philotes-worker/values.yaml`:

```yaml
vault:
  enabled: false
  address: ""
  namespace: ""
  authMethod: "kubernetes"
  role: "philotes"
  tlsSkipVerify: false
  caCert: ""
  secretMountPath: "secret"
  tokenRenewalInterval: "1h"
  secretRefreshInterval: "5m"
  fallbackToEnv: true
  secretPaths:
    databaseBuffer: "philotes/database/buffer"
    databaseSource: "philotes/database/source"
    storageMinio: "philotes/storage/minio"

  # Vault Agent Sidecar (optional)
  agent:
    enabled: false
    image: "hashicorp/vault:1.15"
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 100m
        memory: 128Mi
```

### 4.2 Update ConfigMap templates

Add Vault environment variables to configmap.yaml templates.

### 4.3 Create Vault Agent sidecar template (optional)

Create `templates/vault-agent.yaml` for Agent sidecar injection.

### 4.4 Update ServiceAccount

Add annotations for Vault Kubernetes auth:

```yaml
{{- if .Values.vault.enabled }}
annotations:
  vault.hashicorp.com/agent-inject: "false"
  vault.hashicorp.com/role: {{ .Values.vault.role | quote }}
{{- end }}
```

## Phase 5: Testing

### 5.1 Unit Tests

- `internal/vault/client_test.go` - Mock Vault API responses
- `internal/vault/auth_test.go` - Test K8s auth flow
- `internal/vault/secrets_test.go` - Test secret retrieval and caching

### 5.2 Integration Tests

Add Vault to docker-compose for local testing:

```yaml
vault:
  image: hashicorp/vault:1.15
  container_name: philotes-vault
  environment:
    VAULT_DEV_ROOT_TOKEN_ID: "dev-token"
    VAULT_DEV_LISTEN_ADDRESS: "0.0.0.0:8200"
  ports:
    - "8200:8200"
  cap_add:
    - IPC_LOCK
```

## Task Breakdown

| # | Task | Files | Est. LOC |
|---|------|-------|----------|
| 1 | Add Vault SDK dependency | go.mod, go.sum | 10 |
| 2 | Create vault config types | internal/vault/config.go | 100 |
| 3 | Implement Vault client | internal/vault/client.go | 300 |
| 4 | Implement K8s auth | internal/vault/auth.go | 150 |
| 5 | Implement secret provider | internal/vault/secrets.go | 250 |
| 6 | Add unit tests | internal/vault/*_test.go | 500 |
| 7 | Update main config | internal/config/config.go | 100 |
| 8 | Update API main | cmd/philotes-api/main.go | 50 |
| 9 | Update worker main | cmd/philotes-worker/main.go | 80 |
| 10 | Update Helm charts | charts/*/values.yaml, templates/* | 300 |
| 11 | Add Vault to docker-compose | deployments/docker/docker-compose.yml | 30 |
| 12 | Documentation | charts/*/README.md | 200 |

**Total: ~2,070 LOC**

## Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|---------------|
| Vault client integration in Go | `internal/vault/client.go` |
| Kubernetes auth method | `internal/vault/auth.go` with K8s auth |
| Dynamic database credentials | SecretProvider with refresh loop |
| Secret rotation without restart | Background refresh goroutine |
| Fallback to K8s Secrets | `FallbackToEnv` config option |
| Vault Agent sidecar option | Helm chart templates |
| Audit logging | Structured logging in vault package |

## Rollout Strategy

1. Deploy with `vault.enabled: false` (no change in behavior)
2. Configure Vault server with K8s auth and policies
3. Enable Vault for staging: `vault.enabled: true, vault.fallbackToEnv: true`
4. Test secret retrieval and rotation
5. Enable for production after validation
