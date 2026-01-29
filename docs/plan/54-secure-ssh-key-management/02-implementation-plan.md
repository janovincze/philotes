# Implementation Plan: Issue #54 - Secure SSH Key Management

## Summary

Implement a three-tier SSH key management system that supports:
1. Pulumi secrets (encrypted in config)
2. HashiCorp Vault (production)
3. Local file fallback (development)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        config.go                              │
│  LoadConfig() → GetSSHPrivateKey() → Config.SSHPrivateKey    │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    pkg/secrets/provider.go                    │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐     │
│  │PulumiProvider │  │ VaultProvider │  │ FileProvider  │     │
│  │ (encrypted)   │  │ (production)  │  │ (fallback)    │     │
│  └───────────────┘  └───────────────┘  └───────────────┘     │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│              cluster/kubeconfig.go                            │
│  GetKubeconfig(opts.SSHPrivateKey pulumi.StringOutput)       │
└──────────────────────────────────────────────────────────────┘
```

## Files to Create

| File | Description |
|------|-------------|
| `pkg/secrets/provider.go` | SSHKeyProvider interface and factory |
| `pkg/secrets/pulumi.go` | Pulumi secrets implementation |
| `pkg/secrets/vault.go` | Vault secrets implementation |
| `pkg/secrets/file.go` | File-based fallback implementation |

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add SSHPrivateKey field, secret source config |
| `pkg/cluster/kubeconfig.go` | Accept StringOutput instead of file path |
| `pkg/platform/platform.go` | Pass SSHPrivateKey to kubeconfig retrieval |
| `main.go` | Initialize secrets provider |

## Implementation Tasks

### Task 1: Create Secrets Provider Abstraction

Create `pkg/secrets/provider.go`:
```go
package secrets

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// SSHKeySource defines where to load SSH keys from
type SSHKeySource string

const (
    SourcePulumi SSHKeySource = "pulumi"  // Pulumi encrypted secrets
    SourceVault  SSHKeySource = "vault"   // HashiCorp Vault
    SourceFile   SSHKeySource = "file"    // Local file (fallback)
)

// SSHKeyProvider provides SSH private keys from various sources
type SSHKeyProvider interface {
    GetPrivateKey(ctx *pulumi.Context) (pulumi.StringOutput, error)
    Source() SSHKeySource
}

// NewSSHKeyProvider creates provider based on configuration
func NewSSHKeyProvider(ctx *pulumi.Context, source SSHKeySource, opts Options) (SSHKeyProvider, error)
```

### Task 2: Implement Pulumi Secrets Provider

Create `pkg/secrets/pulumi.go`:
- Read encrypted secret from Pulumi config (`philotes:sshPrivateKey`)
- Return as `pulumi.StringOutput`
- Secret never written to state in plaintext

### Task 3: Implement Vault Provider

Create `pkg/secrets/vault.go`:
- Connect to Vault using address from config
- Read SSH key from path `secret/data/philotes/ssh-key`
- Support Kubernetes auth for production
- Fallback to token auth for development

### Task 4: Implement File Provider (Fallback)

Create `pkg/secrets/file.go`:
- Read SSH key from file path
- Log warning about security implications
- Used when no secrets manager configured

### Task 5: Update Configuration

Modify `pkg/config/config.go`:
- Add `SSHKeySource` field (pulumi/vault/file)
- Add `SSHPrivateKey` field (pulumi.StringOutput)
- Add Vault configuration (address, auth method, secret path)
- Load SSH key using provider during LoadConfig

### Task 6: Update Kubeconfig Retrieval

Modify `pkg/cluster/kubeconfig.go`:
- Change `SSHPrivateKeyPath string` to `SSHPrivateKey pulumi.StringOutput`
- Remove `os.ReadFile()` call
- Pass key directly to remote.ConnectionArgs

### Task 7: Update Platform Deployment

Modify `pkg/platform/platform.go`:
- Pass `cfg.SSHPrivateKey` instead of `cfg.SSHPrivateKeyPath`

### Task 8: Documentation

Add to plan folder:
- Guide for setting Pulumi secrets
- Guide for Vault integration
- CI/CD setup guide

## Configuration Schema

### Pulumi Config (Pulumi.yaml)
```yaml
config:
  philotes:sshKeySource: pulumi  # or "vault" or "file"
  philotes:sshPrivateKey:
    secure: AAABADFg...  # encrypted by `pulumi config set --secret`
  # Vault options (if sshKeySource: vault)
  philotes:vaultAddress: https://vault.example.com
  philotes:vaultAuthMethod: kubernetes  # or "token"
  philotes:vaultSecretPath: secret/data/philotes/ssh-key
```

### Setting Pulumi Secret
```bash
# Set SSH private key as encrypted secret
pulumi config set --secret philotes:sshPrivateKey "$(cat ~/.ssh/id_rsa)"
```

### Vault Secret Path
```bash
# Store SSH key in Vault
vault kv put secret/philotes/ssh-key private_key=@~/.ssh/id_rsa
```

## Security Considerations

1. **Pulumi State**: Private key stored encrypted, decrypted only during execution
2. **Vault**: Key never touches disk, fetched at runtime
3. **File Fallback**: Logs warning, for development only
4. **No Key Logging**: Ensure key content never logged

## Verification

```bash
# Build verification
cd deployments/pulumi && go build ./...

# Test with Pulumi secrets
pulumi config set --secret philotes:sshPrivateKey "$(cat ~/.ssh/id_rsa)"
pulumi preview

# Test with file fallback
pulumi config set philotes:sshKeySource file
pulumi preview
```

## Task Order

1. Create `pkg/secrets/provider.go` - Interface and factory
2. Create `pkg/secrets/file.go` - File provider (needed for testing)
3. Create `pkg/secrets/pulumi.go` - Pulumi secrets provider
4. Create `pkg/secrets/vault.go` - Vault provider
5. Modify `pkg/config/config.go` - Add configuration fields
6. Modify `pkg/cluster/kubeconfig.go` - Accept StringOutput
7. Modify `pkg/platform/platform.go` - Pass SSH key
8. Update `main.go` if needed
9. Build verification
10. Documentation
