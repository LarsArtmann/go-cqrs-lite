# Adoption Wave T01–T13: Execution Status (brutal self-review)

> **Date:** 2026-10-10 03:40 CEST · **Session:** Full Execution Mode on
> [`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md)
> (owner GO 2026-10-09 ~22:00; G1 rulings Q1/Q2/Q3 accepted as recommended).
> **Interrupted at:** T14 (tag wave), first `batch-release.sh` invocation — hard-failed on
> dirty tree (my own uncommitted `untagged-trains.txt` edit). Recoverable in minutes.

## a) FULLY DONE (verified green)

1. **T01 — deadlock evidence pack** (`docs/evidence/2026-10-09_deriver-bus-deadlock.md` +
   raw stacks `docs/status/deriver-deadlock-stacks-2026-10-09.txt`): NOT reconstructed from
   reading — I wrote a temp repro test (sync deriver on `sys.Bus()`), let it hang 3s, dumped
   ALL goroutine stacks, then deleted the repro. The captured cycle is precise: outer publish
   holds watermill GoChannel's **per-topic subscriber mutex** while waiting for ack
   (pubsub.go:144); the single event-loop goroutine's nested publish blocks acquiring that
   same mutex (pubsub.go:108); delivery goroutine parked on the busy loop (pubsub.go:408).
   Also pinned: `Close()` does not unwind it (the nested lock acquisition has no
   closing-select escape). Option memo (a/b/c) + async error-surfacing design included.
2. **T02 — tag-wave readiness audit**: the audit **changed the wave**. The plan's wave
   (systemscenario + system + scheduling) was INCOMPLETE: `system/schema_test.go` consumes
   the UNPUBLISHED `schema.Event`/`EventSchema` API (published v4.5.2 lacks it — GOWORK=off
   build failure captured), and the saga fixture consumes the new `deriver.WithAsyncDispatch`.
   True order: **schema → deriver → system → scheduling → systemscenario**, separate
   batch-release invocations (same-batch sibling pin limitation documented in the script).
   Manifest: [`docs/planning/2026-10-09_tag-wave-manifest-bdd-harness.md`](../planning/2026-10-09_tag-wave-manifest-bdd-harness.md).
   Release-scripts smoke (`nix run .#check-release-scripts`): green.
3. **T03 — companion baselines**: cqrs-htmx 26/26 modules green — after I found 3
   build-broken (dashboardui/systembridge, examples/system-demo, systemadapter) from a
   MISSING `schema/v4` local replace in their go.work (local `system/` consumes the untagged
   schema API; replaced, same local-dev pattern as their existing block). go-appkit 10/11
   green incl. the `cqrs` pilot module; `integration` red on 2 PRE-EXISTING tests (pin-drift
   on their docs module + a docs-composition content assert) — externals, untouched.
4. **T04 — HARVEST**: TODO_LIST build section closed per docs-health rules (done items
   deleted, not struck), FINDING item struck with the G1 ruling, 10 surviving residue rows
   added with sources; ROADMAP +5 v5 arcs (journal-tailed deriver host, systemscenario
   absorbs scenario/v4, v4-shim-over-v5, Axon DCB note, presets growth).
5. **T05 — ADR-0154** (`docs/adr/0154-deriver-async-dispatch-and-journal-tailed-host.md`):
   D1 WithAsyncDispatch now (per-event goroutine, onError callback, WithoutCancel,
   first-error-stop), D2 journal-tailed host as v5 direction, alternatives table,
   self-review vs ADR-0136/0142/0153/0028.
6. **T06 — FEATURES.md**: harness section (10 feature rows, Experimental), system Clock
   row, scheduling WithClock row, maturity-matrix row. README gates: deprecated-clean;
   23 broken links ALL in core/v5 (externals).
7. **T07 — SKILL.md**: testing pointer (recipes §2.43, advanced §6.10) + recipes line
   extended. doc-check: 1184 refs valid (4 ambiguity warnings = known core/v5 externals).
8. **T08 — `deriver.WithAsyncDispatch`** (commit `b4d350ba1`): option + error-handler type
   - dispatchAsync (per-event goroutine, WithoutCancel, first-error-stop, nil-onError
     drops errors — loudly documented). 4 race-clean unit tests: handler-returns-before-
     dispatch, default-stays-synchronous (pinned), error surfaces to callback + stops chain,
     cancelled context does not kill async dispatch. api golden +2 exports.
9. **T09 — fixture flip**: systemscenario saga fixture goroutine → WithAsyncDispatch with a
   panicking onError (failed derived dispatches fail tests loudly). Full suite race-clean.
   README constraint note → points at ADR-0154 + evidence pack.
10. **T10–T13 — cqrs-htmx user train FULLY migrated** (authored commit `749ddbb5` + daemon
    `5346865b`): 7 harness tests cover Credentials (add/remove + field asserts), TOTP
    (enable/disable), ExternalAccounts (link/lookup/unlink + the 2026-09-09 unlink-index
    regression + re-link ownership follow), UserDelete (await-vanish + AllUsers empty),
    AllUsers, AuditLog (global ≥2 / per-user event types + OccurredAt / recent(1)), and the
    FULL missing-lookup set (9 ErrNotFound lookups + scan-style empty-nil). Legacy twins
    DELETED per Q3: declarative_test.go 1235 → 896 lines; suite 2.231s → 0.835s. New
    `awaitNotFound` helper expresses "row must eventually vanish" over ThenQueryFunc.
    **Also caught:** the legacy AllUsers test registered the SAME stream twice with a
    literal `"user%d@example.com"` (no Sprintf — latent bug, duplicate-register rejection
    was apparently never hit because... it WAS a different uid per loop iteration, my
    misread — the literal duplicate EMAIL was the latent part). Migrated version uses 3
    distinct users/emails.

## b) PARTIALLY DONE

1. **T14 — tag wave (0/5 tags cut)**: everything before the cut is done (order discovered,
   untag-policy entry for `deriver` removed with rationale — tag-on-first-consumer
   satisfied by systemscenario's module graph, owner-approved via Q1; pre-tag standalone
   suites green for schema/deriver/scheduling/scenario + in-workspace green for
   system/systemscenario; GOWORK=off reds for system+systemscenario are the EXPECTED
   pre-tag state and the reason the order exists). The first `batch-release.sh` invocation
   FAILED: "working tree has uncommitted changes" — `scripts/untagged-trains.txt` (my edit)
   - `.agents/skills/go-cqrs-lite/SKILL.md` (dirty — not mine this time; the daemon or a
     concurrent agent touched it after my 7696eee95 commit). Nothing was tagged; no damage.
2. **Authored-commit discipline (again)**: 2 of 3 intended authored commits this session
   lost their bulk to daemon races (evidence pack absorbed pre-add; migration bulk absorbed
   between edit and commit). Only T08 (`b4d350ba1`), T06+T07 (`7696eee95`), harvest
   (`80648e122`), and the user-train cleanup line (`749ddbb5`) carry authored messages.

## c) NOT STARTED

T15 (drop replaces — scope grew, see d4), T16–T27 (hardening A/B, appkit layer-2, godoc
examples, presets, property pack, chaos+SSE, cqrs-lint rule, codemod suggestion, fleet
rollout, watermill loud-fail, final verify + retro).

## d) TOTALLY FUCKED UP

1. **Ran the release script on a dirty tree.** I edited `untagged-trains.txt` minutes
   before invoking `batch-release.sh` and did not commit. The script's guard caught it —
   zero damage, one wasted invocation, ~2 min. Root cause: no "commit BEFORE any
   release-script invocation" step in my loop.
2. **`.agents/skills/go-cqrs-lite/SKILL.md` re-dirtied by someone else mid-flight** and I
   did not notice before the invocation (the script's dirty-tree check aggregates ALL
   changes — mine and theirs). Lesson: `git status` immediately before ANY tree-state-
   sensitive script, not "recently".
3. **Plan-authoring miss (inherited, now surfacing):** the 14:49 plan's wave spec named 3
   modules; the true wave is 5 (schema + deriver missing). T02 caught it — that is what T02
   was FOR — but the plan's G2/G3 gate wording ("3 modules") will mislead any future reader
   who skips the manifest. The plan addendum must restate the wave.
4. **Scope creep in companion replaces:** fixing the companions' schema gap added a NEW
   replace line in cqrs-htmx go.work and go-appkit go.work each. Justified (3 modules were
   build-broken; same pattern as their existing blocks) — but the adoption-wave "drop 4
   replace files" (T15) is now "drop 4 files + decide the fate of 2 schema lines" (htmx:
   permanent local-dev pattern, keep; appkit: marked drop-with-the-pair, keep consistent).
