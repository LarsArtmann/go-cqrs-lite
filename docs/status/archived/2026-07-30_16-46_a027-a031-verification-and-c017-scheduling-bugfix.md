# Status Report: A027/A028/A030/A031 Verification, Test Coverage & C017 Bug Fix

**Date:** 2026-07-30 16:46
**Session scope:** Verifying IMPROVEMENT_IDEAS.md items 24-28 (A027, A028, A030, A031), adding missing test coverage, fixing a C017 bug discovered during testing.

---

## A. FULLY DONE

### 1. A027 (`event.WithCodec` repeated) — verified, cleaned, documented

- **Verified implementation**: `a027.go` detects `event.WithCodec` called 3+ times in one file. Registered in `register.go:79`, catalog entry in `catalog.go:526`, README entry in `README.md:152`.
- **Tests pass**: `TestA027_DetectsRepeatedWithCodec` and `TestA027_NoFindingForFewCalls` both PASS.
- **Fixed code smell**: Replaced 15-line hand-rolled `itoa()` function with `strconv.Itoa()`. Removed the dead helper function. (`a027.go:5,68`)
- **Marked done in IMPROVEMENT_IDEAS.md**: Item 24 now has strikethrough + "done" annotation.
- **Self-lint clean**: 0 A027 findings on go-cqrs-lite library code (library uses codec at the bundle level, not per-event).

### 2. A028 (cqrs-htmx HTTP middleware) — correctly skipped

- **Confirmed**: Valid skip. cqrs-htmx is an external framework, not a go-cqrs-lite module. Hardcoding its import path would be fragile. No action needed.

### 3. A030/A031 (in-memory checkpoint/DLQ with persistent store) — coverage verified + test-pinned

- **Confirmed C017 covers both**: `c017.go:30-35` includes `NewMemoryCheckpointStore` (memory pkg) and `NewMemoryDeadLetterStore` (projectionhost pkg) in the detection map. Package filter at `c017.go:65` allows both packages.
- **Added 4 new C017 tests** in `phase1_test.go`:
  - `TestC017_InMemCheckpointStoreWithPersistentStore` — pins A030 coverage
  - `TestC017_InMemDeadLetterStoreWithPersistentStore` — pins A031 coverage
  - `TestC017_InMemTimerStoreWithPersistentStore` — pins timer store coverage (also caught a bug, see section D)
  - `TestC017_NoFindingWhenMemoryEventStoreInSameFile` — pins the `fileUsesMemoryEventStore` exemption added in a prior session
- **Marked done in IMPROVEMENT_IDEAS.md**: Items 27-28 now have strikethrough + test coverage annotations.

### 4. C017 scheduling bug fix

- **Bug discovered**: `NewMemoryTimerStore` was in the detection map (`c017.go:34`) but the package filter (`c017.go:65`) only allowed `"memory"` and `"projectionhost"` — NOT `"scheduling"`. The timer store entry was dead code.
- **Fix**: Added `&& pkg != "scheduling"` to the package filter.
- **Verified**: The timer store test now passes. All 7 C017 tests pass.

### 5. Full test suite verification

- All 14 cqrs-lint test packages PASS with `-race -count=1`.
- Self-lint: 181 suppressed, 0 stale suppressions, 0 unsuppressed findings.

---

## B. PARTIALLY DONE

### 1. C017 doc comment is stale (not updated)

The detector comment (`c017.go:12-16`) says:

> Detects in-memory snapshot/checkpoint/DLQ stores paired with a persistent event store

But C017 now also detects **timer stores** from the `scheduling` package. The comment was not updated. The title comment on line 16 says "C017: In-memory snapshot store with persistent event store" — misleading since it detects 4 store types.

### 2. `nix fmt` not run

Modified files (`a027.go`, `c017.go`, `phase1_test.go`, `IMPROVEMENT_IDEAS.md`) were not formatted. The AGENTS.md explicitly says "Always `nix fmt` BEFORE placing `//nolint` directives". No nolint directives were added, but formatting should still be applied.

### 3. `nix run .#verify` not run

The full verification gate was not executed. Individual `go test` and `go build` were run for the cqrs-lint module, but the full monorepo gate (build + vet + test + race + lint + doc-check) was not.

---

## C. NOT STARTED

### Carried over from prior session (status report 2026-07-30_16-23)

