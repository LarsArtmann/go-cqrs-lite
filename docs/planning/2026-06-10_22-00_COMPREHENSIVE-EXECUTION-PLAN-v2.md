# Comprehensive Execution Plan v2 — go-cqrs-lite

**Date:** 2026-06-10
**Scope:** ALL remaining open TODOs across TODO_LIST.md, ROADMAP.md, session context, code quality scan, and planning docs.
**Task Size:** Max 12 minutes each.
**Sorting:** Impact × (1/Effort) × Customer Value — highest first.

---

## Summary Statistics

| Tier                          | #    | Tasks | Total Est. | Avg Impact    |
| ----------------------------- | ---- | ----- | ---------- | ------------- |
| **A: Type Safety & API**      | A1–A | 5     | ~33min     | HIGH          |
| **B: Test Coverage**          | B1–B | 14    | ~156min    | HIGH          |
| **C: Code Quality Polish**    | C1–C | 5     | ~32min     | MED           |
| **D: CI & DevEx**             | D1–D | 3     | ~36min     | MED           |
| **E: Developer Docs**         | E1–E | 5     | ~44min     | MED           |
| **F: Experiments**            | F1–F | 2     | ~24min     | LOW           |
| **X: [v2] Breaking Changes**  | X1–X | 5     | ~12hr      | HIGH (defer)  |
| **Y: [BLOCKED]**              | Y1–Y | 11    | N/A        | N/A           |
| **Z: [FUTURE]**               | Z1–Z | 22    | N/A        | N/A           |
| **ACTIONABLE TOTAL**          |      | **49**| **~325min**|               |

---

## Tier A: Type Safety & API Consistency (HIGH impact, LOW effort)

*Why first: API surface IS the product. Inconsistency erodes consumer trust immediately.*

| #   | Task                                                      | Module       | Impact | Effort | Customer Value                                                                    |
| --- | --------------------------------------------------------- | ------------ | ------ | ------ | --------------------------------------------------------------------------------- |
| A1  | Add `SchemaVersion.Add(n int) SchemaVersion`              | event        | HIGH   | 4min   | `Version` has `Add(n)` but `SchemaVersion` only has `Increment()` — API gap      |
| A2  | Replace `SchemaVersion.Cmp` manual comparison → `cmp.Compare` | event   | HIGH   | 3min   | `Version.Cmp` uses `cmp.Compare`, `SchemaVersion.Cmp` has manual if/else — split brain |
| A3  | Add `Version.MarshalJSON` / `Version.UnmarshalJSON`       | event        | HIGH   | 10min  | Explicit JSON control: `{"version":1}` not `1`, prevents accidental float parse   |
| A4  | Add `SchemaVersion.MarshalJSON` / `SchemaVersion.UnmarshalJSON` | event | MED    | 6min   | Same as A3, for schema version — consistency with Version                         |
| A5  | Remove `omitempty` from `pebble.Metadata` struct tag (gopls: no effect on struct types) | pebble | LOW | 3min | Code cleanliness — `omitempty` is misleading on struct type, currently suppressed with nolint |

---

## Tier B: Test Coverage (HIGH impact, MEDIUM effort)

*Why second: Untested code is untrusted code. Consumers check coverage before importing.*

### B1–B5: storage/sql coverage (37.4% → target 70%+)

| #   | Task                                                        | Target Function(s)              | Impact | Effort |
| --- | ----------------------------------------------------------- | ------------------------------- | ------ | ------ |
| B1  | Add sqlmock tests: `SharedInsertEvents`                     | `sql/helpers.go:49`             | HIGH   | 12min  |
| B2  | Add sqlmock tests: `SharedCheckVersion`                     | `sql/helpers.go:91`             | HIGH   | 12min  |
| B3  | Add sqlmock tests: `SharedCheckpointLoad` + `SharedCheckpointSave` | `sql/helpers.go:117,156` | HIGH   | 12min  |
| B4  | Add sqlmock tests: `ScanSlice` + `ReconstructEvent` + `UnmarshalEventMetadata` | `sql/reconstruction.go` | HIGH | 12min |
| B5  | Add sqlmock tests: `CommitTx` + `MarshalMetadata` + `DeleteByAggregate` | `sql/reconstruction.go, helpers.go` | MED | 12min |

### B6–B8: otel coverage (73.0% → target 85%+)

| #   | Task                                                        | Target Function(s)              | Impact | Effort |
| --- | ----------------------------------------------------------- | ------------------------------- | ------ | ------ |
| B6  | Add tests: `NewMeter` (0% → covered)                        | `otel/meter.go:15`             | MED    | 10min  |
| B7  | Add tests: `WithAttributes`, `WithSpanKind`, `AttrString/Int/Int64` | `otel/types.go:32-56` | LOW    | 12min  |
| B8  | Add tests: `MetricWithAttributes/Description/Unit`, `SpanFromContext` | `otel/types.go:60-64` | LOW | 12min  |

### B9–B11: turso coverage (28.6% → target 50%+)

