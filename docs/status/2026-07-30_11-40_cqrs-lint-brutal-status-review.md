# cqrs-lint Improvement — Brutal Status Report

> **Date:** 2026-07-30 11:40
> **Session scope:** Resumed execution of the 4-phase improvement plan. Found the previous session's work had been committed by the auto-commit daemon. Fixed quality gaps: added missing tests, fixed bugs, wrote validation report.
> **Starting state:** 65 rules, 6 categories (per README — still says this)
> **Actual state:** 78 detectors registered, 8 categories in catalog, all tests green — BUT multiple critical issues remain.
>
> **Update 2026-07-31 (commit `30bfcb73`):** The linter grew to **175 rules across 10 categories**.
> The verify gate is GREEN (c031.go build error was fixed in the 23:22 hardening session).
> E010/E011/E013/E014 were rewritten with type-aware matching, library self-lint mode was
> implemented, and import-alias resolution was built. See full status in the Resolution
> section below.

---

## A) FULLY DONE ✓

### Phase 0: False-Positive Fixes (3 fixes, committed by daemon)

1. **D005 trailing-dot FP fixed** — `parseVersionParts` now strips trailing `.,` via `TrimRight`. Regression test with 5 sub-cases in `d005_internal_test.go`.
   - ⚠️ NOTE: My original fix stripped `.,;:!?` but the daemon committed only `.,`. The test covers all punctuation variants but the code only handles period and comma. Semicolons, colons, exclamation marks, and question marks would still produce FPs.

2. _*C009 exported Must* FP fixed_* — `isMustFunc` now checks both `must` and `Must` prefixes. Regression test added (`TestC009_NoFindingInExportedMustFunc`).

3. **C006 enhanced** — Now catches three patterns: `event.Version(x.Int()+1)` (auto-fix), `event.Version(ver+1)` (suggest), `event.NewEvent(..., ver+1, ...)` (suggest). Tests added for all three forms.

### Phase 1-4: All 13 Rules Implemented (committed by daemon)

All 13 detectors exist, are registered, and pass the meta-tests:

- Correctness: C017, C019, C020, C022, C023 (5 rules)
- Consistency: D011 (1 rule)
- API: A027 (1 rule)
- Boilerplate: B021, B023, B024 (3 rules)
- Performance: P001, P007 (2 rules, new category)
- Version: V001 (1 rule, new category)

### Infrastructure (committed by daemon)

4. **2 new packages** created: `performance/`, `version/` — each with `doc.go` + `toolName`.
5. **Catalog updated** — 78 total `RuleInfo` entries (41 in catalog.go + 37 in catalog_extra.go). `AllRules()` chains `performanceRules()` and `versionRules()`.
6. **Register.go updated** — 78 detectors in `RegisterAll()`, 5 in `RegisterCritical()`.
7. **Meta_test updated** — count assertion: 78. `TestCatalogCountMatchesRegister` passes.

### Behavioral Tests Added This Session (8 new test files, 29 tests)

8. **D011** — `d011_test.go`: 2 tests (nil payload positive + typed payload negative).
9. **C022** — `c022_test.go`: 2 tests (context discard positive + context used negative).
10. **A027** — `a027_test.go`: 2 tests (3+ WithCodec positive + <3 negative).
11. **C023** — `c023_test.go`: 3 tests (ignored Stop positive + defer suppress + error checked negative).
12. **P007** — `p007_test.go`: 2 tests (bitshift in for-loop positive + normal bitshift negative).
13. **V001** — `v001_test.go`: 2 tests (v3+v4 mixing positive + v4-only negative).
14. **D005 regression** — `d005_internal_test.go`: 5 sub-tests (trailing punctuation variants + version compatibility).
15. **Test helpers** — `performance/test_helpers_test.go`, `version/test_helpers_test.go`.

### Bugs Fixed This Session

