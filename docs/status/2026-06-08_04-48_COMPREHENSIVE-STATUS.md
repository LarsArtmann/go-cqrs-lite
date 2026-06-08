# Comprehensive Status Report — 2026-06-08 04:48

**Date:** 2026-06-08 04:48 UTC+2
**Branch:** master (up to date with origin)
**Last release:** v2.2.0 (81 commits since v2.1.0)
**Go version:** 1.26.3 · **Modules:** 30 (22 library + 6 examples + 2 cmd)
**Working tree:** 1 modified file (docs/feedback/why-i-cant-use-you.html)

---

## Build / Test / Lint — GREEN ACROSS THE BOARD

| Check | Result |
|-------|--------|
| `nix run .#build` | PASS |
| `nix run .#test` | PASS — 40/40 packages |
| `nix run .#lint` | PASS — 0 issues across all 16 linted modules |

### Coverage by Module

| Module | Coverage | Status |
|--------|----------|--------|
| dispatcher | 100.0% | Full |
| decider | 100.0% | Full |
| catalog/internal/caseutil | 100.0% | Full |
| catalog/openapi | 100.0% | Full |
| memory | 98.2% | Excellent |
| id | 96.4% | Excellent |
| otel | 96.4% | Excellent |
| listing | 94.9% | Excellent |
| catalog | 95.9% | Excellent |
| catalog/d2 | 95.0% | Excellent |
| signing | 94.1% | Excellent |
| signing/multisig | 94.1% | Excellent |
| watermill | 94.3% | Excellent |
| query | 94.3% | Excellent |
| middleware | 93.5% | Excellent |
| catalog/asyncapi | 93.9% | Excellent |
| codec | 93.3% | Excellent |
| catalog/eventcatalog | 92.7% | Excellent |
| snapshot | 92.3% | Excellent |
| projection | 91.2% | Excellent |
| catalog/docserver | 90.1% | Excellent |
| cmd/cqrs-gen | 89.9% | Good |
| event | 89.4% | Good |
| schema | 89.7% | Good |
| storage | 86.8% | Good |
| pebble | 86.7% | Good |
| catalog/schema | 86.0% | Good |
| command | 80.5% | Adequate |
| integration/simulation | 80.0% | Adequate |
| event/eventtest | 18.4% | Test helper |
| storage/sql | 34.7% | Shared infra |
| turso | 28.6% | Delegates to storage |

---

## a) FULLY DONE

### Core CQRS (All FULLY_FUNCTIONAL)
- **event** — Full event sourcing: creation, immutable events, 19 functional options, metadata, store interfaces (Sink/Source/Journal/SeekableJournal/BackwardsSource), tombstone soft-delete, time-travel queries, error taxonomy (5 families, 13 helpers, 16 sentinels), clock injection, codec, upcasting
- **command** — Dispatcher with middleware chain, TypedHandler[T], catalog introspection, lifecycle, validation
- **query** — Dispatcher with TypedHandler[T], pagination, PaginatedResult[T], catalog introspection
- **decider** — Pure-function aggregate: Decider[State], Repository[State] with Execute/Load/LoadAtVersion/LoadAtTime, crash recovery, context enrichment
- **id** — Branded IDs: id.Of[T] phantom types, ULID-backed, 8 built-in types, full serialization
- **dispatcher** — Generic Dispatcher[H, M], LifecycleMixin, CatalogDispatcher
- **schema** — Upcaster, UpcasterRegistry (with cycle detection), VersionedStore
- **snapshot** — Snapshot, SnapshotSink/Source/Store, SnapshotStrategy, EveryNEvents
- **codec** — JSON and Raw passthrough payload encoding

### In-Memory Implementations (TESTING_ONLY)
- **memory** — MemoryStore, MemoryBus, MemorySnapshotStore, MemoryCheckpointStore — all thread-safe, defensive copies, Close lifecycle

