# Contributing to go-cqrs-lite

Thank you for your interest in contributing! This guide covers the multi-module workflow.

## Module Structure

This is a multi-module Go workspace with 5 independent modules:

| Module | Import Path | Purpose |
|--------|------------|---------|
| `core/` | `github.com/larsartmann/go-cqrs-lite/core` | Command, query, event, aggregate, IDs |
| `memory/` | `github.com/larsartmann/go-cqrs-lite/memory` | In-memory test implementations |
| `catalog/` | `github.com/larsartmann/go-cqrs-lite/catalog` | AsyncAPI + EventCatalog generators |
| `middleware/` | `github.com/larsartmann/go-cqrs-lite/middleware` | Cross-cutting middleware (logging, retry, recovery) |
| `xtypes/` | `github.com/larsartmann/go-cqrs-lite/xtypes` | Typed wrappers with branded IDs |

## Prerequisites

- Go 1.26+
- `golangci-lint` v2+ (for linting)
- `gofumpt` (for formatting)

## Development Workflow

### Workspace Mode (Recommended)

From the repo root, the `go.work` file ties all modules together:

```bash
# Build all modules
go build ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...

# Test all modules
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/... -count=1

# Test with race detection
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/... -race -count=1
```

### Per-Module Isolation

When working on a single module, use `GOWORK=off` to isolate it:

```bash
cd core && GOWORK=off go test ./... -count=1
cd memory && GOWORK=off go test ./... -count=1
cd catalog && GOWORK=off go test ./... -count=1
cd middleware && GOWORK=off go test ./... -count=1
cd xtypes && GOWORK=off go test ./... -count=1
```

### Makefile Targets

```bash
make build       # Build all modules
make test        # Test all modules (verbose)
make test-race   # Test with race detection
make test-cover  # Coverage report
make lint        # Lint all modules
make fmt         # Format with gofumpt
make imports     # Sort imports with goimports
make check       # fmt + imports + lint + build + test
```

## Code Style

- Max 250 lines per file, max 30 lines per function
- Use `fmt.Errorf` for contextual errors, `errors.New` for sentinel errors
- Wrap errors with `fmt.Errorf("...: %w", err)` or `errors.Wrap`
- Context as first parameter in all public functions
- No `any` types
- Table-driven tests preferred
- Use `t.Parallel()` for independent tests

## Adding a New Module

1. Create the directory with its own `go.mod`
2. Add `replace` directives for local development:

```go
replace (
    github.com/larsartmann/go-cqrs-lite/core => ../core
    github.com/larsartmann/go-cqrs-lite/memory => ../memory
)
```

3. Add the module to `go.work`
4. Add the module path to the `MODULES` variable in the `Makefile`
5. Add the module to CI matrix in `.github/workflows/test.yml` and `lint.yml`
6. Update `AGENTS.md` with the new module's info

## Replace Directives

Each module that depends on another uses `replace` directives in `go.mod` for local development. These point to sibling directories:

```
replace github.com/larsartmann/go-cqrs-lite/core => ../core
```

The `memory` replace directive is needed even if you don't directly import it, because `core`'s tests use `memory` as a transitive test dependency. Running `GOWORK=off go mod tidy` requires it.

## Examples

Examples live in `example/` and use `GOWORK=off` with their own `go.mod` files. They are not part of `go.work` to avoid cluttering the main workspace.

## Pull Request Checklist

- [ ] `make check` passes
- [ ] New code has tests
- [ ] No `any` types introduced
- [ ] Error messages include context
- [ ] Public APIs have Go doc comments
