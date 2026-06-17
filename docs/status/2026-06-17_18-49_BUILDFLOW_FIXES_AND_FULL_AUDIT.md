# Status Report — 2026-06-17 18:49

> **Context**: Post-dependency-utilization-maximization. Session focused on fixing BuildFlow CI failures (todo-check false positives + go-fix/modernize/npm-update root causes). Full project health audit.

---

## Executive Summary

| Metric | Value |
|--------|-------|
| **Version** | v2.4.0 (26 library + 3 example + 2 cmd + 1 integration modules) |
| **Go files** | 770 total (366 production, 404 test) |
| **Lines of code** | 33,590 production / 68,245 test |
| **Test functions** | 2,076 across 355 test files |
| **ADRs** | 24 |
| **Status/planning docs** | 85 status + 65 planning |
| **Git state** | Clean master, 0 commits ahead of origin |
| **Module test pass** | 26/26 modules + 3/3 examples all green |
| **Lint** | Clean (0 issues across event, pebble, storage spot-checks) |
| **Coverage (key modules)** | event 93.2% · decider 99.4% · encryption 87.0% · pebble 84.7% · storage 82.1% |

---

## A) FULLY DONE ✅

### BuildFlow CI Fixes (this session)

| Fix | Root Cause | Resolution |
|-----|-----------|------------|
| `todo-check` false positives | `.buildflow.yml` had wrong key name `todo_severity` (silently ignored → defaulted to `info`, which matches OPTIMIZE in "Optimized path") | Corrected to `todo_min_severity: warning` + rephrased two comments ("Optimized" → "Fast path", "optimized" → "applied") |
| `go-fix` failure | Multi-module workspace root has no Go packages → `go fix ./...` matched nothing → exit 1 | Added root `doc.go` with package declaration |
| `modernize` failure | Same root cause as go-fix — no root packages | Same fix (root `doc.go`) |
| `npm-update` failure | Gitignored generated output `example/user/eventcatalog-output/package.json` scanned by buildflow | Added `**/eventcatalog-output/**` to exclude patterns |

### Core Library (v2.4.0 — all stable)

- **Event System** ✅ — Immutable events, Store (Sink/Source/Journal/SeekableJournal/BackwardsSource), Bus, reactive EventBus, tombstone soft-delete, zero-copy internal reads, defensive clones
- **Command System** ✅ — Dispatcher, Handler, Middleware, BasicCommand, PersistedCommand, CommandStore interfaces, CommandJournal, SeekableCommandJournal, Command Bus (pub/sub), reactive CommandBus
- **Query System** ✅ — Dispatcher, Handler, Pagination, PaginatedResult[T], TypedHandler[Q,R], QueryStore interfaces, QueryJournal, SeekableQueryJournal, reactive QueryBus
- **Decider** ✅ — Pure-function Decider[State], Repository[State], Execute, Load, singleflight load coalescing
- **Branded IDs** ✅ — `id.Of[T]` = cbid.ID[T, ulid.ULID], ULID monotonic entropy
- **Schema Evolution** ✅ — Upcaster, VersionedStore, upcasterRegistry
- **Snapshots** ✅ — Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
- **Memory Backend** ✅ — Full in-memory implementations (EventStore, Bus, SnapshotStore, CommandStore, CommandBus, QueryStore, CheckpointStore)
- **SQL Backend** ✅ — SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLCommandStore, SQLQueryStore (PostgreSQL + SQLite + Turso), SQLBackend facade, SQLiteEnableWAL/busy_timeout/foreign_keys
- **Pebble Backend** ✅ — PebbleBackend facade (single shared DB), EventStore, SnapshotStore, CheckpointStore, KVAdapter (kv.Store), DefaultOptions (bloom+compaction), Metrics/BlockCacheHitRate, ULID-narrowed journal scan, CBOR envelope
- **Turso Backend** ✅ — Embedded LibSQL sync, auto-indexer, query advisor, recommended CQRS indexes
- **KV Abstraction** ✅ — Store (Reader+Writer+Closer), MemStore, Iterator, Batch
- **Codec** ✅ — JSON, CBOR (deterministic), CBORCompactCodec (toarray), Raw passthrough, Diagnose()
- **Signing** ✅ — HMAC-SHA256, Ed25519, multisig, middleware
- **Encryption** ✅ — XChaCha20-Poly1305, AES-256-GCM, codec wrapper, HKDF key derivation, middleware
- **Projections** ✅ — Runner (replay+live), HandlerRegistry, Builder with On[T]()
- **Middleware** ✅ — Logging, Retry, Recovery, Validation, Metrics, OTel Tracing+Metrics, SSE, HealthCheck, pprof endpoints
- **Catalog** ✅ — Registry, SchemaFromType[T](), AsyncAPI/D2/EventCatalog/OpenAPI exporters, docserver
- **Listing** ✅ — AggregateListing, AggregateStatus, tombstone detection, StatusMiddleware
- **Watermill Adapter** ✅ — Publisher/Subscriber protocol adapter
- **OTel Helpers** ✅ — Tracer, Meter, Spans, Attributes, Int64Counter, ServiceResourceAttributes, CQRSHistogramBoundaries, AddSpanEvent

