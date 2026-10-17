# Status: Architecture-Reference Completion Session (A1–A27 plan execution)

**Date:** 2026-10-17 23:15 CEST · **Repo:** `~/projects/go-cqrs-lite` · **Branch:** `master`
**Session window:** ~09:40–11:05 (execution) + 23:15 (this report; the tree moved on underneath via daemon commits and at least two concurrent sessions in between)
**Executing:** `docs/planning/2026-10-17_09-02_SUPERB-architecture-reference-completion-pareto-plan.md` (A1–A27)
**Scope discipline:** only this session's run and what it noticed; other sessions' plans are referenced by pointer, not re-audited.

---

## a) FULLY DONE (verified green, committed)

### Hardening + tests (plan A1-residual, A4, A5)

| Item | Evidence |
|---|---|
| **A1-residual: `writeTable` fold** | All **six** table renderers (`attributeTable`, `parameterTable`, `entityPropertyTable`, `outputTable`, `agentToolTable`, `changeTable`) rewritten onto one `writeTable` helper in the new `table.go` (56 lines). Net ~-190 lines of cloned ladder. `golangci-lint run ./architecture/...` → **0 issues** (after fixing one `exhaustruct_v5` finding it surfaced). Full catalog module suite green. |
| **A4: anchor-integrity tests** | New `anchor_test.go`: `TestExportLinksResolveToSections` (every `href="#x"` AND every search-index `"h":"x"` must hit an `id="x"`), `TestExportNavMatchesSections` (sidebar↔body bijection, both directions + count), `TestExportAnchorSlugsAreUnique` (case-collision guard). **Mutation-verified ×3**: renamed nav anchor → red; dropped overview nav entry → red; renamed a search-index JSON key → red. All restored, tree diff clean. |
| **A5: surface pins** | `TestSearchIndexKeyContractPinned` (decodes the inlined JSON, pins the exact `{a,c,d,g,h,k,s,t}` key set — mutation-verified), `TestExportToWritesSameBytes` (Export vs ExportTo byte parity), `TestLogoRendersWhenAllowed` + `TestLogoDroppedWhenUnsafe` (4 hostile logos incl. `javascript:` case variants — **no trace** in output). |
| **A5: `WithLogo` decision** | Was dead API (stored, never rendered). Now renders `<img class="brand-logo">` in the top bar behind the new `safeImageSrc` allow-list (http/https/data:image only; anything else dropped entirely, not rewritten). CSS rule added. Decision recorded in `catalog/AGENTS.md`. |

### Gates repaired to green (A6 — all receipts from this session)

