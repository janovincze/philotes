# philotes-api

Helm chart for the Philotes API Server - Management REST API for CDC pipelines.

## Installation

```bash
helm install philotes-api ./charts/philotes-api
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/janovincze/philotes-api` |
| `image.tag` | Image tag | `""` (uses appVersion) |
| `api.listenAddr` | API listen address | `:8080` |
| `api.corsOrigins` | CORS allowed origins | `*` |
| `api.rateLimitRPS` | Rate limit (req/sec) | `100` |
| `database.host` | PostgreSQL host | `postgresql` |
| `database.port` | PostgreSQL port | `5432` |
| `database.name` | Database name | `philotes` |
| `database.user` | Database user | `philotes` |
| `database.existingSecret` | Existing secret for password | `""` |
| `ingress.enabled` | Enable ingress | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `pdb.enabled` | Enable PodDisruptionBudget | `true` |
| `networkPolicy.enabled` | Enable NetworkPolicy | `false` |
| `metrics.enabled` | Enable Prometheus metrics | `true` |
| `metrics.serviceMonitor.enabled` | Create ServiceMonitor | `false` |

### Using Existing Secrets

Instead of putting passwords in values, use existing secrets:

```yaml
database:
  existingSecret: "my-db-credentials"
```

The secret must have a `password` key:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-db-credentials
type: Opaque
data:
  password: <base64-encoded-password>
```

### Enabling Ingress

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: api.philotes.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: philotes-api-tls
      hosts:
        - api.philotes.example.com
```

### Enabling Autoscaling

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Overall health status |
| `GET /health/live` | Liveness probe |
| `GET /health/ready` | Readiness probe |
| `GET /api/v1/sources` | List CDC sources |
| `GET /api/v1/pipelines` | List pipelines |
| `GET /metrics` | Prometheus metrics (port 9090) |

## Resources

Default resource limits:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```
