# lakekeeper

Helm chart for Lakekeeper - Apache Iceberg REST Catalog.

Lakekeeper provides a REST catalog implementation for Apache Iceberg tables, allowing query engines like Trino, Spark, and DuckDB to discover and access Iceberg tables.

## Installation

```bash
helm install lakekeeper ./charts/lakekeeper
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `lakekeeper/catalog` |
| `image.tag` | Image tag | `latest` |
| `database.host` | PostgreSQL host | `postgresql` |
| `database.port` | PostgreSQL port | `5432` |
| `database.name` | Database name | `philotes` |
| `database.schema` | Schema for Lakekeeper tables | `lakekeeper` |
| `database.user` | Database user | `philotes` |
| `database.existingSecret` | Secret for password | `""` |
| `config.authz.enabled` | Enable authorization | `false` |
| `config.openid.enabled` | Enable OpenID Connect | `false` |

### Using Existing Secrets

```yaml
database:
  existingSecret: "my-db-credentials"  # Must have key: password
```

### Enabling Authentication

For production, enable OpenID Connect authentication:

```yaml
config:
  authz:
    enabled: true
  openid:
    enabled: true
    issuerUrl: "https://auth.example.com"
    clientId: "lakekeeper"
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /catalog/v1/config` | Catalog configuration |
| `GET /catalog/v1/namespaces` | List namespaces |
| `GET /catalog/v1/namespaces/{ns}/tables` | List tables |
| `GET /catalog/v1/namespaces/{ns}/tables/{table}` | Get table metadata |

## Usage with Query Engines

### Trino

```properties
connector.name=iceberg
iceberg.catalog.type=rest
iceberg.rest-catalog.uri=http://lakekeeper:8181
```

### Spark

```python
spark = SparkSession.builder \
    .config("spark.sql.catalog.iceberg", "org.apache.iceberg.spark.SparkCatalog") \
    .config("spark.sql.catalog.iceberg.type", "rest") \
    .config("spark.sql.catalog.iceberg.uri", "http://lakekeeper:8181") \
    .getOrCreate()
```

### DuckDB

```sql
INSTALL iceberg;
LOAD iceberg;

SELECT * FROM iceberg_scan('http://lakekeeper:8181/warehouse/db/table');
```

## Resources

Default resource limits:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 256Mi
```
