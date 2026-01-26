# Research - Issue #17: HashiCorp Vault Integration

## Configuration System Overview

**Current State** (`internal/config/config.go`):
- Configuration loaded via environment variables using helper functions
- `Load()` function reads ~40+ environment variables into `Config` struct
- No secret redaction - passwords stored as plain text in Config struct
- All secrets currently come from environment variables (set by K8s Secrets)

**Secrets Currently Managed**:
- Database passwords (buffer): `PHILOTES_DB_PASSWORD`
- Source database password: `PHILOTES_CDC_SOURCE_PASSWORD`
- MinIO/S3 access key: `PHILOTES_STORAGE_ACCESS_KEY`
- MinIO/S3 secret key: `PHILOTES_STORAGE_SECRET_KEY`

## Service Entry Points

**API Service** (`cmd/philotes-api/main.go`):
- Loads config via `config.Load()`
- Creates database connection using `sql.Open("pgx", cfg.Database.DSN())`
- Pattern: `config.Load()` → `sql.Open()` → `db.PingContext()`

**Worker Service** (`cmd/philotes-worker/main.go`):
- Loads config via `config.Load()`
- Initializes multiple clients: PostgreSQL reader, checkpoint manager, buffer manager, Iceberg writer
- Pattern: Configuration loaded → clients instantiated sequentially

## Existing Client Patterns

**MinIO/S3 Client** (`internal/iceberg/writer/s3.go`):
- Uses `minio.New()` with static credentials
- No support for credential refresh at runtime

**Iceberg Writer** (`internal/iceberg/writer/writer.go`):
- `NewIcebergWriter()` takes configuration and logger
- Creates catalog client and S3 client

## Helm Charts - Current Secret Injection

**API Deployment** (`charts/philotes-api/templates/deployment.yaml`):
- Uses `envFrom` for ConfigMap
- Pulls database password from K8s Secret via `secretKeyRef`

**Worker Deployment** (`charts/philotes-worker/templates/deployment.yaml`):
- Multiple secrets injected as environment variables
- Supports `existingSecret` pattern for external secret management

## Dependencies

**go.mod**: No HashiCorp Vault SDK currently included

**Recommended Vault SDK**: `github.com/hashicorp/vault/api` v1.x

## Recommended Approach

### Hybrid: SDK + Optional Agent Sidecar

**SDK Approach** (Primary):
- Direct Vault SDK in Go services using Kubernetes auth
- Simple, no extra container, dynamic credential rotation possible

**Agent Sidecar** (Optional):
- Vault Agent as sidecar injecting secrets
- Handles rotation, simpler application code

### Integration Points

1. **New package**: `internal/vault/` - Vault SDK wrapper
2. **Config enhancement**: Add Vault settings to config
3. **Entry points**: Initialize Vault client after config load
4. **Credential refresh**: Background goroutine for lease renewal

### Secret Paths Strategy

```
secret/data/philotes/
├── database/buffer     # PHILOTES_DB_PASSWORD
├── database/source     # PHILOTES_CDC_SOURCE_PASSWORD
└── storage/minio       # Access key + Secret key
```

### Kubernetes Auth Flow

1. Pod starts with service account token
2. Vault client reads SA token from `/var/run/secrets/kubernetes.io/serviceaccount/token`
3. Authenticate to Vault: `POST /auth/kubernetes/login`
4. Receive client token + lease info
5. Use client token to fetch secrets
6. Implement lease renewal/rotation

## Files to Create

| File | Purpose |
|------|---------|
| `internal/vault/client.go` | Vault SDK wrapper |
| `internal/vault/config.go` | Vault configuration structs |
| `internal/vault/auth.go` | Kubernetes auth implementation |
| `internal/vault/secrets.go` | Secret retrieval and caching |
| `internal/vault/client_test.go` | Unit tests |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add Vault config section |
| `cmd/philotes-api/main.go` | Vault client initialization |
| `cmd/philotes-worker/main.go` | Vault client + refresh loop |
| `go.mod` | Add Vault SDK dependency |
| `charts/*/values.yaml` | Vault configuration values |
| `charts/*/templates/deployment.yaml` | Vault Agent sidecar option |
