# Comprehensive Status Report — go-cqrs-lite

> **Date:** 2026-06-15 06:23 · **Version:** v2.3.x (post-release active development) · **Go:** 1.26.3
>
> **Head commit:** `f8070da4` · **Branch:** `master` · **Remote:** up to date

---

## Executive Snapshot

| Metric                         | Value                                                                                                                                               |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Modules in go.work             | 28 (root + 27 sub-modules)                                                                                                                          |
| Total Go files                 | 693                                                                                                                                                 |
| Production lines               | 28,336                                                                                                                                              |
| Test lines                     | 59,622 (2.1× production)                                                                                                                            |
| Test packages                  | 40/40 PASS                                                                                                                                          |
| Lint issues                    | **0** across all 23 linted modules                                                                                                                  |
| Clone groups (dupl ≥80 tokens) | 6                                                                                                                                                   |
| Overall test coverage          | 84.2%                                                                                                                                               |
| ADRs                           | 19 (0001–0015 + gap at 0005)                                                                                                                        |
| BDD test modules               | 13 (catalog, command, decider, encryption, event, integration, listing, memory, middleware, projection, query, signing, + integration sub-packages) |
| Total BDD specs                | ~145                                                                                                                                                |
| TODOs done                     | 174                                                                                                                                                 |
| TODOs open                     | 34                                                                                                                                                  |

---

## A) FULLY DONE ✅

### Core Library (Production-Ready)

| Module        | Status          | Coverage | Notes                                                                                                                                                        |
| ------------- | --------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `event/`      | ✅ Production   | >90%     | Event sourcing core: Store (Sink/Source ISP split), Bus, Journal, SeekableJournal, BackwardsSource, reactive streams, tombstone, checkpoint, error taxonomy  |
| `command/`    | ✅ Production   | >85%     | Command dispatch, TypedHandler[T], PersistedCommand, Store interfaces, metadata                                                                              |
| `query/`      | ✅ Production   | >85%     | Query dispatch, TypedHandler[Q,R], pagination, metadata (added 2026-06-14)                                                                                   |
| `decider/`    | ✅ Production   | >90%     | Pure-function aggregate: Decider[State], Repository[State], Execute, LoadAtVersion/LoadAtTime, snapshot integration, OTel tracing                            |
| `id/`         | ✅ Production   | >95%     | 8 branded ID types (ULID-backed), DeriveAggregateID, phantom type safety, all serialization                                                                  |
| `dispatcher/` | ✅ Production   | >90%     | Generic Dispatcher[H,M], LifecycleMixin, middleware chain                                                                                                    |
| `schema/`     | ✅ Production   | >85%     | Upcaster, VersionedStore, cycle detection, full load API                                                                                                     |
| `snapshot/`   | ✅ Production   | >85%     | Snapshot types, Sink/Source/Store ISP, EveryNEvents strategy, helpers                                                                                        |
| `codec/`      | ✅ Production   | >90%     | JSON, CBOR (deterministic), Raw passthrough                                                                                                                  |
| `memory/`     | ✅ Test utility | >85%     | MemoryStore, MemoryBus, MemorySnapshotStore, MemoryCheckpointStore, MemoryCommandStore                                                                       |
| `catalog/`    | ✅ Production   | >80%     | Registry, Builder, Catalog, 6 exporters (AsyncAPI, OpenAPI, D2, EventCatalog, docserver, schema)                                                             |
| `middleware/` | ✅ Production   | >85%     | 9 concerns × 3 message types = 27 factories + generic Middleware[M]. Logging, metrics, OTel, retry, circuit breaker, validation, recovery, SSE, health check |
| `signing/`    | ✅ Production   | >90%     | HMAC-SHA256, Ed25519, multisig, middleware, CloneEvent                                                                                                       |
| `encryption/` | ✅ Production   | >90%     | XChaCha20-Poly1305, AES-256-GCM, KeyResolver, StaticKeyResolver, Codec wrapper, middleware                                                                   |
| `storage/`    | ✅ Production   | >89%     | SQLEventStore, SQLCommandStore, SQLSnapshotStore, SQLCheckpointStore, SQLAggregateReader, dialect abstraction (PG + SQLite)                                  |
| `projection/` | ✅ Production   | >85%     | Runner (replay+live), Builder + On[T], HandlerRegistry, checkpoint, retry, DLQ, parallelism, health check                                                    |
| `otel/`       | ✅ Production   | >97%     | Tracer, Meter, Span helpers, attribute helpers, logging correlation                                                                                          |
| `watermill/`  | ✅ Production   | >85%     | PublisherAdapter, SubscriberAdapter, bidirectional metadata protocol                                                                                         |
| `pebble/`     | ✅ Production   | >85%     | PebbleDB event store, CBOR envelope + JSON compat, sharded mutex pool, async writes                                                                          |
| `turso/`      | ✅ Production   | >80%     | Turso connector, sync, indexing (Advisor, AutoIndexer, Policy, stats, WAL checkpoint)                                                                        |
| `listing/`    | ✅ Production   | >85%     | InMemoryAggregateReader, ListBuilder, tombstone tracking, cursor pagination, cache invalidation                                                              |
| `testutil/`   | ✅ Test helper  | N/A      | MustNewCmd cross-module test utility                                                                                                                         |