| #   | Task                                                        | Target Function(s)              | Impact | Effort |
| --- | ----------------------------------------------------------- | ------------------------------- | ------ | ------ |
| B9  | Add tests: `Open` + `OpenInMemory` connector paths          | `turso/connector.go`            | MED    | 12min  |
| B10 | Add tests: `NewEventStore` + store lifecycle (open/close)   | `turso/store.go`                | MED    | 12min  |
| B11 | Add tests: `Save` + `Load` + `LoadFromVersion` happy paths  | `turso/store.go`                | MED    | 12min  |

### B12–B14: go-snaps snapshot tests (12 remaining modules)

| #   | Task                                                        | Module(s)                       | Impact | Effort |
| --- | ----------------------------------------------------------- | ------------------------------- | ------ | ------ |
| B12 | Add snapshot tests: `signing` (HMAC/Ed25519 signatures, middleware output) | signing     | MED    | 12min  |
| B13 | Add snapshot tests: `middleware` (SSE frames, circuit breaker states, retry timing) | middleware | MED | 12min  |
| B14 | Add snapshot tests: `storage` (DDL schemas per dialect, metadata roundtrip JSON) | storage | MED | 12min  |

---

## Tier C: Code Quality Polish (MED impact, LOW effort)

| #   | Task                                                      | Module       | Impact | Effort | Customer Value                                                        |
| --- | --------------------------------------------------------- | ------------ | ------ | ------ | --------------------------------------------------------------------- |
| C1  | Add `SchemaVersion.Add` + `Cmp` tests                     | event        | MED    | 8min   | New methods need test coverage                                        |
| C2  | Add `Version.MarshalJSON` / `UnmarshalJSON` tests         | event        | MED    | 8min   | JSON serialization round-trip verification                            |
| C3  | Add `SchemaVersion.MarshalJSON` / `UnmarshalJSON` tests   | event        | MED    | 6min   | Same as C2 for schema version                                        |
| C4  | Audit `otel/types.go` — 9 re-export functions at 0% coverage, consider if they're needed or just passthrough wrappers | otel | LOW | 5min | Dead passthrough wrappers inflate the module without adding value |
| C5  | Verify `event.TypeOf[T]()` design — catalog has 80% of machinery, but naming convention is unresolved (struct name vs dot-notation) | event, catalog | MED | 5min | Design decision blocks implementation                               |

---

## Tier D: CI & DevEx (MED impact, MED effort)

| #   | Task                                                      | Module       | Impact | Effort | Customer Value                                                     |
| --- | --------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------ |
| D1  | Add Docker build CI step: linux/amd64 + linux/arm64       | CI           | MED    | 12min  | Multi-arch Docker builds are broken without CI verification        |
| D2  | Add `nolint` justification audit — all `//nolint` comments should have `// reason` | all    | LOW    | 12min  | Blind suppressions hide real bugs                                  |
| D3  | Add per-module `go vet` CI step (separate from lint)      | CI           | LOW    | 12min  | Defense in depth — `go vet` catches issues golangci-lint doesn't   |

---

## Tier E: Developer Documentation (MED impact, MED effort)

*Why: pkg.go.dev is the primary consumer touchpoint. Examples drive adoption.*

| #   | Task                                                      | Module       | Impact | Effort | Customer Value                                                     |
| --- | --------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------ |
| E1  | Add godoc examples: `decider` — Execute, Load, Repository patterns | decider | MED  | 10min  | Most complex module, no runnable examples on pkg.go.dev            |
| E2  | Add godoc examples: `projection` — Runner, Builder, On[T]() | projection | MED    | 10min  | Complex replay+live API, hardest to learn without examples         |
| E3  | Add godoc examples: `signing` — HMAC + Ed25519 setup, middleware | signing  | MED    | 8min   | Security-critical, easy to misconfigure without examples           |
| E4  | Add godoc examples: `schema` — Upcaster, VersionedStore usage | schema    | MED    | 8min   | Schema evolution is a hard topic, examples reduce friction         |
| E5  | Add godoc examples: `listing` — List, StatusMiddleware, InMemoryAggregateReader | listing | LOW | 8min  | Newest module, no usage examples yet                               |

---

## Tier F: Experiments (LOW impact, speculative)

| #   | Task                                                      | Module       | Impact | Effort | Customer Value                                                     |
| --- | --------------------------------------------------------- | ------------ | ------ | ------ | ------------------------------------------------------------------ |
| F1  | `jsonv2` codec experiment behind build tag                | codec        | LOW    | 12min  | Potential performance win, behind experimental tag                  |
| F2  | Arena allocation experiment in event creation             | event        | LOW    | 12min  | Go 1.26+ feature, potential alloc reduction for hot paths           |

---

## Tier X: [v2] Breaking Changes — DEFERRED

*These are high impact but require major version bump and migration guide.*

