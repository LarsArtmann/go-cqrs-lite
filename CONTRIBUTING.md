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

# 4. Install the pre-commit hook (scope detection: skips lint for doc-only commits)
./scripts/install-hooks.sh
```

## Development Setup

### Prerequisites

| Tool | Version | Purpose           |
| ---- | ------- | ----------------- |
| Go   | 1.26+   | Language runtime  |
| Nix  | latest  | Build environment |

> **Build tag:** All Go commands require `-tags "goexperiment.jsonv2"` to enable
> JSON v2 encoding (`encoding/json/v2`). This is a Go experiment flag, NOT a
> standard build tag — it requires `GOEXPERIMENT` support in the toolchain.
> `nix run .#build`, `nix run .#test`, and CI apply it automatically. If running
> `go` commands directly, always pass `-tags "goexperiment.jsonv2"`.

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

### CGo / DuckDB

The project is pure-Go by default (`CGO_ENABLED=0` works for everything except
the DuckDB modules). The only module that **requires** CGo is `stack/duckdb`
(and its CGo-gated wiring in `stack/bench` and `cmd/cqrs-bench`). DuckDB
statically links a C++ engine (~30-50MB binary), so it is isolated in its own
module — consumers who never import it never need CGo (ADR-0071).

To build/test DuckDB locally you need a C compiler (`gcc`, included in the Nix
devShell via `pkgs.gcc`):

```bash
# Inside nix develop (gcc is already on PATH)
CGO_ENABLED=1 go build -tags "goexperiment.jsonv2" ./stack/duckdb/...
cd stack/duckdb && GOWORK=off CGO_ENABLED=1 go test -tags "goexperiment.jsonv2" ./...

# The cqrs-bench DuckDB factory is CGo-gated; without CGo a stub is compiled:
CGO_ENABLED=0 go build ./cmd/cqrs-bench/...   # uses the no-cgo stub (duckdb backend errors at runtime)
CGO_ENABLED=1 go build ./cmd/cqrs-bench/...   # full DuckDB backend available
```

CI runs a dedicated `cgo` job (`CGO_ENABLED=1`) so DuckDB regressions are caught
even though the default build is pure-Go.

## Project Structure

Multi-module Go workspace with 68 modules (verify: `find . -name go.mod -not -path './vendor/*' | wc -l`):

```
event/         # Event system (Event, EventSink, EventSource, Bus)
command/       # Command dispatcher
query/         # Query dispatcher
decider/       # Pure-function event sourcing
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
prometheus/    # OTel→Prometheus metrics bridge
listing/       # Stream listing
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

### Integration Tests (Database)

Three approaches, all Docker-free, powered by Nix:

```bash
# 1. Ephemeral PostgreSQL — fastest (no VM, no Docker)
nix run .#integration-pg                          # all PG integration tests
nix run .#integration-pg -- -run TestPostgresEventStore_CRUD

# 2. NixOS VM tests — hermetic CI verification
nix build .#checks.x86_64-linux.postgres-vm -L   # PG service health
nix build .#checks.x86_64-linux.mysql-vm -L      # MySQL service health

# 3. VM launcher scripts — interactive developer use
nix run .#integration-pg-vm                       # PG VM + Go tests on host
nix run .#integration-mysql-vm                    # MySQL VM + Go tests on host
```

See [ADR-0095](docs/adr/0095-nix-based-integration-testing.md) for the rationale.

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
- Max 350 lines/file (production), 30 lines/function
- Descriptive names over comments

### Two-Layer Pattern (Primitive + Adapter)

When a feature spans a Tier-0 primitive and a higher-tier adapter, put the
generic, schema-free logic in the primitive and the integration glue in the
adapter. Example: `codec.TranscodeToJSON` (Tier-0, no event dependency) +
`transport/http.CBORToJSONTransform` (Tier-4, composes event + codec). This
keeps the primitive reusable and the adapter thin. See ADR-0052.

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
4. Run `nix run .#check-arch` to verify architecture constraints
5. Update docs if behavior changes
6. Request review from maintainers

## Session & Branch Discipline

**Multiple concurrent sessions committing to `master` has caused issues**
(file corruption, unexpected diffs). Recommended practices:

1. Always `git status` before starting work
2. If you see changes you didn't author, investigate before touching them
3. Commit early and often to minimize conflicts
4. Use feature branches for long-running work: `git switch -c my-feature`
5. The BuildFlow pre-commit hook runs golangci-lint + gitleaks + gofumpt.
   Install it via `./scripts/install-hooks.sh` after cloning.
   It skips lint for doc-only commits (`.md`, `.html`, `.d2`, `.svg`, `.txt`,
   `.yaml`) and runs BuildFlow in `--staged-only` mode otherwise.

## Security & Architecture Checks

### Module Layer Validation

The project enforces a layered module dependency graph via `scripts/check-module-layers.sh`:

```bash
nix run .#check-layers
```

This validates that modules only depend on their allowed tier. The dependency model
is the **Seven-Tier Model** ([ADR-0046](docs/adr/0046-seven-tier-model.md),
[full reference](docs/architecture-understanding/FOUR-TIER-MODEL.md)):

```
Tier 0 — Primitives:    id/, dispatcher/, codec/, kv/, dedup/
Tier 1 — Core Domain:   event/, command/, query/, scheduling/, metadata/
Tier 2 — Utilities:     schema/, snapshot/, projection/, idempotency/, deriver/
Tier 3 — Aggregation:   decider/, graph/, scenario/, projectionhost/, listing/
Tier 4 — Infrastructure: storage/memory/, storage/, middleware/, signing/, encryption/,
                          otel/, watermill/, transport/http/, transport/grpc/,
                          storage/pebble/, storage/turso/, prometheus/
Tier 5 — Composition:   stack/, stack/memory/, stack/sqlite/, stack/pebble/,
                          stack/postgres/, stack/turso/
Tier 6 — Tooling:       catalog/, integration/, stack/bench/, examples/, cmd/*
```

Each tier may only import from its own tier or lower.

### Dependency Budgets

Each module has a maximum direct dependency limit enforced by `check-module-layers.sh`. Adding a new dependency to a module may require updating the budget in the script.

### Security Scanning