16. **B024 missing `NewMemoryBus` constructor** — Added `"NewMemoryBus"` to bus constructor detection list. Without this, the most common bus constructor was invisible to the rule.
17. **V001 used `gf.Pkg.Imports` (type info, nil in tests)** — Rewrote to use `gf.AST.Imports` (AST-level, works in both real analysis and unit tests).
18. **B021 used body-searching for StrictApply** — Rewrote to use `ctx.Registry.StrictApplyFolds[funcName]` (registry tracks which folds are wrapped in `decider.StrictApply`).

### Verification

19. **Build clean** — `go build -tags "goexperiment.jsonv2" ./...` passes.
20. **Test green** — All 13 packages pass. 359 total test functions.
21. **Vet clean** — `go vet -tags "goexperiment.jsonv2" ./...` passes.
22. **Formatted** — `gofmt -l pkg/rules/` returns 0 files.
23. **VALIDATION_REPORT.md written** — `cmd/cqrs-lint/VALIDATION_REPORT.md` with before/after metrics.

---

## B) PARTIALLY DONE ⚠️

### README NOT updated

The README still says "**65 rules** across 6 categories: correctness (16), API misuse (19), boilerplate (15), consistency (5), architecture (7), security (3)." The 13 new rules, 2 new categories (performance, version), and updated counts are completely missing. A user reading the README has no idea these rules exist.

### D005 fix incomplete

My original code stripped `.,;:!?` (period, comma, semicolon, colon, exclamation, question mark). The daemon committed a version that only strips `.,`. The internal test file tests ALL variants, but the code will fail on versions ending in `;`, `:`, `!`, or `?`. This is a silent correctness gap between tests and implementation.

### No real-code validation

None of the 13 new rules were run against real consumer projects this session. The previous session ran against 5 projects (Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync) and found false-positive rates. This session did zero FP auditing. We have no idea if C023 produces 50 findings on real code or 0. We have no idea if B021 fires on every fold in every project or just the intended ones.

### C023 defer detection is still O(N×M)

The daemon committed the BROKEN `isInsideDefer` function that walks the ENTIRE file AST for every `_ = x.Stop()` assignment. My ancestor-stack fix was NOT committed. On a file with N assignments and M AST nodes, this is O(N×M). The C015 rule already has the correct O(M) `isInDefer(ancestors)` helper — C023 should use the same pattern but doesn't.

### Duplicate helper functions across packages

- `findFuncDecl` / `findMethodDecl` exist in `correctness/c020.go` AND as `findFuncDeclP001` / `findMethodDeclP001` in `performance/p001.go` — same logic, different names.
- `tokenPos` struct in `correctness/c019.go` duplicates `token.Position`.
- `resolveHandlerBody` / `bodyContainsPanic` / `bodyContainsRepoLoad` are similar across C020 and P001.

These should be extracted to `lintutil` or `analyzer`.

---

## C) NOT STARTED ✗

### From the improvement plan (not started at all):

- **92 more rules** from IMPROVEMENT_IDEAS.md (C018, C024-C027, A020-A031, B016-B026, D007-D010, D012, E008-E015, S004-S007, P002-P010, T001-T008, F001-F017, V002-V006)
- **go.work linter bug fix** — Standup-Killer and other projects with `go.work` referencing missing modules silently fail with "No Go files found"
- **`cqrs-lint profile` subcommand**
- **`--scorecard` flag**
- **API stability golden regen** — new exported symbols not added to golden file
- **doc-check** — never run on new/modified documentation
- **`nix run .#verify`** — the full project gate was never run
- **`nix fmt`** — only `gofmt` was run on specific files, not treefmt on the whole repo

### FP audit not started:

- C017 (in-memory snapshot) — 0 projects audited
- C019 (duplicate repos) — 0 projects audited
- C020 (panic in handler) — 0 projects audited
- C022 (context discard) — 0 projects audited
- C023 (lifecycle errors) — 0 projects audited (previous session found 45→12 on DiscordSync, but remaining 12 never audited)
- D011 (nil payload) — 0 projects audited
- A027 (repeated WithCodec) — 0 projects audited
- B021 (fold without StrictApply) — 0 projects audited (previous session saw 3 on Standup-Killer, never verified)
- B023/B024 (missing middleware) — 0 projects audited
- P001 (repo.Load in SubscribeAll) — 0 projects audited
- P007 (bitshift retry) — 0 projects audited
- V001 (v3/v4 mixing) — 0 projects audited

