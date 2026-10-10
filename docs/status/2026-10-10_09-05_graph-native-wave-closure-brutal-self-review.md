# Graph-Native Wave Closure — Status + Brutal Self-Review (session of 2026-10-10, waves 3–5)

> **Scope:** this report covers ONLY this session's run (Wave 3 closure → Wave 5 push,
> `018202325..f12a849b8`) and what I noticed while doing it. No fresh research beyond
> self-verification of my own changes. Format: user-demanded `.md` at this path (overrides
> the status-report skill's HTML default — flagged per skill contract).
>
> **Inputs:** plan V2 (`docs/planning/2026-10-10_06-11_SUPERB-…-V2.md`), the 07-37
> interrupt report, both T19 dry-run agent reports, gate outputs, git history.

---

## a) FULLY DONE (verified green)

1. **Wave 3 closure** — D6: `check-readme-deprecated` clean; the 23 broken README links
   are ALL pre-existing `core/v5` (broken since 2026-10-09, out of scope per user default).
   D11: TestRecipes compile green, doc-check 1234 refs valid, md-go 105-baseline green.
   The 07-37 status report committed with authored message (`5dac329cd`).
2. **Wave 4 — ADR-0157** (engine fleet two-level story, Proposed): blank-import candidates
   at compile time + `cqrs.yaml`/`LoadConfig` routing at runtime; **allengines convenience
   module REJECTED on measurement** — all 10 pure-Go engines: 68,185,216 B / 221 unique
   modules / 871 graph edges vs sqlite-only 12,747,630 B / 64 / 168 (**5.3× / 3.5× / 5.2×
   tax**); 12-driver membership table (duckdb `//go:build cgo`, bigtable excluded-heavy,
   pebble+bbolt graph-less).
3. **Wave 4 — ADR-0156** (graph edge labels at v5, Proposed): `Edge{From,To}` stays
   label-less in v4.x (collection-per-relation is the model, `Driver()` the hatch);
   labeled edges reopen ONLY on a named fleet-consumer need. Signature survey pinned:
   `GraphAddEdge(ctx, collection, Edge)` uniform across 9 engines + graphadapter.
4. **ADR numbering collision resolved** — parallel session's Accepted 0155 (declarative
   schema evolution, decided 2026-10-09) kept the number; mine renumbered to 0157 via
   `git mv` + reference sweep; ADR index deduplicated (parallel rows for 0152–0154
   removed, titles now match files exactly, rows 0152–0157 once each).
5. **Wave 5 — T18 ADT-recipe ratchet** (`cmd/doc-check/adt_coverage_test.go`): parses
   `AllADTs()` live from `metaengine/enum_validation.go` (no module dep, drift-proof both
   directions); 6 ADTs covered by recipes.md headings, 5 waived with reasons visible in
   test output; **mutation-tested** (dropped waiver, empty reason, missing pattern — all
   three correctly fail, restore green).
6. **Wave 5 — T19 consumer dry-run, GREEN at iteration 2 of ≤3**: round 1 (fresh agent,
   docs-only diet) produced a fully correct implementation answer + 12 stalls; 9 fixed:
   write-side decider fence added to recipes §2.44 (compile-verified as block #3 with
   catalog entry), `metaengine/graphadapter/README.md` NEW, system/README `Events` field +
   "Declaring projections" section, projectionadapter typed-decoder section
   (`NewTypeDecoder`/`Register`/`EventWithID`), metaengine README modernized off
   deprecated `OnTyped`, SKILL.md fold enumeration canonicalized, bbolt exclusion +
   `postgres` driver-name honesty in §6.13, `Reachability` name aligned, placeholders
   annotated. Round 2 confirmed all blockers closed; 5 micro-fixes applied (Undirected
   field shape, unordered-results semantics, `id.ParseStreamID` line, iroh driver +
   graph-memory-not-deployable note, `sys.Start` lifecycle line).
7. **G8 harvest** — TODO_LIST IN FLIGHT marker → DONE with adopt-if-needed framing + new
   ADT-recipe-gap row (5 waivers, gate refuses stale waivers).
8. **G9 closure** — non-destructive resolution appendices on BOTH historical reports
   (05-48, 07-37); V1 plan supersede banner verified intact.
9. **Final push** — authorized boundary only, `018202325..f12a849b8`, tree clean.
10. **Post-hoc self-verification (this report):** `metaengine/graphadapter` compiles
    standalone after my doc.go edit; example suite green at close; every daemon-race
    content loss verified landed via pickaxe/grep.

## b) PARTIALLY DONE

1. **Authored-commit coverage ≈ 50%**: 3 authored messages survived (`5dac329cd`,
   `273e33c91`, renumber commit) out of ~8 logical changes; Wave-5 T18+fixes and the
   final closure commit lost messages to daemon ref-lock races (content verified landed;
   amending mixed 500-file daemon commits ruled out). One loss was partly self-inflicted
   (heredoc syntax error on first final-commit attempt).
