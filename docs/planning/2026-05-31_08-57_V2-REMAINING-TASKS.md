# V2.0.0 Remaining Execution Plan — Fine-Grained Tasks

> **Date:** 2026-05-31 08:57  
> **Status:** Continuation of sessions 161–162  
> **Previous progress:** 34/55 micro-tasks done (62%)  
> **This plan:** 48 remaining tasks, max 12min each  
> **Estimated total:** ~6.5 hours

---

## Methodology

All remaining TODOs from the v2.0.0 audit broken into bite-sized tasks. Each task is:
- **Max 12 minutes** — single focused change
- **Independently verifiable** — build + test after each
- **Sorted by Pareto impact** — 1%→51%, 4%→64%, 20%→80%

---

## Priority Tiers

| Tier | Color | Meaning | % of effort |
|------|-------|---------|-------------|
| **P0** | 🔴 | Must fix before ANY release — crashes, data loss, security | 8% |
| **P1** | 🟠 | Should fix before v2.0.0 — wrong behavior, API lies | 17% |
| **P2** | 🟡 | Code quality — duplication, naming, minor API roughness | 42% |
| **P3** | 🔵 | Test coverage, docs, polish | 21% |
| **P4** | ⚪ | Nice-to-have — examples, architecture polish | 12% |

---

## Master Task Table

