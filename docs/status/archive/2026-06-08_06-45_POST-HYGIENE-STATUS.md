# Comprehensive Status Report — Post-Hygiene Session

**Date:** 2026-06-08 06:45 UTC+2
**Branch:** master (up to date with origin)
**Last release:** v2.2.0 (81 commits since v2.1.0)
**Go version:** 1.26.3 · **Modules:** 30 (22 library + 6 examples + 2 cmd)
**Working tree:** clean

---

## Build / Test / Lint — GREEN ACROSS THE BOARD

| Check             | Result                                       |
| ----------------- | -------------------------------------------- |
| `nix run .#build` | PASS                                         |
| `nix run .#test`  | PASS — 37/37 packages                        |
| `nix run .#lint`  | PASS — 0 issues across all 22 linted modules |

---

## a) FULLY DONE

### Core CQRS (All FULLY_FUNCTIONAL)

- **event** — Full event sourcing: creation, immutable events, 19 functional options, metadata, store interfaces (Sink/Source/Journal/SeekableJournal/BackwardsSource), tombstone soft-delete, time-travel queries, error taxonomy (5 families, 13 helpers, 16 sentinels), clock injection, codec, upcasting, reactive streams (samber/ro)
- **command** — Dispatcher with middleware chain, TypedHandler[T], catalog introspection, lifecycle, validation, metadata (aliased to event.Metadata)
- **query** — Dispatcher with TypedHandler[Q, R], pagination, PaginatedResult[T], catalog introspection
- **decider** — Pure-function aggregate: Decider[State], Repository[State] with Execute/Load/LoadAtVersion/LoadAtTime, crash recovery, context enrichment
- **id** — Branded IDs: id.Of[T] phantom types, ULID-backed, 8 built-in types, full serialization
- **dispatcher** — Generic Dispatcher[H, M], LifecycleMixin, CatalogDispatcher
- **schema** — Upcaster, UpcasterRegistry (with cycle detection), VersionedStore
- **snapshot** — Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
- **codec** — JSON and Raw passthrough payload encoding

### In-Memory Implementations (TESTING_ONLY)

- **memory** — MemoryStore, MemoryBus, MemorySnapshotStore, MemoryCheckpointStore — all thread-safe, defensive copies, Close lifecycle

### Middleware Suite (FULLY_FUNCTIONAL)

- **middleware** — 8 concerns × 3 message types = 24 middleware factories: Logging, Metrics, Recovery, Retry (exp backoff + jitter), Tracing (OTel), Validation, Circuit Breaker, OTel Metrics Recorder + SSE broker + Health check

### Event Signing (FULLY_FUNCTIONAL)

- **signing** — HMAC-SHA256, Ed25519, canonical encoding, SignMiddleware/VerifyMiddleware/RequireSignature
- **signing/multisig** — Multi-party signing with heterogeneous algorithms, VerifyAll, VerifierMap

### Auto-Documentation (FULLY_FUNCTIONAL)

- **catalog** — Registry, SchemaFromType[T](<>), immutable Catalog, validation, rich resource model
- **catalog/asyncapi** — AsyncAPI 3.0 YAML/JSON export + golden tests
- **catalog/d2** — D2 diagram generation with cross-service flows + golden tests
- **catalog/openapi** — OpenAPI 3.0.3 export + golden tests
- **catalog/eventcatalog** — EventCatalog MDX + llms.txt generation + golden tests
- **catalog/docserver** — HTTP doc server with Scalar UI, embedded assets

### Storage Backends (FULLY_FUNCTIONAL)

- **storage** — SQLEventStore (PostgreSQL/SQLite), SQLSnapshotStore, SQLCheckpointStore, stream loading, metadata roundtrip, dialect abstraction, TursoSyncDB
- **turso** — Local + cloud sync via Turso (OpenSync → Push/Pull/Checkpoint), delegates to storage
- **pebble** — Embedded KV event store, in-memory backend for testing, async writes option
- **watermill** — Bidirectional event ↔ Watermill message protocol adapter

### Projection & Read Models (FULLY_FUNCTIONAL)

- **projection** — Runner (replay → live), Builder + On[T](<>), HandlerRegistry, dead letter queue, retry with backoff, parallelism option + golden tests
- **listing** — Aggregate listing, tombstone detection, StatusMiddleware, InMemoryAggregateReader, SQL reader

### Infrastructure (FULLY_FUNCTIONAL)

