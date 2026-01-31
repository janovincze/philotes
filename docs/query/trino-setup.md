# Trino Query Layer Setup

This guide explains how to set up and use Trino as the SQL query engine for Philotes' Iceberg data lake.

## Overview

Trino is a distributed SQL query engine that enables fast, interactive analytics on the Iceberg tables created by Philotes CDC pipelines. It provides:

- Standard SQL interface for querying replicated data
- JDBC/ODBC connectivity for BI tools (Metabase, Superset, Tableau)
- Web UI for query monitoring
- Time travel queries on Iceberg tables

## Local Development Setup

### Docker Compose

The local development environment includes Trino pre-configured with the Iceberg catalog:

```bash
# Start all services including Trino
docker compose -f deployments/docker/docker-compose.yml up -d

# Or start just Trino and dependencies
docker compose -f deployments/docker/docker-compose.yml up -d minio lakekeeper trino
```

### Access Trino

- **Web UI**: http://localhost:8085
- **JDBC URL**: `jdbc:trino://localhost:8085/iceberg/philotes`

### Using Trino CLI

```bash
# Connect to Trino
docker exec -it philotes-trino trino --catalog iceberg --schema philotes

# Run queries
trino> SHOW SCHEMAS;
trino> SHOW TABLES;
trino> SELECT * FROM my_table LIMIT 10;
```

## Kubernetes Deployment

### Using Helm

```bash
# Add Philotes Helm repo
helm repo add philotes https://philotes.example.com/charts

# Install Trino
helm install trino philotes/trino \
  --set catalogs.iceberg.restCatalogUri=http://lakekeeper:8181 \
  --set catalogs.iceberg.s3.endpoint=http://minio:9000 \
  --set catalogs.iceberg.s3.accessKey=$MINIO_ACCESS_KEY \
  --set catalogs.iceberg.s3.secretKey=$MINIO_SECRET_KEY
```

### Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `enabled` | Enable Trino deployment | `true` |
| `coordinator.replicas` | Number of coordinators | `1` |
| `worker.replicas` | Number of workers | `2` |
| `worker.autoscaling.enabled` | Enable HPA for workers | `false` |
| `catalogs.iceberg.restCatalogUri` | Lakekeeper REST catalog URL | `http://lakekeeper:8181` |
| `catalogs.iceberg.warehouse` | Warehouse name | `philotes` |
| `catalogs.iceberg.s3.endpoint` | MinIO/S3 endpoint | `http://minio:9000` |
| `ingress.enabled` | Enable ingress | `false` |

### Using Existing Secrets

For production, use Kubernetes secrets for S3 credentials:

```bash
# Create secret
kubectl create secret generic minio-credentials \
  --from-literal=access-key=$MINIO_ACCESS_KEY \
  --from-literal=secret-key=$MINIO_SECRET_KEY

# Use in Helm values
helm install trino philotes/trino \
  --set catalogs.iceberg.s3.existingSecret=minio-credentials
```

## API Endpoints

Philotes provides REST API endpoints for query layer management:

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/query/status` | Query layer status |
| `GET /api/v1/query/health` | Health check |
| `GET /api/v1/query/catalogs` | List catalogs |
| `GET /api/v1/query/catalogs/{catalog}/schemas` | List schemas |
| `GET /api/v1/query/catalogs/{catalog}/schemas/{schema}/tables` | List tables |
| `GET /api/v1/query/catalogs/{catalog}/schemas/{schema}/tables/{table}` | Table details |

## Connecting BI Tools

### JDBC Connection

Use these parameters for JDBC connections:

- **Driver**: Trino JDBC Driver
- **URL**: `jdbc:trino://trino-host:8080/iceberg/philotes`
- **User**: Any username (when auth is disabled)

### Metabase

1. Add Database → Trino
2. Host: `trino-coordinator` (or `localhost` for local dev)
3. Port: `8080` (or `8085` for local dev)
4. Catalog: `iceberg`
5. Schema: `philotes`

### Apache Superset

```python
# SQLAlchemy URI
trino://user@trino-host:8080/iceberg/philotes
```

## Environment Variables

Configure Trino integration in Philotes:

| Variable | Description | Default |
|----------|-------------|---------|
| `PHILOTES_TRINO_ENABLED` | Enable Trino integration | `false` |
| `PHILOTES_TRINO_URL` | Trino coordinator URL | `http://localhost:8085` |
| `PHILOTES_TRINO_USERNAME` | Trino username | (empty) |
| `PHILOTES_TRINO_PASSWORD` | Trino password | (empty) |
| `PHILOTES_TRINO_CATALOG` | Default catalog | `iceberg` |
| `PHILOTES_TRINO_SCHEMA` | Default schema | `philotes` |

## Troubleshooting

### Trino Cannot Connect to Lakekeeper

Check that Lakekeeper is healthy:

```bash
curl http://localhost:8181/catalog/v1/config
```

### Trino Cannot Access MinIO

Verify MinIO credentials and bucket permissions:

```bash
# Check MinIO health
mc alias set local http://localhost:9000 minioadmin minioadmin
mc ls local/philotes
```

### Query Errors

Check Trino logs:

```bash
docker logs philotes-trino
```

Or in Kubernetes:

```bash
kubectl logs -l app.kubernetes.io/name=trino,app.kubernetes.io/component=coordinator
```
