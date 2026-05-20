# Session 59 — Comprehensive Status: Zero Lint in 6/7 Modules

**Date:** 2026-05-07 · **Time:** 08:32 UTC
**Branch:** master · **Ahead of origin:** 5 commits
**Working tree:** Dirty (3 storage lint auto-fixes from pre-commit hook)

---

## Executive Summary

| Metric               | Value                             | Status                                                        |
| -------------------- | --------------------------------- | ------------------------------------------------------------- |
| **Tests**            | 22/22 pass                        | ✅                                                            |
| **Race detector**    | 20/20 pass                        | ✅                                                            |
| **Lint (6 modules)** | 0 issues                          | ✅ core, memory, catalog, middleware, integration, projection |
| **Lint (storage)**   | 66 issues                         | ⚠️ Pre-existing from Pebble addition (was 73)                 |
| **Total coverage**   | 84.5%                             | ⚠️ Excludes testhelpers (always 0%)                           |
| **Total LOC**        | ~36,638                           | —                                                             |
| **Test functions**   | ~805 across 88 files              | —                                                             |
| **Uncommitted**      | 4 files (storage lint auto-fixes) | Needs commit                                                  |

---

## A) FULLY DONE ✅

### Session 58-59 Work (This Conversation)

| #   | Task                                                                  | Files                                | Result        |
| --- | --------------------------------------------------------------------- | ------------------------------------ | ------------- |
| 1   | Fix `modernize`: backward loops → `slices.Backward()`                 | `dispatcher.go`, `bus.go`            | ✅ 0 issues   |
| 2   | Fix `wsl_v5`: whitespace before `close(p.done)`                       | `outbox_publisher.go`                | ✅            |
| 3   | Fix `nestif` (complexity 5→3): extract `persistDirect()`              | `aggregate/repository.go`            | ✅            |
| 4   | Fix `noinlineerr`: inline error → separate assignment                 | `aggregate/repository.go`            | ✅            |
| 5   | Fix `gochecknoinits`: add nolint directive                            | `errors_taxonomy.go`                 | ✅            |
| 6   | Fix `exhaustive`: add `CommandMessage` case to `kindToTagName`        | `asyncapi/exporter.go`               | ✅            |
| 7   | Fix `goconst`: extract `asyncAPIVersion`, `contentType`, `typeObject` | `asyncapi/exporter.go`, `helpers.go` | ✅            |
| 8   | Fix `goconst`: extract `testVersion` constant                         | `cattest/catalog.go`                 | ✅            |
| 9   | Fix `nonamedreturns` + `prealloc`: refactor `collectMessageIDs`       | `eventcatalog/exporter.go`           | ✅            |
| 10  | Add 9 Pebble edge-case tests (error paths, empty, nil)                | `pebble_event_store_test.go`         | ✅            |
| 11  | Add 4 aggregate coverage tests (outbox+tx, load error, apply error)   | `aggregate/repository_test.go`       | ✅            |
| 12  | Remove dead `sqliteUnmarshalEventMetadata`, fix `noctx`               | `sqlite_helpers.go`                  | ✅            |
| 13  | Migrate `gomodguard` → `gomodguard_v2`                                | `.golangci.yml`                      | ✅            |
| 14  | Auto-fix: depguard allowlist for pebble/sqlite/turso                  | `.golangci.yml`                      | ✅ Pre-commit |
| 15  | Auto-fix: `embeddedstructfieldcheck` blank lines                      | 3 storage files                      | ✅ Pre-commit |
| 16  | Auto-fix: `revive` unused-parameter nolint                            | 2 storage files                      | ✅ Pre-commit |
| 17  | All golden test files updated                                         | 3 catalog golden files               | ✅            |
| 18  | Comprehensive status reports written                                  | 2 status reports                     | ✅            |

### Coverage Improvements This Session

| Module               | Before | After      | Delta                                         |
| -------------------- | ------ | ---------- | --------------------------------------------- |
| core/aggregate       | 92.1%  | **96.9%**  | **+4.8%**                                     |
| storage (total)      | 83.8%  | **85.4%**  | **+1.6%**                                     |
| Pebble Save          | 77.8%  | **94.4%**  | **+16.6%**                                    |
| Pebble Close         | 66.7%  | **100.0%** | **+33.3%**                                    |
| Pebble AppendBatch   | 81.8%  | **90.9%**  | **+9.1%**                                     |
| catalog/asyncapi     | 95.8%  | **93.9%**  | **-1.9%** (constant extraction shifted lines) |
| catalog/eventcatalog | 95.6%  | **95.7%**  | **+0.1%**                                     |

### Lint Reduction This Session

