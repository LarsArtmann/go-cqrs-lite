# Post-Review Defaults Executed: Harvest, Capability Gate, ADR Gate (session of 2026-10-10 17:00)

> **Scope:** execution of the three 09-05 §g questions' recorded defaults
> (user said continue without answering → harvest now / capability gate / ADR
> numbering gate) plus the highest-value structural f-items. The go-cqrs-lite
> repo skill + docs-health (HARVEST/BUILD) were loaded and followed.
>
> **Commits:** the auto-commit daemon absorbed every edit mid-session (clean
> tree at close; content verified landed via pickaxe: `5757375ed`,
> `db0bb7907`, `71b677dbf`, `3af09dd65`). No authored boundary commit was
> possible — amending mixed daemon commits stays ruled out. No push
> (authorization remains consumed at `f12a849b8`).

---

## a) FULLY DONE (verified green)

1. **Q1 — HARVEST (the entombment is closed):** 07-37 §f 24–45 + 09-05 §f
   routed. 7 new TODO_LIST rows (coeffect RawQuery product fix → Cordis
   section; Then-baseline §2.43 doc + systemscenario-Memory()-tag dependency
   [BLOCKED] → BDD section; graph-native version-manifest row [BLOCKED] →
   Release; pre-existing-red ONE-owner-decision triage row → CI; COOKBOOK
   fences ZERO-automated-checking row (verified against both gate scan sets)
   → Docs truth; example --dgraph integration candidate → CRM debut section).
   The DONE block's "no fleet consumer requires it" claim AMENDED — the Kith
   CRM consumed the Graph ADT the same morning (parallel session's row
   preserved, cross-referenced).
2. **Q2 — engine-capability single source + gate:**
   `docs/engine-capabilities.md` GENERATED from source (register.go driver
   census; GraphAddEdge/GraphRemoveEdge/GraphNeighborsUndirected probes —
   the same methods the runtime capability asserts use; turso→sqlite
   delegation derived; CGo detection) by `scripts/check-engine-capabilities.sh`
   (`nix run .#check-engine-capabilities`, CI leg added, hermetic --self-test,
   5 mutation legs green). The 4 prose places (ADR-0157 D2, advanced §6.13,
   system/README, COOKBOOK) now LINK the canonical table.