2. **T19 residuals, by design**: `follow()`/`unfollow()` decide bodies and `FollowCmd`
   construction live only in `example/graph-native` (recipes point there); per-engine
   EdgeRemoval/undirected capability discoverable at runtime rather than documented
   per-engine. Acceptable, but a consumer without repo access hits the pointer wall.
3. **Example standalone build**: workspace-green + standalone audit green, but `GOWORK=off
   go vet` fails until the parallel session TAGS `systemscenario.Memory()` (pinned v4.0.0
   lacks it). The dependency is invisible to the owning session — no row anywhere.
4. **Engine-capability matrix**: now correct in 5 places (ADR-0157, advanced §6.13,
   system/README, COOKBOOK, modules.md) but kept in lockstep by nothing.

## c) NOT STARTED (this session's own backlog)

1. **THE ENTOMBMENT (biggest miss):** the 07-37 report §f carried ~45 next-items
   (coeffect-gate RawQuery blindness product-fix, Then-baseline doc gap, deriver
   async-delivery product TODO, 7 classes of pre-existing red, …). G8 harvested only the
   ADT waivers + marker flip. **Verified just now:** the specific coeffect rawQuerySpec
   item is NOT in TODO_LIST (the 6 coeffect mentions are other work). A timestamped file
   is now the only home of live work items — the exact anti-pattern the status-report
   skill warns about.
2. **FEATURES.md graph-native row: VERIFIED 0 mentions.** The keystone example (registered
   in 6 gates in Wave 1) and the Graph-ADT recipe wave are invisible in the feature
   inventory that owns the examples maturity matrix.
