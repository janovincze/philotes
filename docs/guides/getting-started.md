# Getting Started with Philotes

This guide walks you through setting up Philotes and running your first CDC pipeline from PostgreSQL to Apache Iceberg.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker** (v20.10+) and **Docker Compose** (v2.0+)
- **Go** (v1.22+) for running the API server locally
- **curl** or similar HTTP client for API interactions
- **git** for cloning the repository

## Quick Start

```bash
# Clone the repository
git clone https://github.com/janovincze/philotes.git
cd philotes

# Start the Docker environment
docker compose -f deployments/docker/docker-compose.yml up -d

# Wait for services to be healthy (~30 seconds)
docker compose -f deployments/docker/docker-compose.yml ps

# Start the API server
go run cmd/philotes-api/main.go
```

## Step 1: Start the Development Environment

The Docker Compose environment includes all necessary services:

```bash
docker compose -f deployments/docker/docker-compose.yml up -d
```

Wait for all services to be healthy:

```bash
docker compose -f deployments/docker/docker-compose.yml ps
```

You should see these services running:

| Service | Port | Description |
|---------|------|-------------|
| postgres | 5432 | Buffer database (metadata storage) |
| postgres-source | 5433 | Source database with sample e-commerce data |
| minio | 9000, 9001 | S3-compatible object storage |
| lakekeeper | 8181 | Iceberg REST catalog |
| trino | 8085 | SQL query engine |
| prometheus | 9090 | Metrics collection |
| grafana | 3000 | Monitoring dashboards |
| vault | 8200 | Secrets management |

### Verify Sample Data

The source database comes pre-loaded with an e-commerce dataset:

```bash
docker exec philotes-postgres-source psql -U source -d source -c "
SELECT 'customers' as table_name, count(*) as row_count FROM customers
UNION ALL
SELECT 'products', count(*) FROM products
UNION ALL
SELECT 'orders', count(*) FROM orders
UNION ALL
SELECT 'order_items', count(*) FROM order_items;
"
```

Expected output:
```
 table_name  | row_count
-------------+-----------
 customers   |       100
 products    |        50
 orders      |       500
 order_items |      ~2000
```

## Step 2: Start the API Server

In a new terminal, start the Philotes API server:

```bash
go run cmd/philotes-api/main.go
```

The API will start on `http://localhost:8080`. Verify it's running:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy"}
```

## Step 3: Create a Source Connection

Create a connection to the source PostgreSQL database:

```bash
curl -X POST http://localhost:8080/api/v1/sources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ecommerce-source",
    "host": "localhost",
    "port": 5433,
    "database_name": "source",
    "username": "source",
    "password": "source",
    "ssl_mode": "disable",
    "slot_name": "philotes_ecommerce",
    "publication_name": "philotes_pub"
  }'
```

Save the returned source ID. Example response:
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "ecommerce-source",
  "host": "localhost",
  "port": 5433,
  "database_name": "source",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Test the Connection

```bash
curl -X POST http://localhost:8080/api/v1/sources/{SOURCE_ID}/test
```

Expected response:
```json
{"success": true, "message": "Connection successful"}
```

### Discover Available Tables

```bash
curl http://localhost:8080/api/v1/sources/{SOURCE_ID}/tables
```

This returns the tables available for CDC:
```json
{
  "tables": [
    {"schema": "public", "name": "customers"},
    {"schema": "public", "name": "products"},
    {"schema": "public", "name": "orders"},
    {"schema": "public", "name": "order_items"}
  ]
}
```

## Step 4: Create a Pipeline

Create a CDC pipeline that will replicate data to Iceberg:

```bash
curl -X POST http://localhost:8080/api/v1/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ecommerce-pipeline",
    "source_id": "{SOURCE_ID}",
    "destination_type": "iceberg",
    "config": {
      "batch_size": 1000,
      "flush_interval_seconds": 10
    }
  }'
```

Save the returned pipeline ID.

## Step 5: Add Table Mappings

Add the tables you want to replicate. Let's start with the `customers` table:

```bash
# Add customers table
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/tables \
  -H "Content-Type: application/json" \
  -d '{
    "source_schema": "public",
    "source_table": "customers",
    "destination_schema": "ecommerce",
    "destination_table": "customers",
    "enabled": true
  }'

# Add products table
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/tables \
  -H "Content-Type: application/json" \
  -d '{
    "source_schema": "public",
    "source_table": "products",
    "destination_schema": "ecommerce",
    "destination_table": "products",
    "enabled": true
  }'

# Add orders table
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/tables \
  -H "Content-Type: application/json" \
  -d '{
    "source_schema": "public",
    "source_table": "orders",
    "destination_schema": "ecommerce",
    "destination_table": "orders",
    "enabled": true
  }'

# Add order_items table
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/tables \
  -H "Content-Type: application/json" \
  -d '{
    "source_schema": "public",
    "source_table": "order_items",
    "destination_schema": "ecommerce",
    "destination_table": "order_items",
    "enabled": true
  }'
```

## Step 6: Start the Pipeline

Start the CDC pipeline to begin replicating data:

```bash
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/start
```

Check the pipeline status:

```bash
curl http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/status
```

Expected response when running:
```json
{
  "status": "running",
  "lsn_position": "0/1234567",
  "events_processed": 0,
  "last_event_at": null
}
```

## Step 7: Insert Test Data

Let's insert some new data to verify CDC is working:

```bash
docker exec philotes-postgres-source psql -U source -d source -c "
INSERT INTO customers (first_name, last_name, email, phone, metadata)
VALUES ('Test', 'User', 'test.user@example.com', '+1-555-9999', '{\"test\": true}');
"
```

Wait a few seconds for the CDC pipeline to process the change.

## Step 8: Verify Data in Iceberg

### Using Trino

Connect to Trino and query the replicated data:

```bash
docker exec -it philotes-trino trino
```

In the Trino CLI:

```sql
-- Show available schemas
SHOW SCHEMAS FROM iceberg;

