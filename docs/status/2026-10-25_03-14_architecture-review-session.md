# Status report: architecture review + visualization session

**Timestamp:** 2026-10-25 03:14 CET (session ran 2026-10-24 16:48 → 2026-10-25 03:14)
**Scope:** one session's work only (architecture review #8, D2 diagrams, TODO harvest,
opportunistic doc fixes) plus what that work surfaced about the repo.
**Measurement base:** clean tree at `ed984c6d1` (2026-10-24 16:48). The tree moved during the
session (auto-commit daemon + at least one parallel session editing `scheduling/sqlstore`).
**Format note:** the `status-report` skill declares HTML canonical and asks for
`docs/status/<ts>_<name>.html`; the operator explicitly requested `<ts>.md`, so this is an
intentional, flagged format override. Do not propagate the override into the skill.

---

## 0. Direct answers to your three questions

### 0.1 What did I forget?

1. **Interface count in the stat cards.** The architecture-review skill explicitly asks for
   stat cards covering "module/package count, coupling score, interface count". I shipped
   module count, packages, cycles, edges, symbols, LOC, test ratio, deprecations. The
   225-interface figure only appears in prose.
2. **A CHANGELOG / FEATURES row.** `AGENTS.md` ("Close a Wave / Feature Session") asks for a
   FEATURES row plus a CHANGELOG `[Unreleased]` row for anything user-visible. A new review
   artifact plus 10 harvested TODO rows is at least a docs-visible change; I wrote neither.
3. **Indexing the artifact.** `docs/README.md` links the `architecture-understanding/`
   directory, but there is no README or index inside it, and I added no entry. The report is
   discoverable only by `ls`.
4. **Independent verification of two inherited claims.** "core/v5 is 9 of 32 packages" and
   "batch-release's `verify=ok` builds but does not test" both came from `TODO_LIST.md` /
   `AGENTS.md`, not from my own count. Presumably true; unverified in this session.
5. **The other repo gates.** I ran `check-module-layers`, `check-file-size`, `check-doc-links`,
   `doc-check`. I did not run `check-md-go`, `check-arch`, `check-duplication`,
   `check-coverage`, or `nix run .#verify`. All were scoped out (concurrent session, heavy),
   but "unrun" is not "green".
6. **Browser-level rendering of the report.** I validated structure (parser, anchors, no
   external refs) and diffed the CSS against the template, but I never looked at the page in
   a browser. Headless Chrome was not used because `chrome` is a banned command here.
7. **PNG previews of the diagrams.** `d2` can emit PNG; I generated them only in `/tmp` for my
   own sanity check and deleted nothing, but never committed them. SVG is what the skill asks
   for, so this is a convenience gap, not a spec miss.
8. **A reproduction script.** Every metric in the review was computed with ad-hoc Python
   heredocs in `/tmp`. Nothing is committed, so the next review re-derives everything.
9. **A blast-radius check.** Not applicable (zero library code changed), but I also did not
   confirm that no other session was mid-edit in the files I _did_ touch (two markdown files).

### 0.2 What could I have done better?

1. **Diagram layout, first attempt.** I burned roughly ten render cycles fighting D2 because
   nested containers with cross-container edges get laid out sideways. The pattern that
   actually works (flat markdown text nodes in a chain, containment expressed by labels and
   colour) was the fifth idea, not the first. I should have started with the smallest layout
   that survives dagre, then added detail.
2. **I should have asked about ownership of the red files before writing a P0.** I wrote "split
   the three files" as the top roadmap item, then discovered one of them was growing (363 →
   382 lines) under another session. The finding is correct; the _action_ should have been
   scoped to "owner does this, or I do it with the owner's go-ahead".
3. **One metrics pass, saved.** Instead of six ad-hoc scripts, one `scripts/arch-metrics.py`
   with committed output would have made the whole report auditable and re-runnable.
4. **Provenance tags per claim.** I mixed first-hand measurements (cycles, LOC, API counts)
   with inherited ones (9/32 packages, release-tool behaviour). A `[measured]` / `[inherited]`
   marker per claim would have made the confidence distribution obvious.
5. **Cross-linking.** The 10 TODO rows cite "review findings §N" but not the HTML anchors
   (`#findings`), and the report does not link back to the TODO section.
6. **Verifying the "pre-existing" nature of the dprint drift more precisely.** I checked three
   specific files against `HEAD` and found all three unformatted. I did not run a repo-wide
   `dprint check`, so "at least three files are unformatted at HEAD" is what I can defend.
