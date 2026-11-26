# Full Code Review Session: Red CI Root-Caused and Repaired Locally, Fixes Landed in the Working Tree, Every-File Fan-Out Blocked by Provider Quotas

> **Status report (point-in-time snapshot).** Session: 2026-11-25 04:35 to 11:25 CET.
> Commission: "Review this repo!" under the `full-code-review` skill (every code and test file,
> fix on the spot, plan, report).

## Scope line

**Verified here (command and result, all run in this session):**

- Baseline at HEAD `28db93fe7`: `gh run view 45531845418` (push, 2026-11-25T02:17Z) lists 141 jobs,
  30 success, 111 failure; `gh run list --workflow CI --branch master --limit 30` lists 30 push runs,
  none successful. All 111 failed job logs were downloaded and classified by cause.
- Tag truth: `git ls-remote --tags origin` (3,370 lines) plus a script over all 105 `go.mod` files:
  70 distinct sibling pins, 0 without an origin tag. `proxy.golang.org` serves `event/v4 v4.15.3`
  (fetched `.info`: hash `12c42cb6d`, ref `refs/tags/event/v4.15.3`).
- After the tidy sweep: `GOWORK=off go vet ./...` in each of the 105 modules: 105 ok, 0 failed
  (`~/.cache/gcl-review/vet-sweep.out` was in `/tmp` and is gone after the crash; the run printed
  `vet ok: 105 / vet failed (0)` before).
- Targeted tests after each fix, all `GOWORK=off`, all `ok`:
  `cmd/cqrs-lint/v4` `./pkg/rules/{adoption,correctness,performance,version}`;
  `metaengine/v4` full suite once (29.9 s, after the `deriveKeys` fix) and the filtered
  compare/filter run after the `compare.go` fix; `system/v4` filtered
  (`ReadyCheck|HealthCheck|Timers|Close|Health`, then `TestSystem_Find_*|TestSystem_Runtime_Find`);
  `middleware/v4` filtered (`NilIsRetryable|TestCommandRetry|RetryConfig|DefaultRetry`).
- Gates run locally: api-stability check (`API surface OK: 6546 exports`, before the later API
  additions), `scripts/check-private-deps.sh` (rc 0) and `--audit` (18 repos, all PUBLIC),
  `scripts/check-package-surface.sh` (green after one reviewed re-pin),
  `scripts/pin-sweep.sh --check` ("All sibling pins are at their latest tags").
- Reviewer output: 13 of 183 scopes hold findings (9 complete, 4 interrupted mid-scope):
  257 findings, P0 0, P1 33, P2 146, P3 78 (counted by `~/.cache/gcl-review/agg.py` at 11:24).

**NOT verified here:**

- Nothing was pushed. Every fix lives in the working tree only: `git status` at 11:23 shows
  175 modified and 6 untracked files against HEAD `28db93fe7`; the last commit is still
  `28db93fe7`. CI on origin is unchanged and still red.
- The full test suites of the changed modules after the LAST edit were not run, nor any consumer
  module (`systemscenario/v4`, `systemtest/v4`, `system/integration/v4`, `benchkit/v4`, examples,
  `cmd/*`). No lint, no `#verify`, no `#verify-ci`.
- The api-stability golden was regenerated BEFORE `ReadyCheck`, `ErrSystemNotStarted`,
  `ErrTimersStopped`, `Unlimited` and `Exact` were added: the API check fails right now until it is
  regenerated again. The package-surface ratchet for `system/v4` (baselined 230) will trip too.
- `scripts/bump-cqrs-lint.sh` (nix build to re-pin the cqrs-lint vendorHash after the go-finding
  bump) ran during the first host crash; `flake.nix` shows no diff, the log was in `/tmp`: outcome
  unknown, must be re-run.
- The erraudit leg (`iroh-go ... could not import C`) was classified from the CI log only; not
  reproduced locally.
- Reviewer findings other than the three I fixed were not re-verified by me at source except
  `metaengine/v4/store.go:569` (read, confirmed, not fixed).

---

## a) FULLY DONE

All items are in the working tree, uncommitted.

