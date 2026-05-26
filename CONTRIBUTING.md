# Contributing to go-cqrs-lite

> **Thank you for contributing!** This guide covers everything you need to know to contribute effectively.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Quick Start](#quick-start)
- [Development Setup](#development-setup)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Architecture](#architecture)
- [Linting & Quality](#linting--quality)
- [Security](#security)
- [Pull Request Process](#pull-request-process)
- [Commit Messages](#commit-messages)

---

## Code of Conduct

We are committed to providing a welcoming and respectful environment. All contributors are expected to:

- **Be respectful** — Treat others with kindness and professionalism
- **Be inclusive** — Welcome diverse perspectives and experiences
- **Be constructive** — Provide feedback that helps improve the project
- **Be collaborative** — Work together to achieve the best outcomes

**Unacceptable behavior** will not be tolerated.

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/LarsArtmann/go-cqrs-lite.git
cd go-cqrs-lite

# 2. Run the setup script
./CONTRIBUTING-setup.sh

# 3. Install development dependencies
just install

# 4. Verify everything works
just lint
just test
```

---

## Development Setup

### Prerequisites

| Tool           | Version | Purpose              |
| -------------- | ------- | -------------------- |
| Go             | 1.21+   | Language runtime     |
| Just           | latest  | Task runner          |
| golangci-lint  | v2.6.0  | Code linting         |
| go-arch-lint   | v1.14.0 | Architecture linting |
| branching-flow | latest  | Semantic analysis    |
| gofumpt        | latest  | Code formatting      |
| goimports      | latest  | Import management    |

### Installation

```bash
# Install all tools via just
just install

# Or install individually
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0
go get -tool github.com/fe3dback/go-arch-lint@v1.14.0
```

### Pre-commit Hooks

```bash
# Install basic hooks (formatting only - fast)
just install-hooks

# Install comprehensive hooks (includes architecture validation)
just install-hooks-full
```

---

## Code Standards

### Mandatory Rules

1. **Centralized Error Management**
   - All errors MUST be defined in `pkg/errors/`
   - No direct `errors.New()` or `fmt.Errorf()` outside `pkg/errors`
   - Use `pkg/errors.Wrap()`, `pkg/errors.New()` for error creation
   - Use `errors.Is()`, `errors.As()` for error checking

2. **Strong ID Types**
   - No raw string/numeric IDs — use branded types
   - Example: `type UserID = brsdt.Type[string, "user_id"]`

3. **Composition Over Inheritance**
   - Prefer interfaces and struct embedding over class hierarchies
   - Use dependency injection for testability

4. **Early Returns**
   - Use guard clauses to reduce nesting
   - Keep functions small and focused

### File Organization

```
├── cmd/                    # Application entry points
│   └── go-cqrs-lite/main.go
├── internal/               # Private application code
│   ├── domain/             # Domain layer (business logic)
│   │   ├── entities/       # Business entities
│   │   ├── values/         # Value objects
│   │   ├── repositories/   # Repository interfaces
│   │   └── services/      # Domain services
│   ├── application/        # Application layer
│   │   └── handlers/      # HTTP handlers
│   ├── infrastructure/     # Infrastructure layer
│   │   └── db/            # SQLC generated code
│   └── config/            # Configuration
├── pkg/                    # Public packages
│   └── errors/            # Centralized error definitions
└── go-cqrs-lite.go     # Module definition
```

### Naming Conventions

| Type       | Convention                  | Example                        |
| ---------- | --------------------------- | ------------------------------ |
| Packages   | lowercase, single word      | `domain`, `handlers`           |
| Interfaces | PascalCase with "er" suffix | `Repository`, `Service`        |
| Functions  | PascalCase                  | `CreateUser`, `GetByID`        |
| Variables  | camelCase                   | `userID`, `isActive`           |
| Constants  | PascalCase                  | `MaxRetries`, `DefaultTimeout` |
| Files      | lowercase, descriptive      | `user_repository.go`           |

---

## Testing

### Test Structure

```go
// Package naming: same package or package_test for black-box
package domain_test  // Preferred for integration

// or

package domain  // For white-box testing
```

### Test Organization

```go
// Use descriptive names with Given-When-Then pattern
func TestCreateUser_GivenValidInput_WhenUserDoesNotExist_ShouldCreateUser(t *testing.T)
```

### Coverage Requirements

| Metric          | Minimum | Target |
| --------------- | ------- | ------ |
| Line Coverage   | 70%     | 85%    |
| Branch Coverage | 60%     | 75%    |
| Critical Paths  | 100%    | 100%   |

```bash
# Run tests with coverage
just test

# Check coverage threshold
just coverage 80

# Detailed coverage analysis
just coverage-detailed
```

---

## Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    INFRASTRUCTURE                           │
│   (External systems: DB, HTTP clients, file system)          │
└─────────────────────────┬───────────────────────────────────┘
                          │ implements
┌─────────────────────────▼───────────────────────────────────┐
│                    APPLICATION                              │
│   (Use cases, handlers, orchestration)                      │
└─────────────────────────┬───────────────────────────────────┘
                          │ uses
┌─────────────────────────▼───────────────────────────────────┐
│                       DOMAIN                                 │
│   (Entities, value objects, domain services)                 │
└─────────────────────────────────────────────────────────────┘
```

### Dependency Rules

- **Domain** → Domain only (entities, values, repositories interfaces)
- **Application** → Domain + Infrastructure interfaces
- **Infrastructure** → Domain interfaces (implements them)
- **pkg/errors** → Available everywhere (mandatory)

### Architecture Validation

```bash
# Check architecture compliance
just lint-arch

# Generate architecture graph
just graph

# Verbose architecture checking
just verbose
```

---

## Linting & Quality

### Quick Commands

```bash
# Run all linters
just lint

# Fix issues automatically
just fix

# Format code
just format

# Run pre-commit checks
just check-pre-commit
just check-pre-commit-fast  # Fast version for hooks
```

### Detailed Commands

```bash
# Code quality only
just lint-code

# Architecture only
just lint-arch

# Security only
just lint-security

# Vulnerability scanning
just lint-vulns

# Nil panic detection
just lint-nilaway

# Capability analysis
just lint-capslock
```

### Branching-Flow Analysis

```bash
# Semantic context analysis (error handling)
branching-flow context .

# Find duplicate types
branching-flow dupe .

# Check for phantom types
branching-flow phantom .

# Panic condition analysis
branching-flow panic .

# Strong ID type analysis
branching-flow strong-id .

# Boolean blindness analysis
branching-flow boolblind .

# Anti-pattern detection
branching-flow anti-patterns .

# Run all analyzers
branching-flow all .
```

### Quality Gates

Before merging, all checks must pass:

- [ ] `golangci-lint` passes
- [ ] `go-arch-lint` passes
- [ ] `branching-flow all .` passes
- [ ] Tests pass with 80%+ coverage
- [ ] Code is formatted (`gofumpt`)
- [ ] Imports are organized (`goimports`)
- [ ] No security vulnerabilities (`govulncheck`)

---

## Security

### Security Scanning

```bash
# Full security audit
just security-audit

# Quick security check
just capslock-quick

# Vulnerability scanning
just lint-vulns

# Docker security scan
just docker-security
```

### Security Best Practices

1. **Input Validation** — Validate all inputs at boundaries
2. **Parameterized Queries** — Use SQLC for type-safe SQL
3. **No Secrets in Code** — Use environment variables
4. **Principle of Least Privilege** — Request only required capabilities
5. **Error Messages** — Don't leak sensitive information

---

## Pull Request Process

### PR Requirements

1. **Branch Naming**

   ```
   feat/description
   fix/description
   docs/description
   refactor/description
   test/description
   ```

2. **PR Description Template**

   ```markdown
   ## Summary

   Brief description of changes

   ## Type

   - [ ] Feature
   - [ ] Bug fix
   - [ ] Refactoring
   - [ ] Documentation

   ## Test Plan

   - [ ] Unit tests added/updated
   - [ ] Integration tests added/updated
   - [ ] Manual testing performed

   ## Checklist

   - [ ] Code follows style guidelines
   - [ ] Architecture rules pass
   - [ ] Security scan clean
   - [ ] Documentation updated
   ```

### Review Process

1. **Self-review first** — Run `just lint` and `just test` locally
2. **Small PRs** — Keep changes focused and digestible
3. **Explain "why"** — Not just "what", but rationale
4. **Be responsive** — Address feedback promptly

---

## Commit Messages

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type     | Description              |
| -------- | ------------------------ |
| feat     | New feature              |
| fix      | Bug fix                  |
| docs     | Documentation changes    |
| style    | Formatting, whitespace   |
| refactor | Code restructuring       |
| test     | Adding/updating tests    |
| chore    | Build, tooling, CI       |
| perf     | Performance improvements |
| ci       | CI/CD changes            |
| revert   | Reverting changes        |

### Examples

```bash
# Good
feat(auth): add JWT token refresh mechanism

Implements automatic token refresh before expiration
to improve user experience and reduce authentication
failures.

Closes #123

# Bad
fix stuff

# Good
docs(readme): update installation instructions

Added Go 1.21+ requirement and just installation
instructions for macOS users.

# Bad
updated README
```

---

## Getting Help

### Resources

- [Go Documentation](https://go.dev/doc/)
- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Effective Go](https://go.dev/doc/effective_go)

### Getting Unblocked

1. **Read the docs** — Check `docs/` folder first
2. **Check existing issues** — Someone may have solved it
3. **Ask questions** — Open a discussion, don't struggle alone

---

_Thank you for contributing to go-cqrs-lite!_