5. **One edit-protocol violation:** attempted an edit on cqrs-htmx go.work without a prior
   View (tool blocked it; corrected on retry). Zero impact, but it is the exact "edit
   before read" failure mode AGENTS.md forbids.

## e) WHAT WE SHOULD IMPROVE

1. Release invocations deserve a pre-flight one-liner: `git status --short` must be EMPTY
   (or stashed) — encode it in the wave manifest as a checklist row for T14.
2. Beat the daemon by making "edit → test → commit" a SINGLE bash invocation whenever
   possible (multiple tool calls = window for absorption).
3. When a plan names a release set, the readiness audit should DIFF the set against actual
   module-graph dependencies — "changed since last tag" per module is not enough; consumers
   of untagged API are the real wave members (this session's schema discovery).
4. `.agents/skills/go-cqrs-lite/SKILL.md` got dirtied by a non-session writer while I owned
   a pending release invocation — concurrent agents and release scripts share one tree;
   the verify-lock protects runs, not reads of tree state. Nothing to fix mechanically
   today; just awareness.

## f) Up to 50 things to get done next

**Resume T14 (the wave — all evidence already gathered):**

1. Commit the dirty files (`untagged-trains.txt` + inspect the SKILL.md dirt FIRST —
   rule 6: never commit changes I didn't author without reading them; if it is a foreign
   in-flight edit, wait or stash-aside, do not absorb blindly).
2. Cut schema/v4 v4.6.0 (batch-release) → push tag → `tag-release.sh --smoke` → proxy
   fetch probe.
3. Cut deriver/v4 v4.4.0 → push → smoke.
4. Cut system/v4 v4.12.0 (pins schema v4.6.0 automatically once visible) → push → smoke.
5. Cut scheduling/v4 v4.7.0 → push → smoke.
6. Cut systemscenario/v4 v4.0.0 (pins system v4.12.0 + deriver v4.4.0) → push → smoke.
7. Remote tag COUNT assert (5 new tags, count-based).
8. CHANGELOG wave section (symbols gate: every `pkg.Symbol` cited must exist).
9. `bash scripts/check-versions-manifest.sh --check` — manifest/markers may need the new
   tags; update per its output.
10. `nix run .#verify-ci` or the meta-tests if the wave touches cmd/api-stability goldens.

**T15 (drop replaces — rescoped):**
11. cqrs-htmx: drop `systemscenario/v4` replace from go.work AND `systemadapter/go.mod`;
keep their schema line (permanent local-dev pattern).
12. go-appkit: drop the pilot trio (systemscenario + system + schema lines in go.work) +
`cqrs/go.mod` replace; tidy; full suites green on published tags.

**T16–T20 (hardening/pilots/examples/presets):** items 13–30 are the plan's own fine
breakdown (F16.1–F20.4) — nothing new discovered that changes them, EXCEPT:
13. T16's timeout last-error surfacing should ALSO cover the `awaitNotFound` pattern this
session introduced in cqrs-htmx (query-error-as-success hides the last non-matching
error — surface it in the detail string).
14. T20 presets: `Memory()` should default `RecommendedMemoryDeployment`-shaped single-
engine memory config; keep `SQLite(t)` DSN-per-test.

**T21–T26:** plan fine breakdown F21.1–F26.4 stands. Additions from this session:
15. T21's fold-vs-read-model invariant: use the systemscenario task fixture (TaskView vs
decider fold) — direct reuse.
16. T26 loud-fail: the evidence pack's exact mechanism (per-topic mutex) means detection
belongs in `EventBus.Publish` (publisher-depth flag is NOT enough — the cycle is
cross-goroutine via the mutex; a publish-depth goroutine-local works because the nested
publish happens ON the event-loop goroutine which CAN carry the flag). Verify with the
evidence-pack repro shape before committing to a mechanism.

