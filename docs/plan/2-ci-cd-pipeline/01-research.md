# Research Findings: FOUND-002 - CI/CD Pipeline Setup

## Current State Analysis

### 1. Existing CI/CD Setup
- **No GitHub Actions workflows exist** - The `.github/workflows/` directory is empty
- CI/CD needs to be created from scratch

### 2. Build Infrastructure

#### Makefile Targets Available
```makefile
make build          # Builds all 3 binaries (api, worker, cli)
make test           # Runs tests with race detector
make test-coverage  # Generates coverage reports
make lint           # Runs golangci-lint
make fmt            # Formats code
make vet            # Runs go vet
make tidy           # Tidy go modules
make check          # Runs lint, vet, test (all checks)
make docker-up      # Starts development environment
make install-tools  # Installs development tools
```

#### Go Configuration
- **Go Version:** 1.25.5
- **Module:** github.com/janovincze/philotes
- **Three Binaries:** philotes-api, philotes-worker, philotes-cli

### 3. Testing Infrastructure

#### Unit Tests
- 21 test files distributed across internal packages
- Table-driven test patterns
- Tests use standard Go testing library

#### Test Commands
- `go test -v -race -coverprofile=coverage.out ./...`
- Coverage HTML report generation available

#### Integration Test Directories
- `/test/integration/` - exists but empty
- `/test/e2e/` - exists but empty

### 4. Linting Configuration

#### golangci-lint (.golangci.yml)
- 27+ enabled linters configured
- Comprehensive static analysis
- Run timeout: 5 minutes

#### Pre-commit Hooks (.pre-commit-config.yaml)
- Go fmt, imports, vet
- Module tidy
- Hadolint for Dockerfiles
- Markdownlint for documentation

### 5. Docker Configuration

#### Local Development (docker-compose.yml)
Services available:
- PostgreSQL buffer (port 5432)
- PostgreSQL source (port 5433)
- MinIO S3 storage (ports 9000, 9001)
- Lakekeeper catalog (port 8181)
- Prometheus (port 9090)
- Grafana (port 3000)
- HashiCorp Vault (port 8200)

#### Docker Images Needed
Currently no Dockerfiles exist for the Go applications. Need to create:
- `Dockerfile` for philotes-api
- `Dockerfile` for philotes-worker
- `Dockerfile` for philotes-cli

### 6. Helm Charts
- Located in `/charts/`
- philotes-api (version 0.1.0)
- philotes-worker (version 0.1.0)
- lakekeeper chart

### 7. Version Management
- Semantic versioning ready via `git describe --tags --always --dirty`
- Conventional commits specified in CONTRIBUTING.md

## Acceptance Criteria Mapping

| Criterion | Current State | Implementation Needed |
|-----------|--------------|----------------------|
| PR checks (lint, test, build) | Makefile targets exist | GitHub Actions workflow |
| Automated release with semantic versioning | Git tags ready | Release workflow + goreleaser |
| Docker image builds & push to GHCR | No Dockerfiles | Create Dockerfiles + build workflow |
| Integration tests with docker-compose | docker-compose exists | Integration test workflow with services |
| Code coverage reporting | Local coverage works | Upload to Codecov |
| Security scanning | Not implemented | GitHub CodeQL + Trivy for images |

## Recommended Approach

### Workflows to Create

1. **`.github/workflows/ci.yml`** - PR checks
   - Lint (golangci-lint)
   - Test (with race detector)
   - Build (all binaries)
   - Coverage upload to Codecov

2. **`.github/workflows/release.yml`** - Release automation
   - Triggered on version tags (v*)
   - Build binaries with goreleaser
   - Create GitHub release with changelog
   - Build and push Docker images to GHCR

3. **`.github/workflows/security.yml`** - Security scanning
   - CodeQL for Go code analysis
   - Trivy for container scanning
   - Dependency review on PRs

4. **`.github/workflows/integration.yml`** - Integration tests
   - Spin up docker-compose services
   - Run integration test suite
   - Cleanup

### Dockerfiles to Create

Multi-stage Dockerfiles for minimal image size:
- `deployments/docker/Dockerfile.api`
- `deployments/docker/Dockerfile.worker`
- `deployments/docker/Dockerfile.cli`

### Additional Files

- `.github/dependabot.yml` - Automated dependency updates
- `.goreleaser.yml` - Release automation configuration
- `codecov.yml` - Coverage configuration

## Key Dependencies

- Go 1.25.5+
- golangci-lint v1.64+
- goreleaser for releases
- Codecov for coverage
- Trivy for security scanning