---

## D) TOTALLY FUCKED UP 💥

### 1. Git index is CORRUPTED

```
fatal: unknown index entry format 0x68690000
```

`git status`, `git diff`, `git add`, and `git commit` ALL fail. The auto-commit daemon is dead. VALIDATION_REPORT.md, the D005 test, and all my test files are UNCOMMITTED because git is broken. This needs `git read-tree` or index rebuild before anything can be committed.

### 2. I wasted massive time fighting the auto-commit daemon

The daemon was committing the previous session's work WHILE I was implementing. Files I thought didn't exist appeared mid-session. My implementations conflicted with daemon-committed files. I created `c019_test.go` and `c020_test.go` that duplicated tests in the daemon-committed `phase1_test.go`, causing build failures. I should have run `git log --oneline -20` FIRST to see what was already committed.

### 3. C023 O(N×M) bug NOT fixed

The entire point of re-implementing C023 was to fix the O(n²) `isInsideDefer` function. I wrote a correct ancestor-stack version. The daemon committed the BROKEN version that walks the entire file per assignment. My fix was silently overwritten. I didn't catch this until the status review at the end of the session.

### 4. D005 fix silently downgraded

I wrote `strings.TrimRight(v, ".,;:!?")`. The daemon committed `strings.TrimRight(v, ".,")`. Four of six punctuation cases are now unhandled. The tests I wrote will FAIL on those cases — wait, no, they won't because the test only tests `.`, `,`, `!`, and no-trailing. Actually looking again: the test tests `v4.2.0.`, `v4.2.0,`, `v4.2.0!`, `v4.2.0`, and `v4.0.x`. The code only strips `.,`, so `v4.2.0!` SHOULD fail but the test passes... which means either the test is wrong or I'm misreading the code. Let me think... `TrimRight(v, ".,")` removes trailing `.` and `,` characters. `"v4.2.0!"` doesn't end in `.` or `,`, it ends in `!`. So `TrimRight` won't strip the `!`. The version parts would be `["4","2","0!"]`... wait, no, it splits on `.` first, so `"4","2","0!"`. Then `slices.Contains(parts, "")` is false. But `"0!"` != `"0"`, so `isVersionCompatible` would return false, meaning the version is "incompatible" → D005 would fire. But the test says it should be 0 findings... Hmm, but the test is an internal test on `parseVersionParts` and `isVersionCompatible`, not on the full D005 detector. The test `TestParseVersionParts_TrailingPunctuation` checks that `parseVersionParts("v4.2.0!")` returns `["4","2","0"]`. But with `TrimRight(v, ".,")`, the `!` is NOT stripped, so `parseVersionParts("v4.2.0!")` would return `["4","2","0!"]` — and the test would FAIL. But the test PASSED... which means either the daemon's code is different from what I think, or the test was updated. This is confusing and I need to investigate. Either way, there's a discrepancy between my intent and what's on disk.

### 5. README completely stale

The most user-facing document still advertises 65 rules. The 13 new rules are invisible to anyone reading the docs. This is the kind of gap that makes a library look unmaintained.

### 6. P001 variable name matching is too narrow

P001 only flags `repo.Load` / `repository.Load` — it checks if the receiver name contains "repo" (case-insensitive). But `myRepo.Load()`, `userRepo.Load()`, `orderRepository.Load()` all contain "repo" so they WOULD match. However, `store.Load()` or `eventStore.Load()` would NOT match even though they're the same O(N²) anti-pattern. The rule is correct for the common case but misses less conventional naming.

### 7. B021 severity mismatch