| # | Tier | Task | Module | Effort | Impact | Depends |
|---|------|------|--------|--------|--------|---------|
| | | | | | | |
| **🔴 TIER P0 — Must Fix (3 tasks, ~25min)** | | | | | | |
| T1 | 🔴 | Replace `panic("multisig: nil signer")` in `MultiSignMiddleware` with error-returning middleware | `signing/multisig` | 8min | Prevents crash | — |
| T2 | 🔴 | Replace `panic("multisig: nil signer")` in `MultiVerifyMiddleware` with error-returning middleware | `signing/multisig` | 8min | Prevents crash | — |
| T3 | 🔴 | Replace `panic` in `MultiVerifyMiddlewareFor` + `RequireMultiSigMiddleware` with error returns | `signing/multisig` | 9min | Prevents crash | — |
| | | | | | | |
| **🟠 TIER P1 — Should Fix Before v2 (8 tasks, ~65min)** | | | | | | |
| T4 | 🟠 | Add nil store guard to `NewVersionedStore` — return error if `store == nil` | `schema` | 8min | Prevents nil panic | — |
| T5 | 🟠 | Change `ErrTypeAssertion` from `NewCorruption` to `NewRejection` | `command` | 5min | Correct taxonomy | — |
| T6 | 🟠 | Update tests referencing `ErrTypeAssertion` family | `command` + `integration` | 7min | Tests match new family | T5 |
| T7 | 🟠 | Create `cmd/api-stability/go.mod` with proper replace directives | `cmd/api-stability` | 10min | Tool actually works | — |
| T8 | 🟠 | Rename `ParseUserAgent` → `NewUserAgent` (trim, not parse) | `event` | 5min | Honest naming | — |
| T9 | 🟠 | Update all `ParseUserAgent` call sites to `NewUserAgent` | `event` tests | 5min | Consistency | T8 |
| T10 | 🟠 | Add `Deprecated: ParseUserAgent` alias for backwards compat | `event` | 3min | Non-breaking | T8 |
| T11 | 🟠 | Validate snapshot store+codec pair in `NewRepository` — error if strategy set but missing deps | `decider` | 12min | Prevents silent skip | — |
| | | | | | | |
| **🟡 TIER P2 — Code Quality (20 tasks, ~155min)** | | | | | | |
| T12 | 🟡 | Extract `markEvent(evt, key)` helper from `MarkTombstone`/`MarkRebirth` in `tombstone.go` | `event` | 10min | Removes duplication | — |
| T13 | 🟡 | Rewrite `MarkTombstone` to call `markEvent` | `event` | 3min | Uses shared helper | T12 |
| T14 | 🟡 | Rewrite `MarkRebirth` to call `markEvent` | `event` | 3min | Uses shared helper | T12 |
| T15 | 🟡 | Run tombstone tests to verify refactor | `event` | 2min | Verification | T13, T14 |
| T16 | 🟡 | Export `ExtractOrPassThrough` from `signing` package (make it public) | `signing` | 5min | Shared utility | — |
| T17 | 🟡 | Update `signing/middleware.go` to use exported `ExtractOrPassThrough` | `signing` | 3min | Uses shared | T16 |
| T18 | 🟡 | Update `signing/multisig/middleware.go` to use `signing.ExtractOrPassThrough` — remove duplicate | `signing/multisig` | 5min | Removes duplication | T16 |
| T19 | 🟡 | Run signing + multisig tests to verify | `signing` | 3min | Verification | T17, T18 |
| T20 | 🟡 | Extract generic `recoveryMiddleware(kind string, opts)` in `recovery.go` | `middleware` | 10min | Removes triplication | — |
| T21 | 🟡 | Rewrite `CommandRecovery` to use generic helper | `middleware` | 3min | Uses shared | T20 |
| T22 | 🟡 | Rewrite `EventRecovery` to use generic helper | `middleware` | 3min | Uses shared | T20 |
| T23 | 🟡 | Rewrite `QueryRecovery` to use generic helper | `middleware` | 3min | Uses shared | T20 |
| T24 | 🟡 | Run middleware recovery tests to verify | `middleware` | 3min | Verification | T21-T23 |
| T25 | 🟡 | Add doc comments to `CommandID`, `NewCommandID`, `ParseCommandID`, `MustParseCommandID` in `id/command_id.go` | `id` | 5min | API docs | — |
| T26 | 🟡 | Remove dead `CommandTypeError` from `example/todo/commands/mixin.go` | `example/todo` | 3min | Removes dead code | — |
| T27 | 🟡 | Fix `example/user/catalog.go` — use command payload types instead of event payload types for command catalog entries | `example/user` | 8min | Correct catalog | — |
| T28 | 🟡 | Add nil check for `signing/multisig/extract.go:89` — panic on nil `*MultiSigner` in `VerifierMap` | `signing/multisig` | 5min | Prevents crash | — |
| T29 | 🟡 | Remove `context.Background()` from `OTelMetricsRecorder.Observe` — add `ctx context.Context` param | `middleware` | 8min | Trace correlation | — |
| T30 | 🟡 | Update `MetricsRecorder` interface + callers for new `Observe` signature | `middleware` | 8min | API update | T29 |
| T31 | 🟡 | Run full build + lint + test after P2 batch | all | 5min | Verification | T12-T30 |
| | | | | | | |
| **🔵 TIER P3 — Tests + Docs (12 tasks, ~90min)** | | | | | | |
| T32 | 🔵 | Add test for `NewVersionedStore(nil)` returning error | `schema` | 8min | Covers nil guard | T4 |
| T33 | 🔵 | Add test for `NewUpcaster` with nil function → rejection error on `Upcast()` | `schema` | 8min | Covers nil guard | — |
| T34 | 🔵 | Add race detector test for `CatalogDispatcher` concurrent RegisterHandlerMeta | `dispatcher` | 10min | Proves data race fix | — |
| T35 | 🔵 | Add test for `RequireMultiSigMiddleware` nil/empty verifier map returning error (not panic) | `signing/multisig` | 8min | Covers panic→error | T3 |
| T36 | 🔵 | Add test for `listing.StatusMiddleware` | `listing` | 12min | Coverage boost | — |
| T37 | 🔵 | Add test for `pebble` store concurrent Save (two goroutines, same aggregate) | `pebble` | 10min | Proves locking | — |
| T38 | 🔵 | Add test for `watermill` subscriber edge cases (nil message, empty batch) | `watermill` | 10min | Coverage | — |
| T39 | 🔵 | Add test for `catalog/docserver` error paths (missing spec, invalid YAML) | `catalog/docserver` | 12min | Coverage | — |
| T40 | 🔵 | Add test for `decider.NewRepository` — error when strategy set but snapshotStore nil | `decider` | 8min | Covers validation | T11 |
| T41 | 🔵 | Update `TODO_LIST.md` — mark all 30 completed items as `[x]` | meta | 12min | Honesty | T1-T31 |
| T42 | 🔵 | Update `FEATURES.md` with naming changes (`ErrHandlerNotFound`, `NewUserAgent`) | meta | 5min | Docs accuracy | T5, T8 |
| T43 | 🔵 | Update `AGENTS.md` Quick Reference with new naming conventions | meta | 5min | Future sessions | T5, T8 |
| | | | | | | |
| **⚪ TIER P4 — Examples + Polish (5 tasks, ~45min)** | | | | | | |
| T44 | ⚪ | Add README.md to `example/storage/` | `example/storage` | 10min | Discoverability | — |
| T45 | ⚪ | Add README.md to `example/projection/` | `example/projection` | 10min | Discoverability | — |
| T46 | ⚪ | Add README.md to `example/listing/` | `example/listing` | 10min | Discoverability | — |
| T47 | ⚪ | Add README.md to `example/saga-pattern/` | `example/saga-pattern` | 10min | Discoverability | — |
| T48 | ⚪ | Run `dupl -t 25` to verify clone count reduction post-dedup | meta | 5min | Metrics | T12-T24 |

