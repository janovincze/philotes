# Philotes Helm Charts

Production-ready Helm charts for deploying Philotes CDC platform to Kubernetes.

## Charts Overview

| Chart | Description |
|-------|-------------|
| [philotes](./philotes/) | Umbrella chart for full-stack deployment |
| [philotes-api](./philotes-api/) | Management REST API server |
| [philotes-worker](./philotes-worker/) | CDC pipeline worker |
| [lakekeeper](./lakekeeper/) | Apache Iceberg REST Catalog |

## Quick Start

### Prerequisites

- Kubernetes 1.25+
- Helm 3.x
- kubectl configured for your cluster

### Install Full Stack (Development)

```bash
# Add chart dependencies
cd charts/philotes
helm dependency update

# Install with development values
helm install philotes ./charts/philotes -f ./charts/philotes/values-dev.yaml
```

### Install Full Stack (Production)

```bash
# Install with production values
helm install philotes ./charts/philotes \
  -f ./charts/philotes/values-production.yaml \
  -n philotes \
  --create-namespace \
  --set worker.source.host=your-source-db.example.com \
  --set worker.source.database=your_database \
  --set worker.source.user=replication_user
```

### Install Individual Components

```bash
# API only
helm install philotes-api ./charts/philotes-api

# Worker only
helm install philotes-worker ./charts/philotes-worker \
  --set source.host=your-source-db.example.com \
  --set source.database=your_database

# Lakekeeper only
helm install lakekeeper ./charts/lakekeeper
```

## Configuration

### Environment-Specific Values

The umbrella chart includes pre-configured values for different environments:

| File | Environment | Description |
|------|-------------|-------------|
| `values.yaml` | Default | Base configuration |
| `values-dev.yaml` | Development | Minimal resources, single replicas |
| `values-staging.yaml` | Staging | HA enabled, HPA, ingress with TLS |
| `values-production.yaml` | Production | Full HA, KEDA scaling, external services |

### Common Configuration Options

#### Database Credentials

Use existing Kubernetes secrets for sensitive data:

```yaml
api:
  database:
    existingSecret: "my-db-credentials"  # Must have key: password

worker:
  source:
    existingSecret: "source-db-credentials"  # Must have key: password
  storage:
    existingSecret: "minio-credentials"  # Must have keys: access-key, secret-key
```

#### Ingress

Enable ingress for external access:

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
      - secretName: philotes-api-tls
        hosts:
          - api.philotes.example.com
```

#### Autoscaling

Enable HPA for the API:

```yaml
api:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 80
```

Enable KEDA for the worker (scales based on replication lag):

```yaml
worker:
  keda:
    enabled: true
    minReplicas: 1
    maxReplicas: 5
    triggers:
      - type: postgresql
        metadata:
          query: "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn) FROM pg_replication_slots WHERE slot_name = 'philotes_cdc'"
          targetQueryValue: "10000000"  # 10MB lag threshold
```

#### Monitoring

Enable Prometheus ServiceMonitors:

```yaml
api:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      labels:
        release: prometheus

worker:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      labels:
        release: prometheus
```

## Source Database Requirements

Before the CDC worker can replicate data, ensure your source PostgreSQL is configured:

```sql
-- Enable logical replication (requires restart)
ALTER SYSTEM SET wal_level = logical;
ALTER SYSTEM SET max_replication_slots = 10;
ALTER SYSTEM SET max_wal_senders = 10;

-- Create replication slot
SELECT pg_create_logical_replication_slot('philotes_cdc', 'pgoutput');

-- Create publication for tables to replicate
CREATE PUBLICATION philotes_pub FOR TABLE your_table1, your_table2;
-- Or for all tables:
CREATE PUBLICATION philotes_pub FOR ALL TABLES;
```

## Accessing Services

### API Server

```bash
# Port forward
kubectl port-forward svc/philotes-api 8080:8080

# Check health
curl http://localhost:8080/health

# List sources
curl http://localhost:8080/api/v1/sources
```

### Worker Health

```bash
kubectl port-forward svc/philotes-worker 8081:8081
curl http://localhost:8081/health
```

### Lakekeeper Catalog

```bash
kubectl port-forward svc/philotes-lakekeeper 8181:8181
curl http://localhost:8181/catalog/v1/config
```

## Upgrading

```bash
# Update dependencies
cd charts/philotes
helm dependency update

# Upgrade release
helm upgrade philotes ./charts/philotes -f ./charts/philotes/values-production.yaml
```

## Uninstalling

```bash
helm uninstall philotes
```

## Troubleshooting

### View Logs

```bash
# API logs
kubectl logs -l app.kubernetes.io/name=philotes-api -f

# Worker logs
kubectl logs -l app.kubernetes.io/name=philotes-worker -f

# Lakekeeper logs
kubectl logs -l app.kubernetes.io/name=lakekeeper -f
```

### Check Pod Status

```bash
kubectl get pods -l app.kubernetes.io/part-of=philotes
kubectl describe pod <pod-name>
```

### Common Issues

1. **Worker can't connect to source database**
   - Verify source database host/port are accessible from the cluster
   - Check NetworkPolicy if enabled
   - Verify credentials in secret

2. **Lakekeeper failing health checks**
   - Ensure PostgreSQL is running and accessible
   - Check database credentials
   - Verify schema exists: `lakekeeper`

3. **KEDA not scaling workers**
   - Verify KEDA is installed in the cluster
   - Check ScaledObject status: `kubectl get scaledobject`
   - Verify PostgreSQL metrics query returns valid data

## Chart Development

### Linting

```bash
helm lint charts/philotes-api
helm lint charts/philotes-worker
helm lint charts/lakekeeper
helm lint charts/philotes
```

### Template Rendering

```bash
helm template philotes ./charts/philotes -f ./charts/philotes/values-dev.yaml
```

### Packaging

```bash
helm package charts/philotes-api
helm package charts/philotes-worker
helm package charts/lakekeeper
helm package charts/philotes
```
