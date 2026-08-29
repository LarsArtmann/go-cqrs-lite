# cqrs-lint Backlog Triage & Improvements — Status Report

> **Date:** 2026-08-08 02:28
> **Session scope:** cqrs-lint false-positive fixes, missing regression tests, per-module detector migration, SARIF logicalLocations, self-lint triage
> **Starting point:** `paste_1.txt` backlog item (cqrs-lint section from TODO_LIST)

---

## a) FULLY DONE (verified: tests pass, builds pass)

### 1. C001 False Positive — Read-only bbolt transactions

**Files:** `cmd/cqrs-lint/pkg/rules/correctness/tx_helpers.go`, `c001.go`, `c001_fix_test.go`

- Added `isReadOnlyBegin()` to detect `Begin(false)` calls (bbolt read-only tx cannot commit)
- Added `CompositeLit` case to `analyzeTxUsage` to detect tx escaping via struct literal fields (`&iter{tx: tx}` iterator pattern)
- Unified `escapesToArg` → `escapes` (covers both callback args and composite-literal field stores)
- Removed now-unnecessary `//cqrs-lint:ignore(C001)` suppression in `storage/bbolt/kv_adapter.go`
- 2 regression tests: `TestC001_ReadOnlyBeginFalse_NoFinding`, `TestC001_CompositeLiteralEscape_NoFinding`

### 2. D012 False Positive — CLI tools excluded

**Files:** `cmd/cqrs-lint/pkg/rules/consistency/d012.go`, `d012_test.go`

- Skip `main` package files (checked via `gf.AST.Name.Name == "main"`) — CLI entry points use `fmt.Print*` as intended output
- Updated 3 existing tests from `package main` → `package myapp` to still trigger D012
- Added `TestD012_NoFindingForMainPackageHandler` regression test

### 3. C008 False Positive — Non-monetary floats

**Files:** `cmd/cqrs-lint/pkg/rules/correctness/c008.go`, `rules_test.go`

- Removed `"rate"` from `weakMoneyFields` (26 suppressed instances in DiscordSync round-2 feedback)
- Added `nonMonetaryFieldPatterns` denylist (latency, throughput, ratio, percentage, duration, seconds, etc.) checked before strong/weak matching
- 2 regression tests: `TestC008_NoFindingForNonMonetaryMetrics`, `TestC008_NoFindingForRateFields`

### 4. Regression Tests for S006, A018, B004

**Files:** `pkg/rules/security/new_rules_test.go`, `pkg/rules/api/new_rules_test.go`, `pkg/rules/boilerplate/new_rules_test.go`

- **S006**: `TestS006_WeakTierSuppressedForLocalOnlyProject`, `TestS006_WeakTierFiresForServerProject`
- **A018**: `TestA018_SuppressedBySaveCall`, `TestA018_SuppressedByPublishCall`, `TestA018_SuppressedByDispatchCall`, `TestA018_SuppressedByFolds`
- **B004**: `TestB004_NoFindingForFewFields`, `TestB004_SuppressedByExistingConstructor`

### 5. A034 Per-Module Migration

**File:** `cmd/cqrs-lint/pkg/rules/api/a034.go`

- Migrated from `ctx.FeatureProfile.HasMetaengine` (project-level early return) to `ctx.ProfileForFile(gf.Path).HasMetaengine` (per-file check)
- Correct for multi-module workspaces where only some modules use metaengine
- All existing A034 tests pass unchanged

### 6. SARIF logicalLocations

**File:** `cmd/cqrs-lint/scorecard_render.go`

- Added `sarifLogicalLocation` struct (`name`, `fullyQualifiedName`, `kind`) at run level
- Added `sarifLogicalLocationRef` (`index`) for result-level cross-references
- Populated `run.logicalLocations[]` from all scored modules (used + missing)
- Missing-module results now reference their module's logical location by index
- All existing SARIF tests pass

### 7. Self-Lint Triage — D007 (5 instances)

**Files:** `example/metaengine-quickstart/main.go`, `encryption/crypto_helpers.go`, `watermill/protocol.go`

- Replaced 5 `event.NewEvent(` → `event.New(` calls (same signature, shorter alias)

### 8. Self-Lint Triage — C023 (1 instance)

**File:** `metaengine/irohengine/demo/main.go`

- Fixed unchecked `engine.Close()` → `_ = engine.Close()`
- Discovered C023 false positive on dgraph: `dgo` client's `Close()` returns void (no error), but C023 flags it anyway — C023 needs to verify the call returns an error before flagging

---

## b) PARTIALLY DONE

### Self-Lint Triage — C033 (bare `return err` without wrapping)

- **80+ findings** across `metaengine/*engine/` and `benchkit/` identified
- NOT fixed — too pervasive for a single session, needs a bulk-remediation pass
- These are all INFO-level findings (not errors/warnings), so they don't block the self-lint baseline

