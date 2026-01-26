# philotes-worker

Helm chart for the Philotes CDC Worker - Change Data Capture pipeline processor.

## Installation

```bash
helm install philotes-worker ./charts/philotes-worker \
  --set source.host=your-source-db.example.com \
  --set source.database=your_database \
  --set source.user=replication_user
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/janovincze/philotes-worker` |
| `source.host` | Source PostgreSQL host | `""` |
| `source.port` | Source PostgreSQL port | `5432` |
| `source.database` | Source database name | `""` |
| `source.user` | Source database user | `""` |
| `source.existingSecret` | Secret for source password | `""` |
| `database.host` | Buffer database host | `postgresql` |
| `storage.endpoint` | MinIO/S3 endpoint | `""` |
| `storage.bucket` | Storage bucket | `philotes` |
| `storage.existingSecret` | Secret for storage credentials | `""` |
| `iceberg.catalogUrl` | Lakekeeper catalog URL | `""` |
| `keda.enabled` | Enable KEDA autoscaling | `false` |
| `keda.minReplicas` | KEDA minimum replicas | `1` |
| `keda.maxReplicas` | KEDA maximum replicas | `5` |

### CDC Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `cdc.bufferSize` | Event buffer size | `10000` |
| `cdc.batchSize` | Batch size for flushing | `1000` |
| `cdc.flushInterval` | Flush interval | `5s` |
| `cdc.replication.slotName` | Replication slot name | `philotes_cdc` |
| `cdc.replication.publicationName` | Publication name | `philotes_pub` |
| `cdc.checkpoint.enabled` | Enable checkpointing | `true` |
| `cdc.retry.maxAttempts` | Max retry attempts | `3` |
| `cdc.deadLetter.enabled` | Enable dead-letter queue | `true` |
| `cdc.backpressure.enabled` | Enable backpressure | `true` |

### Source Database Setup

Before using the worker, configure your source PostgreSQL:

```sql
-- Enable logical replication
ALTER SYSTEM SET wal_level = logical;
ALTER SYSTEM SET max_replication_slots = 10;
ALTER SYSTEM SET max_wal_senders = 10;
-- Restart PostgreSQL

-- Create replication slot
SELECT pg_create_logical_replication_slot('philotes_cdc', 'pgoutput');

-- Create publication
CREATE PUBLICATION philotes_pub FOR TABLE table1, table2;
```

### Using Existing Secrets

```yaml
source:
  existingSecret: "source-db-credentials"  # key: password

storage:
  existingSecret: "minio-credentials"  # keys: access-key, secret-key

database:
  existingSecret: "buffer-db-credentials"  # key: password
```

### KEDA Autoscaling

Scale workers based on PostgreSQL replication lag:

```yaml
keda:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
  pollingInterval: 30
  cooldownPeriod: 300
  postgresql:
    host: source-db.example.com
    database: mydb
    user: monitor
    existingSecret: keda-pg-credentials
  triggers:
    - type: postgresql
      metadata:
        query: "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn) FROM pg_replication_slots WHERE slot_name = 'philotes_cdc'"
        targetQueryValue: "10000000"  # Scale up when lag > 10MB
```

**Prerequisites for KEDA:**
- KEDA must be installed in your cluster
- PostgreSQL user needs permission to query `pg_replication_slots`

## Health Endpoints

| Endpoint | Port | Description |
|----------|------|-------------|
| `/health` | 8081 | Overall status |
| `/health/live` | 8081 | Liveness probe |
| `/health/ready` | 8081 | Readiness probe |
| `/metrics` | 9090 | Prometheus metrics |

## Resources

Default resource limits (workers are CPU/memory intensive):

```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi
```