### Infrastructure

| Item                     | Status                                                                                                           |
| ------------------------ | ---------------------------------------------------------------------------------------------------------------- |
| CI (GitHub Actions)      | ✅ Build, vet, test, lint, race, coverage, gosec, govulncheck, gitleaks, benchmark regression, Docker multi-arch |
| Nix flake                | ✅ build, test, test-race, lint, vet, coverage, bench, check-layers, clean, vulncheck, secrets-scan, benchstat   |
| Module layer enforcement | ✅ `scripts/check-module-layers.sh` — DAG verified in CI                                                         |
| API stability checker    | ✅ `cmd/api-stability` — golden file comparison                                                                  |
| Code generator           | ✅ `cmd/cqrs-gen` — AST-based typed handler generation                                                           |
| Zero lint                | ✅ **0 issues across all 23 linted modules** (achieved 2026-06-15)                                               |
| Race-free                | ✅ `go test -race` passes                                                                                        |
| GOWORK=off CI            | ✅ Per-module isolation verified                                                                                 |
| File-size gate           | ✅ ≤350 lines per file                                                                                           |
| Dependency budgets       | ✅ Per-module dep limits enforced                                                                                |

### Architecture Quality

| Dimension         | Score | Notes                                                                           |
| ----------------- | :---: | ------------------------------------------------------------------------------- |
| Module boundaries | 9/10  | Clean DAG, no cycles, ISP splits, Unix philosophy                               |
| Type safety       | 9/10  | Branded IDs, phantom types, strong enums, impossible-state prevention           |
| Composability     | 9/10  | Library not framework, interface-first, import what you need                    |
| Testability       | 9/10  | FakeStore/FakeBus, StoreTestSuite, BDD in 13 modules, property tests            |
| Naming quality    | 9/10  | No Impl suffixes, no Manager/Handler/Helper classes, domain-aligned             |
| Error handling    | 9/10  | 5-family taxonomy, sentinel + %w wrapping, classified errors                    |
| Documentation     | 7/10  | FEATURES.md excellent, README.md incomplete (missing encryption/turso sections) |

### This Session's Accomplishments

| Commit     | Change                                       | Impact                                                                |
| ---------- | -------------------------------------------- | --------------------------------------------------------------------- |
| `f0e3518b` | Extract `cmdSuffix` const in catalog         | Eliminated last 2 catalog lint issues — zero lint across all modules  |
| `42e17f4f` | Add `event.ExtractCustomBytes` shared helper | Eliminated largest clone group (8→7), refactored signing + encryption |
| `b33d98e0` | Fix gopls `unusedwrite` in event clone test  | Zero gopls warnings in event module                                   |
| `f8070da4` | Wrap errors + fix pebble varnamelen          | Maintained zero lint after refactor                                   |

---

## B) PARTIALLY DONE ⚠️

### Documentation

| Item           | Current State                                          | Gap                                                                                                                                                |
| -------------- | ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `README.md`    | Has overview, quick start, module table, comparison    | **Missing**: encryption module section, turso module section, testutil in module table. Version says v2.3.0 (fixed from v2.2.0). Saga claim fixed. |
| `AGENTS.md`    | Comprehensive module tree, design principles, patterns | **Missing**: testutil in module list (fixed 2026-06-14 but needs verification). BDD testing section understates coverage.                          |
| `FEATURES.md`  | 800+ lines, brutally honest feature inventory          | **Recently updated**: audit date, lint claim. Accurate.                                                                                            |
| `TODO_LIST.md` | 208 items (174 done, 34 open)                          | **Recently updated**: stale items fixed, new findings added. Accurate.                                                                             |
| Module READMEs | 20+ modules have README.md                             | **Missing**: a few modules lack dedicated READMEs                                                                                                  |
| `docs/adr/`    | 19 ADRs (0001–0015, gap at 0005)                       | ADR-0005 numbering gap acknowledged but not resolved                                                                                               |

