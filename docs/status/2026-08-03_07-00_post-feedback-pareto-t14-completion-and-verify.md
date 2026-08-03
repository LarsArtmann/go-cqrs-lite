# Status: Post-Feedback Pareto Plan T14 Completion + Verify Gate Repair

**Date:** 2026-08-03 07:00
**Session goal:** Finish T14 (F-series feature-profile gating), fix broken tests, run verify gate

---

## A) FULLY DONE

### This session specifically

1. **Verified all 3 "broken" F-series tests already pass** — The prior session's auto-commit daemon had already committed the feature-profile overrides (`ctx.FeatureProfile.HasServer = true`, `ctx.FeatureProfile.HasAsyncBus = true`) in the test source code. Tests F009, F015, F017 all PASS.

2. **Ran `goimports -w` on all changed files** — 10 files cleaned. No unused imports found (build + vet clean).

3. **Regenerated api-stability golden** — `cd cmd/api-stability && GOWORK=off go run . -update` → 3207 exports verified. Exported `FeatureProfile.HasAsyncBus` field, C037 catalog name change, D007 `AutoFix` change all captured.

4. **Full cqrs-lint test suite** — 16/16 packages PASS (`go test ./cmd/cqrs-lint/...`).

5. **Full workspace build + vet** — `go build ./...` and `go vet ./...` both clean.

6. **Verified gate results** — Ran `nix run .#verify` twice. Results:
   - Documentation assertions: PASS (92 ADRs indexed, CHANGELOG OK, license OK)
   - Build: PASS
   - Vet: PASS
   - Test (all modules): PASS (every module green)
   - Race detector: PASS
   - Lint (golangci-lint per module): PASS (0 issues across all modules)
   - Check Layers: PASS
   - **Check Duplication: FAIL** — 3 new clone groups (see section D)

7. **Fixed `universal_adt_test.go` compilation** — The test file referenced `newQueryForADT()` which didn't exist. Initially added the helper to `fixtures_test.go`, then discovered the daemon had already rewritten the test to not need it. Removed the now-unused helper. Metaengine tests PASS (7/7 universal ADT tests).

8. **Fixed api-stability run command** — `go run main.go` → `go run .` (the `collectExports` function lives in `collect.go`, not `main.go`; must compile the whole package).

---

## B) PARTIALLY DONE

### T14: F-series feature-profile gating (~95%)
- **Code changes:** DONE and committed (F009, F015, F017 gated)
- **Tests:** PASS
- **TODO_LIST.md update:** NOT YET DONE — T14 still shows as `- [ ]` in TODO_LIST.md

### Verify gate GREEN
- Everything passes except the duplication check (3 new clone groups in cqrs-lint files)
- The clones are in files NOT touched by this session (`c038.go`, `d017.go`, `c032.go`, `ast_helpers.go`) — they were introduced by prior sessions' auto-commit daemon commits

---

## C) NOT STARTED

1. **Mark T14 done in TODO_LIST.md** — Single line change from `- [ ]` to `- [x]`
2. **Push tags** — 3 local tags need user approval to push (safety rule)
3. **Update prior status report** — `docs/status/2026-08-03_01-12_post-feedback-pareto-execution.md` doesn't reflect T14 completion

---

## D) TOTALLY FUCKED UP

### 1. The "broken tests" were never broken
I was handed a briefing that said 3 tests were broken. The FIRST thing I should have done was run them to confirm. Instead, I:
- Read all the test source files
- Read the detector source files
- Read the feature profile detection code
- Read the feature_profile.go struct definition
- THEN ran the tests — and they all passed

**Lesson:** Verify the problem exists before investigating its cause. The prior session had already fixed the tests via feature-profile overrides (`ctx.FeatureProfile.HasServer = true`).

### 2. Added a function then immediately removed it
I added `newQueryForADT()` to `fixtures_test.go` to fix a compilation error in `universal_adt_test.go`. But the daemon had already rewritten that test file to not need the function. So I had to immediately remove it.

**Lesson:** The auto-commit daemon changes files between sessions. Always re-read the file's CURRENT state before acting on a briefing about its state. The daemon can rewrite tests, add imports, reformat code — all between sessions.