7. **Report scoring granularity.** I scored integers per dimension, then wrote the average as
   3.9 before recomputing to 4.0. Caught and fixed, but it shows the arithmetic was done last
   instead of verified.

### 0.3 What could still be improved (beyond this session's deliverables)

1. **A package-size gate.** Nothing in the repo fails when one package reaches 829 exported
   symbols. `check-file-size` caps files, `check-arch` caps dependencies; there is no cap on
   symbols-per-package or files-per-package. That is the exact gap the review found.
2. **A production-cycle gate.** Cycles were computed by hand. A 30-line script comparing
   non-test imports would make "0 production cycles" a maintained fact instead of a snapshot.
3. **API-golden tiering.** One 6,301-symbol golden treats a fold inference helper and
   `system.New` as equally promised.
4. **dprint / `nix fmt` truth.** At least three files are unformatted at `HEAD` while
   `nix fmt --fail-on-change` is documented as a CI gate. One of them is `TODO_LIST.md`,
   which is edited every session. Either the gate is not wired as documented, or formatting of
   that file is deliberately excluded and undocumented.
5. **Diagram legibility convention.** The skill's D2 guidance does not warn about the
   nested-container flattening trap, nor about wide output at scale. Future sessions will hit
   the same wall.
6. **Status-report volume.** `docs/status/` holds 123 live `.md` reports before this one;
   `check-doc-links.sh` warns above 10. The archived/ index exists but the live set keeps
   growing.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                    | Evidence                                                                                                                                                                                     |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| a1  | Read the series context before writing: prior review 2026-07-09 (score 7.0, 48 modules, 6 roadmap items) plus the 2026-08-04 metaengine goal-gap report, and reconciled all six prior items in the new report (§01).                                    | `docs/architecture-understanding/archive/2026-07-09_05-18_ARCHITECTURE-REVIEW.html`; report §01 table                                                                                        |
| a2  | Produced the review report (1,990 lines, 12 sections, 7 findings, 10-step roadmap, evidence tables, method/caveats), built by copying the kit template so the CSS is byte-identical.                                                                    | `docs/architecture-understanding/2026-10-24_16-54_architecture-review.html`; `diff` of lines 1-826 against the template shows only `<title>` changed; committed in `2658c066c` (10-24 19:47) |
| a3  | Scored all seven rubric dimensions with cited evidence: Coupling 4, Cohesion 3, Modularity 4, Composability 5, Scalability 3, Service orientation 4, Dependency direction 5, overall 4.0.                                                               | report §scorecard, §02 tables                                                                                                                                                                |
| a4  | Measured the production import graph: 101 modules, 322 production edges, 0 production cycles, 10 cycles only when `_test.go` edges are counted.                                                                                                         | report §02, §05; method table in §10                                                                                                                                                         |
| a5  | Quantified the API surface: 6,301 exported symbols across 82 tracked modules, 2,151 funcs / 1,924 methods / 652 structs / 537 consts / 449 named types / 363 vars / 225 interfaces; median 69 per module.                                               | `docs/api_surface.txt` grouped by owning module                                                                                                                                              |
| a6  | Quantified the cohesion problem precisely: `metaengine` root = 166 production files, 27,701 LOC, 829 symbols (13.2% of API) in 9 filename-clustered concerns (48/28/16/16/13/13/11/8/13 files).                                                         | report finding 1; cluster listing in appendix caveats                                                                                                                                        |
| a7  | Quantified the dual-track cost: 214 `// Deprecated:` markers, `core/v5` = 9 of 32 packages, mirror twins `event` 253 = `core/v5/event` 253 and `id` 94 = `core/v5/id` 94, `cmd/cqrs-upgrade` has zero `core/v5` references (verified by grep, no hits). | report finding 2; `grep -rn "core/v5" cmd/cqrs-upgrade --include=*.go` → empty                                                                                                               |
| a8  | Quantified tooling gravity: `cmd/cqrs-lint` 48,721 production LOC (largest module, 1.4× `metaengine`, 7× `system`), `catalog` 22,246, tooling+catalog = 32.8% of production LOC.                                                                        | report finding 4                                                                                                                                                                             |
| a9  | Rendered the current-state diagram: D2 source + SVG, exit 0.                                                                                                                                                                                            | `2026-10-24_16-54_architecture-review-current.d2` / `.svg` (100,571 B); commit `341fab141` (10-24 19:45)                                                                                     |
| a10 | Rendered the target-state diagram: D2 source + SVG, exit 0.                                                                                                                                                                                             | `2026-10-24_16-54_architecture-review-improved.d2` / `.svg` (65,227 B); commit `2658c066c`                                                                                                   |
| a11 | Harvested 10 roadmap rows into `TODO_LIST.md` under a new indexed section ("Architecture review (2026-10-24 harvest)"), each with effort tags and source pointers.                                                                                      | `TODO_LIST.md:2738+`; section-index row at `TODO_LIST.md:112`; committed `2088b7a5e` (10-24 21:05)                                                                                           |
| a12 | Fixed 3 pre-existing broken doc links so `check-doc-links.sh` is green: `faq.md` (missing one `../` before `docs/adr/0161…`) and two ADR links in `docs/status/2026-10-23_17-20_…-gap-status.md` (wrong depth).                                         | before: "3 broken"; after: "checked 1146 relative link targets across 574 files; 0 broken"                                                                                                   |
| a13 | Ran `cmd/doc-check` on the skill surface after the markdown edits: rc=0, "All 1356 references valid across 57 package(s)".                                                                                                                              | `cmd/doc-check` run at 19:42                                                                                                                                                                 |
| a14 | Re-verified the layer gate cited in the report: `bash scripts/check-module-layers.sh` → "Module layer check passed" (100 layer entries).                                                                                                                | run at 03:15                                                                                                                                                                                 |
| a15 | Verified the review's central red-gate claim twice and recorded the drift honestly (363 → 382 lines within an hour; `system/adapter_event.go` 351 → 352).                                                                                               | `check-file-size.sh` rc=1 at 19:44 and 03:15                                                                                                                                                 |
| a16 | Validated the HTML deliverable mechanically: no unclosed tags, no duplicate ids, no dangling in-page anchors, no external `src`/`href`, no placeholder text, `<title>` set.                                                                             | parser + regex checks; §10 caveats                                                                                                                                                           |
| a17 | Corrected four numeric claims during self-review before shipping: overall average 3.9 → 4.0, presets dead 7-of-9 → 8-of-9, engine drivers "other ten" → "eleven", "5× system" → "7× system", trains "105+" → 112.                                       | report history / final text                                                                                                                                                                  |

