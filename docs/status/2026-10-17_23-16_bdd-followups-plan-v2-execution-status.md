# BDD Follow-Ups Plan v2 — Execution Status (2026-10-17 23:16)

> Session: owner approved plan v2 at 20:36 ("GET SHIT DONE — the WHOLE TODO
> LIST") after the 20:13 self-review + 20:18/20:31 plan docs. This report
> covers the execution run 20:36→23:15 ONLY (what I did, what I noticed in
> passing). Verdict up front: **11 of 16 tracked items fully done and
> verified, 1 verified-green-but-needs-doc-close-out, 3 not started (M06,
> M12, M25-tail); the mine-list across every repo gate is EMPTY, the race
> flake is CLOSED with evidence, and the tag wave's hardest prerequisite
> (projectionadapter v4.6.0 with the corruption fix) was shipped mid-session
> by a parallel release wave.**

Plan: [`docs/planning/2026-10-17_20-18_SUPERB-bdd-depth-followups-pareto-plan.md`](../planning/2026-10-17_20-18_SUPERB-bdd-depth-followups-pareto-plan.md)
(live execution addendum §8 kept current as mediums landed).
Decision pack: [`docs/reviews/2026-10-17_bdd-depth-followups-owner-decision-pack.md`](../reviews/2026-10-17_bdd-depth-followups-owner-decision-pack.md).

## a) FULLY DONE (verified)

1. **M00 — quiescence + re-baseline (20:37).** 3/3 clean `git status`
   samples; M05 mode = FULL. Foreign-red attribution re-basedlined: file-size
   offenders grew 2→4 (all foreign, attributed by commit:
   `recipes_catalog_meta3.go` ← `f1f0e82d5`, `exporter.go` growth ←
   docserver session). Recorded in plan addendum §8.
2. **M01 — cqrs-htmx full suite vs the workspace tree: GREEN (exit 0).**
   Companion base `3f67fe92`, temp `GOWORK=/tmp/gclhtmx/go.work`
   (both use-blocks absolutized; zero edits to either tree; companion left
   clean). The corruption reclassification + nextVersion fix are now
   consumer-verified against the biggest fleet consumer.
3. **M02 — go-appkit full suite vs the workspace tree: GREEN.** Root + all
   10 submodules (cqrs, health, realtime, errorpages, flightrecorder,
   flightrecorderhealth, otel, security, systemd, docs) green; their
   `integration/` module deliberately excluded per its pin charter.
4. **M03 — api-stability golden proof: CLEAN.** "API surface OK: 6233
   exports verified", zero drift, no `--update`. TestEvery red is FOREIGN —
   worktree-proven identical 76-module go.sum untidiness at pre-session
   commit `445130ee7` (root-cause candidate: authored re-pin `78ae14659`,
   Oct 15).
5. **M04 — GOWORK=off standalone legs: analyzed, verdict = pin-version gap
   (not a defect).** Both modules' standalone reds are the pins pointing at
   pre-fix versions: the DLQ spec fails against pinned pre-fix
   projectionadapter EXACTLY as designed (worker crash-loops, DLQ never
   fires) — captured as the tag-wave justification + ordering input
   (projectionadapter → system → systemscenario + pin bumps).
6. **M05 — #verify-fast triaged; mine-list EMPTY.** 4 runs: two unblocked by
   my own repairs (ADR index, taskmanager), one polluted by a concurrent
   158-file foreign flush (daemon 20:57-20:59 — the final clean-tree run is
   authoritative). Final triage: 140+ packages ok; foreign = api-stability
   (go.sum ×76), cqrs-lint rules born-red (tests did not exist at
   pre-session commit), cqrs-upgrade E2E (go.sum class); flake verdict =
   projectionhost `TestLiveCheckpoint_CrashWindowIsBounded` load-sensitive
   (passes solo); lint 0 issues in my modules; duplication = exactly the 2
   known foreign groups.
7. **M05b — race reproduce-or-close: 5/5 GREEN under quiet window → CLOSED.**
   `quiet-window-run` (window opened at load 3.22), `go test -count=5 -race
   ./systemscenario/...` → ok 3.7s. The 11.78s-timeout class was machine
   load, not a race bug. TODO row ready to close with this evidence.
8. **M07 — taskmanager dep restore: grew into a 4-part repair, all green.**
   (i) missing `servertiming` imports restored in 3 files + `go mod tidy`
   made the require direct; (ii) dot-mangled identifiers un-mangled
   (`systemRegisterQuery`→`system.RegisterQuery(srv.Sys,…)`,
   `idStreamID`→`id.StreamID`, `parseIDString`→`id.ParseStreamID`) — a
   foreign codemod had stripped package dots; (iii) bare-ID laggers migrated
   across tests AND a REAL PROD BUG: `handleGetTask` keyed
   `TaskReader.Get` by prefixed `String()` → every GET /api/tasks/{id}
   404'd under the bare-ID contract; (iv) suite green (workspace + spot
   -race), standalone build/vet green, tree-wide `#build` UNBLOCKED.