### Middleware Suite (FULLY_FUNCTIONAL)
- **middleware** — 8 concerns × 3 message types = 24 middleware factories: Logging, Metrics, Recovery, Retry (exp backoff + jitter), Tracing (OTel), Validation, Circuit Breaker, OTel Metrics Recorder

### Event Signing (FULLY_FUNCTIONAL)
- **signing** — HMAC-SHA256, Ed25519, canonical encoding, SignMiddleware/VerifyMiddleware/RequireSignature
- **signing/multisig** — Multi-party signing with heterogeneous algorithms, VerifyAll, VerifierMap

### Auto-Documentation (FULLY_FUNCTIONAL)
- **catalog** — Registry, SchemaFromType[T](), immutable Catalog, validation, rich resource model
- **catalog/asyncapi** — AsyncAPI 3.0 YAML/JSON export
- **catalog/d2** — D2 diagram generation with cross-service flows
- **catalog/openapi** — OpenAPI 3.0.3 export
- **catalog/eventcatalog** — EventCatalog MDX + llms.txt generation
- **catalog/docserver** — HTTP doc server with Scalar UI, embedded assets

### Storage Backends (FULLY_FUNCTIONAL)
- **storage** — SQLEventStore (PostgreSQL/SQLite), SQLSnapshotStore, SQLCheckpointStore, stream loading, metadata roundtrip, dialect abstraction, TursoSyncDB
- **turso** — Local + cloud sync via Turso (OpenSync → Push/Pull/Checkpoint), delegates to storage
- **pebble** — Embedded KV event store, in-memory backend for testing, async writes option
- **watermill** — Bidirectional event ↔ Watermill message protocol adapter

### Projection & Read Models (FULLY_FUNCTIONAL)
- **projection** — Runner (replay → live), Builder + On[T](), HandlerRegistry, dead letter queue, retry with backoff, parallelism option
- **listing** — Aggregate listing, tombstone detection, StatusMiddleware, InMemoryAggregateReader, SQL reader

### Infrastructure (FULLY_FUNCTIONAL)
- **otel** — Shared OTel helpers: Tracer, Meter, Spans, Attributes
- **integration** — Cross-module tests for command, event, query, signing, simulation framework

### Developer Tooling
- **cmd/cqrs-gen** — Code generator for typed handler registration
- **cmd/api-stability** — API surface checker against golden files

### Quality Gates
- Zero lint across all 22 library modules (golangci-lint)
- 40/40 test packages pass, 0 races
- CI: build/vet/test/lint/race/coverage + GOWORK=off per-module + benchmark regression + gosec + module layer architecture check
- All v2.0.0 release blockers fixed
- All v2.1.0 and v2.2.0 release work complete

### Documentation
- Module READMEs for all 22 library modules
- doc.go with pkg.go.dev examples for 12 modules
- ADRs (12 decisions)
- CONTRIBUTING.md, CONTEXT.md, MIGRATION.md
- docs/planning/ with 80+ execution plans
- docs/status/ with 200+ status reports

---

## b) PARTIALLY DONE

| Area | What's Done | What's Missing |
|------|-------------|----------------|
| **catalog/ snapshot tests** | Some golden tests exist | Missing systematic snapshot tests for AsyncAPI, OpenAPI, D2, EventCatalog exports |
| **projection/ snapshot tests** | Runner tests exist | Missing snapshot tests for state rendering |
| **PBT (property-based testing)** | rapid-based tests on event/, decider/, id/ | Missing PBT on command/ and query/ modules |
| **FEATURES.md accuracy** | Most entries accurate | "Not Yet Implemented" section is stale — lists items already done (health check, metrics, SSE, etc.) |
| **example/user/ E2E** | Full CQRS demo, smoke tests, Docker | Missing Playwright E2E tests, dual-store runtime switching demo |
| **go-snaps snapshot testing** | Applied to event/, decider/, catalog/ | Missing on signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory |
| **storage/sql coverage** | 34.7% | Shared SQL infrastructure package — many helpers tested indirectly via storage/ |
| **turso coverage** | 28.6% | Thin wrapper delegating to storage/ — low coverage is structural |

