# Status: Architecture-Reference Completion, Waves A10–A27 (plan resumed and executed to its code-tail)

**Date:** 2026-10-23 17:35 CEST · **Repo:** `~/projects/go-cqrs-lite` · **Branch:** `master`
**Session window:** resumed 2026-10-17 ~23:20 from the 23:15 handoff report; execution ran 23:20–00:30 (A10 finish, A11, A12–A16), continued 2026-10-23 14:40–15:15 (A17–A20, A21–A27), then armed the composed proof and has been storm-riding it since 15:13 (this report is written while the chain waits).
**Executing:** `docs/planning/2026-10-17_09-02_SUPERB-architecture-reference-completion-pareto-plan.md` (A1–A27) — every task except the composed-proof receipt is now done.
**Scope discipline:** only this session's run and what it noticed; the census session and the disk-truth session are active on this tree and are referenced by pointer only.

---

## a) FULLY DONE (implemented, verified, committed)

### A10 — selection pins + bookkeeping (the handoff's open item)

| Item | Evidence |
|---|---|
| Four pinning tests | `selection_test.go`: `TestSelectionExcludeWins` (both directions: exclude beats include inside a domain; exclude does not cascade), `TestSelectionDomainBringsMembers` (domain→services→messages closure; everything else absent), `TestSelectionNeverExportBeatsInclude` (beats explicit include AND container closure, siblings survive), `TestNoSelectionByteIdentical` (no options / empty include / empty exclude / empty neverExport / all empty — all byte-identical to the whole-catalog export, all 13 sections present). |
| Mutation-verified ×4 | (1) exclude-ban removed → `TestSelectionExcludeWins` red; (2) domain closure broken → `TestSelectionDomainBringsMembers` red; (3) neverExport-ban removed → `TestSelectionNeverExportBeatsInclude` red; (4) whole-catalog branch of `keptSet` dropped → `TestNoSelectionByteIdentical` red (the naive mutant — dropping `allKeys`' doc block — was UNOBSERVABLE because empty rule lists take the `empty()` shortcut; a realistic observable mutant was used instead). All restored, suite green. |
| API bookkeeping | api-stability golden already carried the selection exports (daemon-era regen had landed them); `--update` re-run produced identical bytes. CHANGELOG `[Unreleased]` Added row written; `check-changelog-symbols` green. |
| Lint debt paid | `exhaustive` switches enumerate every `resourceKind` (registration safety), `exhaustruct_v5`-clean literals, and `writeRef` no longer renders an empty `<span class="rl rl-plain"></span>` for dropped refs (§f43 polish, fixed in the same pass). |

### A11 — references modes (`WithReferences`)

- `references.go`: `ReferencesMode` — `ReferencesOmit` (default, byte-identical) / `ReferencesNameOnly` (out-of-scope refs degrade to display names as plain text) / `ReferencesInclude` (pull ONE level of pointed-at resources in: service→stores+entities+message channels, channel→messages, flow→step services+messages, ADR→appliesTo targets).
- `keptSet` now returns `(kept, banned)`; `refIndex` carries banned-by-key + banned-by-id + full name tables (banned names are NEVER recorded — the recording guard).
- Degrade paths (`owners`, `anyOf`, `messagesOf`, `link`) became three-way: banned → drop in EVERY mode; name-only → plain-text name; no-selection → today's exact behaviour. Drop = SKIP (never append a zero ref) — `refBlock` gates on `len(refs)==0` and block headings carry counts, so appending empties would render "Produced by (1)" with an empty body.
- Tests: `TestReferencesOmitDropsOutOfScope`, `TestReferencesNameOnlyKeepsNames`, `TestReferencesNameOnlyKeepsExcludedHidden`, `TestReferencesIncludePullsOneLevel` (3 subtests incl. exclude-wins-over-the-pull), `TestSelectorSpellingsForEveryKind`.
- **REAL BUG FIXED:** the `"decision"`, `"person"` and `"doc"` selector spellings promised by the ADR-0160 doc comment were silently ignored by `adrKind` — `WithInclude({Kind:"doc"})` kept NOTHING and `WithNeverExport({Kind:"decision"})` redacted NOTHING (a leak risk). Now they resolve; pinned.
- Second real bug found by the tests: `edges()` emitted internal kind strings (`data-store`, `adr`) that `adrKind` rejects — fixed via `selectorSpelling` mapping; include-mode coverage proved it (service include now actually pulls its store).
- Leak test mutation-verified: only disabling the name-recording guard AND both ban checks leaks the excluded name — the test pins the composed invariant (defense in depth), documented as such.

### A12–A16 — hide, check report, leak test, frontmatter, CLI

| Item | Evidence |
|---|---|
| A12 `WithHide` | `hide.go`: `HideOwners/Repository/Examples/Specifications/Docs` (NO `contacts` — no model surface; recorded decision). Owners suppression enforced at the resolver choke point (`idx.owners()` returns nil) so all 20 render sites vanish together; `collectDocs` gates early. `TestHideDefaults`, `TestHideEmptyShowsAll` (byte-equal), `TestHideSections` (5 subtests). The team's own section survives owner-hiding (assertion corrected accordingly). |
| A13/A14 report | `report.go`: `Exporter.Inspect` → `ExportReport{Included, Excluded, NotFound, Dropped, Hidden}` with per-entry reasons (`included`, `in domain X`, `in service X`, `referenced by kind id`, `whole catalog`, `exclude`, `neverExport`, `omit`); `Failed()` = exit-1 contract (a redaction matching nothing is a silent leak risk); stable `String()` renderer. `keptSet` refactored into `Selection.plan()` producing `keptPlan{kept, banned, reasons, notFound}`. Tests: `TestReportReasonsAndExitCode`, `TestReportNotFoundFails` (3 subtests), `TestReportDroppedListsEdgeTargets`. |
| CLI | `catalog-export` grew repeatable `-include/-exclude/-never-export kind:id`, `-references`, `-hide`, `-check` (report-only dry run, writes NO files, exits 1 on `Failed()`), and `-spec <file>`. Smoke-verified end-to-end: `-check` dry run, spec-driven slice with `-never-export` exclusion visible in the report, exit codes 0/1 correct. `-check` reordered BEFORE the MDX write after the first smoke showed it still wrote files. |
| A15 leak test | `TestExcludedResourceNameNeverAppears`: hostile store name (URL-shaped "secret") excluded → absent in omit, name-only AND include modes; presence check without the exclude guards the fixture (`test guards nothing` fatal). Mutation-verified (triple mutation → name-only subtest red). |
| A16 frontmatter | `LoadExport(path)`: `---` fenced YAML (`go-faster/yaml`, already a dep), `title` + ≥1 `include` required (Rejections `catalog.architecture.6/7`), `x-*` selectors rejected pointing at ADR-0160, body → `WithIntroduction` markdown rendered in the overview with `[[kind|id]]` links resolved against the export's own index. `ExportSpec.Options()` maps spec → options. Six rejection cases tested + the render round-trip. |

### A17–A20 — serving, example, proof, measurement

| Item | Evidence |
|---|---|
| A17 docserver route | `Config.ArchitectureReference` (opt-in; off = byte-identical route table), `/docs/architecture` streams the self-contained file, cached per `*catalog.Catalog` instance (`architectureCache`, mutex-guarded; stable provider = render once, rebuilt provider = honest re-render; render ERRORS cached too — pinned by `TestArchitectureCacheRendersErrorPins`). Served WITHOUT the docserver CSP header (the artifact's inline scripts ARE the feature — documented). Tests: route 200 + 404-when-off, cache render-once + invalidation (new catalog instance → new bytes with the added service). Lint 0; docserver suite green. README routes table + prose added. |
| A18 example | `catalog/example/architecture/main.go` — whole catalog + redacted partner slice via `ExportSpec`; both outputs smoke-run (51.6 KB / 49.3 KB). Deviation recorded: NO `flake.nix examplePaths` registration — `catalog/example/` lives INSIDE the catalog module (like `example/rest`), unlike the root `example/` dirs; the plan's registration step does not apply. |
| A19 embed-and-serve | `Example_exportAndServe` (godoc-renderable, Output-verified 200 + content-type) + `TestServedReferenceIsSelfContained` (no `src="/href="http`, `<link`, `@import` through the serve path; Export↔ExportFile byte parity). |
| A20 benchmark | `BenchmarkExportLargeCatalog` (1 000 services / 5 000 messages generated): **~175 ms warm (~615 ms cold first render) / 15.65 MB HTML**. README gained a Scale section; ADR-0159 consequences gained a dated measured-scale addendum (§f50). My first comment guessed "~0.9 s / 3.2 MB" — corrected to the measured numbers before commit. The EventCatalog "~0.1 s to interactive" claim covers browser layout, not render — recorded as complementary. |

### A21–A27 — docs, decisions, receipts

- **A21:** `docs/agents/module-map.md` catalog row extended; `SKILL.md` gained "The architecture reference" beside the catalog-export paragraphs; doc-check green over SKILL+references+AGENTS (1349 refs) AND the full corpus (2751 refs / 86 pkgs, exit 0).
- **A22:** `catalog/AGENTS.md` exhaustruct_v5 own-package-literal rule; `docs/agents/gotchas-tooling-build.md` golines row — with the observed direction CORRECTED vs the plan's guess: `nix fmt` said "0 changed" while golangci flagged the file (treefmt golines defaults vs golangci `max-len: 120`); cure is `golines -w <file>` directly.
- **A23:** demo artifact — decided **NO** with reasoning in `catalog/AGENTS.md` (regenerable in ~1 s; a 50 KB blob rots; the fixture gate already validates correctness).
- **A24:** recipe-gate cold start — **documented deferral** stamped in `catalog/AGENTS.md` (warm 5–15 s; offline-only gate).
- **A25:** `simple.ExportArchitecture` — **REJECTED** with written reason (NewExporter+Export IS the one-liner; option-mirroring wrapper adds no ceremony win; simple is construction, not export plumbing).
- **A26:** downstream decisions recorded on the TODO_LIST: eventcatalog-hub = YES per-source `architecture.html` (add `-architecture` to its export step); cqrs-htmx/go-appkit = SERVE don't export (one config line via the docserver flag).
- **A27:** owner decision for the pre-existing reds stamped on the TODO_LIST (go.sum catalog tail fixed in place; coverage drift deferred to a quiet window; arch-lint 40+14 record-only with the fork campaign; file-size `types_resources.go` owed a split by the next catalog session, foreign offenders left to owners).
- **§f37/§f39/§f41/§f42:** arm-chain-last + build-to-/tmp lessons written to `gotchas-tooling-build.md`; stray-binary sweep clean; dead 09:54 chain superseded by a fresh arm; `go.work.sum` committed by the daemon.

### Handoff leftovers from the 23:15 report — all closed

`catalog/go.mod`/`go.sum` tidy committed (daemon wave); `check-duplication` **GREEN** ("No new clones detected, baseline 189" — my `index.go`/`anyOf` accidental shape-pair got a `//art-dupl:accept` annotation in source; the templ pair was absorbed by the baseline re-pin a concurrent session made); fixture gate `--self-test` re-verified green after all selection work (one corrupted shared-toolchain extraction in `/tmp/gomod-verify` had to be chmod+trashed first — the gate re-downloaded and passed).

---

## b) PARTIALLY DONE

1. **A3 — the composed proof is ARMED but has NO green receipt yet.** `scripts/verify-chain.sh` armed 15:13 (durable log `build/verify-chain-20261023-151354.log`); it is at attempt ~47, refused throughout by (a) the concurrent census session's tree churn resetting the 60 s stability window and (b) host load 26–177 vs ceiling 10. Max-wait expires ~19:13. All CHEAP legs are individually green today: error-taxonomy, lint-config, file-size (after my split), duplication, doc-check ×2, fixture gate + self-test, changelog-symbols, catalog+docserver+architecture suites, `nix fmt`. `#verify-ci` deliberately NOT started (exclusivity: must not overlap the chain's `#verify` leg).
2. **Authored history is poor:** of the four intended phase commits, only ONE survived as authored (`9afa65d02` — and it carries only CHANGELOG+golden; its source files had been daemon-absorbed minutes earlier). A10 pins, A11 references modes, and A17–A20 all landed inside `chore: auto-commit` blobs (`7fe2e0ad2`, `ecc4a725d`, `c7f4852fc`, `020775f9b`, …). All work IS committed and verified — but a reader cannot find it by message. Root cause in d)/e).
3. **`selection.go` at one point reached 636 lines (a NEW file-size offender I created).** Caught by the receipt-writing step ("largest new file ~450" was a guess that a `wc -l` refuted), fixed in-session by splitting into `selection.go` (103) + `plan.go` (148) + `resolver.go` (232) + `edges.go` (165) + `degrade.go` (53); `check-file-size` green ("no new offenders, no growth"). The slip and fix are stamped in the TODO_LIST A27 receipt.
4. **The census session's files (`docs/evidence/fleet-census-*.json` deletions, `scripts/census-fleet.sh`) sit dirty in the tree** — not mine, untouched, and they keep resetting the chain's stability window. The chain is designed to wait them out.