---

## b) PARTIALLY DONE

| #  | Item                                  | What works                                                                  | What remains                                                                                                                                                                               | Blocker                                                                                | Effort |
| -- | ------------------------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------- | ------ |
| b1 | Review artifact completeness          | Report, both diagrams, TODO harvest, evidence tables all exist and validate | No CHANGELOG/FEATURES row; not indexed in `docs/README.md` or a directory README; diagrams linked, not embedded                                                                            | none, just not done                                                                    | S      |
| b2 | Roadmap routing                       | 10 rows landed in `TODO_LIST.md` with effort + source                       | Not mirrored into the v5-cut-readiness checklist, which `TODO_LIST` declares the "cut-order authority" — so the P1 ordering is not yet in the document that actually sequences the v5 work | needs a decision on whether review rows re-sequence the checklist                      | S      |
| b3 | Diagram legibility                    | Correct content, real values, exit-0 renders, valid XML                     | Layout is wide (current 5,550 × 3,248; improved 6,420 × 2,168), so text is small when opened zoomed-out; no inline preview in the HTML                                                     | D2/dagre layout behaviour (nested containers flatten when cross-container edges exist) | M      |
| b4 | Metric reproducibility                | Numbers are correct and documented per-claim                                | Only as ad-hoc `/tmp` Python; nothing committed                                                                                                                                            | none                                                                                   | M      |
| b5 | Gate verification breadth             | 4 gates run and reported                                                    | `#verify`, `check-md-go`, `check-arch`, `check-duplication`, `check-coverage`, `nix fmt` not run this session                                                                              | concurrent session + heavy runs                                                        | M      |
| b6 | dprint compliance of touched markdown | My inserted TODO rows are not in dprint's change hunks                      | At least `TODO_LIST.md`, `references/faq.md` and the 2026-10-23 status report are unformatted at `HEAD`; left untouched to avoid a large unrelated diff                                    | policy decision: format now vs exclude                                                 | M      |
| b7 | Inherited-claim verification          | Assumptions are stated in the report's caveats                              | "9 of 32 packages" and "batch-release builds but does not test" are inherited, not re-derived                                                                                              | needs the v5 plan doc + release script read                                            | S      |

