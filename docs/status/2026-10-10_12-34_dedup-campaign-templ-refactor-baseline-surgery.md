# Status Report: Duplication Review & Deduplication Campaign (2026-10-10, 12:34 CEST)

> **Scope:** this session only — seeded by two art-dupl runs the user pasted (repo-wide `-t 4`, then repo-wide `-t 3`), plus the self-review demanded at close. Not a project-wide audit.
> **Session shape:** wave 1 = systemscenario clone groups; wave 2 = the 19 groups shown at `-t 3`; closeout = gates, baseline surgery, memory updates. A concurrent session (v5/presets author) was writing to the same tree throughout; the auto-commit daemon absorbed work continuously.

---

## a) FULLY DONE (verified green)

| #  | Item                                                                                                                                                                                                                                                                                              | Evidence                                                                                       |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| 1  | **systemscenario Then-prologue dedup** — extracted `(*WhenPhase).thenScenario(method)`; replaced 18 identical `s := p.sc; Helper; requireAct` sites across then.go, then_query.go, then_error.go, then_commands.go                                                                                | systemscenario/then.go:171ff; tests green                                                      |
| 2  | **Act-prologue dedup** — `(*WhenPhase).beginAct()`; 3 sites (Event/Query/TimeAdvances)                                                                                                                                                                                                            | systemscenario/when.go:60ff                                                                    |
| 3  | **Given-guard dedup** — `(*GivenPhase).givenScenario(where)`; 2 sites; fatal messages preserved byte-identical via `where` param                                                                                                                                                                  | systemscenario/given.go:34ff                                                                   |
| 4  | **save→publish invariant named** — `(*Scenario).journalThenPublish(events)`; 2 sites; also fixed stale comment names (`appendGiven`→`appendEvents`, `publishGiven`→`publishEvents`)                                                                                                               | systemscenario/given.go:55ff                                                                   |
| 5  | **Sorted-map-keys modernization** — `slices.Sorted(maps.Keys(m))` replaced make/append/Sort blocks in systemscenario/equivalence.go and cmd/cqrs-lint/output.go (killed a cross-module clone with stdlib, no abstraction needed)                                                                  | equivalence.go:57; output.go:174                                                               |
| 6  | **Renderer trio accepted** — `//art-dupl:accept intentional: per-type diagnostic renderers…` on describeEvents/formatTypes/describeCommands (generic joiner would take more params than the cloned lines)                                                                                         | capture.go, then.go, then_commands.go                                                          |
| 7  | **templ deep dedup** — `eventCatalogDetailHeader` (breadcrumbs+PageHeader+DefinitionList chrome, 4 detail pages), `catalogTableProps` (9 table call sites), `noscriptFallback` (2 SPA pages); regenerated from canonical cwd; docserver tests + CSP browser test green                            | catalog/docserver/eventcatalogview.templ:190ff, specview.templ:12ff, eventcatalog_rows.go:16ff |
| 8  | **Pre-existing check-templ drift FIXED** — 5 `_templ.go` files (3 untouched by me) were stale on master; regenerated from `catalog/docserver/` (the canonical cwd); gate fully green incl. FileName tripwire                                                                                      | `nix run .#check-templ` ✅                                                                     |
| 9  | **KV-engine doctrine coverage** — extended the existing `cross-module KV engine pattern — separate go.mod` accept banner to 9 uncovered sibling sites (badger/bbolt/pebble map_backends + MapUpdate trio, both stream_log nextStreamSeq, dgraph keyStr pair with its own per-op reason)           | 6 files, engine builds ✅                                                                      |
| 10 | **specview SPA-bootstrap residue → surgical baseline entry** — one shape-hash entry (`1e0a4ac4b5570d91`) inserted into the committed baseline (186→187) instead of a full re-pin; templ has no directive lever (AGENTS.md #14b)                                                                   | `.art-dupl-baseline.json`; `art-dupl check` ✅ 0 new                                           |
| 11 | **Gates verified at close:** `#check-duplication` ✅ (0 new, baseline 187) · `#check-templ` ✅ · `#check-csp` ✅ · doc-check canonical ✅ (1270 refs) · gofmt clean on all touched files · module builds+tests: systemscenario ✅, cqrs-lint main ✅ (17s suite), catalog ✅, 4 engine modules ✅ |                                                                                                |
| 12 | **Memory updated** — AGENTS.md #14 THIRD semantics (surgical single-entry baseline insert vs full re-pin); gotchas-tooling-build.md templ-cwd nuance (module root `catalog/` ALSO drifts; only the package dir is canonical)                                                                      | AGENTS.md:105; docs/agents/gotchas-tooling-build.md:27                                         |
| 13 | **Foreign WIP untouched** — presets.go/presets_test.go, core/v5/, and their go.mod/go.sum changes were never edited or reverted by me; foreign test breakage verified via overlay, not by "fixing" it                                                                                             | —                                                                                              |

**Net effect on the raw scan:** user's `-t 3` run showed **19 groups**; current run shows **14** — all templ groups (3) and all KV-family groups (4+) gone; the remainder is baselined idiomatic residue (see b3/c6).

---

## b) PARTIALLY DONE

1. **Lint proof for MY touched modules.** Tree-wide `nix run .#lint` is red across 30+ modules (incl. `core/v5`, the concurrent session's tree). I attributed it to foreign WIP — correctly for the failure I inspected (gocognit in a test function I never touched) — but I never ran a scoped per-module lint on my 7 touched modules to positively prove they add nothing. Claim made on attribution, not verification.
2. **File-size gate.** Verified by `wc -l` (all touched files ≤ 278 lines) but never ran the authoritative `nix run .#check-file-size`.
3. **Accept-annotation asymmetry.** KV family got directives; the other idiomatic classes (lock rituals ×7 groups, duckdb builder prologues ×2, pg/sqlite aggregate builder pair, storage GracefulClose pair, contracttest prologue) were judged "leave baselined" for churn control. Deliberate, but the raw scan still surfaces them unsuppressed — inconsistent optics until policy is decided (question 1).
4. **Baseline hygiene.** The committed baseline still carries now-dead shape hashes (the pre-refactor specview/eventcatalogview templ shapes). Harmless (nothing matches them) but stale entries accumulate.
5. **specview dedup depth.** noscript extracted (30-line pair → 1-liners), but the two SPA bootstrap IIFEs remain a paired shape (now the single baselined entry). Full kill would require JS interpolation through templ params — rejected on the file's own CSP/auditability stance (question 3).

## c) NOT STARTED (all foreign-owned or policy-gated — deliberately)

1. `presets_test.go` missing `fmt` import (concurrent session's WIP; doesn't compile).
2. api-stability golden regen for the presets' new `Memory`/`SQLite` exports (their move, per repo rule "regen in the same edit").
3. V007 deprecation-table entries for the 14 new `core/v5/` markers (cqrs-lint `pkg/rules/version` test red).
4. `hardening_test.go` gofmt debt (committed-but-unformatted; pre-existing).
5. Tree-wide lint triage (30+ modules red).
6. Directive migration for the 14 remaining baselined groups (policy question).
7. CHANGELOG entry — judged N/A: all changes internal/unexported; no exported API surface touched (api-stability diff listed ONLY the foreign Memory/SQLite pair).

## d) TOTALLY FUCKED UP (made, caught, and fixed — but still: made)

1. **Full baseline re-pin executed before thinking.** I ran `art-dupl baseline .` (186→351 groups, +1418 lines) when the surgical need was ONE new shape. The gate's own source comment warns exactly against pinning in-flight foreign code — I had read that comment hours earlier. The daemon committed the mistake mid-recovery, so `git restore` was a no-op against a moved HEAD; recovery required fishing the pre-pin revision (`git show c2bc55767:…`), rebuilding 186+1 by script, and re-verifying. History now carries an over-permissive baseline commit followed by the correction. Lesson written to AGENTS.md #14(c), but the mistake should never have happened: the right move (single-entry insert) was obvious in hindsight and available in the tool's own data model.
2. **First templ extraction was too shallow.** I extracted the 6-line Breadcrumbs block when the flagged region was 19 lines of page chrome. Predictably, the residual scaffolding re-paired as a NOVEL shape and broke `#check-duplication` — costing a second regen+test cycle. Root cause: I deduplicated the first line of the region instead of reading the whole region's semantic unit (breadcrumbs+header+deflist are ONE concept: the detail-page chrome).
3. **`git stash` raced the auto-commit daemon.** Used stash to test pre-change check-templ state; it never landed/popped cleanly (daemon absorbed the working tree mid-sequence). Required forensic verification (git log --stat, rg for my symbols) that nothing was lost. Nothing was — but I knew the daemon existed (AGENTS.md documents it); `git worktree` (which I used correctly for the SECOND pre-change test) should have been the first tool out of the box.
4. **Non-canonical doc-check invocation.** Ran `go run . ../../AGENTS.md` alone, hit a false "broken §2.11" failure (cross-refs resolve within the full scanned doc set, which AGENTS.md itself documents). Wasted a diagnostic cycle on a self-inflicted artifact.

## e) WHAT WE SHOULD IMPROVE (self-review answers, brutal)

- **What did you forget?** The gate's dirty-tree warning before the re-pin; the canonical doc-check invocation; scoped lint proof; formal file-size gate run. All small, all cheap, all skipped under "the gate I ran was green" momentum.
- **Did you lie?** One soft lie by omission: my closing summary said "tests green" per touched module while tree-wide lint was red INCLUDING modules I touched — the redness was foreign in every case I inspected, but "my modules are lint-clean" was never actually tested. This report corrects the record (see b1).
- **Ghost systems / split brains?** None created. Every extracted helper has live call sites (thenScenario ×18, beginAct ×3, givenScenario ×2, journalThenPublish ×2, catalogTableProps ×9, eventCatalogDetailHeader ×4, noscriptFallback ×2). `eventCatalogBreadcrumbs` briefly existed as a standalone component and was folded into `eventCatalogDetailHeader` in the second pass — no orphan remained.
- **Did we remove something useful?** No. All removals were mechanical duplication; behavior preserved (byte-identical fatal messages, identical rendered HTML, tests green).
- **Scope creep?** One justified hop outside systemscenario (cqrs-lint/output.go, killed the cross-module pair, module tests run). The templ second pass was self-inflicted churn, not scope creep.
- **How to be less stupid:** (1) when a gate has a written warning in its source, re-read it before invoking the nuclear option; (2) deduplicate the REGION's semantic unit, not its first line — if extraction leaves ≥3 shared statements standing, the clone will re-pair; (3) on a daemon-driven tree, `git worktree` for any before/after comparison, never stash; (4) close every session with scoped lint + formal file-size gate on touched modules — cheap, and they close the verification honesty gap.

## f) Next actions (impact-sorted, ~30)

**Verification debt from this session:**

1. Scoped per-module golangci lint on the 7 touched modules (systemscenario, cmd/cqrs-lint, catalog, badgerengine, bboltengine, pebbleengine, dgraphengine).
2. Run `nix run .#check-file-size` formally.
3. Re-run `nix run .#check-duplication` after the daemon settles (idempotence proof).
4. Run `#verify` (composed gate) once the tree is quiet and foreign suites compile.
5. `#verify-ci` per-module matrix leg.

**Foreign-owned red items (coordinate or take over — question 2):**
6. Add `fmt` import to presets_test.go.
7. api-stability golden regen after presets stabilize (`cd cmd/api-stability && GOWORK=off go run . --update`).
8. V007 `deprecatedV5Symbols`/`deprecatedV5Modules` entries for the 14 core/v5 markers.
9. gofmt hardening_test.go.
10. Tree-wide lint triage (30+ modules; classify pre-existing vs v5-WIP).

**Duplication residue (policy-gated — question 1):**
11. Decide: migrate the 14 baselined idiomatic groups to `//art-dupl:accept` directives, or keep baseline-only.
12. duckdb aggregations: in-module builder prologue ×4 — extract a small `aggQuery` state struct (no dep-isolation excuse; same go.mod).
13. pg/sqlite grouped-aggregate builder pair — candidate for metaengine core per contract #27 precedent.
14. storage bbolt/pebble `GracefulClose` pair — consider `record.GracefulClose(ctx, closeFn)` (Tier-0 home; costs an exported API + golden + CHANGELOG).
15. Prune dead hashes from `.art-dupl-baseline.json` (pre-refactor templ shapes).
16. contracttest t.Helper prologue pair — tiny helper or accept.
17. Repo-wide sweep for remaining make/append/Sort idioms replaceable by `slices.Sorted(maps.Keys())`.

**Tooling (upstream/process):**
18. art-dupl feature request: directive support in `.templ` files (would have made specview annotatable).
19. art-dupl feature request: `--insert-entry <hash>` mode so surgical baseline inserts aren't a hand-run script.
20. Guard idea: refuse full `art-dupl baseline` when the tree differs from HEAD beyond the caller's own files.
21. BuildFlow: fix pre-commit templ tool cwd (tracked known bug; still re-corrupts codegen from repo root).
22. Consider `sweep` app scheduling to keep daemon-absorbed commits lint-clean.

**docserver/systemscenario polish:**
23. Watch for the concurrent presets session regenerating the api golden — avoid duplicate regen conflicts.
24. After presets land: run systemscenario full suite WITHOUT the overlay crutch and delete /tmp overlay artifacts.
25. If presets grow more Then-style prologues, point them at `thenScenario`/`beginAct` (convention now exists).

**Bigger rocks visible from this session (not mine to start):**
26. `#lint` findings list suggests real gocognit debt in several modules — worth a dedicated pass.
27. The docserver "kv alias ambiguous (core/v5/kv vs kv)" doc-check warning will keep firing until core/v5 naming settles — track it.
28. Consider a review pass over the daemon's chore-commits touching .art-dupl-baseline.json (it committed my bad pin without any human gate).

## g) Questions I cannot answer myself

1. **Baseline vs directives policy:** should the 14 remaining idiomatic groups (lock rituals, SQL-builder prologues, GracefulClose, contracttest) be migrated to explicit `//art-dupl:accept` directives for self-documentation, or stay baseline-only? I chose baseline for churn control; the KV-family directives now make the report asymmetric.
2. **Foreign WIP boundary:** the concurrent session left four red items (presets fmt import, api-golden Memory/SQLite drift, V007 core/v5 gaps, tree-wide lint). Strictly hands-off, or am I authorized to fix the mechanical ones (fmt import, V007 tables, golden regen) on their behalf?
3. **specview JS stance:** the file's own comment forbids dynamic JS interpolation (CSP posture). Is that stance firm enough to keep the two SPA bootstrap IIFEs as a baselined clone pair forever, or would a `spaBootstrap(…)` templ component with static JS string params be acceptable?

---

_Point-in-time snapshot. Section (f) is TODO_LIST/HARVEST fuel if instructed. WAITING FOR INSTRUCTIONS._