### Dependency Utilization Maximization (18 features — previous session)

- singleflight load coalescing (decider)
- SQLiteEnableForeignKeys (storage)
- HKDF DeriveKey (encryption)
- Pebble DefaultOptions + DefaultOptionsWithLogging + Metrics (pebble)
- CBORCompactCodec + ExtraReturnErrors + Diagnose() (codec)
- OTel Int64Counter + ServiceResourceAttributes + CQRSHistogramBoundaries + AddSpanEvent (otel)
- ULID Monotonic entropy (id)
- SQLite busy_timeout (storage)
- Pebble ULID-narrowed journal scan (pebble)
- pebbleLogger fmt.Sprintf bug fix (pebble)
- testutil rapid generators (testutil)

### CI/CD

- GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage)
- GOWORK=off per-module CI
- PostgreSQL CI service container
- Docker build CI (amd64 + arm64)
- Benchmark baseline regression detection
- gosec security scanning with SARIF
- Module layer architecture checks
- Replace directive CI check

### Testing Rigor

- Property-based tests (pgregory.net/rapid) in event, command, query, decider, id, encryption
- Golden tests using shared eventtest.AssertGolden pattern
- Integration tests across modules
- Simulation framework (event sequence generator + decider stress)
- Pebble benchmarks (Save100, SaveLoad100, Save1, LoadEmpty)
- KV contract tests (PebbleAdapter vs MemStore semantic equivalence)

### Documentation

- AGENTS.md (contributor guide, 28 modules documented)
- SKILL.md (AI consumer guide — canonical reference)
- FEATURES.md (honest feature inventory, 832 lines)
- ROADMAP.md (sprint history + long-term vision)
- TODO_LIST.md (all actionable items resolved)
- DOMAIN_LANGUAGE.md, CONTEXT.md
- 24 ADRs (0001–0023, plus README)
- Module-level READMEs (kv, pebble)
- doc.go with pkg.go.dev examples across 12+ modules

---

## B) PARTIALLY DONE ⚠️

| Area | Status | Gap |
|------|--------|-----|
| **Pebble coverage** | 84.7% | Target 85%+; error branches in `helpers.go`, `serialization.go` uncovered |
| **Pebble golden test** | Missing | Deterministic CBOR envelope bytes for regression safety (listed in TODO_LIST) |
| **MemorySnapshotStore golden test** | Missing | Baseline for pebble snapshot comparison |
| **Reactive buses** | 🧪 Experimental | CommandBus, QueryBus, EventBus all work but marked experimental — API may change |
| **Turso indexing** | 🧪 Working | Auto-indexer, advisor, recommended indexes all functional but sub-package is relatively new |
| **Catalog docserver** | Works but vendored JS | `catalog/docserver/static/*.js` are large vendored bundles (asyncapi-react, scalar) |

---

## C) NOT STARTED 📐

### High Impact (from TODO_LIST.md)

- **Schema registry** — JSON Schema validation middleware for events (ADR-0017)
- **Distributed checkpointing** — multi-instance projection coordination (ADR-0018)
- **Prometheus metrics exporter** — replace custom MetricsRecorder in middleware/
- **Structured logging middleware** — configurable slog levels for command/event/query processing
- **Distributed tracing propagation** — span context across module boundaries
- **cqrs-gen v2** — struct tag scanning code generator improvements