3. **Two live doc lies found by the census and killed:** (a) system/README
   listed `iroh` as a blank-import self-registering driver — irohengine has
   NO RegisterDriver (programmatic `Replicated` wrapper; 11 registry drivers
   total, not 12 — the handoff's "12 driver names" was itself wrong);
   (b) COOKBOOK credited duckdb with undirected traversal + edge removal —
   duckdb implements NEITHER (no GraphRemoveEdge, no GraphNeighborsUndirected;
   graph add + traversal only). Also fixed the same implication in
   system/README's graph-capability paragraph and advanced §6.13. ADR-0157 D2
   carries a dated correction note (f-31 closed).
4. **Q3 — ADR numbering gate:** `scripts/check-adr-numbering.sh`
   (`nix run .#check-adr-numbering`, CI leg added, --self-test 8 legs green).
   FAILs on duplicate numbers (filesystem scan — untracked parallel-session
   collisions included, the double-0155 class), index lockstep (row-per-file,
   title==H1), non-conforming filenames; warns on undocumented gaps. First
   real run found and fixed: **60 drifted index titles** synced to file H1s,
   **undocumented gap 0138** (README note + gate allowlist now 0036/0041/0138),
   **`0099a` suffix-numbered ADR** handled as a documented exception (renaming
   would break frozen archived-report links), and a non-conforming-filename
   guard so nothing escapes the NNNN- prefix again.
5. **Inventory truth:** FEATURES.md gains the Graph ADT feature row (next to
   Vector ADT — the previously-missing read-model maturity row) and the
   `example/graph-native` Examples row. CHANGELOG [Unreleased] gains 2 Added
   rows (graph-native wave; gates wave) — symbols gate green at 38 citations
   (it caught and I fixed one bad citation: `sys.Start` → `System.Start`).
6. **measure-engine-fleet.sh:** ADR-0157's numbers are re-runnable
   (proxy mode by default, `--tree` for working-tree replaces). Validated:
   edges EXACT (168/871), sizes within 0.01% (12,748,254 / 68,181,344 B),
   ratios hold 5.35×/3.53×.
7. **Cheap closures in-session (rows NOT created):** core.md quickstart →
   canonical `sys.Start(ctx)` + lifecycle paragraph (f-17); modules.md iroh
   row "NO registry driver" (f-18); system/README links COOKBOOK (#41);
   §2.44 full-write-side callout (f-15); gotchas LSP false-positive bullet
   for example/graph-native (f-28); AGENTS.md gains "Close a Wave / Feature
   Session" procedure (e-4) + 3 Quick Reference rows.
8. **31 MB stray binary:** `example/graph-native/graph-native` (rebuilt +
   daemon-absorbed into history AND pushed to origin) untracked + gitignored
   per the established mesh-demo pattern. History keeps the blobs (no force
   push); the class is now blocked at the ignore layer.
9. **Verifications green:** doc-check 1270 refs; TestRecipesCompile (46 s) +
   ADT coverage + catalog tests; md-go 1509 blocks; readme-deprecated clean;
   readme-links: only the 23 pre-existing core/v5 breaks; api-stability
   `TestEvery` meta ok; example/graph-native tests green; cqrs-lint over the
   example: 4 advisory warnings, 0 errors (#44 closed); both new nix apps
   evaluate + run green; symbols gate green.

## b) NOT DONE / DEFERRED (owned)

1. **Full `nix run .#verify` deferred (f-27):** `nix fmt` is currently
   tree-red from the parallel session's in-flight
   `system/schema_declarations.go:61` ("method must have no type parameters" —
   illegal Go mid-edit; the file compiles seconds later, i.e. live churn).
   A composed #verify now would report foreign in-flight red. Every leg my
   session touched ran green individually (list in a-9). Re-run #verify in a
   quiet window once schema_declarations settles — the CI legs added today
   cover the rest at push time.
2. **e-5 (doc-check error dump prints catalog keys):** not done — read the
   harness, judged the failure output already carries recipe_lNNNN line
   numbers; the heading mapping is a nice-to-have, not worth the churn today.
3. **f-29 (audit-agent prompt template):** lives in the crush-config repo,
   not here — surfaced, not actionable in this repo.

## c) HARVEST DROPS (with reasons, per docs-health)

- 07-37 #26 example README (already exists, real content) · #29 StreamID
  gotcha (FAQ entry exists) · #38 (note-only by its own text) · #39
  WithAwaitTimeout (conditional trigger unmet + systemscenario is
  parallel-owned) · #40 undirected sentence (T19 round-2 field note + CRM
  recipe row cover it) · #43/#44/#45 (executed as verifications this session,
  all green) · f-9..13 (ADT recipes row already exists) · f-16 (closed by the
  capability gate's columns) · f-30 (content verified landed; deliberate) ·
  f-33 (-race suite: deliberate, revisit trigger documented).

## d) PROCESS NOTES

- The capability census corrected THIS session's own handoff: the "12 driver
  registry names incl. iroh" fact was wrong (11 registry drivers; iroh +
  graphadapter are programmatic). Source-greps beat inherited facts — again.
- The ADR gate's 60-row title sync was done by a one-shot awk script (not
  checked in); the gate now holds the invariant.
- Tree-wide `nix fmt` red from parallel churn is the documented transient
  class — confirmed again, not chased.

---

_Point-in-time snapshot. Stale by design. ANNOTATE, never rewrite. Auto-commit
daemon absorbs this file; no manual commit per harness contract._