1. **Research and baseline.** Inventory (7,212 tracked files; Go 3,802 files / 590k lines, split
   1,856 production / 1,935 test), digest of the three newest reviews, plans and status reports,
   census of the open owner issues, full cause classification of the 111 red CI legs.
   Evidence: this report's section d; findings reproduced by the commands in the scope line.
2. **Root cause of the red matrix fixed.** Commit `12c42cb6d` (2026-11-24 21:19) bumped pins in
   105 `go.mod` files and was never tidied: no `go.sum` carried a `go-codec v0.3.4` line. Repair per
   `docs/agents/gotchas-module-management.md`: `GOWORK=off go mod tidy` in the 91 modules whose
   `go mod tidy -diff` was non-empty (13 were clean, the root module checked separately, clean).
   Result: 66 `go.mod` and 93 `go.sum` files changed; `go vet` passes in 105 of 105 modules.
   The tidy also removed 301 unneeded indirect requires (ginkgo, pprof, gopsutil, go-winio lines)
   and lifted `tidwall/sjson` v1.2.5 to v1.2.6 where the graph selects it.
3. **The 13 red cqrs-lint fixture tests** (`BuildContext(typedfixture|scanfixture): ... module
   graph is likely broken`) pass again after the three testdata fixtures were tidied.
4. **API golden** regenerated for the two exports of `28db93fe7` (`NewB032Detector`,
   `NewB033Detector`): `6544 expected, 6546 actual` became `API surface OK: 6546 exports`.
5. **Pin-drift guard repaired** (`cmd/api-stability/v4/pin_drift_test.go`). The test derived tag
   names as `<module dir>/<version>` (`event/v4/v4.15.3`), but tags are cut as
   `<module dir without /vN>/<version>` (`event/v4.15.3`). Since the `/v4` path move it was dead
   for every module except the few that carry legacy double-shape tags, where it false-failed
   ("BROKEN PIN ... no such tag exists" in CI). Now maps through `tagDirFor` and uses the
   previously unused `tagExists`. Mutation-tested: a planted `v4.9.9` pin turns it red with
   `no such tag exists (metaengine/sqliteengine/v4.9.9)`; file restored byte-identical
   (sha256 prefix `9d350fc43cd8b651` before and after).
6. **Package-surface ratchet**: `metaengine/v4` re-pinned 859 to 860 after reviewing the one added
   symbol (`WithSSEReconnectAfter`, present in published tag `metaengine/v4.26.1`, cited in the
   CHANGELOG). Diff is that single row.
7. **Private-deps gate**: `example/taskmanager` requires `github.com/larsartmann/go-health`, which
   was not on the audited list. `gh api repos/LarsArtmann/go-health`: `private=false`. Added to
   `AUDITED_PUBLIC` in `scripts/check-private-deps.sh`; offline gate and live `--audit` both pass.
8. **Stale external pin**: `cmd/cqrs-lint/v4` bumped `github.com/larsartmann/go-finding`
   v1.15.1 to v1.16.0 (root module only: `pipeline` has no v1.16.0, `toolsdk` was already there).
   `pin-sweep.sh --check` green; cqrs-lint and cqrs-upgrade test suites passed GOWORK=off before
   the crash.
9. **Issue #141 (middleware nil `IsRetryable`).** Already non-panicking since the go-retry
   adoption (go-retry v0.9.0 `retry.go:106` falls back to `errorfamily.IsRetryable`). Made explicit
   in `middleware/v4/retry.go` (same nil guard as the circuit breaker), field doc now states that
   unclassified errors classify as Transient and are retried. Two regression tests with the
   issue's hand-built config shape pass.
10. **Issue #133 (nil key extractors).** `metaengine/v4/fold_classify.go`: without an insert fold,
    an update or remove fold that has no key source now fails at `Query` registration with
    `fold for <event> (<kind>): no key source ...` instead of panicking on the first applied event.
    Full `metaengine/v4` suite green; two regression tests (`fold_classify_keys_test.go`).