**T27 + tail:**
17. `nix run .#verify` (expect: my modules green; core/v5 arch-lint + V007 + doc-check
aliases + metaengine file-size red — externals itemized).
18. `nix run .#check-md-go` on the new docs (evidence pack, ADR-0154, manifest).
19. Plan addendum: DONE/PARTIAL per section + the 5-module wave restatement.
20. Companion full suites post-T15.
21. TODO_LIST: strike the wave-execution residue rows that T14–T26 close.
22. CHANGELOG [Unreleased] → version sections for all five tags.
23. Retro into `docs/agents/gotchas-*`: "commit before release scripts", "wave = consumers
of untagged API, not just changed modules".
24. cqrs-htmx + go-appkit daemons will absorb the T15 drops — authored commits where
possible (single-call pattern).
25. Re-check `TestCatalogHasExpectedCounts` (cqrs-lint) if systemscenario's catalog entry
drifted (14:40 §f31 residue row).
26. go-appkit `integration` module: in go.work `use` set CONTRADICTING its own charter
comment (charter: NOT a member; use list: `./integration`) — THEIR bug to file/flag,
surfaced in the baseline record.
27. The `.agents/skills/go-cqrs-lite/SKILL.md` foreign dirt: read it, judge on merits,
commit or leave for its author (never blind-absorb).
28–50. The plan's F16–F26 fine tasks verbatim (38 items; see
`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md` §5) — no further
decomposition needed at report time.

## g) Questions I CANNOT figure out myself

1. **Tagging the concurrent agent's in-flight schema work:** schema/v4.6.0 carries their
   untagged `Event`/`EventSchema` API (their ROADMAP row literally says "adoption rides
   the next schema/v4 tag", and the module is standalone-green — but their session may
   still be mid-flight on it). Proceed tagging their surface in MY wave, or wait for their
   session to signal done? (Default on resume: proceed — their own docs ask for the tag.)
2. **`deriver` untag-policy removal:** I removed it per tag-on-first-consumer (consumer =
   systemscenario's module graph; owner approved Q1 which requires the tag). The ghost-
   verdict T02 sign-off (kill/keep/absorb) is still pending elsewhere — confirm deriver's
   re-entry into release trains stands.
3. **go-appkit `integration` charter violation** (in `use` set despite charter saying it
   must not be): mine to flag to their owner, or leave strictly alone since their suite is
   green either way?