9. **M08 — owner decision pack written** (docs/reviews/, one artifact):
   D1 resolved-by-events (directive + parallel wave + companion evidence),
   D2 erroring-folds ruling requested with a v5 recommendation, C1
   classification proposal for the apply-wrap (cause-family pass-through)
   and `buildEventDecoder`'s three wraps (Rejection/Corruption/Corruption).
10. **M09 — depth pins: all three GREEN.** OCC spec now asserts
    `ThenError(event.ErrVersionConflict)` (sentinel survives the opError
    chain via errors.Is); upcaster spec asserts the journal read-back
    `SchemaVersion()==2` (the read path IS the upcast path — contract
    holds); DLQ spec pins `ProjectionName=="projections"`. Retry-journey
    half = D2-deferred with memo note (accepted by DoD).
11. **M10 — payload-trap sweep v2: CLEAN.** No `any`-blob Unmarshal /
    DeepEqual-on-decoded / %v-assertion instances in systemscenario prod or
    recipes; `sse.go:140` decodes into caller-typed `V` (sanctioned pattern).
12. **M11 — coeffect FAQ + cross-link DONE.** New FAQ entry ("Why did
    declaring a Schema suddenly fail validation…") + recipes §2.43 paragraph
    (prose-only, no catalog churn). doc-check green (verify-docs leg).
13. **M17 — foreign routing rows: 7 items (a-g) written into TODO_LIST**
    with gate commands: file-size ×4, mesh-demo budget, dup ×2, ~~taskmanager
    (fixed by me)~~, go.sum ×76 + cqrs-upgrade E2E, cqrs-lint born-red
    rules, and the `go 1.27.2` directive wave vs minor-form lockstep.

## b) PARTIALLY DONE

1. **M25 — lockstep close-out.** Plan addendum §8 IS current (M00-M11
   entries + parallel-session events recorded); TODO rows updated live for
   M11 + M17 + the race row's evidence pointer. REMAINING: close the
   companion-verification row (needs M06), close the race row formally,
   14-49 addendum pointer, completion banner — blocked on M06/M12 below.
2. **Docs gates after tonight's edits.** doc-check (full), TestRecipes, and
   changelog-symbols were green for MY last full runs, but the M09 pin
   edits + FAQ/recipes + memo landed after the final verify-docs run;
   `nix fmt` has not run over tonight's tail edits. One 15-min gate sweep
   owed before the tag wave.

## c) NOT STARTED

1. **M06 — `#check-coverage` + `scripts/erraudit-sweep.sh`** over the
   changed set (systemscenario + projectionadapter). 45 min.
2. **M12 — THE TAG WAVE (critical path remainder).** projectionadapter
   v4.6.0 ALREADY tagged by the parallel wave (fix included — verified in
   the tag). Remaining: tag `system` (13 untagged commits; requires pin
   bump projectionadapter v4.5.3→v4.6.0 to ship the fix through
   system.New), tag `systemscenario` (28 untagged commits: depth specs +
   nextVersion fix), `pin-sweep.sh` (so M04's standalone legs go green),
   `check-example-standalone.sh --build` (graph-native unblock), proxy
   smoke. Pre-flight MUST re-check for a live parallel release (one was
   actively tagging at 20:47-20:48).
3. **CHANGELOG/FEATURES rows for tonight's work** (taskmanager repair, ADR
   index fix, FAQ/pins) — owed at the M12 wave close per the close-wave
   procedure.

## d) TOTALLY FUCKED UP

Nothing red of mine stands, but six honest failures:

1. **Plan-ordering flaw I executed into: M05 needed M07.** verify-fast ran
   twice into walls (taskmanager build break, ADR index) that the plan had
   sequenced AFTER the verify phase (P0 verify → P1 taskmanager). A
   gate-blocking repair belongs BEFORE the gate run — I should have
   triaged `#build` once, fixed, then run the sweep. Cost: two ~7-min
   verify-fast cycles.
2. **ADR index over-correction.** My first fix indexed 0099a as a DUPLICATE
   (the pre-existing row differs only in title case; my grep pattern
   `0[0-9]{3}-` cannot match `0099a-`) → 159-vs-158 → another wasted
   verify-fast leg. I grepped before reading the checker's counting rule.
3. **Malformed M05b invocation.** `nix run .#quiet-window-run -- bash -c …`
   without the required `--` separator → exit 2; I read the script usage
   only AFTER it failed instead of before.
4. **An edit in the plan addendum REPLACED the M12-consequence paragraph**
   instead of appending (old_string = the paragraph, new_string omitted
   it). Caught by re-reading and restored within one step — but the
   failure class (replace-that-deletes) is exactly what the memory rules
   warn about.
5. **Raced a foreign 158-file flush.** My first solo cqrs-lint triage ran
   during the daemon's 20:57-20:59 absorb window; I briefly suspected my
   own taskmanager repair for `TestLintExampleTaskmanager` redness. The
   worktree-attribution discipline caught it (test passes on the settled
   tree; golden = 31 = current findings), but I paid ~4 extra attribution
   cycles that a "is the tree churning right now?" check before triage
   would have avoided.
6. **Unverified edits sat in the tree for ~90 minutes.** The M09 pins were
   written at ~21:25 but the interruption paused execution before their
   test run; they were only verified at 23:15 (green — lucky, not
   disciplined). Daemon commits absorbed untested edits; had they failed
   to compile, the tree carried red tests through three daemon commits.

## e) WHAT WE SHOULD IMPROVE (process)

1. **Quiescence is not a phase, it's a pre-flight per gate.** M00 said
   "quiescent at 20:36"; the tree then flushed 158 files at 20:57 and a
   release wave tagged 6 modules at 20:47-20:48. Any expensive gate
   (verify-fast, tag wave) should sample `git status` + recent tag list
   immediately before starting, not trust an hour-old verdict.
2. **"Fix the build before running the build gate" is now a rule.**
   `nix run .#build` failing on taskmanager was KNOWN from LSP diagnostics
   at session start — M07 could have run first at zero cost (the plan's
   dependency edge M05→M07 existed in reality, just not in the graph).
3. **Worktree-attribution is cheap insurance and it keeps paying.** Three
   attribution proofs this session (go.sum ×76, cqrs-lint born-red,
   taskmanager pre-existence) each cost <2 min and prevented wrong-fix
   churn. Should be the DEFAULT first move on any foreign-looking red.
4. **The dot-stripped-identifier incident suggests the cqrs-upgrade
   codemod has a destructive mode** (taskmanager's
   `system.RegisterQuery`→`systemRegisterQuery` etc.). Whoever owns
   cqrs-upgrade should add a fixture proving dotted qualified identifiers
   survive; the example module silently didn't compile for 2 days because
   of it.
5. **Bare-ID migration is still rolling the fleet.** taskmanager was the
   third+ site (projectionadapter tests, then prod handleGetTask). The
   companions' integration modules (cqrs-htmx `integration_test/`,
   go-appkit `integration/`) pin published tags and will meet this at the
   NEXT tag wave — worth a proactive grep for `StreamID…String()` used as
   KEY (not display) across the fleet before tags land.