### 3. Did not mark T14 done in TODO_LIST.md
I had this in my todo list as the last step and didn't execute it before the user asked for a status report.

### 4. Duplication baseline is stale (3 new clone groups)
The verify gate fails on the duplication check. These clones are in cqrs-lint rule files:
- `ast_helpers.go:66-71` vs `c038.go:207-212` — `*ast.BasicLit` extraction pattern
- `d017.go:122-131` vs `lintutil.go:210-219` — `len(call.Args) == 0` guard pattern
- `d017.go:99-104` vs `c032.go:164-169` — `call.Fun.(*ast.SelectorExpr)` assertion pattern

These are NOT from my changes — they were introduced in prior sessions. But the verify gate won't be GREEN until they're either deduplicated or the baseline is updated.

---

## E) WHAT WE SHOULD IMPROVE

1. **Stop trusting session handoff briefings blindly** — Always verify the claimed state matches reality before acting. Run the tests first.

2. **The auto-commit daemon is a double-edged sword** — It keeps the tree clean, but it also silently rewrites files between sessions. Always re-read current state.

3. **Duplication baseline drift** — The baseline is at 47 clone groups but the codebase now has 50. This should be either fixed (dedup the 3 clones) or the baseline updated. Leaving it broken means the verify gate is never GREEN.

4. **TODO_LIST.md is stale** — T14 is functionally complete (code + tests pass) but the TODO list still shows it as open. This is the exact "stale status" anti-pattern the AGENTS.md warns about.

5. **Tags are piling up locally** — 3 unpushed tags from prior sessions. These block consumers from resolving "latest" for those modules.

6. **`api-stability` run command is error-prone** — `go run main.go` fails because the package has multiple files. Should document `go run .` in the AGENTS.md or add a comment.

---

## F) THINGS TO GET DONE NEXT (up to 50)

### Critical (blocks GREEN verify gate)
1. **Fix or baseline the 3 new duplication clones** in cqrs-lint (`c038.go`, `d017.go`, `c032.go`, `ast_helpers.go`)
2. **Mark T14 as DONE in TODO_LIST.md**
3. **Update `docs/status/2026-08-03_01-12_post-feedback-pareto-execution.md`** to reflect T14 completion

### Releases
4. **Push the 3 local tags** (needs user approval)
5. **Verify all module tags are monotonically increasing** — `git tag -l '<module>/v4*' | sort -V | tail -1`
6. **Run `nix run .#vulncheck`** to verify no version-sequence breaks in published tags

### Code quality
7. **Deduplicate the `*ast.BasicLit` extraction** — shared helper in `ast_helpers.go`, used by `c038.go`
8. **Deduplicate the `len(call.Args) == 0` guard** — shared helper in `lintutil.go`, used by `d017.go`
9. **Deduplicate the `call.Fun.(*ast.SelectorExpr)` assertion** — shared helper, used by `c032.go` + `d017.go`
10. **Remove unused `layoutComplexity` function** in `metaengine/layout_type.go:37` (gopls flagged)
11. **Remove unused `op` type** in `metaengine/property_test.go:12` (gopls flagged)
12. **Fix `unusedwrite` warnings** in `metaengine/reliability.go:52-54` (NsPerOp/NsPerRead/NsPerWrite)
13. **Remove unused `close` method** in `metaengine/transaction.go:67`
14. **Modernize `context.WithCancel` → `t.Context()`** in `metaengine/features4_test.go:137,501`
15. **Remove unnecessary type arguments** in `metaengine/features4_test.go:1016,1045`

### cqrs-lint improvements
16. **Add more F-series tests** — test the gating suppression paths (F009 when !HasServer && CommandFlow != Commands, F015 when !HasServer, F017 when !HasAsyncBus)
17. **Add C037 test for mixed codecs across all 4 stores** — currently only snapshot has "same codec no finding" test
18. **Add D007 auto-fix integration test** — verify the fix pipeline actually applies the replacement
19. **Self-lint the cqrs-lint codebase** — run cqrs-lint on itself to catch its own anti-patterns
20. **Add `HasAsyncBus` to `FeatureProfile.String()`** — it's missing from the doctor output

