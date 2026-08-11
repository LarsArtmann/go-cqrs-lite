# Session Status — 5 Quick Wins (2026-08-11 ~16:30 → 17:55)

**Date:** 2026-08-11 17:53
**Scope:** Execute the 5 agreed quick wins from TODO_LIST.md; negotiate scope when blocked.
**Update 2026-08-11:** 4/5 done (benchkit tag, batch atomicity test, compile-time assertions, SQLite e2e graph fallback) — see CHANGELOG `[Unreleased]`. The 5th (commandlifecycle tag) is BLOCKED on the `id.ActorID` gap — tracked in TODO_LIST → Release/Tagging.

---

## a) FULLY DONE ✅

1. **benchkit/v4.4.0 tag verified** — quick win #1 was already satisfied: tag exists (7d5cd10c7, contains `Truncate`/`TitleCase`). Both TODO entries (Metaengine Coverage Gaps + Phase 6b duplicate) closed as `[x]`.
2. **Batch atomicity rollback test** (quick win #3) — `metaengine/batch_atomicity_rollback_test.go` (committed in `b2beac1c1`):
   - `TestBatchAtomicity_RollbackOnSecondFoldFailure` — 2 folds, same transactional engine, 2nd fold fails → 1st fold rolled back, no poison, exactly 1 rollback observed, recovery verified.
   - `TestBatchAtomicity_CommitOnAllFoldsSuccess` — commit path, 1 commit / 0 rollbacks.
   - `TestBatchAtomicity_NonTransactionalEngineNoRollback` — documents the documented limitation (no rollback without `Transactional`).
   - Race-clean (`-race` passes).
3. **Engine compile-time assertions** (quick win #4) — committed in `b2beac1c1`:
   - bbolt: `_ metaengine.HealthChecker` + `_ metaengine.StreamingScan` (both interfaces implemented; verified by compile).
   - mysql: `_ metaengine.HealthChecker` (verified; `HealthCheck` exists).
   - **Correction applied**: TODO said mysql needed `Calibratable` too — FALSE. mysqlengine has no `SetCalibration`, so the assertion was correctly NOT added. It's a feature gap, not an assertion gap. TODO text is wrong.
4. **SQLite-backed e2e graph fallback test** (quick win #5) — `metaengine/graph_fallback_sqlite_e2e_test.go`, external package (committed `46bacb50d`):
   - Full `Plan → Apply → ExecuteCtx` round-trip on a real SQLite engine (in-memory, `modernc.org/sqlite`), which implements `MultimapBackend` but only `ADTGraph` as DEGRADED.
   - Verifies BFS neighbor results through real SQL + checks a `DEGRADED` diagnostic is emitted.
   - Race-clean.
5. **id.ActorID release gap documented** — `docs/status/2026-08-11_17-05_actorid-release-gap-blocks-commandlifecycle.md` (committed in `a0c4b0205`): full evidence table (tags, `git cat-file -e`, merge-base checks), affected modules, proposed 6-step fix. This is the blocker for quick win #2.
6. **TODO_LIST.md updated** — commandlifecycle entry annotated BLOCKED with the release-gap fix path; both benchkit entries closed.

## b) PARTIALLY DONE ⚠️

7. **commandlifecycle/v4.0.0 tag** (quick win #2) — **BLOCKED, not done, by user decision**: the choice was "stop and report" (no tags created). The block is real and documented: published `record/v4.1.0`, `command/v4.4.0`, `metaengine/v4.8.0` reference `id.ActorID`, but newest `id/v4` tag is v4.2.0 (missing `actor_id.go`). Consumers resolving those published versions fail with `undefined: id.ActorID`. Fix: tag `id/v4.3.0` → re-tag record/v4.2.0, command/v4.5.0, metaengine/v4.9.0 → then commandlifecycle/v4.0.0 + projections/v4.0.0 → bump downstream go.mod requires (66 modules reference id/v4; 61 reference record v4.1.0).
8. **Docs/status + TODO cleanup** — the benchkit TODO closure edit I made was later committed by the daemon in a different commit (`3e55baf89` + follow-ups) — net effect correct, but attribution/history is messy (my edit was reverted once by the BuildFlow hook failure, then re-applied by daemon).

## c) NOT STARTED (by design) 🚫

9. Full release-chain fix (tagging id/record/command/metaengine) — explicitly declined by user (stop-and-report).
10. No `nix run .#verify` full gate ran to completion on my changes — I ran targeted `go build`/`go test`/`go vet`/`-race` on affected modules instead, because the daemon's WIP breaks the full suite (see d). This is my main gap (see e1).

## d) TOTALLY FUCKED UP / PRE-EXISTING BREAKAGE 💥

11. **5 metaengine test failures (203/208 pass)** — `layout_followup_test.go` (3) + `relayout_test.go` (2). Root cause: the **auto-commit daemon committed its own INCOMPLETE layout-priority refactor** (`priorityForQuery`) in the SAME commit that carried my batch test (`b2beac1c1`, 33 files!) and again in `f50f9c64f`. LayoutWarnings behavior changed (JOIN_AMPLIFICATION warning now emitted where tests expect none; `SetPriority`/`ReplanLayout` expectation drift). **NOT caused by my changes** — verified my files are additive tests + assertions only.
12. **Daemon fight: my files were reverted/deleted multiple times mid-session**:
    - `batch_atomicity_rollback_test.go` deleted once (recreated).
    - `graph_fallback_e2e_test.go` toggled between 158/221-line states ≥4 times; daemon committed a BROKEN version (internal test referencing undefined `newIsolatedSQLiteEngine` + import cycle) which I had to fix and re-commit (`46bacb50d`).
    - Attribution confusion: daemon credited my work in `b2beac1c1` while also bundling its own 30-file WIP.
13. **5 layout test failures remain in HEAD right now** — verify gate is RED until the daemon's layout WIP lands properly. `system/query_constructors.go:80: undefined: layoutPriority` also flagged by lint during my commit hook run.

## e) WHAT WE SHOULD IMPROVE 🛠️

E1. **Never trust a full-suite claim mid-session when the daemon is committing.** I burned ~30 min chasing "newSQLiteEngine undefined" / file churn that was the daemon, then verified my scope in isolation. Lesson: check `git status` + `git log` FIRST when a file "changes itself".
E2. **Stage-and-commit my work in smaller batches immediately after each green test.** Once committed, the daemon respects it; uncommitted files get clobbered. I committed late (only after the 3rd revert).
E3. **The TODO items "engine compile-time assertion gaps" (mysql Calibratable) is factually wrong** — either fix the TODO text or add the missing `SetCalibration` to mysqlengine as a real feature task. Today I only noted it.
E4. **The mixed-commit hazard**: daemon's `b2beac1c1` bundled my test with its incomplete refactor, making blame/revert impossible. Recommend daemon exclude `*_test.go` from its own feature commits, or commit tests separately.
E5. **Release hygiene**: the `id/v4.2.0`-without-ActorID gap means **consumers are broken TODAY** (not just commandlifecycle). `go get` of record/command/metaengine latest tags fails to compile for any consumer. This outranks most TODO items.
E6. **Quick win #5 could have been caught as already-done**: an e2e test already existed in `graph_fallback_e2e_test.go` (memory wrapper). The TODO asked for a *real* engine; I added SQLite — but a 2-minute check earlier would have saved rediscovering it.

## f) UP TO 50 THINGS TO GET DONE NEXT 📋

**Release / Publish (do first — consumers are broken)**
1. Tag `id/v4.3.0` (publishes ActorID; api_surface already contains exports).
2. Re-tag `record/v4.2.0` with go.mod requiring `id/v4 v4.3.0`.
3. Re-tag `command/v4.5.0` with go.mod requiring `id/v4 v4.3.0`.
4. Re-tag `metaengine/v4.9.0` with go.mod requiring `id/v4 v4.3.0`.
5. Tag `commandlifecycle/v4.0.0`.
6. Tag `commandlifecycle/projections/v4.0.0` (needs CL first — go.mod already pins it).
7. Bump downstream go.mod requires: all modules that use ActorID (event, query, metadata, storage/bbolt, storage/pebble, watermill, etc. — 66 modules require id/v4).
8. Run `nix run .#verify` + `nix run .#vulncheck` before pushing tags.
9. Push tags (needs user approval), then verify proxy pickup via `GOPROXY=proxy.golang.org go list -m`.
10. Add CHANGELOG entries for the new versions (Unreleased section).
11. Regenerate `docs/api_surface.txt` if any export changed (probably just id).

**Fix daemon/layout WIP (verify gate is RED)**
12. Finish the `priorityForQuery` refactor so `LayoutWarnings` behavior matches tests (or update tests to the new intended behavior).
13. Fix `relayout_test.go` ReplanLayout expectations (2 failures).
14. Fix `layout_followup_test.go` SetPriority/GetLayoutInfo expectations (3 failures).
15. Fix `system/query_constructors.go:80: undefined: layoutPriority` lint error.
16. Run full `nix run .#verify` and get GREEN before any further releases.
17. Investigate `markdown-lint`/`codespell`/`gomod-check` pre-commit findings (90+ go.mod mixed requires) — part of buildflow preflight.

**Metaengine follow-ups from this session**
18. Fix TODO text: "mysqlengine missing Calibratable" → either add `SetCalibration` to mysqlengine or correct the TODO.
19. Add `SetCalibration` micro-benchmark impl to mysqlengine (Calibratable) — needs a probe/measure approach.
20. Consider `bboltengine` Calibratable parity is already there — verify mysql only missing piece is SetCalibration.
21. Add failure-path test for `Multi-collection batch atomicity` (Phase 7 🔥) — extend my rollback test to 3 collections.
22. Add rollback test for `ApplyRecord` (Record-aware folds) — my test uses `Apply`; ensure Record path rolls back too.
23. Add rollback test with `InTransaction` wrapper + failing fold inside (Store.InTransaction → Apply → Apply).
24. Consider `per-fold mutex` (TODO item) — my fake engine serializes foldMu globally; per-fold would parallelize; note the interaction.
25. Verify `spike_batch_atomicity_test.go` recommendations still hold (memory engine snapshot/rollback) — flagged as ~6h task; my fake engine already implements it for tests.

**Test/Quality**
26. Run my 3 new test files with `-count=3 -race` across metaengine + bboltengine (lean-budget threshold discipline).
27. Add property-based test for `metadataPayload` CBOR roundtrip (from TODO M).
28. Consider moving `metadataPayload` extraction to `storage/serialization/` when a 3rd KV engine is added.
29. Fix `.golangci.yml` exclusion audit (system/ 20, cqrs-lint 17, metaengine 24 disabled) — track removability after migrations.
30. Add `#check-lint-config` nix app + `#verify-ci` mirror (TODO M).
31. Wire `#sweep` to pre-commit/cron.
32. Consolidate engine `register.go` boilerplate (7 modules).
33. Audit/trim indirect deps in `metaengine/go.mod` (modernc sqlite chain).

**Docs**
34. Document commandlifecycle in skill references (modules.md, recipes.md, advanced.md) — TODO S.
35. Update `calibration-baseline.md` with bbolt HealthChecker/StreamingScan calibration results.
36. Add a release-gap runbook entry to CONTRIBUTING.md ("if a module references an untagged symbol, tag the dependency first").
37. Update the status report above's findings into TODO_LIST (any new items).
38. Consider adding a "daemon safety" note to AGENTS.md: staged work is safe, unstaged/untracked work can be clobbered.

**Integration / infra**
39. macOS verification of ephemeral PG script (TODO M).
40. Write actual Redis/NATS integration tests (ephemeral scripts exist, no Go tests).
41. Write actual Dgraph integration tests in Go.
42. Tag `benchkit/v4.4.0` — verify `cmd/cqrs-bench` now works under GOWORK=off (release validation).
43. Publish go-finding + go-must tagged modules (BLOCKED item) — unblocks consumers of cmd/cqrs-lint.

**v5 / layout (context)**
44. Run `nix run .#verify` clean for fold inference (Infer) — TODO 🔥.
45. Run `nix run .#verify` clean for layout planning — TODO 🔥 (overlaps #12-16).
46. cqrs-bench layout CLI subcommand.
47. Calibrate cost model multipliers (replace 0.5/1.0/1.3/2.0 placeholders).
48. Role transition API (Backup→Active promote) + async replication.
49. Multi-engine integration test with two real backends.
50. e2e Store test for graph fallback already done (this session) — mark TODO `[x]`; same for batch atomicity rollback test TODO.

## g) 3 Questions I CANNOT answer myself ❓

1. **Push policy**: The `id/v4.3.0` + re-tag chain fixes *currently-broken consumers*. Do you want me to (a) prepare all tags locally and list them for your explicit push approval, or (b) keep this in the dedicated release session and NOT touch tags at all?
2. **Daemon coordination**: The auto-commit daemon committed its incomplete layout refactor mixed into the commit carrying my tests. Should I (a) coordinate by committing my work in micro-batches first (recommended), or (b) disable/suspend the daemon during feature sessions, or (c) is this acceptable noise you'd rather I just work around?
3. **Scope of "verify gate" for MY changes**: Since the daemon's layout WIP makes the full suite RED, is it acceptable that I validated only my affected modules (metaengine, bboltengine, mysqlengine) + race, deferring `nix run .#verify` until the layout WIP lands? Or do you want me to fix the 5 layout failures myself (they're the daemon's WIP, but I can take them over)?