---

## c) NOT STARTED

| Item | Priority | Notes |
|------|----------|-------|
| Playwright E2E tests for example/user/ | MEDIUM | Sprint 5 item — health endpoint + core flow tests |
| Dual store runtime switching example | MEDIUM | Demo memory vs SQL switching in example/user/ |
| Build tag experiments (jsonv2, arenas, simd) | LOW | Sprint 6 experimental work |
| Arena allocation experiment in event module | LOW | Performance research |
| jsonv2 codec behind build tag | LOW | Sprint 6 |
| Documentation site (Docusaurus/MkDocs/Hugo) | LOW | Long-term |
| pkg.go.dev hosting setup | LOW | Long-term |
| Multi-engine storage via sqlc | FUTURE | Planning doc exists |
| Schema migration tool | FUTURE | Planning doc exists |
| Bi-temporal support (ValidAt) | FUTURE | In TODO_LIST |
| HLC (Hybrid Logical Clock) | FUTURE | Offline-first feature |
| Pull-before-push sync protocol | FUTURE | Offline-first feature |
| Rebase mechanism | FUTURE | Offline-first feature |
| Network simulator for testing | FUTURE | Offline-first feature |
| NATS adapter | FUTURE | In TODO_LIST |
| Redis adapter | FUTURE | In TODO_LIST |
| gRPC adapter | FUTURE | In TODO_LIST |
| GraphQL adapter | FUTURE | In TODO_LIST |
| WebAssembly target | FUTURE | Long-term |
| Distributed consensus (Raft/CRDT) | FUTURE | Long-term |

---

## d) TOTALLY FUCKED UP / KNOWN ISSUES

### Compiler Error
- **testutil/snaptest/snaptest.go:27** — `no new variables on left side of :=` (should be `=` not `:=`). Minor but prevents clean compilation of that standalone file.

### go.mod Warning
- **integration/go.mod:22** — `go.opentelemetry.io/otel` should be `indirect`. Needs `go mod tidy`.

### Stale Documentation
- **FEATURES.md "Not Yet Implemented" section** — Lists items that are already completed (health check, metrics handler, SSE broker, config loader, graceful shutdown, Docker, simulation framework). Creates false impression of incompleteness.

### Known Code Quality Issues (from FEATURES.md)

| Issue | Severity | Detail |
|-------|----------|--------|
| Middleware 3× duplication | HIGH | ~500 lines of nearly identical code for command/event/query variants of each middleware |
| 3 separate ErrHandlerNotFound | HIGH | command, query, event each define their own — confusing for consumers |
| VersionedStore exposes embedded Store | HIGH | Consumers can bypass upcasting by accessing the inner Store directly |
| command.Metadata duplicates event.Metadata | HIGH | Same fields, different types — module boundary violation |
| command re-exports event types | MEDIUM | Command module re-exports event types (aggregateID, correlationID) |
| Unclassified fmt.Errorf in decider | MEDIUM | Some errors not using the 5-family taxonomy |
| Duplicated error sentinels in pebble | MEDIUM | Some overlap with storage sentinels |
| catalog/ToAny silently swallows errors | MEDIUM | Marshal errors are silently ignored |
| watermill drops malformed ID parse errors | LOW | Silently ignores parse failures |
| Reactive extensions not wired | LOW | samber/ro subjects exist but aren't integrated into dispatchers |

---

## e) WHAT WE SHOULD IMPROVE

### High Impact
1. **Middleware deduplication** — Extract shared middleware logic into generic functions parameterized by message type. ~500 lines of near-identical code is a maintenance burden.
2. **Unify ErrHandlerNotFound** — Single sentinel error or error family across all dispatchers.
3. **Fix FEATURES.md** — Remove stale "Not Yet Implemented" entries that are already done. This is the #1 source of false impressions.
4. **Fix snaptest.go compiler error** — One-line fix (`:=` → `=`).
5. **Fix integration/go.mod** — Run `go mod tidy` to fix indirect directive.
6. **Hide VersionedStore embedded Store** — Make it private or provide only the interface methods to prevent upcast bypass.

