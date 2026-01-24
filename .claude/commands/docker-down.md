# Stop Development Environment

Stop the Docker development environment for Philotes.

## Quick Stop

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Stop all services (keep data)
docker compose -f deployments/docker/docker-compose.yml down

# Stop all services and remove data
docker compose -f deployments/docker/docker-compose.yml down -v
```

## Options

| Command                                                                    | Effect                                |
|---------------------------------------------------------------------------|---------------------------------------|
| `docker compose -f deployments/docker/docker-compose.yml down`            | Stop containers, keep volumes         |
| `docker compose -f deployments/docker/docker-compose.yml down -v`         | Stop containers, remove volumes       |
| `docker compose -f deployments/docker/docker-compose.yml down --rmi all`  | Stop, remove volumes and images       |

## Verify Stopped

```bash
# Check no containers running
docker compose -f deployments/docker/docker-compose.yml ps

# Check ports are free
lsof -i :5432
lsof -i :9000
lsof -i :8181
```

## Clean Up

```bash
# Remove unused Docker resources
docker system prune

# Remove all unused volumes
docker volume prune

# Full cleanup (careful - removes all unused Docker resources)
docker system prune -a --volumes
```