---

## c) NOT STARTED

| #  | Item                                                                                                                                    | Why not started                                                                                    | Still wanted?                     |
| -- | --------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | --------------------------------- |
| c1 | Splitting the 3 red file-size offenders (`scheduling/sqlstore/store.go` 382, `system/adapter_event.go` 352, `metaengine/layout.go` 360) | One file is actively being edited by another session; touching it risks clobbering foreign work    | Yes: P0, but owner-scoped         |
| c2 | Any v5 structural work (metaengine 5-package split, memory-engine extraction)                                                           | 100% of my session budget went to assessment; the review's job is to recommend, not to execute P1s | Yes, that is the next program     |
| c3 | `cqrs-upgrade --to-v5` codemod                                                                                                          | Large, and it is a decision-gated deliverable (see question 2)                                     | Yes, critical for fleet migration |
| c4 | API-golden tiering (stable vs experimental)                                                                                             | Depends on the v5 package split defining boundaries                                                | Yes                               |
| c5 | Tooling extraction (`cmd/cqrs-lint`, `catalog` exporters)                                                                               | Strategic, owner-level decision                                                                    | Yes, P2                           |
| c6 | Package-size / symbol-budget gate                                                                                                       | Discovered as a gap while reviewing; no code written                                               | Yes, high value                   |
| c7 | A committed architecture-metrics script                                                                                                 | Same                                                                                               | Yes                               |
| c8 | `docs/DOMAIN_LANGUAGE.md` disambiguation of "engine"                                                                                    | Listed in the harvest; not written                                                                 | Yes, S                            |
| c9 | Directory README / report index for `docs/architecture-understanding/`                                                                  | Same                                                                                               | Yes, S                            |

---

## d) TOTALLY FUCKED UP

Honest list. None of it was caused by this session, but all of it is the state I leave behind.

| #  | What is broken                                                                                                                                                                                                                                                                   | Severity                                                                                                      | Root cause                                                                                                                                                               | Mitigation available                                                                                                             |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| d1 | **The file-size ratchet is red and drifting.** `check-file-size.sh` exits 1 on three "NEW offender" files; one grew from 363 → 382 lines within the hour and another 351 → 352 while the review was being written.                                                               | Blocks `#verify`/CI trust. Does not corrupt data.                                                             | The auto-commit daemon absorbs working-tree changes, so growth commits itself; nothing split the files at the moment they crossed 350.                                   | Split each file (S each) or let the owning session do it; do not re-pin the baseline.                                            |
| d2 | **Formatting truth is broken: at least 3 files are unformatted at `HEAD`** (`TODO_LIST.md`, `.agents/.../references/faq.md`, `docs/status/2026-10-23_17-20_…gap-status.md`) while `nix fmt --fail-on-change` is documented as a CI gate. `TODO_LIST.md` is edited every session. | Trust: a gate that is documented as blocking is not holding.                                                  | Either the gate is not wired the way `AGENTS.md` says, or markdown formatting of those files was silently abandoned; the drift hunks sit in regions nobody re-formatted. | Decide policy (question 3), then run `dprint fmt` once and commit the mechanical diff, or add explicit `excludes` + document it. |
| d3 | **The repo's largest module is a linter**, and nothing prevents that class of drift. `cmd/cqrs-lint` = 48.7k production LOC; tooling + catalog = 32.8% of all production LOC, sharing the SDK's CI and release trains.                                                           | Slow burn: review attention and release coordination are split between a runtime SDK and two developer tools. | Each linter rule was a reasonable local addition; no ownership boundary existed.                                                                                         | Product-line split (P2), or at minimum separate release cadence + CI triggers.                                                   |
| d4 | **One package holds 829 exported symbols.** No gate fails on symbols-per-package or files-per-package, so the god package is invisible to CI.                                                                                                                                    | Change cost + godoc noise + golden churn.                                                                     | File-size and dep budgets exist; package surface has no budget.                                                                                                          | Add a package-surface budget gate; split in v5.                                                                                  |
| d5 | **Dual maintenance is real, not theoretical.** `event` and `core/v5/event` are hand-mirrored (253 symbols each), and the bridge that would let consumers move (`--to-v5`) does not exist (verified: zero `core/v5` references in `cmd/cqrs-upgrade`).                            | Every fix in a mirrored package is a two-copy decision; fleet migration is blocked.                           | ADR-0152 chose dual support; the tooling half of the migration was never built.                                                                                          | Ship the codemod, then delete-at-zero.                                                                                           |
| d6 | **The API-golden headline understates the surface.** It tracks 82 of 101 modules; examples/tools/test-only modules are excluded by design. "6,301 symbols" is the tracked surface, not the repo's true exported surface.                                                         | Reporting risk: a reader can over-trust the number.                                                           | By-design exclusion, but the report's caveat is the only place it appears.                                                                                               | State module coverage next to the figure everywhere it is cited.                                                                 |
| d7 | **Layout fight in D2 was avoidable churn.** I shipped diagrams that are correct but wide, after ~10 iterations, because nested containers with cross-container edges get laid out sideways.                                                                                      | Cosmetic + wasted session time.                                                                               | Skill guidance does not warn about the trap.                                                                                                                             | Add the flat-node pattern to the visualization skill.                                                                            |
| d8 | **`docs/status/` has 123 live reports** (+ this one) while the link gate warns above 10. The archived index exists, so the hygiene loop is documented but not run.                                                                                                               | Navigational debt; every docs-health pass pays it down again.                                                 | No scheduled archival trigger.                                                                                                                                           | Archive + index in a docs-health pass.                                                                                           |