---

## c) NOT STARTED

- Nothing from the plan's A1–A27 remains unstarted. Open AFTER this session: the `#verify` + `#verify-ci` receipts (b1), the `types_resources.go` split (A27's open sub-row), the hub/cqrs-htmx/go-appkit one-liners (A26's consumer-repo rows), and the fleet-wide reds owned by other sessions (fork-campaign mirror directives, task-45 go.sum sweep).

---

## d) TOTALLY FUCKED UP (honest list)

1. **I authored a 636-line file in a repo whose own gate caps files at 350.** Every AGENTS.md session briefing carries the rule; I grew `selection.go` across five features without once running `wc -l`. The catch came from writing "largest new file ~450, under the ratchet" into a TODO receipt — a GUESS presented as a fact, refuted 30 seconds later by `wc -l`. If the receipt hadn't demanded the number, `#verify` would have failed on a NEW offender days later, far from the cause. Lesson stamped: check the gate you know exists BEFORE the receipt forces it.
2. **Every phase commit lost the race to the daemon.** The BuildFlow pre-commit pipeline takes ~2–2.5 min per commit while the auto-commit daemon sweeps every ~2 min; two of my commit attempts ALSO hit `index.lock` contention and one hit a HEAD-move (`cannot lock ref`). Net: authored messages exist for 1 of 4 phases, and one commit (A17–A20) went to a background shell whose final state I never explicitly verified until today's audit. Work survived; history did not. The fix (e1) was learned too late to help this session.
3. **A mutation test lied to me twice before I believed it.** (a) The `go-env.sh` source chain exits non-zero, so `source && go test` SILENTLY SKIPPED the test — a mutant was reported "ok" because nothing ran. (b) The leak property needed a TRIPLE mutation to turn red (recording guard + both ban checks); the single and double mutants stayed green because the defense is layered. Good outcome (layered redaction verified as composed), bad process (I initially reported "mutation 3 verified" from an empty output line).
4. **My benchmark comment shipped a fabricated number for one edit.** "~0.9 s / 3.2 MB" was written before measuring; the run said 175 ms / 15.65 MB (and a cold cache run 615 ms). Corrected in the same session — but the instinct to write the comment before the measurement is exactly the "measure, then write" failure this repo documents.
5. **The first `-check` smoke showed the flag still wrote the EventCatalog tree** — I had wired the dry-run AFTER the MDX export. Caught by my own smoke (`ls work` after `-check`), fixed by reordering before any write.

---

## e) WHAT WE SHOULD IMPROVE

1. **Beat the daemon with staging discipline, not speed:** `git add` + `git commit` immediately after the LAST edit of a phase (the daemon only sweeps what it sees), and for anything bigger, prefer `git commit` WITHOUT the BuildFlow hook on doc-only phases (the repo already fast-paths "Doc-only commit: skipping code gates") — or accept daemon absorption and put the authored narrative in the CHANGELOG (which is where this session's story actually lives).
2. **Never chain `source scripts/go-env.sh && cmd`** — it exits non-zero and silently skips `cmd`. Use `source …; cmd` (semicolon). This cost one false-positive mutation run and should be a lint-worthy shell pattern.
3. **Mutation-verify the MUTANT, not the report:** a "mutation verified" claim requires the mutant's red output to be SEEN (grep for the assertion text), not inferred from exit codes.
4. **Run `wc -l` on every touched file before ending a feature batch** — the file-size gate is known; a one-line shell loop is cheaper than a CI failure days later.
5. **Write benchmark comments AFTER the first run** (or leave them numeric-free until measured).
6. **Shared `/tmp` caches (`gomod-verify`, toolchain extractions) need a repair recipe:** a corrupted `golang.org/toolchain@…` extraction broke every gate until `chmod -R u+w` + trash + re-download. A `scripts/fix-gotmp.sh` one-liner would save the next session 20 minutes of "not in std" confusion.
7. **The chain's stability window vs a concurrent session:** consider a `--ignore-dirty-globs` flag (e.g. ignore `docs/evidence/*`) so one session's evidence churn cannot starve another's verify window; alternatively coordinate verify slots by a tree-writable lockfile both sessions respect.

---

## f) NEXT 50 (ordered; ✦ = must precede dependent work)

**Finish A3 (this session's own tail)**
1. ✦ Wait out the armed chain (`build/verify-chain-20261023-151354.log`, max-wait ~19:13) OR re-arm with a longer window if it expires under storm; the receipt is `#verify` green + the chain's own log path.
2. ✦ Then `nix run .#verify-ci`; record both receipts on the TODO_LIST "composed proof" row with the chain log reference.
3. Quiet-window `check-coverage --update` (6 modules drifted; the gate's load>60 warning blocked it during the storm) + root AGENTS coverage line.
4. Re-run `nix run .#verify` legs list once more AFTER the receipts (doc-only edits since the last full pass: TODO_LIST, gotchas, module-map, SKILL — doc-check already re-run green, but the full chain should see the final tree).
5. Commit the status report + receipts as one authored commit FIRST THING after the chain finishes (before the daemon wakes).

**catalog-local debt (owed by A27's ruling)**
6. Split `catalog/types_resources.go` (359 NEW offender) — CATALOG-LOCAL, next catalog session.
7. Extend `scripts/check-architecture-reference.sh` fixture with a redaction case (A15's fixture can feed it) so the repo-level gate pins a selected export too.
8. Add `-check` exit-code documentation to the CLI's `-h` text (contract discoverability, §f49).
9. `writeRef`'s dropped-ref skip is in — sweep for OTHER empty-element renderers (`refList` count badges vs skipped spans) for the same polish class.
10. Fold `safeColor`/`safeHref`/`safeImageSrc` into one scheme allow-lister, or accept-annotate the trio (§f44).
11. Table-drive `selectorResolver.exists` off `kindTable` so a new kind cannot forget registration without relying on the exhaustive lint (§f46 — the lint already covers it; the table would self-document).
12. `selection_test.go`'s `renderSelection` helper and `export_test.go`'s `renderFixture` overlap — fold or accept-annotate before art-dupl finds them.
13. Name the lint-config allow-list entries by OWNING module (§f48: taskmanager's `go-etag/server`, `httputil`).
14. The `hide` vocabulary should cross-check `HideSection` values against EventCatalog's keys in one test so a typo'd constant fails loudly.
15. `LoadExport`: validate `references` values against the three-mode enum (currently any string passes into `ReferencesMode` and unknown values behave as omit — validation would reject typos like `name-only `).

**A3 aftermath / verification hygiene**
16. After the quiet-window coverage re-pin, re-run `check-file-size` and `check-duplication` to confirm the final tree keeps both green.
17. Sweep `git log --grep="chore: auto-commit" --catalog/architecture` and link the four absorbing commits from the CHANGELOG entries' next edit (audit trail for readers who arrive by message search).
18. Consider a `docs/status/` index entry for this report (the status README tracks reports).
19. Re-verify `#verify`'s doc-assertions leg against the SKILL.md paragraph added today (it was doc-checked, but the assertion leg is stricter than doc-check).
20. If the chain times out at max-wait under the census session's churn, add the census session to the handoff note so tonight's operator knows WHY the window never opened.

**Downstream (A26 rows, consumer repos)**
21. eventcatalog-hub: add `-architecture work/out/<source>/architecture.html` to the hub export step + per-source download link.
22. eventcatalog-hub: partner-slice pilot — one `-spec` file per audience, the redaction story's first real consumer.
23. cqrs-htmx: one-line adoption `docserver.Config{ArchitectureReference: true}` on its docs boot.
24. go-appkit: same one-liner on its admin shell.
25. Both repos: decide whether their CHANGELOGs mention the served route at the next tag wave.

**Documentation truth**
26. `catalog/README.md`: link the new `example/architecture` from the architecture section (run line exists only in the example's doc comment).
27. `catalog/README.md`: the routes table row says "opt-in" — add the one-line config snippet beside it.
28. ADR-0160: add an implementation-status addendum (selections shipped, `contacts`/`x-*` rejected, `references`/`hide`/`--check`/frontmatter shipped) — the ADR predates the implementation.
29. `docs/agents/gotchas-testing.md`: the leak-test's triple-mutation lesson (layered defenses need composed-property tests, not single-site mutants) belongs there.
30. SKILL.md architecture paragraph: add `Inspect`/`-check` to the option list (it currently ends at `LoadExport`).
31. `catalog/AGENTS.md`: record the `/tmp` toolchain-repair recipe next to the gowork-modes pointer.
32. Root AGENTS quick-ref: the `check-architecture-reference` row exists — add its `--self-test` fault-injection count (3) for parity with other gates' rows.

**Fleet-wide (record-only owners)**
33. Fork campaign: ~180 `art-dupl:accept` directives or the single deliberate re-pin (existing 🔥 row, unchanged).
34. task-45 go.sum sweep: merge with the arch-lint triage row (existing errata, unchanged).
35. `core: { in: . }` arch-lint root cause: stays with the v4↔v5 campaign.
36. Coverage re-pin: quiet window only (this session deferred it twice; the gate warning was right both times).

**Smaller polish (noticed, not done)**
37. `report.go`'s `Inspect` walks `resolve.edges` per kept key — for 1 000-service catalogs a cached adjacency would drop the O(N·E) inspect cost; benchmark `Inspect` separately from `Export` when the hub adopts it.
38. `TestReferencesNameOnlyKeepsNames` asserts exact `<span class="rl rl-plain">` markup — brittle against a CSS-class rename; prefer text-presence + link-absence (the leak test's shape).
39. `architectureCache` never expires by wall clock — a long-lived server with an unchanged catalog INSTANCE but regenerated content underneath (impossible today, possible if Catalog ever gains mutation) would serve stale bytes; the immutability assumption is load-bearing and should live in a comment on `CatalogProvider`.
40. `ExportReport.Dropped` only counts structured edges — owner/anyOf drops are invisible to the report; either instrument the degrade paths or document the boundary in the field's doc comment.
41. `-spec` + `-include` currently COMPOSE (spec options first, flags append) — pin that precedence in a test before the hub relies on it.
42. `LoadExport` does not enforce `.md/.mdx` extensions — any path with frontmatter parses; fine, but the doc comment says ".md/.mdx" — align either way.
43. The example writes `architecture.html` into CWD — a `-o` flag would keep it from littering repo roots when someone runs it in-repo (the daemon would commit it).
44. `benchmark_test.go`'s generator is deterministic but unseeded IDs (`svc-%04d`) — if the sidebar's alphabetical sort ever becomes load-bearing for perf, randomize in a second benchmark variant.
45. `check-architecture-reference.sh` could reuse `largeCatalog`'s generator (currently a separate fixture) once the gate grows a scale assertion.
46. The docserver route serves regardless of `EnableCSP` — if a deployment NEEDS the header on every route, offer `Config.ArchitectureReferenceCSP bool` rather than surprising them.
47. `ExportSpec.Options()` always sets `WithTitle` — a spec that wants the CATALOG's title cannot express that; a `title: catalog` sentinel or an omitted-title allowance would fix it.
48. `hide` + `references: include` compose untested: hiding `owners` while pulling one level could re-introduce owner names via edges — worth one test before the hub ships partner slices.
49. `TestDocsServer_ArchitectureReference/enabled` asserts any `id="service-` — with a fixture catalog it could assert the exact anchor; cheap tightening.
50. The status-report index (docs/status/README.md) may need this report's row — not verified in this session's run.

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Verify-slot ownership tonight:** the census session (fleet-census evidence churn) and the disk-truth session (authored commit 17:18) are LIVE on this tree while my verify-chain waits for a quiet window that may not open before its ~19:13 max-wait. Should I re-arm the chain with a longer window (it can run overnight), or do you want the composed `#verify` claimed by one of the other sessions instead? I will not double-arm.
2. **Authored-history retrofit:** my four phase commits collapsed into `chore: auto-commit` blobs (d2). Do you want a rebase-and-reword pass over the 2026-10-17 23:20 → 2026-10-23 15:15 window to restore authored messages (interleaved with OTHER sessions' commits — a rebase would need their cooperation), or is the CHANGELOG narrative the canonical story and the blob history acceptable?
3. **A26 adoption timing:** the hub/cqrs-htmx/go-appkit one-liners (next-items 21–24) live in CONSUMER repos I did not touch this session. Do you want me to open them as rows only (already stamped on TODO_LIST), or actually go implement the hub's export-step line and the two app config lines now — and if now, in which order?

---

*Prepared per the standing status-report contract; sections (a)–(g). No unrelated research performed — everything above is this session's own run plus what its gates surfaced.*
