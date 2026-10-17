# Status Report — Delete-at-Zero First Application + Brutal Self-Review

- **Generated:** 2026-10-17 08:56 CEST
- **Session scope:** THIS session only (~08:30–08:56): owner ruling on the v4→v5 transition mechanism, plan amendment, first delete-at-zero application (transport/http, transport/grpc, idempotency/kvstore), gate cascade, self-review. Format: operator demanded `.md` (standing override class; HTML default intentionally not used).
- **Inputs:** `docs/planning/2026-10-17_08-02_SUPERB-v4-to-v5-fleet-migration-pareto-execution-plan.md` (amended this session), ADR-0152 (+ addendum), who-uses census 08:37, gate runs logged below.
- **Harvest note:** section (f) items map 1:1 onto the plan's task table (already live in TODO_LIST §v5 Unification blockquote + plan doc). The NEW items (gate-closure 1–8) are not yet in TODO_LIST — harvest pending owner instruction.

## Executive summary

The owner rejected the copy-forward feel of the migration plan. After an A/B/C mechanism analysis, the owner ruled: **phase out v4 submodule-by-submodule; delete each module the moment zero consumers remain; full v5 cut at the end; fleet migration window 1–2 weeks.** The ruling was folded into the plan/ADR-0152/TODO_LIST the same morning, and the standing rule was applied immediately: three deprecated modules (~6,900 LOC) were deleted with the full gate de-registration cascade. Verification caught that the previous census's "11 free deletes" was wrong for 9 of 11 (hidden in-repo couplings) — only the transport pair + kvstore were genuinely free. Final tree: all touched gates green; three pre-existing red classes flagged, not owned.

## a) FULLY DONE

1. **Mechanism question resolved.** Presented copy-forward vs move-forward+`release/v4` vs hard-cut tradeoff table; owner answered with the delete-at-zero goal (recorded verbatim in the plan).
2. **Plan amended** (9 edits, in place — annotated, not rewritten): standing delete-at-zero rule; deletes continuous (D-phase shrinks to coupled residue); twin policy (fork = point-in-time snapshot, v4 side fix-only) explicitly labeled *derived, not owner-ruled*; errata rows for the 9-of-11 coupling discovery and the missing bench-port predecessor; task 38 moved to Phase 0, tasks 38.0 (bench-infra port) and 38b (post-batch delete-at-zero sweep) added; census task gains `--zero-consumers` trigger mode; guardrails #11 (coupling check before every delete) and #12 (no feature dev on forked v4 modules); GO-status note (delete-at-zero GO'ed; Phase A sprint still pending explicit GO).
3. **ADR-0152 addendum** (2026-10-17): continuous delete-at-zero, 1–2 week window, snapshot/freeze mechanics, coupling-check lesson, first application recorded.
4. **TODO_LIST §v5 Unification**: ruling blockquote added at the section head pointing at the amended plan.
5. **Census re-verified at 08:37** (time-gap discipline): transport/http, transport/grpc, idempotency/kvstore = 0 fleet requires; stack/sqlite = 3 (the "8th preset"); idempotency core = 7; sqlstore = 4. In-repo `rg` found the hidden couplings the requires-only census missed.
6. **Deleted** `transport/http` (3,625 LOC), `transport/grpc` (2,526), `idempotency/kvstore` (746) via `git rm` (history-preserving; tags still resolve from the module proxy).
7. **De-registration cascade complete:** go.work (3 use-lines), flake.nix (3 testModules entries, the `test-grpc` app, the CI grpc test line), api-stability modules slice (3) + golden regenerated (6,225 exports), check-module-layers (3 LAYER + 3 DEP_BUDGET rows), cqrs-lint module catalog (kvstore hint) + 3 stale test-map entries + F030 reworded to "deleted 2026-10-17" (rule kept — it now coaches pinned-tag consumers).
8. **Integration suite rewired:** kvstore stripped from contract + property tests (memory+sqlstore remain), `kvstoreTTLParams` → `idemTTLParams`, go.mod/go.sum tidied, vet + targeted tests green.
9. **Docs honest post-delete:** recipes.md §2.15 rewritten onto `codec.TranscodeToJSON` + `event` (fence compiles — TestRecipes green; catalog meta entry updated); advanced.md §6.8 → REMOVED with migration pointers; MIGRATION-grpc-to-v5.md banner; FEATURES (8 edits incl. section headers + maturity rows); README (3 rows); module-map (2 rows + kvstore note); AGENTS (module count, tier listing); CHANGELOG `[Unreleased]/Removed` ×2 rows; docs/status/README.md live index backfilled (5 missing rows, date bumped).
10. **Gates green (final state):** integration tests; api-stability meta tests (incl. mirror lockstep); doc-check (1,340 references, 57 packages); TestRecipes; canonical-facts ("all citations match repo-derived truth"); changelog-symbols (15 verified); root workspace build; post-delete who-uses re-confirmation (0 consumers).
11. **Pre-existing reds diagnosed and flagged (not touched):** `example/mesh-demo` dep budget 8/7; cqrs-lint 4 test failures (TestLintExampleTaskmanager + 3 busfixture) = the fixtures-untidy disease (`go 1.27` vs `1.27.2` tidy drift, zero references to deleted modules); taskmanager gopls errors.

