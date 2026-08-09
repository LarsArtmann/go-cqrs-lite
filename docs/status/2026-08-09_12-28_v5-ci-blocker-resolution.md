# Status: v5 Consumer API — CI Blocker Resolution Session

**Date:** 2026-08-09 12:28
**Session goal:** Resolve the two critical CI blockers from the P1-P4 execution: (1) destroyed go.work, (2) 3 failing GOWORK=off tests due to unpublished metaengine fix.
**Outcome:** Blockers resolved at the source (tags created, go.work restored), but **system/go.mod version bump is NOT committed** — blocked by Go module constraint (workspace mode cannot resolve unpublished tags).

---

## a) FULLY DONE

### 1. go.work restored (commit `247e69347`)

- Commit `53dc0ecb2` (dep bump) had emptied `go.work` from 94 lines of `use` directives down to just `go 1.26.5`, breaking workspace mode entirely.
- Restored from `git show 53dc0ecb2~1:go.work` — exact recovery, 94 lines, all 71 module `use` directives + genproto replace pin.
- Verified: workspace mode builds and all system tests pass.
- The auto-commit daemon committed this as `247e69347 chore(workspace): expand go.work and pin genproto to resolve import conflict`.

### 2. `metaengine/v4.8.0` tagged

- The `verifyEventParam` nil-check fix (commit `8eb4cad96`, `metaengine/fold.go:424`) was committed but NOT included in `metaengine/v4.7.0`.
- Verified: `git merge-base --is-ancestor 8eb4cad96 metaengine/v4.7.0` → NO (fix absent from v4.7.0).
- Created annotated tag `metaengine/v4.8.0` via `scripts/tag-release.sh` — strips local replaces, creates temp commit, tags, undoes temp commit.
- Verified: annotated tag type, fix IS ancestor, v4.8.0 IS descendant of v4.7.0 (monotonic ancestry).
- **This fixes the 3 originally-failing tests** (TestSystem_Count_E2E, TestSystem_Runtime_GetCount, TestSystem_Evolution_ExplicitFold).

### 3. `watermill/v4.3.0` tagged (DISCOVERED during verification)

- While running the full GOWORK=off test suite, a **4th test failure** surfaced: `TestSimpleBus_HandlerIndependence`.
- Root cause: commit `60b63ea16 fix(watermill): continue dispatching to all handlers when one returns an error` was committed but NOT published as a tag. The test checks exactly this behavior.
- Created annotated tag `watermill/v4.3.0` via `scripts/tag-release.sh`.
- Verified: fix IS ancestor, proper ancestry from v4.2.0.

### 4. GOWORK=off verification (temp replaces)

- Added temporary local replaces for both metaengine and watermill in system/go.mod.
- Ran `GOWORK=off go test -tags "goexperiment.jsonv2" -count=1 .` → **ALL tests pass** (previously 4 failures: 3 from metaengine + 1 from watermill).
- Removed temp replaces afterward (restored clean working tree).
- Also verified with race detector: `go test -race` passes in workspace mode.

### 5. BuildFlow auto-fix improvements committed (`69b2b14ee`)

- The pre-commit hook applied legitimate auto-fixes:
  - `metaengine/dgraphengine/retry.go`: simplified delay clamp to `min()` builtin (Go 1.21+)
  - `metaengine/spike_batch_atomicity_test.go`: struct conversion instead of field-by-field copy
  - `system/evolutions.go`: removed redundant `//nolint:errcheck`
  - `system/integration_badger_test.go`: removed redundant `//nolint:contextcheck`
  - `cmd/api-stability/main_test.go`: blank-line formatting from dprint
- All verified to compile and pass tests.

---

## b) PARTIALLY DONE

### system/go.mod version bump — NOT COMMITTED

- **What I tried:** `go mod edit -require=...metaengine/v4@v4.8.0` and `...watermill/v4@v4.3.0`, then removed temp replaces.
- **What happened:** Workspace mode fails with `unknown revision metaengine/v4.8.0` because the tags are **local-only** (not pushed to remote). Go workspace mode validates go.sum entries against the module proxy, which doesn't have these tags yet.
- **Current state:** system/go.mod still shows `metaengine/v4 v4.7.0` and `watermill/v4 v4.2.0`. Working tree is clean (changes were restored).
- **What's needed:** Push tags first, THEN bump go.mod, THEN `go mod tidy` to populate go.sum with real hashes.
- **Why this is a gap:** CI runs `GOWORK=off` per-module tests. Without the bump, CI still pulls v4.7.0/v4.2.0 and the 4 tests will fail.

---