6. **76 modules' go.sums disagree with `go mod tidy`.** Whatever
   `78ae14659`'s re-pin script did, TestEvery and cqrs-upgrade E2E are now
   structurally red until someone re-tidies or the pin script and the
   tidy-check agree on one truth. This is the single biggest foreign
   blocker to a green `#verify-ci`.

## f) NEXT — up to 50, ranked roughly by Pareto

**Finish the plan (do first):**
1. M06: `nix run .#check-coverage` over changed modules vs pin.
2. M06: `scripts/erraudit-sweep.sh` over systemscenario + projectionadapter
   changed files; fix ≤5m findings on sight, file the rest.
3. Docs sweep for tonight's tail edits: `nix fmt`, doc-check full run,
   TestRecipes, changelog-symbols.
4. M12 pre-flight: `git status` ×3 + `git for-each-ref refs/tags` freshness
   + verify-lock check (parallel release session may still be live).
5. M12: bump system's projectionadapter pin v4.5.3→v4.6.0 (+ go mod tidy,
   self-sourcing env).
6. M12: tag `system` via `scripts/tag-release.sh system <ver> "<desc>"`
   (read script flags first; --dry-run preview).
7. M12: tag `systemscenario` (depth specs + nextVersion fix).
8. M12: `bash scripts/pin-sweep.sh` (bump sibling pins + cqrs-lint goldens
   refresh + stale-pin check + standalone verify).
