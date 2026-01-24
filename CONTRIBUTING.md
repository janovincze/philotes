# Contributing to Philotes

Thank you for your interest in contributing to Philotes! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## How to Contribute

### Reporting Issues

- Check if the issue already exists in [GitHub Issues](https://github.com/janovincze/philotes/issues)
- Use the issue templates when available
- Provide as much detail as possible:
  - Steps to reproduce
  - Expected vs actual behavior
  - Environment details (OS, Go version, etc.)
  - Relevant logs or error messages

### Suggesting Features

- Open an issue with the `type:feature` label
- Describe the use case and expected behavior
- Explain why this feature would be valuable

### Pull Requests

1. **Fork the repository** and create your branch from `main`

2. **Set up your development environment**
   ```bash
   git clone https://github.com/YOUR_USERNAME/philotes.git
   cd philotes
   make docker-up
   make install-tools
   ```

3. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make your changes**
   - Follow the coding guidelines below
   - Add tests for new functionality
   - Update documentation as needed

5. **Run checks locally**
   ```bash
   make check  # Runs lint, vet, and tests
   ```

6. **Commit your changes**
   - Use meaningful commit messages
   - Follow [Conventional Commits](https://www.conventionalcommits.org/):
     - `feat:` new feature
     - `fix:` bug fix
     - `docs:` documentation changes
     - `refactor:` code refactoring
     - `test:` test changes
     - `chore:` maintenance tasks

7. **Push and create a Pull Request**
   - Link any related issues
   - Fill out the PR template
   - Wait for CI checks to pass

## Development Guidelines

### Code Style

#### Go Code

- Follow standard Go conventions and [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint` before committing
- Write meaningful variable and function names
- Add comments for exported functions and types
- Group imports: standard library, external packages, internal packages

```go
import (
    "context"
    "fmt"

    "github.com/gin-gonic/gin"

    "github.com/janovincze/philotes/internal/config"
)
```

#### Error Handling

- Always handle errors explicitly
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use custom error types for domain errors
- Log errors at the appropriate level

```go
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}
```

### Testing

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Aim for meaningful test coverage, not just high percentages
- Use `testify/assert` for assertions where helpful
- Mock external dependencies

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "foo", "bar", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Something(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported types and functions
- Update OpenAPI specs for API changes
- Include examples where helpful

### Commit Messages

Follow the Conventional Commits specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Examples:
- `feat(cdc): add support for PostgreSQL 16`
- `fix(api): handle connection timeout gracefully`
- `docs: update quickstart guide`
- `refactor(iceberg): simplify schema evolution logic`

## Project Structure

```
philotes/
├── cmd/                    # Entry points (minimal code)
├── internal/               # Private packages
│   ├── api/                # HTTP handlers
│   ├── cdc/                # CDC pipeline
│   ├── config/             # Configuration
│   ├── iceberg/            # Iceberg integration
│   └── storage/            # Storage layer
├── pkg/                    # Public packages (client SDK)
├── api/openapi/            # API specifications
├── deployments/            # Deployment configs
├── docs/                   # Additional documentation
└── web/                    # Dashboard frontend
```

## Getting Help

- Read the [documentation](docs/)
- Ask questions in [GitHub Discussions](https://github.com/janovincze/philotes/discussions)
- Check existing [issues](https://github.com/janovincze/philotes/issues)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
