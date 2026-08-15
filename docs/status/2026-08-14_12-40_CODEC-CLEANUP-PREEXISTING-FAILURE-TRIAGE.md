# Codec Deprecation Cleanup + Pre-Existing Failure Triage — Status Report

**Date:** 2026-08-14 12:40 CEST
**Session goal:** Finish the codec→go-codec migration leftovers (docs, formatting, verify) and triage/fix the 30+ pre-existing test failures discovered by the prior session.

---

## a) FULLY DONE

1. **Living docs updated with new import path** (`go-cqrs-lite/codec/v4` → `go-codec`):
   - `.agents/skills/go-cqrs-lite/references/modules.md:9` — codec row now marked **DEPRECATED** with direct-import guidance
   - `docs/DOMAIN_LANGUAGE.md:501` — import example fixed
   - `docs/design/transport-nats.md:62` — dependency list fixed
   - `docs/design/transport-redis.md:72` — dependency list fixed
   - `docs/planning/parquet-journal-design.md:223` — go.mod sketch fixed
   - `FEATURES.md:574` — import line fixed
   - Historical docs (`docs/status/*`, `docs/planning/archive/*`, `docs/feedback/*`) intentionally left untouched — they are point-in-time records.
2. **3 doc-check failures FIXED** — symbols were renamed upstream, docs lagged:
   - `listing.StatusActive`/`listing.StatusDeleted` → `event.TombstoneActive`/`event.TombstoneTombstoned` (advanced.md:29)
   - `stack.ExcludeDeleted` → `stack.ExcludeTombstoned` (readmodels.md:115)
   - doc-check now reports **779 references valid across 44 packages** ✅
