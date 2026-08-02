# cqrs-lint — Consumer Feedback: crush-daily

**Consumer:** [crush-daily](https://github.com/LarsArtmann/crush-daily) — Daily AI-powered insights from Crush development databases. CQRS event-sourcing app with HTTP server (cqrs-htmx), cron scheduler, OTel tracing, and SQLite event store.
**Version used:** go-cqrs-lite v4.2.0 (event, decider, storage, middleware, command, query, catalog, codec, otel, id, snapshot), cqrs-htmx v4.6.1
**lint version:** `cqrs-lint v0.2.2`
**Date:** 2026-08-02

---

## Executive Summary

cqrs-lint v0.2.2 runs cleanly, is fast (554ms for 27 files), and the feature-profile detection is largely correct. The **signal-to-noise ratio is roughly 25%** — of 40 findings, about 10 are genuinely actionable or correctly flagged, and 30 are false positives or not-applicable module suggestions. The biggest noise sources are a single root cause (C036 backend-mismatch, 6 findings) and a known limitation (E007 runtime registration, 6 findings).

The findings break down into three categories:

| Category                                             | Count | Assessment                                                                                                                                                                             |
| ---------------------------------------------------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **False positives** (detector logic gap)             | 13    | C036 (6), E007 (6), B022 (1)                                                                                                                                                           |
| **Consciously accepted** (project-specific decision) | 15    | C008 (13), S006 (2)                                                                                                                                                                    |
| **Actionable / correctly flagged**                   | 12    | C009 (1), C016 (1), C034 (1), E016 (1), F005 (1), F013 (1), F017 (1), B005 (1), B010 (1), B017 (1), A005 (1), B022 (1 — the finding is valid even though the suggestion text is wrong) |

---

## Part 1: False Positives (Detector Logic Gaps)

### FP 1: C036 — Event store classified as "custom" when it IS the library's SQLite store

**Severity: HIGH** — 6 false positives, all from one root cause.

#### What happens

The feature profile reports `store: custom`, but the event store is created via the library's own SQLite constructor:

```go
// internal/infrastructure/setup.go:318
store, err := storage.NewSQLiteEventStore(db)
```

Because the linter sees `store: custom`, every other `storage.*` call in the same file fires C036:

```
WARNING  C036  setup.go:220  snapshot store uses sqlite backend but event store uses custom
WARNING  C036  setup.go:303  store uses sqlite backend but event store uses custom
WARNING  C036  setup.go:308  store uses sqlite backend but event store uses custom
WARNING  C036  setup.go:312  store uses sqlite backend but event store uses custom
WARNING  C036  setup.go:316  store uses sqlite backend but event store uses custom
WARNING  C036  setup.go:347  store uses sqlite backend but event store uses custom
```

All 6 are false positives. The snapshot store, event store, schema init, WAL, and pool configuration all use the same SQLite `*sql.DB` connection — they are perfectly aligned.

#### Root cause

The feature-profile detector does not recognize `storage.NewSQLiteEventStore(db)` as a library-provided SQLite store. It likely scans for a direct `sql.Open` call or a constructor pattern it knows, and classifies anything else as "custom."

The storage module's public API surface includes:

- `storage.OpenSQLite(path)` — opens the DB
- `storage.NewSQLiteEventStore(db)` — creates the event store
- `storage.NewSQLiteSnapshotStore(db)` — creates the snapshot store
- `storage.SQLiteInitSchema(ctx, db)` — applies schema
- `storage.SQLiteEnableWAL(ctx, db)` — enables WAL
- `storage.ConfigureSQLitePool(db)` — configures connection pool

The detector should recognize `storage.NewSQLiteEventStore` as a `sqlite`-backed event store (not `custom`).

#### Fix suggestion

Add `storage.NewSQLiteEventStore` to the known-store-constructor list in the feature-profile analyzer. The pattern to match:

```go
// Recognize these as sqlite-backed stores:
storage.NewSQLiteEventStore(db)
storage.NewSQLiteSnapshotStore(db)
// Recognize these as sqlite setup calls (not "custom backend"):
storage.OpenSQLite(path)
storage.SQLiteInitSchema(ctx, db)
storage.SQLiteEnableWAL(ctx, db)
storage.ConfigureSQLitePool(db)
```

This would fix all 6 C036 findings and correct the feature profile from `store: custom` to `store: sqlite`.

---

### FP 2: E007 — "Query type has no registered handler" for runtime-registered queries

**Severity: MEDIUM** — 6 false positives, known limitation already reported by other consumers.

#### What happens

crush-daily registers query handlers at runtime in an init function:

```go
// internal/queries/queries.go:65
func Register(dispatcher *query.Dispatcher, readModel *infrastructure.ReadModelStore) error {
    h := handlers{readModel: readModel}
    err := register(dispatcher, TypeListReports, h.listReports)
    // ...
}
```

This `Register` function is called from `server.go` at startup. The linter scans statically and sees the query type definitions but not the runtime registration, so it fires E007 for all 6 query types:

```
WARNING  E007  queries.go:40   Query type "ListReportsQuery" has no registered handler
WARNING  E007  queries.go:49   Query type "GetReportQuery" has no registered handler
WARNING  E007  rollup.go:15    Query type "RollupQuery" has no registered handler
WARNING  E007  rollup.go:94    Query type "CompareQuery" has no registered handler
WARNING  E007  search.go:24    Query type "SearchQuery" has no registered handler
WARNING  E007  trend.go:15     Query type "TrendQuery" has no registered handler
```

#### Context

This was reported in [2026-07-17_cqrs-htmx_cqrs-lint-feedback.md](../2026-07-17_cqrs-htmx_cqrs-lint-feedback.md) and [2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md](../2026-08-02_bank-sync_cqrs-lint-improvement-proposals.md) already. The cqrs-htmx pattern (Path A dispatch) inherently registers at runtime via `query.RegisterTyped` — there is no static registration to detect.

#### Suggested mitigation

Two options short of full cross-file call tracking:

1. **Recognize the `Register` function pattern:** If a package contains a function named `Register` that takes `*query.Dispatcher` as a parameter, assume queries in that package are registered at runtime and suppress E007 for all types in that package.

2. **Config-based suppression** (already requested by bank-sync): `"disabled-rules": ["E007"]` in `.cqrs-lint.json`.

---

### FP 3: B022 — Suggests `decider.CommandCausalityEnricher` (wrong package)

**Severity: MEDIUM** — wrong suggestion text, already reported by bank-sync.

#### What happens

crush-daily uses a custom `correlation.ContextEnricher()`:

```go
// internal/infrastructure/setup.go:239
decider.WithEnricher[domain.DailyReportState](correlation.ContextEnricher()),
```

The linter correctly detects this is not the canonical enricher, but suggests the wrong package:

```
WARNING  B022  Custom enricher (WithEnricher) passed to decider.NewRepository —
use decider.CommandCausalityEnricher for typed command causality
```

The function is `event.CommandCausalityEnricher`, not `decider.CommandCausalityEnricher`. This was reported in the bank-sync feedback (Bug 1). The suggestion text in `b022_b025.go` hardcodes `decider.CommandCausalityEnricher`.

#### Additional context from crush-daily

crush-daily deliberately uses `correlation.ContextEnricher()` instead of `event.CommandCausalityEnricher` because the enricher stamps a correlation ID from the context (not command causality metadata). This is a different enricher purpose. The B022 rule assumes the only reason to use `WithEnricher` is command causality, but correlation ID propagation is a separate, valid use case.

#### Fix suggestions

1. **Fix the package name** in the suggestion text: `event.CommandCausalityEnricher`.
2. **Broaden the exemption** to recognize correlation ID enrichers as a valid pattern, not just command causality.

---

## Part 2: Consciously Accepted Findings (Correct Detection, Wrong Project)

These findings are **correctly detected** but consciously accepted for this project. They are not false positives — the linter is doing its job. Documented here for completeness.

### C008 — float64 for monetary values (13 findings)

crush-daily uses `float64` for `CostUSD` fields throughout. These are API-reported cost estimates from LLM providers (OpenAI, Anthropic, etc.), not financial transactions. The precision requirements are "roughly how much did we spend" — float64 is appropriate. Changing to `decimal` or `int64` cents would:

- Break serialized event payloads (breaking schema change)
- Add a dependency for no practical benefit
- Require migration of all stored events

### S006 — Financial data without encryption (2 findings)

The "financial data" is LLM API cost estimates in a personal analytics dashboard. This data is not sensitive — it's aggregate usage statistics. Encryption would be overkill.

### F013 — Manual HTTP handlers without transport module (1 finding)

crush-daily uses `cqrs-htmx` for its HTTP layer, which provides its own dispatch, middleware, and error handling. The `transport/http` module is for raw net/http integration; `cqrs-htmx` is a higher-level abstraction on top of the same CQRS dispatchers. F013 doesn't recognize cqrs-htmx as a transport layer.

**Suggestion:** If the linter can detect `cqrshtmx.New` or `cqrshtmx.MustNew` calls, it should suppress F013 — cqrs-htmx IS a transport layer.

### F017 — No dedup module (1 finding)

Single-instance deployment with synchronous bus. Duplicates are impossible. Already reported by bank-sync as needing feature-profile gating (`HasAsyncBus`).

### C009 — NewCollectCommand panic (1 finding)

`NewCollectCommand` panics if `command.New()` fails. This is a must-pattern like `regexp.MustCompile` — `command.New` can only fail on empty type or zero stream ID, both impossible with constant/generated inputs. The panic is correct for a constructor that wraps an infallible construction.

**Suggestion:** C009 could recognize the must-pattern when the panic is inside a constructor function (function name starts with `New` and returns a pointer to a struct). This is the Go idiom for must-constructors.

---

## Part 3: Actionable Findings (Correctly Flagged, Worth Investigating)

These are genuine quality suggestions the linter correctly identified. Listed in priority order.

### B005 — Fold uses switch instead of StrictApply (1 finding)

```go
// internal/domain/state.go:121
func FoldDailyReport(state DailyReportState, evt event.Event) (DailyReportState, error) {
```

The fold function uses a switch statement over event types. `decider.StrictApply` would provide compile-time exhaustiveness checking. **Worth investigating** — need to verify the API exists in v4.2.0 and understand the migration path.

### B017 / A005 — Full replay on startup instead of incremental catch-up (2 findings)

```go
// internal/infrastructure/setup.go:47
func (c *Container) Rehydrate(ctx context.Context) error {
```

The read model is rebuilt from scratch on every startup by replaying ALL events. `projectionhost.Host` with a checkpoint store would only replay new events. **Actionable** — this is a real startup performance concern as the event store grows. Was a deliberate trade-off (simplicity over performance) but worth revisiting.

### E016 — Server-mode without HealthCheck bundle (1 finding)

The OTel middleware bundle is created but `bundle.HealthCheck(ctx)` is not wired into the `/api/health` endpoint. **Correctly flagged** — the health endpoint currently only checks "is the server running" but not "are all CQRS components healthy."

### C016 / C034 — Context not passed to goroutines / context.Background() in handler (2 findings)

- `server.go:276` uses `context.Background()` for the shutdown timeout — this is actually correct (you WANT a fresh context for graceful shutdown, not the cancelled parent context). But the linter can't distinguish "intentional detached context for shutdown" from "accidentally discarded context."
- `server.go:266` spawns a goroutine for `httpServer.ListenAndServe()` without passing ctx — this is the standard Go HTTP server pattern (the goroutine is intentionally long-lived). **Borderline** — the goroutine IS cleaned up via `httpServer.Shutdown()`.

### B010 — Manual catalog registration (1 finding)

4 `catalog.Event[T]()` calls could be auto-generated by `cqrs-gen`. **Low priority** — the manual calls are only 4 lines and give full control over descriptions.

### F005 — SchemaVersion without upcaster (1 finding)

Events use `event.WithSchemaVersion(1)` but no upcaster is registered. This is expected — upcasters are added when schemas actually change, not preemptively at v1. **Correct detection, premature to fix.**

---

## Part 4: Feature Profile Detection Accuracy

The feature profile is:

```
store:         custom      ← WRONG (should be sqlite)
command-flow:  sync        ← correct
server:        true        ← correct
soft-delete:   false       ← correct
tracing:       on          ← correct
snapshot:      on          ← correct
domain:        unknown     ← would be nice to detect
```

Only one detection error (`store: custom`), but it cascades into 6 C036 false positives. Fixing the store detection would be the single highest-impact improvement for this project.

---

## Part 5: Summary of Requested Improvements

Priority-ordered by impact on this project:

| Priority | Improvement                                                                  | Impact                                                    | Effort       |
| -------- | ---------------------------------------------------------------------------- | --------------------------------------------------------- | ------------ |
| **P0**   | Recognize `storage.NewSQLiteEventStore` as sqlite-backed store               | Eliminates 6 C036 false positives + fixes feature profile | Low          |
| **P1**   | Config-level rule disabling (`"disabled-rules": [...]` in `.cqrs-lint.json`) | Eliminates E007 (6), C008 (13), S006 (2) from output      | Medium       |
| **P1**   | Fix B022 suggestion text: `decider` → `event`                                | Stops sending users to wrong package                      | Trivial      |
| **P2**   | Recognize `Register(dispatcher, ...)` pattern for runtime registration       | Eliminates E007 (6) at the detector level                 | Medium       |
| **P2**   | Recognize `cqrshtmx.New`/`MustNew` as a transport layer                      | Eliminates F013                                           | Low          |
| **P3**   | Feature-profile gating for F017 (sync bus → no dedup needed)                 | Eliminates F017                                           | Medium       |
| **P3**   | C009 must-pattern recognition (New* constructor → panic OK)                  | Eliminates C009                                           | Medium       |
| **P3**   | C016 exemption for shutdown-timeout context.Background()                     | Eliminates C016                                           | Medium       |
| **P4**   | C008 opt-out for non-financial float64 (e.g. cost estimates)                 | Eliminates 13 C008 findings                               | Low (config) |

---

## Part 6: What the Linter Got Right

The linter correctly identified several real architectural improvements:

- **B017/A005** (full replay → projectionhost): genuinely actionable performance improvement
- **E016** (missing health check in bundle): correctly detected a gap
- **B005** (switch → StrictApply): good suggestion for compile-time safety
- **F005** (schema version without upcaster): correct, just premature at v1
- **B022** (custom enricher): the finding is valid even though the suggestion text points to the wrong package

The feature-profile system is working well for `server`, `tracing`, `snapshot`, `command-flow`, and `soft-delete` — all correctly detected. The `store` detection is the only miss, and it's a single pattern-matching gap, not a fundamental design flaw.

The rule taxonomy (A/B/C/E/F/S prefixes) and severity levels are well-designed and make it easy to triage findings by category.
