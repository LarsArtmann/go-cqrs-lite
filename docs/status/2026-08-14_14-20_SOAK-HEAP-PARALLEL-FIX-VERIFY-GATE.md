# Session Status: Soak Heap-Pollution Fix + Verify Gate — 2026-08-14 14:20

Session start context: handoff from `2026-08-14_12-40_CODEC-CLEANUP-PREEXISTING-FAILURE-TRIAGE.md`.
Goal: kill the last verify failure (`TestSoak_AutoCRUDByConvention` heap growth), get `nix run .#verify` GREEN, commit, run remaining gates.

---

## a) FULLY DONE (this session)

1. **Root-caused the soak failure — NOT a leak.**
   - Standalone run: `45,650 events, 500 keys → 68,624 bytes heap growth (0.1 MB)` — PASS with 15MB budget. ~500x under budget.
   - Root cause: `RunAutoCRUDSoak` (and 3 sibling tests) assert on **process-global** `runtime.ReadMemStats().HeapAlloc` while running `t.Parallel()`. In the verify harness (~300 parallel tests in the same metaengine test binary), concurrent tests' live allocations land in the same global heap snapshot → measured "growth" of 63MB was **other tests' data**, misattributed as a leak.
   - Failure was probabilistic (scheduling-dependent): my own full-package run passed once before the fix.
2. **Fixed 13 files — removed `t.Parallel()` from every heap-measuring test:**
   - `metaengine/soak_autocrud_test.go`, `soak_10m_test.go`, `soak_record_test.go`, `stress_test.go` (also dropped the now-unneeded `//nolint:tparallel`)
   - All 8 engine soak callers: `bboltengine/`, `pebbleengine/`, `sqliteengine/`, `badgerengine/`, `tursoengine/`, `pgengine/`, `dgraphengine/`, `duckdbengine/soak_autocrud_cgo_test.go`
   - Each got a `// NOT parallel:` doc comment stating why.
   - `enginetest/soak.go` — `RunAutoCRUDSoak` doc comment now documents the caller contract: MUST NOT `t.Parallel()` because the measurement window must be exclusive. Sequential tests never overlap other tests in Go, so removing `t.Parallel()` guarantees exclusivity.
3. **Verified the fix:**
   - metaengine package: 3x separate non-race invocations — all `ok` (~15s each)
   - metaengine `-race` — `ok` (149s)
   - Local engine modules (bbolt, pebble, sqlite, badger): all `ok`; pg/dgraph/turso/duckdb vet clean (runtime-skipped engines)
   - All 13 edited files `gofmt`-clean (deliberately avoided repo-wide `nix fmt` — known gci conflict)
4. **Pre-commit diff audit of the whole working tree (86 modified files + 1 untracked):**
   - Go diffs contain ONLY: import regrouping (gci), comment additions, `t.Parallel()` removals, whitespace
   - `storage/memory/command_store.go` confirmed byte-identical to HEAD (debug Printfs fully reverted in prior session)
   - Untracked: the prior session's status report doc only

## b) PARTIALLY DONE

- **`nix run .#verify`** — first full run **completed** (job 117) but I piped it through `tail -60`, so only the lint phase tail was visible: every module `0 issues` through `flightrecorder`. I could NOT see the final phases (Check Arch → Doc Check) or the ✅ line — verdict unknown. Relaunched with phase-marker filtering (job 11C) — **still running at report time**.

## c) NOT STARTED

1. Commit the 86 modified files (+ untracked status doc) once verify is GREEN
2. Post-commit `go build -tags "goexperiment.jsonv2" ./...` (daemon-commit rule)
3. Remaining gates: `#vulncheck`, `#check-coverage`, api-stability golden check (`#check-arch`, `#check-depguard`, `#check-duplication`, `#check-api-stability` run inside `#verify`)
4. Module release tagging (esp. `metaengine/sqliteengine` — `register.go` self-registration unpublished, latest tag v4.0.1 predates it; blocks all standalone driver-registry consumers)
5. `codec/` deletion lifecycle decision
6. Porting ~40 follow-up tasks from the 12-40 report into TODO_LIST.md