3. **Root CHANGELOG [Unreleased]**: no Added rows for example/graph-native, recipes
   §2.44, ADR-0156/0157, T18 gate, graphadapter README. Nothing forces these to exist
   (the symbols gate only validates what's cited).
4. **Capability single-source / generation gate**: not started (see e-2, f-5).
5. **Measurement harness**: ADR-0157 numbers came from /tmp ad-hoc probe modules — not
   re-runnable, not checked in.
6. **Full `nix run .#verify` post-wave**: not run (doc gates + touched-module builds only;
   #verify exclusivity + parallel churn). Risk assessed low — one comment-only .go edit
   (now verified compiling) and docs otherwise — but "low risk assessed" is not "ran".

## d) TOTALLY FUCKED UP (owned)

1. **ADR numbering collision, handled reactively**: I claimed 0155 from an `ls docs/adr |
   tail` snapshot without checking untracked files; the parallel session's Accepted 0155
   landed on disk at 08:12, mine committed 08:15. Caught by LUCK (the `??` line in my
   commit output), not by process. Cost: renumber churn, one authored commit whose pushed
   message says "0155", index duplicate rows (parallel session also added its own
   0152–0155 block — I deduped).
2. **Echoed the plan's engine list before verifying it**: built the 8-engine probe per
   plan V2's membership line, THEN noticed dgraph+pebble missing and re-measured with 10.
   First ADR draft carried plan-echo numbers. Self-caught — but this is the known
   "instructions over evidence" failure mode, committed twice in two sessions now.
3. **Recipe-catalog fumbles (3 wasted compile cycles):** inserted fence mid-section
   without realizing specs key by heading+occurrence (cross-assigned specs); then wrote a
   `var follow` preamble without knowing `splitTypeDecls` hoists only `type`/`func` decls.
   Should have read `generateRecipeSource` before writing spec #1.
4. **Repeated a documented gotcha**: the stray `graph-native` binary from `go build ./...`
   in the module dir — known from the PRIOR session, done again this session (trashed).
5. **Broke 2 README links** by copying `../../` depth from a nested module
   (projectionadapter) into top-level `system/README.md` — caught only in the final
   link-gate sweep, not at edit time.
6. **Sub-agent contamination**: the round-1 dry-run agent cited IDE diagnostics (LSP
   false-positive lint warnings bleeding into its context) as evidence the reference app
   was "partially unwired" — wrong, verified — but my audit design couldn't prevent
   ambient diagnostics from polluting its stalls, and I had to burn a verification cycle
   disproving it.

## e) WHAT WE SHOULD IMPROVE (structural)

1. **Harvest discipline at session close**: any report with next-items must feed
   TODO_LIST before the session ends. The skill says it; I skipped it under wave-closure
   pressure. Cheap fix, expensive rot.
2. **Generate the engine-capability matrix** (driver census from `register.go` +
   capability probes via `HasXxx` interfaces) into ONE table that the other four docs
   embed/link. Kills the 5-way split brain I just widened.
3. **ADR number claiming needs a guard**: check committed + untracked + recent-daemon
   files before assigning; a tiny `check-adr-numbering.sh` (gaps, duplicates,
   untracked-collision warnings) closes the class.
4. **Wave checklist should include FEATURES.md + CHANGELOG rows** — module registration
   in 6 gates (Wave 1) with zero feature-inventory presence is how inventory drift starts.
5. **Harness DX**: TestRecipesCompile failures would be half as costly if the error dump
   printed the catalog KEY each `recipe_lNNNN` matched.
6. **Audit-agent hygiene**: instruct sub-agents to ignore IDE diagnostics entirely; treat
   meta-observations (lint output, file mtimes) as unverified until source-checked.
7. **Commit strategy**: pre-stage + slow hooks lose to the daemon ~50% of the time.
   Either accept `chore:` absorption as norm and author only the final boundary commit,
   or build a commit-queue; racing it per-wave is pure overhead.

## f) NEXT — up to 50, impact-sorted (session-derived only)

**Harvest & inventory (do first):**

1. HARVEST the entombed 07-37 §f items into TODO_LIST (docs-health HARVEST mode).
2. TODO_LIST row: coeffect gate blind to `RawQuery` — product fix = `buildProjections`
   extracting event types from `rawQuerySpec` folds (or rule documented-forever).
3. FEATURES.md: graph-native example row + Graph-ADT read-model maturity row.
4. Root CHANGELOG [Unreleased] Added rows: example/graph-native, recipes §2.44,
   ADR-0156/0157, T18 gate, graphadapter README.
5. TODO_LIST row: example standalone-build blocked on `systemscenario` tagging `Memory()`
   (owner: parallel session; invisible to them today).

**Split-brain kills:**
6. Engine-capability generation gate (single source; ADR-0157/advanced/system/COOKBOOK/
modules embed or link). My top structural recommendation.
7. `scripts/measure-engine-fleet.sh` checked in; re-runs ADR-0157's numbers; optional
nightly-bench leg.
8. `scripts/check-adr-numbering.sh` (duplicates + gaps + untracked-collision) + CI leg.

**Recipe-gap closures (each drops a T18 waiver):**
9. ADTSet recipe. 10. ADTLog recipe. 11. ADTStreamLog recipe. 12. ADTSortedMap recipe.
13. ADTMultimap recipe.

**T19 residual polish:**
14. §2.43: Then-baseline = first-act snapshot semantics (07-37 #25).
15. §2.44 appendix or formatting: explicit "full write side incl. guards → example" callout.
16. Per-engine EdgeRemoval/undirected matrix (falls out of item 6).
17. core.md Lifecycle paragraph: `sys.Start` vs `ProjectionHost().Start` canonical order.
18. modules.md: verify `irohengine`/`graphadapter` rows carry driver-name strings.

**Pre-existing red triage (from 07-37 §f-31…37 — needs ONE owning decision):**
19. core/v5 README 23 broken links. 20. cqrs-bench go.sum untidy. 21. core/v5 missing
`.go-arch-lint.yml`. 22. BuildFlow pseudo-version hygiene (sqliteengine v4.5.2).
23. 34 modules need `go mod tidy`. 24. golangci red: system/, scheduling/sqlstore,
stack/sqlite. 25. go-licenses FAIL (bigtableengine). 26. govulncheck toolchain mismatch
(go1.26 vs 1.27).

**Verification & hygiene:**
27. Full `nix run .#verify` in a quiet window as the post-wave confirmation.
28. LSP false-positive warnings in example/graph-native (unused/gci) — stale single-file
analysis; consider an LSP restart note in gotchas or an exclusion; do NOT "fix" by
deleting used symbols.
29. Audit-agent prompt template: add "ignore IDE diagnostics" clause (crush-config skill
repo, not fan-out).
30. Recommit authored message for the T18+fixes and closure changes? Only if history
readability matters — content is verified landed; otherwise drop.
31. ADR-0157 D2: add a "verified from source on" date stamp so future drift is detectable.
32. recipes §2.44 fence #3: fold the `follow` guard bodies inline once round-2 stall #1 is
ruled worth closing (currently example-pointer by design).
33. Consider `graph-native` example: `-race` suite into `testModules`? Currently
build-only example — deliberate; revisit if the BDD suite grows.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Harvest ownership:** should I harvest the entombed 07-37 §f items into TODO_LIST
   right now (mechanical, ~15 min), or does the weekly docs-health standing pass own it
   (accepting rot risk until then)?
2. **Capability matrix:** generated single-source gate (item 6, ~S effort, my
   recommendation) — or do you prefer the 5-place prose duplication with manual
   discipline because the matrix changes rarely?
3. **ADR numbering:** add the hard gate (item 8, CI-failing on duplicate/untracked
   collision), or is convention + my post-hoc dedupe enough given how often parallel
   sessions run?

---

_Point-in-time snapshot. Stale by design. ANNOTATE, never rewrite. Auto-commit daemon
absorbs this file; no manual commit per harness contract._