[gosec](https://github.com/securego/gosec) scans run in CI:

```bash
# Run locally via nix
nix develop -c gosec ./event/... ./command/... ./decider/...
```

CI uploads SARIF results to GitHub Security tab. Fix any findings before merging.

## Quality Gates & Nix Apps

The CI pipeline runs these gates. All are available locally via `nix run`:

```bash
# Full verification gate (build + vet + test + race + lint + api-stability + doc-check)
nix run .#verify

# Fast feedback (skips soak tests — use during development)
nix run .#verify-fast

# Parallel test execution (~4min → ~1-2min)
nix run .#verify-parallel

# Individual quality gates (all wired into CI)
nix run .#check-api-stability    # API surface golden check (no breaking changes)
nix run .#check-duplication      # clone detection (art-dupl, threshold 3, semantic)
nix run .#check-layers           # dependency-tier + budget enforcement
nix run .#check-coverage         # coverage drift vs AGENTS.md claims

# Recovery
nix run .#sweep                  # auto-fix formatting + lint drift (run after daemon commits)
```

**Before pushing:** run `nix run .#verify` (or at minimum `nix run .#verify-fast`).
**After the auto-commit daemon touches code:** run `nix run .#sweep` — the daemon
occasionally ships unformatted code or breaking dependency bumps (e.g. the
go-output v0.33.0 incident that broke cqrs-lint for 3+ sessions).
**When you add/remove an exported symbol:** run `cd cmd/api-stability && GOWORK=off
go run main.go -update` to regenerate the golden in the same change.

### Soak Test Environment Variables

The metaengine soak tests verify memory bounding for high-volume event streams.

| Variable | Effect |
|----------|--------|
| `SOAK_SKIP_10M=1` | Skips `TestSoak_MemoryBounded_10M` (~5s/25s-race). Use in CI or when the full verify gate is already running heavy parallel tests. The 50K-event `TestSoak_MemoryBounded` always runs as the smoke variant. |

The soak test uses a double `runtime.GC()` before measuring heap — this ensures
the Go scavenger returns freed spans to the OS, giving accurate retained-heap
readings instead of fragmentation noise.

## cqrs-lint — Domain-Aware Linter

The linter (`cmd/cqrs-lint`) enforces go-cqrs-lite best practices with 186 rules
across 10 categories. It auto-detects which modules a consumer uses and adapts
context-dependent rules accordingly.

```bash
# Self-lint (library mode — skips consumer-coaching rules)
nix run .#lint

# Consumer linting (run from a consumer project)
cqrs-lint ./...
cqrs-lint --scorecard ./...        # module-adoption scorecard
cqrs-lint --health-score ./...     # 0-100 health score with breakdown
cqrs-lint rules                    # list all rules
cqrs-lint explain c008             # interactive rule/preset docs
cqrs-lint doctor                   # detected feature profile
cqrs-lint init                     # generate .cqrs-lint.json config
```

### Config File (`.cqrs-lint.json`)

JSONC format (comments allowed). Supports presets, disabled rules, and
per-rule config overrides:

```jsonc
{
	// Preset: local-cli | production | library | read-only
	"preset": "production",
	// Disable specific rules by ID
	"disabled": ["c008"],
	// Per-rule config
	"c008-ignore-fields": ["ID", "CreatedAt"],
	"c008-ignore-structs": ["TestEvent"],
}
```

### Output Formats

- **Text** (default): colored, grouped by module or aggregate via `--group-by`
- **JSON**: `--format json` — machine-readable for CI integrations
- **SARIF**: `--format sarif` — GitHub Security tab integration
- **Markdown**: `--format markdown` — PR comments

### Grouping

`--group-by none|module|aggregate` controls how findings are organized.
Module groups by go.mod directory; aggregate groups by domain entity (derived
from event type prefixes + decider state types).

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

## Working with AI Agents

This repo is co-developed with AI agents. Follow these rules to avoid chaos:

### Debug-print discipline

Never leave `fmt.Printf("DEBUG ...")` or `panic("DEBUG ...")` in production
code (non-test, non-cmd, non-example packages). These crash tests and pollute
output. If you need debug output, use `slog.Debug()` and remove it before
committing.

### Don't revert changes you didn't author

If you find unexpected uncommitted changes, **investigate before touching**.
Another agent or the user made those changes intentionally. Reverting them is
sabotage. The exception: obvious debug instrumentation that is actively
crashing the test suite (e.g., slice-out-of-range panics on empty payloads).

### Always format before lint directives

Run `nix fmt` BEFORE placing `//nolint` directives. The formatter (golines,
max-len: 120) reformats long lines and moves nolint comments to wrong
positions if placed first.

### Verify before declaring done

Never mark a task complete without reading the actual code path it covers.
Grep ALL documentation for related references after adding or removing public
API. Run `nix run .#test`, not just `go test` on changed modules.

## Release Process

### Per-module tagging

Each module is independently versioned via git tags. Module paths with `/v4`
suffix use semver (e.g., `event/v4.0.0`, `retry/v4.0.1`). Modules without a
version suffix (e.g., `cmd/cqrs-lint`) use `v0.x.y` pre-v1 tags.

```bash
# 1. Verify everything passes
nix run .#build && nix run .#test && nix run .#lint

# 2. Create annotated tag (NEVER lightweight tags)
git tag -a "event/v4.0.1" -m "event/v4.0.1: Brief description of changes"

# 3. Verify tag
git cat-file -t "event/v4.0.1"  # should print "tag"

# 4. Push tags (requires explicit approval)
git push origin "event/v4.0.1"

# 5. Verify Go proxy picks it up (after GitHub Actions CI passes)
GOPROXY=proxy.golang.org go list -m "github.com/larsartmann/go-cqrs-lite/event/v4@v4.0.1"
```

### Critical rules

- **NEVER commit code that doesn't compile.** Run `go build ./...` before every commit.
- **NEVER push tags without running the full verification gate** (`nix run .#build && nix run .#test && nix run .#lint`).
- **ALWAYS use annotated tags** (`git tag -a`), never lightweight tags.
- **ALWAYS update CHANGELOG.md** with release notes before tagging.
- **ALWAYS update `docs/api_surface.txt`** when adding new exported symbols (`cd cmd/api-stability && GOWORK=off go run . -update`).

### Release CI

The `release.yml` workflow triggers on any tag matching `v*` or `*/v*`. It:

1. Auto-discovers all modules from `go.work`
2. Builds, tests (with `-race`), and runs `govulncheck` on each module independently (GOWORK=off)
3. Creates a GitHub Release with auto-generated notes
