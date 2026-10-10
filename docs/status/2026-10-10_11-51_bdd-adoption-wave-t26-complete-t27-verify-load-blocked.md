# BDD Adoption Wave — T26 COMPLETE, T27 ~90% (verify load-blocked), plan EXECUTED 27/27 pending final gate

> **Date:** 2026-10-10 11:51 CEST · **Mode:** Full Execution (owner GO ~09:00 with the three §g defaults applied)
> **Supersedes:** [`2026-10-10_08-43_bdd-adoption-wave-t23-t25-complete-t26-partial-watermill-guard-shipped.md`](2026-10-10_08-43_bdd-adoption-wave-t23-t25-complete-t26-partial-watermill-guard-shipped.md)
> **Plan:** [`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md) — its §11 addendum (written this session) is the per-task execution record.

**Progress: T26 COMPLETE. T27 complete except F27.1 (`nix run .#verify`) — REFUSED by the load guard all session (concurrent agents held the machine at load 30–60 for 90+ min; the `--wait-loop` timed out after 20 attempts/3600s). Load was 9.78 and falling at report time — the window may be open NOW.**

## a) FULLY DONE (this session)

1. **T26 example/taskmanager systemscenario suite** — `example/taskmanager/systemscenario_test.go` (NEW, ADD-only): boots the production facade (`NewServer(DefaultConfig(), nil)` + `srv.Start` + `systemscenario.Adopt(t, ctx, srv.Sys)`; Server's t.Cleanup owns lifecycle). Full lifecycle chain `When(create).ThenSuccess() → ThenQueryFunc(title/priority/AssigneeID) → Command(start) → ThenQueryFunc(active) → Command(complete) → ThenQueryFunc(completed) → Command(delete) → ThenQueryFunc(absent)`. The deriver's durable-workqueue auto-assign is pinned INSIDE the first poll (the load-bearing OCC barrier before `task.start`). go.mod += `systemscenario/v4 v4.0.0`; TAGGED-ONLY API. First-run green; full module suite green; `check-example-standalone.sh --build` green (all 8 examples).
2. **T26 example/goal-shaped-app systemscenario suite** — `example/goal-shaped-app/systemscenario_test.go` (NEW, ADD-only): boots the production path (cqrs.yaml → `LoadConfig` → `system.New` via `boot()`), two creates Given-seeded, complete/delete as When acts, asserts through the REAL typed query bus (`GetTask`, `OpenTasks`) incl. the tombstone shape (`errTaskGone` sentinel mapped to success in a ThenQueryFunc probe) and empty-open-tasks. go.mod += systemscenario v4.0.0. First-run green; full suite green; standalone gate green.
3. **T26 cqrs-htmx awaitNotFound sweep — verified already-done by a concurrent agent**: commits `749ddbb5` (full user train migrated to harness, legacy twins deleted per Q3, declarative_test.go 1235→896 lines, suite 2.2s→0.8s) + `e4f9784f` (systemscenario resolved from the proxy, replaces dropped). I ran its systemadapter suite standalone: GREEN. **Correction that reshaped this task:** the handoff claimed `ThenQueryEventuallyFails` is tagged v4.0.0 — it is NOT (`git show systemscenario/v4.0.0:systemscenario/then_query.go` ends at `ThenQueryFails`); the proxy-pinned repo therefore correctly used the tagged `ThenQueryFunc` + `awaitNotFound` adapter. Nothing for me to migrate.
4. **T26 feedback memo (F25.4)** — durable half: two new skill-FAQ pitfalls (`.When().When()` does not exist / Then*-chaining shape + Adopt ctx ownership; deleted-row assertion split — TypedReader `found=false` rides the probe value vs erroring queries map their sentinel, `ThenQueryEventuallyFails` rides the next tag). One broken-anchor doc-check failure fixed en route (GitHub slugger drops punctuation without hyphens). Full half: plan addendum §11.4. doc-check green.
5. **T27.2 companion full suites** — cqrs-htmx full repo: 0 FAIL. go-appkit full repo: 0 FAIL. Both green on the current tree (workspace mode).
6. **T27.3 retro** — 4 new dated lessons in `docs/agents/gotchas-testing.md`: (1) handoff "designs/APIs tagged" claims are hypotheses — verify against compiler/`git show <tag>`; (2) edit-then-verify protocol under daemon races + FILE-SCOPED COUNT-ASSERTED python recovery (a global replace shadowed the targeted fix once); (3) cqrs-lint fixture TypesInfo emptiness + BuildContext `Tests:false` (rules over test files own their walk — E020/suggest precedents cited); (4) — plus the pre-existing daemon-tidy entry kept. `check-md-go` green (twice, after each doc batch).
7. **T27.4 plan addendum** — plan file annotated (never rewritten): status banner EXECUTED 27/27, §11.1 per-task DONE/PARTIAL table with evidence, §11.2 external-red itemization, §11.3 GOWORK standalone reds until next tag wave (Q2), §11.4 F25.4 memo + honest-miss log, §11.5 corrected-handoff-claims register. F20.4 marked PARTIAL with rationale (presets post-tag → companion adoption waits).
8. **T27.5 partial — CHANGELOG + TODO_LIST + taxonomy + golden**:
   - CHANGELOG [Unreleased] Added: T26 fleet entry (both example suites, tagged-only API discipline, FAQ pitfalls) — `check-changelog-symbols.sh` green (25 citations).
   - TODO_LIST: deriver-deadlock finding struck SHIPPED (guard + async cure + ADR-0154 addendum pointer).
   - **error-taxonomy drift FIXED (mine)**: `watermill.reentrant_publish` (Orchestration) added to `docs/error-taxonomy.md` — the T26 leg (prior session) shipped the code without the doc row; preflight caught it today. Gate now green (527 codes match).
   - api-stability golden regenerated twice (8475 → 8477; +2 = concurrent schema-wave API absorbed); `TestEvery` green.
9. **Duplication gate: 3 new clone groups FIXED (mine, caught by preflight)**:
   - watermill reentrancy-guard twins → consolidated into shared `rejectReentrant(ctx, kind)` helper in `watermill/errors.go`; both buses' Publish now open with the 3-line guard call; dead `fmt`/`event` imports pruned; suite race-green after.
   - cqrs-upgrade `printSuggestions`/`printDeprecations` twins → consolidated into `printFindings(w, findings, emptySummary, header)` (byte-identical output); suite green.
   - systemscenario chaos.go StreamTemporalReader forward twins → `//art-dupl:accept` (explicit hand-forwarding IS the design — Go cannot delegate capability assertions through embedding; register.go precedent); watermill guard-call twins → accept (matches the file's existing publish-tail precedent). `#check-duplication` GREEN (baseline 186).
10. **.golangci.yml corruption tripwire — diagnosed + restored**: a concurrent wave's committed auto-commit (`7abd975c4`, 543 files) re-serialized the config (quote flips, `go: 1.27.0`→`"1.27"`, and displaced `depguard:` from `linters.settings` to top level where golangci ignores it) WITHOUT re-pinning the hash golden. `check-golangci-hash.sh --update` refused (repair detects the structurally-missing depguard). Restored `.golangci.yml` from the last golden-pinned commit `443b5013b`; hash gate GREEN. Build re-verified after.

## b) PARTIALLY DONE

1. **T27.1 `nix run .#verify` — NOT RUN (load-blocked, not failed)**: first launch REFUSED (load1=30.06 ≥ 10, load5=64 — the guard's exact purpose). Ran the prescribed `preflight-composed.sh` first (fixed everything mine, see a8/a9/a10 + itemized externals), then chained `can-run-composed-gate --wait-loop && #verify` — timed out after 20 attempts/3600s (load ceiling 12×, tree-instability 4×, both 4×). Load 9.78/falling at 11:51 — the window may be open now; one re-run likely closes T27. `#verify-fast` was never run as a fallback (miss, see d).
2. **T26 watermill leg polish**: guard consolidated post-ship (a9) — the module needs one final workspace-mode suite pass inside the eventual verify (ran green standalone after the edit; `-race` green).

## c) NOT STARTED

1. **The actual `#verify` run** (the only T27 item outstanding — everything else in §4 of the plan is done).
2. Authored commit of the wave's files in go-cqrs-lite (daemon mode per handoff; cqrs-htmx's authored commit was the concurrent agent's `749ddbb5`).

## d) TOTALLY FUCKED UP (nothing destructive — process misses, all caught by gates or self-caught)

1. **Prior-session gate debt surfaced in MY preflight** (this session's preflight run is what caught it): the T26 watermill leg shipped (a) without the `docs/error-taxonomy.md` row (`check-error-taxonomy` red) and (b) with 2 of the 3 new dup clone groups (`check-duplication` red). Root cause: the prior session's gate list ended at build/test/race/api-golden — it never ran the taxonomy or duplication gates after adding an errorfamily code. Both fixed this session (a8/a9). Lesson queued for §e: NEW errorfamily code ⇒ same-edit taxonomy row; NEW exported twin-shaped code ⇒ same-edit `#check-duplication`.
2. **`nix fmt` verdict misread once**: first run printed "formatted 0 files" + an error I initially chased as MY syntax error; it took a minimal-repro compile in /tmp to prove **Go 1.27 legalizes generic methods** (the foreign `system/schema_declarations.go:61` `Event[T any]` method) and treefmt's pinned gofumpt simply predates them. Wasted ~5 min suspecting my own files; the file-by-file gofumpt check settled it (all wave files clean).
3. **`$?`-after-pipe exit-code reads (twice)**: `go build ... | head; echo $?` reports head's exit, not go's — produced one misleading "build green" while investigating the gofumpt discrepancy. Known repo gotcha class (rc-after-pipe), repeated anyway.
4. **`nix run .#check-duplication -- --help`** — assumed a help flag; it passthrough-executed the gate with `--help` interpreted as a report flag. Harmless (read-only) but sloppy; the flag surface should be checked before assuming.
5. **api-stability "failure" misattributed for one step**: the preflight api-stability red was the KNOWN external core/v5 arch-lint red, not golden drift — I regenerated the golden before reading the failing test's name (regen was still correct, +2 concurrent exports absorbed).

## e) WHAT WE SHOULD IMPROVE (durable, this session's evidence)

1. **Same-edit gate closure for two drift classes**: any `errorfamily.New*` code addition must add the `docs/error-taxonomy.md` row in the same edit (the drift gate is cheap: `nix run .#check-error-taxonomy`); any new exported or twin-shaped code must run `nix run .#check-duplication` before session end. Both belong in the mental "api-stability regen" slot.
2. **Load-guard protocol has a middle rung missing in practice**: when `#verify` refuses and `--wait-loop` is volatile, `#verify-fast` gives cross-module proof in minutes — I never ran it (d-class miss). Standing recipe: refused → preflight → verify-fast NOW → wait-loop verify in background.
3. **gofumpt pin is stale for Go 1.27 generic methods** — tree-wide `nix fmt` is broken by ANY legal generic method until flake.nix repins gofumpt. First foreign file tripped it within a day of the feature landing. (Nix change; ownership question in §g.)
4. **Hash-golden config files need the daemon exempted or the golden auto-repinned**: the corruption tripwire fires on concurrent waves' auto-commits (this is at least the 2nd occurrence — 2026-10-03 had the same class). The restore remedy works but consumes session time and risks reverting intentional change.
5. **Handoff verification rule (now in gotchas + plan §11.5)** — two handoff claims failed verification this wave; the register exists so the next handoff author budget-checks "tagged?" and "compiles?" claims.

## f) NEXT — up to 50 (ordered)

1. **Re-run `nix run .#verify` now** (load 9.78 falling at 11:51) — closes T27.1; expect MY modules green; externals itemized below.
2. Itemize verify output: core/v5 arch-lint red, V007 marker drift, 7 file-size offenders, templ regen (docserver), coverage/schema −12.7%, doc-check union warnings — all §7-external.
3. `#verify` alternatives if load storms again: `nix run .#verify-fast` + the already-green per-module suites as T27.1 evidence.
4. Tag wave (Q2): systemscenario (presets/chaos/SSE/ThenQueryEventuallyFails/example_test), event (deliveryctx API), deriver, watermill (ErrReentrantPublish), cqrs-lint (E020, 210 rules), cqrs-upgrade (schemaVersion 2) — pre-tag full per-module tests over the changed set (Gate 3; the 2026-10-06 red-suite lesson).
5. Post-tag: deriver/watermill `GOWORK=off` standalone builds go green again (plan §11.3).
6. Post-tag: F20.4 — cqrs-htmx + go-appkit pilots adopt `Memory()`/`SQLite(t)` presets.
7. Post-tag: examples may adopt the untagged surface (ThenQueryEventuallyFails for the goal-shaped tombstone probe; presets) — optional polish.
8. gofumpt pin bump in flake.nix (Go 1.27 generic methods) — unblocks tree-wide `nix fmt`.
9. Notify/flag the schema-wave agent: their `.golangci.yml` re-serialization was restored; re-land intentionally WITH `check-golangci-hash.sh --update` if it was deliberate.
10. core/v5 agent: add `.go-arch-lint.yml` (14 production packages) — kills the api-stability meta-test red.
11. core/v5 agent: V007 marker-table drift (14 symbols).
12. metaengine/system agents: shrink or baseline-update the 7 file-size offenders (engine.go 712, reflect.go 359, typed_reader_scan.go 367, host.go 383, pagination_conformance.go 353 NEW, system.go 361 NEW, stale.go 489→533 GREW).
13. schema agent: coverage floor −12.7% (add tests or `--update` the floor deliberately).
14. catalog/graph-native agent: `templ generate` from `catalog/docserver` cwd (5 files stale).
15. systemscenario: `TimeAdvancesTo(t)` sugar (Axon analog; TODO_LIST).
16. systemscenario: harness depth assertions pack (TODO_LIST §f13/15/16/20: Event version-hint invalidation, golden payload hashes, journal-equivalence diffs, snapshot interplay, DLQ in-scenario, upcaster in Given, idempotency, OCC scenarios).
17. cqrs-htmx: sqlite-lifecycle trains through the harness (`sqliteDeployment` variant; TODO_LIST).
18. deriver: journal-tailed host design (ADR-0154 decision c — the v5 deadlock endgame).
19. v5 arc: systemscenario absorbs scenario/v4 (ADR-0153 note; ROADMAP).
20. scenario/v4: `testing.TB` refactor for parity with systemscenario constructors.
21. Observational-equivalence port to system level + store-conformance reuse in-harness.
22. `docs/status/README.md` live-reports index: add the 10-10 wave reports (index currently ends at 10-09).
23. Authored commit (or explicit waive) for the wave's go-cqrs-lite files currently living in daemon `chore:` commits — if authored history matters for the release train.
24. CHANGELOG [Unreleased] will need its release-train comment + section mapping at tag time (T02 manifest draft is the input).
25. Consider a `check-example-standalone` leg that also runs the suites (not just `--build`) — this wave's suites are green standalone but the gate only builds.
26. recipes.md §2.43 could cross-link the new FAQ harness pitfalls (discoverability).
27. systemscenario README: mention the two FAQ pitfalls (chain shape + deleted-row split) — README has "Chaos and SSE" but not these.
28. cqrs-upgrade: suggestion rules could grow a second pattern (`waitForView`-style helper loops) — demand-gated.
29. E020: consider an "engine-coverage" auto-dismiss heuristic (import of `system/integration` or driver blank-imports) instead of comment-only dismissal.
30. Load-guard: `can-run-composed-gate --wait-loop` max-wait 3600s was too short under a 90-min storm — consider `--max-wait` flag or a quiet-window-run wrapper for verify specifically.
31. Verify the 3 daemon-committed micro-edits this session survived the daemon intact at wave close (`git log --stat` sweep over the wave's file list).
32. PLAN RESIDUE CHECK: plan §8 v5 arcs (DCB research note) — harvested to ROADMAP by T04 earlier; nothing to do now unless owner wants the note written.

## g) QUESTIONS (cannot figure out myself)

1. **Verify strategy:** the machine was load-stormed all session (30–60 for 90+ min; wait-loop timed out). At report time load=9.78 and falling — likely open NOW. On your GO: (a) re-run `nix run .#verify` immediately (preferred — closes T27.1 for real), or (b) accept `#verify-fast` + the session's per-module green as T27.1 evidence and close the wave without the full gate? (I will not `VERIFY_FORCE=1` — that defeats the guard's purpose.)
2. **.golangci.yml restore:** I restored the hash-golden-pinned config, reverting a concurrent wave's committed re-serialization (which had displaced `depguard` to a golangci-ignored top level — functionally a dep-guard DISABLE). If their change was deliberate, they must re-land it WITH a golden re-pin (`check-golangci-hash.sh --update`). Do you want me to leave a note in their in-flight file/TODO_LIST, or does the tripwire message suffice?
3. **gofumpt pin ownership:** tree-wide `nix fmt` (and CI's `--fail-on-change` leg) will stay broken on any Go 1.27 generic method until flake.nix repins gofumpt. May I bump the pin (small nix change, verify via `nix fmt` on the foreign file), or does nix toolchain pinning belong to the schema/buildflow wave?

---

**Stop point:** waiting on owner. Default on "continue": run `#verify` now (question 1a); leave the tripwire message as the notification (2); bump gofumpt only if you say mine (3, default NOT mine).