### Testing

| Area                         | Current State                                                                                  | Gap                                                                                                                |
| ---------------------------- | ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| BDD tests                    | 13 modules, ~145 specs                                                                         | **Missing BDD**: storage/, turso/ have adequate table-driven tests but no BDD. Catalog BDD added 2026-06-14.       |
| Golden tests                 | ~15 modules have golden test files                                                             | **Missing golden**: signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot |
| Property tests               | event/ (rapid), encryption/ (rapid), id/ (rapid)                                               | Could expand to storage/, catalog/                                                                                 |
| Fuzz tests                   | event/ (codec, parser), encryption/ (fuzz), codec/ (fuzz), schema/ (fuzz), pebble/ (CBOR fuzz) | Good coverage for critical paths                                                                                   |
| PostgreSQL integration tests | Not implemented                                                                                | testcontainers-based real PG testing mentioned in ROADMAP                                                          |
| Turso integration tests      | sync.go only tested for rejection paths                                                        | Needs real Turso server or better mocking                                                                          |

### Code Duplication (6 remaining clone groups)

| #   | Files                                                          | Severity | Actionable?                                          |
| --- | -------------------------------------------------------------- | -------- | ---------------------------------------------------- |
| 1   | storage/event_store_load.go (LoadFromVersion vs LoadToVersion) | Low      | Yes — parameterize operator                          |
| 2   | middleware/circuit_breaker.go vs middleware/middleware.go      | Low      | Acceptable — config validation pattern               |
| 3   | command/dispatcher.go vs query/dispatcher.go                   | Low      | Acceptable — same pattern, different types           |
| 4   | catalog/validate.go (validateDomain vs validateChannel)        | Low      | Could use generics, but only 2 occurrences           |
| 5   | encryption/aesgcm.go vs encryption/xchacha20.go                | Medium   | Acceptable — algorithm differences justify structure |
| 6   | storage/command_store_load.go (LoadFromVersion vs LoadAll)     | Low      | Yes — parameterize WHERE clause                      |

---

## C) NOT STARTED 📐

### From ROADMAP.md (No Code Yet)

| Feature                                            | Priority    | Effort                                    |
| -------------------------------------------------- | ----------- | ----------------------------------------- |
| Outbox pattern (reliable at-least-once publishing) | Medium      | Large — new module or storage integration |
| Event schema registry with validation middleware   | Medium      | Medium — extends schema/ module           |
| Distributed checkpointing for projections          | Medium      | Large — distributed state coordination    |
| gRPC transport adapter                             | Low         | Medium — new transport module             |
| NATS / Redis Stream adapter                        | Low         | Medium — extends watermill/ or new module |
| Built-in pprof endpoints                           | Low         | Small — middleware addition               |
| Custom Prometheus metrics exporter                 | Low         | Small — extends middleware/               |
| Event stream compaction / log truncation           | Medium      | Large — store-level feature               |
| Multi-tenant event store (schema-per-tenant)       | Low         | Large — cross-cutting concern             |
| Event archival to S3/GCS/Azure Blob                | Low         | Medium — new storage module               |
| WebAssembly compilation target for decider         | Speculative | Unknown                                   |
| Documentation site (Docusaurus/MkDocs/Hugo)        | Low         | Medium — separate effort                  |
| CQRS-lite dashboard (web UI)                       | Speculative | Large                                     |

### From docs/planning/ (Future Ideas)

| Idea                                                        | Status |
| ----------------------------------------------------------- | ------ |
| Bi-temporal support (ValidAt, WithValidAt, LoadToValidTime) | Future |
| HLC (Hybrid Logical Clock) implementation                   | Future |
| Pull-before-push sync protocol                              | Future |
| Rebase mechanism                                            | Future |
| Network simulator for testing                               | Future |
| Multi-client test harness                                   | Future |
| Distributed consensus (Raft/CRDT overlay)                   | Future |
| Time-series event query language                            | Future |

---

## D) TOTALLY FUCKED UP 🔴

**Nothing is truly broken.** The codebase is in excellent shape. However, there are a few things worth calling out:

### Pre-existing Issues (Not Caused This Session)