## b) PARTIALLY DONE

1. **Plan/mermaid consistency:** task 38 moved to P0 in the table, but the mermaid graph still renders D1 inside the Phase D subgraph (label updated, structure not). Task numbering appended (38.0/38b) rather than renumbered.
2. **Gate coverage:** targeted gates only. NOT run this session: composite `#verify`, `check-md-go` (MIGRATION-grpc fences parse post-banner — expected fine, unverified), `check-versions-manifest`, `check-readme-links`/`-deprecated`, file-size gate, duplication gate, coverage, engine-capabilities, ADR-numbering, eventcatalog, CSP.
3. **cqrs-upgrade tool:** never checked whether its pin-bump/V007 module registry references the deleted trio (drift possible).
4. **api_surface.txt golden:** regenerated + meta-tests green, but the diff itself was never eyeballed to confirm deletions-only (no accidental additions/losses).
5. **go.work.sum:** may carry stale entries for the deleted modules (harmless, uncleaned).
6. **Ruling tension unflagged to owner:** Q1 answer "slowly phase out" vs Q2 selection "1–2 weeks sprint" — interpreted as incremental-mechanism + sprint-paced-migration, but the interpretation was never surfaced for confirmation (see g).

## c) NOT STARTED

Everything downstream in the amended plan:

- **Phase 0:** copy-forward playbook (task 1, F14–F17); census-fleet.sh JSON artifact + `--zero-consumers` mode (task 2); deriver + graph ghost verdicts (task 3).
- **Phase A — core/v5 completion (9→32):** storage core (part 1 incl. view/relational alias strips — the submodules die here), storage core part 2, storage/memory, snapshot, eventtest (M23 close), decider, middleware, metaengine core (×2), projectionadapter, system (×2), scenario→systemscenario absorb, projection+projectionhost, scheduling(+engine,+sqlstore), schema (minus shells), commandlifecycle(+projections), listing+claiming, otel+prometheus, encryption+signing, catalog, mirror-pair registration sweep + lockstep green.
- **Phase B — codemod v2:** rewrite engine; require collapse + go.work/replace handling; hard-refusal + rename table; golden fixtures.
- **Phase C — fleet migration:** cqrs-htmx (stack/v4 removal first), go-appkit, browser-history, crm (+exec-stage go.mod conflict fix), InboxClean, bank-sync batch, residue batches ×2, delete-at-zero sweep after each batch (38b).
- **Phase D/E:** bench-infra port off presets (38.0 — unblocks preset + stack deletion); stack + 8th preset + deprecated-API deletes; gate-pruning cascade; v5 docs finalization; v5.0.0 cut (REAL test runs, never verify=ok-only); post-cut pin wave; retry/flightrecorder re-pins; fleet tidy sweep; taskmanager LSP triage; automation identification (task 48); signature-level lockstep (task 47).
- **Carried-over open questions** (from 2026-10-16 session): automation source (partially = g/Q3), tool import-level mode, — sequencing now answered by the ruling.

## d) TOTALLY FUCKED UP

Nothing catastrophic; all recovered in-session. The honest list:

1. **Intermediate broken-gate commits (worst miss).** `git rm` ran BEFORE the reference rewrites were finished; the auto-commit daemon absorbed 3 mid-flight states (`8e39faad5`, `ba3a14af2`, …) — committed tree snapshots existed where canonical-facts was RED (AGENTS claimed "100 `go.mod` files" — wrong, actual 101; stale "98/98 classified"; status-README missing 5 rows). Root causes: wrong ordering (deletes first instead of rewrites-then-delete) and a guessed count instead of a measured one. Correct ordering for every future delete: rewrite registrations → regen goldens → gate-check → `git rm` last.
2. **False-green near-miss on cqrs-lint.** First run piped output through `rg -v | head` and echoed `$?` of the *pipe* (0), not `go test` (1). Almost reported cqrs-lint green; caught on reflection and rerun (9 pre-existing failures found and correctly diagnosed). Exit-code discipline with pipes is non-negotiable.
3. **Edit-tool guard round trip wasted:** 5 multiedits rejected simultaneously ("must View first" — bash `sed -n` reads don't count). Recovered via exact-match python with per-edit occurrence asserts (which actually improved rigor).
4. **Arithmetic guess:** wrote 103−3=100 into AGENTS; truth was 101 because the old "103" was itself drifted (actual pre-delete count 104). Measure, then write.
5. **advanced.md long-block transcription mismatch** (first edit failed) → recovered with an anchored `re.sub` section replace. For long blocks, anchored replaces should have been the first move.
6. **Question-tool schema fumble** (missing `description` field) — one wasted round trip; also only 2 of 5 allowed questions used while the asking window was open (the twin-policy question should have been asked then — see g/Q2).

## e) WHAT WE SHOULD IMPROVE

Mapped to the self-review questions:

- **What did I forget (Q1):** `check-md-go` after editing a docs/ file with code fences; `check-versions-manifest`; `check-readme-*` after README edits; cqrs-upgrade registry check; eyeballing the golden diff; a start-of-session `git log` for parallel-session writes (the 08:23 testfixtures report and a foreign `catalog/AGENTS.md` modification were noticed late, via gate failures and status surprise — the foreign diff was never identified because the daemon committed it first).
- **What is stupid that we do anyway (Q2):** deleting before deregistering under a live auto-commit daemon; writing derived numbers into gate-checked docs without running the derivation; piping test output through filters that eat exit codes.
- **What could I have done better (Q3):** ask all clarifying questions in one batch (5 allowed, 2 used); surface the "slowly vs sprint" tension immediately; use anchored section replaces for long markdown blocks from the start; run the cheap doc gates as part of the deletion definition-of-done.
- **What can still improve (Q4):** make the delete runbook (plan task 1) absorb today's lessons — ordering, coupling-check `rg`, gate checklist, count-derivation step; consider a canonical-facts-style check for backticked count forms (the gate regex misses them; today's wrong "100" was invisible to the gate until hand-fixed).
- **Did I lie (Q5):** No deliberate lies. One accidental false signal ("CQRS-LINT EXIT: 0") was caught and corrected within the session, in the same session's log. All capability claims in the final report were test-verified.
- **Ghost systems (Q7):** none created; three deprecated ghosts deleted. Pending verdicts: deriver, graph (plan task 3). The 7 presets + storage/view+relational are now *correctly classified* as coupling-held rather than free — the census lie is fixed.
- **Scope creep (Q8):** mild and justified — status-README backfill + AGENTS "98/98" fix were gate repairs on files already touched; mesh-demo was deliberately left alone (budget changes need owner review).
- **Removed something useful (Q9):** No — all three were deprecated, zero fleet consumers, replacements documented (MIGRATION guide, go-sse/watermill, metaengine.DedupStore); F030 retained to coach pinned-tag consumers; the CBOR→JSON recipe was ported onto supported primitives, not lost.
- **Split brains (Q10):** one live risk and one cosmetic: (risk) the twin policy says v4-fix-only while the mirror-lockstep gate enforces sync on the 9 existing pairs — task 25/47 must scope the gate to primitives-only or the policy and gate will fight; (cosmetic) plan mermaid still shows task 38 in Phase D while the table says P0. Also noted: doc-check printed an advisory "alias kv maps to 2 repo packages (core/v5/kv, kv)" — inherent to dual-support, informational.
- **Tests (Q11):** deletion verified by builds + integration/recipes/meta-tests + double who-uses census; NOT run: full per-module test sweep of every module whose registrations changed, and the composite `#verify`. Deletions are the one change class where "it compiled everywhere" is nearly sufficient — nearly.

## f) Up to 50 things to get done next

**Gate-closure from this session (do first, cheap):**
1. Run `nix run .#verify` in an exclusive window (composite proof of the deletion).
2. `nix run .#check-md-go` (MIGRATION-grpc fences post-banner).
3. `bash scripts/check-versions-manifest.sh --check` (deleted-module version rows?).
4. `bash scripts/check-readme-links.sh` + `check-readme-deprecated.sh`.
5. Audit `cmd/cqrs-upgrade` for deleted-module references + run its tests.
6. Eyeball `git diff` of `docs/api_surface.txt` — confirm deletions-only.
7. Prune stale `go.work.sum` entries for the deleted trio.
8. Run `nix run .#check-adr-numbering` (ADR-0152 addendum added).

