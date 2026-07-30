# Status Report: S007 In-Memory Session Store Linter Rule

**Date:** 2026-07-30 17:01
**Session scope:** Implement linter rule S007 (in-memory session/token store detection) for `cmd/cqrs-lint`

---

## Executive Summary

S007 is **fully implemented, tested, and documented**. The rule fires correctly, passes all tests with `-race`, and appears in the CLI `rules` output. The auto-commit daemon committed the work (`f89c9ebf`). However, the **lint gate does not pass** due to **9 pre-existing lint issues** (none from S007), and the **benchkit soak suite is flaky** (pre-existing, machine-load-sensitive). The `nix run .#verify` gate cannot reach GREEN until these pre-existing issues are resolved.

---

## a) FULLY DONE

| # | Item | Verification |
|---|------|-------------|
| 1 | **S007 detector** (`security/s007.go`, 130 lines) | Compiles, vet clean, 7 tests pass with `-race` |
| 2 | **Two-signal conjunction heuristic** — `inmemory`/`memory` AND (`session` OR (`token`+`store`)) | Tested against false positives: CQRS event store, rate limiter bucket, test files |
| 3 | **HasServer gating** — suppressed for CLIs/batch/test contexts | `TestS007_SuppressedWithoutServer` confirms |
| 4 | **Registration** — `register.go`, `catalog_extra.go` (S007 RuleInfo), `meta_test.go` (count→117) | `TestAllDetectorsInstantiate` passes, `TestCatalogCountMatchesRegister` passes |
| 5 | **Catalog entry** — S007 in `securityRules()` with Warning/Medium severity | Visible in `cqrs-lint rules` CLI output |
| 6 | **7 unit tests** — positive (call+composite), no-server gate, test-file skip, event-store FP guard, token-bucket FP guard, empty-input | All pass with `-race -count=1` |
| 7 | **Documentation** — README (117 rules, security table with S007 row), AGENTS.md (117/9 categories), IMPROVEMENT_IDEAS (S007 struck through, summary updated) | Doc-check passes |
| 8 | **Formatting** — `gofumpt` + `goimports` clean on all S007 files | `gofumpt -l` / `goimports -l` return empty |
| 9 | **Fixed pre-existing T-series formatting** — `testrules/rules_test.go` was flagged by gofumpt/goimports; ran both `-w` | `gofumpt -l pkg/rules/testrules/` now clean |
| 10 | **Corrected stale rule counts** — discovered the real count is **117** (not 113 or 105); performance had 5 rules not 2, security now has 4 | Verified via `grep -c` on RegisterAll + catalog |

---

## b) PARTIALLY DONE

| # | Item | Status | What remains |
|---|------|--------|-------------|
| 1 | **`nix run .#verify`** | Build ✓, Vet ✓, Test ✓ (except soak), Race ✓ (cqrs-lint) | Lint step fails on **9 pre-existing issues** (see below); API-stability not reached |
| 2 | **`nix run .#verify-fast`** | Started, lint output captured | Lint step fails on same 9 pre-existing issues |

---

## c) NOT STARTED

