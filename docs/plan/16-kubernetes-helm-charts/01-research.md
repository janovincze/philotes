# Research - Issue #16: Kubernetes Helm Charts

## Docker Compose Analysis

**Location:** `deployments/docker/docker-compose.yml`

### Services Identified (9 total)

| Service | Image | Ports | Purpose |
|---------|-------|-------|---------|
| postgres | postgres:16 | 5432 | Buffer/metadata database |
| postgres-source | postgres:16 | 5433 | Source for CDC testing |
| minio | minio/minio | 9000, 9001 | S3-compatible storage |
| minio-init | minio/mc | - | Bucket initialization |
| lakekeeper | lakekeeper | 8181 | Iceberg REST Catalog |
| prometheus | prom/prometheus | 9090 | Metrics collection |
| grafana | grafana/grafana | 3000 | Visualization |

### Application Configuration

**Location:** `internal/config/config.go`

#### API Configuration
- `PHILOTES_API_LISTEN_ADDR` (default: `:8080`)
- `PHILOTES_API_READ_TIMEOUT` (default: 15s)
- `PHILOTES_API_WRITE_TIMEOUT` (default: 15s)
- `PHILOTES_API_CORS_ORIGINS` (default: `*`)
- `PHILOTES_API_RATE_LIMIT_RPS` (default: 100)

#### Database Configuration
- `PHILOTES_DB_HOST` (default: localhost)
- `PHILOTES_DB_PORT` (default: 5432)
- `PHILOTES_DB_NAME` (default: philotes)
- `PHILOTES_DB_USER` (default: philotes)
- `PHILOTES_DB_PASSWORD` (default: philotes)
- `PHILOTES_DB_SSLMODE` (default: disable)

#### CDC Configuration
- `PHILOTES_CDC_BUFFER_SIZE` (default: 10000)
- `PHILOTES_CDC_BATCH_SIZE` (default: 1000)
- `PHILOTES_CDC_FLUSH_INTERVAL` (default: 5s)
- `PHILOTES_CDC_SOURCE_HOST/PORT/DATABASE/USER/PASSWORD`
- `PHILOTES_CDC_REPLICATION_SLOT` (default: philotes_cdc)
- `PHILOTES_CDC_PUBLICATION` (default: philotes_pub)

#### Storage Configuration (MinIO/S3)
- `PHILOTES_STORAGE_ENDPOINT` (default: localhost:9000)
- `PHILOTES_STORAGE_ACCESS_KEY/SECRET_KEY`
- `PHILOTES_STORAGE_BUCKET` (default: philotes)

#### Health Configuration
- `PHILOTES_HEALTH_ENABLED` (default: true)
- `PHILOTES_HEALTH_LISTEN_ADDR` (default: `:8081`)

#### Metrics Configuration
- `PHILOTES_METRICS_ENABLED` (default: true)
- `PHILOTES_METRICS_LISTEN_ADDR` (default: `:9090`)

#### Alerting Configuration
- `PHILOTES_ALERTING_ENABLED` (default: true)
- `PHILOTES_PROMETHEUS_URL` (default: `http://localhost:9090`)

## Health Check Endpoints

**API Server (port 8080):**
- `GET /health` - Overall status
- `GET /health/live` - Liveness probe (always 200)
- `GET /health/ready` - Readiness probe

**Worker (port 8081):**
- `GET /health` - Overall status
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe

## Services Requiring Helm Charts

| Service | Type | Main Port | Health Port | Metrics Port | Scaling |
|---------|------|-----------|-------------|--------------|---------|
| philotes-api | Deployment | 8080 | 8080 | 9090 | HPA |
| philotes-worker | Deployment | - | 8081 | 9090 | KEDA |
| philotes-dashboard | Deployment | 3000 | - | - | HPA |

## Recommended Chart Structure

```
charts/
├── philotes/                    # Umbrella chart
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-dev.yaml
│   ├── values-staging.yaml
│   ├── values-prod.yaml
│   └── templates/
│       ├── _helpers.tpl
│       └── namespace.yaml
├── philotes-api/                # API sub-chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       ├── configmap.yaml
│       ├── secret.yaml
│       ├── hpa.yaml
│       ├── pdb.yaml
│       ├── networkpolicy.yaml
│       └── servicemonitor.yaml
├── philotes-worker/             # Worker sub-chart
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── configmap.yaml
│       ├── secret.yaml
│       ├── scaledobject.yaml   # KEDA
│       ├── pdb.yaml
│       └── networkpolicy.yaml
└── philotes-dashboard/          # Dashboard sub-chart (future)
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        ├── deployment.yaml
        ├── service.yaml
        ├── ingress.yaml
        └── configmap.yaml
```

## External Dependencies Strategy

| Dependency | Recommendation | Chart |
|------------|----------------|-------|
| PostgreSQL | External chart or managed service | bitnami/postgresql |
| MinIO | External chart or managed S3 | minio/minio |
| Lakekeeper | Custom deployment (no official chart) | Include in umbrella |
| Prometheus | External stack | prometheus-community/kube-prometheus-stack |
| Grafana | Included with Prometheus stack | - |

## Key Files Analyzed

- `internal/config/config.go` - All configuration options
- `cmd/philotes-api/main.go` - API entry point
- `cmd/philotes-worker/main.go` - Worker entry point
- `internal/cdc/health/health.go` - Health check system
- `internal/api/server.go` - API server implementation
- `deployments/docker/docker-compose.yml` - Docker services
- `deployments/docker/prometheus.yml` - Prometheus config