## d) TOTALLY FUCKED UP

1. **Verify output capture (this session's own mistake).** I piped the first background verify through `tail -60`, destroying phase progression and the pass/fail verdict. Cost: a full duplicate 10+ min verify run. Correct pattern: run unfiltered in background, grep the saved output afterward.
2. **`-count=3` against the ginkgo suite** — wasted a run: ginkgo `RunSpecs` refuses to run twice per process. Should have known from AGENTS.md BDD notes; correct pattern is 3 separate invocations (which I then did).
3. **Scoping gap:** I searched for heap-measuring parallel tests **only inside metaengine/**. Other modules (storage/pebble, storage/bbolt, dedup, watch_leak-style tests elsewhere) may hide the same global-heap + `t.Parallel()` anti-pattern. Unchecked.

## e) WHAT WE SHOULD IMPROVE

1. **Enforce the heap-measurement contract mechanically.** Comments document it; nothing prevents reintroduction. Options: a cqrs-lint rule (file uses `runtime.ReadMemStats` ⇒ no `t.Parallel()` in any test func in that file), or a tiny repo-wide grep in a check script.
2. **AGENTS.md gotcha entry missing** — "heap-measuring tests must never be t.Parallel()" is exactly the durable, hard-to-discover lesson AGENTS.md exists for. Not yet written (pending — will do with the commit).
3. **The whole 63MB "leak" was a false positive that cost two sessions of investigation.** A per-test isolation convention (or process-isolated soak runners) would have made this failure impossible rather than merely rare. The 10M soak's own budget comment even says "generous headroom for ... parallel test load" — the codebase already half-knew this and shipped the flake anyway.
4. **Background job hygiene:** always run long gates unfiltered; filter on read. (Same class as (d1).)
5. **Prior session carryovers still open:** `nix fmt` vs golangci-gci import-grouping conflict WILL recur on every future import addition (95+ files re-fixed last session); deserves a permanent fix (treefmt gci settings or fmt exclusion), not repeated manual repair.

## f) NEXT (up to 50, ordered)

**Immediate pipeline (blocked only on verify job 11C finishing):**
~~1. Read job 11C output → confirm ✅ GREEN end-to-end~~ done at 5f2198189 (first fully green verify since ADR-0128; three GREENs since)
~~2. Commit: soak parallel fix + 85 codec-cleanup files (one commit, message covering docs/pins/golden/gci/soak)~~ done - landed via the daemon commits 5127039da + 875bb689b
~~3. Add AGENTS.md gotcha entry: heap tests never parallel (same commit or follow-up)~~ done - AGENTS.md gotcha 'Heap-measuring tests must NEVER be t.Parallel()' (docs-health session 2026-08-15)
~~4. Post-commit `go build -tags "goexperiment.jsonv2" ./...`~~ done - build phase green in every verify since 5f2198189
5. `nix run .#vulncheck` (per-module standalone — also catches version-sequence breaks) <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist)
~~6. `nix run .#check-coverage` (coverage drift)~~ done - runs inside #verify (GREEN 3x since 5f2198189)
~~7. Confirm api-stability golden is current (`#check-api-stability` ran inside verify; regen only if drifted)~~ done - Check API Stability phase green in every verify since
~~8. Audit OTHER modules for `ReadMemStats` + `t.Parallel()` co-occurrence; fix any found~~ done - repo-wide audit came back clean (13 files were the full set; recorded in TODO_LIST heap-contract item)
9. Decide + implement enforcement for the heap/parallel rule (cqrs-lint rule vs check script) <- OPEN. TODO_LIST 'Code Quality' (Enforce the heap-measurement contract mechanically); enforcement-flavor choice = g) Q3

**Releases (blocked on user answers below):**
10. Tag `metaengine/sqliteengine` v4.0.2 (publishes register.go driver self-registration) <- OPEN. TODO_LIST 'Release / Tagging'
11. Audit every module with uncommitted exported-surface changes vs its latest tag; tag each (monotonic semver + ancestry — see AGENTS.md Release) <- OPEN. TODO_LIST 'Release / Tagging'
12. Verify `go get`/build for `example/taskmanager` standalone (GOWORK=off) after sqliteengine tag <- OPEN. gated on the engine tags - TODO_LIST 'Release / Tagging'
13. Confirm codec/v4 v4.4.0 propagates through all proxy-resolved consumers <- NOT-DO - codec/ shims deleted at 5127039da (ADR-0128); consumers moved to external go-codec; propagation concern moot
14. CONTRIBUTING.md release-process pass if tagging surfaces friction

**codec/ lifecycle (blocked on Q1):**
~~15. User decision → either schedule codec/ deletion (major-bump path + deprecation window) or keep documented alias forever~~ done - decided: deletion, executed at 5127039da (ADR-0128)

**Technical debt / hygiene:**
16. Fix `nix fmt` (treefmt/golines) vs golangci gci import-grouping conflict at the tooling level <- OPEN. TODO_LIST 'Code Quality' (nix fmt vs gci tooling fix)
~~17. Port ~40 follow-up tasks from `2026-08-14_12-40` report §f into TODO_LIST.md with triage~~ done - 12-40 f-list resolved item-by-item by the docs-health annotation pass 2026-08-15 (see its Resolution appendix)
18. Consider making soak runners process-isolated (subtest-free, own binary via build tag) if flakiness ever recurs <- **Won't implement - sequential-by-design fixed the class; revisit only if flakiness recurs.**
19. metaengine suite: ginkgo suite + stdlib soak tests coexist; consider consistent harness (low priority)
20. Re-check `verify-parallel.sh` script interplay with now-sequential soaks (wall-time impact was nil-to-positive in measurement, but confirm on CI)
21. Sweep remaining `//nolint:tparallel` instances repo-wide for staleness (one was removed this session)
22. Update `docs/sessions/SESSION_MILESTONES.md` with codec-migration + soak-fix outcomes <- OPEN. TODO_LIST 'Docs Honesty' (SESSION_MILESTONES reconciliation)
~~23. Update FEATURES.md/CHANGELOG.md entries if soak behavior change merits a line~~ done - CHANGELOG [Unreleased] 'Fixed - repo gates' entry covers the 13 heap-parallel fixes
24. Add a CI annotation or doc note that soak tests are sequential-by-design (prevents "make it parallel for speed" regressions)

_(24 items — the remaining carried tasks are enumerated in the 12-40 report §f and deliberately not duplicated here.)_

## g) QUESTIONS (cannot be answered from the repo)

1. **codec/ deletion:** Are there external consumers of `go-cqrs-lite/codec/v4` you know of beyond this monorepo? This decides delete-with-deprecation-window vs keep-alias-forever. (Carried from 12-40 report, still unanswered.)
2. **Release tagging now or later:** Should I tag module releases (sqliteengine minimum, ideally the full audit) immediately after the commit lands, or do you want to review the 86-file diff first? (Carried, still unanswered.)
3. **Heap/parallel enforcement:** For the no-`t.Parallel()`-on-heap-tests rule — cqrs-lint rule (fits the 202-rule linter, ~a day of work) vs a 5-line grep in a check script (cheap, less discoverable)? Preference?

---

**Bottom line:** The blocking failure was a measurement artifact, not a leak — fixed at the root across 13 files, independently verified (3x + race). Verify gate re-run in flight; commit + gates + releases are the remaining runway, gated on the verify result and 3 answers.

---

## Resolution (2026-08-15)

19 of 24 items carry verdicts. The immediate pipeline (1-9 minus the
pre-tag items) closed across `5127039da`, `875bb689b`, `5f2198189` and the
docs-health pass (AGENTS heap gotcha, CHANGELOG entry, repo-wide
heap/parallel audit). codec/ lifecycle resolved by deletion (ADR-0128,
`5127039da`), mooting item 13. Release items (10-12) and g) Q2 live in
TODO_LIST "Release / Tagging" + ROADMAP Open Questions #1. Open-unrouted:
14, 19, 20, 21, 24 (minor hygiene). Stays active.
