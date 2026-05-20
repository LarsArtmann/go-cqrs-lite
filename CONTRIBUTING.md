# Contributing to go-cqrs-lite

## Architecture

go-cqrs-lite is a **library/SDK**, not an application. Consumers import modules into their own projects. There is no main app.

### Module Structure

```
core/          — Zero-dep interfaces and types (event, command, query, aggregate, decider)
memory/        — In-memory implementations for testing
catalog/       — Auto-documentation: AsyncAPI, D2, EventCatalog generators
middleware/    — Cross-cutting: logging, retry, recovery, validation, metrics
testhelpers/   — Shared test utilities
projection/    — Projection runner with replay + live subscription
storage/       — SQL (Postgres/SQLite/Turso) and Pebble event stores
sync/          — CRDT-inspired sync primitives (LWW Register, Vector Clock)
integration/   — Cross-module integration tests
example/       — Usage demos (user, todo)
```

### Design Principles

1. **Library, not framework** — No opinionated transport, broker, or driver
2. **Interface-first** — All core types are interfaces
3. **Strong types** — Branded IDs, typed versions, sentinel errors
4. **Composition over inheritance**
5. **Context-aware** — All handlers accept `context.Context`
6. **No panics in production code** — Return errors, use `Must*` for test helpers

## Development

### Prerequisites

- Nix with flakes enabled (recommended), or Go 1.26+

### Commands

```bash
nix develop             # Enter dev shell
nix run .#build         # Build all modules
nix run .#test          # Run all tests
nix run .#test-race     # Race detector
nix run .#coverage      # Coverage report
nix run .#lint          # golangci-lint (8 modules)
nix fmt                 # Format all Go files
```

### Without Nix

```bash
go test ./core/... ./memory/... ./catalog/... ./middleware/... ./testhelpers/... ./integration/... ./projection/... ./storage/... -count=1
```

## Code Conventions

- **Max 250 lines per file**, max 30 lines per function
- **Sentinel errors** via `errors.New` in `errors.go`, wrapped with `fmt.Errorf("%w", ...)`
- **No `any` types** (use generics or concrete types)
- **No TODO/FIXME** in committed code
- **Table-driven tests** preferred, `t.Parallel()` for independent tests
- **Doc comments** on all exported types and functions

### Error Handling

```go
// Sentinel errors
var ErrNotFound = errors.New("not found")

// Contextual errors
return fmt.Errorf("load user %s: %w", id, ErrNotFound)

// Classified errors (for retry/circuit-breaker logic)
return event.NewRejection("user.create.empty_email", "email is required")
return event.NewConflict("user.create.duplicate", "user already exists")
```

### Adding New Features

1. Define interfaces in `core/`
2. Implement in the appropriate module (memory, storage, etc.)
3. Add tests with >80% coverage
4. Add doc comments to all exported symbols
5. Register error classifications via `init()` + `event.RegisterClassification()`
6. Update `FEATURES.md` and `AGENTS.md`

### Adding New Event Store Methods

All `event.Store` implementations must implement every method. When adding a method:

1. Add to `core/event/store.go`
2. Implement in `memory/store.go`, `storage/event_store.go`, `storage/pebble_event_store.go`, `testhelpers/fake_store.go`
3. Update any test mock stores that implement `event.Store`
