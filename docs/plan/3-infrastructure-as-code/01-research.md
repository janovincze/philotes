# Research Findings - Issue #3 Infrastructure as Code

## Executive Summary

The Philotes project already has a **substantial IaC foundation** in place:
- ✅ Provider abstraction pattern (CloudProvider interface)
- ✅ Hetzner Cloud fully implemented
- ✅ Scaleway fully implemented
- ✅ K3s cluster bootstrapping
- ✅ Helm charts for all Philotes components
- ✅ Cost estimation
- ❌ OVHcloud, Exoscale, Contabo not yet implemented
- ❌ DNS configuration not implemented

## Current Architecture

```
deployments/pulumi/
├── main.go                      # Entry point with provider selection
├── Pulumi.yaml                  # Project config
├── Pulumi.{dev,staging,prod}.yaml
└── pkg/
    ├── config/config.go         # Configuration loading
    ├── cluster/
    │   ├── k3s.go               # K3s bootstrapping
    │   └── kubeconfig.go        # Kubeconfig retrieval
    ├── output/cost.go           # Cost estimation
    ├── platform/
    │   ├── platform.go          # Deployment orchestration
    │   ├── certmanager.go       # TLS management
    │   ├── ingress.go           # Ingress-nginx
    │   ├── monitoring.go        # Prometheus stack
    │   └── philotes.go          # Philotes Helm charts
    └── provider/
        ├── provider.go          # CloudProvider interface
        ├── hetzner/             # ✅ Complete
        │   ├── provider.go
        │   ├── compute.go
        │   ├── network.go
        │   ├── storage.go
        │   └── loadbalancer.go
        └── scaleway/            # ✅ Complete
            ├── provider.go
            ├── compute.go
            ├── network.go
            ├── storage.go
            └── loadbalancer.go
```

## CloudProvider Interface

```go
type CloudProvider interface {
    CreateNetwork(ctx, name, opts) (*NetworkResult, error)
    CreateFirewall(ctx, name, rules) (*FirewallResult, error)
    CreateServer(ctx, name, opts) (*ServerResult, error)
    CreateVolume(ctx, name, sizeGB, opts) (*VolumeResult, error)
    CreateLoadBalancer(ctx, name, opts) (*LBResult, error)
    Name() string
}
```

## What Needs Implementation

### 1. New Cloud Providers (3 remaining)

| Provider | Priority | API Documentation |
|----------|----------|-------------------|
| OVHcloud | High | openstack-based, Pulumi provider available |
| Exoscale | Medium | openstack-based, Pulumi provider available |
| Contabo | Low | API is limited, may need custom approach |

### 2. DNS Configuration (Optional)

Currently not implemented. Would need:
- DNS provider interface (Cloudflare, Route53, etc.)
- Record management for load balancer IPs
- Integration with cert-manager for DNS-01 challenges

### 3. Missing Acceptance Criteria Status

| Criteria | Status |
|----------|--------|
| Pulumi project structure with provider abstraction | ✅ Done |
| K3s cluster deployment per provider | ✅ Done (Hetzner, Scaleway) |
| Networking (VPC, firewall, load balancer) | ✅ Done |
| Storage provisioning (block storage for MinIO) | ✅ Done |
| DNS configuration (optional) | ❌ Not implemented |
| SSL/TLS via cert-manager | ✅ Done |
| Cost estimation output | ✅ Done |
| Destroy/cleanup support | ✅ Done (Pulumi built-in) |

## Cost Estimates (Monthly EUR)

### Hetzner (Default config)
- Control Plane (cx22): €4.35
- 2 Workers (cx32 each): €15.18
- Storage (50GB): €2.35
- Load Balancer: €5.39
- **Total: €27.27/month**

### Scaleway (Default config)
- Control Plane (DEV1-M): €9.99
- 2 Workers (DEV1-L each): €35.98
- Storage (50GB): €4.00
- Load Balancer: €9.99
- **Total: €59.96/month**

## Recommended Approach

### Phase 1: Add OVHcloud Provider
1. Create `pkg/provider/ovh/` directory
2. Implement CloudProvider interface using Pulumi OVHcloud provider
3. Add cost estimation for OVH
4. Test deployment

### Phase 2: Add Exoscale Provider
1. Create `pkg/provider/exoscale/` directory
2. Implement CloudProvider interface
3. Add cost estimation
4. Test deployment

### Phase 3: Add Contabo Provider
1. Research Contabo API capabilities
2. Determine if Pulumi provider exists or needs custom implementation
3. Implement if feasible

### Phase 4: DNS Configuration (Optional)
1. Add DNS provider abstraction
2. Implement Cloudflare DNS provider
3. Integrate with deployment flow

## Dependencies

- `github.com/pulumi/pulumi-hcloud/sdk/go/hcloud` (Hetzner)
- `github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway` (Scaleway)
- `github.com/ovh/pulumi-ovh/sdk/go/ovh` (OVHcloud - to add)
- `github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale` (Exoscale - to add)

## Questions for Clarification

1. Is Contabo a hard requirement? Their API is limited compared to others.
2. Should DNS be implemented in this issue or deferred?
3. What's the priority order for the remaining providers?