11. **Issue #139 (readiness).** Additive: `system/v4/readiness.go` adds `System.ReadyCheck`
    (not ready before Start when a projection host or managed timers exist, after Close, or when
    managed schedulers died), sentinels `ErrSystemNotStarted` and `ErrTimersStopped`, a
    `timers:not-started|running|stopped` segment in `Health()`, `HealthCheck` documented as a
    liveness probe, shared `healthCheckLocked`. Three tests in `system/v4/readiness_test.go` pass.
    `HealthCheck` semantics are unchanged on purpose (38 direct fleet consumers).
12. **Issue #135 (`system.Find` cap).** Additive: `system.Unlimited()` (forwards
    `metaengine.WithLimit(0)`) and `system.Exact()` (forwards `metaengine.WithExact`, so a
    truncated scan returns an error matching `metaengine.ErrScanTruncated`). The "All tasks" doc
    example no longer shows a call that stops at 100 rows; the `After` doc no longer points at a
    `PaginatedResult` that does not exist in package `system`. Four tests in
    `system/v4/runtime_find_limit_test.go` pass (the default cap of 100 is now pinned).
13. **Issue #132, timers half.** `system/v4/timers.go`: the scheduler context became a
    `WithCancelCause` child; when the context passed to `Start` is cancelled while the system is
    not closed, a warning is logged (was: silent death). `System.Start` documents the context
    lifetime contract. Covered by the third readiness test.
14. **Reviewer finding fixed:** `metaengine/v4/compare.go` `toFloat64` matched only unnamed numeric
    types, so a named numeric type (`type Priority int`) fell through to the printed-form
    comparison where `10 < 9`. Now kind-based through `reflect`; `compare.go` shrank to 218 lines;
    three tests in `compare_named_test.go`.
15. **Pareto plan** written and spliced:
    `docs/planning/2026-11-25_06-06_SUPERB-full-code-review-pareto-execution-plan.html` (parse,
    anchors, zero style attributes, zero placeholder words). Needs errata, see b).

## b) PARTIALLY DONE

1. **Every-file review.** 9 of 183 scopes complete (S001 to S005, S007, S010, S019, N-infra-1),
   4 interrupted with findings on disk but no summary (S009, S012, S013, S014). That is roughly
   9 percent of the planned reading (complete scopes cover about 60k of 590k Go lines plus the
   flake, workflows and lint config). Open: the other 170 scopes. Blocker: provider quotas (see d).
   Effort to finish: L.
2. **31 P1 findings not fixed.** 18 bugs, 5 test-quality, 4 doc-lie, 3 split-brain, 1 race. Only
   `store.go:569` was re-read by me: `ApplyIdempotent` records the dedup key before the apply, so a
   failed event is dropped on retry (confirmed; `store.go` is a baselined 613-line file, so the fix
   must not grow it). The rest carry the reviewer's confidence only. Open: source verification,
   then fix or harvest. Effort: L.
3. **Issue #132, projection half.** Not done: `projectionhost/v4/host.go` (381 lines) and
   `worker.go` (388 lines) are baselined oversized files that may only shrink, so the warning for
   workers needs a new file plus an offsetting removal. Effort: S.
4. **Plan document errata.** Two statements in the plan are wrong and not yet corrected:
   (1) "21 incoming sibling tags ... exist neither on origin nor locally" is false, they all exist
   (I checked the wrong tag-name shape first); (2) scope counts 223 + 8 became 171 + 12 after
   same-module packing. Effort: S.
5. **API golden and package-surface for the new exports.** Not regenerated after items 11 and 12
   of section a. Both gates fail until `cd cmd/api-stability/v4 && GOWORK=off go run . --update`
   and a reviewed `check-package-surface.sh --update-baseline` run. Effort: S.
6. **cqrs-lint vendorHash.** Unknown whether `flake.nix` needs a new hash after the go-finding
   bump (the `bump-cqrs-lint.sh` run died with the host). Effort: S, one serialized nix build.