| Module    | Before Session | After Session | Delta               |
| --------- | -------------- | ------------- | ------------------- |
| core      | 5              | **0**         | **-5**              |
| memory    | 1              | **0**         | **-1**              |
| catalog   | 7              | **0**         | **-7**              |
| storage   | 73             | **66**        | **-7** (auto-fixes) |
| **Total** | **86**         | **66**        | **-20**             |

---

## B) PARTIALLY DONE ⚠️

| #   | Task                           | Status                          | Blocker                                                              |
| --- | ------------------------------ | ------------------------------- | -------------------------------------------------------------------- |
| 1   | Storage lint cleanup           | 66→0 in progress                | 66 issues remain (err113: 7, tagliatelle: 9, wrapcheck: 8, etc.)     |
| 2   | Storage coverage               | 85.4%                           | Pebble error paths (iterateEvents, deserializeEvent) still uncovered |
| 3   | Core/decider coverage          | 92.7%                           | Execute/publish error paths at 85.7%                                 |
| 4   | Pre-commit hook not executable | Hook ran but reported "ignored" | Needs `chmod +x .git/hooks/pre-commit`                               |

---

## C) NOT STARTED ❌

| #   | Task                                                           | Impact | Notes                                          |
| --- | -------------------------------------------------------------- | ------ | ---------------------------------------------- |
| 1   | Storage 66 lint issues → 0                                     | HIGH   | Pebble code doesn't follow project conventions |
| 2   | Pebble `tagliatelle` (9 issues) — JSON tags use snake_case     | MEDIUM | Would break on-disk format — migration needed  |
| 3   | Pebble `err113` (7 issues) — dynamic errors                    | MEDIUM | Need sentinel error extraction                 |
| 4   | Pebble `wrapcheck` (8 issues) — unwrapped external errors      | MEDIUM | Need error wrapping                            |
| 5   | Pebble `exhaustruct` (6 issues) — missing struct fields        | LOW    | Mostly pebble.Options                          |
| 6   | Storage `dupl` (6 issues) — PostgreSQL/SQLite code duplication | MEDIUM | Inherent to dual-engine design                 |
| 7   | PostgreSQL integration tests                                   | HIGH   | All storage tests use sqlmock or SQLite        |
| 8   | Saga / Process Manager implementation                          | HIGH   | Design doc exists, no code                     |
| 9   | Watermill module (Kafka/NATS adapter)                          | MEDIUM | Research doc exists, no code                   |
| 10  | Tagged releases / versioning strategy                          | LOW    | All modules at v0.0.0                          |
| 11  | `io.Closer` removal from interfaces                            | MEDIUM | Breaking change, deferred                      |
| 12  | `CatalogMeta` consolidation across packages                    | LOW    | Circular dependency blocks shared type         |

---

## D) TOTALLY FUCKED UP 💥

**Nothing is catastrophically broken.** But here's what's concerning:

| #   | Issue                                                           | Severity  | Detail                                                                  |
| --- | --------------------------------------------------------------- | --------- | ----------------------------------------------------------------------- |
| 1   | Pebble was merged with 73 lint issues                           | 🔴 HIGH   | Bypassed quality gate. 66 remain. Every other module is at 0.           |
| 2   | Storage `dupl` — checkpoint/transactional store 95%+ duplicated | 🟠 MEDIUM | PostgreSQL ↔ SQLite code is nearly identical (only `$1` vs `?` differs) |
| 3   | Pebble `tagliatelle` — on-disk format uses `snake_case` JSON    | 🟠 MEDIUM | Changing tags would break existing persisted data. Decision needed.     |
| 4   | Pre-commit hook "not executable" warning                        | 🟡 LOW    | Hook ran but git warns. Needs `chmod +x`.                               |
| 5   | Golden test churn — 3 golden file updates this session          | 🟡 LOW    | Constants changed output. Stable now, but fragile to future refactors.  |

---

## E) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Pebble needs the same quality treatment as every other module** — it was added in session 56 with 73 lint issues. Every other module has 0. The quality gate ("Would a consumer trust this enough to import it?") fails for Pebble.
2. **Storage code duplication** — PostgreSQL and SQLite implementations are 95%+ identical. Extract shared logic or generate from a template.
3. **Missing integration tests** — All storage tests use go-sqlmock (unit) or SQLite (in-memory). No real PostgreSQL test exists.

### Architecture

4. **Pebble `CQRSAdapter` naming** — The type name doesn't follow Go conventions. Should be `PebbleEventStore` or `EventStore` (package-scoped).
5. **Pebble ignores `context.Context`** — All methods take `_ context.Context`. This blocks cancellation, tracing, timeout propagation.
6. **Dual-engine SQL pattern** — PostgreSQL/SQLite duplication is structural. Consider code generation or shared base type.