| #   | Task                                                      | Effort   | Note                            |
| --- | --------------------------------------------------------- | -------- | ------------------------------- |
| X1  | Add global `TransactionID` branded type                   | 60min    | ADR needed first                |
| X2  | Remove `io.Closer` from core interfaces                  | 4hr      | ADR-0010 exists                 |
| X3  | Split `event.Store` into Writer/Reader/Deleter           | 3hr      | Breaking change                 |
| X4  | Make event Core truly immutable                          | 2hr      | Breaking change                 |
| X5  | Move HTTP code out of middleware → `transport/` module   | 2hr      | SSE/healthcheck/metrics_http    |

---

## Tier Y: [BLOCKED] — External action required

| #   | Blocker                                    | What's needed              |
| --- | ------------------------------------------ | -------------------------- |
| Y1  | Move `example/todo` to own repository      | Manual repo creation       |
| Y2  | Add PostgreSQL integration tests (testcontainers) | Docker setup        |
| Y3  | Remove cockroachdb/errors from go-localsync | Different repo            |
| Y4  | Create go-branded-id v0.2.0                | Different repo             |
| Y5  | Design ActaFlow event sourcing overlay     | Different project          |
| Y6  | Extract shared golangci.yml into larsartmann/library-policy | Different repo |
| Y7  | Change LICENSE from proprietary → MIT/Apache-2.0 | Owner decision       |
| Y8  | Migrate ActaFlow build to flake.nix        | Different project          |
| Y9  | Integrate TypeSpec types → catalog.Registry | Different project         |
| Y10 | Playwright setup + E2E tests (4 items)     | Infrastructure setup       |
| Y11 | Push signing v1.0.0 tag                    | Manual tag + push          |

---

## Tier Z: [FUTURE] — Speculative, no design yet

| #   | Item                                                       | Source               |
| --- | ---------------------------------------------------------- | -------------------- |
| Z1  | Add catalog diff / breaking-change detection tool          | SESSION_04           |
| Z2  | Add high-level test utilities: AggregateTester, ProjectionTester, BusTester fluent API | MONOREPO_PLAN |
| Z3  | Add bi-temporal support: ValidAt, WithValidAt, LoadToValidTime | TIME_TRAVEL     |
| Z4  | Add HLC (Hybrid Logical Clock) implementation              | OFFLINE_FIRST         |
| Z5  | Implement pull-before-push sync protocol                   | OFFLINE_FIRST         |
| Z6  | Implement rebase mechanism                                 | OFFLINE_FIRST         |
| Z7  | Build network simulator for testing                        | OFFLINE_FIRST         |
| Z8  | Build multi-client test harness                            | OFFLINE_FIRST         |
| Z9  | Build thin PostgreSQL store adapter (no Watermill)         | WATERMILL_PRO_CONTRA  |
| Z10 | Build thin NATS bus adapter (no Watermill)                 | WATERMILL_PRO_CONTRA  |
| Z11 | Add Filter, Predicate types for context queries            | HYBRID_ARCHITECTURE   |
| Z12 | Add ContextQuerier, ContextAppender, QueryResult interfaces | HYBRID_ARCHITECTURE  |
| Z13 | Make transactional projection contract explicit            | LIVESTORE_DEEP_DIVE   |
| Z14 | Add multi-engine storage support via sqlc                  | MONOREPO_PLAN         |
| Z15 | Add schema migration tool                                  | MONOREPO_PLAN         |
| Z16 | Add hybrid service example                                 | HYBRID_ARCHITECTURE   |
| Z17 | Add distributed consensus capability (Raft/CRDT overlay)   | COMPARISON_REPORT     |
| Z18 | Add time-series event query language                       | COMPARISON_REPORT     |
| Z19 | Create documentation site (Docusaurus/MkDocs/Hugo)         | Multiple sessions     |
| Z20 | Set up pkg.go.dev documentation hosting                    | SESSION_57            |
| Z21 | Add ServerReceivedAt / ServerStoredAt server-side timestamps | OFFLINE_FIRST       |
| Z22 | Absorb projection/ into core/event                         | SESSION_77            |

---

## Execution Order

```
Phase 1 (33min):  A1 → A2 → A3 → A4 → A5        — Type safety & API consistency
Phase 2 (60min):  B1 → B2 → B3 → B4 → B5        — storage/sql coverage
Phase 3 (46min):  B6 → B7 → B8                   — otel coverage
Phase 4 (36min):  B9 → B10 → B11                 — turso coverage
Phase 5 (22min):  C1 → C2 → C3                   — New method tests
Phase 6 (36min):  B12 → B13 → B14                — go-snaps snapshot tests
Phase 7 (10min):  C4 → C5                        — Design decisions
Phase 8 (36min):  D1 → D2 → D3                   — CI & DevEx
Phase 9 (44min):  E1 → E2 → E3 → E4 → E5        — Developer docs
Phase 10 (24min): F1 → F2                        — Experiments
```

**Total actionable: ~347 min (~5.8 hr)**
**Deferred: X1–X5 (~12 hr), Y1–Y11 (blocked), Z1–Z22 (speculative)**