## c) NOT STARTED

| Item                                      | Phase    | Notes                                                                      |
| ----------------------------------------- | -------- | -------------------------------------------------------------------------- |
| P5: Domain/Deployment restructure         | Deferred | Split DomainConfig into Domain + Deployment structs                        |
| P6: Migrate metaengine-quickstart example | Deferred | Update example to new API                                                  |
| P7: Graph query types                     | Deferred | Traversal, Path constructors + Traverse/FindPath runtime                   |
| P8: Docs + ADR-0124                       | Deferred | Document the v5 consumer API design                                        |
| Full `nix run .#verify` gate              | Not run  | Only ran system/metaengine/watermill tests, not the full repo verification |
| API stability golden regen                | Not run  | No exported symbols changed this session, so likely not needed             |

---

## d) TOTALLY FUCKED UP

**Nothing this session.** The prior session's mistakes (go.work destruction, unpublished fix) were inherited and resolved cleanly.

One thing to call out: **the auto-commit daemon races with manual commits.** When I tried to commit go.work restoration, the daemon had already committed it as `247e69347`. The pre-commit hook then failed with `fatal: cannot lock ref 'HEAD'`. I recovered by checking status, seeing the daemon's commit, and moving on. This is fragile but worked this time.

---

## e) WHAT WE SHOULD IMPROVE

### Process

1. **Tag before bumping, always.** The Go module proxy constraint means you cannot bump a require to a local-only tag. The correct sequence is: commit fix → tag → push tag → bump consumers → tidy → commit bump. I discovered this mid-flow.
2. **Check ALL dependencies for unpublished changes before starting CI verification.** I only knew about metaengine. The watermill failure was discovered reactively. A proactive sweep (`git log --oneline <tag>..HEAD -- <module>/` for every direct dep) would have caught both upfront.
3. **Run GOWORK=off tests as part of every feature session, not just at the verify gate.** The workspace-vs-per-module divergence is the #1 source of "works on my machine" failures. Workspace mode masks unpublished dependency fixes.
4. **The auto-commit daemon should be paused during release workflows.** The `tag-release.sh` script creates temp commits and undoes them — if the daemon races, it can leave the repo in an inconsistent state.
5. **The go.work destruction (`53dc0ecb2`) should never have been committed.** Whatever process ran the dep bump should have verified go.work wasn't collateral damage.

### Codebase Health (noticed during this session)

6. **13 of system's 14 direct go-cqrs-lite dependencies have unpublished local changes.** This is a massive backlog of un-released work. Any CI run in GOWORK=off mode will eventually hit failures from version drift.
7. **decider has 42 unpublished commits, projectionhost has 64, event has 40.** These modules have significant unreleased work. A batch-release session is overdue.
8. **`example/metaengine-quickstart/metaengine-quickstart` binary is tracked in git** (BuildFlow flagged this as ERROR). Should be `git rm --cached`'d.
9. **82 modules have mixed direct/indirect require blocks** (BuildFlow gomod-check finding). Cosmetic but violates Go 1.17+ convention.

---

## f) Next 50 Things to Get Done

### CRITICAL — Do these FIRST (blocks CI)

1. **Push both tags:** `git push origin metaengine/v4.8.0 watermill/v4.3.0`
2. **Bump system/go.mod:** `go mod edit -require=...metaengine/v4@v4.8.0` and `...watermill/v4@v4.3.0`
3. **Tidy system/go.sum:** `cd system && GOWORK=off go mod tidy`
4. **Commit the bump:** `git commit -m "chore(deps): bump metaengine to v4.8.0 and watermill to v4.3.0"`
5. **Verify GOWORK=off passes without temp replaces** (the real CI test)
6. **Run `nix run .#verify`** — the full gate (build + vet + test + race + lint + doc-check)
7. **Mark P9 complete** in the plan document

### Batch Release Backlog (unpublished modules)

8. Audit all 13 unpublished modules for breaking changes
9. Tag `decider/v4.3.0` (42 commits — largest backlog)
10. Tag `projectionhost/v4.3.0` (64 commits)
11. Tag `event/v4.5.0` (40 commits)
12. Tag `snapshot/v4.3.0` (18 commits)
13. Tag `metaengine/sqliteengine/v4.1.0` (15 commits)
14. Tag `metaengine/projectionadapter/v4.4.0` (11 commits)
15. Tag `metaengine/duckdbengine/v4.1.0` (9 commits)
16. Tag `metaengine/pgengine/v4.1.0` (7 commits)
17. Tag `id/v4.3.0` (5 commits)
18. Tag `codec/v4.3.0` (12 commits)
19. Tag `command/v4.5.0` (1 commit)
20. Tag `metaengine/badgerengine/v4.1.0` (2 commits)
21. Tag `metaengine/pebbleengine/v4.1.0` (2 commits)
22. After all tags pushed: batch-bump all consumer go.mod files
23. Run full `GOWORK=off` test suite across all modules