### Metaengine
21. **Wire `ByteSize` type** if it exists (plan mentioned it but prior session concluded it doesn't)
22. **Add SSE reconnection tests** with the new `SSEReplay[V]` ring buffer
23. **Add cursor-encoded prefetch tests** — `WithCursorString` parsing + matching keys
24. **Add materialize-vs-replay integration test** — `ShouldMaterialize` with real workload stats
25. **Document the planner rule pipeline** in an ADR (ADR-pending per AGENTS.md)
26. **Add `VectorExecuteTyped`/`SearchExecuteTyped`/`SpatialExecuteTyped` tests** — the new ADTs from ADR-0085

### Testing
27. **Run `-race -count=3` on the MySQL testcontainer test** — verify the fix holds under repeated race detection
28. **Run `-race -count=3` on the idempotency/sqlstore TTL test** — verify the timing fix is stable
29. **Add a CI soak test for the 10M event scenario** — the soak test from 2026-08-02 passed but isn't in regular CI
30. **Run coverage check** — `nix run .#check-coverage` to verify no drift

### Documentation
31. **Update AGENTS.md** — document `go run .` (not `go run main.go`) for api-stability
32. **Add ADR for the F-series feature-profile gating pattern** — document how rules suppress based on project profile
33. **Update FEATURES.md** — mark F009/F015/F017 gating as DONE
34. **Update ROADMAP.md** — move feature-profile gating from planned to done
35. **Review and close stale ADRs** — 92 ADRs, some may be superseded

### Architecture
36. **Review whether `HasAsyncBus` should also detect NATS/Redis/Kafka** directly (not just Watermill)
37. **Consider adding `HasDispatch` as a separate flag** from `CommandFlow == CommandFlowCommands`
38. **Evaluate whether F015's Store exclusion (SQLite/Memory/Pebble) is correct** — Postgres is the main beneficiary
39. **Add metaengine to the cqrs-lint feature detection** — `HasMetaEngine` flag for future rules
40. **Review the seven-tier model accuracy** — metaengine is Tier 0 by deps but Tier 3 conceptually

### DevOps
41. **Run `nix flake check`** — verify the flake is healthy
42. **Verify CI workflow matches local verify gate** — ci.yml vs nix verify
43. **Add a pre-commit hook that runs `go build ./...`** — prevents broken-code commits
44. **Review the auto-commit daemon's diff before accepting** — it can ship breaking bumps
45. **Add `nix run .#check-duplication` to the PR review checklist**

### Polish
46. **Clean up the `docs/status/` directory** — 400+ files, many stale; archive old ones
47. **Add a `make verify-quick` alias** for the common dev loop (build + vet + test, skip lint/race/docs)
48. **Review gopls diagnostics** — 38+ hints/warnings across the project
49. **Standardize error wrapping helper names** — `wrapXOrOK` vs `wrapX` inconsistency
50. **Review all `//nolint` directives** — some may be stale after refactoring

---

## G) QUESTIONS (that I cannot figure out myself)

### 1. Should I update the duplication baseline or deduplicate the 3 new clone groups?
The 3 clones are small AST patterns (`*ast.BasicLit` extraction, `len(call.Args) == 0` guard, `*ast.SelectorExpr` assertion) that appear naturally in cqrs-lint rule detectors. Extracting shared helpers for 5-line patterns may reduce readability. **Option A:** Update baseline with `art-dupl baseline . --threshold 3 --semantic`. **Option B:** Extract shared helpers. Which do you prefer?

### 2. Should I push the 3 local tags now?
The tags are for modules that were changed in prior sessions. Pushing them makes the new APIs available to consumers resolving "latest". But the safety rule says never push without explicit approval. Do you want me to push them, or will you review them first?

### 3. Is the auto-commit daemon still running?
Several files changed between the prior session's handoff and this session (notably `universal_adt_test.go` was rewritten, a new ADR 0094 appeared, `metaengine` was bumped to v4.3.0). If the daemon is still running, it will commit this status report and the TODO_LIST.md update automatically. Should I be aware of any pending daemon work?