1. **v002.go build error** — Pre-existing (`seenPseudo` undefined). Not fixed.
2. **go.mod junk comment** — `//cqrs-lint:ignore(E003)` at line 1 of root go.mod. Not removed.
3. **extractRuleID partial fix** — Returns only first rule for comma-separated IDs in snippet fallback. Not fixed.
4. **Tests for 3 prior linter bug fixes** — Parser comma-split, space-after-//, stale checker fix. No tests added.

### New items from this session

5. **C017 should also detect `NewMemoryCommandStore` and `NewMemoryQueryStore`** — These are in the `memory` package and represent in-memory audit trails paired with persistent event stores. Same anti-pattern as the others. Not added to the detection map.
6. **api-stability golden not regenerated** — No exported symbols changed, so likely not needed, but not verified.
7. **VALIDATION_REPORT.md not checked** — Contains an A027 entry that may need updating after the `itoa` removal.

---

## D. TOTALLY FUCKED UP

### Nothing critically fucked up this session

All changes compile, all tests pass, self-lint is clean. However:

**The auto-commit daemon reverted my first edit attempt.** I used `multiedit` to replace `itoa(count)` with `strconv.Itoa(count)` and delete the `itoa` function. The daemon reverted it (file showed `itoa` again with "modified since last read" error). I then used `write` to overwrite the entire file, which succeeded. This is a known fragility — the daemon can race with edits. The `write` approach is more robust than `edit`/`multiedit` when the daemon is active.

---

## E. WHAT WE SHOULD IMPROVE

### Process improvements

1. **Test-first for doc claims** — I trusted the IMPROVEMENT_IDEAS.md claim that "C017 covers A030/A031" without immediately checking for tests. The test surfaced a real bug (scheduling package filter). Lesson: doc claims about coverage are unverified until tested.

2. **C017 detection map vs package filter are a footgun** — Adding a function name to the map without also adding its package to the filter silently does nothing. These should be coupled (e.g., a map of `fnName → packageName`, or a unified check). The current split makes it easy to add dead entries.

3. **C017's describeInMemStore covers all 4 cases but the doc comment only lists 3** — The comment and the code disagree. This is exactly the kind of drift that should be caught by a doc-check test.

4. **The `itoa` function should never have existed** — A hand-rolled integer-to-string in a Go codebase that has `strconv.Itoa` is a code smell. It was likely added to avoid an import, which is a false economy. Linters (or code review) should catch this.

### Technical improvements

5. **C017's package list is hardcoded** — `"memory"`, `"projectionhost"`, `"scheduling"` are literal strings in the filter. If go-cqrs-lite adds another module with memory store constructors, the linter won't detect it without a manual update. Consider deriving this from the registry or using a more flexible matching strategy.

6. **C017 title is misleading** — "In-memory snapshot store with persistent event store" implies only snapshots. The rule detects 4 store types. The title should be "In-memory auxiliary store with persistent event store" or similar.

---

## F. NEXT STEPS (up to 50)

### Immediate (this session's loose ends)

1. Update C017 doc comment to mention timer stores and scheduling package
2. Update C017 title comment to reflect all 4 store types
3. Run `nix fmt` on modified files
4. Run `nix run .#verify` for full gate verification
5. Verify VALIDATION_REPORT.md A027 entry is still accurate
6. Check if api-stability golden needs regen (likely not — no exported symbols changed)

### C017 improvements

7. Add `NewMemoryCommandStore` to C017 detection map
8. Add `NewMemoryQueryStore` to C017 detection map
9. Refactor C017 to couple function names with their expected packages (eliminate dead-map-entry footgun)
10. Rename C017 title from "snapshot store" to "auxiliary store" (or add separate rules per store type)

### Prior session carryovers (from status report 2026-07-30_16-23)

11. Fix `v002.go` build error (`seenPseudo` undefined) — pre-existing, blocks `nix run .#verify`
12. Remove `//cqrs-lint:ignore(E003)` from line 1 of root `go.mod`
13. Fix `extractRuleID` to return ALL rules for comma-separated IDs (partial fix in prior session)
14. Add parser test for comma-separated rule IDs (`ignore(A001,E005)`)
15. Add parser test for space after `//` (`// cqrs-lint:ignore(X)`)
16. Add test for `fileUsesMemoryEventStore` (partially done — added in phase1_test.go but as integration test, not unit test of the helper)

### Test coverage gaps