7. **Docs truth fixes found, not applied:** `AGENTS.md` claims a uniform patch-form
   `go 1.27.2` directive (102 of 105 `go.mod` files say `go 1.27` since `12c42cb6d`; the three
   cqrs-lint testdata fixtures still say `1.27.2`); `AGENTS.md` says `docs/research/` has no index
   README (`docs/research/README.md` exists); `.go-version` says `1.26.6`;
   `docs/agents/module-map.md` says "204 rules" and "98 go.mod files" (DOMAIN_LANGUAGE says 217,
   the HEAD commit says 219, there are 105 `go.mod` files); `SKILL.md` carries the heading
   "Catalog exports: render tree vs governance tree" twice (lines 167 and 205);
   `docs/reviews/README.md` rows 10 and 11 misdescribe their reports (reported by the digest
   subagent, not re-checked by me); `DOMAIN_LANGUAGE.md` verification block still names
   `http.NewSSEBroker`, `grpc.NewCommandClient`, `http.BackfillHandler`, `flightrecorder.New` and
   the deleted `idempotency/kvstore`. Effort: M.

## c) NOT STARTED

1. Issues #138 (cqrs-lint registry blind to `event.NewEvents`), #134 (systemscenario await
   contract), #136 (readiness drain; collides with the declined `System.WaitReady`).
2. HTML review report `docs/reviews/2026-11-25_HH-MM_full-code-review.html`, its row in
   `docs/reviews/README.md`, the `AGENTS.md` report entry.
3. `TODO_LIST.md` harvest of every open finding; CHANGELOG `[Unreleased]` and FEATURES.md rows for
   the fixes of section a.
4. Comments on the eight owner issues with their verified status.
5. Tag-audit baseline prune (CI prints 12 "baseline entry no longer violating, drop it" notes).
6. erraudit leg (iroh cgo import failure in CI).
7. Final verification: full suites of `middleware/v4`, `metaengine/v4`, `system/v4` and their
   consumers, `GT_LINT_MODULES=... nix run .#lint-scope`, `cmd/doc-check`, `check-canonical-facts`,
   `check-md-go`, `check-file-size`, `nix run .#verify-ci`.
8. CI workflow defects reported by the infra reviewer: `paths-ignore` skips doc gates on doc-only
   pushes (`.github/workflows/ci.yml:4`), golden-freshness job hides test failures
   (`ci.yml:974`), benchmark regression gate cannot fail (`benchmarks.yml:207`), pre-commit doc-only
   shortcut (`.githooks/pre-commit:64`). Reviewer confidence: verified or likely; not re-checked.

## d) TOTALLY FUCKED UP

1. **master is red and stays red until this working tree lands.** 111 of 141 CI jobs at
   `28db93fe7`; 30 consecutive push runs without a success. Root cause: the wave-prep commit
   `12c42cb6d` bumped pins without tidy. The repair exists only in the working tree. The
   auto-commit daemon committed nothing all session (first edit 08:13, still uncommitted at 11:23).
   Severity: blocks all CI signal. Workaround: commit and push the tree.
2. **The published tags of the 2026-11-24 wave point at `12c42cb6d`, whose `go.sum` files are not
   tidy.** Every module tagged in that wave fails a `GOWORK=off` build from its own tag (consumers
   are not affected; building the repo at those tags, and the cold-cache gate, are). Needs a patch
   re-tag by the owner, or acceptance.
3. **A guard was silently dead.** `TestSiblingModulePinsResolve` checked nothing for most modules
   since the `/v4` path move and false-failed elsewhere. Fixed locally (uncommitted). I have not
   audited the other meta-tests for the same class.
4. **Two hard host crashes during the session** (`journalctl --list-boots`: previous boots end at
   08:57:47 and 11:05:51 with no shutdown records). Cause unknown. My heavy local jobs (a nix
   build, then a cold `go test` compile) were running at both moments on a box that already ran at
   load 68 when the session started; I cannot rule out that I contributed. Consequences: `/tmp`
   scratch lost (CI logs, scripts, reviewer findings of the first hours; rebuilt under
   `~/.cache/gcl-review`), nine reviewers interrupted twice, one test file lost and rewritten.
5. **The reviewer fleet cannot run at the needed size.** zai: rate limit at 4 to 5 concurrent
   (subagents share the pool with this session); synthetic `syn:large:text`: "Too many concurrent
   requests" at 8, "exceeded your subscription rate limits" at 5 to 6; hyper, gemini and
   huggingface: payment required. 21 of 31 reviewer launches failed or were interrupted. Roughly
   USD 0.15 to 0.30 per completed scope on synthetic.