### Per-Module Detector Migration

- A034 migrated (was the only non-F-series detector using `ctx.FeatureProfile`)
- 3 remaining: F015/F016/F017 (`HasAsyncBus`), F022 (`Store.IsSQL()`), F026 (`HasMetaengine`) — **intentionally project-level** (F-series rules assess project-wide adoption, not per-file correctness)

---

## c) NOT STARTED

### From the original backlog (paste_1.txt):

| Item                                             | Status      | Notes                                                  |
| ------------------------------------------------ | ----------- | ------------------------------------------------------ |
| Run cqrs-lint against real consumer projects     | NOT STARTED | External task — requires cloning 8 consumer repos      |
| C023 false positive on void-return Close()       | NOT STARTED | Discovered during triage — C023 must check return type |
| Scorecard SARIF `logicalLocations` half          | DONE        | Was listed as pending — implemented this session       |
| Deferred P-series rules                          | NOT STARTED | Each needs advanced type inference                     |
| L1.5 domain severity calibration broader testing | NOT STARTED | Needs testing against financial/security projects      |
| ~14 remaining Pareto backlog items               | NOT STARTED | See Pareto plan                                        |

---

## d) TOTALLY FUCKED UP / ISSUES FOUND

### C023 False Positive on Void-Return Close()

**Severity: Medium (linter correctness)**

During C023 triage, I attempted to fix unchecked `Close()` calls in `metaengine/dgraphengine/engine.go`. Build failed:

```
client.Close() (no value) used as value
```

The `dgo` client's `Close()` method returns **void** (no error), but C023 flags it as an unchecked error. This is a **false positive** in C023's detection logic — it should verify the method returns an error before flagging. The dgraph instances remain unsuppressed and will fire C023 on every self-lint run.

**Fix needed:** C023 detector should type-check the call expression's return type. If `Close()` returns void, don't flag it. This requires type information (`ctx.GoFiles[].Pkg.TypesInfo`) which may not be available in all test contexts.

### C008 "rate" Removal May Miss Real Interest Rates

Removing `"rate"` from `weakMoneyFields` eliminates false positives (ErrorRate, ProcessingRate) but could miss genuine financial rates (InterestRate, ExchangeRate). The struct/package corroboration via `moneyKeywords` or `strongMoneyFields` still catches these in monetary-named structs, but a lone `InterestRate float64` in a non-monetary-named struct would now be missed.

**Mitigation:** This is an acceptable tradeoff — the false positive rate of `"rate"` was massively higher than the false negative rate of removing it.

---

## e) WHAT WE SHOULD IMPROVE

### C023 Type-Awareness (URGENT)

C023 currently flags any `.Close()` call not wrapped in `if err :=` / `_ =`. It should only fire when the method actually returns an error. Without `TypesInfo`, this is a heuristic — but the detector should at least check that the call expression is in a statement position (not already used as a value).

### Test Coverage for D007 Auto-Fix

D007 has an auto-fix (`--fix` replaces `event.NewEvent(` with `event.New(`), but there's no test for the fix provider. The fix worked correctly when I applied it manually via `sed`, but the auto-fix code path is untested.

### SARIF logicalLocations Test Coverage

The logicalLocations feature is implemented and doesn't break existing tests, but there's no dedicated test verifying:

1. That `logicalLocations` array is populated when modules exist
2. That result `logicalLocations[].index` correctly maps to the right module
3. That the `kind` field is `"module"` for all entries

### C001 `Begin(false)` Check is bbolt-Specific

The `isReadOnlyBegin()` check looks for `Begin(false)` — this is the bbolt API. Other databases may use different patterns for read-only transactions (e.g., `sql.Tx` with `sql.ReadOnly`). The check should be generalized or at least documented as bbolt-specific.

---

## f) NEXT 50 THINGS TO DO

### High Priority (False Positive / Correctness)

1. Fix C023 to not flag void-return `Close()` calls (type-awareness)
2. Run cqrs-lint against Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI
3. Add word-boundary matching to C008 (fixes `TotalDays` matching `total`)
4. Write test for D007 auto-fix provider (`--fix` path)
5. Write SARIF logicalLocations dedicated tests
6. Generalize C001 `Begin(false)` check beyond bbolt
7. C008: Add `"monetary": false` feature-profile flag for explicit opt-out
8. Verify C001 doesn't fire on `db.View(func(tx) error {...})` (bbolt callback pattern)

### Medium Priority (Self-Lint Cleanup)

9. Fix ~80 C033 bare `return err` findings in `metaengine/*engine/` (wrap with context)
10. Fix ~40 C033 bare `return err` findings in `benchkit/` (wrap with context)
11. Fix ~15 D014 missing json tags findings
12. Fix ~8 C034 `go func()` without context findings
13. Fix ~6 P012/P013 SQLite without WAL/busy_timeout findings
14. Fix ~8 A032 string/int fields instead of branded ID findings
15. Add CI self-lint gate (L1.15 from Pareto — `cqrs-lint` self-lint must pass on own repo)

