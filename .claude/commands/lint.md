# Lint and Format Code

Run code quality checks and formatting for Philotes.

## Go Code

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Run golangci-lint
make lint

# Or directly
golangci-lint run ./...

# Fix auto-fixable issues
golangci-lint run --fix ./...

# Format code
go fmt ./...

# Run go vet
go vet ./...
```

## Dashboard Code

```bash
cd /Volumes/ExternalSSD/dev/philotes/web

# Run ESLint
pnpm lint

# Fix auto-fixable issues
pnpm lint:fix

# Run Prettier
pnpm format

# Check formatting
pnpm format:check

# Type checking
pnpm typecheck
```

## All Checks

```bash
cd /Volumes/ExternalSSD/dev/philotes

# Run all linting (Go + Dashboard)
make lint-all

# Or manually:
golangci-lint run ./...
cd web && pnpm lint && pnpm typecheck
```

## Pre-commit Hook

The project uses pre-commit hooks for automatic linting:

```bash
# Install pre-commit hooks
make setup-hooks

# Run hooks manually
pre-commit run --all-files
```

## Configuration Files

### Go Linting (`.golangci.yml`)

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true
```

### ESLint (`web/.eslintrc.js`)

```js
module.exports = {
  extends: [
    'next/core-web-vitals',
    'plugin:@typescript-eslint/recommended',
  ],
  rules: {
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/explicit-function-return-type': 'off',
  },
}
```

### Prettier (`web/.prettierrc`)

```json
{
  "semi": false,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5"
}
```

## Common Issues

### Go Import Ordering

```bash
# Fix import ordering
goimports -w .
```

### TypeScript Errors

```bash
# Check types only
cd web && pnpm typecheck

# Generate types from OpenAPI
cd web && pnpm generate:api
```

### Prettier Conflicts

```bash
# Format everything
cd web && pnpm format

# Then lint
cd web && pnpm lint
```