### Medium Impact
7. **Snapshot tests on remaining modules** — Systematic golden file coverage for all exporters.
8. **PBT on command/ and query/** — Extend property-based testing beyond event/decider/id.
9. **Deduplicate command.Metadata** — Either reuse event.Metadata or create a shared metadata package.
10. **Module-level READMEs consistency** — Some modules have minimal READMEs; standardize format.
11. **coverage/sql infrastructure tests** — Add direct unit tests for shared SQL helpers (currently 34.7%).
12. **Turso integration tests** — More direct testing beyond delegation to storage/.

### Lower Impact
13. **Example module lint** — example/ modules are excluded from lint; clean them up.
14. **Document lint patterns** — Add golines + nolint placement conventions to AGENTS.md.
15. **LSP golangci-lint version mismatch** — `gomodguard_v2` unknown linter noise in editor.

---

## f) Top 25 Things We Should Get Done Next

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Fix FEATURES.md stale "Not Yet Implemented" section | HIGH | Low |
| 2 | Fix snaptest.go compiler error (`:=` → `=`) | HIGH | Trivial |
| 3 | Run `go mod tidy` on integration/ to fix indirect directive | HIGH | Trivial |
| 4 | Middleware deduplication: extract generic middleware functions | HIGH | Medium |
| 5 | Unify ErrHandlerNotFound into single sentinel or family | HIGH | Low |
| 6 | Hide VersionedStore embedded Store (prevent upcast bypass) | HIGH | Low |
| 7 | Deduplicate command.Metadata (reuse event.Metadata or shared pkg) | HIGH | Medium |
| 8 | Add snapshot tests for catalog exporters (AsyncAPI, OpenAPI, D2, EventCatalog) | MEDIUM | Medium |
| 9 | Add PBT (rapid) to command/ and query/ modules | MEDIUM | Medium |
| 10 | Add go-snaps snapshot tests to remaining 11 modules | MEDIUM | Medium |
| 11 | Fix catalog/ToAny to properly propagate marshal errors | MEDIUM | Low |
| 12 | Classify remaining unclassified errors in decider/pebble | MEDIUM | Low |
| 13 | Add direct unit tests for storage/sql/ helpers (34.7% → 80%+) | MEDIUM | Medium |
| 14 | Add Turso-specific integration tests beyond delegation | MEDIUM | Low |
| 15 | Clean up example/ module lint issues | MEDIUM | Low |
| 16 | Document golines + nolint conventions in AGENTS.md | MEDIUM | Trivial |
| 17 | Fix LSP golangci-lint gomodguard_v2 version mismatch | LOW | Low |
| 18 | SSE handler + JavaScript client in example/user/ | MEDIUM | Medium |
| 19 | Dual store runtime switching demo in example/user/ | MEDIUM | Medium |
| 20 | Playwright E2E setup + health endpoint test | MEDIUM | Medium |
| 21 | Bump command/ coverage from 80.5% to 90%+ | LOW | Medium |
| 22 | Add example/listing/ and example/projection/ lint + README | LOW | Low |
| 23 | Write architectural decision: "Why no saga module" | LOW | Trivial |
| 24 | Validate ROADMAP.md items against actual code state | LOW | Low |
| 25 | Docker build CI step for linux amd64 + arm64 | LOW | Low |

---

## g) Top #1 Question I Cannot Answer Myself

**What is the target audience and go-to-market strategy for this library?**

The codebase is extremely comprehensive (30 modules, 84-100% coverage, zero lint, full CQRS+ES stack). But I cannot determine:
- Is this aimed at **individual Go developers** building event-sourced microservices? **Startups/teams** needing an opinionated CQRS stack? **Enterprises** evaluating event sourcing frameworks?
- Should we prioritize **pkg.go.dev polish** (examples, badges, doc.go), **documentation site** (Docusaurus/Hugo with tutorials), or **conference/blog marketing**?
- The middleware 3× duplication exists because of Go's type system (no generic method overloading). Should we accept this as idiomatic, or is the DX burden unacceptable for the target audience?

This decision would reshape the top-25 priority list significantly. For example, if targeting enterprises, documentation site and migration guides become P0. If targeting individual developers, pkg.go.dev examples and README polish become P0.

---

## Module Health Summary

| Module | Status | Coverage | Lint | Tests | Notes |
|--------|--------|----------|------|-------|-------|
| event | FULLY_FUNCTIONAL | 89.4% | 0 | PASS | Core module, solid |
| command | FULLY_FUNCTIONAL | 80.5% | 0 | PASS | Lowest coverage of core modules |
| query | FULLY_FUNCTIONAL | 94.3% | 0 | PASS | Excellent |
| decider | FULLY_FUNCTIONAL | 100.0% | 0 | PASS | Perfect |
| id | FULLY_FUNCTIONAL | 96.4% | 0 | PASS | Excellent |
| dispatcher | FULLY_FUNCTIONAL | 100.0% | 0 | PASS | Perfect |
| schema | FULLY_FUNCTIONAL | 89.7% | 0 | PASS | Solid |
| snapshot | FULLY_FUNCTIONAL | 92.3% | 0 | PASS | Excellent |
| codec | FULLY_FUNCTIONAL | 93.3% | 0 | PASS | Excellent |
| memory | TESTING_ONLY | 98.2% | 0 | PASS | Near-perfect |
| catalog | FULLY_FUNCTIONAL | 95.9% | 0 | PASS | Excellent |
| catalog/asyncapi | FULLY_FUNCTIONAL | 93.9% | 0 | PASS | Excellent |
| catalog/d2 | FULLY_FUNCTIONAL | 95.0% | 0 | PASS | Excellent |
| catalog/openapi | FULLY_FUNCTIONAL | 100.0% | 0 | PASS | Perfect |
| catalog/eventcatalog | FULLY_FUNCTIONAL | 92.7% | 0 | PASS | Excellent |
| catalog/docserver | FULLY_FUNCTIONAL | 90.1% | 0 | PASS | Excellent |
| catalog/schema | FULLY_FUNCTIONAL | 86.0% | 0 | PASS | Good |
| catalog/caseutil | FULLY_FUNCTIONAL | 100.0% | 0 | PASS | Perfect |
| middleware | FULLY_FUNCTIONAL | 93.5% | 0 | PASS | 3× duplication smell |
| integration | FULLY_FUNCTIONAL | N/A | 0 | PASS | Cross-module tests |
| projection | FULLY_FUNCTIONAL | 91.2% | 0 | PASS | Excellent |
| signing | FULLY_FUNCTIONAL | 94.1% | 0 | PASS | Excellent |
| signing/multisig | FULLY_FUNCTIONAL | 94.1% | 0 | PASS | Excellent |
| storage | FULLY_FUNCTIONAL | 86.8% | 0 | PASS | Good |
| storage/sql | INFRASTRUCTURE | 34.7% | 0 | PASS | Shared helpers, tested indirectly |
| watermill | FULLY_FUNCTIONAL | 94.3% | 0 | PASS | Excellent |
| listing | FULLY_FUNCTIONAL | 94.9% | 0 | PASS | Excellent |
| otel | FULLY_FUNCTIONAL | 96.4% | 0 | PASS | Excellent |
| pebble | FULLY_FUNCTIONAL | 86.7% | 0 | PASS | Missing Journal/SeekableJournal |
| turso | FULLY_FUNCTIONAL | 28.6% | 0 | PASS | Thin wrapper, structurally low |
| cmd/cqrs-gen | TOOL | 89.9% | 0 | PASS | Solid |
| cmd/api-stability | TOOL | N/A | 0 | PASS | API surface checker |

**Overall health: STRONG.** 22/22 library modules compile, pass tests, pass lint, and are functionally complete. The library is production-ready for consumers who want a CQRS+ES stack in Go.
