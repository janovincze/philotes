# Implementation Plan: Issue #52 - Dynamic Cloud-Init IP Generation

## Approach Overview

Use Pulumi's `ApplyT` pattern to dynamically generate cloud-init scripts with actual server IPs. This requires:

1. Changing `ServerOptions.UserData` from `string` to `pulumi.StringInput` to accept both static strings and dynamic outputs
2. Updating provider implementations to handle the new type
3. Modifying `platform.go` to use Apply pattern for worker cloud-init generation

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/provider/provider.go` | Change `UserData` to `pulumi.StringInput` |
| `pkg/provider/hetzner/compute.go` | Handle `StringInput` for UserData |
| `pkg/provider/scaleway/compute.go` | Handle `StringInput` for UserData + fix PrivateIP bug |
| `pkg/platform/platform.go` | Use Apply pattern for dynamic cloud-init |
| `pkg/cluster/k3s.go` | No changes needed (functions already accept string params) |

## Implementation Tasks

### Task 1: Update Provider Interface (provider.go)

Change `ServerOptions.UserData` from `string` to `pulumi.StringInput`:

```go
type ServerOptions struct {
    // ... other fields unchanged ...
    // UserData is the cloud-init script content (can be string or StringOutput).
    UserData pulumi.StringInput
}
```

### Task 2: Update Hetzner Provider (hetzner/compute.go)

Handle the `StringInput` type for UserData:

```go
// Add cloud-init user data if provided
if opts.UserData != nil {
    serverArgs.UserData = opts.UserData.ToStringOutput()
}
```

### Task 3: Update Scaleway Provider (scaleway/compute.go)

1. Handle `StringInput` type for UserData
2. Fix the PrivateIP bug (currently uses `nic.ID()` which is wrong)

```go
// Add cloud-init user data if provided
if opts.UserData != nil {
    serverArgs.CloudInit = opts.UserData.ToStringOutput()
}

// Fix PrivateIP - use nic.PrivateIps instead of ID
privateIP = nic.PrivateIps.Index(pulumi.Int(0))
```

### Task 4: Update Platform Orchestrator (platform.go)

Change the deployment flow to:

1. Create control plane with actual public IP for TLS-SAN
2. Use `ApplyT` to generate worker cloud-init dynamically based on control plane's private IP

**Control Plane** - Two-step approach:
- First create the server (need to get the public IP first)
- Use Apply to pass actual public IP to cloud-init

**Workers** - Use Apply pattern:
```go
workerUserData := controlPlane.PrivateIP.ApplyT(func(ip interface{}) string {
    return cluster.WorkerCloudInit(ip.(string), clusterToken)
}).(pulumi.StringOutput)
```

### Task 5: Handle Control Plane Public IP for TLS-SAN

**Challenge**: We need the control plane's public IP for cloud-init, but the IP isn't known until after server creation.

**Solution**: K3s actually handles this well - it auto-adds the server's IP to TLS SANs. We can:
- Use `0.0.0.0` for initial TLS-SAN (K3s will auto-add actual IP)
- OR use a remote.Command to restart K3s with correct TLS-SAN after IP is known

For simplicity, we'll rely on K3s's auto-detection for TLS-SAN and focus on the critical worker IP issue.

## Detailed Implementation

### Step 1: provider.go Changes

```go
type ServerOptions struct {
    ServerType   string
    Region       string
    Image        string
    SSHPublicKey string
    UserData     pulumi.StringInput  // Changed from string
    NetworkID    pulumi.IDOutput
    FirewallID   pulumi.IDOutput
    Labels       map[string]string
}
```

### Step 2: hetzner/compute.go Changes

```go
// Add cloud-init user data if provided
if opts.UserData != nil {
    serverArgs.UserData = opts.UserData.ToStringOutput()
}
```

### Step 3: scaleway/compute.go Changes

```go
// Add cloud-init user data if provided
if opts.UserData != nil {
    serverArgs.CloudInit = opts.UserData.ToStringOutput()
}

// Fix: Get actual private IP from NIC
if opts.NetworkID != (pulumi.IDOutput{}) {
    nic, nicErr := scaleway.NewInstancePrivateNic(...)
    if nicErr != nil {
        return nil, fmt.Errorf("failed to attach server to private network: %w", nicErr)
    }
    // Use IpIds array to get actual IP - Scaleway assigns IPs from DHCP
    // Note: For simplicity, we'll keep the nic.ID() for now as a placeholder
    // since Scaleway's private IP assignment is complex
    privateIP = nic.PrivateIps.Index(pulumi.Int(0)).ToStringOutput()
}
```

### Step 4: platform.go Changes

```go
// Create control plane node
controlPlane, err := cp.CreateServer(ctx, cfg.ResourceName("control-plane"), provider.ServerOptions{
    ServerType:   cfg.ControlPlaneType,
    Region:       cfg.Region,
    SSHPublicKey: cfg.SSHPublicKey,
    UserData:     pulumi.String(cluster.ControlPlaneCloudInit("0.0.0.0", clusterToken)),
    NetworkID:    network.NetworkID,
    FirewallID:   firewall.FirewallID,
    Labels: map[string]string{...},
})

// Create worker nodes with dynamic cloud-init
for i := 0; i < cfg.WorkerCount; i++ {
    workerName := fmt.Sprintf("%s-worker-%d", cfg.ResourceName(""), i)

    // Generate cloud-init dynamically based on control plane's actual private IP
    workerUserData := controlPlane.PrivateIP.ApplyT(func(ip interface{}) string {
        return cluster.WorkerCloudInit(ip.(string), clusterToken)
    }).(pulumi.StringOutput)

    worker, workerErr := cp.CreateServer(ctx, workerName, provider.ServerOptions{
        ServerType:   cfg.WorkerType,
        Region:       cfg.Region,
        SSHPublicKey: cfg.SSHPublicKey,
        UserData:     workerUserData,  // Now uses dynamic IP
        NetworkID:    network.NetworkID,
        FirewallID:   firewall.FirewallID,
        Labels: map[string]string{...},
    })
}
```

## Test Strategy

1. **Unit Tests**: Verify type conversions work correctly
2. **Integration Test**: Deploy a test cluster with Pulumi preview
3. **Manual Verification**:
   - Control plane is accessible via public IP
   - Workers successfully join the cluster
   - `kubectl get nodes` shows all nodes ready

## Verification Commands

```bash
# Build the project
cd /Volumes/ExternalSSD/dev/philotes/deployments/pulumi
go build ./...

# Verify with pulumi preview (requires provider credentials)
pulumi preview --stack dev

# After deployment, verify cluster
kubectl get nodes
```

## Notes

- The TLS-SAN issue (control plane public IP) is less critical because K3s auto-adds the server's IP
- The worker join IP issue is the critical fix - this is what causes cluster formation failures
- Scaleway's private IP handling is more complex; the fix may need adjustment based on their API