9. M12: re-run M04 legs (GOWORK=off standalone both modules) — must be
   green now; this is the F12 smoke DoD.
10. M12: `bash scripts/check-example-standalone.sh --build` — graph-native
    unblock; close its [BLOCKED] row with receipts.
11. M12: proxy smoke — scratch module `go get` each new tag.
12. M12: companion `go get` smoke (cqrs-htmx, go-appkit against new tags).
13. CHANGELOG rows for: taskmanager repair (Fixed), depth pins (Added to
    the depth-pack row), FAQ/recipes (docs row), ADR index fix.
14. FEATURES.md: taskmanager row update (bare-ID migration complete).
15. M25: close companion-verification row + race row (evidence in place).
16. M25: 14-49 addendum pointer + plan completion banner + final addendum
    entry.

**Owner-gated (memo awaits):**
17. D2 ruling (erroring folds v4.x/v5/never) — unblocks the retry-journey
    depth spec.
18. C1 ruling (apply-wrap pass-through + buildEventDecoder three wraps).
19. Whether the dedup-middleware recipes fence (TODO row, plan M12-v1)
    should still happen this wave or ride the next harness touch.

**Foreign drift routing (owners, all rowed in TODO §M17):**
20. catalog file-size ×4 (split or baseline).
21. mesh-demo 8/7 dep budget (bump or review).
22. 2 duplication groups (accept-directives or specview insert).
23. go.sum-untidy ×76 (re-pin script vs tidy reconciliation).
24. cqrs-lint born-red rule tests (perf/resilience/version).
25. `go 1.27.2` 95-file directive wave vs minor-form lockstep (watch
    check-go-version at next #verify-ci).
26. projectionhost `TestLiveCheckpoint_CrashWindowIsBounded` load-sensitivity
    (quiet-window CI treatment or timeout bump — same class as the closed
    systemscenario flake).
27. cqrs-upgrade codemod dot-stripping bug (see e4).

**Depth-spec hardening (from the 20:13 report, still open):**
28. Multi-hop upcast chain spec (v1→v2→v3, DiscordSync `UpcastChain()`).
29. ManualClock-aware TTL documentation in the dedup fixture.
30. OCC impure-decide convention note in the fixtures header.
31. `TestDepth_*` naming-convention note in the harness README.

**Adjacent noticed-in-passing (small, no owner):**
32. taskmanager `setup.go` URL builder now emits bare URLs — verify the
    demo README/docs examples match.
33. `query_http_test.go:243-245` dead code (`raw, _ := io.ReadAll…; _ = raw`
    — the "must not leak" assertion was never actually written).
34. `var _ = time.Second` at query_http_test.go:259 — leftover import
    keeper; delete with the import.
35. plan v2 §7 says "TODO rows added at v1 stay valid" — verify none of the
    killed v1 rows leaked back in.

(35 rows — the honest remaining slots stay empty rather than padded.)

## g) Questions only the owner can answer

1. **D2 (erroring consumer folds):** v4.x additive, v5, or never? This is a
   product-surface decision (growing the declaration API) — the memo
   recommends v5; the retry-journey depth spec waits on it. Full context:
   decision pack §D2.
2. **C1 (projection-boundary error classification):** green-light the
   three-wrap proposal (apply-wrap = cause-family pass-through;
   buildEventDecoder = Rejection/Corruption/Corruption)? Each changes
   worker/DLQ behavior for every system.New consumer — judgment, not
   inference. Decision pack §C1.
3. **Tag-wave authority:** a parallel session's release wave was actively
   tagging tonight (20:47-20:48; adapter v4.6.0 shipped MY fix). Should I
   proceed to tag `system` + `systemscenario` + pin-sweep myself (M12 as
   planned), or is another session's release train mid-flight and I should
   hand it the remainder? I cannot detect "another agent is about to tag"
   from the repo alone — only that none is running this instant.

---

*Arte in Aeternum — report written 2026-10-17 23:16 CEST, immediately after
the interrupted M06-prep point; execution halted per instruction, awaiting
direction.*
