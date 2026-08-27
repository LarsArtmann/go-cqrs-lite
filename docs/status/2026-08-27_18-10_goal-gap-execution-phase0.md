# Status Report: Goal-Gap-Closure Plan — Phase 0 + P05 Executed — 2026-08-27

> **Session:** "Execute the goal-gap-closure Pareto plan" (user approval turn:
> "Break down into actionable steps... repeat until done").
> **Plan of record at session start:**
> [`2026-08-27_16-18_GOAL-GAP-CLOSURE-PARETO-PLAN.md`](../planning/2026-08-27_16-18_GOAL-GAP-CLOSURE-PARETO-PLAN.md).
> **Mid-session supersession:** the 17:35 ALL-TODOS v2 plan
> ([`2026-08-27_17-35_ALL-TODOS-PARETO-PLAN.md`](../planning/2026-08-27_17-35_ALL-TODOS-PARETO-PLAN.md),
> commit `08af6b225`) folded in the two review follow-up waves. Execution
> continued under v2 numbering where the plans overlap; deltas noted below.
> **Concurrent sessions:** two review sessions ran in parallel until ~17:40
> (gap-wave session closed 17:15; deep-review session committed `607e16e71`).
> Modules with foreign dirty files were yielded throughout; tree clean at
> every commit.

## a) FULLY DONE (verified end-to-end)

1. **T01 — release-chain truth (commit `574018277`)**: TODO Release section
   corrected against git evidence. Stranded-commit repair already landed
   (`491379a2b` on master; command/query pin metadata v4.6.0; bench has zero
   pseudo-versions → `4907b6afc` obsolete). Wave-4 claims event v4.8.0 /
   metadata v4.6.0 / metaengine v4.12.0 / storage v4.8.0 already LIVE.
   Pebble+bbolt standalone builds re-verified GREEN (the 🔥🔥 RED TODO item
   was fixed by the 08-22 pin bumps). Deliverable: 7-batch pending tag-wave
   plan ([`2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md`](../planning/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md))
   awaiting user sign-off — **[USER-BLOCKED: tags]**.