---

## e) WHAT WE SHOULD IMPROVE

| #   | Practice                                                                                                                                                         | Impact                                                                                           | Concrete fix                                                                                                                             |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| e1  | **Structural budgets stop at files and dependencies.** A package can grow to 166 files / 829 symbols unnoticed.                                                  | The single biggest structural risk found.                                                        | Add `check-package-surface.sh`: max exported symbols and files per package, with a shrink-only baseline like `check-file-size`.          |
| e2  | **Cycles are a snapshot, not a gate.** "0 production cycles" cost a hand-written parser to prove.                                                                | Regression risk; the metric is re-derived every review.                                          | Commit a 30-line `scripts/check-cycles.sh` (non-test imports only) and wire it into `#verify`.                                           |
| e3  | **Reviews are not reproducible.** Metrics live in a chat session.                                                                                                | Next reviewer re-does everything; numbers can drift between reports.                             | Commit `scripts/arch-metrics.{py,sh}` emitting JSON (modules, edges, cycles, LOC, API counts, top fan-in/out).                           |
| e4  | **One undifferentiated API promise.**                                                                                                                            | Every internal helper is as "stable" as `system.New`.                                            | Split `docs/api_surface.txt` into stable-core and experimental sections, derived from package paths.                                     |
| e5  | **Formatting policy is undocumented and unenforced.**                                                                                                            | A "blocking" gate that is red on `HEAD` teaches that red is normal.                              | Decide policy, then either format once or add explicit excludes with a comment explaining why.                                           |
| e6  | **Foreign in-flight work has no ownership marker.** I had to guess whether I may split `scheduling/sqlstore/store.go`.                                           | Rework risk; agents either stomp or stall.                                                       | Convention: in-flight edits get a `TODO_LIST`/status note naming the owner and a "do not touch" scope, or work in a documented worktree. |
| e7  | **Review artifacts are not indexed.**                                                                                                                            | Discoverability depends on `ls`; the next review re-reads the wrong files.                       | `docs/architecture-understanding/README.md` with date, focus, scores, and the open roadmap row IDs of each report.                       |
| e8  | **Big diagrams have no legibility convention.**                                                                                                                  | Readers get a 6,000px-wide canvas.                                                               | Skill guidance: flat text nodes for big maps, containers only for small groups; export SVG + PNG; keep the D2 source dependency-free.    |
| e9  | **Cross-skill filename drift.** `architecture-review` wants `YYYY-MM-DD_HH-mm_name.html`; `architecture-visualization` wants `YYYY-MM-DD_HH_MM-SESSION_NAME.d2`. | Pairing files by hand; inconsistent archive.                                                     | One session slug + timestamp shared by the report and both diagram pairs.                                                                |
| e10 | **Status-report volume is unbounded.**                                                                                                                           | 123 live files; link gate warns.                                                                 | docs-health archival pass after each report wave; index in `docs/status/README.md`.                                                      |
| e11 | **Provenance is implicit.**                                                                                                                                      | Mixed measured/inherited claims read with equal confidence.                                      | Tag claims `[measured]` / `[inherited]` in review method tables.                                                                         |
| e12 | **TODO harvest routing stops at the TODO file.**                                                                                                                 | v5 ordering lives in the cut-readiness checklist; rows that never reach it do not sequence work. | When harvesting, add a one-line "affects v5 order: yes/no" field; update the checklist when yes.                                         |

