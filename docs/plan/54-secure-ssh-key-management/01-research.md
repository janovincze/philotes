# Research Findings: Issue #54 - Secure SSH Key Management

## Current Implementation Analysis

### 1. SSH Key Usage Locations

The SSH private key is currently used in **one critical location**:

**`deployments/pulumi/pkg/cluster/kubeconfig.go`** (lines 16-36):
- Reads SSH private key from local filesystem path
- Passes raw key content to Pulumi's remote command provider
- Key gets stored in Pulumi state (even if encrypted)
- Used to SSH into control plane to fetch kubeconfig

### 2. Configuration Flow

**`deployments/pulumi/pkg/config/config.go`**:
- Line 32-33: Defines `SSHPrivateKeyPath` in Config struct
- Line 162-170: Loads SSH private key path from config
  - First looks for `sshPrivateKeyPath` config value
  - Falls back to deriving from public key path (removes `.pub` suffix)
  - Final fallback: `~/.ssh/id_rsa`
- Currently reads public key content directly (line 156)

**`deployments/pulumi/main.go`**:
- Line 48: Exports kubeconfig as secret using `pulumi.ToSecret()`
- Passes SSH key path through to `platform.Deploy()`

**`deployments/pulumi/pkg/platform/platform.go`**:
- Line 131: Passes `SSHPrivateKeyPath` to `cluster.GetKubeconfig()`

### 3. Provider SSH Key Usage

SSH **public key** is used for cloud infrastructure:

| Provider | File | Pattern |
|----------|------|---------|
| Hetzner | `compute.go:25-27` | Creates SSH key resource |
| Scaleway | `compute.go:22-24` | IAM SSH key |
| Exoscale | `compute.go:25-27` | SSH key resource |
| OVH | Uses Managed K8s | No direct SSH |
| Contabo | Pre-provisioned VPS | Placeholder |

### 4. Existing Vault Integration

Excellent news: Vault is already integrated in the main application:

**`internal/vault/`**:
- **client.go** (325 lines): Full Vault client with auth methods
  - Kubernetes and Token authentication
  - Token renewal with background goroutines
  - Health checks and error handling

- **config.go** (113 lines): Configuration structure
  - Secret paths for database and storage credentials

- **secrets.go** (352 lines): Secret provider interface
  - `VaultSecretProvider`: Fetches from Vault with caching
  - `EnvSecretProvider`: Fallback to environment variables
  - `SecretProvider` interface for pluggable implementations

### 5. Pulumi SDK Capabilities

From `go.mod`:
- Pulumi SDK v3.190.0
- Pulumi ESC v0.17.0 included
- Pulumi command provider v1.0.1

### 6. Security Concerns with Current Implementation

1. **SSH key stored in Pulumi state** - Even encrypted state contains key material
2. **Private key file dependency** - Must exist on deployment machine
3. **No audit trail** - No logging/tracking of key access
4. **File system race conditions** - Key readable by local processes
5. **CI/CD challenges** - Key must be provisioned in CI environment
6. **No key rotation support** - Fixed key path assumption

## Recommended Implementation Approach

### Three-tier Strategy

1. **Tier 1 - Pulumi Secrets (Development)**
   - Store SSH private key in Pulumi secrets
   - Configure via `Pulumi.*.yaml` encrypted config
   - Never write key to file system during deployment

2. **Tier 2 - HashiCorp Vault (Production)**
   - Leverage existing Vault integration
   - Store SSH key in Vault KV secrets
   - Kubernetes auth for production deployments

3. **Tier 3 - Local File Fallback (Development Only)**
   - Keep current file-based approach as final fallback
   - Warn users about security implications

## Files Requiring Modification

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add SSH key content field, secret source config |
| `pkg/cluster/kubeconfig.go` | Accept key content instead of path |
| `main.go` | Load SSH key from secrets manager |
| `Pulumi.yaml` | Add config schema for secret source |
| **New:** `pkg/secrets/` | Create secrets abstraction layer |

## Implementation Strategy

1. **Phase 1**: Create secrets abstraction layer with file fallback
2. **Phase 2**: Implement Pulumi secrets provider
3. **Phase 3**: Integrate with existing Vault infrastructure
4. **Phase 4**: Documentation and examples
