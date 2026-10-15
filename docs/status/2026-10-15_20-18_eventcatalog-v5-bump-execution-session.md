# Status — EventCatalog v5 Pin Execution Session (spike → BUMP → shipped)

- **Written:** 2026-10-15 20:18 CEST
- **Session window:** ~19:20 → 20:18 CEST (single session, this repo; hub receipts in `~/projects/eventcatalog-hub`)
- **Scope:** execution of
  [`docs/planning/2026-10-15_19-17_SUPERB-eventcatalog-v5-pin-and-leverage-plan.md`](../planning/2026-10-15_19-17_SUPERB-eventcatalog-v5-pin-and-leverage-plan.md)
  — all 14 comprehensive tasks / 58 micro-tasks, including the bump-contingent band.
- **Sources:** the 19-17 plan, the 18:02 licence-impact analysis + its addenda,
  the live render-gate runs (`/tmp/ec-v5spike`, `/tmp/ec-v5gate-master`), the
  parallel 03-01 session's commits observable in the shared tree.
- **Format note:** `.md` per explicit operator demand (status-report skill's
  canonical default is styled HTML — one-off override, not propagated).

## Executive summary

The stay-vs-bump question is **CLOSED by evidence and SHIPPED**: a
scratch-worktree spike (pin `^5.1.0` → core 5.1.1) render-gated the entire
every-resource-kind fixture clean, proved the 4.6.3 agent-changelog crash
fixed upstream, confirmed the 3-collection sidebar WARN persists (cosmetic),
and found custom docs now route standalone under community v5. Ruling:
**BUMP**. Landed as one decision commit (`8bdc12d76`) with pin + goldens +
guard drop + fixture/gate simplification; master render gate re-run green
(108 pages, semantic checks + linter clean). The mesh-demo `order-lifecycle`
flow shipped (`288004068`) — the hub's "Flows: 0" becomes 1 on its next
merge. Tripwires, procedures, verdicts, sketches, and receipts landed on
both sides of the repo boundary. Foreign WIP untouched (one absorbed-file
incident caught and split out within minutes).

---

## a) FULLY DONE

1. **T1 spike** — worktree `../gcl-v5spike`, pin bump, full
   `check-eventcatalog` (exit 0), forced crash-path probe (agents +
   `changelog: { enabled: true }` → exit 0, agent changelog page renders,
   zero `getBadgeHref`), custom-docs routing check (`/docs/custom/<slug>/`),
   evidence table recorded. Worktree + branch trashed after extraction.
2. **T2 ruling + receipt** — BUMP, evidence-ruled; TODO_LIST pin row closed
   (`[x]`) with two dated receipts (licence leg + spike evidence).
3. **T12a format re-verification** — every format-contract point (pointer
   objects, channel pointers, `x-*`, sidecars, flow node kinds, badges,
   custom docs) built clean under 5.1.1 via the fixture; documented in
   catalog/AGENTS.md contract header.
4. **T12b** — sacred goldens regen (`UPDATE_SNAPS=true` scoped → restore
   cross-deletions; only `package.snap` moved), `shouldEnableChangelog`
   agent exclusion dropped, ec-fixture changelog profile folded into
   default, gate simplified 2 profiles → 1 (+ folded assertions incl. a new
   agent-page check), master render gate green.
5. **T12c** — licence behavior recorded (BSL 1.1 on the package; builds
   never need a key), CHANGELOG `[Unreleased]` Changed row
   (check-changelog-symbols green), hub sync receipt row, api-surface
   verified unchanged (hook gate green; no exported symbol touched).
6. **T3 flow pilot** — `order-lifecycle` flow in the mesh-demo orders
   catalog (7 steps: customer → orders-svc → order.placed → Billing
   external → invoice.issued → orders-svc → order.completed), `Flow.Owners`
   set, billing as EXTERNAL node (honest cross-context representation),
   tests green, `flows/order-lifecycle/index.mdx` export verified, hub
   hand-off receipt (their merge picks it up; zero hub edits).
7. **T4 tripwire + procedure** — catalog/AGENTS.md: agent-changelog crash
   marked FIXED (regression recipe included), sidebar WARN marked OPEN with
   exact route/`itemTypes` evidence, and a 6-step reusable pin-bump
   procedure recorded from this execution.