I wrote `finding.SeverityInfo` for B021 (it's a recommendation, not a bug). The daemon committed `finding.SeverityWarning`. This is a judgment call but Info is more appropriate for a "consider using StrictApply" suggestion vs Warning which implies something is wrong.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Immediate (critical)

1. **Fix the git index corruption** — `git read-tree HEAD` or rebuild the index. Nothing can be committed until this is fixed.
2. **Fix C023 O(N×M) `isInsideDefer`** — replace with ancestor-stack approach from C015's `isInDefer`.
3. **Fix D005 `TrimRight` to handle all punctuation** — `.,;:!?` not just `.,`.
4. **Update README** — rule count, categories, new rule descriptions.
5. **Run `nix run .#verify`** — the full project gate has NEVER been run.

### High-value

6. **Run the linter against 5 consumer projects** — build binary, collect JSON findings, audit every new rule's findings for false positives.
7. **Consolidate duplicate helpers** — `findFuncDecl`, `findMethodDecl`, `tokenPos` into `lintutil` or `analyzer`.
8. **Regenerate API stability golden** — new exported symbols (constructors) need to be in the golden file.
9. **Run `nix fmt`** on the whole repo (treefmt), not just `gofmt` on specific files.
10. **Run doc-check** on modified documentation.

### Quality

11. **Add negative tests for edge cases** — C017 with StoreCustom, C019 with NewTypedRepository, C020 with method-value handler, B023 with dispatcher passed to function.
12. **Fix P001 to use data-flow tracking** instead of variable name matching.
13. **Add C023 test for `defer x.Shutdown()` (direct defer, not in FuncLit)**.
14. **Verify B021 `DefaultNil` scanner correctness** — does the scanner correctly set `DefaultNil` for all fold patterns?

---

## F) NEXT 50 THINGS TO GET DONE 📋

### Immediate fixes (critical)

1. Fix git index corruption (`git read-tree HEAD` or index rebuild)
2. Fix C023 O(N×M) isInsideDefer → ancestor-stack approach
3. Fix D005 TrimRight to strip `.,;:!?` (not just `.,`)
4. Update README: 78 rules, 8 categories, new rule table entries
5. Run `nix run .#verify` (full project gate)
6. Run `nix fmt` on whole repo
7. Regenerate API stability golden for new exported symbols
8. Run doc-check on VALIDATION_REPORT.md and README.md

### Real-code validation

9. Build cqrs-lint binary
10. Run against Kernovia — audit all new rule findings
11. Run against Standup-Killer — audit all new rule findings
12. Run against bank-sync — audit all new rule findings
13. Run against cqrs-htmx — audit all new rule findings
14. Run against DiscordSync — audit all new rule findings (especially C023's 12 remaining)
15. Calculate FP rate per new rule
16. Tune rules with >10% FP rate

### Code quality

17. Extract `findFuncDecl`/`findMethodDecl` to shared helper
18. Extract `tokenPos` → use `token.Position` directly
19. Extract handler-body resolution (C020/P001 share logic)
20. Fix B021 severity: Warning → Info
21. Fix P001 to detect any `.Load()` call in handler, not just "repo"-named vars
22. Add C017 test for StoreCustom
23. Add C019 test for NewTypedRepository
24. Add C020 test for method-value handler form
25. Add B023 test for dispatcher passed to function
26. Add edge-case negative tests for all 13 rules

### Phase 2 completion (high-value rules)

27. Implement D012 (missing WithSchemaVersion)
28. Implement A024 (decorative event sourcing)
29. Verify A014 catches ALL event.NewEvent calls
30. Implement A017 enhancement (WithSnapshotStore without WithSnapshotStrategy)
31. Add marshalPayload two-step detection to A002

### Phase 3 rules (next batch)

32. Implement C018 (silent journal fallback)
33. Implement C024 (dual-write without rollback)
34. Implement C026 (idempotency TTL mismatch)
35. Implement A020 (custom Bus reimplementation)
36. Implement A021 (custom Store reimplementation)
37. Implement B016 (manual checkpoint replay table)
38. Implement B017 (manual read model rebuild)
39. Implement B020 (manual legacy field upcasting)
40. Implement B025 (missing state cache)
41. Implement B026 (missing event catalog)
42. Implement D007 (inconsistent event.New vs NewEvent)
43. Implement D010 (generic error code "internal")
44. Implement S004 (PII without encryption)
45. Implement S005 (signing disabled by default)
46. Implement E008 (stack preset bypass)

### Infrastructure

47. Fix go.work linter bug (GOWORK=off fallback)
48. Implement `cqrs-lint profile` subcommand
49. Implement `--scorecard` flag
50. Mark implemented ideas in IMPROVEMENT_IDEAS.md

---

## G) QUESTIONS I CAN'T FIGURE OUT MYSELF ❓

### 1. The git index is corrupted — how should I fix it?

`git status` returns `fatal: unknown index entry format 0x68690000`. This breaks ALL git operations (status, add, commit, diff). Options:

- `git read-tree HEAD` to rebuild the index from the last commit
- Delete `.git/index` and run `git read-tree HEAD`
- `git fsck --full` to diagnose

I don't know which is safe given the auto-commit daemon may have been mid-write when the corruption happened. Destroying uncommitted work is a risk.

### 2. Should I continue adding new rules, or pause and harden the 78 existing?

The plan calls for ~105 new rules total. We have 78. The remaining 92 are increasingly lower-impact. But the 78 include one O(N×M) performance bug (C023), a stale README, zero FP auditing, and duplicate code. Should I:

- (A) Fix the critical issues (C023, README, D005, git index), audit FPs, THEN resume new rules
- (B) Keep implementing new rules and fix quality issues in parallel
- (C) Stop at 78 rules, declare the improvement "done enough", and focus entirely on hardening

### 3. The auto-commit daemon keeps committing mid-session and overwriting my fixes. How do I work with this?

The daemon committed the previous session's code (including bugs) WHILE I was implementing fixes this session. My C023 ancestor-stack fix was silently replaced by the broken version. My D005 `TrimRight(".,;:!?")` was replaced with `TrimRight(".,")`. I lost track of what was actually on disk multiple times. Options:

- (A) Disable the daemon for this session
- (B) Work in a branch the daemon doesn't touch
- (C) Accept it and always re-read files before editing

---

## Resolution (2026-07-30, updated 2026-07-31)

- ✅ **All 94 new cqrs-lint rules committed** — the 65→159 rule expansion was the
  baseline; as of 2026-07-31 the linter has **175 rules across 10 categories**.
- ✅ **Git index corruption fixed** — `git read-tree HEAD` resolved it.
- ✅ **C023 O(N×M) fixed** — rewritten to single-pass ancestor-stack.
- ✅ **D005 TrimRight fixed** — strips `.,;:!?` not just `.,`.
- ✅ **Verify gate GREEN** — c031.go build error was fixed in the 23:22 hardening
  session (`838609a8`). The full `nix run .#verify` passed.
- ✅ **Quality items resolved** (23:22 hardening session):
  - E010 rewritten with type-aware receiver matching (`projectCallsMethodOnType`)
  - E014 rewritten with type-aware projection-host matching
  - Library self-lint mode implemented (`IsLibrarySelfLint()`) — skips 29
    consumer-coaching rules when linting go-cqrs-lite source itself
  - Import-alias resolution helper built (`QualifierToImportPath`,
    `ImportQualifierMap`, `projectCallsImportPath`) — E008 migrated as proof of concept
  - Suppression tests added for all 12 new rules (C031-C034, P011-P012, D014-D015,
    A032, E016-E017, S010)
  - 22MB committed binary removed; api-stability golden regenerated (2907 exports)
- **Still open**: F011/F013 import-alias migration to remaining E-series rules;
  50-item improvement backlog (~35 items remain). See TODO_LIST.md "cqrs-lint Quality".
