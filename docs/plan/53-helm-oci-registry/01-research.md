# Research Findings: Issue #53 - Deploy Helm Charts from OCI Registry

## Current State Analysis

### Helm Chart Deployments in Pulumi

| Chart | File | Source | Status |
|-------|------|--------|--------|
| **Philotes** | `philotes.go` | `../charts/philotes` (local) | ❌ Needs OCI |
| Cert-Manager | `certmanager.go` | `https://charts.jetstack.io` | ✅ Registry |
| Ingress-Nginx | `ingress.go` | `https://kubernetes.github.io/ingress-nginx` | ✅ Registry |
| Monitoring | `monitoring.go` | `https://prometheus-community.github.io/helm-charts` | ✅ Registry |

**Key Finding:** Only the Philotes chart uses local paths. External charts are already properly configured.

### Local Helm Charts Structure

```
charts/
├── philotes/          (v0.1.0) - UMBRELLA CHART
│   ├── Chart.yaml     - dependencies use file:// paths
│   └── values.yaml
├── philotes-api/      (v0.1.0)
├── philotes-worker/   (v0.1.0)
└── lakekeeper/        (v0.1.0)
```

### Umbrella Chart Dependencies (Chart.yaml)

```yaml
dependencies:
  - name: philotes-api
    repository: "file://../philotes-api"    # LOCAL PATH
  - name: philotes-worker
    repository: "file://../philotes-worker" # LOCAL PATH
  - name: lakekeeper
    repository: "file://../lakekeeper"      # LOCAL PATH
  - name: postgresql
    repository: "https://charts.bitnami.com/bitnami"  # Already external
  - name: minio
    repository: "https://charts.min.io/"              # Already external
```

### CI/CD Status

- Docker images published to GHCR: `ghcr.io/janovincze/philotes-*`
- **No Helm chart publishing** in release workflow
- Registry infrastructure ready (GHCR already in use)

### Configuration Status

- No chart version or registry configuration in Pulumi config
- Config system can be extended easily

## Recommended Approach

### Files to Modify

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add ChartRegistry, ChartVersion, UseLocalCharts |
| `pkg/platform/philotes.go` | Use OCI URL from config |
| `Pulumi.yaml` | Add chart configuration keys |
| `.github/workflows/release.yml` | Add Helm chart publishing |

### Configuration Design

```yaml
config:
  philotes:chartRegistry: oci://ghcr.io/janovincze/philotes/charts
  philotes:chartVersion: "0.1.0"
  philotes:useLocalCharts: false  # true for development
```

### Pulumi Helm v4 OCI Support

```go
// OCI registry deployment
chart, err := helmv4.NewChart(ctx, name, &helmv4.ChartArgs{
    Chart:   pulumi.String("oci://ghcr.io/janovincze/philotes/charts/philotes"),
    Version: pulumi.String("0.1.0"),
})

// Local path fallback for development
chart, err := helmv4.NewChart(ctx, name, &helmv4.ChartArgs{
    Chart: pulumi.String("../charts/philotes"),
})
```