### Process

7. **Quality gate enforcement** — The Pebble merge bypassed the "0 lint" convention. Need CI gate or PR review enforcement.
8. **Pre-commit hook** — Not executable, produces warnings. Fix with `chmod +x`.
9. **Golden test fragility** — Constant extraction caused golden churn. Consider more stable test patterns.

---

## F) Top 25 Things to Get Done Next

| #   | Priority | Task                                                             | Impact | Effort    | Module                 |
| --- | -------- | ---------------------------------------------------------------- | ------ | --------- | ---------------------- |
| 1   | 🔴       | Fix Pebble `err113` (7×) — extract sentinel errors               | HIGH   | 30min     | storage                |
| 2   | 🔴       | Fix Pebble `wrapcheck` (8×) — wrap external errors               | HIGH   | 20min     | storage                |
| 3   | 🔴       | Fix Pebble `noinlineerr` (4×) — separate error assignments       | MED    | 15min     | storage                |
| 4   | 🔴       | Fix Pebble `nestif` (1×) — reduce complexity in deserializeEvent | MED    | 10min     | storage                |
| 5   | 🔴       | Fix Pebble `godoclint` (2×) — consistent package doc             | LOW    | 5min      | storage                |
| 6   | 🟠       | Decide on Pebble `tagliatelle` (9×) — snake_case vs camelCase    | HIGH   | Design    | storage                |
| 7   | 🟠       | Fix Pebble `exhaustruct` (6×) — add nolint for pebble.Options    | LOW    | 10min     | storage                |
| 8   | 🟠       | Fix Pebble `varnamelen` (8×) — rename `db` to longer name        | LOW    | 10min     | storage                |
| 9   | 🟠       | Fix Pebble `noctx` (5×) — use ExecContext/PingContext            | MED    | 15min     | storage                |
| 10  | 🟠       | Fix Pebble `errcheck` (4×) — check Close/Publish errors          | MED    | 10min     | storage                |
| 11  | 🟠       | Fix Pebble `nilnil` (1×) — return sentinel instead of nil,nil    | LOW    | 5min      | storage                |
| 12  | 🟠       | Fix Pebble `unconvert` (1×) — remove unnecessary conversion      | LOW    | 2min      | storage                |
| 13  | 🟠       | Fix Pebble `gosec` (1×) — SQL formatting                         | LOW    | 5min      | storage                |
| 14  | 🟠       | Address storage `dupl` (6×) — extract shared SQL helpers         | MED    | 60min     | storage                |
| 15  | 🟠       | Add Pebble coverage: iterateEvents/deserializeEvent error paths  | HIGH   | 30min     | storage                |
| 16  | 🟠       | Add PostgreSQL integration tests                                 | HIGH   | 120min    | storage                |
| 17  | 🟡       | Core/decider coverage 92.7% → 95%+ (Execute/publish paths)       | MED    | 30min     | core/decider           |
| 18  | 🟡       | Add `WithLogger` to `OutboxPublisher`                            | MED    | 15min     | core/event             |
| 19  | 🟡       | Add `ContextEnricher` wiring to repositories                     | MED    | 30min     | core/aggregate,decider |
| 20  | 🟡       | Rename `CQRSAdapter` → `PebbleEventStore`                        | LOW    | 20min     | storage                |
| 21  | 🟡       | Wire `context.Context` through Pebble methods                    | MED    | 45min     | storage                |
| 22  | 🟢       | Saga/Process Manager implementation                              | HIGH   | Multi-day | new module             |
| 23  | 🟢       | Watermill module (Kafka/NATS adapter)                            | MED    | Multi-day | new module             |
| 24  | 🟢       | Tagged releases / versioning strategy                            | LOW    | 60min     | meta                   |
| 25  | 🟢       | Fix pre-commit hook executable permission                        | LOW    | 1min      | meta                   |

---

## G) My Top #1 Question I Cannot Figure Out Myself

**Q: Should we change Pebble's JSON tags from `snake_case` to `camelCase` (tagliatelle lint)?**

The `serializableEvent` and `serializableMetadata` structs use `snake_case` JSON tags:

```go
AggregateID   string `json:"aggregate_id"`    // tagliatelle wants: aggregateId
AggregateType string `json:"aggregate_type"`   // tagliatelle wants: aggregateType
OccurredAt    int64  `json:"occurred_at"`      // tagliatelle wants: occurredAt
```

