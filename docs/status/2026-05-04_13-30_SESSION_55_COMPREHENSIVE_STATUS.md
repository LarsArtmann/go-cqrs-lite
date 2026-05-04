# Session 55 — Comprehensive Status Report

**Date:** 2026-05-04 15:30 (reporting as 2026-05-04 13:30)
**Branch:** master
**Commits since May 1:** 118+

---

## a) Fully Done

### Sessions 51–54 (since last status report)

| Session | Commits | Key Changes                                                                                                                                              |
| ------- | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 51      | 5       | EveryNEvents no-panic, sentinel error audit (38→42 sentinels), decider error registration, errors.go split                                               |
| 52      | 4       | mustNewCatalogEvent rename, interface check for everyN, strconv.Itoa, outbox Ack batch chunking (500 max), snapshot extraction, godoc for asyncapi+types |
| 53      | 3       | Godoc for d2+adapters (14 symbols), reconstructEvent deduplication in storage, coverage updates                                                          |
| 54      | 5       | **TypedHandler[T]** for query, **cockroachdb/errors → stdlib**, **go-json-experiment/json → encoding/json**, middleware sentinel errors, gci formatting  |

### Cumulative Quality Metrics

| Metric                     | Value                                       |
| -------------------------- | ------------------------------------------- |
| Test packages              | 22 (all pass)                               |
| Production LOC             | 10,330                                      |
| Test LOC                   | 22,444                                      |
| Total coverage             | 84.8%                                       |
| Benchmarks                 | 43                                          |
| Sentinel errors            | 42 (all classified via Classify())          |
| Registered classifications | 26 across 7 packages                        |
| Interface checks           | 30 compile-time `var _` checks              |
| Lint issues                | 0                                           |
| TODO/FIXME in production   | 0                                           |
| Files >250 lines           | 0 (max is exactly 250)                      |
| Nolint directives (prod)   | 51 (all legitimate)                         |
| Dependencies removed       | cockroachdb/errors, go-json-experiment/json |

### Per-Package Coverage

| Package              | Coverage |
| -------------------- | -------- |
| core/command         | 100.0%   |
| core/query           | 100.0%   |
| core/pkg/dispatcher  | 100.0%   |
| core/pkg/id          | 100.0%   |
| middleware           | 100.0%   |
| memory               | 99.1%    |
| catalog/d2           | 97.6%    |
| catalog/asyncapi     | 95.9%    |
| catalog/eventcatalog | 95.6%    |
| catalog/adapters     | 95.5%    |
| core/aggregate       | 95.5%    |
| core/decider         | 95.0%    |
| storage              | 94.8%    |
| core/event           | 94.4%    |
| catalog              | 94.4%    |
| projection           | 92.5%    |

### Completed Infrastructure

- **No-panic convention**: All constructors return errors. Only `Must*` helpers panic.
- **Error taxonomy**: 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure) with extensible `RegisterClassification`.
- **ISP**: `event.Publisher` and `event.Subscriber` sub-interfaces.
- **Shared helpers**: `PublishChanges()`, `SaveSnapshot()`, `ShouldSnapshot()`, `reconstructEvent()`.
- **TypedHandler[T]**: Type-safe query handlers eliminate `any` at call sites.
- **Dependency migration**: Removed cockroachdb/errors (-169 lines from go.sum) and go-json-experiment/json. Now only external deps are `oklog/ulid`, `go-branded-id`, `go-faster/yaml`.

---

## b) Partially Done

| Item                          | Status     | Detail                                                                                                                                                 |
| ----------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `query.Handler` returns `any` | Mitigated  | `TypedHandler[T]` + `RegisterTyped[T]` + `DispatchTyped[T]` provide escape hatches. Core `Handler` type still uses `any` for middleware compatibility. |
| `CatalogMeta` duplication     | Identified | `event.CatalogMeta`, `command.CatalogMeta`, `query.CatalogMeta` — nearly identical structs. Low priority.                                              |
| `opError` duplication         | Identified | Different signatures in aggregate vs decider. 4 lines each. Acceptable.                                                                                |

---

## c) Not Started

| #   | Item                                      | Priority | Effort |
| --- | ----------------------------------------- | -------- | ------ |
| 1   | Transactional outbox (atomic save+outbox) | HIGH     | 8h     |
| 2   | Saga/process manager implementation       | MEDIUM   | 18h    |
| 3   | PostgreSQL integration tests for storage  | MEDIUM   | 4h     |
| 4   | `CatalogMeta` consolidation               | LOW      | 2h     |
| 5   | Projection replay→live gap fix            | MEDIUM   | 6h     |
| 6   | Memory concurrent publish benchmark       | LOW      | 1h     |
| 7   | Storage `insertEvents` bulk INSERT        | LOW      | 2h     |
| 8   | CONTRIBUTING.md                           | LOW      | 2h     |
| 9   | Tag `v0.1.0-alpha`                        | LOW      | 0.5h   |
| 10  | CI race detector + coverage threshold     | LOW      | 1h     |

---

## d) Totally Fucked Up

Nothing is fundamentally broken. One known flaky test:

- **`TestRunner_WildcardProjection`** — timing-sensitive, flakes ~5% under load. Uses channel-based sync but projection runner's live subscription path has inherent timing. Not a correctness issue, just test reliability.