---

## f) Top 50 things to get done next

Ranked by impact. `Category`: Bug / Quality / Cleanup / Doc / Feature / Decision.

| #  | Task                                                                                                                                            | Impact   | Effort | Category |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------- |
| 1  | Split `scheduling/sqlstore/store.go` (382 lines) to ≤350, coordinated with its current owner                                                    | Critical | S      | Bug      |
| 2  | Split `system/adapter_event.go` (352 lines) to ≤350                                                                                             | Critical | S      | Bug      |
| 3  | Split `metaengine/layout.go` (360 lines) to ≤350                                                                                                | Critical | S      | Bug      |
| 4  | Add a staged-aware pre-commit leg that runs `check-file-size.sh` on changed files so growth cannot be auto-committed                            | High     | M      | Quality  |
| 5  | Add `check-package-surface.sh` (max symbols + files per package, shrink-only baseline)                                                          | High     | M      | Quality  |
| 6  | Commit `scripts/check-cycles.sh` (production imports only) and wire it into `#verify`                                                           | High     | M      | Quality  |
| 7  | Commit `scripts/arch-metrics` (JSON: modules, edges, cycles, LOC, API counts, fan-in/out) and regenerate the review tables from it              | High     | M      | Quality  |
| 8  | Decide the formatting policy for markdown (`TODO_LIST.md` et al.): format once or document excludes                                             | High     | S      | Decision |
| 9  | If formatting: run `dprint fmt` repo-wide and commit the mechanical diff                                                                        | High     | M      | Cleanup  |
| 10 | Declare v4 fix-only in a short ADR/one-pager (no new packages, no new exported symbols, no structural splits)                                   | High     | S      | Decision |
| 11 | Write the v5 metaengine split plan: 5 packages, dependency rule, symbol ceilings, file-by-file move table                                       | High     | M      | Feature  |
| 12 | Add a "v5 order impact" line to the harvested TODO rows and update `docs/planning/2026-10-01_v5-cut-readiness-checklist.md` where it applies    | High     | S      | Doc      |
| 13 | Build the `cqrs-upgrade --to-v5` mapping table (v4 import path → `core/v5` package) from the module list                                        | Critical | M      | Feature  |
| 14 | Implement `cqrs-upgrade --to-v5` (import rewrite + pin bump + deleted-API report)                                                               | Critical | L      | Feature  |
| 15 | Add a `core/v5` fixture repo to `cqrs-upgrade` tests (migrate a sample consumer end-to-end)                                                     | High     | M      | Feature  |
| 16 | Verify and record the real `core/v5` package count (9 of 32) from the v5 plan; put the number in one canonical doc                              | Medium   | S      | Doc      |
| 17 | Inventory the 23 missing v5 packages with fleet consumer counts, marking which can be dropped at cut                                            | High     | M      | Quality  |
| 18 | Run the delete-at-zero census for `stack/*` presets (8 of 9 have zero requires) and decide retirement                                           | Medium   | S      | Cleanup  |
| 19 | Decide fate of `storage/view`, `storage/relational`, `listing` before v5 (deprecated / zero-consumer?)                                          | Medium   | S      | Decision |
| 20 | Add `core/v5` awareness to the fleet census tooling so v5 adoption is measurable                                                                | Medium   | M      | Feature  |
| 21 | Split `docs/api_surface.txt` into stable-core and experimental goldens, generated from package paths                                            | High     | M      | Quality  |
| 22 | Update `cmd/api-stability` to emit both sections and fail on cross-section moves                                                                | High     | M      | Feature  |
| 23 | Document the two meanings of "engine" (`metaengine.Engine` vs `scheduling/engine`) in `docs/DOMAIN_LANGUAGE.md`                                 | Medium   | S      | Doc      |
| 24 | Write one "dialect seam" note per engine family (pg, badger, duckdb, mysql, memory) explaining intentional clone groups                         | Medium   | M      | Doc      |
| 25 | Add `//art-dupl:accept` directives to remaining engine-family clone groups on next file touch                                                   | Low      | S      | Cleanup  |
| 26 | Split `cmd/cqrs-lint/pkg/analyzer` (187 symbols) along rule families                                                                            | Medium   | L      | Quality  |
| 27 | Decide tooling product-line boundary: separate repos vs in-repo `tools/` with own cadence (question 2)                                          | High     | S      | Decision |
| 28 | If separate: extract `cmd/cqrs-lint` to its own repo, keep a thin in-repo shell                                                                 | High     | L      | Feature  |
| 29 | If separate: extract `catalog` exporters (architecture/eventcatalog/d2/docserver) from the declaration API                                      | High     | L      | Feature  |
| 30 | Add a `docs/architecture-understanding/README.md` index (date, focus, scores, open row IDs)                                                     | Medium   | S      | Doc      |
| 31 | Link the new review from `docs/README.md` (or the new index)                                                                                    | Low      | S      | Doc      |
| 32 | Add a CHANGELOG `[Unreleased]` docs row for the review + harvest                                                                                | Low      | S      | Doc      |
| 33 | Embed the current-state diagram as an `<img>` (or inline SVG) in the report appendix                                                            | Low      | S      | Doc      |
| 34 | Export PNG companions for both diagrams for chat/PR embedding                                                                                   | Low      | S      | Doc      |
| 35 | Add the flat-node D2 pattern + the nested-container trap to the architecture-visualization skill                                                | Medium   | S      | Doc      |
| 36 | Align review/visualization filename conventions on one shared session slug                                                                      | Low      | S      | Cleanup  |
| 37 | Add a "verify prior-report claims before writing" checklist item to the architecture-review skill                                               | Low      | S      | Doc      |
| 38 | Add provenance markers (`[measured]` / `[inherited]`) to the review method table                                                                | Low      | S      | Doc      |
| 39 | Cross-link TODO rows to the review's finding anchors and vice versa                                                                             | Low      | S      | Doc      |
| 40 | Add an explicit owner/foreign-work marker convention for in-flight files                                                                        | Medium   | S      | Decision |
| 41 | Run `nix run .#check-md-go` and record the result for the docs changed this session                                                             | Medium   | S      | Quality  |
| 42 | Run `nix run .#check-arch` (go-arch-lint layer 2) once and record its state                                                                     | Medium   | M      | Quality  |
| 43 | Run `nix run .#check-duplication` to confirm the 189-group baseline is still what the gate sees                                                 | Medium   | M      | Quality  |
| 44 | Run `nix run .#check-coverage` and record the current floor                                                                                     | Medium   | M      | Quality  |
| 45 | Investigate why `nix fmt --fail-on-change` is documented as blocking while 3 files are unformatted at `HEAD` (verify the CI wiring in `ci.yml`) | High     | S      | Bug      |
| 46 | Archive finished `docs/status` reports (123 live) into `archived/` and update `docs/status/README.md`                                           | Medium   | M      | Cleanup  |
| 47 | Remove the stray 18 MB untracked `cmd/api-stability/api-stability` binary (gitignored but present on disk)                                      | Low      | S      | Cleanup  |
| 48 | Decide whether `stack/metaengine` is a "preset" for census purposes and make the naming explicit                                                | Low      | S      | Doc      |
| 49 | Add production-vs-test edge distinction to `check-module-layers.sh` output (it reads `go.mod`, so test-only requires are indistinguishable)     | Medium   | M      | Quality  |
| 50 | Schedule the next architecture review trigger: after the v5 metaengine split lands, re-run the same metrics script and diff the scores          | Medium   | S      | Doc      |

