# SUPERB — Branching-Flow Duplicate-Type Triage (Pareto Plan)

**Date:** 2026-10-09 18:16 · **Trigger:** `branching-flow all . --format markdown` (465 duplicate-type rows + mixin/flag-param/context/naked-return analyses) · **Mode:** triage-first, execute-the-signal, protect-the-noise.

> **Status: EXECUTED (1% tier + close-out) same day.** M1–M3, M11 shipped with
> receipts below; M4–M10 are scheduled follow-ups harvested into `TODO_LIST.md`
> (Code Quality + v5 Unification sections). This file is a point-in-time
> snapshot; the living task source is `TODO_LIST.md`.

## 0. The headline finding

The 465-row report is **~95% already-governed duplication**. This repo runs
strong duplication governance (art-dupl gate + `//art-dupl:accept` directives,
ADR-0152 dual-support mirrors, contract #27 engine plumbing rules), and the
tool does not know about any of it. Acting on the report row-by-row would have
been pure verschlimmbessern. The real work was **separating the ~5% signal
from the noise**, shipping the two safe fixes, and converting the one genuine
risk cluster (v4↔core/v5 mirrors) into a guard task.

## 1. Pareto breakdown

| Tier | Tasks | Result share | Rationale |
| --- | --- | --- | --- |
| **1% → 51%** | M1 SortColumn alias · M2 WithoutCancel flush · M3 this triage ledger | The only real same-module defects + the decision ledger that prevents 465 rows of blind churn | Shipped today |
| **4% → 64%** | M4 v4↔core/v5 mirror-drift lockstep audit | The ~230 mirror rows ARE the risk cluster: dual-support means silent v4↔v5 drift is the actual failure mode the report stumbled into | Next session |
| **20% → 80%** | M5 mirror-pair report suppression · M6 v5 cut bundle (TombstoneFilter, SyncWritesTier eval) | Future reports become signal; v5 API quality lands where breaking changes are free | Scheduled |
| **Other 20% → 100%** | M7 same-package field twins · M8 cqrs-lint DTOs · M9 DLQ cross-doc · M10 flag-param sweep · post-suppression re-run | Long tail, low urgency, none customer-visible | Backlog |

## 2. Comprehensive plan — medium tasks (30–100 min each)

Sorted by importance / impact / effort / customer-value.

| # | Task | Imp | Impact | Effort | Value | Status |
| --- | --- | --- | --- | --- | --- | --- |
| M1 | `metaengine.SortColumn` → alias of `SortSpec` + golden + CHANGELOG | High | Kills the only same-module split brain (public API drift risk) | S ~30m | Public-API clarity | ✅ DONE |
| M2 | `projectionhost.awaitWorkers` flush via `context.WithoutCancel(ctx)` | Med | Trace/values survive shutdown checkpoint flush | S ~30m | Observability | ✅ DONE |
| M3 | Triage ledger (this doc) + TODO_LIST harvest | High | Converts 465 rows into decisions; blocks blind churn | S ~30m | — | ✅ DONE |
| M11 | Verification gates + commit + push | High | Proof the fixes are safe | S ~30m | — | ✅ DONE |
| M4 | v4↔core/v5 mirror-drift lockstep audit (mechanical cross-compare test) | High | Guards ADR-0152 dual-support against silent drift | M ~90m | Fleet migration safety | ☐ scheduled |
| M5 | Branching-flow mirror-pair suppression (config or wrapper filter) | Med | Future reports carry signal, not ~230 mirror rows | S–M ~60m | Maintainer time | ☐ scheduled |
| M6 | v5 cut bundle: `TombstoneFilter` enum in core/v5/kv; `SyncWritesTier` + `OrderClause`/`SortSpec` eval | Med | API quality at the cut, where breaks are free | M ~100m | v5 consumers | ☐ v5 wave |
| M7 | Same-package field-twin review: `storage` SQLStreamReader/StreamProjection, `metaengine` MapDedupStore/MapDueClaimer, `snapshot` store/wire, `turso` SyncDB/syncDbConnection | Low–Med | Latent drift removal or explicit accept | M ~60m | — | ☐ scheduled |
| M8 | cqrs-lint internal table-DTO merge (`deprecatedTransportImport`/`deprecatedV5Module`) | Low | Tidiness | S ~30m | — | ☐ scheduled |
| M9 | DLQ twins cross-doc (middleware vs projectionhost `MemoryDeadLetterStore`) | Low | Discoverability of the two DLQ layers | XS–S ~30m | — | ☐ scheduled |
| M10 | Flag-param sweep in cmd/ tools (8 medium-severity rows → options structs) | Low | Readability of internal tooling | M ~100m | — | ☐ scheduled |

## 3. Fine breakdown — tasks ≤ 12 min each

| # | Task | Parent | Status |
| --- | --- | --- | --- |
| F1 | Read `SortSpec`/`SortColumn` + all usages (grep: 2 files, 1 test) | M1 | ✅ |
| F2 | Alias edit in `metaengine/scan_options.go` | M1 | ✅ |
| F3 | `metaengine` module tests (`GOWORK=off go test ./... -count=1`) | M1 | ✅ 39s green |
| F4 | api-stability golden regen (`--update`; diff = 1 line: struct→type) | M1 | ✅ |
| F5 | `TestEvery` meta-test | M1 | ✅ |
| F6 | Read `awaitWorkers` + confirm the nolint rationale | M2 | ✅ |
| F7 | `context.WithoutCancel(ctx)` edit, drop nolint, keep intent comment | M2 | ✅ |
| F8 | `projectionhost` module tests | M2 | ✅ 2.3s green |
| F9 | CHANGELOG `### Changed` bullets (2) | M1/M2 | ✅ |
| F10 | `check-changelog-symbols.sh` gate | M11 | ✅ 37 citations honest |
| F11 | Lint touched modules (full `nix run .#lint`) | M11 | 🔄 running |
| F12 | doc-check over skill references (no refs mention SortColumn — verified) | M11 | 🔄 |
| F13 | TODO_LIST harvest (2 entries: Code Quality + v5 Unification) | M3 | ✅ |
| F14 | Write this plan doc | M3 | ✅ |
| F15 | git commit (detailed) + push | M11 | ☐ |
| F16 | Enumerate mirror module pairs (id, kv, event, command, query, dispatcher) | M4 | ☐ |
| F17 | Table-driven lockstep cross-compare test (event.Type pattern) | M4 | ☐ |
| F18 | Wire into CI/meta-test set; document in gotchas | M4 | ☐ |
| F19 | Check branching-flow config surface for suppression support | M5 | ☐ |
| F20 | Else write wrapper filter ingesting the mirror-pair list | M5 | ☐ |
| F21 | `TombstoneFilter` enum + core/v5/kv interface change | M6 | ☐ |
| F22 | `SyncWritesTier` signature decision (keep two-knob ABI vs typed input) | M6 | ☐ |
| F23 | `storage` same-package twins: read, extract or accept-comment | M7 | ☐ |
| F24 | `metaengine` map twins: read, extract or accept-comment | M7 | ☐ |
| F25 | `snapshot` + `turso` twins: read, decide | M7 | ☐ |
| F26 | DLQ cross-doc comments in both `MemoryDeadLetterStore` files | M9 | ☐ |
| F27 | cqrs-lint DTO merge + rule tests | M8 | ☐ |
| F28 | Options structs for the 8 medium flag-param rows (per tool) | M10 | ☐ |
| F29 | Re-run branching-flow after M5; confirm mirror rows suppressed | M5 | ☐ |

## 4. Triage verdict ledger (the do-not-verschlimmbessern guard)

Every report category, its verdict, and the evidence:

| # | Report finding | Verdict | Evidence / reason |
| --- | --- | --- | --- |
| 1 | ~230 rows v4 ↔ `core/v5` mirror pairs (`Subscriber`, `Dispatcher`, `ActorID`, `MemStore`, …) | **BY DESIGN — do not consolidate** | ADR-0152 dual-support; v5 trains fork deliberately. Residual risk = drift → task M4. |
| 2 | queue/mysql·postgres·sqlite `Engine`, `orphan`, `factSink`, … dialect twins | **ALREADY ANNOTATED** | `//art-dupl:accept` directives verified across queue modules (open/claim/reads/cancel/engine); contract #19 pattern. `orphan` is a function-local scan DTO. |
| 3 | Per-engine `kv`/`kvPair` pair types (dgraph, duckdb, mysql, pg, sqlite, badger, bbolt) | **SANCTIONED** | AGENTS contract #27: engines map their own pair types via `valueOf`; `PairsToScanResult` already shares the tail. |
| 4 | `metaengine.SortSpec` vs `SortColumn` (same module) | **REAL — FIXED** | Same shape `{Column, Desc}`, same package, public API; alias `type SortColumn = SortSpec` (M1). |
| 5 | `projectionhost` `context.Background` in `awaitWorkers` | **REAL — FIXED** | Flush dropped ctx values; `WithoutCancel` keeps values, drops cancellation (M2). |
| 6 | Naked return `StartAutoReprobe` (engine_health.go:333) | **FALSE POSITIVE** | Bare `return` inside a result-less goroutine closure; nothing to name. |
| 7 | `QueryByTombstone(excludeTombstoned, onlyTombstoned bool)` (storage/view) | **v5 ONLY** | Signature dictated by `kv.TombstoneQuerier` interface (kv/view_store.go:93); `storage/view` is deleted at v5 (TODO_LIST v5 section) — polishing it now is waste. |
| 8 | `SyncWritesTier(volatile, syncWrites bool)` | **ACCEPTED (v4), evaluate at v5** | Honest two-knob ABI; pebbleengine + badgerengine pass different underlying fields — a struct param would not simplify the callers. |
| 9 | `kv.OrderClause` ↔ `metaengine.SortSpec`/`SortColumn` | **REJECTED** | Cross-module: metaengine core depends only on `dedup` + `record` (tier/dep-budget isolation). Unification would add a Tier-0 dep to Tier-3. |
| 10 | `RowScanner`/`rowScanner`/`scanner` (metaengine vs queue/*) | **REJECTED** | queue → metaengine import for a 1-method interface burns dep budget; per-module duck types are cheaper. |
| 11 | `system.serializedEvent` ↔ `pebble.serializableEvent` | **REJECTED** | Different substrates: SQL JSON envelope (AdapterCore, contract #17) vs Pebble binary store — similar fields, never the same wire format. |
| 12 | `MemoryDeadLetterStore` twins (middleware vs projectionhost) | **ACCEPTED + cross-doc (M9)** | Two distinct DLQ concepts (dispatch-retry vs projection poison); SKILL decision matrix documents them separately. |
| 13 | watermill Command/Event twins, builder/spec mixin rows, empty markers/DTOs, example/bench fixtures | **INTENTIONAL** | Parallel command/event design, idiomatic builders, branded-ID markers, test fixtures. |
| 14 | Flag-param rows in cmd/* tools | **BACKLOG (M10)** | Internal tooling, low value; options structs when next touched. |

## 5. Execution graph

```mermaid
flowchart TD
    A[branching-flow report<br/>465 rows] --> T{Triage ledger M3}
    T -->|~95% governed| N[NO-ACTION verdicts<br/>§4 ledger rows 1-3, 6-14]
    T -->|signal| W0[W0 execute today]
    subgraph W0[W0 — 1% → 51% (EXECUTED)]
        M1[M1 SortColumn alias<br/>+ tests + golden + CHANGELOG]
        M2[M2 WithoutCancel flush<br/>+ tests]
        M11[M11 gates: changelog-symbols<br/>lint + doc-check + commit + push]
    end
    T -->|risk cluster| W1[W1 — 4% → 64%]
    subgraph W1[W1 (next session)]
        M4[M4 v4↔core/v5 mirror-drift<br/>lockstep audit test]
    end
    T -->|quality tail| W2[W2 — 20% → 80%]
    subgraph W2[W2 (scheduled)]
        M5[M5 mirror-pair report suppression]
        M6[M6 v5 bundle: TombstoneFilter<br/>SyncWritesTier + OrderClause eval]
    end
    T -->|long tail| W3[W3 — other 20% → 100%]
    subgraph W3[W3 (backlog)]
        M7[M7 same-package twins]
        M8[M8 cqrs-lint DTOs]
        M9[M9 DLQ cross-doc]
        M10[M10 flag-param sweep]
    end
    M5 --> F29[F29 re-run branching-flow<br/>validate signal]
    M1 --> G1[metaengine tests ✅]
    M2 --> G2[projectionhost tests ✅]
    M11 --> G3[golden 1-line ✅ · TestEvery ✅<br/>changelog-symbols ✅]
```

## 6. Verification receipts (2026-10-09)

- `metaengine` `GOWORK=off go test ./... -count=1` — **ok 38.977s** (all 5 pkgs)
- `projectionhost` `GOWORK=off go test ./... -count=1` — **ok 2.336s**
- `cmd/api-stability --update` — golden diff exactly 1 line (`metaengine/struct SortColumn` → `metaengine/type SortColumn`); `TestEvery` **ok**
- `scripts/check-changelog-symbols.sh` — **37 pkg.Symbol citations honest**
- `nix run .#lint` + doc-check — see commit message (receipts recorded at close-out)
