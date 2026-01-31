# Trino Configuration and Integration Plan (Issue #21)

## Summary

Integrate Trino as the primary SQL query engine for Philotes' Iceberg data lake, enabling BI tools and analysts to query replicated data using standard SQL.

## Approach

Following existing project patterns:
1. **Docker Compose** - Add Trino service for local development
2. **Helm Chart** - Create `charts/trino/` for Kubernetes deployment
3. **Configuration** - Extend `internal/config/config.go` with TrinoConfig
4. **API Endpoints** - Add query layer status/health endpoints
5. **Sample Queries** - Provide documentation and examples

## Files to Create

| File | Purpose |
|------|---------|
| `charts/trino/Chart.yaml` | Helm chart definition |
| `charts/trino/values.yaml` | Default configuration values |
| `charts/trino/templates/deployment.yaml` | Coordinator deployment |
| `charts/trino/templates/deployment-worker.yaml` | Worker deployment |
| `charts/trino/templates/configmap.yaml` | Trino configuration |
| `charts/trino/templates/service.yaml` | Kubernetes service |
| `charts/trino/templates/ingress.yaml` | External access |
| `charts/trino/templates/hpa.yaml` | Horizontal Pod Autoscaler |
| `charts/trino/templates/servicemonitor.yaml` | Prometheus metrics |
| `charts/trino/templates/_helpers.tpl` | Template helpers |
| `deployments/docker/trino/config.properties` | Coordinator config |
| `deployments/docker/trino/node.properties` | Node properties |
| `deployments/docker/trino/jvm.config` | JVM settings |
| `deployments/docker/trino/catalog/iceberg.properties` | Iceberg catalog |
| `internal/api/handlers/query.go` | Query layer API handlers |
| `internal/api/models/query.go` | Query layer models |
| `internal/api/services/query.go` | Query layer service |
| `docs/query/trino-setup.md` | Setup documentation |
| `docs/query/sample-queries.sql` | Sample SQL queries |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add TrinoConfig struct (~40 LOC) |
| `internal/api/server.go` | Register query layer endpoints (~5 LOC) |
| `deployments/docker/docker-compose.yml` | Add Trino service (~20 LOC) |
| `charts/philotes/Chart.yaml` | Add Trino dependency |
| `charts/philotes/values.yaml` | Add Trino values |

## Docker Compose Configuration

```yaml
trino:
  image: trinodb/trino:457
  container_name: philotes-trino
  hostname: trino
  ports:
    - "8085:8080"  # Trino UI (8085 to avoid conflict)
  volumes:
    - ./trino:/etc/trino:ro
  environment:
    - TRINO_ENVIRONMENT=development
  depends_on:
    - minio
    - lakekeeper
  networks:
    - philotes-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/v1/info"]
    interval: 10s
    timeout: 5s
    retries: 5
```

## Trino Catalog Configuration

```properties
# catalog/iceberg.properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=http://lakekeeper:8181
iceberg.rest-catalog.warehouse=philotes
iceberg.file-format=PARQUET
fs.native-s3.enabled=true
s3.endpoint=http://minio:9000
s3.path-style-access=true
s3.region=us-east-1
s3.aws-access-key=minioadmin
s3.aws-secret-key=minioadmin
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/query/status` | Query layer status |
| `GET` | `/api/v1/query/catalogs` | List Trino catalogs |
| `GET` | `/api/v1/query/catalogs/:catalog/schemas` | List schemas |
| `GET` | `/api/v1/query/catalogs/:catalog/schemas/:schema/tables` | List tables |
| `GET` | `/api/v1/query/health` | Health check |

## Helm Chart Values Structure

```yaml
trino:
  enabled: true
  image:
    repository: trinodb/trino
    tag: "457"
  coordinator:
    replicas: 1
    resources:
      requests:
        memory: "2Gi"
        cpu: "1"
      limits:
        memory: "4Gi"
        cpu: "2"
  worker:
    replicas: 2
    autoscaling:
      enabled: true
      minReplicas: 1
      maxReplicas: 5
      targetCPUUtilizationPercentage: 70
    resources:
      requests:
        memory: "4Gi"
        cpu: "2"
      limits:
        memory: "8Gi"
        cpu: "4"
  catalogs:
    iceberg:
      connector: iceberg
      catalogType: rest
      restCatalogUri: "http://lakekeeper:8181"
      warehouse: philotes
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: false
    className: nginx
    hosts: []
  auth:
    enabled: false
    type: password  # password, ldap, oidc
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
```

## Configuration Struct

```go
type TrinoConfig struct {
    Enabled      bool   `envconfig:"PHILOTES_TRINO_ENABLED" default:"false"`
    URL          string `envconfig:"PHILOTES_TRINO_URL" default:"http://localhost:8085"`
    Username     string `envconfig:"PHILOTES_TRINO_USERNAME" default:""`
    Password     string `envconfig:"PHILOTES_TRINO_PASSWORD" default:""`
    Catalog      string `envconfig:"PHILOTES_TRINO_CATALOG" default:"iceberg"`
    Schema       string `envconfig:"PHILOTES_TRINO_SCHEMA" default:"philotes"`
    QueryTimeout time.Duration `envconfig:"PHILOTES_TRINO_QUERY_TIMEOUT" default:"5m"`
}
```

## Task Order

1. Create Docker compose Trino configuration files
2. Add Trino service to docker-compose.yml
3. Add TrinoConfig to internal/config/config.go
4. Create API handlers, models, services for query layer
5. Register routes in server.go
6. Create Helm chart for Trino
7. Update umbrella chart with Trino dependency
8. Add documentation and sample queries
9. Run tests and lint

## Verification

1. `docker compose up trino` - Verify Trino starts
2. Access Trino UI at http://localhost:8085
3. Query Iceberg tables via Trino CLI
4. `go build ./...` - Verify Go code compiles
5. `make lint` - Verify code quality
6. `make test` - Verify tests pass

## Estimate

~2,500 LOC total (reduced from original 5,000 LOC estimate as we leverage existing patterns)