6. **My own mistakes this session.**
   - Diagnosed "the wave tags do not exist" from a wrong tag-name shape and wrote it into the plan
     before checking the module cache and the proxy. Corrected in my understanding within the
     hour; the plan file still carries the wrong sentence.
   - Kept all scratch in `/tmp` (tmpfs) on a box that had crashed the day before.
   - Three malformed tool commands (one runaway string, two quote errors). No side effects.
   - Added exported API to `system/v4` and did not regenerate the api golden in the same edit,
     which is ground rule 5 of `AGENTS.md`.
   - Ran a nix build and a cold full-package `go test` concurrently with nine reviewers on a
     loaded production box.
   - Started the fan-out with 9 concurrent reviewers before measuring the provider limits; the
     first pilot took about three hours because each file was read in 300-line pages.

## e) WHAT WE SHOULD IMPROVE

1. **Pin bumps must not land without tidy.** The documented class (gotchas, 2026-09-11) happened
   again on a 105-file scale and cost the whole matrix. Make `scripts/pin-sweep.sh` (or the
   release flow) refuse to finish while any module's `go mod tidy -diff` is non-empty, and gate it
   on the release branch before tags are cut. Impact: 104 red legs for a day. Category: Quality.
2. **Do not tag from an untidied tree.** The wave's tags froze untidy `go.sum` files into the
   proxy. Run the per-module tidy before `batch-release.sh`, not after.
3. **Meta-tests need a liveness proof.** A guard that returns early on an empty result reads as
   green for weeks (the pin-drift test; earlier, the package-surface and file-size gates). Every
   gate script already carries `--self-test`; Go meta-tests need the same: one planted-fault case
   per guard. Impact: silent loss of the only tripwire for a known incident class.
4. **Review scratch belongs under `~/.cache`, never `/tmp`, on this host.** And heavy local jobs
   run one at a time, `nice -n 15`, `-p 1`, on a box that hosts live services and crashes under
   load.
5. **Size the fan-out to the quota before launching.** One pilot, then 2 concurrent on synthetic;
   batch file reads (about 600 lines per call) from the start. The protocol with batched reads
   finished a 6,400-line scope in about 15 minutes.
6. **`system/v4` exports grew by five symbols for two owner issues.** Right call for additive
   fixes, but the package sits at 235 exported symbols against a planned v5 split; route further
   additions through the split instead of re-pinning again.
7. **Tag naming has two live shapes** (`event/v4.15.3` canonical, 10 legacy
   `metaengine/sqliteengine/v4/v4.9.x`). Any tool that derives tag names must strip the major
   suffix; the repaired test now does, `scripts/*` should be audited for the same assumption.

## f) Up to 50 things we should get done next

