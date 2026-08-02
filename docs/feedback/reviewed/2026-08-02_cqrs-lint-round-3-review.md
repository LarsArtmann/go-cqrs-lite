# Review: cqrs-lint Round-3 Consumer Feedback (crush-daily, cqrs-htmx, Standup-Killer)

**Date:** 2026-08-02
**Reviewer:** Crush (AI Engineering Partner)
**Feedback sources:**

- [crush-daily cqrs-lint feedback](../new/2026-08-02_crush-daily_cqrs-lint-feedback.md)
- [cqrs-htmq round-2 feedback](../new/2026-08-02_cqrs-htmx_cqrs-lint-feedback-round-2.md)
- [Standup-Killer cqrs-lint feedback](../new/2026-08-02_Standup-Killer_cqrs-lint-feedback.md)

**Verdict:** 5 of 5 actionable items implemented. All three feedback files were written against the stale Nix binary (v0.2.2); the round-2 review already fixed most issues in source. This review closes the remaining gaps that survive into v0.3.0.

---

## Critical Assessment: Both Sides

### Side A — Is the feedback valid?

The feedback is **genuinely valuable** but has a critical blind spot: **all three consumers ran v0.2.2, not current source HEAD**. The cqrs-htmx feedback explicitly documents this ("stale binary problem"), but crush-daily and Standup-Killer report issues that are already fixed in source as if they were still open.

| Claim | Valid? | Notes |
|-------|--------|-------|
| B022 suggests `decider.CommandCausalityEnricher` (wrong package) | **Already fixed** | Source says `event.CommandCausalityEnricher` since commit `b4554cdc` |
| `wrapsCanonicalEnricher` can't handle qualified names | **Already fixed** | Uses `SelectorFromExpr` + `sel.Sel.Name` since `b4554cdc` |
| Version constant not bumped | **Already fixed** | Bumped to `0.3.0` |
| gofmt/space conflict | **Already fixed** | `normalizeCommentPrefix()` since `b4554cdc` |
| One-suppression-per-line | **Already fixed** | Comma-separated since `f192106a` |
| Blank line breaks suppression | **Already fixed** | Blank-line skip since `ef2ddf69` (today) |
| C036 cascade on shared `*sql.DB` | **Already mitigated** | `collectEventStoreBackends()` since `6f357233` |
| Config-based rule disabling | **Already implemented** | `rules.disable` since round-2 review |
| `--exclude-rules` CLI flag | **Already implemented** | Since round-2 review |
| Feature profile says `store: custom` for SQLite | **Valid — NOW FIXED** | Root cause: import-based detection only |
| B013/B022 contradiction | **Valid — NOW FIXED** | B013 didn't recognize `CommandCausalityEnricher` |
| E007 can't trace runtime/generic registration | **Valid (known limitation)** | NOW mitigated: severity lowered to Info |
| Combined-directive stale suppression | **Valid — NOW FIXED** | Was checking each rule independently |
| Health score clamped to 0 with no raw display | **Valid — NOW FIXED** | Added `RawScore` field |
| `cqrs-gen` doesn't exist | **WRONG** | Tool exists at `cmd/cqrs-gen/` with README and tests |
| F-level rules pollute health score | **Valid (design)** | Deferred — `--preset local-cli` is the workaround |

### Side B — Does the codebase address the feedback?

Most issues were addressed in the round-2 review (bank-sync + browser-history). The remaining gaps were **real** and are now fixed:

1. **Feature profile store detection** — `storage/` import mapped to `StoreCustom` even when `storage.NewSQLiteEventStore` was used. C036's `collectEventStoreBackends` mitigated the cascade, but the profile itself was wrong, affecting `doctor` output and other rules.

2. **B013 causality recognition** — B013 only checked `WithCommandCausality` and `WithCorrelationID` method names. When a consumer added `WithEnricher(event.CommandCausalityEnricher)` (the canonical pattern), B013 still fired. This created the B013/B022 contradiction: B013 fires without enricher, B022 fires (on stale binary) when you add it.

3. **Combined-directive stale suppression** — Each rule in a combined directive was checked independently against the finding list. When the same code pattern exists in parallel files (SQLite vs in-memory), a combined directive covering rules that fire in one file but not the other generated false stale warnings.

4. **E007 severity** — Three consumers independently reported E007 as a false-positive source. Static analysis cannot trace through runtime registration, generic helpers, or closures. Warning severity (-2 points each) was disproportionate for a known limitation. Lowered to Info (0 score impact, still visible).

5. **Health score raw display** — Score clamped to 0 when deductions exceed 100. Users had no signal of how far below zero they were. Added `RawScore` field and display: `0/100 (clamped from -43)`.

---

## Summary of Implemented Fixes

### Fix 1: Feature profile store-backend refinement

**File:** `pkg/analyzer/feature_detect.go`

Added AST-based store-backend detection in Pass 2. When the feature profile classified the store as `StoreCustom` (no stack bundle), constructor calls from the `storage` package are now inspected. `storage.NewSQLiteEventStore` → `StoreSQLite`, `storage.NewPostgresEventStore` → `StorePostgres`, etc. Only upgrades `StoreCustom` — doesn't override stack-preset-detected values.