**HARVEST status:** rows 1-3, 10-12, 21, 23-25, 27, 29-30 already exist as the 10 rows in
`TODO_LIST.md` → "Architecture review (2026-10-24 harvest)". Rows 4-9, 13-20, 22, 26, 28,
31-50 are **new in this report** and are not yet harvested; run docs-health HARVEST if they
should become backlog.

---

## g) Questions I cannot answer myself (max 3)

1. **Who owns the three red file-size files right now, and may I split them?**
   I verified `scheduling/sqlstore/store.go` grew 363 → 382 while I was writing, which means a
   live session is editing it. Attempted alternatives: git log dates, TODO_LIST search for an
   in-flight row, and reading the file's recent changes; none names an owner. Answering this
   unblocks the entire P0 row and stops me from either stalling or stomping.

2. **Should `cmd/cqrs-lint` and the `catalog` exporters stay in this repo as product lines, or
   move to their own repositories?**
   I can prove the size (32.8% of production LOC, linter is the largest module) and I can list
   the coupling (versions.json pins, CI legs, lint module catalog). I cannot decide ownership:
   separate repos mean independent cadence but cross-repo release choreography; staying means
   the SDK keeps paying for two tools' structural care. This choice determines roadmap items
   27-29 and half of the scalability score.

3. **Is markdown formatting of `TODO_LIST.md` deliberately excluded, or is the `nix fmt` gate
   simply not holding?**
   I can prove at least three files are unformatted at `HEAD` and that my inserted region is
   not among dprint's hunks. I cannot tell whether formatting that file was consciously
   abandoned (huge diffs on a session-edited file) or whether the gate regressed. The answer
   decides between a one-time mechanical commit and a documented exclusion.