**Known architectural limitation:**

- **Projection replay→live gap**: Events published between `replay()` finishing and `subscribeLive()` starting are missed. Requires hybrid subscribe-then-replay approach. Documented, not a bug for current test-utility scope.

---

## e) What We Should Improve

1. **Dependency diet is complete** — stdlib errors + encoding/json is the right call for a library. Only 3 external deps remain.
2. **TypedHandler[T] was the highest-value API improvement** — eliminates the biggest `any` in the codebase at the call site level.
3. **Next highest-value feature: Transactional outbox** — Without atomic save+outbox, consumers cannot guarantee at-least-once publishing. This is the #1 gap for production use.
4. **Projection replay→live gap** is the #2 gap — Any consumer using projections in production will hit this.
5. **Storage bulk INSERT** — `insertEvents` does one `ExecContext` per event inside a transaction. For aggregates with many events, this is O(n) round-trips. A single `unnest()` or multi-value INSERT would be O(1).

---

## f) Top 25 Next Items

| #   | Item                                                    | Impact | Effort | Category    |
| --- | ------------------------------------------------------- | ------ | ------ | ----------- |
| 1   | Transactional outbox (`TransactionalStore`)             | HIGH   | 8h     | Feature     |
| 2   | Projection hybrid replay (subscribe-then-replay)        | HIGH   | 6h     | Correctness |
| 3   | PostgreSQL integration tests                            | HIGH   | 4h     | Quality     |
| 4   | Storage bulk INSERT (`unnest()` or multi-value)         | MEDIUM | 2h     | Performance |
| 5   | Memory concurrent publish benchmark                     | MEDIUM | 1h     | Performance |
| 6   | Saga/process manager implementation                     | MEDIUM | 18h    | Feature     |
| 7   | `CatalogMeta` consolidation across 3 packages           | LOW    | 2h     | Cleanup     |
| 8   | CI race detector + coverage threshold gate              | LOW    | 1h     | CI          |
| 9   | CONTRIBUTING.md with architecture guidelines            | LOW    | 2h     | Docs        |
| 10  | Tag `v0.1.0-alpha` first public release                 | LOW    | 0.5h   | Release     |
| 11  | Example/user integration test                           | LOW    | 2h     | Quality     |
| 12  | `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch    | LOW    | 1h     | Cleanup     |
| 13  | `CHANGELOG.md` update for v0.2.0                        | LOW    | 1h     | Docs        |
| 14  | Event upcaster integration test                         | LOW    | 1h     | Quality     |
| 15  | `event.Bus.Use()` middleware chain test                 | LOW    | 0.5h   | Quality     |
| 16  | API stability audit for pre-release                     | MEDIUM | 2h     | Quality     |
| 17  | Performance regression CI check                         | LOW    | 2h     | CI          |
| 18  | `projection.Runner.WithRetry` default config validation | LOW    | 0.5h   | Fix         |
| 19  | Connection pool metrics for storage                     | LOW    | 2h     | Feature     |
| 20  | Go doc audit for newly added symbols                    | LOW    | 1h     | Docs        |
| 21  | `decider` example in README                             | LOW    | 1h     | Docs        |
| 22  | Catalog AsyncAPI 3.0 schema validation                  | LOW    | 1h     | Quality     |
| 23  | `MemoryBus.Publish` RLock scope reduction               | LOW    | 2h     | Performance |
| 24  | Flaky test fix: `TestRunner_WildcardProjection`         | LOW    | 1h     | Fix         |
| 25  | Offline-first metadata conventions doc                  | LOW    | 1h     | Docs        |

---

## g) Top #1 Question

**Should we pursue the transactional outbox as a new `storage` feature, or as a cross-cutting `core/event` abstraction?**

The design doc at `docs/planning/OUTBOX_TRANSACTION_API.md` proposes a `TransactionalStore` interface that wraps `event.Store` + `event.Outbox` behind a single transaction. The question is:

- **Option A**: `storage` package adds `NewTransactionalEventStore(db *sql.DB)` that does atomic `INSERT events + INSERT outbox` in a single `sql.Tx`. Tightly coupled to PostgreSQL, but simple and correct.
- **Option B**: `core/event` adds a `TransactionalStore` interface with `BeginTx()` → `SaveInTx()` → `Commit()`. More abstract, works across any SQL backend, but adds complexity to the core API surface.

**My recommendation**: Option A in `storage`. It's PostgreSQL-specific by nature, keeps `core/event` transport-agnostic, and consumers who don't need transactional outbox don't pay the complexity tax.

---

## Dependency Profile (After Migration)

| Dependency     | Version | Purpose                  | Used By |
| -------------- | ------- | ------------------------ | ------- |
| oklog/ulid/v2  | v2.1.0  | ULID generation          | core    |
| go-branded-id  | v0.1.0  | Branded ID type backing  | core    |
| go-faster/yaml | v0.4.6  | YAML marshaling          | catalog |
| onsi/ginkgo/v2 | v2.28.1 | BDD testing (test-only)  | core    |
| onsi/gomega    | v1.39.1 | BDD matchers (test-only) | core    |
| go-sqlmock     | v1.5.2  | SQL mocking (test-only)  | storage |

Zero production dependencies on cockroachdb/errors or go-json-experiment/json.