### v5 Consumer API (P5-P8)

24. P5: Design Domain/Deployment split — extract DeploymentConfig from DomainConfig
25. P5: Create `system/domain.go` with new Domain struct
26. P5: Create `system/deployment.go` with new Deployment struct
27. P5: Migrate constructor to accept Domain + Deployment instead of DomainConfig + DeploymentConfig
28. P5: Update all tests to new config shape
29. P6: Audit `example/metaengine-quickstart/main.go` for v5 API migration
30. P6: Rewrite example using Lookup/QuerySet/Count + Evolutions
31. P6: Verify example compiles and runs
32. P7: Design graph query types (Traversal, Path)
33. P7: Implement `system.Traversal[R](name)` constructor
34. P7: Implement `system.Path(name)` constructor
35. P7: Implement `system.Traverse[R]()` runtime function
36. P7: Implement `system.FindPath()` runtime function
37. P7: Add graph query tests
38. P8: Write ADR-0124 documenting the v5 consumer API design
39. P8: Update SKILL.md references with new API
40. P8: Update `.agents/skills/go-cqrs-lite/references/recipes.md`
41. P8: Run doc-check: `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ...`

### Codebase Health

42. `git rm --cached example/metaengine-quickstart/metaengine-quickstart` (tracked binary)
43. Add `example/metaengine-quickstart/metaengine-quickstart` to .gitignore
44. Fix 82 modules with mixed direct/indirect require blocks (run `go mod tidy` in each)
45. Regenerate API stability golden if any exported symbols changed
46. Run `nix run .#check-arch` (dependency budget enforcement)
47. Run `nix run .#check-duplication` (no-new-clones gate)
48. Run `nix run .#check-coverage` (coverage drift)

### Improvements identified in prior session

49. Make `Count.On` generic to avoid `any`-typed sample (eliminates the need for the `verifyEventParam` nil-check entirely)
50. Wire Evolution folds to command state loading (auto-build Deciders from Evolution definitions)

---

## g) Questions (cannot resolve without user input)

### Q1: Push the tags now?

The tags `metaengine/v4.8.0` and `watermill/v4.3.0` are local-only. I cannot bump `system/go.mod` until they're on the remote (Go workspace mode validates against the proxy). Should I push them, or do you want to review the tagged commits first?

**Why I can't figure this out myself:** Pushing to remote is explicitly prohibited by my instructions ("NEVER PUSH TO REMOTE unless explicitly asked"). But without the push, the version bump is impossible.

### Q2: Batch-release all 13 unpublished modules, or just the 2 that block tests?

13 of system's 14 direct dependencies have unpublished local changes (see items 8-21 above). Only metaengine and watermill cause test failures right now, but the others represent significant unreleased work (decider: 42 commits, projectionhost: 64 commits). Should I do a full batch-release session, or just unblock CI with the minimum?

**Why I can't figure this out myself:** This is a release-strategy decision (risk tolerance vs. thoroughness) that depends on your priorities.

### Q3: Fix the root cause (Count.On `any`-typed sample) or keep the nil-check workaround?

The `verifyEventParam` nil-check in `metaengine/fold.go:424` is a workaround for `Count.On` using `OnTyped` with `any`-typed handlers. The "proper" fix (identified in the prior session) is to make `Count.On` generic so it never needs `any`. Should I pursue that refactor, or is the nil-check acceptable as a permanent design?

**Why I can't figure this out myself:** This is an API ergonomics vs. implementation simplicity tradeoff that depends on your design philosophy for the v5 API.

---

## Session Metrics

| Metric               | Value                                                                             |
| -------------------- | --------------------------------------------------------------------------------- |
| Commits this session | 2 (`247e69347`, `69b2b14ee`)                                                      |
| Tags created         | 2 (`metaengine/v4.8.0`, `watermill/v4.3.0`)                                       |
| Tests verified       | system (35 tests, workspace + GOWORK=off + race), metaengine, watermill           |
| Files modified       | 5 (BuildFlow auto-fixes only)                                                     |
| Blockers resolved    | 2 of 3 (go.work + metaengine fix); watermill discovered and resolved as bonus 3rd |
| Blockers remaining   | 1 (system/go.mod bump — blocked on tag push)                                      |