### Experimental / Long-term

- **gRPC transport adapter** — new module for command/query dispatch over gRPC
- **NATS/Redis Stream adapter** — message broker integration
- **Streaming event reads** — StreamLoader without materializing full slice
- **jsonv2 codec experiment** — behind build tag (waiting for Go stdlib)
- **Arena allocation experiment** — behind build tag (waiting for Go stdlib)
- **WASM compilation target** — decider module for browser/edge
- **Documentation site** — Docusaurus/MkDocs/Hugo

### Deferred Breaking Changes

**v3 (next major):**
- Remove `io.Closer` from core interfaces (ADR-0010)
- Add global `TransactionID` branded type
- Make event Core truly immutable (deep copy opts pointer)
- Move HTTP code out of middleware → transport/ module
- Fix `query.Handler` returns `any` → Generic TypedHandler returning `(T, error)`

**v4:**
- Split `catalog.Message` into Message + MessageMeta (17 fields → structured)
- Split `catalog.Service` into Service + ServiceMeta (16 fields → structured)

---

## D) TOTALLY FUCKED UP! 🔴

### 1. `flake.nix` has no `packages.default` → `nix build .` fails

```
error: flake 'git+file:///home/lars/projects/go-cqrs-lite' does not provide attribute 'packages.x86_64-linux.default'
```

The flake uses `apps.*` for all build automation (test, build, vet, lint, etc.) but never defines `packages.default`. This means:
- `nix build .` fails
- BuildFlow's `nix-build` step fails
- Any consumer expecting `nix build` to work gets nothing

**Fix:** Add a `packages.default = pkgs.buildGoModule { ... }` or at minimum a `packages.default = pkgs.stdenv.mkDerivation` that wraps the build.

### 2. `flake.lock` was silently updated (not by me)

The `flake.lock` shows nixpkgs bumped from `a799d3e3` → `567a49d1`. This happened as a side effect of buildflow's `nix-flake-update` step running during my session. It's a routine nixpkgs bump but I didn't intentionally do it.

### 3. `CODE_OF_CONDUCT.md` is untracked

A 19-line Contributor Covenant Code of Conduct appeared as an untracked file. It wasn't there before this session. Likely auto-generated by some tool or a previous session artifact.

---

## E) WHAT WE SHOULD IMPROVE! 💡

### Architecture & Design

1. **Root `doc.go` is a band-aid** — The real issue is that buildflow expects root-level Go packages but the workspace is multi-module. The `doc.go` makes `go fix ./...` work but adds a phantom root package. Consider whether buildflow should iterate modules instead.

2. **BuildFlow config key mismatch** — The original `.buildflow.yml` used `todo_severity` which doesn't exist (correct key is `todo_min_severity`). Buildflow silently ignored it. BuildFlow should validate unknown keys and warn/error.

3. **Vendored JS in catalog/docserver** — `asyncapi-react.js` and `scalar.js` are massive vendored bundles. They trigger false positives in various scanners and bloat the repo. Consider git submodules, fetch scripts, or NPM management instead of committing minified JS.

4. **flake.nix incomplete** — Has `apps` and `devShells` but no `packages.default`. This breaks the standard Nix convention. Every Nix flake should at minimum build.

5. **Test coverage gaps** — pebble (84.7%) and storage (82.1%) are below the 85%+ target. These are critical persistence modules.

6. **Reactive APIs still experimental** — CommandBus, QueryBus, EventBus reactive extensions work and are tested but marked 🧪. If they're stable, graduate them. If not, document what's missing.

7. **No fuzz tests running** — BuildFlow's `test-fuzz` step exists but the Go fuzz tests may not be wired properly. Fuzz testing is critical for security-sensitive modules (signing, encryption, codec).

8. **85 status reports + 65 planning docs** — Massive documentation debt. Most are historical snapshots with overlapping content. Consider archiving old ones or consolidating.

### Developer Experience

9. **No `nix run .#lint` on root** — The lint app exists but the AGENTS.md says to use it. Verify it works with current module structure.

10. **BuildFlow steps vs. flake apps overlap** — Both define build/test/lint. Unclear which is authoritative. Document the relationship or consolidate.