**Phase 0 (plan tasks 1–3):**
9. Write the copy-forward playbook (F14–F17) — absorb today's ordering/coupling/gate lessons.
10. `scripts/census-fleet.sh` JSON artifact (F18–F20).
11. `--zero-consumers` delete-trigger report mode.
12. deriver ghost verdict (keep-or-kill row).
13. graph ghost verdict (keep-or-kill row).

**Phase A — core/v5 completion (the 23):**
14. storage core fork part 1 (inventory + exclusions incl. view/relational alias strips).
15. storage core fork part 2 (production files, command-store family, import rewrite).
16. storage/memory fork + tests.
17. snapshot fork.
18. eventtest fork → core/v5/event/eventtest (M23 close; prune last register row).
19. decider fork (otter, singleflight).
20. middleware fork (failsafe budget row).
21. metaengine core fork part 1.
22. metaengine core fork part 2.
23. projectionadapter fork.
24. system fork part 1 (constructor, configs).
25. system fork part 2 (health, lifecycle, timers).
26. scenario → systemscenario absorb.
27. projection + projectionhost fork.
28. scheduling + engine + sqlstore fork.
29. schema fork (minus ADR-0126 shells).
30. commandlifecycle + projections fork.
31. listing + claiming fork.
32. otel + prometheus fork.
33. encryption + signing fork.
34. catalog fork.
35. Mirror-pair registration sweep + lockstep green + arch-lint/dep budgets — INCLUDING the gate×policy scoping decision (see e/Q10).

**Phase B — codemod:**
36. `cqrs-upgrade --to-v5` rewrite engine.
37. go.mod require collapse + go.work/replace handling.
38. Hard-refusal mode + rename table.
39. Golden fixtures per migration class.

**Phase C — fleet (1–2 week window per ruling):**
40. cqrs-htmx (stack/v4 off-ramp first) + suites.
41. go-appkit (3 submodules).
42. browser-history (DedupStore migration).
43. crm + crm-exec-stage go.mod conflict fix.
44. InboxClean.
45. bank-sync + github-local-sync + SEC + KeyCountdown + SwettySwipperWeb batch.
46. Residue batches ×2 + delete-at-zero sweep after each (38b).

**Phase D/E:**
47. Bench-infra port off presets (38.0) → then delete 7 presets + stack + deprecated APIs.
48. Gate-pruning cascade → v5 docs → v5.0.0 cut (REAL tests) → fleet pin wave.
49. Owner-triage queue: mesh-demo 8/7 budget; ~34-module untidy disease (fixes the 4 cqrs-lint reds); taskmanager LSP errors; parallel-session `catalog/AGENTS.md` change identification.
    > **IDENTIFIED 2026-10-17 (owner session):** the foreign `catalog/AGENTS.md`
    > change was the architecture-reference section (`catalog/AGENTS.md:176`,
    > shipped with ADR-0159). Yours to keep as-is.
50. Adopt an authored-commit convention at phase boundaries (this session's work lives only in `chore: auto-commit` commits — acceptable per harness rule, but thin history for a 6.9K-LOC deletion).

## g) Questions I can NOT figure out myself

1. **Pacing contradiction in the ruling:** you said *"slowly phase out/migrate deprecated submodule by submodule"* (mechanism) and selected *"1–2 weeks, sprint it"* (window). Which governs when they conflict — e.g., may I hold a migrated-but-green fleet repo on frozen v4 tags longer than 2 weeks if a repo is deployment-sensitive (crm/browser-history class), or is the 1–2 weeks a hard wall for the whole fleet?
2. **Twin policy veto/confirm (I derived it, you did not rule it):** fork into core/v5 = point-in-time snapshot; the v4 side goes fix-only; NO new mirror-lockstep registrations for the Big-23 (existing 9 primitive pairs keep sync). Confirm, or do you want full mirror-sync (or full freeze) instead?
3. **The scheduled fleet automation (carried over, still unanswered):** what runs the fleet-wide tidy/pin waves (the 2026-10-16 05:07–06:48 sweep), and where is it configured? I need to know what can race the codemod waves before Phase C starts.

— End of report. Waiting for instructions.