8. **T5 hub corrections** — hub TODO_LIST: flows-row receipt (mesh-demo
   shipped), exporter-gap correction row (asyncapi/openapi exporters exist;
   channels/entities/agents first-class; only SYSTEMS lack a config type),
   coordinated-bump sync row. Dated receipt rows only; no new hub files.
9. **T6 doc-truth gates** — doc-check (1336 refs valid), check-md-go (1512
   blocks, no new errors — the new planning sketch's Go fence parses),
   check-adr-numbering (156 ADRs, lockstep), check-changelog-symbols.
10. **T7 version sweep + policy** — all live-doc `4.6.3` prose updated
    (README §338/§522/custom-docs table row, FEATURES, AGENTS); caret
    rationale recorded at the constant; historical mentions verified
    intentional and left.
11. **T8 upstream watch** — ROADMAP open-question #19 converts the v4-EOL
    trigger into a post-bump security/regression watch (no calendar
    trigger; procedure-verified at each future bump).
12. **T9 llms.txt contract** — godoc on `writeLLMsTxt` + AGENTS bullet: the
    file is the standalone MCP server's only input; format changes are
    breaking.
13. **T10 ADR groundwork** — design-only sketch filed
    (`docs/planning/2026-10-15_20-05_adr-emission-groundwork-sketch.md`):
    `catalog.ADR` type sketch + 8 acceptance criteria + non-goals;
    TODO_LIST row receipt; implementation still blocked on hub T7.1.
14. **T11 verdicts** — ROADMAP #20–22: SSO not-adopting, Studio-v2
    after-content, 3-level-diagrams vs catalog/d2 coexist.
15. **Anti-entombment** — the genuinely new follow-ups discovered during
    execution harvested into TODO_LIST § "EventCatalog v5 bump follow-ups"
    (6 rows) BEFORE this report was written.
16. **Committed receipts** — three authored commits: `288004068` (flow),
    `8bdc12d76` (v5 decision, 11 files), `503853c67` (verdicts/receipts/
    sketch, 4 files). Hub receipts committed by its daemon (`096823c`,
    content verified present).

## b) PARTIALLY DONE

1. **CI licence-key behavior (T12c.1)** — recorded from the blog contract
   ("builds never break", reminder banner only) + local gate evidence; NOT
   empirically verified in an actual CI run (needs the nightly/CI leg to
   run once post-bump and its log read).
2. **T5 format** — the plan's "receipts under hub T15/T18" was impossible:
   the hub rebuilt its TODO_LIST (10-12/10-14 passes) and those rows no
   longer exist. Adapted to dated receipt rows under the current rows;
   intent honored, letter diverged.
3. **T6 breadth** — all three doc gates green, but the skill-references
   truth-check (grep references/*.md for stale two-profile/4.6.3 prose) ran
   only after the report demand — result: clean, no stale hits (closed, but
   late).
4. **Master-run WARN count** — the "4 broken sidebar links" figure is
   spike-verified; the master gate's grep window showed 3 (likely tail
   truncation, unverified in that specific log).
5. **Verification breadth** — no full `#verify`/`#verify-fast` after the
   bump commit (deliberate: no exported API change; catalog suite + render
   gate + BuildFlow pre-commit pipeline covered the touched modules). The
   composed chain should still run at the next session close.
6. **Status live-index** — this report IS registered (see index update);
   the two sibling 10-15 reports (18-02 licence analysis, 03-01
   query-http) are still missing from `docs/status/README.md` — not mine to
   register, flagged in (e).
7. **Daemon-authored hub receipts** — content landed (verified by grep +
   commit stat) but under a `chore: auto-commit` message; my authored
   attempt no-oped because the daemon had already absorbed the edit.

## c) NOT STARTED

1. **Push** — 3 authored commits sit unpushed on this repo's master
   (no-push rule; needs owner say-so).
2. **Tag/release wave** — the catalog module behavior changed but nothing
   is tagged (deliberate; the 2026-10-06 batch-release lesson requires
   per-module test runs before any wave — done for catalog/mesh-demo, but
   the wave itself is a separate decision).
3. **Free-Use licence key claim** — owner-side self-certification via
   EventCatalog Cloud + `EC_LICENSE` wiring in the hub (hub [owner] row;
   unchanged).
4. **Hub 5.1.x coordinated bump** — hub still runs `^5.0.1`; the sync note
   targets 5.1.x on both sides (their move).
5. **ADR emission implementation** — blocked:upstream on hub T7.1 schema
   confirmation (by design; sketch ready).
6. **bank-sync flow pilot** — hub row's other half, their repo.
7. **Upstream issue filing** (sidebar WARN + itemTypes ask) — candidates
   harvested, verify-before-filing pass not run.
8. **cqrs-lint `System` config type** (the one real exporter gap) —
   demand-gated on a hub ask; not started.
9. **Studio v2 / swimlanes evaluation** — parked by verdicts #20–22 until
   content exists (flow merged + ADR emission).

## d) TOTALLY FUCKED UP

1. **One real incident, self-inflicted and fixed in minutes:** my FIRST
   flow-pilot commit absorbed the parallel session's three STAGED
   `cmd/doc-check/recipes_*` files (I ran `git commit` bare while knowing
   the handoff said foreign files were staged). Caught by reviewing the
   commit stat (4 files ≠ 1), repaired via `git reset --soft` +
   pathspec-limited re-commit (`288004068`, exactly 1 file); foreign staged
   state restored intact. No damage; the lesson is now procedure:
   **pathspec-first commits whenever a parallel session is live.**
2. **Nothing else.** No data loss, no foreign work destroyed, no gate left
   red, no ghost code. The bump commit's BuildFlow "9 step(s) failed" ERROR
   is report-only preflight noise (pre-existing pseudo-version hygiene in
   `metaengine/go.mod` + taskmanager tidiness — both foreign/pre-existing,
   both flagged in (f)).
3. **Observed, not mine, worth escalating:** the parallel 03-01 session's
   `f1f0e82d5` (20:11) deleted `tools/erraudit/erraudit-v0.6.1-src.tgz`
   while `.github/workflows/ci.yml`'s error-audit job references it —
   if their rebuild doesn't re-add it before push, that CI leg breaks on
   master. Left untouched per the foreign-WIP rule.

## e) WHAT WE SHOULD IMPROVE

1. **Pathspec-first commits under live parallel sessions** — the handoff
   warned about staged foreign files; obeying it mechanically from commit
   #1 would have avoided the only incident.
2. **Blast-radius grep BEFORE declaring doc gates done** — the references
   truth-check should have been part of T6, not a post-report afterthought
   (it came back clean; luck is not a gate).
3. **Gate warn-budgeting** — the render gate tolerates all warns; asserting
   the known signature would have made the master-run "3 vs 4 links"
   question impossible (harvested as a TODO row).
4. **Golden coverage followed the guard drop immediately** — the
   config-with-changelogs gap existed because the golden fixture catalog
   has no changelogs; noticed only when `eventcatalog-config.snap` didn't
   move (harvested).
5. **Spike evidence durability** — evidence lived in /tmp logs until
   receipt time; the status addendum now carries the table, but writing the
   evidence table BEFORE trashing the worktree (T1.8) would be the safer
   order next time.
6. **Commit the hub receipts faster than the daemon ticks** — or accept
   `chore:` absorption knowingly (it matched their repo's norm).
7. **Register every session's status report in the live index at write
   time** — three 10-15 reports exist; only mine is indexed. Index rot is
   the exact class the 13th docs-health pass killed.
8. **The plan's micro-task format paid off** — 58 ≤12-min steps executed
   with zero re-planning; keep the format for future SUPERB plans.

## f) Next things (up to 50; brainstorm, not commitment — ROADMAP fuel)

1. Push the 3 authored commits (owner go-ahead).
2. Tag-wave decision for the catalog change (per-module tests done; wave +
   receipts per the batch-release lesson).
3. Claim the Free-Use licence key; wire `EC_LICENSE` into hub build.sh +
   build.yml (hub [owner] row).
4. Verify CI's error-audit leg after the parallel session re-adds the
   erraudit tarball (currently deleted on master).
5. Confirm hub merge picks up `order-lifecycle` → dashboard "Flows: 1" +
   v5 swimlanes render it.
6. Field-verify agent changelog + `/docs/custom/<slug>/` routing on the
   REAL hub build (fixture evidence ≠ production evidence).
7. Hub coordinated bump to `@eventcatalog/core` 5.1.x (both-sides target).
8. Warn-budget hardening for `check-eventcatalog` (fail on NEW warns;
   TODO row harvested).
9. Add the config-with-changelogs golden (TODO row harvested).
10. File upstream: 3-collection sidebar changelog WARN (verify-first).
11. File upstream ask: extend changelog route `itemTypes` (verify-first).
12. ROADMAP `[Unreleased]` wave prose: mention the v5 pin bump (TODO row).
13. Record node/npm versions + page-count baseline in the bump procedure
    (TODO row).
14. `-race` pass for catalog/eventcatalog post-guard-drop (TODO row).
15. Run the composed `#verify` chain over the bump commit at next close.
16. Run `check-readme-links` + `check-readme-deprecated` once over the
    README edits (nightly gates; cheap local pass).
17. Register the two sibling 10-15 status reports in the live index
    (their sessions' rows — needs coordination, not silent fixing).
18. bank-sync flow pilot (hub row's other half; their repo).
19. ADR emission: execute the sketch when hub confirms T7.1 schema.
20. Decide `catalog.System` config type (only real exporter gap) on a hub
    ask; otherwise keep out (budget is FULL).
21. Hub: `explainBusinessFlow` MCP tool against the merged flow.
22. Hub: npm-audit revisit now on 5.1.x (their [ready] row, refreshed
    target).
23. Hub: `llms-full.txt` content review ([ready] row; our llms.txt is the
    input — format contract note landed this session).
24. Hub: integrations-visibility A-vs-B call ([owner] pending).
25. Evaluate v5 flow grouping/swimlanes config for the hub once flows
    merge.
26. Normal `#verify` blocker cleanup seen in BuildFlow preflight:
    `metaengine/go.mod` pseudo-version drift (sqliteengine v4.5.3 → zero
    pseudo-version) — pre-existing, needs `normalize_versions` + `go work
    vendor`.
27. `example/taskmanager` go.mod tidy + 4 compile errors — the parallel
    session's; re-run doc/lint gates after they land.
28. Re-run doc-check after `.agents/skills` foreign edits land (references
    are gate inputs; my run covered their current state — green).
29. Studio v2 eval trigger: revisit after ADR emission + flow merge
    (verdict #21).
30. Upstream watch items at next bump: WARN signature fixed? agent crash
    regressed? custom-docs still routed? (procedure step 1 covers).
31. Consider a fixture profile matrix note in ec-fixture docs if profiles
    ever grow again (one-profile simplicity is now load-bearing).
32. Check whether any consumer doc quotes "two fixture profiles" outside
    this repo (cqrs-htmx examples/catalog-demo docs) — sweep candidate.
33. `ec-fixture` usage strings in other scripts: grep for `changelog`
    profile callers repo-wide (gate was the only known caller; a sweep
    makes it certain).
34. Add `-count=1` freshness to the closing catalog test invocation
    habitually (this session's final run used it; make it default).
35. Golden-test snap-dir fragility (4-package shared dir) — already
    documented; a `snaps_clean_test` guard exists; no action, listed for
    awareness.
36. Free-tier licence banner check: read one hub build log for the
    reminder banner text (documents T12c.1 empirically at zero cost).
37. mesh-demo: consider a second flow only if the hub asks for
    swimlane-specific shapes (orders flow covers the pilot).
38. llms.txt: snapshot the current structure into the AGENTS contract
    bullet if it ever changes (tripwire exists in godoc; nothing to do
    now).
39. `docs/drafts/2026-10-03_issue_comments.md` carries 4.6.3 mentions —
    frozen draft, leave (verified intentional; listed so nobody "fixes"
    it).
40. Consider teaching the pin-bump procedure's step 2 (forced crash-path
    probe) as a tiny script so future bumps can't skip it (candidate
    `scripts/` helper; only if bumps become regular).
41. Watch the hub's ROADMAP foreign mods (they were mid-edit at 20:10) —
    coordinate before pushing anything hub-side.
42. The 18:02 report's §g list: remaining items are either answered by
    this session (pin, CI licence) or hub-owned; sweep it into an ANNOTATE
    pass at next docs-health.
43. Plan-file addendum: optionally append a one-paragraph "executed
    2026-10-15" banner to the 19-17 plan (its own harvest note says
    optional; receipts live in TODO_LIST).
44. BuildFlow preflight warn (root AGENTS.md 323 > 220 lines) — pre-existing;
    a future restructure could move gotchas to topic files (the split
    already exists; the TL;DR block is the overage).
45. `check-canonical-facts` counts unchanged this session (no go.mod
    changes) — nothing to do; recorded so nobody re-runs the recount
    unnecessarily.
46. Consider pinning the render gate's npm install duration expectation
    (~2m cold) in the gate header to distinguish cold-cache CI from
    warm-cache local runs.
47. Upstream feature ask candidate: `x-go-*` extension preservation is
    already our pattern; nothing new — listed to close the loop on the
    T12a checklist.
48. When the hub merges the flow: check catalog.index.json picks up the
    flow entry (their drift-gate soak week interacts).
49. Next docs-health pass: verify today's five edited living docs
    (catalog/AGENTS, catalog/README, FEATURES, TODO_LIST, ROADMAP) against
    the code (three files changed behavior-adjacent prose).
50. Owner Q3 below determines the immediate next session's shape — push +
    tag vs hold.

## g) Questions I cannot answer myself

1. **Push now or hold?** Three authored commits (`288004068`,
   `8bdc12d76`, `503853c67`) sit unpushed on master, and the parallel
   session just landed `f1f0e82d5` on top (with the erraudit tarball
   currently deleted). Do you want the v5 work pushed as-is, held until
   the parallel session's tarball repair lands, or bundled into the next
   tag wave?
2. **Licence key: claim or stay keyless?** Builds never break without it,
   but the Free-Use key (self-certified via EventCatalog Cloud) removes
   the reminder banner and unlocks the standalone MCP server against
   catalog.home.lan. Claim it now and wire `EC_LICENSE` into the hub, or
   keep deferring?
3. **Tag timing for the catalog change?** The exporter behavior changed
   (pin + changelog emission). Ride it into the next natural tag wave
   soon, or hold until the hub has merged the flow and you can co-verify
   end-to-end on catalog.home.lan?

---

## Verification ledger (what was actually run)

| Gate / test | Result |
| --- | --- |
| `nix run .#check-eventcatalog` (spike worktree, pin ^5.1.0) | exit 0; 98 + 103 pages; linter clean |
| Forced crash-path probe (agents + changelog on) | exit 0; 108 pages; agent changelog page renders |
| `nix run .#check-eventcatalog` (master, post-decision-commit) | exit 0; 108 pages; semantic checks + linter clean |
| `catalog` module build + vet + tests (incl. `-count=1` eventcatalog) | green; only `package.snap` golden moved |
| `example/mesh-demo` tests + orders export | green; `flows/order-lifecycle/` correct |
| doc-check (full file set) | 1336 references valid |
| `check-md-go` | 1512 blocks valid, 0 new errors |
| `check-adr-numbering` | OK (156 ADRs) |
| `check-changelog-symbols` | 13 citations honest |
| Pre-commit pipeline (×4 commits) | api_surface OK (6302 exports); syntax gate green; BuildFlow report-only warns are pre-existing |
| Sacred goldens procedure | `UPDATE_SNAPS=true` scoped → `git ls-files -d … restore --` → module suite green |

## Files & commits (this session, authored)

| Commit | Content |
| --- | --- |
| `288004068` | mesh-demo `order-lifecycle` flow (1 file, +66) |
| `8bdc12d76` | v5 pin decision: exporter/writer/writer_llms, ec-fixture ×2, `package.snap`, gate script, catalog/AGENTS+README, FEATURES, CHANGELOG (11 files, +144/−134) |
| `503853c67` | verdicts + receipts + ADR sketch: TODO_LIST, ROADMAP, 18-02 addendum, 20-05 sketch (4 files, +195/−3) |
| hub `096823c` (daemon) | hub TODO_LIST receipt rows (+29) |

Plus, written after the working session: TODO_LIST follow-up harvest
section (6 rows), this report, and its live-index row.
