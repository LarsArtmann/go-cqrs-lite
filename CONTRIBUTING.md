# Contributing to go-cqrs-lite

> **Thank you for contributing!** This guide covers everything you need to know.

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/LarsArtmann/go-cqrs-lite.git
cd go-cqrs-lite

# 2. Enter the Nix dev shell
nix develop

# 3. Verify everything works
nix run .#test
nix run .#lint
```

## Development Setup

### Prerequisites

| Tool | Version | Purpose           |
| ---- | ------- | ----------------- |
| Go   | 1.26+   | Language runtime  |
| Nix  | latest  | Build environment |

### Using Nix (Recommended)

```bash
# Enter dev shell with all tools
nix develop

# Build all modules
nix run .#build

# Run all tests
nix run .#test

# Run linter
nix run .#lint

# Run all benchmarks
nix run .#bench

# Format code
nix fmt
```

### Without Nix

```bash
# Run tests (per module)
cd event && go test ./... -count=1

# Run linter
golangci-lint run

# Format code
gofumpt -w .
```

## Project Structure

Multi-module Go workspace with 29 modules:

```
event/         # Event system (Event, EventSink, EventSource, Bus)
command/       # Command dispatcher
query/         # Query dispatcher
decider/       # Pure-function aggregate
id/            # Branded IDs
dispatcher/    # Generic dispatcher
schema/        # Schema evolution (upcasters)
snapshot/      # Snapshot support
memory/        # In-memory implementations (testing)
catalog/       # Schema registry + exporters
middleware/    # Middleware suite (logging, tracing, metrics)
signing/       # Event signing (HMAC, Ed25519)
projection/    # Projection runner (replay + live)
storage/       # SQL event store (PostgreSQL, SQLite, Turso)
otel/          # OpenTelemetry helpers
listing/       # Aggregate listing
watermill/     # Watermill protocol adapter
pebble/        # PebbleDB event store
codec/         # Payload encoding (JSON, Raw)
turso/         # Turso database connector
cmd/cqrs-gen/  # Code generator
cmd/api-stability/  # API surface checker
```

## Testing

### Running Tests

```bash
# All modules
nix run .#test

# Single module
cd event && go test ./... -count=1

# With race detector
go test -race ./... -count=1

# With coverage
go test ./... -count=1 -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Coverage Requirements

| Module Type  | Minimum |
| ------------ | ------- |
| Core (event) | 85%     |
| Other        | 80%     |

## Code Standards

### Key Principles

1. **Library, not framework** — Consumers import what they need
2. **Interface-first** — All core types are interfaces
3. **Strong types** — No `any` except for dialect interop
4. **Composition over inheritance** — No deep hierarchies
5. **Errors as values** — No panics, explicit error returns
6. **Context-aware** — All handlers accept `context.Context`

### Style

- Functional programming: immutability, pure functions
- Early returns over nested conditionals
- Max 250 lines/file, 30 lines/function
- Descriptive names over comments

## Commit Messages

Format: `type(scope): description`

Examples:

- `perf(event): remove Payload() clone`
- `test(memory): add concurrent stress benchmarks`
- `docs: update benchmark results`

## Pull Request Process

1. Run `nix run .#test` and `nix run .#lint` locally
2. Ensure tests pass with `-race`
3. Run `nix run .#check-layers` to verify module dependency rules
4. Update docs if behavior changes
5. Request review from maintainers

## Security & Architecture Checks

### Module Layer Validation

The project enforces a layered module dependency graph via `scripts/check-module-layers.sh`:

```bash
nix run .#check-layers
```

This validates that modules only depend on their allowed layer. The dependency graph is:

```
Layer 0: id/, dispatcher/, codec/         (leaf modules, no internal deps)
Layer 1: event/, command/, query/          (depend on Layer 0)
Layer 2: schema/, snapshot/                (depend on Layer 1)
Layer 3: decider/                          (depends on event, snapshot)
Layer 4: memory/, signing/, otel/          (leaf or Layer 1 deps)
Layer 5: middleware/, storage/, projection/ (depend on Layers 0-4)
Layer 6: integration/, catalog/, examples/ (depend on everything)
```

### Dependency Budgets

Each module has a maximum direct dependency limit enforced by `check-module-layers.sh`. Adding a new dependency to a module may require updating the budget in the script.

### Security Scanning

[gosec](https://github.com/securego/gosec) scans run in CI:

```bash
# Run locally via nix
nix develop -c gosec ./event/... ./command/... ./decider/...
```

CI uploads SARIF results to GitHub Security tab. Fix any findings before merging.

## Golden File Tests

Several modules use golden file tests to detect output format regressions:

```bash
# Run golden tests (verify output matches golden files)
go test ./catalog/asyncapi/... -count=1
go test ./signing/... -count=1

# Update golden files after intentional format changes
go test ./catalog/asyncapi/... -update -count=1
go test ./signing/... -update -count=1
```

## Architecture Decisions

See `docs/adr/` for recorded decisions.