2. **T02/P06 — stale-pin sweep (commit `5d7d42ba7`)**: 19 modules bumped to
   latest published sibling tags, every one GOWORK=off build-verified;
   stack/* and transport/* deliberately skipped (v5 deletes them).
   Discovered: `event/v4/eventtest/v4.0.0` + `v4.2.0` are DEAD tags (module
   path lacks /vN suffix; `go mod download` rejects them) — recorded in TODO
   + tag plan §6.
3. **T04 — PG integration isolation (commit `5ec4b1b39`)**: storage,
   storage/relational, and benchkit migrated onto the shared
   `testutil/pgtestcontainer` helper (deleting local copies that shared ONE
   database under explicit DSN); helper gained `AfterRun` (post-run hook for
   snaps.Clean) and PID-qualified database names (parallel test binaries can
   no longer collide — root cause of two flaky integration failures,
   reproduced then fixed). Full `#integration-pg` suite GREEN (3rd run, after
   two environment/collision failures diagnosed). **NOTE: v2 plan moved this
   item to §Declined — the fix was already committed+green by then; flagged
   in TODO_LIST for explicit user revert if the decline rationale
   outweighs.**
4. **T05/P08(a-c) — listing type-driven status (commit `5ec4b1b39`)**:
   `listing.Status` + `StatusClassifier` + `WithStatusClassifier`; wire
   values match legacy TombstoneStatus ints (golden parity verified,
   stream-status.json unchanged); `StreamStatus.Status` retyped; SQL stream
   reader migrated; `StatusMiddleware` Deprecated; skill refs (core/advanced/
   modules) rewritten to the classifier.
5. **T06/P08(d-e) — taskmanager**: verified ALREADY migrated (domain-event
   `evtTaskDeleted` fold, zero stack imports, tests green). Plan premise was
   stale; no work needed.
6. **P05 — listing honesty + system validation (commit `c9e464eda`)**:
   migration-doc recipe rewritten to the shipped classifier (the old
   StatusMiddleware recipe could not work with decider save-then-publish);
   system shutdown-deps validated against the POPULATED engine set (synthetic
   "default"/"projections" accepted — the documented example works again);
   empty names → ErrShutdownDependencyInvalid. Tests + 3× race green.
7. **P04 — watermill at-least-once catch-up (commit `c9e464eda`)**:
   checkpoint advances only on `msg.Acked()` (was: at handoff — the
   at-most-once loss class); Nack stops the subscription with the checkpoint
   left behind; the invariant-bounded 1024-entry dedup ring replaced by a
   last-replayed-ID watermark. Three regression tests (no-ack crash,
   Nack-stop, watermark suppression) + full suite + 3× race green. The
   Close-panic/double-Subscribe sub-item verified already-safe in current
   code (single-sender close, per-Subscribe channels).

## b) IN FLIGHT

- **P06.5/F02.5 — `#verify-ci` matrix mirror**: launched exclusive (nothing
  else heavy running). Result appended below.

## c) NOT DONE (deliberate)

- **P03 — metaengine recHolder race + Record threading**: deferred. It needs
  its full budget (v2 plan: 100min): failing race test, invoke-closure
  Record threading across all fold types (fold.go, auto_naming.go,
  infer_composite.go, record_fold.go dispatch in runtime_backend.go:304 +
  replicator.go:165), then optional `Record` on EventLog entries +
  Backfill/Demote/Verify replay threading. Design note recorded in session:
  keep `SetCurrentRecord` exported (API compat), move dispatch to
  pass-Record-as-argument via the sealed interface's unexported surface.
- **P07 [USER] tags, P17 [USER] v5 cut**: tag plan awaits sign-off.
- **P02.4/5 `#verify-standalone` app + CI leg**: the standalone signal is
  owned by `#verify-ci` (exists, mirrors CI per-module GOWORK=off) — the
  TODO's alternative "explicit decision that CI owns that signal" is
  effectively satisfied; formalize in a later pass.

## d) TOTALLY FUCKED UP (honest log)

1. Two integration runs wasted on environment/dead-mount issues before
   exporting GOCACHE per-command (run 1: `#integration-pg` without env
   redirect → /mnt/buildcache; run 2: cross-process `test_1` DB collision —
   my own per-test DBs racing across parallel package binaries). Both fixed
   (env export + PID-qualified names).
2. One multiedit applied to the wrong duplicate text (classifier landed on
   EmptyJournal's reader instead of List's) — detected by grep audit, fixed.
3. Attempted `Status: listing.Status(statusInt)` before realizing GOWORK=off
   storage resolves the PUBLISHED listing tag (workspace-masking — the exact
   documented class); fixed with the sanctioned sibling replace.
4. First draft of the synthetic-engine test invented a nonexistent
   `ProjectionConfig` literal — read the sealed declaration types before
   rewriting with a real 2-engine setup.
5. Test nil-guard miss: closed-channel nil message dereferenced in a Fatalf
   branch (panic instead of clean failure) — guarded in both new tests.

## e) NEXT (priority order)

1. **P03 metaengine race pair** (design note above) — the last open Phase-0
   sextet member.
2. **P07 [USER]**: authorize the pending tag wave (7 batches, module order in
   the tag plan) — gates the entire v5 deletion wave.
3. **Phase 1 deletions** (P09→P13) once tags flow; P08's listing+taskmanager
   prerequisites are already done (this session).
4. Harvest note: 7 TODO items ticked with evidence this session.

*Reported 2026-08-27 ~18:10 from HEAD `c9e464eda` (verify-ci result pending
below). Master ahead of origin by this session's commits — push policy
unchanged: only on request.*

---

## verify-ci result (appended after completion)

<!--VERIFY_CI_RESULT-->
