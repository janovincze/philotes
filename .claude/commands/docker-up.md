# Start Development Environment

Start the Docker development environment for Philotes.

## Quick Start

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Start all services
docker compose -f deployments/docker/docker-compose.yml up -d

# Check status
docker compose -f deployments/docker/docker-compose.yml ps

# View logs
docker compose -f deployments/docker/docker-compose.yml logs -f
```

## Services Started

| Service         | Port  | Description                    |
|-----------------|-------|--------------------------------|
| PostgreSQL      | 5432  | Metadata + buffer database     |
| MinIO           | 9000  | S3-compatible object storage   |
| MinIO Console   | 9001  | MinIO web UI                   |
| Lakekeeper      | 8181  | Iceberg REST catalog           |
| Prometheus      | 9090  | Metrics collection             |
| Grafana         | 3001  | Monitoring dashboards          |

## Default Credentials

| Service    | Username  | Password       |
|------------|-----------|----------------|
| PostgreSQL | philotes  | philotes_dev   |
| MinIO      | minioadmin| minioadmin     |
| Grafana    | admin     | admin          |

## Service URLs

- **MinIO Console**: http://localhost:9001
- **Lakekeeper API**: http://localhost:8181
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001

## Available Commands

| Command                                                              | Description              |
|----------------------------------------------------------------------|--------------------------|
| `docker compose -f deployments/docker/docker-compose.yml up -d`     | Start services           |
| `docker compose -f deployments/docker/docker-compose.yml down`      | Stop services            |
| `docker compose -f deployments/docker/docker-compose.yml ps`        | Show status              |
| `docker compose -f deployments/docker/docker-compose.yml logs -f`   | Follow logs              |
| `docker compose -f deployments/docker/docker-compose.yml restart`   | Restart all              |

## Verify Services

After starting, verify all services are healthy:

```bash
# Check PostgreSQL
docker compose -f deployments/docker/docker-compose.yml exec postgres pg_isready

# Check MinIO
curl -s http://localhost:9000/minio/health/live

# Check Lakekeeper
curl -s http://localhost:8181/catalog/v1/config

# Check Prometheus
curl -s http://localhost:9090/-/healthy
```

## Development Workflow

1. Start Docker services: `docker compose up -d`
2. Run API locally: `go run cmd/philotes-api/main.go`
3. Run worker locally: `go run cmd/philotes-worker/main.go`
4. Run dashboard: `cd web && pnpm dev`

## Stopping Services

```bash
# Stop but keep volumes
docker compose -f deployments/docker/docker-compose.yml down

# Stop and remove volumes (clean reset)
docker compose -f deployments/docker/docker-compose.yml down -v
```

## Troubleshooting

### Port Already in Use

```bash
# Find what's using a port
lsof -i :5432

# Kill the process
kill -9 <PID>
```

### Container Won't Start

```bash
# View container logs
docker compose -f deployments/docker/docker-compose.yml logs postgres

# Rebuild container
docker compose -f deployments/docker/docker-compose.yml build --no-cache postgres
```

### Reset Everything

```bash
# Stop all, remove volumes, and restart
docker compose -f deployments/docker/docker-compose.yml down -v
docker compose -f deployments/docker/docker-compose.yml up -d
```