- **otel** — Shared OTel helpers: Tracer, Meter, Spans, Attributes
- **integration** — Cross-module tests for command, event, query, signing, simulation framework

### Developer Tooling

- **cmd/cqrs-gen** — Code generator for typed handler registration
- **cmd/api-stability** — API surface checker against golden files

### Quality Gates

- Zero lint across all 22 library modules (golangci-lint)
- 37/37 test packages pass, 0 races
- CI: build/vet/test/lint/race/coverage + GOWORK=off per-module + benchmark regression + gosec + module layer architecture check

### Property-Based Testing

- **event/** — property_test.go (immutability, idempotency, version monotonicity)
- **decider/** — property_test.go (deterministic decide, fold idempotency)
- **id/** — property_test.go (ULID validity, prefix correctness)
- **command/** — property_test.go (creation, metadata roundtrip, dispatch)
- **query/** — property_test.go (creation, dispatch, pagination bounds)

### Snapshot/Golden Testing

- **catalog/asyncapi** — golden_test.go (JSON + YAML)
- **catalog/openapi** — golden_test.go (JSON)
- **catalog/d2** — golden_test.go (D2 diagram)
- **catalog/eventcatalog** — golden_test.go (MDX + config + llms.txt)
- **projection/** — golden_test.go (replay-order snapshot)
- **integration/** — snapshot_test.go (event JSON serialization, catalog exports)

### Documentation

- Module READMEs for all 22 library modules
- doc.go with pkg.go.dev examples for 12+ modules
- ADRs (12 decisions)
- CONTRIBUTING.md, CONTEXT.md, MIGRATION.md
- docs/EXPERIMENTAL_BUILD_TAGS.md (new)
- docs/STORAGE_GUIDE.md, docs/DOMAIN_LANGUAGE.md, docs/ARCHITECTURE_PATTERNS.md
- docs/planning/ with 80+ execution plans
- docs/status/ with 200+ status reports

### Example Applications

- **example/user/** — Full CQRS demo with: config usage, SSE broker, dual store switching, signing, tombstone/rebirth, catalog, projections, smoke tests
- **example/saga-pattern/** — Saga orchestration pattern demo
- **example/todo/** — Todo list with commands, events, projections
- **example/projection/** — Projection runner demo
- **example/storage/** — SQL storage demo
- **example/listing/** — Aggregate listing demo

---

## b) PARTIALLY DONE

| Area                               | What's Done                                                      | What's Missing                                                                                                                 |
| ---------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **Extended snapshot testing**      | Golden tests on catalog (4 exporters) + projection + integration | Missing on signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory (11 modules) |
| **Sprint 5 example/user features** | Config usage, SSE handler, dual store switching examples         | Missing Playwright E2E tests, JavaScript SSE client                                                                            |
| **Sprint 4 CI**                    | Dockerfile, docker-compose, benchmark regression                 | Missing Docker build CI step (linux amd64 + arm64)                                                                             |
| **command/ coverage**              | 80.5% (rapid PBT added)                                          | Could improve to 90%+ with more edge case tests                                                                                |

---

## c) NOT STARTED

| Item                                        | Priority | Notes                                                  |
| ------------------------------------------- | -------- | ------------------------------------------------------ |
| Playwright E2E tests for example/user/      | MEDIUM   | Sprint 5 — needs npm + browser install                 |
| JavaScript SSE client for example/user/     | MEDIUM   | Sprint 5 — simple EventSource consumer                 |
| Docker build CI step (linux amd64 + arm64)  | MEDIUM   | Sprint 4 — multi-arch build config                     |
| go-snaps across remaining 11 modules        | MEDIUM   | Sprint 6 — signing, middleware, storage, listing, etc. |
| jsonv2 codec experiment behind build tag    | LOW      | Sprint 6 — experimental                                |
| Arena allocation experiment in event module | LOW      | Sprint 6 — performance research                        |
| Documentation site (Docusaurus/MkDocs/Hugo) | LOW      | Long-term                                              |
| pkg.go.dev hosting setup                    | LOW      | Long-term                                              |
| Outbox pattern                              | FUTURE   | Reliable at-least-once publishing                      |
| Schema registry                             | FUTURE   | JSON Schema validation middleware                      |
| NATS adapter                                | FUTURE   | In TODO_LIST                                           |
| Redis adapter                               | FUTURE   | In TODO_LIST                                           |
| gRPC adapter                                | FUTURE   | In TODO_LIST                                           |
| GraphQL adapter                             | FUTURE   | Declined — out of library scope                        |
| WebAssembly target                          | FUTURE   | Long-term                                              |
| Bi-temporal support (ValidAt)               | FUTURE   | In TODO_LIST                                           |
| HLC (Hybrid Logical Clock)                  | FUTURE   | Offline-first feature                                  |

---

## d) TOTALLY FUCKED UP / KNOWN ISSUES

### 1. LSP golangci-lint Version Mismatch (LOW)

The LSP (golangci-lint language server) reports `unknown linters: 'gomodguard_v2'` on every file. The Nix flake uses golangci-lint v2.12.2 where `gomodguard_v2` is correct. The LSP appears to use an older version. `nix run .#lint` passes — CI is unaffected. Creates editor noise.

### 2. example/todo Depguard Violation (LOW)

`example/todo/cmd/api/middleware.go:9` imports `github.com/larsartmann/httputil` which is blocked by depguard. This is an older example module using an external dependency. Not a library module — excluded from CI lint.

### 3. example/ Cosmetic Lint Issues (LOW)

Remaining issues in example/ modules are cosmetic: magic numbers in demo data (mnd), short variable names (varnamelen), repeated strings (goconst). These are acceptable for example/demo code.

### 4. LSP `go mod tidy` Errors (COSMETIC)

LSP reports `pgregory.net/rapid is not in your go.mod file` for command/ and query/. This is because LSP runs with `GOWORK=off` while our go.mod files have replace directives resolved via go.work. `go mod tidy` succeeds in all modules. Pure editor noise.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Fix FEATURES.md accuracy** — DONE this session. Removed 15 stale entries, updated Module Maturity Matrix.
2. **Fix integration/go.mod** — DONE. `go mod tidy` fixed otel indirect directive.
3. **Add PBT to command/ and query/** — DONE. 9 new property-based tests.
4. **Document lint conventions in AGENTS.md** — DONE. golines + nolint placement guide.
5. **Add golden tests to projection/** — DONE. replay-order.json snapshot.
6. **Add example/user features** — DONE. Config usage, SSE broker, dual store switching.
7. **Document experimental build tags** — DONE. docs/EXPERIMENTAL_BUILD_TAGS.md.
8. **Lint example/ modules** — DONE. Fixed noctx, errchkjson, errcheck across 4 modules.

### What I Could Have Done Better

1. **FEATURES.md was stale from the start** — I should have caught that "Dual store runtime example" and "Build tag experiments" were done during my session, not after the fact. I updated FEATURES.md twice — once to remove the big batch, then again to catch the items I just implemented.

2. **Double `---` separator in FEATURES.md** — My first edit left a duplicate horizontal rule. Caught on second pass.

3. **nlreturn lint fights** — My property_test.go files had `return` without blank lines before them, causing lint failures. I should have run `nix fmt && nix run .#lint` before committing.

4. **example/storage/smoke_test.go breakage** — My sed-based replacement of `json.Marshal` broke the function call syntax. Should have used the Edit tool with exact matching instead of python sed.

5. **Didn't test example/user/ new functions** — The `demonstrateConfig()`, `demonstrateSSE()`, `demonstrateDualStore()` functions in example/user/ compile but have no direct test coverage. They're called from main() so the smoke test covers them indirectly.

### Remaining Architecture Improvements

| Issue                                    | Severity | Status                                                                                                                      |
| ---------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------- |
| Middleware 3× duplication                | HIGH     | Mitigated via `middleware/generic.go` (95 lines of shared logic + 27 thin wrappers). Accepted as Go type system limitation. |
| 2 separate ErrHandlerNotFound            | MEDIUM   | Accepted — each module has unique error codes for independent importability.                                                |
| command.Metadata alias to event.Metadata | DONE     | `type Metadata = event.Metadata` alias in place.                                                                            |
| VersionedStore embedded Store            | DONE     | Changed to unexported `inner` field.                                                                                        |
| catalog/ToAny error propagation          | DONE     | Returns `(any, error)` with proper error propagation.                                                                       |

---

## f) Top #25 Things We Should Get Done Next

| #   | Task                                                   | Impact | Effort | Source         |
| --- | ------------------------------------------------------ | ------ | ------ | -------------- |
| 1   | Verify GitHub Actions CI passes for all recent commits | HIGH   | 10min  | 03-41          |
| 2   | Regenerate benchmark baselines post-hygiene changes    | MEDIUM | 8min   | 03-41          |
| 3   | Add go-snaps golden tests on signing/ module           | MEDIUM | 10min  | Sprint 6       |
| 4   | Add go-snaps golden tests on middleware/ module        | MEDIUM | 10min  | Sprint 6       |
| 5   | Add go-snaps golden tests on storage/ module           | MEDIUM | 10min  | Sprint 6       |
| 6   | Add go-snaps golden tests on listing/ module           | MEDIUM | 10min  | Sprint 6       |
| 7   | Add go-snaps golden tests on watermill/ module         | MEDIUM | 10min  | Sprint 6       |
| 8   | Add go-snaps golden tests on pebble/ module            | MEDIUM | 10min  | Sprint 6       |
| 9   | Add go-snaps golden tests on codec/ module             | MEDIUM | 10min  | Sprint 6       |
| 10  | Add go-snaps golden tests on otel/ module              | MEDIUM | 10min  | Sprint 6       |
| 11  | Add go-snaps golden tests on schema/ module            | MEDIUM | 10min  | Sprint 6       |
| 12  | Add go-snaps golden tests on snapshot/ module          | MEDIUM | 10min  | Sprint 6       |
| 13  | Add go-snaps golden tests on memory/ module            | MEDIUM | 10min  | Sprint 6       |
| 14  | Add go-snaps golden tests on turso/ module             | MEDIUM | 10min  | Sprint 6       |
| 15  | Bump command/ coverage from 80.5% to 90%+              | MEDIUM | 30min  | Coverage audit |
| 16  | Add JavaScript SSE client to example/user/             | MEDIUM | 2h     | Sprint 5       |
| 17  | Add Docker build CI step (linux amd64 + arm64)         | MEDIUM | 2h     | Sprint 4       |
| 18  | Fix LSP golangci-lint version mismatch                 | LOW    | 1h     | Editor DX      |
| 19  | Add Playwright setup + health endpoint E2E test        | MEDIUM | 4h     | Sprint 5       |
| 20  | jsonv2 codec experiment behind build tag               | LOW    | 6h     | Sprint 6       |
| 21  | Arena allocation experiment in event module            | LOW    | 8h     | Sprint 6       |
| 22  | Documentation site (Docusaurus/Hugo)                   | LOW    | 20h    | Long-term      |
| 23  | Outbox pattern implementation                          | HIGH   | 40h    | Future         |
| 24  | NATS adapter                                           | MEDIUM | 20h    | Future         |
| 25  | gRPC transport adapter                                 | MEDIUM | 30h    | Future         |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How do we align the LSP's golangci-lint version with the Nix flake's version?**

The LSP (golangci-lint language server) reports `unknown linters: 'gomodguard_v2'` on every file. In the Nix flake (v2.12.2), `gomodguard_v2` is correct. The LSP uses an older version where it's just `gomodguard`.

**What I need to know:**

1. How is the LSP configured in this project? (Neovim/VS Code settings?)
2. Can we point the LSP to use the Nix-installed golangci-lint binary?
3. Or should we add a separate `.golangci_lsp.yml` with `gomodguard` for LSP-only use?

**Why this matters:** Every file shows red squiggles, training developers to ignore LSP diagnostics entirely.

---

## Session Metrics

| Metric                                   | Before Session             | After Session                   |
| ---------------------------------------- | -------------------------- | ------------------------------- |
| FEATURES.md stale entries                | 15                         | 0                               |
| Module Maturity Matrix stale core/ paths | 7                          | 0                               |
| integration/go.mod issues                | 1 (otel indirect)          | 0                               |
| Modules with PBT                         | 3 (event, decider, id)     | 5 (+command, query)             |
| Modules with golden tests                | 5 (integration, 4×catalog) | 6 (+projection)                 |
| example/user features                    | 0 config/SSE/dual-store    | 3 added                         |
| Build tag documentation                  | None                       | docs/EXPERIMENTAL_BUILD_TAGS.md |
| Lint conventions documented              | No                         | AGENTS.md section added         |
| ROADMAP unchecked items                  | 31                         | 24                              |
| Example module lint fixes                | 0                          | Fixed across 4 modules          |
| Build                                    | PASS                       | PASS                            |
| Tests                                    | 37/37                      | 37/37                           |
| Lint                                     | 22/22 × 0                  | 22/22 × 0                       |

---

_Report generated: 2026-06-08 06:45 UTC+2_
_Status: ALL GREEN — 37 packages pass, 0 lint issues, working tree clean_
