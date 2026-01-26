# Implementation Plan - Issue #16: Kubernetes Helm Charts

## Overview

Create production-ready Helm charts for deploying Philotes to Kubernetes clusters. The implementation follows a sub-chart architecture with an umbrella chart for full-stack deployment.

## Phase 1: Chart Structure Setup

### 1.1 Create Directory Structure
```
charts/
├── philotes/                    # Umbrella chart
├── philotes-api/                # API sub-chart
├── philotes-worker/             # Worker sub-chart
└── lakekeeper/                  # Lakekeeper dependency chart
```

### 1.2 Files per Sub-Chart
- Chart.yaml - Chart metadata
- values.yaml - Default configuration
- templates/_helpers.tpl - Template helpers
- templates/deployment.yaml
- templates/service.yaml
- templates/configmap.yaml
- templates/secret.yaml
- templates/pdb.yaml
- templates/networkpolicy.yaml

## Phase 2: philotes-api Chart

### 2.1 Core Resources
- **Deployment**: Main API server with configurable replicas
- **Service**: ClusterIP service on port 8080
- **Ingress**: Optional ingress with TLS support
- **ConfigMap**: Non-sensitive configuration
- **Secret**: Database passwords, API keys

### 2.2 Production Features
- **HPA**: Horizontal Pod Autoscaler (CPU/memory based)
- **PDB**: PodDisruptionBudget (minAvailable: 1)
- **NetworkPolicy**: Restrict ingress/egress
- **ServiceMonitor**: Prometheus metrics scraping

### 2.3 Configuration Variables
```yaml
image:
  repository: ghcr.io/janovincze/philotes-api
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 2

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

api:
  listenAddr: ":8080"
  readTimeout: 15s
  writeTimeout: 15s
  corsOrigins: "*"
  rateLimitRPS: 100

database:
  host: postgresql
  port: 5432
  name: philotes
  user: philotes
  existingSecret: ""

metrics:
  enabled: true
  port: 9090

autoscaling:
  enabled: false
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilization: 80

ingress:
  enabled: false
  className: nginx
  hosts: []
  tls: []
```

## Phase 3: philotes-worker Chart

### 3.1 Core Resources
- **Deployment**: CDC worker(s)
- **Service**: ClusterIP for health checks
- **ConfigMap**: CDC configuration
- **Secret**: Source DB credentials, storage keys

### 3.2 Production Features
- **KEDA ScaledObject**: Scale based on PostgreSQL replication lag
- **PDB**: PodDisruptionBudget
- **NetworkPolicy**: Allow egress to source DB, MinIO, Lakekeeper

### 3.3 Configuration Variables
```yaml
image:
  repository: ghcr.io/janovincze/philotes-worker
  tag: latest

replicaCount: 1

resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

cdc:
  bufferSize: 10000
  batchSize: 1000
  flushInterval: 5s

source:
  host: ""
  port: 5432
  database: ""
  existingSecret: ""

storage:
  endpoint: ""
  bucket: philotes
  existingSecret: ""

iceberg:
  catalogUrl: ""
  warehouse: philotes

keda:
  enabled: false
  minReplicas: 1
  maxReplicas: 5
  pollingInterval: 30
  cooldownPeriod: 300
```

## Phase 4: Lakekeeper Chart

### 4.1 Core Resources
- **Deployment**: Lakekeeper catalog server
- **Service**: ClusterIP on port 8181
- **ConfigMap**: Lakekeeper configuration

### 4.2 Configuration Variables
```yaml
image:
  repository: lakekeeper/catalog
  tag: latest

replicaCount: 1

resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 512Mi

config:
  baseUri: ""
  authz:
    enabled: false
  openid:
    enabled: false

database:
  host: postgresql
  port: 5432
  name: philotes
  schema: lakekeeper
  existingSecret: ""
```

## Phase 5: Umbrella Chart

### 5.1 Dependencies
```yaml
dependencies:
  - name: philotes-api
    version: "0.1.0"
    condition: api.enabled
  - name: philotes-worker
    version: "0.1.0"
    condition: worker.enabled
  - name: lakekeeper
    version: "0.1.0"
    condition: lakekeeper.enabled
  - name: postgresql
    version: "15.x.x"
    repository: https://charts.bitnami.com/bitnami
    condition: postgresql.enabled
  - name: minio
    version: "5.x.x"
    repository: https://charts.min.io/
    condition: minio.enabled
```

### 5.2 Values Files
- **values.yaml**: Default/development values
- **values-staging.yaml**: Staging environment
- **values-production.yaml**: Production environment

### 5.3 Environment Differences

| Setting | Dev | Staging | Prod |
|---------|-----|---------|------|
| API replicas | 1 | 2 | 3 |
| Worker replicas | 1 | 1 | 2 |
| HPA enabled | false | true | true |
| KEDA enabled | false | false | true |
| NetworkPolicy | false | true | true |
| Resource limits | low | medium | high |
| Ingress TLS | false | true | true |

## Phase 6: Template Helpers

### 6.1 Common Helpers (_helpers.tpl)
```
{{- define "philotes.name" -}}
{{- define "philotes.fullname" -}}
{{- define "philotes.chart" -}}
{{- define "philotes.labels" -}}
{{- define "philotes.selectorLabels" -}}
{{- define "philotes.serviceAccountName" -}}
```

## Implementation Order

1. Create philotes-api chart (foundation)
2. Create philotes-worker chart
3. Create lakekeeper chart
4. Create umbrella chart with dependencies
5. Create environment-specific values files
6. Test with `helm template` and `helm lint`

## Testing Strategy

### Lint and Template
```bash
helm lint charts/philotes-api
helm template philotes-api charts/philotes-api
```

### Local Testing (kind/minikube)
```bash
kind create cluster --name philotes-test
helm install philotes charts/philotes -f charts/philotes/values-dev.yaml
```

## Acceptance Criteria Mapping

| Criteria | Implementation |
|----------|---------------|
| Helm chart per service | philotes-api, philotes-worker, lakekeeper |
| Umbrella chart | charts/philotes with dependencies |
| ConfigMaps and Secrets | Per-chart with existingSecret support |
| Service and Ingress | Service always, Ingress optional |
| PodDisruptionBudgets | pdb.yaml in each chart |
| Resource limits/requests | Configurable in values.yaml |
| HPA definitions | hpa.yaml in API chart |
| KEDA ScaledObject | scaledobject.yaml in worker chart |
| NetworkPolicies | networkpolicy.yaml in each chart |
| Values files | values-dev, values-staging, values-prod |
