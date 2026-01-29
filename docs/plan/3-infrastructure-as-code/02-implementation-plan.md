# Implementation Plan - Issue #3 Infrastructure as Code

## Summary

The Philotes IaC foundation is already substantially implemented with Hetzner and Scaleway providers. This issue focuses on completing the remaining providers (OVHcloud, Exoscale, Contabo) to fulfill the multi-provider promise.

**Current State:** 2/5 providers implemented (Hetzner, Scaleway)
**Target State:** 5/5 providers implemented + optional DNS

## Approach

Follow the established provider pattern:
1. Each provider implements `CloudProvider` interface
2. Provider-specific logic isolated in `pkg/provider/<name>/`
3. Cost estimation added to `pkg/output/cost.go`
4. Configuration defaults in `pkg/config/config.go`

## Files to Create

### OVHcloud Provider
```
deployments/pulumi/pkg/provider/ovh/
├── provider.go      # Provider struct and interface impl
├── compute.go       # CreateServer implementation
├── network.go       # CreateNetwork, CreateFirewall
├── storage.go       # CreateVolume implementation
└── loadbalancer.go  # CreateLoadBalancer implementation
```

### Exoscale Provider
```
deployments/pulumi/pkg/provider/exoscale/
├── provider.go
├── compute.go
├── network.go
├── storage.go
└── loadbalancer.go
```

### Contabo Provider
```
deployments/pulumi/pkg/provider/contabo/
├── provider.go
├── compute.go
├── network.go
├── storage.go
└── loadbalancer.go
```

## Files to Modify

| File | Changes |
|------|---------|
| `deployments/pulumi/main.go` | Add provider cases for ovh, exoscale, contabo |
| `deployments/pulumi/pkg/config/config.go` | Add defaults for new providers |
| `deployments/pulumi/pkg/output/cost.go` | Add cost estimates for new providers |
| `deployments/pulumi/go.mod` | Add new Pulumi provider dependencies |

## Task Breakdown

### Phase 1: OVHcloud Provider (~3,000 LOC)
1. [ ] Add pulumi-ovh dependency to go.mod
2. [ ] Create `pkg/provider/ovh/provider.go` - Provider struct
3. [ ] Create `pkg/provider/ovh/network.go` - VPC, Firewall
4. [ ] Create `pkg/provider/ovh/compute.go` - Instances
5. [ ] Create `pkg/provider/ovh/storage.go` - Block storage
6. [ ] Create `pkg/provider/ovh/loadbalancer.go` - Load balancer
7. [ ] Update main.go with OVH case
8. [ ] Add OVH cost estimation
9. [ ] Add OVH config defaults

### Phase 2: Exoscale Provider (~2,500 LOC)
1. [ ] Add pulumi-exoscale dependency
2. [ ] Create `pkg/provider/exoscale/provider.go`
3. [ ] Create `pkg/provider/exoscale/network.go`
4. [ ] Create `pkg/provider/exoscale/compute.go`
5. [ ] Create `pkg/provider/exoscale/storage.go`
6. [ ] Create `pkg/provider/exoscale/loadbalancer.go`
7. [ ] Update main.go with Exoscale case
8. [ ] Add Exoscale cost estimation
9. [ ] Add Exoscale config defaults

### Phase 3: Contabo Provider (~2,500 LOC)
1. [ ] Research Contabo API and Pulumi provider availability
2. [ ] Create `pkg/provider/contabo/provider.go`
3. [ ] Create `pkg/provider/contabo/network.go`
4. [ ] Create `pkg/provider/contabo/compute.go`
5. [ ] Create `pkg/provider/contabo/storage.go`
6. [ ] Create `pkg/provider/contabo/loadbalancer.go`
7. [ ] Update main.go with Contabo case
8. [ ] Add Contabo cost estimation

### Phase 4: Documentation & Testing
1. [ ] Add provider-specific stack configs (Pulumi.ovh.yaml, etc.)
2. [ ] Update README with provider comparison
3. [ ] Create deployment guide per provider
4. [ ] Verify builds and lint pass

## Provider API Research

### OVHcloud
- **Pulumi Provider:** `github.com/ovh/pulumi-ovh/sdk/go/ovh`
- **Regions:** GRA (France), SBG (France), UK, DE, PL
- **Compute:** Instances via OpenStack
- **Network:** vRack (private network), security groups
- **Storage:** Block storage
- **Load Balancer:** OVH Load Balancer service

### Exoscale
- **Pulumi Provider:** `github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale`
- **Regions:** CH-GVA-2, CH-DK-2, DE-FRA-1, DE-MUC-1, AT-VIE-1, etc.
- **Compute:** Compute instances
- **Network:** Private networks, security groups
- **Storage:** Block storage, SOS (S3-compatible)
- **Load Balancer:** Network Load Balancer

### Contabo
- **Pulumi Provider:** Community provider exists but limited
- **Regions:** EU, US, Asia
- **Compute:** VPS (API available)
- **Network:** Limited - may need manual VPN setup
- **Storage:** Block storage available
- **Load Balancer:** Not available via API - may need HAProxy

## Cost Estimation (Monthly EUR)

| Provider | Control Plane | 2 Workers | Storage | LB | Total |
|----------|---------------|-----------|---------|-----|-------|
| Hetzner | €4.35 | €15.18 | €2.35 | €5.39 | ~€27 |
| Scaleway | €9.99 | €35.98 | €4.00 | €9.99 | ~€60 |
| OVHcloud | €8.00 | €32.00 | €3.00 | €10.00 | ~€53 |
| Exoscale | €12.00 | €48.00 | €5.00 | €15.00 | ~€80 |
| Contabo | €5.00 | €10.00 | €2.00 | N/A | ~€17* |

*Contabo requires manual load balancer setup

## Dependencies

```go
// go.mod additions
require (
    github.com/ovh/pulumi-ovh/sdk/go/ovh v0.x.x
    github.com/pulumiverse/pulumi-exoscale/sdk/go/exoscale v0.x.x
    // Contabo - TBD based on provider availability
)
```

## Verification

```bash
# Build verification
cd deployments/pulumi && go build ./...

# Provider-specific testing (requires credentials)
pulumi config set provider ovh
pulumi preview

pulumi config set provider exoscale
pulumi preview

pulumi config set provider contabo
pulumi preview
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Contabo API limitations | May not support all features | Implement partial support, document limitations |
| Provider SDK breaking changes | Build failures | Pin specific versions |
| Different networking models | Complex implementation | Abstract behind CloudProvider interface |

## Timeline Estimate

- Phase 1 (OVHcloud): ~8-10 hours
- Phase 2 (Exoscale): ~6-8 hours
- Phase 3 (Contabo): ~8-10 hours
- Phase 4 (Docs/Testing): ~4-6 hours

**Total:** ~26-34 hours