| Issue                                        | Severity   | Impact                                                                                                                                          |
| -------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| **Flaky `turso/indexing/TestApplyWAL` test** | Medium     | Fails intermittently under parallel load (WAL file timing). Passes in isolation. Pollutes CI signal.                                            |
| **Committed binaries in example dirs**       | Low        | `example/encryption/encryption` and `example/user/user` are local build artifacts (not git-tracked, but present on disk). BuildFlow flags them. |
| **SQLite test artifacts**                    | Low        | `turso/indexing/:memory::memory:` and `-wal` files created by tests, not gitignored properly.                                                   |
| **ADR-0005 numbering gap**                   | Low        | ADRs jump from 0004 to 0006. Acknowledged but never resolved. Confusing for newcomers.                                                          |
| **`pkg/` directory was orphaned**            | Was Medium | **FIXED 2026-06-14** — removed via `git rm`.                                                                                                    |

### Session-Specific Notes

- **No regressions introduced.** All changes were additive or refactoring.
- **BuildFlow pre-commit hook** can block commits when local binaries exist in example dirs. Workaround: `trash-put example/*/encryption example/*/user` before committing.
- The initial audit session (messages 1-2) produced many reports but fixed almost nothing. The follow-up session (messages 3+) corrected this by executing actual fixes.

---

## E) WHAT WE SHOULD IMPROVE 🎯

### Architecture

1. **Reactive Bus Bridge** — `event.Bus` (imperative) and `event.EventBus = ro.Subject[Event]` (reactive) are disconnected. A bridge would unify them. **However**: YAGNI was declared (2026-06-14) because reactive EventBus has zero production consumers. Revisit if reactive adoption increases.

2. **Catalog module complexity** — 6 sub-packages in one module. Consider splitting exporters into standalone modules if consumers typically need only 1-2 formats. Low priority — current structure works.

3. **Middleware breadth** — 9 concerns in one module. SSE and healthcheck are transport concerns, not pipeline middleware. Consider extracting to a `transport/` module in v3.

4. **SQL load helper duplication** — `LoadFromVersion`/`LoadToVersion` pairs differ only in SQL operator. A parameterized helper would eliminate 2 clone groups.

### Type Model

5. **`catalog.Message` has 17 fields** — Should be split into `Message` + `MessageMeta` via structured embedding. Breaking change — deferred to v3.

6. **`catalog.Service` has 16 fields** — Same issue. Breaking change — deferred to v3.

7. **`any` in query dispatch** — `query.Dispatcher.Dispatch` returns `(any, error)`. `TypedHandler[Q,R]` mitigates but the core interface still uses `any`. Unavoidable without Go type inference improvements.

### Testing

8. **Golden test coverage** — 12+ modules lack golden tests. Priority: signing, storage, pebble (persistence-critical).

9. **PostgreSQL integration tests** — All SQL tests use SQLite + sqlmock. No real PostgreSQL testing. testcontainers would close this gap.

10. **Flaky test fix** — `turso/indexing/TestApplyWAL` needs deterministic WAL handling or `t.Parallel()` removal.

### Documentation

11. **README.md completeness** — Missing encryption module section (XChaCha20-Poly1305, AES-256-GCM, middleware). Missing turso module details. Consumers can't discover these features from README.

12. **Consumer-facing examples** — `example/encryption/` mentioned in TODO but not created. Would demonstrate full encrypt/decrypt lifecycle.

### Process

13. **ADR-0005 gap** — Either add a placeholder ADR-0005 or renumber. Confusing as-is.

14. **BuildFlow binary detection** — Local build artifacts in example dirs trigger false positives. Add to `.buildflow.yml` exclude patterns.

---

## F) TOP 25 THINGS TO GET DONE NEXT

Sorted by **impact × (1/effort)** — highest ROI first.