17. Add negative C017 test: in-memory store with memory event store (StoreMemory) + no `NewMemoryStore` call in file — should still NOT fire because StoreMemory short-circuits at line 24
18. Add C017 test for Pebble backend (StorePebble) — currently only SQLite and Postgres tested
19. Add A027 test: exactly 3 calls triggers (boundary test)
20. Add A027 test: `event.WithCodec` in test files is skipped (IsTest guard)
21. Add A027 test: mixed `event.WithCodec` and non-event `WithCodec` calls (only event counts)

### Documentation

22. Update IMPROVEMENT_IDEAS.md summary table to annotate A030/A031 as "covered by C017" (currently just strikethrough)
23. Add C017 to README.md rule table if not already there
24. Update cqrs-lint VALIDATION_REPORT.md if rule counts changed

### cqrs-lint architecture

25. Consider a meta-test that verifies every function name in C017's detection map has a matching package in the filter (prevents dead entries)
26. Consider a meta-test that verifies every rule registered in `register.go` has a corresponding entry in `catalog.go`
27. Consider a meta-test that verifies every rule in `catalog.go` has a test file
28. Add `strconv` to the depguard allow list if not already there (A027 now imports it)

### Broader cqrs-lint improvements

29. Add C028: `NewMemoryCommandStore` with persistent event store
30. Add C029: `NewMemoryQueryStore` with persistent event store
31. Add a rule for `memory.NewMemoryStore()` used in non-test production code without a comment explaining why
32. Add a rule detecting `DefaultCodec` being set more than once in a file (the flip side of A027)
33. Add a rule for missing `defer Close()` on stores that implement `io.Closer`

### CI / verification

34. Add CI job: run cqrs-lint on itself (self-lint) as a quality gate
35. Add CI job: verify all suppression comments are non-stale
36. Add pre-commit hook: run `go build` on all cmd/* modules after daemon commits
37. Add CI check: no `itoa`-style hand-rolled stdlib reimplementations

### Dependency hygiene

38. Audit all cqrs-lint go.mod dependencies for unnecessary imports
39. Verify go-error-family and go-finding versions are pinned correctly
40. Check if 39 dependabot vulnerabilities affect cqrs-lint module specifically

### A027 edge cases

41. A027 doesn't detect `event.WithCodec` assigned to a variable then passed (e.g., `opt := event.WithCodec(c); event.New(..., opt)`)
42. A027 doesn't detect codec set via `event.DefaultCodec = ...` followed by 0 `WithCodec` calls (which is the CORRECT pattern — could add a positive detection for good practice)
43. A027 counts calls across the entire file, not per-function — a file with 3 functions each using WithCodec once would trigger, which may be a false positive for large files

### Cleanup

44. Consolidate the 9 duplicated `runDetector`/`assertRule` test helpers into a shared test utility package
45. Remove any stale `//nolint` directives that may have been orphaned
46. Audit all `//nolint:ireturn` comments in detector factories — are they still needed?
47. Verify no other detectors have hand-rolled `itoa`-style helpers
48. Add file-length guard: `phase1_test.go` is at 331 lines (limit 350) — approaching the limit

---

## G. QUESTIONS (that I CANNOT figure out myself)

### 1. Should A027 also count `WithCodec` calls assigned to variables?

Currently A027 only detects direct `event.WithCodec(...)` calls inside `event.New(...)` argument lists. If a consumer writes `opt := event.WithCodec(codec.JSONCodec{})` and reuses `opt`, the detector misses it. Should we expand detection to count any `event.WithCodec` call, regardless of whether it's inline or assigned to a variable? This is a detection scope decision, not a technical one.

### 2. Should C017 be split into separate rules per store type?

C017 currently bundles snapshot, checkpoint, DLQ, and timer store detection into one rule. The anti-pattern is the same (in-memory auxiliary store + persistent event store), but splitting would give more precise suppression (`//cqrs-lint:ignore(C017)` currently suppresses all four). Alternatively, should I keep it as one rule with a parameterized message? This is an API design decision for the linter.

### 3. Should the auto-commit daemon be disabled during active editing sessions?

The daemon reverted one of my edits this session (the `multiedit` call). Using `write` as a workaround works but is less surgical. Is there a way to temporarily pause the daemon, or should I always use `write` for multi-edit changes? This is an environment/workflow question I can't answer by reading code.
