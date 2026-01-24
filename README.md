# Philotes

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev/)

**Philotes** is an open-source relational database to data lake integration platform with Change Data Capture (CDC) for near-real-time data pipelines. Built for simplicity, reliability, and cost-effectiveness.

## Features

- **CDC Pipeline**: PostgreSQL to Apache Iceberg via pgstream
- **REST Catalog**: Lakekeeper for Iceberg table management
- **S3 Storage**: MinIO for object storage
- **Query Layer**: Support for Trino, RisingWave, and DuckDB
- **Dashboard**: Web UI with setup wizard for easy configuration
- **Auto-scaling**: KEDA-based scaling with customizable policies
- **Multi-cloud IaC**: Pulumi support for Hetzner, OVHcloud, Scaleway, Exoscale, Contabo

## Architecture

```
PostgreSQL ──► pgstream CDC ──► Buffer DB ──► CDC Worker ──► Iceberg (MinIO + Lakekeeper)
                                                    │
                                                    └──► Query Layer: Trino / RisingWave / DuckDB
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Make

### Development Setup

1. **Clone the repository**

```bash
git clone https://github.com/janovincze/philotes.git
cd philotes
```

2. **Start the development environment**

```bash
make docker-up
```

This starts:
- PostgreSQL (buffer database): `localhost:5432`
- PostgreSQL (source for testing): `localhost:5433`
- MinIO: `http://localhost:9000` (console: `http://localhost:9001`)
- Lakekeeper: `http://localhost:8181`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

3. **Build the binaries**

```bash
make build
```

4. **Run the API server**

```bash
make run-api
```

5. **Run tests**

```bash
make test
```

### Default Credentials

| Service | User | Password |
|---------|------|----------|
| PostgreSQL | philotes | philotes |
| PostgreSQL (source) | source | source |
| MinIO | minioadmin | minioadmin |
| Grafana | admin | admin |

## Project Structure

```
philotes/
├── cmd/                    # Application entry points
│   ├── philotes-api/       # Management API server
│   ├── philotes-worker/    # CDC Worker service
│   └── philotes-cli/       # CLI tool
├── internal/               # Private application code
│   ├── api/                # API handlers and middleware
│   ├── cdc/                # CDC pipeline logic
│   ├── config/             # Configuration management
│   ├── iceberg/            # Iceberg integration
│   └── storage/            # Storage abstractions
├── pkg/                    # Public libraries
│   └── client/             # Go client SDK
├── api/openapi/            # OpenAPI specifications
├── deployments/            # Deployment configurations
│   ├── docker/             # Docker Compose for local dev
│   ├── kubernetes/         # Helm charts
│   └── pulumi/             # Infrastructure as Code
├── docs/                   # Documentation
└── web/                    # Dashboard (Next.js)
```

## Make Targets

```bash
make help           # Show all available targets
make build          # Build all binaries
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
make docker-up      # Start development environment
make docker-down    # Stop development environment
make docker-logs    # Show container logs
make check          # Run all checks (lint, vet, test)
```

## Configuration

Philotes is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PHILOTES_API_LISTEN_ADDR` | API server listen address | `:8080` |
| `PHILOTES_DB_HOST` | Database host | `localhost` |
| `PHILOTES_DB_PORT` | Database port | `5432` |
| `PHILOTES_DB_NAME` | Database name | `philotes` |
| `PHILOTES_DB_USER` | Database user | `philotes` |
| `PHILOTES_DB_PASSWORD` | Database password | `philotes` |
| `PHILOTES_ICEBERG_CATALOG_URL` | Lakekeeper URL | `http://localhost:8181` |
| `PHILOTES_STORAGE_ENDPOINT` | MinIO endpoint | `localhost:9000` |

See [internal/config/config.go](internal/config/config.go) for all options.

## API Documentation

Once the API server is running, access the OpenAPI documentation at:
- Swagger UI: `http://localhost:8080/docs`
- OpenAPI spec: `http://localhost:8080/openapi.json`

## Roadmap

- [x] Project foundation and development environment
- [ ] Core CDC pipeline (pgstream → Iceberg)
- [ ] Management API and dashboard
- [ ] Authentication (API keys, SSO/OIDC)
- [ ] Auto-scaling with KEDA
- [ ] Query layer integration (Trino, RisingWave, DuckDB)
- [ ] One-click cloud installer

See our [GitHub Project](https://github.com/users/janovincze/projects/7) for detailed progress.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Philotes is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

- [pgstream](https://github.com/xataio/pgstream) - PostgreSQL CDC library
- [Lakekeeper](https://lakekeeper.io/) - Iceberg REST catalog
- [Apache Iceberg](https://iceberg.apache.org/) - Table format