-- Show tables in the ecommerce schema
SHOW TABLES FROM iceberg.ecommerce;

-- Query the customers table
SELECT * FROM iceberg.ecommerce.customers LIMIT 10;

-- Verify the test user was replicated
SELECT * FROM iceberg.ecommerce.customers
WHERE email = 'test.user@example.com';

-- Count records by table
SELECT 'customers' as table_name, count(*) FROM iceberg.ecommerce.customers
UNION ALL
SELECT 'products', count(*) FROM iceberg.ecommerce.products
UNION ALL
SELECT 'orders', count(*) FROM iceberg.ecommerce.orders
UNION ALL
SELECT 'order_items', count(*) FROM iceberg.ecommerce.order_items;
```

### Using MinIO Console

You can also browse the Parquet files directly:

1. Open http://localhost:9001 in your browser
2. Login with `minioadmin` / `minioadmin`
3. Navigate to the `warehouse` bucket
4. Explore the Iceberg table structure under `ecommerce/`

## Step 9: Test UPDATE and DELETE Operations

CDC captures all change types. Let's test:

```bash
# UPDATE: Change the test user's name
docker exec philotes-postgres-source psql -U source -d source -c "
UPDATE customers SET first_name = 'Updated' WHERE email = 'test.user@example.com';
"

# DELETE: Remove the test user
docker exec philotes-postgres-source psql -U source -d source -c "
DELETE FROM customers WHERE email = 'test.user@example.com';
"
```

Query Trino again to see the changes reflected in the data lake.

## Step 10: Monitor with Grafana

Open Grafana at http://localhost:3000 (login: `admin`/`admin`).

The Philotes dashboard shows:
- Pipeline status and health
- Events processed per second
- Replication lag
- Error rates

## Step 11: Stop the Pipeline

When you're done, stop the pipeline gracefully:

```bash
curl -X POST http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/stop
```

## API Reference

### Sources

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sources` | Create a new source |
| GET | `/api/v1/sources` | List all sources |
| GET | `/api/v1/sources/:id` | Get source details |
| PUT | `/api/v1/sources/:id` | Update a source |
| DELETE | `/api/v1/sources/:id` | Delete a source |
| POST | `/api/v1/sources/:id/test` | Test connection |
| GET | `/api/v1/sources/:id/tables` | Discover tables |

### Pipelines

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/pipelines` | Create a new pipeline |
| GET | `/api/v1/pipelines` | List all pipelines |
| GET | `/api/v1/pipelines/:id` | Get pipeline details |
| PUT | `/api/v1/pipelines/:id` | Update a pipeline |
| DELETE | `/api/v1/pipelines/:id` | Delete a pipeline |
| POST | `/api/v1/pipelines/:id/start` | Start the pipeline |
| POST | `/api/v1/pipelines/:id/stop` | Stop the pipeline |
| GET | `/api/v1/pipelines/:id/status` | Get pipeline status |
| POST | `/api/v1/pipelines/:id/tables` | Add table mapping |
| DELETE | `/api/v1/pipelines/:id/tables/:mappingId` | Remove table mapping |

## Troubleshooting

### Pipeline Won't Start

1. **Check source connection**: Ensure the source database is reachable
   ```bash
   curl -X POST http://localhost:8080/api/v1/sources/{SOURCE_ID}/test
   ```

2. **Check replication slot**: The slot may already exist
   ```bash
   docker exec philotes-postgres-source psql -U source -d source -c "SELECT * FROM pg_replication_slots;"
   ```

3. **Check logs**: View API server logs for detailed error messages

### No Data in Iceberg

1. **Check pipeline status**: Ensure the pipeline is running
   ```bash
   curl http://localhost:8080/api/v1/pipelines/{PIPELINE_ID}/status
   ```

2. **Check Lakekeeper**: Ensure the catalog is accessible
   ```bash
   curl http://localhost:8181/catalog/v1/config?warehouse=ecommerce
   ```

3. **Check MinIO**: Verify the warehouse bucket exists
   ```bash
   docker exec philotes-minio mc ls local/warehouse/
   ```

### Connection Refused

1. **Docker services**: Ensure all services are running
   ```bash
   docker compose -f deployments/docker/docker-compose.yml ps
   ```

2. **Network issues**: When running the API locally, use `localhost` instead of container hostnames

### Reset Everything

To start fresh:

```bash
# Stop and remove all containers and volumes
docker compose -f deployments/docker/docker-compose.yml down -v

# Start again
docker compose -f deployments/docker/docker-compose.yml up -d
```

## Next Steps

- **Dashboard**: Run `cd web && pnpm dev` to start the web dashboard
- **Production deployment**: See the deployment guide for Kubernetes/Helm setup
- **Custom sources**: Learn how to connect other PostgreSQL databases
- **Scaling**: Configure auto-scaling for high-volume workloads

## Getting Help

- **Documentation**: Check the `/docs` folder for detailed documentation
- **Issues**: Report bugs at https://github.com/janovincze/philotes/issues
- **Discussions**: Join the community discussions for questions and ideas