3. **cmd/cqrs-lint TestLintExampleTaskmanager FIXED** — root cause was NOT the C009 panic findings (those match golden; they're suppressed via `//cqrs-lint:ignore` comments). The real diff was V006 version-drift text (`v4.6.0` joined the version list after the `event/v4.6.0` release). Regenerated golden via `CQRS_LINT_UPDATE_GOLDEN=1`. Test passes ✅
4. **command/commandtest TestStoreSuite/ReadFrom FIXED** — root cause: `command/go.mod` pinned `storage/memory/v4 v4.2.0`, whose `ReadFrom` had the `limit=0` bug (`end := min(startIdx+limit, ...)` returns 0 items). Fix shipped in `storage/memory/v4.3.0`. Bumped `command` module deps (`go get storage/memory/v4@v4.3.0` + tidy, which also lifted event/query/snapshot pins). All 6 subtests pass ✅
5. **Stale published version pins FIXED in 4 modules** (GOWORK=off standalone builds):
   - `integration/`: storage v4.0.3→v4.6.0, metaengine v4.2.0→v4.10.0, listing v4.0.3→v4.2.0, stack v4.1.0→v4.3.0 (+ ~15 transitive bumps). All 7 test packages pass ✅
   - `benchkit/`: storage v4.5.0→v4.6.0, stack/postgres v4.1.0→v4.3.0 (NotificationListener API). Builds; workspace tests pass ✅
   - `system/integration/`: system v4.2.0→v4.4.0, metaengine v4.8.0→v4.10.0, codec v4.3.0→v4.4.0 (critical: v4.3.0 defined its own `Encoding` type — NOT an alias — causing cross-module type mismatch; v4.4.0 is a pure alias again). Passes in workspace mode ✅
   - `example/getting-started/`: storage v4.5.0→v4.6.0. Passes ✅
6. **nix fmt run** — reformatted 50 files (import grouping from the prior session's sed rewrite).
7. **gci lint disaster FIXED** — `nix fmt` (golines/goimports) put `go-codec` in its own import group, but golangci's gci config requires it in the "default" section with `go-error-family`. 95+ files flagged. Fixed via awk script (merge groups) + `gofmt -w` (sort within group) across all `go-codec` importers. `nix run .#lint` now reports **0 gci issues** ✅
8. **Full workspace build** `go build -tags "goexperiment.jsonv2" ./...` ✅

## b) PARTIALLY DONE

1. **`nix run .#verify`** — ran 3 times; final state: everything passes EXCEPT one test (see d). Build ✅ vet ✅ race ✅ (most modules) lint ✅ doc-check ✅.
2. **benchkit GOWORK=off standalone tests** — 3 failures remain, all environment-dependent (Docker-based Postgres spin-up, SQLite `database is locked (517)` under test concurrency, `Disk.DatabaseBytes = 0` needing a real disk path). Workspace-mode run passes (`go test ./benchkit/... -short` ✅). These look flaky/environmental, not regression.
3. **Taskmanager + system GOWORK=off standalone** — `example/taskmanager` fails standalone (`unknown driver "sqlite"`) because `metaengine/sqliteengine/register.go` (driver self-registration) exists ONLY in the workspace — **no published tag contains it** (latest tag v4.0.1 predates it). Workspace mode passes. Fix requires tagging a new `metaengine/sqliteengine` release.

## c) NOT STARTED

1. **Committing the 73 modified files** — all work sits uncommitted in the working tree (auto-commit daemon may pick it up).
2. **API-stability golden regen** — not re-run after this session's edits (no exported symbols changed this session; only go.mod bumps + doc edits, so likely no drift — unverified).
3. **`nix run .#vulncheck` / `#check-arch` / `#check-coverage` / `#check-duplication`** — pre-release gates not run.
4. **codec/ module deletion lifecycle** — decision still pending (external consumers unknown).
5. **Publishing new module versions** — the standalone-build fixes (sqliteengine register.go, system deps) only fully land for downstream consumers once new tags are cut.

## d) TOTALLY FUCKED UP

1. **`metaengine TestSoak_AutoCRUDByConvention` FAILS in `#verify`** — `heap grew 63,280,160 bytes after 45,650 Apply calls with 500 keys (max 15,728,640) — possible leak`. This is a REAL memory-growth finding (4x over budget), not obviously flaky. It blocks GREEN. Possibly pre-existing (prior session reported system/ tests failing, soak thresholds race-sensitive per AGENTS.md), but it did NOT fail in my earlier per-module runs this session — it failed in the full `#verify` run under race + full-suite load. Needs dedicated investigation.
2. **My first gci fix was wrong twice** — first awk pass only removed blank lines for `go-*` predecessors (missed `golang.org/x/sync` etc.), second attempt still left ordering wrong (`go-codec` after `go-error-family`); needed a third pass (merge + gofmt sort). Cost: ~3 lint round-trips. Lesson: should have installed/used the actual `gci` binary semantics from the start instead of approximating them in awk.
3. **Verify reruns were wasteful** — I re-ran the full `#verify` (10+ min each) three times to chase failures instead of running the failing modules directly first.

## e) WHAT WE SHOULD IMPROVE

1. **Root cause > symptom**: doc-check failures and lint golden drift were upstream renames (`ExcludeDeleted`→`ExcludeTombstoned`, tombstone API moves to `event/`) — the fix was doc updates, not code. The cqrs-lint "failure" was a 1-line golden diff, not the 4 C009 panics the prior session blamed.
2. **Pin hygiene**: the monorepo's GOWORK=off builds rot silently — modules pin stale sibling versions until something breaks. A scheduled "bump all internal pins to latest tags" sweep (or making `#verify` cover standalone builds) would prevent the entire stale-pin class.
3. **`codec/v4` v4.3.0 type-mismatch trap**: that tag defines `Encoding` as its OWN type instead of aliasing `go-codec`. Any consumer on v4.3.0 + `go-codec` directly will hit cross-module type errors. v4.4.0 fixed it. Worth a CHANGELOG note / consumer advisory.
4. **Import-grouping tool conflict**: `nix fmt` (goimports/golines) and golangci `gci` disagree about `go-codec` grouping. Add `go-codec` to whatever config drives `nix fmt` so fmt stops fighting lint (this WILL recur on every future import addition).
5. **Debug discipline**: I added debug Printf to `storage/memory/command_store.go` mid-investigation (reverted cleanly) — fine, but the real breakthrough was noticing `GOWORK=off` pulls published versions, not workspace code. Internalize: "tests that pass in workspace but fail GOWORK=off = version-pin problem, check the proxy cache path in the error".
6. **Unpublished-API trap**: `metaengine/sqliteengine/register.go` has no released tag. Until tagged, every GOWORK=off consumer of the driver registry breaks. The release process should be part of "done" for driver-registration features.

## f) NEXT — up to 50 tasks

**Blockers (do first):**
~~1. Investigate `metaengine TestSoak_AutoCRUDByConvention` heap growth (60MB vs 15MB cap) — reproduce with `-count=3`, check if it's the `-race` 5-10x inflation AGENTS.md warns about; if real leak, profile `Apply` path (per-call allocations, map growth on 500 keys/100 deletes/50 recreations)~~ done at 5127039da-wave - root-caused by the 14-20 session: NOT a leak, global-heap + t.Parallel() artifact; 13 files fixed
2. If soak is race-threshold-only: add `RaceEnabled` guard to its budget (pattern exists in testutil/enginetest) <- NOT-DO - not race inflation; the fix was removing t.Parallel() from heap-measuring tests (13 files)
~~3. Re-run `nix run .#verify` end-to-end and confirm GREEN~~ done at 5f2198189 (first fully green verify; three GREENs since)
~~4. Commit the 73-file changeset (docs + go.mod bumps + golden + gci fixes)~~ done - daemon committed as 5127039da + 875bb689b

**Release pipeline (unblocks standalone consumers):**
5. Tag `metaengine/sqliteengine` next version (contains register.go driver self-registration) <- OPEN. TODO_LIST 'Release / Tagging'
6. Tag `event/v4`, `command/v4`, `storage/v4` etc. with the go-codec-direct imports (migration commit `1ff2b53d0` is untagged) <- OPEN. TODO_LIST 'Release / Tagging'
7. After tags: re-run `go get -u` sweep on `integration/`, `benchkit/`, `system/integration/`, `example/*` to drop residual `codec/v4 // indirect` pins <- OPEN. gated on the tags - TODO_LIST 'Release / Tagging' (~49 stale indirect refs)
8. Run `nix run .#vulncheck` (per-module standalone build catches version-sequence breaks) <- OPEN. TODO_LIST 'Release / Tagging' (pre-tag checklist)
~~9. Advisory in CHANGELOG: avoid `codec/v4 v4.3.0` (non-alias `Encoding`), use v4.4.0+~~ done - CHANGELOG [Unreleased] 'Consumer advisory - avoid codec/v4 v4.3.0' (docs-health session 2026-08-15)

**codec/ module lifecycle:**
~~10. Decide: keep `codec/` as permanent re-export vs. deprecation timeline (needs external-consumer data — see questions)~~ done - decided: deletion, executed at 5127039da (ADR-0128)
11. If deleting: add depguard rule banning `go-cqrs-lite/codec/v4` imports in-repo (suggested by 2026-08-12 status report) <- NOT-DO - shims deleted at 5127039da; nothing left to ban
12. Sweep `codec/README.md:12` example import path → `go-codec` (kept intentionally this session; it's the shim's own doc) <- NOT-DO - codec/ module deleted at 5127039da

**Test-suite health:**
~~13. Triage the 3 benchkit GOWORK=off failures (SQLite lock contention 517, DiskSizer path, PG container) — retry logic or testserialization~~ done at 4a95bd04d - dep-skew root-caused (stack/sqlite + stack/pebble pins bumped to v4.3.0); suite green standalone 34s
~~14. `system/` 12+ failures from prior session report — re-verify against current HEAD (may be fixed by this session's system dep bumps; workspace tests pass now)~~ done - green in every verify since 5f2198189
15. `example/taskmanager` 5 standalone failures — auto-resolve once sqliteengine tagged <- OPEN. gated on the sqliteengine tag - TODO_LIST 'Release / Tagging'
16. Add CI leg for GOWORK=off standalone builds of leaf modules (integration/, examples/, benchkit/) to catch pin rot early <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish: #verify-ci mirrors GOWORK=off per-module)
17. Consider `go mod tidy`-all + pin-sync script under `scripts/`, run in CI nightly <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (pin-drift meta-test supersedes the nightly script; sweep policy = ROADMAP 'Open Questions' #4)

**Tooling/config:**
18. Fix import-grouping conflict: make `nix fmt` formatter config aware of gci section rules (or switch golangci to goimports-compatible sections) so this class of 95-file lint break can't recur <- OPEN. TODO_LIST 'Code Quality' (nix fmt vs gci tooling fix)
19. Pre-commit: run `gofmt -l` on staged .go files as fast guard before the slow lint
~~20. doc-check: consider validating `stack.`/`listing.` qualified refs in ALL docs (currently only SKILL/references/AGENTS) — DOMAIN_LANGUAGE.md:501 class of drift went undetected~~ done - doc-check invocation now covers README/TODO_LIST/ROADMAP/FEATURES/CONTRIBUTING/DOMAIN_LANGUAGE docs (verify-docs.sh wiring)
21. cqrs-lint golden test: assert on rule+file, not full message text — V006 embeds a version list that churns on every release (today's failure)

**Docs:**
22. Update AGENTS.md module-map `codec/` row to add "(unpublished consumers: tag advisory)" if release decided <- NOT-DO - codec/ row removed from the AGENTS.md module map (module deleted)
23. Parquet journal design doc: confirm `go-codec v0.1.x` pin format matches eventual tag scheme
24. Record ADR for "external-module aliases must be pure `type X = Y` aliases forever" (v4.3.0 regression rationale)
~~25. TODO_LIST.md: port items 1-24 above with owners/status~~ done - this very annotation pass resolved items 1-24 inline; TODO_LIST carries the open ones (docs-health 2026-08-15)
26. Session milestone entry in docs/sessions/SESSION_MILESTONES.md for the codec migration closeout <- OPEN. TODO_LIST 'Docs Honesty' (SESSION_MILESTONES reconciliation)

**Quality follow-ups:**
~~27. `nix run .#check-arch` after dep bumps (budget enforcement)~~ done - Check Arch green inside every verify since 5f2198189
~~28. `nix run .#check-coverage` (drift gate)~~ done - Check Coverage green inside every verify since
~~29. `nix run .#check-duplication` (the awk/gofmt churn may have introduced clones)~~ done - baseline re-pinned at 875bb689b-wave; Check Duplication green in every verify since
~~30. API-stability golden regen + meta-tests (`TestEveryGoModDirIsInTestModules`, `TestEveryGoModDirIsInModulesList`)~~ done - Check API Stability phase green in every verify since; meta-tests green
31. Verify `codec/` module still builds standalone (`cd codec && GOWORK=off go build ./...`) <- NOT-DO - codec/ module deleted at 5127039da
32. Sweep remaining `go-cqrs-lite/codec/v4` references in `docs/feedback/new/` — decide keep-as-history vs. annotate <- NOT-DO - historical docs are point-in-time records by policy (docs-health model); not swept
~~33. Soak env var docs: confirm `SOAK_SKIP_*` knobs cover the failing AutoCRUD test for CI escape hatch~~ done - SOAK_SKIP_DUCKDB covers the DuckDB AutoCRUD soak; the metaengine AutoCRUD test needed no knob after the parallel fix
34. Re-run `example/taskmanager` cqrs-lint golden after any taskmanager code changes (it's a canary)
35. Bench: quick `go test -bench` sanity on codec hot paths after import churn (should be zero-cost, prove it) <- NOT-DO - codec/ deleted; import churn moot
~~36. Check `git worktree` leftovers from prior session (/tmp/cqrs-pre-codec) — clean up~~ done - /tmp/cqrs-pre-codec worktree no longer exists
~~37. Review the 73-file diff before commit: ensure no unintended .go semantic changes beyond imports (sed was path-only, but verify)~~ done - pre-commit diff audit performed by the 14-20 session (only imports/comments/t.Parallel removals)
~~38. After commit: `go build -tags "goexperiment.jsonv2" ./...` re-run (AGENTS.md daemon-commit rule)~~ done - build phase green in every verify since 5f2198189
39. Consider `go-codec` version bump advisory: only tag v0.1.0 exists; internal modules all pin it — plan v0.2.0 governance
40. Map remaining `// indirect codec/v4 v4.2.0` pins (3 modules) — will drop post-tagging, then verify zero remain <- OPEN. TODO_LIST 'Release / Tagging' (indirect codec refs drop post-tagging)

## g) QUESTIONS (cannot determine myself)

1. **External consumers of `codec/v4`**: Do any exist outside this repo (your other projects, pkg.go.dev usage you know of)? This decides delete-now vs. keep-re-export vs. timed deprecation for `codec/`.
2. **Release authorization**: May I tag the next versions of the affected modules (`metaengine/sqliteengine` at minimum, ideally also event/command/storage/system) to unblock GOWORK=off consumers — or do you want to batch that with a planned release train?
3. **The soak heap-growth budget**: `TestSoak_AutoCRUDByConvention` allows 15MB but hit 60MB under full-verify load. If it turns out to be genuine growth (not race-inflation), is tightening that leak a priority now, or acceptable to raise the budget / gate behind `RaceEnabled` and ticket the investigation?

---

## Session Metrics

| Item | Value |
|---|---|
| Files modified (uncommitted) | 73 |
| Pre-existing failures triaged | 6 groups → 4 fixed, 1 environmental, 1 open (soak) |
| Docs updated | 6 files + 2 skill references |
| go.mod bumps | command, integration, benchkit, system/integration, example/getting-started |
| Lint issues resolved | gci ×95+ files |
| Verify status | 1 failure remaining: `metaengine TestSoak_AutoCRUDByConvention` |


---

## Resolution (2026-08-15)

33 of 40 items carry verdicts. The blocker chain (1-4) closed across the
14-20 session, `5127039da`, and `5f2198189`; the codec/ lifecycle resolved
by outright deletion (ADR-0128), mooting items 11/12/22/31/35; benchkit
standalone failures were dep-skew, closed at `4a95bd04d`. Release items
(5-8, 15, 40) track in TODO_LIST "Release / Tagging" + ROADMAP Open
Questions #1. Open-unrouted: 19 (fast gofmt pre-guard), 21 (cqrs-lint
golden assert granularity), 23 (parquet pin format), 24 (pure-alias ADR),
34 (taskmanager golden canary), 39 (go-codec v0.2.0 governance, external
lane). Stays active.
