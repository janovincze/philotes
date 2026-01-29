# Implementation Plan: Issue #53 - Deploy Helm Charts from OCI Registry

## Summary

Update Pulumi to deploy the Philotes Helm chart from GHCR OCI registry instead of local paths, with configuration for chart versions and a development fallback.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Pulumi Config                             │
│  chartRegistry: oci://ghcr.io/janovincze/philotes/charts    │
│  chartVersion: 0.1.0                                         │
│  useLocalCharts: false                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 philotes.go                                  │
│  if useLocalCharts → ../charts/philotes                     │
│  else → oci://ghcr.io/.../philotes:version                  │
└─────────────────────────────────────────────────────────────┘
```

## Files to Create

| File | Description |
|------|-------------|
| `.github/workflows/helm-release.yml` | GitHub Action to publish Helm charts to GHCR |

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add ChartRegistry, ChartVersion, UseLocalCharts |
| `pkg/platform/philotes.go` | Use OCI URL from config, fallback to local |
| `Pulumi.yaml` | Document new config options |

## Implementation Tasks

### Task 1: Update Configuration (`pkg/config/config.go`)

Add new fields to Config struct:
```go
type Config struct {
    // ... existing fields ...

    // ChartRegistry is the OCI registry URL for Helm charts.
    // Example: oci://ghcr.io/janovincze/philotes/charts
    ChartRegistry string

    // ChartVersion is the version of the Philotes Helm chart.
    ChartVersion string

    // UseLocalCharts enables local chart paths for development.
    UseLocalCharts bool
}
```

Load from Pulumi config with defaults:
- `chartRegistry`: `oci://ghcr.io/janovincze/philotes/charts`
- `chartVersion`: `0.1.0`
- `useLocalCharts`: `false`

### Task 2: Update Philotes Deployment (`pkg/platform/philotes.go`)

```go
func DeployPhilotes(ctx *pulumi.Context, cfg *config.Config, k8s *kubernetes.Provider) (*helmv4.Chart, error) {
    // ... values setup ...

    var chartArgs *helmv4.ChartArgs

    if cfg.UseLocalCharts {
        // Development: use local chart path
        ctx.Log.Info("Using local Helm chart for development", nil)
        chartArgs = &helmv4.ChartArgs{
            Chart:     pulumi.String("../charts/philotes"),
            Values:    values,
            Namespace: pulumi.String("philotes"),
        }
    } else {
        // Production: use OCI registry
        chartURL := fmt.Sprintf("%s/philotes", cfg.ChartRegistry)
        ctx.Log.Info(fmt.Sprintf("Using Helm chart from OCI registry: %s:%s", chartURL, cfg.ChartVersion), nil)
        chartArgs = &helmv4.ChartArgs{
            Chart:     pulumi.String(chartURL),
            Version:   pulumi.String(cfg.ChartVersion),
            Values:    values,
            Namespace: pulumi.String("philotes"),
        }
    }

    return helmv4.NewChart(ctx, cfg.ResourceName("philotes"), chartArgs, pulumi.Provider(k8s))
}
```

### Task 3: Create Helm Release Workflow (`.github/workflows/helm-release.yml`)

```yaml
name: Helm Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: read
  packages: write

env:
  REGISTRY: ghcr.io

jobs:
  helm:
    name: Package and Push Helm Charts
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Set up Helm
        uses: azure/setup-helm@v4

      - name: Log in to GHCR
        run: echo "${{ secrets.GITHUB_TOKEN }}" | helm registry login ${{ env.REGISTRY }} -u ${{ github.actor }} --password-stdin

      - name: Package and push charts
        run: |
          VERSION=${GITHUB_REF_NAME#v}
          for chart in philotes philotes-api philotes-worker lakekeeper; do
            helm package charts/$chart --version $VERSION --app-version $VERSION
            helm push $chart-$VERSION.tgz oci://${{ env.REGISTRY }}/${{ github.repository_owner }}/philotes/charts
          done
```

### Task 4: Update Pulumi Config Documentation

Add to `Pulumi.yaml`:
```yaml
config:
  # Helm Chart Configuration
  # chartRegistry: OCI registry URL (default: oci://ghcr.io/janovincze/philotes/charts)
  # chartVersion: Chart version to deploy (default: 0.1.0)
  # useLocalCharts: Use local chart paths for development (default: false)
```

## Configuration Examples

### Development (local charts)
```bash
pulumi config set philotes:useLocalCharts true
```

### Production (OCI registry)
```bash
pulumi config set philotes:chartVersion "1.0.0"
# chartRegistry defaults to GHCR
# useLocalCharts defaults to false
```

## Verification

```bash
# Build
cd deployments/pulumi && go build ./...

# Preview with local charts
pulumi config set philotes:useLocalCharts true
pulumi preview

# Preview with OCI registry (requires published charts)
pulumi config set philotes:useLocalCharts false
pulumi config set philotes:chartVersion "0.1.0"
pulumi preview
```

## Task Order

1. Update `pkg/config/config.go` - Add chart configuration fields
2. Update `pkg/platform/philotes.go` - Use OCI registry from config
3. Create `.github/workflows/helm-release.yml` - Chart publishing workflow
4. Build verification
5. Documentation and PR

## Notes

- External charts (cert-manager, ingress-nginx, monitoring) already use registry URLs
- Only the Philotes umbrella chart and sub-charts need OCI support
- GHCR authentication handled by GitHub Actions token
- Chart dependencies in `Chart.yaml` will still use `file://` paths when packaging (Helm resolves these during `helm package`)