| # | Item | Why |
|---|------|-----|
| 1 | Fix the 9 pre-existing lint issues | Out of S007 scope but blocks verify gate — see section (e) |
| 2 | S004 (PII field-level encryption) | Separate improvement idea (#54) |
| 3 | S005 (event signing disabled by default) | Separate improvement idea (#55) |
| 4 | S006 (financial data without encryption) | Separate improvement idea (#56) |

---

## d) TOTALLY FUCKED UP

Nothing in S007 itself. However:

| # | Issue | Impact |
|---|-------|--------|
| 1 | **Auto-commit daemon modified unrelated files** — `CONTRIBUTING.md`, `FEATURES.md`, `docs/STORAGE_GUIDE.md` show as modified in git status but I never touched them | Working tree has unexpected changes; these are from the daemon, not this session's S007 work |
| 2 | **Docs were already badly stale before this session** — AGENTS.md said "105 rules / 8 categories", README said "113 rules", IMPROVEMENT_IDEAS said "113 rules", but the actual count was already 117 (performance had silently grown from 2→5 rules). The T-series session apparently didn't fix AGENTS.md at all (it was at 105, not even 113). | Multiple doc files had inconsistent counts. Fixed AGENTS.md/README/IMPROVEMENT_IDEAS to 117, but **other files may still reference old counts**. |

---

## e) WHAT WE SHOULD IMPROVE

### Pre-existing Lint Issues Blocking the Gate (9 issues, ALL pre-existing — none from S007)

These are the exact lint failures from `nix run .#lint` on `cmd/cqrs-lint`:

| # | Rule | File | Issue | Fix |
|---|------|------|-------|-----|
| 1 | `gci` | (unidentified — need to trace) | Import ordering | Run `gci write` |
| 2 | `gochecknoglobals` | `catalog.go` (`allRulesCache`) | Global var | Add `//nolint:gochecknoglobals` — it's a `sync.OnceValue` cache, intentional |
| 3 | `gochecknoglobals` | `performance/p009.go:17` (`eventPayloadSuffixes`) | Global slice | Add `//nolint:gochecknoglobals` or make it a function-local |
| 4 | `gochecknoglobals` | `register.go:232` (`ruleCategoryCache`) | Global var | Add `//nolint:gochecknoglobals` — it's a `sync.OnceValue` cache, intentional |
| 5 | `gochecknoglobals` | `testrules/t007_t008.go:52` (`productionStoreSubstrings`) | Global slice | Add `//nolint:gochecknoglobals` or make it a function-local |
| 6 | `golines` | `version/v006.go:94` | Line too long | Run `golines` or break manually |
| 7 | `modernize` | `version/gomod.go:51` | `HasPrefix+TrimPrefix` → `CutPrefix` | Replace with `strings.CutPrefix` |
| 8 | `modernize` | `suppression/parser.go:177` | `strings.Split` → `strings.SplitSeq` | Replace with `strings.SplitSeq` |
| 9 | `unparam` | `testrules/helpers.go:164` | `severity` param always receives `SeverityInfo` | Remove the parameter or make it vary |

**These are all from the T-series session or older code. Fixing them is ~15 minutes of mechanical work.**

### Other Improvements

| # | Improvement | Rationale |
|---|-------------|-----------|
| 1 | **Lint gate was already RED before this session** — the T-series session committed code that fails lint (gochecknoglobals, unparam). The "Stale GREEN" anti-pattern documented in AGENTS.md strikes again. | The verify gate is only useful if run to completion every session |
| 2 | **API-surface golden doesn't track internal detector symbols** — the earlier T-series context claimed the golden was stale (missing 8 T-series symbols), but regeneration produced zero diff. The golden only tracks `cmd/cqrs-lint`'s top-level exports (`ComputeHealthScore`, `AppConfig`, etc.), not `pkg/rules/*` detectors. | The earlier session's "3 CI-breaking issues" list was partially wrong — the golden was never stale |
| 3 | **350-line file limit excludes `_test.go`** — the T-series `rules_test.go` (597 lines) does NOT violate CI. The earlier session's concern about "needs splitting" was unfounded. | Verified in `flake.nix:451`: `find . -name "*.go" -not -name "*_test.go"` |
| 4 | **benchkit soak tests are chronically flaky** — 3 tests (`TestRunSoak_Memory`, `TestRunSoak_TrendsPopulated`, `TestRunSoakJSON_RoundTrip`) fail when the machine is under load. They expect >=2 iterations in 5s but get 1. | These should use `testing.Short()` skip or longer time budgets |
| 5 | **README/IMPROVEMENT_IDEAS performance counts are stale** — performance has 5 rules (P001, P007, P008, P009, +1 more) but docs say "2". I fixed the headline to 117 total but didn't audit per-category performance accuracy. | Run `grep -c` per category and update all tables |

---

## f) Next 50 Things to Do

### Immediate (blocks verify gate)
1. Fix `gci` import ordering issue in cqrs-lint (trace the exact file)
2. Add `//nolint:gochecknoglobals` to `catalog.go` `allRulesCache`
3. Add `//nolint:gochecknoglobals` to `register.go` `ruleCategoryCache`
4. Fix `performance/p009.go` global (`eventPayloadSuffixes`)
5. Fix `testrules/t007_t008.go` global (`productionStoreSubstrings`)
6. Run `golines` on `version/v006.go`
7. Replace `HasPrefix+TrimPrefix` with `CutPrefix` in `version/gomod.go`
8. Replace `strings.Split` with `strings.SplitSeq` in `suppression/parser.go`
9. Fix `unparam` in `testrules/helpers.go` (`severity` always `Info`)
10. Re-run `nix run .#verify` to confirm GREEN

### S-series completion
11. Implement S004 (PII field-level encryption — monetary/PII field detection)
12. Implement S005 (signing module imported but signer disabled via boolean flag)
13. Implement S006 (financial fields without encryption)
14. Harden S007: scope to `store.`/`repo.` selectors instead of bare constructor names
15. Add feature-profile gating: suppress S007 for `example/` directories

### Lint health
16. Audit ALL per-category counts in README against catalog (performance says 2, has 5)
17. Audit ALL per-category counts in IMPROVEMENT_IDEAS summary table
18. Add a meta-test that verifies README/AGENTS.md rule counts match `len(RegisterAll)`
19. Add `//nolint` comments to all `sync.OnceValue` globals (consistent pattern)
20. Investigate why gochecknoglobals fires on `sync.OnceValue` (these are idiomatic caches)

### T-series cleanup (from prior session)
21. Resolve T003/B015 overlap (both fire for event projects without eventtest)
22. Gate T001/T006 on decider feature-profile usage
23. Gate T002/T005 on projection feature-profile usage
24. Decide: should T008 flag `storage/memory` usage too? (currently NOT flagged)

### benchkit soak stability
25. Add `testing.Short()` skip to `TestRunSoak_Memory`
26. Add `testing.Short()` skip to `TestRunSoak_TrendsPopulated`
27. Add `testing.Short()` skip to `TestRunSoakJSON_RoundTrip`
28. Or: increase soak time budget from 5s to 15s for reliability

### cqrs-lint improvements
29. Run `nix run .#lint` BEFORE and AFTER every rule addition to catch issues early
30. Add a CI check that `nix run .#lint` passes (currently it can be RED without anyone noticing)
31. Document the lint-rule-exemption process in AGENTS.md
32. Add `cqrs-lint doctor` output for the testing category
33. Consider grouping S004/S005/S006/S007 detection patterns into shared security helpers

### S007 hardening
34. Add test for `session.NewInMemoryStore()` (package-qualified constructor)
35. Add test for `InMemoryTokenStore{}` (token+store without "session")
36. Add test for multiple findings in one file
37. Add test for `NewMemorySessionManager()` variant naming
38. Consider detecting `map[string]*Session{}` (hand-rolled in-memory store)

### Documentation
39. Update `docs/SPAN_NAMING.md` if S007 needs tracing context
40. Add S007 to the cqrs-lint SKILL.md references if one exists
41. Write ADR for the two-signal conjunction heuristic (reusable pattern)
42. Document the feature-profile gating pattern for future security rules

### Architecture
43. Extract security helper for "volatile indicator + domain indicator" conjunction (S004/S006/S007 share the pattern)
44. Consider a `security/helpers.go` for shared AST scanning (like `testrules/helpers.go`)
45. Add suppression directive test for S007 (`//cqrs-lint:disable S007`)

### Verification
46. Run `nix run .#verify` to full GREEN
47. Run `nix run .#verify` a second time to confirm stability
48. Run `git diff` audit on daemon-modified files (CONTRIBUTING.md, FEATURES.md, STORAGE_GUIDE.md)
49. Clean up `docs/status/2026-07-30_16-13_t-series-linter-rules.md` (mark as superseded)
50. Tag cqrs-lint for release if version bump is warranted

---

## g) Questions

**Q1:** The auto-commit daemon modified `CONTRIBUTING.md`, `FEATURES.md`, and `docs/STORAGE_GUIDE.md` — files I never touched. Should I investigate these changes or trust the daemon? These appeared in `git status` as modified but are outside the S007 scope.

**Q2:** The lint gate has 9 pre-existing issues that block `nix run .#verify`. They're all from T-series/older code (not S007). Should I fix them as part of this task (to get verify GREEN), or is that out of scope for "implement S007"?

**Q3:** The README and IMPROVEMENT_IDEAS show stale per-category counts (e.g., performance says "2" but actually has 5). I fixed the headline totals (117) but not every per-category breakdown. Should I do a full per-category audit now, or defer it?

---

## Technical Details

### S007 Detection Logic

The rule uses a **two-signal conjunction** to minimize false positives:

```
Signal 1 (volatile storage):  "inmemory" OR "memory" in the identifier
Signal 2 (auth state):        "session" OR ("token" AND "store")

Match = Signal 1 AND Signal 2
Gate  = FeatureProfile.HasServer (production server context only)
```

**Why this is smart to lint:**
- Each axis independently kills false positives
- `memory.NewStore()` (CQRS event store) lacks the auth indicator → not flagged
- `NewInMemoryTokenBucket()` (rate limiter) lacks "store" → not flagged
- Construction sites are concrete AST nodes (`CallExpr`, `CompositeLit`) with exact positions
- `HasServer` gate suppresses the entire "dev server" false-positive class
- Test files are skipped

### Files Changed This Session

| File | Change | Committed? |
|------|--------|-----------|
| `security/s007.go` | NEW — S007 detector (130 lines) | ✓ `f89c9ebf` |
| `security/new_rules_test.go` | Added 7 S007 tests | ✓ `f89c9ebf` |
| `register.go` | Added `security.NewS007Detector(ctx)` | ✓ `f89c9ebf` |
| `catalog_extra.go` | Added S007 RuleInfo entry | ✓ `f89c9ebf` |
| `meta_test.go` | Count 113→117 (corrected to real value) | ✓ `f89c9ebf` |
| `README.md` | 113→117 rules, security 3→4, added S007 table row | ✓ daemon |
| `AGENTS.md` | 105/8→117/9 categories + testing | ✓ daemon |
| `IMPROVEMENT_IDEAS.md` | S007 struck through, counts updated | ✓ daemon |
| `testrules/rules_test.go` | gofumpt/goimports formatting fix (pre-existing debt) | ✓ daemon |