**The problem:** Changing these tags is a **breaking change for any persisted Pebble data**. Anyone who saved events with `aggregate_id` would fail to deserialize with `aggregateId`.

**Options:**

1. **Change to camelCase** — Break on-disk format. Requires migration tool.
2. **Add `//nolint:tagliatelle`** — Accept snake_case as intentional (consistent with SQL column naming).
3. **Add both tags** — Use custom `MarshalJSON`/`UnmarshalJSON` that reads both formats.
4. **Mark Pebble as experimental** — Not yet released, so no consumers to break.

**My recommendation:** Option 2 (nolint) or Option 4 (mark experimental). The Pebble store is brand new and snake_case is actually more consistent with how SQL databases name columns.

**Decision needed from you.**

---

## Coverage Matrix (Current)

| Module               | Coverage  | Target | Status     |
| -------------------- | --------- | ------ | ---------- |
| core/command         | 100.0%    | 95%    | ✅         |
| core/query           | 100.0%    | 95%    | ✅         |
| core/pkg/dispatcher  | 100.0%    | 95%    | ✅         |
| core/pkg/id          | 100.0%    | 95%    | ✅         |
| catalog/adapters     | 100.0%    | 95%    | ✅         |
| middleware           | 100.0%    | 95%    | ✅         |
| memory               | 99.5%     | 95%    | ✅         |
| projection           | 98.3%     | 95%    | ✅         |
| catalog/d2           | 97.6%     | 95%    | ✅         |
| **core/aggregate**   | **96.9%** | 95%    | ✅ (+4.8%) |
| catalog/eventcatalog | 95.7%     | 95%    | ✅         |
| catalog              | 94.4%     | 95%    | ⚠️ (-0.6%) |
| core/event           | 94.5%     | 95%    | ⚠️ (-0.5%) |
| catalog/asyncapi     | 93.9%     | 95%    | ⚠️ (-1.9%) |
| core/decider         | 92.7%     | 95%    | ⚠️         |
| **storage**          | **85.4%** | 95%    | ❌ (+1.6%) |

---

## Lint Matrix (Current)

| Module      | Issues | Breakdown                                                                                                                                                                      |
| ----------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| core        | **0**  | —                                                                                                                                                                              |
| memory      | **0**  | —                                                                                                                                                                              |
| catalog     | **0**  | —                                                                                                                                                                              |
| middleware  | **0**  | —                                                                                                                                                                              |
| integration | **0**  | —                                                                                                                                                                              |
| projection  | **0**  | —                                                                                                                                                                              |
| storage     | **66** | err113:7, tagliatelle:9, wrapcheck:8, varnamelen:8, noctx:5, dupl:6, exhaustruct:6, noinlineerr:4, errcheck:4, revive:2, godoclint:2, nestif:1, nilnil:1, unconvert:1, gosec:1 |

---

## Commits This Session (5)

```
f1d793e chore: add aggregate/pebble coverage tests, fix sqlite helpers, update golden files
d46e6e2 chore(lint): migrate gomodguard to gomodguard_v2
87e83e4 chore: resolve catalog lint issues, add pebble edge-case tests
72e8fd4 docs(status): add session 58 comprehensive status report
0dbd51f chore(lint): resolve 5 lint issues in core/memory, update golden tests
```

---

## Uncommitted Changes (4 files, from pre-commit auto-fixes)

```
 .golangci.yml                         | 3 +++  (depguard allowlist for pebble/sqlite/turso)
 storage/sqlite_transactional_store.go | 3 ++-  (embeddedstructfieldcheck + revive nolint)
 storage/transactional_store.go        | 3 ++-  (embeddedstructfieldcheck + revive nolint)
 storage/turso_connector.go            | 1 +    (embeddedstructfieldcheck blank line)
```

These reduce storage lint from 73→66 (7 issues auto-fixed).

---

## Session History (Recent)

| Session | Date       | Focus                     | Key Result                                  |
| ------- | ---------- | ------------------------- | ------------------------------------------- |
| 58-59   | 2026-05-07 | Lint zero + Coverage      | 0 lint in 6/7 modules, aggregate 96.9%      |
| 57-58   | 2026-05-07 | Code quality sweep        | Deduplication, function decomposition       |
| 55-56   | 2026-05-03 | Comprehensive improvement | TransactionalStore, Pebble store added      |
| 54      | 2026-05-03 | Dependency elimination    | Removed cockroachdb/errors, json experiment |
| 53      | 2026-05-02 | Godoc + deduplication     | 91.6% total coverage                        |
| 48      | 2026-05-01 | Execution phases 1-7      | SnapshotStrategy, ISP, error classification |

---

_Arte in Aeternum_
