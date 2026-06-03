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

| Tool    | Version | Purpose           |
| ------- | ------- | ----------------- |
| Go      | 1.26+   | Language runtime  |
| Nix     | latest  | Build environment |

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

Multi-module Go workspace with 30 modules:

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
3. Update docs if behavior changes
4. Request review from maintainers

## Architecture Decisions

See `docs/adr/` for recorded decisions.