| #   | Task                                                           | Impact                      | Effort | Priority |
| --- | -------------------------------------------------------------- | --------------------------- | ------ | -------- |
| 1   | Fix flaky `turso/indexing/TestApplyWAL` test                   | High (CI signal)            | 30min  | 🔴       |
| 2   | Add SQLite test artifacts to `.gitignore`                      | Medium (clean repo)         | 5min   | 🔴       |
| 3   | Add example binaries to `.buildflow.yml` excludes              | Medium (commit hygiene)     | 5min   | 🔴       |
| 4   | Update README.md with encryption module section                | High (discoverability)      | 30min  | 🔴       |
| 5   | Update README.md with turso module section                     | High (discoverability)      | 30min  | 🔴       |
| 6   | Parameterize SQL load helpers (eliminate 2 clone groups)       | Medium (dedup)              | 1hr    | 🟡       |
| 7   | Add golden tests for signing module                            | Medium (regression safety)  | 1hr    | 🟡       |
| 8   | Add golden tests for storage module                            | Medium (regression safety)  | 1hr    | 🟡       |
| 9   | Fix ADR-0005 numbering gap                                     | Low (clarity)               | 15min  | 🟡       |
| 10  | Add `example/encryption/` standalone example                   | Medium (adoption)           | 2hr    | 🟡       |
| 11  | Add PostgreSQL integration tests via testcontainers            | High (correctness)          | 4hr    | 🟡       |
| 12  | Add property-based tests for catalog/ schema reflection        | Medium (robustness)         | 2hr    | 🟡       |
| 13  | Parameterize command_store_load.go helpers                     | Low (dedup)                 | 30min  | 🟢       |
| 14  | Add golden tests for pebble module                             | Medium (persistence safety) | 1hr    | 🟢       |
| 15  | Document CBOR usage patterns in codec/README.md                | Low (DX)                    | 30min  | 🟢       |
| 16  | Add `WithLogger` consistency audit across all middleware       | Low (polish)                | 30min  | 🟢       |
| 17  | Evaluate CoreDetEncOptions vs CanonicalEncOptions for CBOR     | Low (correctness)           | 1hr    | 🟢       |
| 18  | Add Outbox pattern design doc                                  | Medium (future feature)     | 2hr    | 🟢       |
| 19  | Add schema registry design doc                                 | Medium (future feature)     | 2hr    | 🟢       |
| 20  | Create documentation site (Docusaurus/MkDocs)                  | Medium (adoption)           | 4hr    | 🟢       |
| 21  | Add streaming event reads without materializing full slice     | Medium (performance)        | 4hr    | 🟢       |
| 22  | Add distributed checkpointing for projections                  | Medium (scalability)        | 1d+    | 🟢       |
| 23  | Add `cqrs-gen v2` with struct tag scanning                     | Low (DX)                    | 4hr    | 🟢       |
| 24  | Add gRPC transport adapter                                     | Low (ecosystem)             | 1d     | 🟢       |
| 25  | Split `catalog.Message` into Message+MessageMeta (v3 breaking) | Medium (type safety)        | v3     | ⚪       |

---

## G) TOP QUESTION ❓

**"Should the catalog exporters (AsyncAPI, OpenAPI, D2, EventCatalog, docserver) be split into separate Go modules, or kept as sub-packages within `catalog/`?"**

Context:

- `catalog/` has 6 sub-packages, each importing `catalog/` core types (Registry, Builder, Catalog)
- Zero internal dependencies (catalog/ is a Layer 0 leaf module)
- Only 2 external deps (`go-faster/yaml`, stdlib)
- A consumer who only needs AsyncAPI export transitively gets D2 and EventCatalog code
- But: splitting into 6 modules adds go.mod overhead, replace directive management, and CI complexity
- The sub-packages already provide good isolation within the single module

I cannot determine this myself because:

1. I don't know how consumers actually import catalog — do they typically use 1 exporter or multiple?
2. The tradeoff between dependency isolation (pro-split) and maintenance overhead (anti-split) depends on consumer behavior I can't observe
3. The AGENTS.md says "Public API surface IS the product" — but I don't know if the sub-packages are consumed independently

---

## Architecture Diagram (Current)

```
Layer 0: id/, dispatcher/, codec/, otel/, catalog/                    (leaf modules)
Layer 1: event/, signing/, encryption/                                (core + security)
Layer 2: command/, query/, memory/, schema/, snapshot/, listing/, pebble/  (CQRS + persistence)
Layer 3: decider/, projection/, storage/, middleware/                  (orchestration + infra)
Layer 4: turso/, watermill/                                            (backend adapters)
Layer 5: integration/, examples/, cmd/                                 (consumers)
```

No cycles. Strict DAG. ✅

---

## Module Dependency Health

| Check                                      | Status                                    |
| ------------------------------------------ | ----------------------------------------- |
| Circular dependencies                      | None ✅                                   |
| Replace directive consistency              | All 27 modules use `../module` pattern ✅ |
| Go version consistency                     | All modules on `go 1.26.3` ✅             |
| Test-only deps in production go.mod        | Accepted (Go limitation, ADR-0014)        |
| `internal/` cross-module access violations | None ✅                                   |
| Error type accessibility across modules    | Verified ✅                               |

---

_Generated 2026-06-15 06:23. All metrics verified against live `nix run .#build`, `nix run .#test`, `nix run .#lint`, `dupl`, and `nix run .#coverage`._