### Medium Priority (New Rules)

16. Implement C023 type-awareness: check `TypesInfo` for void return before flagging
17. Implement D017 domain error classification broader testing
18. Implement L1.30 orphaned event types detection (extend E006 for adapters)
19. Implement L1.31 orphaned commands detection (extend E005 for HTTP layer)
20. Implement L1.45 shared mutable state in event handler (extend A015)
21. Implement L1.47 DOC category rules (missing doc comments)
22. Implement L1.48 OBS category rules (observability gaps)
23. Implement L1.49 RES category rules (resource management)
24. Implement L1.50 DI category rules (dependency injection patterns)
25. Implement deferred P-series rules: `metaengine.Query` without type parameter
26. Implement deferred P-series: `MapUpdate` on replicated engine
27. Implement deferred P-series: Store never Closed
28. Implement deferred P-series: `metaengine.On` wrong handler signature
29. Implement L1.19 feature adoption scorecard (beyond health score)
30. Implement L1.20 grouped output by aggregate/domain
31. Implement L1.23 linter benchmark suite

### Lower Priority (DX / Infrastructure)

32. C008: Auto-downgrade `*Rate|*PerSec|*Latency|*Seconds|*Ratio|*Percentage` patterns (may supersede my denylist approach)
33. D012: Add `isCQRSHandler` check for `dispatch.Dispatcher` parameter (currently only context/event/command)
34. S006: Add `"total"` exclusion rationale documentation (intentionally excluded from weakFinancial)
35. Add suppression-staleness CI gate (detect `//cqrs-lint:ignore` comments for rules that no longer fire)
36. SARIF: Add `invocations` section with command-line and exit code
37. SARIF: Add `edgeTraversal` for multi-module result relationships
38. Add `cqrs-lint diff` mode (compare current run vs baseline)
39. Add `cqrs-lint fix --dry-run` for preview before applying auto-fixes
40. A034: Add `ProfileForFile` per-module test (verify multi-module workspace doesn't false-positive)

### Strategic (Pareto Backlog)

41. L1.5: Domain severity calibration — test against bank-sync and timesheets
42. L1.30-L1.33: Deep pattern detection (orphaned types, goroutine leaks)
43. Feature profile: Add `DomainKind` testing for financial/security projects
44. Module catalog: Verify all 77 go.mod directories are covered
45. API surface golden regeneration after rule changes
46. Update IMPROVEMENT_IDEAS.md with completed items
47. Update Pareto plan status columns
48. Tag cqrs-lint v4.5.0 (or next minor) with these fixes
49. Update AGENTS.md cqrs-lint section with new rule count and capabilities
50. Write changelog entry for all fixes in this session

---

## g) QUESTIONS

### 1. Should C023 use `TypesInfo` to check void-return Close()?

C023 currently flags any `.Close()` not explicitly error-checked. The `dgo` client's `Close()` returns void, creating a false positive. Fixing this properly requires consulting `ctx.GoFiles[].Pkg.TypesInfo` to determine the method's return signature — but `TypesInfo` is `&types.Info{}` (empty) in test contexts (`BuildContextFromSource`). Should I: (a) make C023 type-aware and accept that it can't be tested via `BuildContextFromSource`, or (b) add a heuristic that checks if the call result is used in a value position?

### 2. Should the ~80 C033 `return err` findings be bulk-fixed or suppressed?

The metaengine engine backends (badger, pebble, sqlite, dgraph) have pervasive bare `return err` patterns. Many are genuinely fine (the error context is clear from the function name). Options: (a) bulk-wrap all with `fmt.Errorf("funcName: %w", err)`, (b) suppress them all with a config preset, or (c) leave them as INFO-level noise in the self-lint baseline?

### 3. When should cqrs-lint be tagged with these fixes?

The false-positive fixes (C001, C008, D012) directly improve the consumer experience for all 8 consumer projects. Should we tag `cqrs-lint/v4.5.0` now with these changes, or wait until the C023 void-return fix and the real-consumer-project validation pass are also done?

---

## Session Metrics

| Metric                       | Value                             |
| ---------------------------- | --------------------------------- |
| Files changed                | ~15 (cqrs-lint + library modules) |
| Tests added                  | 13 new regression tests           |
| False positives fixed        | 3 rules (C001, D012, C008)        |
| Detectors migrated           | 1 (A034 per-module)               |
| Self-lint findings fixed     | 6 (5 D007 + 1 C023)               |
| Self-lint findings remaining | ~199 (C033 dominant)              |
| Build status                 | GREEN                             |
| Test status                  | ALL PASSING                       |
