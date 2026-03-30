# Contributing to go-cqrs-lite

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/go-cqrs-lite.git`
3. Install dependencies: `go mod download`
4. Run tests: `go test ./...`

## Development Workflow

### Branch Naming

- `feature/description` - New features
- `bugfix/description` - Bug fixes
- `refactor/description` - Code refactoring
- `docs/description` - Documentation updates

### Before Submitting

1. **Run tests**: `make test`
2. **Run linter**: `make lint`
3. **Format code**: `make fmt`
4. **Check coverage**: `make coverage`

### Code Standards

- Follow Go best practices
- Add tests for new functionality
- Keep functions under 30 lines
- Keep files under 250 lines
- Use context as first parameter in public methods
- Wrap errors with context using `errors.Wrapf`

## Pull Request Process

1. Update documentation if needed
2. Add tests for new features
3. Ensure CI passes
4. Request review from maintainers

## Code of Conduct

Be respectful and constructive in all interactions.

## Questions?

Open an issue for discussion before major changes.