| Gate | Outcome |
|---|---|
| `check-error-taxonomy` | **GREEN** (538 codes, 19 modules). Two repairs were needed: (1) the gate still walked the deleted `transport/grpc` — removed the row + the doc's `### transport/grpc` section; (2) that unmasked 4 undocumented live mints in `storage/turso/sync.go` (`storage.turso_memory_sync` Rejection, `storage.turso_pull`/`turso_stats`/`open_turso_sync` Infrastructure) — documented with correct families. |
| `check-lint-config` | **GREEN**. Two first-party deps (`go-etag/server`, `httputil`) were missing from the depguard allow-list since yesterday's taskmanager HTTP-binding dogfood — allow-listed with a dated comment + `check-golangci-hash.sh --update` re-pin. |
| `nix fmt` (treefmt) | **GREEN** — 0 files changed. |
| `check-eventcatalog` | **GREEN** — Node build + linter clean (17 files). |
| `check-md-go` | **GREEN** (1506 blocks valid, 105 baselined) after re-pinning the md-go-validator `vendorHash` (its flake input moved upstream — the documented 09-29/10-04/10-11/10-13 recurrence class). |
| `doc-check` (full corpus) | **GREEN** — "All 2742 references valid across 86 package(s)". Was RED with 4 counted warnings from the delete-session residue (DOMAIN_LANGUAGE.md still imported `transport/http|grpc`) + 1 ambiguity my package introduced (`architecture` alias vs cqrs-lint's rules package — fixed by importing the package in the `catalog/README.md` fence). |
| `check-readme-links` | **GREEN** — 0 broken (was 6 dangling from the delete session's `transport/*` + `idempotency/kvstore` removals). Six bullets converted to "REMOVED 2026-10-17 (zero-consumer delete)" notes; `check-readme-deprecated` clean. |
| `check-adr-numbering` | **GREEN** before AND after ADR-0160 (157 → 158 ADRs, index lockstep). |

### New capability (A8, A9)

| Item | Evidence |
|---|---|
| **A8: repo-level artifact gate** | `scripts/check-architecture-reference.sh` + `catalog/cmd/architecture-fixture/main.go` (135 lines): renders a fixture catalog through the real CLI path and asserts self-containment, anchor integrity (hrefs AND search hits), 13 expected sections, ≥13 sidebar entries. `--self-test` fault-injects 3 corruptions (renamed id, injected external stylesheet, deleted section) — **all caught, mutation-verified by construction**. Wired as flake app `check-architecture-reference`, a `#verify` + `#verify-fast` leg, and a CI step. `nix flake check --no-build` passes. Gate row added to root `AGENTS.md`. Note: the first count assertion caught a real bug in itself (`grep -c` counts LINES, the doc is minified) — fixed to occurrence counts before first green. |
| **A9: ADR-0160** | `docs/adr/0160-architecture-reference-selections-kind-id-selectors.md`: selectors are `(kind, id)` pairs; custom-property (`x-*`) selectors **rejected** until the model has a typed custom-property concept (BaseConfig has none; `any`-typed map would violate the strong-types contract); exclude-wins, neverExport beats everything, containers bring their natural closure. Index row added, gate green. |

### Errata / doc-truth (A7)

- `TODO_LIST.md` errata block: fixed "Erratta" typo, replaced two stale line-number pointers (`:1452`/`:1491` → row-name references that survive shifts).
- The two target rows themselves annotated with dated STATEs: the triage row (item (1) generalizes to ~all modules; items (1)+(3) merge with the 08-02 plan's task 45; new item (7) = catalog arch-lint 40+14) and the `core/v5` arch-lint row (same disease reached catalog).
- `docs/status/2026-10-17_08-23_testfixtures…md:98` annotated RESOLVED (the "untracked parallel files" landed with ADR-0159); `docs/status/2026-10-17_08-56_delete-at-zero…md:49` annotated IDENTIFIED (the foreign `catalog/AGENTS.md` change was mine).
- The 6 dangling README links **fixed in place** (standing owner permission for trivial doc-truth fixes), receipt added to the errata row.

### A10 — selection kept-set (code complete, suite green — but see b)

`selection.go` (293 lines: `Selector`, `Selection`, kept-set computation with closure + bans, `selectorResolver`), `WithInclude`/`WithExclude`/`WithNeverExport` options, `docBuilder.kept` + `refIndex.kept` gates (registration gate, owners/anyOf/messagesOf/link drop-not-degrade under selection, `domainOfService` meta suppression), counts derived from the kept map. `GOWORK=off go test ./architecture/` → **ok** (all existing tests pass = the no-selection path is behaviour-identical). Committed by the daemon during the afternoon waves.

---

## b) PARTIALLY DONE

1. **A3 — the composed proof (`#verify` + `#verify-ci`) has NO receipt yet.** `scripts/verify-chain.sh` was armed 09:54 and correctly storm-rided (24 gate refusals: load above ceiling 10, tree-changed windows) but its log ends at attempt 24 — no `#verify` ever ran. The cheap legs A6 covers are green individually (see a), but the repo's definition of done is the composed proof.
2. **A10 — selection: written and green, but unpinned.** The three pinning tests the plan demands (`TestSelectionExcludeWins`, `TestSelectionDomainBringsMembers`, neverExport-beats-include) are NOT written. Until they are, the ADR-0160 semantics are prose, not contract.
3. **API-surface bookkeeping for A10 is outstanding.** `Selector`, `Selection`, `WithInclude/WithExclude/WithNeverExport` are exported → per repo rule, `cmd/api-stability --update` golden regen, CHANGELOG `[Unreleased]` rows, recipe/AGENTS contract updates are all still owed.
4. **A11–A16 (references modes, hide, neverExport reporting, --check, leak test, frontmatter) — not started** (A13's neverExport *option* shipped inside A10's kept-set, but its reporting/CLI/leak-test halves are A14/A15).
5. **A17–A27 — not started** (docserver route, examples, benchmark, docs rows, lint-lesson rows in `gotchas-tooling-build.md`, demo/demo-artifact, recipe caching, `simple.ExportArchitecture`, downstream decisions, owner-decision authoring).
6. **Coverage drift observed, deliberately NOT re-baselined.** `check-coverage` reports 6 modules drifted beyond tolerance (command −2.2, dispatcher −4.6, id −3.4, kv −3.8, metaengine −3.9, schema −3.0). The script's own header warns load1>60 depresses measurements 3-5 points and to re-baseline only from quiet-window numbers — the storm was live, so I left it red rather than corrupt the baseline downward.
7. **`check-file-size` still RED (pre-existing, unchanged by me):** `cmd/doc-check/recipes_catalog_meta3.go` 398 NEW, `catalog/types_resources.go` 359 NEW, `catalog/eventcatalog/exporter.go` 382→392 ratchet GROWTH. None of my new files offend (largest: `selection.go` 293).
8. **`check-duplication` still RED (foreign-owned):** the ~190 novel groups are the fork campaign's `v4 ↔ core/v5` mirror twins. My package contributes **zero** groups (verified at 09:20; `selection.go` + `writeTable` landed after — **re-run needed**, listed in f).
9. **Tree hygiene at report time:** `catalog/go.mod`/`go.sum` modified by my mechanical `go mod tidy` (the 95-file go-error-family v0.11→v0.12 wave left the module one step behind; tidy resolved to v0.12.0). Plus `go.work.sum` from another session.

---

## c) NOT STARTED (plan items, in order)

A11 references modes · A12 hide · A14 --check report + exit code · A15 leak test · A16 frontmatter loading · A17 docserver `/docs/architecture` · A18 example + `examplePaths` · A19 embed-and-serve test · A20 large-catalog benchmark · A21 root AGENTS/module-map/SKILL rows · A22 lint-lesson gotcha rows · A23 demo-artifact decision · A24 recipe-gate caching · A25 `simple.ExportArchitecture` · A26 downstream decisions (eventcatalog-hub, cqrs-htmx/go-appkit) · A27 owner-decision authoring for the pre-existing reds.

---

## d) TOTALLY FUCKED UP (honest list)

1. **A stray 8.4 MB binary got committed.** `go build ./cmd/architecture-fixture/` left `catalog/architecture-fixture` in the tree; the daemon absorbed it within minutes. Caught in the same session, `trash`ed + `git rm --cached` + committed (`0aab92f33`). Lesson: build cmds with `-o /tmp/…` in this repo — the daemon's cadence is faster than your cleanup.
2. **The verify-chain armed at 09:54 produced nothing.** I armed it and then kept editing/committing for an hour, which kept resetting its 60-second tree-stability window; then the session went idle and the chain died with the shell. Two errors: arming it *before* finishing tree-changing work, and not re-checking/re-arming it before going idle. A3 is the plan's 51% and it is still unproven.
3. **Coverage re-baseline temptation under storm — caught, but almost.** I had `--update` halfway typed when the script's own load-warning registered. Re-baselining 6 modules from load-depressed numbers would have enshrined noise as the new floor. The gate warning existed for exactly this; I nearly repeated the "measure, then write" failure the delete-at-zero report just documented.
4. **12-hour session gap, no handoff state.** The execution stopped at ~11:05 mid-A10 (code written, tests unpinned, API bookkeeping owed) without a status note; the daemon + two other sessions then shipped ~120 changed files over the afternoon, including an ecosystem version wave that broke the catalog module build until my `go mod tidy`. Anyone resuming blind would have found a half-wired selection feature with no note. This report is the correction.
5. **`grep -c` vs occurrences in the new gate** — first version under-counted a minified document (4 instead of 13 sections) and failed confusingly. Caught before first green, but the initial assertion set was written against line-mental-model, not byte-mental-model.

---

## e) WHAT WE SHOULD IMPROVE

- **Arm verify-chains LAST, not first.** The correct order is: finish all tree-changing work → commit → go quiet → arm. The chain's stability window and the daemon's cadence make "arm early, work meanwhile" self-defeating.
- **Build binaries to /tmp by default.** One `go build ./cmd/x/` slip becomes a committed artifact in ≤2 daemon minutes. A `.gitignore` entry for `catalog/*-fixture` (or a root `bin/` convention) would make this class impossible.
- **Gate scripts should be occurrence-based from line one.** The repo renders minified HTML; `grep -c` is the wrong tool and it failed in a way (wrong count, right anchors) that looked like a renderer bug rather than a test bug.
- **Version waves should leave per-module tidy receipts.** The 95-file wave tidied most modules but left `catalog` red; a wave-closing `TestEveryModuleGoSumIsTidy` run (which exists!) run *by the wave session* would have caught it before two other sessions tripped over it.
- **Selection semantics need their pinning tests before anything else consumes them.** ADR-0160's rules are currently enforced only by code review; A10 is "written", not "done", until the exclude-wins/closure/neverExport tests exist.
- **API-surface bookkeeping should ride the same edit.** I re-learned (again) that the repo's rule is regen-golden-in-the-same-commit; the daemon's absorption makes "same commit" effectively "same edit session".

---

## f) NEXT 50 (ordered; ✦ = must precede dependent work)

**Finish A10 (selection) properly**
1. ✦ Write `TestSelectionExcludeWins` (exclude beats include inside an included domain).
2. ✦ Write `TestSelectionDomainBringsMembers` (domain → services → messages closure).
3. ✦ Write `TestSelectionNeverExportBeatsInclude` (never-exported absent even when explicitly included).
4. ✦ Write `TestNoSelectionByteIdentical` (export with zero options == export before A10 — pin the compat claim).
5. ✦ `cmd/api-stability --update` golden regen for `Selector`/`Selection`/`With{Include,Exclude,NeverExport}`.
6. ✦ CHANGELOG `[Unreleased]` Added rows (honest `pkg.Symbol` citations, `check-changelog-symbols` green).
7. Mutation-verify the three new selection tests (break exclude-wins → red, etc.).
8. Re-run `art-dupl check` — `selection.go`'s resolver switch may have introduced clone shapes; accept-annotate or fold at source.
9. Re-run `golangci-lint run ./architecture/...` + formatters + full catalog suite; commit as an authored phase commit.
10. Update `catalog/AGENTS.md` architecture section: selection contracts (exclude-wins, closure, drop-not-degrade, nil=keep-all).

**A11–A16 (custom exports)**
11. A11: `references` mode enum (`omit` default / `name-only` / `include`) + option + three tests.
12. A12: `hide` for owners/repository/contacts/docs/specifications/examples + `TestHideDefaults` + `TestHideEmptyShowsAll`.
13. A13 residual: neverExport surfaced in the A14 report (the option exists; the visibility story must be reportable).
14. A14: `ExportReport` (Included/Excluded/Dropped/NotFound/Hidden + reasons) + renderer + `-check` CLI flag + exit-1 contract + test.
15. A15: leak test — hostile fixture, excluded resource's name appears nowhere (HTML + search index + nav + meta text); mutation-verified.
16. A16: frontmatter loading (`LoadExport`, `title`+`include` required, body → markdown intro; `x-*` selectors fail validation pointing at ADR-0160) + fixture test.
17. CLI: `-include kind:id` / `-exclude kind:id` repeatable flags on `catalog-export -architecture`.
18. Recipes.md §2.49 "publish a redacted slice" + `recipeCatalog` entry (compile gate).
19. doc-check over the updated corpus; commit per task.

**A17–A20 (serving, examples, honesty)**
20. A17: `docserver` `/docs/architecture` route + catalog-hash cache + opt-in flag + handler test.
21. A18: `catalog/example/architecture` + `flake.nix examplePaths` (NOT testModules) + `check-example-standalone` green.
22. A19: embed-and-serve `example_test.go` (`http.ServeMux` + httptest 200 + self-containment).
23. A20: 1 000-service / 5 000-message benchmark; correct or confirm the README/ADR performance claim with the measured number.

**Verification debt (the 51% still unproven)**
24. ✦ Re-arm `verify-chain.sh` ONLY after items 1–10 land and the tree is quiet; record the dated receipt in the TODO_LIST architecture row.
25. ✦ Then `nix run .#verify-ci` (GOWORK=off per-module) + receipt.
26. Quiet-window `check-coverage --update` (items 24's gate provides the window) + AGENTS.md coverage line.
27. Confirm `check-duplication` group count after A10 lands; keep my-package-contributes-zero true.
28. `nix run .#check-file-size` — decide the three pre-existing offenders' fate via A27's owner decision, don't grow them.
29. Re-run `check-readme-links` + `check-readme-deprecated` after the recipe/AGENTS edits.

**A21–A27 (docs, decisions)**
30. A21: root `AGENTS.md` quick-ref row (done for the gate; still owed for the package), `docs/agents/module-map.md`, `SKILL.md` paragraph + recipe link.
31. A22: `exhaustruct_v5` own-package-literal rule + golines-100-vs-golangci-120 into `docs/agents/gotchas-tooling-build.md` and `catalog/AGENTS.md`.
32. A23: demo artifact decision (commit `architecture-sample.html` + regenerating gate, or recorded "no").
33. A24: profile the recipe gate's 70 s cold start; cache or documented deferral.
34. A25: accept/reject `simple.ExportArchitecture` in writing.
35. A26: eventcatalog-hub per-source file + cqrs-htmx/go-appkit "download architecture" actions — decided, recorded, TODO rows for the consumer repos.
36. A27: author the owner decision for the pre-existing reds (go.sum wave tail, arch-lint 40+14, file-size offenders, coverage re-pin) as dated receipts on the triage row.

**Session-process repairs**
37. Add the "arm-the-chain-last" + "-o /tmp build" lessons to `docs/agents/gotchas-tooling-build.md`.
38. Commit `catalog/go.mod`/`go.sum` tidy as an authored commit referencing the v0.12.0 wave.
39. Sweep for other stray binaries (`git ls-files | xargs file | grep executable`-style) — one slipped through; check for siblings.
40. Re-verify the fixture gate (`check-architecture-reference --self-test`) still green after selection lands; extend the fixture with a redaction case later (A15's fixture can feed it).
41. Decide whether the armed-chain shell (0C9) needs killing/re-arming after the 12-hour gap (its log is stale; assume dead).
42. Re-check `go.work.sum` drift owner before the next commit wave.

**Smaller polish noticed this session**
43. `writeRef` renders an empty `<span class="rl rl-plain"></span>` for zero-refs — consider skipping the element entirely when `name == ""`.
44. `safeColor`/`safeHref`/`safeImageSrc` are three allow-listers with three shapes — one `scheme allow-list` helper would shrink the surface (accept-annotate if folding is wrong).
45. The errata row in TODO_LIST is growing into a mini-report — split per-row STATEs onto the owning rows only (started, finish the pattern).
46. `selection.go`'s `selectorResolver.exists` switch will grow with every new kind — consider table-driving it off `kindTable` so a new kind cannot forget registration.
47. The `--check` report (A14) and the gate script should share the fixture; keep `cmd/architecture-fixture` the single fixture source.
48. `check-lint-config`'s allow-list comment should name the owning module (taskmanager) not just the date — future readers will ask "who uses this?".
49. Consider `-check` exit-code documentation in the CLI's `-h` text once A14 lands (contract discoverability).
50. When A20's benchmark lands, add its number to the ADR-0159 consequences section (measured, not claimed).

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Push?** This session authored 8 commits (f45c1caa3 → 12abca9ce window plus the doc commits) and none are pushed (`origin/master` was `99ab60bd1` at session start). Push now, or after the selection phase (items 1–10) and the composed-verify receipt land? (The last explicit push instruction was for the earlier session's commits.)
2. **Composed-proof coordination:** should I re-arm `verify-chain.sh` and let it own the exclusive window to completion (it can wait up to 4 h for a quiet window on this shared host), or do you have/expect another session that should take the `#verify` slot first? I cannot see other sessions' intentions, and a double-armed chain would just fight over the verify lock.
3. **A27 pre-existing reds — your ruling:** for the go-error-family v0.12 wave tail I just tidied (`catalog/go.mod`), the 6-module coverage drift, catalog's arch-lint 40+14, and the three file-size offenders — fix-in-place by me in the quiet window after verify, or distribute to the owning sessions (fork campaign / ecosystem wave) and I only record the receipts? My default is fix-in-place for catalog-local items and record-only for fleet-wide ones.

---

*Prepared per the standing status-report contract; sections (a)–(g). No unrelated research performed — everything above is this session's own run plus what its gates surfaced.*
