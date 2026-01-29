# Issue #53: Deploy Helm charts from OCI registry instead of local paths

## Summary

Configure Pulumi to deploy the Philotes Helm chart from an OCI registry (GitHub Container Registry) instead of relying on relative file paths that break in CI/CD environments.

## Current Problem

In `pkg/platform/philotes.go`, the Helm chart is deployed using a relative path:
```go
chartPath := "../charts/philotes"
```

This causes:
- Path fragility in CI/CD environments
- No version management for charts
- Reproducibility issues across environments

## Proposed Solution

1. Publish Helm charts to GHCR (`oci://ghcr.io/janovincze/philotes/charts/philotes`)
2. Update Pulumi to deploy from OCI registry by default
3. Add chart version configuration
4. Keep local path override for development

## Acceptance Criteria

- [ ] Helm charts published to GHCR on release
- [ ] Pulumi deploys from OCI registry by default
- [ ] Chart version configurable via Pulumi config
- [ ] Local path override available for development
- [ ] CI/CD pipeline uses registry-based deployment

## Labels

- `epic:infrastructure`
- `phase:v1`
- `priority:medium`
- `type:infra`
