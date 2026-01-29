# Issue #31: INSTALL-001 - One-Click Cloud Installer

## Summary

Web-based installer that enables anyone to deploy Philotes to their preferred cloud provider in under 15 minutes through a guided web interface, without requiring infrastructure expertise.

## Target Users

- Data teams without dedicated DevOps support
- Organizations evaluating Philotes before investing in expertise
- Developers wanting quick proof-of-concept deployments

## Installation Flow

1. User visits install.philotes.io (or self-hosted installer)
2. Select cloud provider (Hetzner, OVHcloud, Scaleway, Exoscale, Contabo)
3. OAuth login to cloud provider
4. Choose deployment size (small/medium/large) with cost estimates
5. Configure region and basic settings
6. One-click deploy with real-time progress
7. Automatic DNS setup (optional)
8. Post-install wizard for first pipeline

## Acceptance Criteria

- [ ] Landing page with provider selection
- [ ] Cloud provider OAuth integration (API tokens)
- [ ] Cost calculator with real-time pricing
- [ ] Deployment size presets (Small ~€30/mo, Medium ~€60/mo, Large ~€150/mo)
- [ ] Region selection with latency indicators
- [ ] Real-time deployment progress (WebSocket)
- [ ] Automatic SSL/TLS certificate provisioning
- [ ] DNS configuration helper
- [ ] Health check verification before handoff
- [ ] Rollback on failure with cleanup

## Tech Stack

- Next.js frontend (same as dashboard)
- WebSocket for real-time progress
- Pulumi Automation API for deployments
- Provider SDKs for OAuth and pricing

## Security

- Cloud tokens encrypted at rest
- Short-lived tokens (auto-expire after deployment)
- No token storage after successful deployment
- Audit logging for all deployment actions

## Dependencies

- FOUND-003 (Pulumi IaC modules) - ✅ Completed
- Issue #3 (Multi-provider IaC) - ✅ Completed
- Issue #53 (Helm OCI registry) - ✅ Completed
- Issue #54 (SSH key management) - ✅ Completed

## Blocks

- INSTALL-002 (Cloud Provider OAuth Integration)
- INSTALL-003 (Deployment Progress Dashboard)
- INSTALL-004 (Post-Installation Setup Wizard)

## Estimate

~15,000 LOC (large feature)

## Labels

- `epic:installation`
- `phase:mvp`
- `priority:high`