---

## Execution Order — Dependency Graph

```
Phase 1: P0 (parallel) — T1, T2, T3
    │
Phase 2: P1 sequential chains —
    T5 → T6 (ErrTypeAssertion)
    T8 → T9 → T10 (ParseUserAgent)
    T4, T7, T11 (independent)
    │
Phase 3: P2 dedup chains —
    T12 → T13 → T14 → T15 (tombstone)
    T16 → T17 → T18 → T19 (signing extract)
    T20 → T21 → T22 → T23 → T24 (recovery)
    T25, T26, T27, T28, T29 → T30 (independent + metrics)
    T31 (full verification)
    │
Phase 4: P3 tests — T32-T40 (mostly parallel, some depend on P1/P2)
    T41-T43 (docs updates, after all code done)
    │
Phase 5: P4 examples — T44-T48 (parallel)
```

---

## Summary Statistics

| Tier | Tasks | Total Effort | % of Plan |
|------|-------|-------------|-----------|
| 🔴 P0 Must Fix | 3 | 25min | 8% |
| 🟠 P1 Should Fix | 8 | 65min | 17% |
| 🟡 P2 Code Quality | 20 | 155min | 42% |
| 🔵 P3 Tests+Docs | 12 | 96min | 26% |
| ⚪ P4 Polish | 5 | 45min | 12% |
| **TOTAL** | **48** | **~6.5h** | **100%** |

---

## Pareto Analysis

| Effort % | Impact % | Covers |
|----------|----------|--------|
| **6%** (T1-T3) | **51%** | All P0 crashes eliminated — library won't kill your process |
| **15%** (T4-T11) | **64%** | + nil guards, correct taxonomy, honest naming, working tools |
| **40%** (T12-T31) | **80%** | + deduplication, docs, metrics context, dead code removed |
| **60%** (T32-T43) | **90%** | + test coverage, TODO_LIST honesty, FEATURES accuracy |
| **100%** (T44-T48) | **100%** | + example READMEs, clone metrics |

---

## Risk Notes

- **T5** (ErrTypeAssertion family change): Breaking change for consumers using `errors.As` to check Corruption family. Low risk — `ErrTypeAssertion` is rarely used in consumer code.
- **T8-T10** (ParseUserAgent rename): Non-breaking due to deprecated alias. Safe.
- **T16** (export ExtractOrPassThrough): Expands public API surface. Alternative: unexported but shared via internal package.
- **T29-T30** (Observe context param): Breaking `MetricsRecorder` interface change. Consumers must update. Plan for v2.0.0 only.
- **T20-T24** (recovery dedup): Internal refactor only. No public API change. Safe.

---

_Plan created: 2026-05-31 08:57_