---

## Appendix: artifacts and evidence inventory

| Artifact                                                                      | Size                   | Commit      | Verified by                                                                                   |
| ----------------------------------------------------------------------------- | ---------------------- | ----------- | --------------------------------------------------------------------------------------------- |
| `docs/architecture-understanding/2026-10-24_16-54_architecture-review.html`   | 80,630 B / 1,990 lines | `2658c066c` | HTML parser (no unclosed tags, no dup ids), anchor check, template CSS diff, no external refs |
| `…-current.d2` / `…-current.svg`                                              | 4,387 B / 100,571 B    | `341fab141` | `d2` exit 0, valid SVG XML (39 `<g>`)                                                         |
| `…-improved.d2` / `…-improved.svg`                                            | 3,047 B / 65,227 B     | `2658c066c` | `d2` exit 0, valid SVG XML (26 `<g>`)                                                         |
| `TODO_LIST.md` harvest section                                                | 10 rows + index row    | `2088b7a5e` | `grep "## Architecture review (2026-10-24 harvest)"` → line 2738                              |
| `references/faq.md`, `docs/status/2026-10-23_17-20_…gap-status.md` link fixes | 3 links                | auto-commit | `check-doc-links.sh`: 3 broken → 0 broken                                                     |

**Non-deliverable observations recorded this session** (no action taken):

- `cmd/api-stability/api-stability` is an 18 MB untracked binary on disk (gitignored at
  `.gitignore:86`), left in place.
- `docs/status/` contains 123 live `.md` reports before this one; the link gate warns above 10.
- `versions.json` lists 112 trains, including retired directories (by design: tag truth).
- `event/v4/eventtest` carries a `v0.4.0`-series tag on a `/v4` path (legacy mistag series,
  already documented in the release notes).

---

## Errata (added 2026-10-25 04:3x, during plan authoring)

Non-destructive corrections found while writing
`docs/planning/2026-10-25_04-25_SUPERB-architecture-review-followup-pareto-plan.md` (Pareto Step 0).
The original text above is left intact; read these as superseding annotations.

1. **d2 (formatting) was overstated.** I claimed `nix fmt --fail-on-change` is a blocking gate
   while files are unformatted. Verified on 2026-10-25: `treefmt.programs` = `gofumpt`,
   `goimports`, `golines`, `nixfmt`; **dprint is not a treefmt program**, and
   `nix fmt -- --fail-on-change` is **GREEN** (3562 files traversed, 0 changed). A repo-wide
   `dprint check` reports 114 drifted markdown/JSON/YAML files, but markdown formatting is
   _ungated_, not broken. The real finding is a policy gap: `dprint.json` exists and CI never
   runs it. (Plan tasks T15/T18 carry the decision.)
2. **The 3 file-size offenders are already owned** by
   `docs/planning/2026-10-24_19-40_…fleet-audit-followup…md` tasks T03–T05 (with T06 offering an
   owner-baseline alternative). The review's P0 row remains correct as a _finding_; the
   _action_ belongs to that plan.
3. **Codemod, delete-at-zero census, cqrs-lint extraction, and the duplication program** are
   owned by `docs/planning/2026-10-24_19-19_…red-gates-to-v5-wave.md` (T13, T14/T16, T19; the
   duplication program closed green at 21:0x). They are routed, not re-planned.
4. **"Section (f) is not yet harvested" stands**, but the 50 items now have a routing table in
   the plan above (`§4`), including 38 EXEC / 12 ROUTE / 4 OWNER assignments.
