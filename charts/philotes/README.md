# philotes

Umbrella Helm chart for deploying the complete Philotes CDC platform.

This chart includes:
- **philotes-api**: Management REST API
- **philotes-worker**: CDC pipeline worker
- **lakekeeper**: Apache Iceberg REST Catalog
- **postgresql**: Buffer/metadata database (optional, Bitnami chart)
- **minio**: Object storage (optional, MinIO chart)

## Installation

### Quick Start (Development)

```bash
# Update dependencies
cd charts/philotes
helm dependency update

# Install with development values
helm install philotes . -f values-dev.yaml
```

### Production Installation

```bash
helm install philotes . \
  -f values-production.yaml \
  -n philotes \
  --create-namespace \
  --set worker.source.host=prod-db.example.com \
  --set worker.source.database=myapp \
  --set worker.source.user=replication
```

## Environment-Specific Values

| File | Use Case |
|------|----------|
| `values.yaml` | Base configuration |
| `values-dev.yaml` | Local development, minimal resources |
| `values-staging.yaml` | Staging with HA, ingress |
| `values-production.yaml` | Production with KEDA, external services |

## Configuration

### Enable/Disable Components

```yaml
api:
  enabled: true

worker:
  enabled: true

lakekeeper:
  enabled: true

# Use managed services in production
postgresql:
  enabled: false  # Use external PostgreSQL

minio:
  enabled: false  # Use S3 or managed object storage
```

### External PostgreSQL

When using an external PostgreSQL:

```yaml
postgresql:
  enabled: false

api:
  database:
    host: "rds.example.com"
    existingSecret: "external-db-credentials"

worker:
  database:
    host: "rds.example.com"
    existingSecret: "external-db-credentials"

lakekeeper:
  database:
    host: "rds.example.com"
    existingSecret: "external-db-credentials"
```

### External S3/Object Storage

```yaml
minio:
  enabled: false

worker:
  storage:
    endpoint: "s3.amazonaws.com"
    bucket: "my-philotes-bucket"
    useSSL: "true"
    existingSecret: "aws-credentials"
```

### Ingress Configuration

```yaml
api:
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
      - secretName: philotes-tls
        hosts:
          - api.philotes.example.com
```

### Source Database

Configure the PostgreSQL database to replicate FROM:

```yaml
worker:
  source:
    host: "source-db.example.com"
    port: "5432"
    database: "production"
    user: "replication_user"
    sslMode: "require"
    existingSecret: "source-db-credentials"
```

## Dependencies

This chart depends on:

| Chart | Repository | Condition |
|-------|------------|-----------|
| philotes-api | local | `api.enabled` |
| philotes-worker | local | `worker.enabled` |
| lakekeeper | local | `lakekeeper.enabled` |
| postgresql | bitnami | `postgresql.enabled` |
| minio | minio | `minio.enabled` |

Update dependencies:

```bash
helm dependency update
```

## Upgrading

```bash
helm upgrade philotes . -f values-production.yaml
```

## Uninstalling

```bash
helm uninstall philotes
```

**Note:** PersistentVolumeClaims are not deleted automatically. Delete them manually if needed:

```bash
kubectl delete pvc -l app.kubernetes.io/instance=philotes
```

## Architecture

```
                    ┌─────────────┐
                    │   Ingress   │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ philotes-api│
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
   ┌─────▼─────┐    ┌──────▼──────┐   ┌──────▼──────┐
   │ PostgreSQL│    │philotes-    │   │  Prometheus │
   │  (buffer) │    │   worker    │   │  (metrics)  │
   └───────────┘    └──────┬──────┘   └─────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
   ┌─────▼─────┐    ┌──────▼──────┐   ┌──────▼──────┐
   │  Source   │    │  Lakekeeper │   │    MinIO    │
   │ PostgreSQL│    │  (catalog)  │   │  (storage)  │
   └───────────┘    └─────────────┘   └─────────────┘
```
