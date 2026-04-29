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
| `testhelpers/` | `github.com/larsartmann/go-cqrs-lite/testhelpers` | Shared test utilities |

## Prerequisites

- [Nix](https://nixos.org/download.html) with flakes enabled
- Go 1.26+ (provided by the Nix dev shell)

## Development Workflow

### Nix Dev Shell (Recommended)

Enter the development environment with all tools pinned:

```bash
nix develop
```

Or with [direnv](https://direnv.net/):

```bash
echo "use flake" > .envrc
direnv allow
```

### Workspace Mode

From the repo root, the `go.work` file ties all modules together:

```bash
# Build all modules
nix run .#build

# Test all modules
nix run .#test

# Test with race detection
nix run .#test-race

# Lint all modules
nix run .#lint

# Format all Go files
nix fmt

# Coverage report
nix run .#coverage
```

### Per-Module Isolation

When working on a single module, use `GOWORK=off` to isolate it:

```bash
cd core && GOWORK=off go test ./... -count=1
cd memory && GOWORK=off go test ./... -count=1
cd catalog && GOWORK=off go test ./... -count=1
cd middleware && GOWORK=off go test ./... -count=1
cd testhelpers && GOWORK=off go test ./... -count=1
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
4. Add the module to the `testModules` list in `flake.nix`
5. Update `AGENTS.md` with the new module's info

## Replace Directives

Each module that depends on another uses `replace` directives in `go.mod` for local development. These point to sibling directories:

```
replace github.com/larsartmann/go-cqrs-lite/core => ../core
```

The `memory` replace directive is needed even if you don't directly import it, because `core`'s tests use `memory` as a transitive test dependency. Running `GOWORK=off go mod tidy` requires it.

## Examples

Examples live in `example/` and use `GOWORK=off` with their own `go.mod` files. They are not part of `go.work` to avoid cluttering the main workspace.

## Pull Request Checklist

- [ ] `nix run .#build && nix run .#test && nix run .#lint` passes
- [ ] New code has tests
- [ ] No `any` types introduced
- [ ] Error messages include context
- [ ] Public APIs have Go doc comments