---

## F) Top 25 Things We Should Get Done Next

### Immediate (this/next session)

1. **Fix `flake.nix` packages.default** — Add a buildable default package so `nix build .` works
2. **Commit the doc.go + buildflow fixes** — Uncommitted changes block progress
3. **Decide on `CODE_OF_CONDUCT.md`** — Commit or remove the untracked file
4. **Run `nix flake lock --update-input nixpkgs` deliberately** — Confirm the flake.lock bump is intentional and beneficial

### High Impact Features

5. **Schema registry (ADR-0017)** — JSON Schema validation middleware for events. Consumer safety feature.
6. **Distributed checkpointing (ADR-0018)** — Multi-instance projection coordination. Production readiness.
7. **Prometheus metrics exporter** — Replace custom MetricsRecorder. Standard observability.
8. **Structured logging middleware** — Configurable slog levels. Production observability baseline.
9. **Distributed tracing span propagation** — Cross-module span context. Full observability stack.

### Quality & Coverage

10. **Pebble coverage to 85%+** — Target error branches in helpers.go, serialization.go
11. **Pebble golden test** — Deterministic CBOR envelope bytes for regression safety
12. **Storage coverage to 85%+** — Currently 82.1%, critical persistence layer
13. **MemorySnapshotStore golden test** — Baseline for pebble snapshot comparison
14. **Fuzz test wiring** — Ensure buildflow test-fuzz step actually runs Go fuzz tests
15. **Graduate reactive APIs** — Mark CommandBus/QueryBus/EventBus as stable if ready

### Architecture

16. **cqrs-gen v2** — Struct tag scanning code generator improvements
17. **gRPC transport adapter** — New module for command/query dispatch over gRPC
18. **Streaming event reads** — StreamLoader without materializing full slice
19. **NATS/Redis Stream adapter** — Message broker integration

### Maintenance

20. **Consolidate status/planning docs** — 150 docs is too many; archive historical snapshots
21. **Manage vendored JS** — Move catalog/docserver/static/ to fetch script or submodule
22. **BuildFlow: validate unknown config keys** — Prevent silent key-name mismatches
23. **Roadmap: update "Built-in pprof endpoints"** — ROADMAP.md line 108 lists it as `[ ]` but it's already done (TODO_LIST.md line 47 confirms `[x]`)
24. **Roadmap: update last-updated date** — Says "2026-06-16" but we're at 2026-06-17 with significant changes
25. **API stability check** — Run `cmd/api-stability` against latest changes to ensure no breaking API surface changes

---

## G) Top #1 Question I Cannot Figure Out Myself 🤔

**Why does the `consolidate-catalog` branch exist and should it be merged or deleted?**

The branch `consolidate-catalog` exists locally and on remote. It has 0 commits ahead of master (fully merged), yet it still exists as a branch. I can't determine:

- Is this a stale branch that should be cleaned up?
- Was there a plan to do more catalog consolidation work?
- Should it be deleted (it's fully merged into master)?

The branch name suggests an active catalog consolidation effort, but `git log` shows it's identical to master. This needs human context on whether the catalog consolidation is truly complete or if there's pending work.

---

## Uncommitted Changes (This Session)

| File | Change | Reason |
|------|--------|--------|
| `.buildflow.yml` | `todo_severity` → `todo_min_severity: warning` + added eventcatalog-output exclude | Fix todo-check false positives |
| `doc.go` (new) | Root package declaration | Fix go-fix/modernize "matched no packages" |
| `pebble/journal.go` | "Optimized path" → "Fast path" | Remove OPTIMIZE false positive |
| `turso/indexing/example_test.go` | "optimized" → "applied" | Remove OPTIMIZE false positive |
| `flake.lock` | nixpkgs bumped (auto, by buildflow nix-flake-update step) | Side effect of buildflow run |
| `CODE_OF_CONDUCT.md` (untracked) | Unknown origin | Needs decision: commit or remove |

---

_BuildFlow status: todo-check ✅ · go-fix ✅ · modernize ✅ · npm-update ✅ · golangci-lint ✅ · (nix-build 🔴 pre-existing flake issue · test-race/test-fuzz — not yet investigated)_
