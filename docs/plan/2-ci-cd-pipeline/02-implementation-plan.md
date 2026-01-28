# Implementation Plan: FOUND-002 - CI/CD Pipeline Setup

## Overview

Implement a comprehensive CI/CD pipeline using GitHub Actions for the Philotes project. This includes PR checks, release automation, Docker image builds, and security scanning.

## Files to Create

| File | Purpose |
|------|---------|
| `.github/workflows/ci.yml` | PR checks (lint, test, build) |
| `.github/workflows/release.yml` | Automated releases with Docker images |
| `.github/workflows/security.yml` | CodeQL and container scanning |
| `.github/dependabot.yml` | Automated dependency updates |
| `.goreleaser.yml` | Release automation configuration |
| `deployments/docker/Dockerfile` | Multi-stage Dockerfile for all binaries |
| `codecov.yml` | Coverage configuration |

## Task Breakdown

### Task 1: Create CI Workflow (`.github/workflows/ci.yml`)

PR checks workflow that runs on every pull request:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    - golangci-lint with .golangci.yml config

  test:
    - Run tests with race detector
    - Generate coverage report
    - Upload to Codecov

  build:
    - Build all 3 binaries
    - Verify successful compilation
```

**Key decisions:**
- Use `golangci/golangci-lint-action@v6` for linting
- Go version: 1.25.x (from go.mod)
- Upload coverage to Codecov (free for open source)
- Cache Go modules for faster builds

### Task 2: Create Release Workflow (`.github/workflows/release.yml`)

Triggered on version tags (v*):

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    - Create GitHub release with goreleaser
    - Build binaries for linux/darwin, amd64/arm64
    - Generate changelog from commits

  docker:
    - Build multi-platform Docker images
    - Push to ghcr.io/janovincze/philotes-api
    - Push to ghcr.io/janovincze/philotes-worker
    - Push to ghcr.io/janovincze/philotes-cli
```

**Key decisions:**
- Use goreleaser for binary releases
- Multi-platform images: linux/amd64, linux/arm64
- Tag images with version and `latest`
- Use GitHub Container Registry (ghcr.io)

### Task 3: Create Security Workflow (`.github/workflows/security.yml`)

Security scanning on PR and schedule:

```yaml
name: Security
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday

jobs:
  codeql:
    - GitHub CodeQL analysis for Go

  trivy:
    - Scan dependencies for vulnerabilities

  dependency-review:
    - Review dependency changes in PRs
```

**Key decisions:**
- CodeQL for static analysis (free for open source)
- Trivy for dependency/container scanning (free, fast)
- Weekly scheduled scans for ongoing security
- Dependency review on PRs only

### Task 4: Create Dockerfile (`deployments/docker/Dockerfile`)

Multi-stage Dockerfile for minimal images:

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder
# Build all binaries with CGO disabled

# API image
FROM gcr.io/distroless/static-debian12 AS api
COPY --from=builder /app/bin/philotes-api /philotes-api
ENTRYPOINT ["/philotes-api"]

# Worker image
FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=builder /app/bin/philotes-worker /philotes-worker
ENTRYPOINT ["/philotes-worker"]

# CLI image
FROM gcr.io/distroless/static-debian12 AS cli
COPY --from=builder /app/bin/philotes /philotes
ENTRYPOINT ["/philotes"]
```

**Key decisions:**
- Multi-stage build for minimal image size
- Distroless base for security (no shell, minimal attack surface)
- CGO_ENABLED=0 for static binaries
- Build targets for each binary in single Dockerfile

### Task 5: Create GoReleaser Config (`.goreleaser.yml`)

Release automation:

```yaml
builds:
  - id: philotes-api
    main: ./cmd/philotes-api
    binary: philotes-api
    goos: [linux, darwin]
    goarch: [amd64, arm64]

  - id: philotes-worker
    main: ./cmd/philotes-worker
    binary: philotes-worker
    goos: [linux, darwin]
    goarch: [amd64, arm64]

  - id: philotes-cli
    main: ./cmd/philotes-cli
    binary: philotes
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

changelog:
  use: github
  groups:
    - title: Features
      regexp: '^feat.*'
    - title: Bug Fixes
      regexp: '^fix.*'
```

### Task 6: Create Dependabot Config (`.github/dependabot.yml`)

Automated dependency updates:

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule:
      interval: weekly
    groups:
      go-dependencies:
        patterns: ["*"]

  - package-ecosystem: github-actions
    directory: "/"
    schedule:
      interval: weekly
```

### Task 7: Create Codecov Config (`codecov.yml`)

Coverage settings:

```yaml
coverage:
  status:
    project:
      default:
        target: auto
        threshold: 1%
    patch:
      default:
        target: 80%

comment:
  layout: "diff, flags, files"
  behavior: default
```

## Implementation Order

1. **Dockerfile** - Required for release workflow
2. **CI Workflow** - Core functionality, immediate value
3. **Codecov Config** - Configure coverage thresholds
4. **GoReleaser Config** - Required for release workflow
5. **Release Workflow** - Depends on Dockerfile and GoReleaser
6. **Security Workflow** - Independent, can be added after core CI
7. **Dependabot Config** - Independent, low-priority

## Test Strategy

1. **CI Workflow:** Create a test PR to verify lint/test/build jobs
2. **Release Workflow:** Create a test tag (v0.0.1-test) to verify
3. **Security Workflow:** Verify CodeQL and Trivy run successfully
4. **Docker Images:** Build locally and verify they run

## Verification Commands

```bash
# Test Docker build locally
docker build --target api -t philotes-api:test -f deployments/docker/Dockerfile .
docker build --target worker -t philotes-worker:test -f deployments/docker/Dockerfile .

# Test goreleaser locally (dry run)
goreleaser release --snapshot --clean

# Verify CI commands work
make lint
make test
make build
```

## Notes

- Go 1.25.5 is used (from go.mod) - will use 1.25.x in workflows
- No integration tests exist yet - CI workflow will only run unit tests
- Codecov token may need to be added as repository secret
- GITHUB_TOKEN is automatically available for GHCR pushes
