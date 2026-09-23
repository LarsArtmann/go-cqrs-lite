# Status Report: t4 Clone-Elis (in-flight) — Campaign Execution Session

|                   |                                                                                                                                                                                                                              |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Date**          | 2026-09-23, 01:38 CEST                                                                                                                                                                                                       |
| **Session scope** | Execute `docs/planning/2026-09-23_00-03_SUPERB-t4-clone-elimination-campaign.md` (21 actionable art-dupl t4 groups)                                                                                                          |
| **Plan**          | committed authored + pushed as `873eb8ed7` ✓                                                                                                                                                                                 |
| **Progress**      | **13 of 21 groups resolved** (10 extracted into shared code, 3 judged-intentional + correctly annotated). Watermill pair analyzed, mid-flight.                                                                               |
| **Repo state**    | working tree clean; daemon absorbed all code work into `chore:` commits (authored-commit attempts lost the race twice); master pushed                                                                                        |
| **Tests**         | every touched module green at time of its change (core, sqlite, mysql, pg, pebble, badger?, no — badger untouched this session; storage/memory, cqrs-lint pkg). **duckdb's current tree state NOT yet re-tested** (see d.4). |

---

## a) FULLY DONE (verified)

1. **Plan**: Pareto breakdown, 22-task comprehensive + ~57-task micro plan, mermaid graph — written, committed authored, pushed (`873eb8ed7`).
2. **G4+G16 — Vector family (4 engines)**: new `metaengine/vector_insert.go` (`ScanVectorDimensionProbe`, `VectorMetadataArg`) + `ScanVectorResults` in `vector_scan.go`. duckdb/sqlite/mysql/pg rewired; stale duck accept removed; error strings byte-identical; **json/v2 semantics preserved deliberately** (v1 HTML-escapes `<>&` — caught proactively, stored metadata bytes unchanged). Suites: core ✓, sqlite ✓, mysql ✓, pg ✓ (real-DB, 23s), duckdb ✓ (145s, at that tree state).
3. **G1 — PlannedTables trio**: new `metaengine/planned_tables.go` (`ListPlannedTables`); duck/pg/sqlite rewired; lock mode stays caller-owned (RLock/Lock/none). pg+sqlite suites ✓ after clean rerun.
4. **G5+G7 — explain/reset pairs**: judged intentional dialect twins (existing ADR-0136/AGENTS-#19 accepts were misplaced at function level and never suppressed); directives relocated onto the detected regions in all 4 files.
5. **G2 — filter-clause trio**: new `metaengine/planned_filter.go` (`AppendPlannedFilter` + `QuestionPlaceholders`/`DollarPlaceholders` + `beginFilterClause`); mysql/pg/sqlite bodies → one-line delegations (zero caller churn); pg's pinned `IN ($1, $2)` test passes; sqlite/mysql/pg suites ✓.
6. **G3+G8 — storage/memory lock trio**: judged the AGENTS §14-documented idiom; 6 directives placed on the write/read bodies ×3 files. storage/memory ✓.
7. **G18+G21 — pebble layout_planner**: extracted `applyIndexEntries` walker (kind-param preserves exact "write index entry"/"write sort index entry" strings); write/delete became op-injecting wrappers. pebble ✓.
8. **G19 — fold_classify**: `extractorFold` interface + `ensureKeyExtractor` + `keyExtractorPtr()` methods on update/remove folds; switch collapsed to a type assertion loop. core ✓.
9. **G20 — execute.go**: `filterInputValue` helper; both filter paths rewired. core ✓.
10. **G9 — sqlite graph BFS**: extracted `graphBFS(ctx, node, depth, label, expand)`; directed/undirected iterative functions → one-liners with expand closures (undirected wraps outgoing+incoming; final error strings identical). sqlite ✓ (6.5s).
11. **G17 — cqrs-lint helpers**: canonical `lintutil.FileImportsSubstr`; architecture delegates via `fileImportsPath`; testrules' copy deleted, 2 callers rewired. cqrs-lint pkg tests ✓.

## b) PARTIALLY DONE

1. **F11 watermill (G11+G14)** — analysis complete, edits NOT started:
   - G11 (`MessageToCommand`/`MessageToEvent` dual-read window): decision = **extract** `streamIDFromMessage(md)` helper (same package, byte-identical error `watermill.parse_stream_id_failed`); both regions re-read, exact texts captured.
   - G14 (bus Ack/continue loops): decision = **accept** — the two bodies carry deliberately different load-bearing safety comments (Ack-vs-Nack deadlock rationale); merging would orphan them. Directive placement pending (above each `if decodeErr != nil {`, command_bus_internals.go:66 / event_bus_internals.go:87).
2. **Verification debt**: api golden NOT regenerated since F1/F2/F5/F9 (7 new metaengine exports: `ScanVectorDimensionProbe`, `VectorMetadataArg`, `ScanVectorResults`, `ListPlannedTables`, `AppendPlannedFilter`, `QuestionPlaceholders`, `DollarPlaceholders`) — golden is stale right now; `TestEvery`/doc-check pending with it.

## c) NOT STARTED

1. F12: scaffolding annotations G6 (queue/conformance), G10 (testutil containers), G12 (cattest/cqrs-upgrade), G13 (commandtest/eventtest).
2. F13: G15 judgment (adttest vs pgengine tx-isolation — promote or accept).
3. F14: family sweep — turso/badger/bbolt/pebble/dgraph/memory engines vs the new helpers.
4. F15: golden regen + `TestEvery` + doc-check.
5. F16: CHANGELOG `[Unreleased]` entries + `check-changelog-symbols`.
6. F17: SKILL.md references (recipes/modules/faq) for the new helper canon.
7. F18: AGENTS.md internal-contract entry ("engine reads delegate to metaengine helpers").
8. F19: **owner decision** — baseline re-pin policy (69 pre-existing t3 groups + campaign residue).
9. F20: gate hardening (art-dupl nix-provisioned; remove silent SKIP).
10. F21: final verify — t4+t3+t7 scans, full test matrix, `#verify` (blocked on parallel eventcatalog session anyway).
11. F22: TODO_LIST harvest.

## d) TOTALLY FUCKED UP

1. **Mid-edit/test race (self-inflicted)**: started mysql/pg suites, then edited planned_parity files WHILE they compiled → two FAILs from broken intermediate builds. Diagnosed correctly, clean reruns green — but it was my sequencing error and cost a confusing detour. Lesson applied afterward (no tests while editing).
2. **Authored-commit race lost twice**: attempted authored commits for F1/F2 immediately after tests; the daemon had already absorbed the files both times (my `git add` → "nothing to commit"). All campaign work lives in 5+ `chore:` commits; attribution lost. The plan file got an authored commit only because the tree was otherwise clean.
3. **F2 import-cleanup loop**: removed `sort` from 3 files only after the compiler named them — should have predicted unused imports at edit time (same for `strings` in testrules, `database/sql`/`errors` in the vector files, and an unused `database/sql` in my own new planned_tables.go first draft). Not harmful, just sloppy sequencing; builds caught everything.
4. **duckdb tested at a stale tree state**: its 145s green run covers vector changes but NOT the later planned_parity (F2) edit — duckdb's current code is untested as of this report. Must rerun before claiming the family done.

## e) WHAT WE SHOULD IMPROVE

1. **Commit faster or not at all**: with a ~1-min daemon sweep, authored commits only win if issued the instant tests pass (or pause the daemon during campaigns). Decide policy; the current half-fight wastes time and still loses attribution.
2. **Never run tests while editing other files in the same modules** — compile races produce phantom FAILs.
3. **Predict unused imports at extraction time** (build the new import list mentally per rewired file) — the compiler is a slow lint for this.
4. **Regen the api golden WITH each family**, not deferred — right now the golden lies about the surface until F15 runs.
5. The misplaced-annotation pattern (function-level `//art-dupl:accept` vs detected inner region) burned 4 groups (G3/G8, G5, G7 + yesterday's pair): worth a line in AGENTS.md §14 — the directive belongs ON the region's first line, and re-runs after ANY code shift must re-verify suppression.

## f) NEXT (ordered, ≤50)

1. Finish G11: write `streamIDFromMessage` in watermill/protocol.go; rewire MessageToCommand + MessageToEvent; watermill module tests.
2. G14: place two `//art-dupl:accept` directives (rationale: deliberate asymmetric twins, load-bearing comments).
3. duckdb full suite rerun (covers F2 planned_parity change).
4. F12: 4 scaffolding annotations (G6, G10, G12, G13) — directive on region's first line each.
5. F13/G15: read both tx-isolation files; promote pg copy onto adttest if signature-compatible, else accept-annotate.
6. t4 re-scan: confirm 0 shown / iterate any unmasked groups (annotation is iterative — expect 1-2 rounds).
7. F14 sweep: rg each new helper's function family (VectorSearch/VectorInsert/PlannedTables/appendPlannedFilter) across turso/badger/bbolt/pebble/dgraph/memory/iroh; rewire or accept with reason.
8. api golden regen + `TestEvery` + doc-check.
9. CHANGELOG `[Unreleased]`: 7 new metaengine exports (+ yesterday's 4) + the sqlite IN-separator note.
10. Run `check-changelog-symbols` gate.
11. SKILL.md references: document the helper canon (engine-authoring pattern) in recipes.md/modules.md; run doc-check.
12. AGENTS.md §14-adjacent: add annotation-placement gotcha + "engine reads delegate to core helpers" contract.
13. F19 brief for Lars: baseline re-pin numbers (before/after campaign) + options.
14. F20: flake — nix-provisioned art-dupl in check-duplication, remove PATH-SKIP.
15. Full matrix: touched modules' `GOWORK=off go test` (metaengine, 5 engines, storage/memory, cqrs-lint, watermill).
16. t3 gate + t7 scan final numbers; record in TODO_LIST.
17. `nix run .#verify` once parallel eventcatalog session lands (their 370-line file + typecheck errors still pending on master).
18. TODO_LIST harvest from the plan (docs-health).
19. Investigate the 3 high dependabot alerts GitHub reported on push (dependency updates — owner decision).
20. Optional: migrate remaining pre-existing t3 groups next session (69-group triage list from yesterday's report).

## g) QUESTIONS (cannot answer myself)

1. **Baseline policy (blocking CI-green)**: after this campaign, re-pin `.art-dupl-baseline.json` wholesale on a committed tree, or triage the ~69 pre-existing groups first? (Same question as yesterday, now with more residue.)
2. **sqlite EXPLAIN output change is shipped**: IN-list separator unified to `", "` (pg's form; sqlite previously had no space). No tests pinned the old form — confirm you don't depend on the old string anywhere external (dashboards/goldens outside this repo)?
3. **Commit policy under the daemon**: keep racing it for authored commits per family, pause it during campaigns, or accept `chore:` attribution for code and reserve authored commits for docs/plans?

---

_Report generated 2026-09-23 01:38 CEST. Campaign: 13/21 groups resolved, 0 regressions, all touched-module suites green at time of change; duckdb rerun + watermill edits + golden/docs/CHANGELOG are the remaining tail._
