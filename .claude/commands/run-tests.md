# Run Tests

Run the Philotes test suite.

## Instructions

Based on user request, run the appropriate tests:

- Unit tests: `make test`
- Integration tests: `make test-integration`
- All tests: `make test-all`
- Dashboard tests: `cd web && pnpm test`

## Running Go Tests

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Run all Go tests with coverage
make test

# Run tests for specific package
go test ./internal/cdc/... -v

# Run tests with race detection
go test ./... -race

# Run only unit tests (short)
go test ./... -short

# Run integration tests (requires Docker)
make test-integration
```

## Running Dashboard Tests

```bash
cd /Volumes/ExternalSSD/dev/philotes/web

# Run all tests
pnpm test

# Run in watch mode
pnpm test:watch

# Run with coverage
pnpm test:coverage
```

## Running E2E Tests

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Start test environment
docker compose -f deployments/docker/docker-compose.yml up -d

# Run E2E tests
make test-e2e

# Or with Playwright
cd web && pnpm exec playwright test
```

## Test Categories

### Go Tests
- `*_test.go` in package directories - Unit tests
- `*_integration_test.go` - Integration tests (require running services)
- `test/e2e/` - End-to-end tests

### Dashboard Tests
- `*.test.tsx` - Component tests
- `*.test.ts` - Utility tests
- `e2e/*.spec.ts` - Playwright E2E tests

## Test Configuration

### Go Test Tags

```go
// +build integration

package cdc_test

// This test only runs with: go test -tags=integration
```

### Vitest Configuration

```typescript
// web/vitest.config.ts
export default defineConfig({
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      reporter: ['text', 'html'],
      exclude: ['node_modules/', 'src/test/'],
    },
  },
})
```

## Coverage Report

After running tests with coverage:

```bash
# Go coverage
go tool cover -html=coverage.out

# Dashboard coverage
open web/coverage/index.html
```

## Common Test Commands

| Command                    | Description                          |
|---------------------------|--------------------------------------|
| `make test`               | Run Go unit tests                    |
| `make test-integration`   | Run Go integration tests             |
| `make test-all`           | Run all Go tests                     |
| `make lint`               | Run golangci-lint                    |
| `cd web && pnpm test`     | Run dashboard tests                  |
| `cd web && pnpm lint`     | Run ESLint on dashboard              |