| #  | Task                                                                                                                                                                                                                                              | Impact   | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Regenerate the api golden and re-pin the package-surface baseline for the five new `system/v4` exports                                                                                                                                            | Critical | S      | Quality       |
| 2  | Run full GOWORK=off suites for `middleware/v4`, `metaengine/v4`, `system/v4`, `systemscenario/v4`, `systemtest/v4`, `system/integration/v4`, `benchkit/v4`                                                                                        | Critical | M      | Quality       |
| 3  | Commit the working tree (175 modified, 6 untracked) and push; confirm CI run goes green                                                                                                                                                           | Critical | S      | Cleanup       |
| 4  | Re-run `scripts/bump-cqrs-lint.sh` (serialized) and re-pin the vendorHash if needed                                                                                                                                                               | Critical | S      | Bug           |
| 5  | Fix `ApplyIdempotent`: record the dedup key only after a successful apply, without growing `store.go`                                                                                                                                             | Critical | M      | Bug           |
| 6  | Verify and fix reviewer P1 `execute.go:95` (unclassified errors count toward engine quarantine, against the documented contract)                                                                                                                  | Critical | M      | Bug           |
| 7  | Verify and fix reviewer P1 `sqlite_dlq.go:318` (unparsable stored event id replaced by a fresh ULID, fabricating an event for replay)                                                                                                             | Critical | M      | Bug           |
| 8  | Verify and fix reviewer P1 `dlq.go:169` (`DeadLetterStore.Delete`: memory deletes globally, SQLite deletes nothing)                                                                                                                               | High     | M      | Bug           |
| 9  | Verify and fix reviewer P1 `spatial.go:63` (spatial index ignores the collection argument)                                                                                                                                                        | High     | S      | Bug           |
| 10 | Verify and fix reviewer P1 `planned_filter.go:64` (IN filter dropped for non-`[]any` values, unfiltered rows returned)                                                                                                                            | High     | M      | Bug           |
| 11 | Verify and fix reviewer P1 `infer_composite.go:249` and `infer_sort.go:42` (composite key reuse on update events; Go field names used where engines need JSON paths)                                                                              | High     | M      | Bug           |
| 12 | Verify and fix reviewer P1 `typed_reader_cursor.go:44` (`ScanPage` ignores `WithDefaultLimit`)                                                                                                                                                    | High     | S      | Bug           |
| 13 | Verify and fix reviewer P1 `auto_named_events.go:59` and `:111` (nil and pointer samples panic instead of failing at registration)                                                                                                                | High     | S      | Bug           |
| 14 | Verify and fix reviewer P1 `rules.go:37` (plan rules read `Store.queries` and `Store.engines` without the lock, race)                                                                                                                             | High     | S      | Bug           |
| 15 | Verify and fix reviewer P1 `replicator.go:176` and `runtime_backend.go:239` (re-applied folds double-count; filtered rebuilds replay unfiltered)                                                                                                  | High     | M      | Bug           |
| 16 | Verify and fix reviewer P1 `projection_builder.go:92` and `:204` (second Evolution for a result type silently replaces the first; two samples for one wire type decode as the last)                                                               | High     | M      | Bug           |
| 17 | Verify and fix reviewer P1 `collection.go:86` (collection reads silently drop rows that fail to decode)                                                                                                                                           | High     | M      | Bug           |
| 18 | Verify and fix reviewer P1 `export_import.go:78` (Import doc claims Apply replay, writes rows directly and skips the rest)                                                                                                                        | High     | M      | Documentation |
| 19 | Verify and fix reviewer P1 `host_replay.go:15` (`RegisterAndWait` documented to block, returns immediately)                                                                                                                                       | High     | S      | Bug           |
| 20 | Repair the four tautological metaengine tests the reviewer flagged (`cost_assignment_test.go:30`, `cross_engine_meta_test.go:126`, `cross_engine_adt_test.go:73`, `cursor_nonnumeric_test.go:29`, `live_latency_phase3_test.go:205`)              | High     | M      | Quality       |
| 21 | Finish the projectionhost half of #132 (warning when a worker dies with the Start context) in a new file, offsetting lines in `host.go`                                                                                                           | High     | S      | Bug           |
| 22 | Implement #138 (record `[]event.Type` literals passed to `event.NewEvents` in the cqrs-lint registry)                                                                                                                                             | High     | S      | Feature       |
| 23 | Document the await contract for #134 on `TimeAdvances` and `Then` in `systemscenario/v4`                                                                                                                                                          | Medium   | S      | Documentation |
| 24 | Cut patch tags for the 2026-11-24 wave from a tidied tree (owner)                                                                                                                                                                                 | High     | M      | Cleanup       |
| 25 | Apply the plan errata (tag claim, scope counts) and a dated close-out note                                                                                                                                                                        | Medium   | S      | Documentation |
| 26 | Fix the CI workflow defects: `paths-ignore` on doc pushes, golden-freshness job hiding failures, benchmark gate that cannot fail, pre-commit doc-only shortcut                                                                                    | High     | M      | Bug           |
| 27 | Prune the 12 stale tag-audit baseline entries                                                                                                                                                                                                     | Low      | S      | Cleanup       |
| 28 | Reproduce and fix the erraudit iroh cgo failure                                                                                                                                                                                                   | Medium   | S      | Bug           |
| 29 | Make the release flow refuse to tag while any `go mod tidy -diff` is non-empty                                                                                                                                                                    | High     | S      | Quality       |
| 30 | Audit the other api-stability meta-tests for the empty-result-means-green class; add one planted-fault case per guard                                                                                                                             | High     | M      | Quality       |
| 31 | Apply the docs truth fixes listed in b7 (AGENTS.md, module map, SKILL.md heading, reviews index, DOMAIN_LANGUAGE block, `.go-version`)                                                                                                            | Medium   | M      | Documentation |
| 32 | Write CHANGELOG `[Unreleased]` rows for every fix of section a and run `scripts/check-changelog-symbols.sh`                                                                                                                                       | High     | S      | Documentation |
| 33 | Add FEATURES.md rows where the inventory moved (`ReadyCheck`, `Unlimited`, `Exact`)                                                                                                                                                               | Medium   | S      | Documentation |
| 34 | Harvest all open findings into `TODO_LIST.md` (reviewer-reported items marked as unverified)                                                                                                                                                      | High     | M      | Cleanup       |
| 35 | Comment on #132 to #141 with verified status; close #133, #135, #139, #141 once tagged                                                                                                                                                            | Medium   | S      | Cleanup       |
| 36 | Write the HTML review report, its series index row and the `AGENTS.md` report entry, with honest coverage numbers                                                                                                                                 | High     | M      | Documentation |
| 37 | Continue the every-file review with the saved protocol at 2 concurrent reviewers: remaining 26 tier-1 scopes first (core/v5, event/v4 remainder, system/v4 remainder, systemscenario/v4, record, id, command, query, dispatcher, metadata, dedup) | High     | L      | Quality       |
| 38 | Then tiers 2 and 3: engines and storage (about 45 scopes), cqrs-lint (about 20), catalog (about 20), remaining modules (44), scripts and docs (11)                                                                                                | Medium   | L      | Quality       |
| 39 | Save the review kit (protocol, manifest, scope lists) in the repo so a later session can resume without rebuilding it                                                                                                                             | Medium   | S      | Cleanup       |
| 40 | Decide #136 against the declined `System.WaitReady` ruling and record the outcome                                                                                                                                                                 | Medium   | S      | Documentation |
| 41 | Decide whether v5 detaches worker and timer lifetime from the Start context (the first option of #132)                                                                                                                                            | Medium   | S      | Feature       |
| 42 | Investigate the two host crashes of 2026-11-25 (kernel log, memory pressure) independently of this session                                                                                                                                        | High     | M      | Bug           |
| 43 | Re-check the review index rows 10 and 11 against their reports and correct them                                                                                                                                                                   | Low      | S      | Documentation |
| 44 | Run `GT_LINT_MODULES="metaengine/v4 middleware/v4 system/v4 cmd/api-stability/v4" nix run .#lint-scope` and `nix run .#check-file-size` on the changed set                                                                                        | High     | S      | Quality       |
| 45 | Run `cmd/doc-check` and `nix run .#check-md-go` after the docs edits                                                                                                                                                                              | Medium   | S      | Quality       |

## g) Questions I can not answer myself

1. **Commit and push.** 181 files sit uncommitted and the auto-commit daemon has not picked them
   up since 08:13. I am not allowed to commit or push without your word. Say "commit" (and whether
   to push) and I land the tree as one reviewed commit, or tell me to leave it for the daemon.
2. **Budget for finishing the every-file review.** At about USD 0.15 to 0.30 per scope on
   synthetic, the remaining 170 scopes cost roughly USD 25 to 50 and 15 to 20 hours at the quota
   that did not trip the limit (2 concurrent). I could not find a quota or spending ceiling for
   `synthetic.new` in the repo or the Crush config. What is the ceiling I may spend, and should
   the review continue in the background across sessions?
3. **Context lifetime on the v4 train (#132).** Today a request- or test-scoped context passed to
   `System.Start` silently stops projection workers and managed timers. I kept the binding and
   made it loud. Detaching (workers live until `Close`) is the cleaner contract and the issue's
   first option, but it changes behavior for 38 direct fleet consumers. Detach on v4, or keep
   warn-only until v5?

---

_This report is a snapshot. Open items of section f belong in `TODO_LIST.md`; harvest them with
`docs-health` HARVEST rather than re-deriving them from this file._
