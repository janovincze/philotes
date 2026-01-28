# Issue #52 - Dynamic cloud-init IP Generation

## Title
feat(pulumi): Dynamic cloud-init generation with actual server IPs

## Labels
- epic:infrastructure
- phase:v1
- priority:high
- type:infra

## Problem Statement

The current Pulumi IaC implementation uses hardcoded placeholder IPs in cloud-init scripts:

1. **Control plane**: Uses `"0.0.0.0"` for TLS-SAN - doesn't include actual public IP
2. **Workers**: Use hardcoded `"10.0.1.1"` to join control plane - fails if control plane gets different IP

## Why This Matters

1. **Reliability**: Workers assume control plane gets IP `10.0.1.1`. If cloud provider assigns different private IP, workers fail to join.
2. **Multi-region**: Different subnets have different IP ranges
3. **HA Scaling**: Multiple control plane nodes need actual IPs

## Current Code Locations

- `pkg/platform/platform.go:57` - Control plane cloud-init with placeholder
- `pkg/platform/platform.go:80` - Worker cloud-init with hardcoded IP
- `pkg/cluster/k3s.go` - Cloud-init script templates

## Acceptance Criteria

- [ ] Control plane TLS-SAN includes actual public IP
- [ ] Workers dynamically discover control plane private IP
- [ ] Cluster forms successfully without manual intervention
- [ ] Works across Hetzner and Scaleway providers

## Dependencies

- Related to FOUND-003 (Infrastructure as Code)
- Affects cluster reliability in production

## Proposed Approach

Use Pulumi's `Apply` for two-phase deployment:
1. Create control plane server, get assigned IPs
2. Use `Apply` to generate worker cloud-init with actual control plane IP
