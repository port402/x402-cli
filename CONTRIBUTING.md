# Contributing to x402-cli

Thank you for your interest in contributing to x402-cli! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors of all backgrounds and experience levels.

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing [issues](https://github.com/port402/x402-cli/issues) to avoid duplicates.

When filing an issue, include:
- **x402 version**: Run `x402 version`
- **OS and architecture**: e.g., macOS arm64, Linux amd64
- **Go version**: Run `go version`
- **Steps to reproduce**: Minimal commands to trigger the issue
- **Expected vs actual behavior**: What you expected and what happened
- **Error output**: Full error messages or logs

### Suggesting Features

Open an issue with the `enhancement` label to discuss new features before implementing them. Include:
- **Use case**: Why is this feature needed?
- **Proposed solution**: How should it work?
- **Alternatives considered**: Other approaches you've thought about

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** with clear, focused commits
3. **Add tests** for new functionality
4. **Run all checks**: `make check`
5. **Open a pull request** with a clear description

#### Commit Messages

Use clear, descriptive commit messages:

```
Add --timeout flag to batch-health command

- Allows configuring per-request timeout
- Defaults to 30 seconds
- Fixes #123
```

- Start with a verb in imperative mood (Add, Fix, Update, Remove)
- Keep the first line under 72 characters
- Reference related issues when applicable

#### Branch Naming

Use descriptive branch names:
- `feature/add-timeout-flag`
- `fix/health-check-error`
- `docs/update-readme`

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

### Building

```bash
git clone https://github.com/port402/x402-cli.git
cd x402-cli
make build
```

### Available Make Targets

```bash
make build        # Build the binary
make install      # Install to $GOPATH/bin
make test         # Run all tests
make test-cover   # Run tests with coverage report
make lint         # Run linter
make fmt          # Format code
make check        # Run all checks (fmt, vet, lint, test)
make clean        # Remove build artifacts
make help         # Show all available targets
```

### Before Submitting a PR

Run all checks to ensure your code meets project standards:

```bash
make check
```

This runs formatting, vetting, linting, and tests in one command.

## Project Structure

```
x402-cli/
├── cmd/x402/           # CLI entry point
├── internal/
│   ├── client/         # HTTP client
│   ├── commands/       # Cobra commands
│   ├── output/         # Output formatting
│   ├── tokens/         # Token registry
│   ├── wallet/         # Key management, signing
│   └── x402/           # Protocol types, parsing
├── poc/                # Proof-of-concept code (reference only)
└── spec.md             # Protocol specification
```

## Coding Standards

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt -s` for formatting
- Write tests for new functionality
- Keep functions focused and reasonably sized
- Document exported functions and types
- Handle errors explicitly (no silent failures)

## Testing Against Real Endpoints

The CLI has been validated against public x402 APIs:

```bash
# Health check against Elsa (POST required)
./x402 health https://x402-api.heyelsa.ai/api/search_token -X POST

# Health check against x402 Index
./x402 health https://x402index.com/api/all
```

See `test-endpoints.json` for a list of known endpoints.

## Questions?

- **Bug or feature?** Open a [GitHub issue](https://github.com/port402/x402-cli/issues)
- **General questions?** Start a [GitHub discussion](https://github.com/port402/x402-cli/discussions)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
