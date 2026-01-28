# Research: Issue #52 - Dynamic Cloud-Init IP Generation

## Current Implementation Analysis

### 1. Cloud-Init Generation (pkg/cluster/k3s.go)

The current system uses two hardcoded IP patterns:

- **ControlPlaneCloudInit()**: Takes a `publicIP` parameter but is called with `"0.0.0.0"`
  - Used for TLS-SAN configuration in K3s server installation
  - K3s auto-adds server's actual IP to TLS SANs, but `"0.0.0.0"` is non-functional

- **WorkerCloudInit()**: Takes a `controlPlaneIP` parameter but is called with hardcoded `"10.0.1.1"`
  - **Critical reliability issue** - if control plane gets different private IP, workers fail to join
  - Workers poll `https://10.0.1.1:6443/ping` then connect via `K3S_URL`

### 2. Server Creation and IP Availability

Both providers return `ServerResult` with:

| Provider | PublicIP Source | PrivateIP Source |
|----------|-----------------|------------------|
| Hetzner | `server.Ipv4Address` | `attachment.Ip` from `ServerNetwork` |
| Scaleway | `server.PublicIp` | **BUG**: Uses `nic.ID()` instead of actual IP |

### 3. Pulumi Apply Pattern Usage

The codebase demonstrates Apply patterns in:
- **kubeconfig.go:47**: `pulumi.All(fetchCmd.Stdout, opts.ControlPlaneIP).ApplyT(...)`

## Key Findings

### Problem 1: Control Plane TLS-SAN
- Current: `"0.0.0.0"` doesn't help TLS validation
- Need: Pass actual public IP for external kubectl access

### Problem 2: Worker Control Plane Discovery
- Current: `"10.0.1.1"` assumes deterministic IP assignment
- Reality: Cloud providers assign IPs dynamically from CIDR block

### Problem 3: Scaleway Private IP Bug
- `scaleway/compute.go:64` uses `nic.ID()` for PrivateIP
- Should use actual IP address attribute

## Recommended Solution: Apply-Based Cloud-Init

**Approach**: Use Pulumi's `ApplyT` to dynamically generate cloud-init

```
Execution Flow:
1. Create control plane server → Get PublicIP
2. Attach to network → Get PrivateIP
3. Use ApplyT to generate worker cloud-init with actual IPs
4. Create worker servers with dynamically-generated cloud-init
```

**Why ApplyT over remote.Command**:
- ✅ Cloud-init baked into server creation (immutable, reliable)
- ✅ No SSH required, no bootstrap time dependency
- ✅ Follows Pulumi best practices
- ✅ Works for multi-worker scaling

## Files Requiring Modification

1. **pkg/provider/provider.go**
   - Change `UserData` from `string` to `pulumi.StringInput` in `ServerOptions`

2. **pkg/provider/hetzner/compute.go**
   - Update `CreateServer` to handle `pulumi.StringInput` for UserData

3. **pkg/provider/scaleway/compute.go**
   - Update `CreateServer` to handle `pulumi.StringInput` for UserData
   - Fix PrivateIP bug (separate issue consideration)

4. **pkg/platform/platform.go**
   - Control plane: Pass public IP to cloud-init
   - Workers: Use ApplyT pattern for dynamic cloud-init generation

5. **pkg/cluster/k3s.go**
   - Add new `ControlPlaneCloudInitOutput` function returning `pulumi.StringOutput`
   - Add new `WorkerCloudInitOutput` function returning `pulumi.StringOutput`

## Implementation Pattern

```go
// Workers use Apply to generate cloud-init dynamically
workerCloudInit := controlPlane.PrivateIP.ApplyT(func(ip interface{}) string {
    return cluster.WorkerCloudInit(ip.(string), clusterToken)
}).(pulumi.StringOutput)

worker, err := cp.CreateServer(ctx, workerName, provider.ServerOptions{
    UserData: workerCloudInit, // Now accepts StringInput
    ...
})
```

## Alternative Approaches Considered

| Approach | Pros | Cons |
|----------|------|------|
| Two-phase remote.Command | Cloud-init stays simple | Requires SSH, non-idempotent |
| Template token replacement | Minimal type changes | Error-prone, not Pulumi-native |
| DependsOn ordering | No type changes | Fragile, doesn't work for scaling |

**Selected**: ApplyT pattern - follows Pulumi best practices