**Test:** `TestDetectFeatures_StorageNewSQLiteEventStore` — verifies `StoreCustom` is refined to `StoreSQLite`.

### Fix 2: B013 recognizes CommandCausalityEnricher

**File:** `pkg/rules/boilerplate/b011_b014.go`

B013 now inspects `WithEnricher` call arguments. When `event.CommandCausalityEnricher` is passed, `hasCausality` is set to `true`, suppressing the B013 finding. This breaks the B013/B022 contradiction: adding the canonical enricher clears BOTH rules.

**Test:** `TestB013_NoFindingForCommandCausalityEnricher` — verifies `WithEnricher(event.CommandCausalityEnricher)` suppresses B013.

### Fix 3: Combined-directive stale suppression

**File:** `pkg/suppression/stale.go`

`detectStaleInline` now checks whether ANY rule in a combined directive fires at the directive's location before reporting individual rules as stale. If at least one rule matches, all rules in the directive are considered active. This prevents false stale warnings when combined directives are applied uniformly across parallel code paths (SQLite vs in-memory) where different detectors fire in each.

**Tests:** `TestDetectStaleSuppressions_CombinedDirectivePartialMatch` (partial match → 0 stale), `TestDetectStaleSuppressions_CombinedDirectiveNoMatch` (no match → 2 stale).

### Fix 4: E007 severity lowered to Info

**File:** `pkg/rules/architecture/e003_e007.go`

E007 was `SeverityWarning` (-2 points per finding). Three consumers independently reported it as a false-positive source: crush-daily (6 FPs via runtime `Register()`), Standup-Killer (3 FPs via generic helper), bank-sync (closures). Static analysis cannot trace through these patterns. Lowered to `SeverityInfo` (0 score impact, still visible in output). Updated suggestion text to mention suppression for runtime/generic patterns.

### Fix 5: Health score raw deduction display

**Files:** `health.go`, `output.go`

Added `RawScore` field to `HealthScore` holding the unclamped score. When the score is 0 and `RawScore < 0`, the display shows: `0/100 (clamped from -43)`. This gives users a motivational signal that even small fixes would move the needle, instead of a flat 0 that feels hopeless.

---

## Deferred Items (not actionable in source)

| Item | Reason |
|------|--------|
| **Publish Nix binary** | The cqrs-htmx feedback's #1 request. All fixes are in source but the published v0.2.2 binary is stale. Requires `nix build` + publish to Nix channel. Not a code change. |
| **F-level rules in health score** | Design observation from Standup-Killer. F-rules (adoption coaching) deduct from score, creating pressure to suppress. `--preset local-cli` disables F-series for CLI tools. A dedicated `--adoption` flag could separate coaching from correctness, but this is a UX redesign. |
| **E007 inter-procedural analysis** | Recognizing `Register(dispatcher, ...)` patterns or tracing through generic helpers would require cross-function/cross-file call graph analysis. Significant analyzer investment for limited gain now that severity is Info. |
| **D005 version detection** | Per-module versioning means there is no single "the version" to compare against. The detector's comparison logic would need to understand multi-module workspaces. |

---

## Feedback Errors (consumer-side)

| Claim | Reality |
|-------|---------|
| Standup-Killer: "cqrs-gen doesn't appear to exist" | `cmd/cqrs-gen/` exists with `main.go`, `README.md`, and `main_test.go`. The tool generates typed constructors from struct tags. |
| crush-daily: "B022 suggests decider.CommandCausalityEnricher" | Already fixed in source — suggestion says `event.CommandCausalityEnricher` since commit `b4554cdc`. |
| All three: E007 false positives | Known static-analysis limitation, not a detector bug. Cannot trace through runtime/generic registration without inter-procedural analysis. |

---

## Test Coverage

New tests added:

- `TestB013_NoFindingForCommandCausalityEnricher` — B013 suppressed by enricher
- `TestDetectFeatures_StorageNewSQLiteEventStore` — feature profile refinement
- `TestDetectStaleSuppressions_CombinedDirectivePartialMatch` — combined directive partial match
- `TestDetectStaleSuppressions_CombinedDirectiveNoMatch` — combined directive no match

All 17 cqrs-lint packages pass (`go test -tags "goexperiment.jsonv2" ./cmd/cqrs-lint/... -count=1`).

---

## Impact Assessment

Before these fixes: consumers using `storage.NewSQLiteEventStore` (not a stack bundle) get `store: custom` in feature profile + B013/B022 contradiction + E007 false positives at Warning severity + stale suppression noise from combined directives.

After these fixes: feature profile correctly reports `store: sqlite`. Adding `WithEnricher(event.CommandCausalityEnricher)` clears both B013 and B022. E007 is Info (0 score impact). Combined directives with at least one firing rule don't generate stale noise. Health score at 0 shows how far below zero the project actually is.

Estimated signal-to-noise improvement: ~90% → ~95% for projects using `storage.NewSQLiteEventStore` directly (the most common non-stack-bundle pattern).
