# Session Status Report — Graph-Native Docs Audit + All-Engines Design Consult

**Date:** 2026-10-10 05:48 (Saturday)
**Session type:** Read-only research/consulting — ZERO repo changes made
**Scope:** This session only (three questions: graph-native modern API walkthrough → documentation audit → all-engines-by-default + operator-owned DeploymentConfig design consult). Per instruction, no unrelated research was performed.

**Format note:** `.md` per explicit user path instruction (overrides the skill's HTML-canonical default — flagged here per skill contract). Section (f) is HARVEST-ready but TODO_LIST.md was NOT touched: user directive was "report, then WAIT FOR INSTRUCTIONS."

---

## Session Recap

1. **Q1 — "Graph native with the most modern system APIs?"**: Loaded the go-cqrs-lite skill; researched `advanced.md` §6.13, `metaengine/graphadapter`, `metaengine` Edge/EdgeRemoval/ADTGraph/ReadTraversal types, `system` config_types/projection_builder/driver_registry/runtime, examples (`getting-started`, `mesh-demo`, `metaengine-quickstart/graph_demo`), engine profiles (pg/duckdb/mysql sqlite/turso/badger/dgraph), driver names. Delivered a source-verified walkthrough: Edge folds + EdgeRemoval (ADR-0114-style deletion) → `system.RawQuery` → `system.New` with dgraph engine in DeploymentConfig → projection host → `sys.MetaEngine().ExecuteCtx` traversal.
2. **Q2 — "Is that well documented?"**: Audited all doc surfaces by search: metaengine/README.md GOOD (fold→ADT table lines 199–200, capability helpers, ADR-0113), example graph demo GOOD (but raw Plan/Apply, no system composition), SKILL references/recipes/COOKBOOK ZERO modern-graph content, advanced.md §6.13 documents only the DEPRECATED `graph.GraphProjection`, system/README mentions dgraph in a driver list only. Verdict: primitives documented; the end-to-end system-composed story exists nowhere.
3. **Q3 — "Why not ALL engines by default + operator-owned DeploymentConfig (nix)?"**: Discovered the operator half ALREADY EXISTS (`system.LoadConfig`: YAML + `CQRS_*` env overrides, config_loader.go:59; demoed in `example/metaengine-quickstart/configfile_demo.go`; design set by ADR-0123 §3). Explained the three blockers for all-engines-by-default (dep closure incl. CGo for duckdbengine; no runtime driver loading in Go; linking ≠ operating a server). Proposed: opt-in `metaengine/allengines` module (pure-Go set, duckdb behind `//go:build cgo` per its own ADR-0086 pattern) + nix `-tags` per-deployment pattern + ADR for the two-level operator story.

---

## Brutal Self-Review

**1. What did you forget?**

- I **never compile-verified** the code walkthrough I presented in Q1. The repo HAS the exact harness for this (`cmd/doc-check` recipes compile verification) and I neither wrote the recipe nor ran the check. The snippets were assembled from verified pieces (graph_demo.go fold shape verbatim, getting-started.go system wiring verbatim, mesh-demo Events gate verbatim) but the COMBINATION was never compiled. One risk point: dgraph `DSN: "localhost:9080"` — I asserted the gRPC address format without reading `dgraphengine/register.go`'s DSN handling.
- In Q2 I counted modern-pattern keyword matches (0) but SKILL.md's read-model tier table DOES reference `metaengine/graphadapter` for edges-only reads. I said "zero mentions of the modern graph path," which is accurate for the pattern but a reader could take it as "zero pointers." Slightly overbroad phrasing.
- In Q3 I asserted "bigtableengine pulls the GCP SDK" from module naming + AGENTS.md dep-isolation context — I never opened its go.mod. Same for several dep-weight claims (binary size, vulncheck surface): asserted from principle, not measured.
- I left two concrete offers dangling (recipe + §6.13 rewrite; TODO_LIST entry + ADR draft) without recording them anywhere durable when the user moved on. They lived only in chat.

**2. What is stupid that we do anyway? (found, pre-existing)**

- `graphadapter` — the DESIGNATED v5 replacement for the deprecated graph tier — flattens every node to `Label: "entity", KeyProp: "id"` with `fmt.Sprint` keys (adapter.go:59-60, 78). The entire schema/label/property richness of the `graph` module (NodeRef labels, typed key props, Schema validation) is destroyed crossing the bridge. We deprecated a rich API in favor of a lossy one.
- `SKILL.md` names `metaengine/COOKBOOK.md` as THE copy-paste pattern source; COOKBOOK has ZERO graph content. A pointer promising coverage that does not exist.

**3. What could you have done better?**

- Write the Q1 walkthrough AS a recipes.md fence immediately (the harness would have compile-checked my claims for free) — answering and documenting in one step instead of answering first, offering second, doing never.
- Verify per-module dep claims with `go.mod` reads before asserting them in a design recommendation.
- State verification level per claim (verified-in-source / asserted-from-principle) in the original answers rather than only now.

**4. What could you still improve?**

- Turn this session's findings into the doc/code items in (f) — they are all concrete and scoped.
- For graphadapter: either extend `metaengine.Edge` with optional label/kind fields (v5-breaking decision) or document the flattening as a deliberate limitation with an escape hatch (direct `Driver()` access).

**5. Did you lie to you?**
No deliberate lies. Three claims were under-verified at time of assertion and are flagged above: dgraph DSN format, bigtableengine dep weight, overall binary-size impact of all-engines. Everything else cited file:line evidence.

**6. How can we be less stupid?**

- Rule for this repo: any "here's how you compose X" answer must go through the doc-check compile harness before being presented as a pattern. We built the gate; use it on ourselves.
- Dep-weight claims in design discussions require `go.mod` evidence, the same way error-taxonomy claims require the docs gate.

**7. Ghost systems?**

- **`metaengine/graphadapter` is a near-ghost**: no example composes it, no recipe documents it, no system integration references it. Its only footprints are deprecation notices in advanced.md and a modules.md row. Should it be integrated? YES — it is the designated v5 bridge; value = preserving the graph module's rich reads (Traverse/ShortestPath/Neighbors) for metaengine consumers. Integrating means: one example + one recipe + a label-fidelity decision.
- **Suspicion (unverified)**: advanced.md claims "Reads run native Cypher/Gremlin via the driver (only MemoryDriver offers a Go-native read API)" — I found NO non-MemoryDriver GraphDriver implementation in the repo. If none exists, the Cypher/Gremlin sentence documents an aspiration as if it were capability. Needs one grep to confirm before fixing the doc.

**8. Scope creep trap?**
Session stayed inside the three questions. The temptation to launch a full graph-module audit (schema validation coverage, per-driver matrix) was resisted; items go to (f) instead.

**9. Did we remove something useful?**
No removals — read-only session. Nothing reverted.

**10. Split brains?**

- **Graph ADT split brain (pre-existing, confirmed this session)**: the graph concept lives in TWO type systems — `metaengine.Edge{From,To any}` (planner ADT) and `graph.NodeRef/EdgeRef` (labels, key props) — bridged by graphadapter's lossy conversion. Two models of "what a graph edge is" that will drift; the adapter already silently drops half of one side.
- Minor: my Q1 answer vs advanced.md §6.13 now constitute two competing "how to do graph" stories (modern vs deprecated-but-documented) until §6.13 is rewritten — a doc split brain created by documentation lag, not by this session's code.

**11. Tests?**
Nothing to test — zero code changed. The session's equivalent quality gate (compile-verified docs) was available and not used; see (3).

---

## a) FULLY DONE

| Item                                                                    | Evidence                                                                                                                                                                                                                                                                                                                         |
| ----------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Q1 graph-native walkthrough delivered, source-verified piece by piece   | fold shape verbatim from `example/metaengine-quickstart/graph_demo.go:38-47`; system wiring verbatim from `example/getting-started/main.go:129-167`; Events gate from `example/mesh-demo/gate.go:55-63`; undirected flag from `metaengine/reflect.go:225-229`; driver names verified in `register.go` files ("sqlite", "dgraph") |
| Q2 documentation audit with per-surface counts                          | metaengine/README.md:199-200 (fold→ADT table), :579-580 (capability helpers); recipes.md 0 graph recipes of 86; COOKBOOK.md 0; advanced.md §6.13 deprecated-only; system/README.md:237 driver-list only                                                                                                                          |
| Q3 design consult incl. discovery that the operator half already exists | `system/config_loader.go:59` LoadConfig (YAML + CQRS_ env); `configfile_demo.go` boots from cqrs.yaml; ADR-0123 §3 blank-import design; ADR-0086/0071 CGo pattern for duckdb                                                                                                                                                     |
| Confirmed no existing `allengines`/engine-set mechanism                 | searched TODO_LIST.md, FEATURES.md, docs/adr/* — zero hits                                                                                                                                                                                                                                                                       |

No commits, no file changes — evidence is the conversation record itself.

## b) PARTIALLY DONE

| Item                                  | What exists                                                                                                    | What remains                                                                   | Blocker                                                                                    | Effort    |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------ | --------- |
| Graph-native documentation closure    | Audit complete, exact gaps named, fix shape proposed (recipes §graph + §6.13 modern rewrite, compile-verified) | Nothing written                                                                | User decision pending (offered twice, no answer — user redirected to Q3, then this report) | M         |
| allengines + two-level operator story | Design reasoning complete; existing mechanisms mapped (LoadConfig, cgo build-tag precedent)                    | TODO_LIST entry, ADR draft, module itself                                      | Product decision: v4.x additive vs v5-first; engine set membership (bigtable in/out)       | S / M / M |
| Claim verification discipline         | All under-verified claims now flagged (this report)                                                            | dgraph DSN format, bigtableengine go.mod, binary-size delta — 3 facts to check | None — one grep/view each                                                                  | S         |

## c) NOT STARTED

| Item                                                                       | Why not started                             | Still wanted?                                    |
| -------------------------------------------------------------------------- | ------------------------------------------- | ------------------------------------------------ |
| `example/graph-native/` — system.New + follow graph + systemscenario BDD   | Awaiting user decision on docs shape        | High value: the missing end-to-end demonstration |
| graphadapter label-fidelity fix or documented limitation                   | Design decision (breaking surface) not made | Yes — it is the v5 bridge                        |
| `metaengine/allengines` module                                             | Awaiting product call                       | Yes (my recommendation)                          |
| ADR: two-level operator story (nix -tags compile-time + cqrs.yaml runtime) | Awaiting product call                       | Yes                                              |
| SKILL/COOKBOOK pointer reconciliation                                      | Discovered this session                     | Yes                                              |
| Cypher/Gremlin docs-claim verification                                     | Discovered this session, one grep away      | Yes                                              |

## d) TOTALLY FUCKED UP (found; pre-existing — this session broke nothing)

| Item                                                                                                                                                                                                                                  | Severity                                                   | Root cause                                                                              | Mitigation                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **graphadapter destroys node labels** — every node becomes `Label:"entity"`, keys via `fmt.Sprint`; the designated v5 bridge loses the graph module's core richness (adapter.go:59-60,78,99-100)                                      | High (v5 API surface)                                      | `metaengine.Edge` carries only `From/To any`; adapter had nothing to map labels from    | Decide: extend Edge with optional label/kind (v5 breaking) OR document limitation + `Driver()` escape hatch |
| **The modern graph-native story is undocumented end-to-end** — a consumer asking exactly this session's Q1 must synthesize 3 sources; recipes.md/COOKBOOK have ZERO graph recipes; advanced.md §6.13 teaches the DEPRECATED API first | High (adoption friction for the strategic metaengine path) | Docs followed the legacy tier; the system-composition rewrite never got a graph chapter | Section (f) items 1–8                                                                                       |
| **§6.13's deprecation pointer aims at a lossy bridge** — "replacement is metaengine/graphadapter over the metaengine Graph ADT" steers v5 migrators into label flattening                                                             | Medium-High                                                | Same Edge-shape root cause as row 1                                                     | Same decision as row 1                                                                                      |
| **SKILL.md → COOKBOOK.md pointer promises graph patterns that don't exist** (SKILL.md:18 names COOKBOOK as copy-paste source; zero graph content)                                                                                     | Medium                                                     | COOKBOOK predates the graph ADT surface                                                 | Add graph chapter to COOKBOOK or fix the pointer                                                            |
| **Suspected docs overclaim: "reads run native Cypher/Gremlin via the driver"** — no non-MemoryDriver GraphDriver found in repo                                                                                                        | Medium if confirmed (docs lie)                             | advanced.md §6.13 describes portability targets as if shipped                           | One grep; fix sentence or mark ROADMAP                                                                      |
| My Q1 snippets were presented without compile verification                                                                                                                                                                            | Low (nothing shipped)                                      | Process miss — harness existed, unused                                                  | Re-issue as recipes.md fences through doc-check                                                             |

## e) WHAT WE SHOULD IMPROVE

| Pattern/practice                                                                              | Impact                                                           | Concrete fix                                                                                                                                     |
| --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Answers-as-patterns skip the doc-check compile harness                                        | Undetected API drift in advice; consumer copy-paste breakage     | House rule: composition walkthroughs get written as recipes.md fences (harness compiles them) in the same session                                |
| Dep-weight assertions without go.mod evidence                                                 | Design decisions on folklore                                     | Require go.mod/`go mod graph` citation for "X pulls Y" claims in ADRs                                                                            |
| Designated-replacement modules ship without a consumer (graphadapter near-ghost)              | v5 bridge rots untested; label flattening found only by accident | Gate "deprecated, replaced by X" notices on X having one example + one recipe                                                                    |
| Dangling session offers live only in chat                                                     | Work items evaporate on session end                              | Record offered work in TODO_LIST immediately at offer time (HARVEST discipline)                                                                  |
| Two type systems for graph edges (metaengine.Edge vs graph.NodeRef/EdgeRef) with lossy bridge | Silent data-model flattening at the boundary; drift              | Resolve with the label decision; if Edge stays flat, document the contract ("labels are a graph-module-only concern") explicitly in both modules |

## f) Next Tasks (session-derived; ordered by impact)

| #  | Task                                                                                                                                                            | Impact | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1  | Add recipes.md §graph: Edge folds + EdgeRemoval + traversal query through `system.RawQuery` + DeploymentConfig dgraph/sqlite — compile-verified via doc-check   | High   | M      | Documentation |
| 2  | Rewrite advanced.md §6.13: modern metaengine path first, `GraphProjection` demoted to migration note                                                            | High   | S      | Documentation |
| 3  | Decide Edge label strategy (extend `metaengine.Edge` with optional label/kind vs document flattening + `Driver()` escape hatch) — unblocks items 9–11           | High   | M      | Feature       |
| 4  | Add TODO_LIST entry for `metaengine/allengines` decision                                                                                                        | High   | S      | Process       |
| 5  | Draft ADR: two-level operator story (nix `-tags` compile-time engine set + cqrs.yaml runtime config, LoadConfig as the runtime half)                            | High   | M      | Documentation |
| 6  | Create `example/graph-native/` — system.New + follow graph + systemscenario BDD tests                                                                           | High   | M      | Feature       |
| 7  | Verify whether any non-MemoryDriver GraphDriver exists; fix or ROADMAP-mark the Cypher/Gremlin sentence in §6.13                                                | High   | S      | Documentation |
| 8  | Add graph chapter to metaengine/COOKBOOK.md (or fix SKILL.md pointer if COOKBOOK is the wrong home)                                                             | Medium | S      | Documentation |
| 9  | graphadapter: preserve node labels (collection-scoped label config or Edge labels, per item 3 decision)                                                         | High   | M      | Feature       |
| 10 | graphadapter: tests pinning label fidelity + undirected-capability reporting behavior                                                                           | Medium | S      | Quality       |
| 11 | graphadapter: de-ghost — compose it in one example (e.g. alongside item 6)                                                                                      | Medium | S      | Feature       |
| 12 | Create `metaengine/allengines` module: one blank import for the pure-Go engine set + dep budget review                                                          | Medium | M      | Feature       |
| 13 | allengines: keep duckdb out or behind `//go:build cgo` (ADR-0086 pattern); document choice                                                                      | Medium | S      | Feature       |
| 14 | Verify bigtableengine go.mod (GCP SDK weight) — evidence for include/exclude in allengines                                                                      | Medium | S      | Quality       |
| 15 | Measure allengines binary size + dep count vs single-engine build (ADR evidence)                                                                                | Medium | S      | Quality       |
| 16 | Document the nix per-deployment engine-tag pattern (flake profiles passing `-tags engines.X`)                                                                   | Medium | M      | Documentation |
| 17 | system/README: add graph-native section (engine matrix incl. dgraph native traversal, CTE engines, BFS fallback)                                                | Medium | S      | Documentation |
| 18 | recipes.md: operator story for `system.LoadConfig` (cqrs.yaml + CQRS_ env) — check for existing coverage first, add if missing                                  | Medium | S      | Documentation |
| 19 | readmodels.md tier table: promote the metaengine Graph ADT row with per-engine traversal capabilities                                                           | Medium | S      | Documentation |
| 20 | FAQ entry: "How do I model relations/graphs?" → Edge folds + EdgeRemoval + engine matrix                                                                        | Low    | S      | Documentation |
| 21 | Document the `ExecuteCtx` traversal return-type contract ([]any regardless of R type parameter — the runtime cast)                                              | Medium | S      | Documentation |
| 22 | Check `ExecuteTypedByName` (modules.md:105) coverage for traversal queries; document if it gives typed results                                                  | Medium | S      | Documentation |
| 23 | systemscenario recipe: asserting graph projections (poll-mode ThenQuery over traversal inputs)                                                                  | Medium | M      | Documentation |
| 24 | modules.md `graph` row: state explicitly that only MemoryDriver ships; Cypher/Gremlin is portability target (pending item 7 outcome)                            | Medium | S      | Documentation |
| 25 | SKILL.md: add graph pointer (readmodels tier table already points at graphadapter; add the modern recipe link once item 1 lands)                                | Medium | S      | Documentation |
| 26 | graphadapter doc.go: document the label-flattening behavior explicitly pending item 3                                                                           | Medium | S      | Documentation |
| 27 | Engine matrix table (one table, all engines × ADTGraph support: native CTE / native graph DB / BFS fallback / undirected) — currently scattered across profiles | Medium | S      | Documentation |
| 28 | root README: graph-native mention in the "deployment-time storage choice" bullet (currently lists engines but not the graph ADT story)                          | Low    | S      | Documentation |
| 29 | If Edge grows labels (item 3): update dgraphengine/badgerengine GraphAddEdge signatures and conformance tests                                                   | High   | M      | Feature       |
| 30 | If Edge grows labels: extend `adttest` graph conformance matrix with label round-trip assertions                                                                | Medium | M      | Quality       |
| 31 | dgraphengine register.go: verify DSN/address handling; document the DSN format in EngineConfig docs                                                             | Low    | S      | Documentation |
| 32 | Add md-go/doc-check fences for any graph snippets added by items 1/8 (parse + compile gates)                                                                    | Medium | S      | Quality       |
| 33 | Consider `Undirected` support matrix doc (badger + dgraph native; others report missing capability per graphadapter comment)                                    | Low    | S      | Documentation |
| 34 | HARVEST this report's section (f) into TODO_LIST.md / ROADMAP.md once user confirms                                                                             | High   | S      | Process       |
| 35 | Annotate this report when items close (docs-health ANNOTATE discipline)                                                                                         | Low    | S      | Process       |

(35 items — all session-derived. Padding to 50 would require researching unrelated areas, which the user explicitly forbade. Items 12–16 and 29–30 are ROADMAP-grade pending the item-3/4 decisions.)

## g) Questions I Cannot Answer Myself

1. **Graph docs shape (blocks f#1/2/6):** Close the doc gap as recipes-only, or also build `example/graph-native/` with systemscenario tests? Both is ~a day; recipes-only is ~an hour. What's the budget/appetite?
2. **allengines timing & membership (blocks f#4/5/12):** Should `metaengine/allengines` land as a v4.x additive module now, or be parked for v5? And must bigtableengine (GCP SDK weight, unverified) be in the default set, or pure-Go engines only?
3. **graphadapter label flattening (blocks f#3/9/29):** Is `Label:"entity"` flattening an acceptable v5 limitation to document, or must `metaengine.Edge` grow optional label/kind fields (breaking engine interface signatures) before v5? This is a product-intent call — the graph module's Schema/labels were an ADR-0039 feature, so silently dropping them feels wrong, but extending Edge touches every engine's GraphAddEdge.

---

_Point-in-time snapshot. Stale by design. ANNOTATE, never rewrite. Auto-commit daemon will absorb this file; no manual commit per harness contract._

---

## Appendix — dispatch/closure record (added 2026-10-10, execution session)

Non-destructive closure note; the snapshot above is untouched. Disposition of
this report's questions by the executed plan
([V2](../planning/2026-10-10_06-11_SUPERB-graph-native-adoption-closure-and-operator-story-V2.md),
waves 0–5 all complete):

- **§g Q1 (graph demand / A-R1):** default carried — full waves executed;
  graph-native is now ADOPT-IF-NEEDED (TODO_LIST marker flipped to DONE with
  this framing). No fleet consumer requires it today.
- **§g Q2 (graphadapter label flattening):** RULED as documented limitation —
  flat node identity + label-less edges both pinned in
  [ADR-0156](../adr/0156-graph-edge-labels-at-v5.md) (v4.x: collection-per-relation
  + `Driver()` hatch; labeled edges reopen only on a named fleet consumer).
  `Edge` does NOT grow fields in v4.x.
- **allengines (D3):** RULED NO convenience module — measured 5.3× binary tax
  (68.2 MiB vs 12.7 MiB sqlite-only, 221 vs 64 modules); the two-level story is
  [ADR-0157](../adr/0157-engine-fleet-operation-two-level-story.md) (blank-import
  candidates at compile time, cqrs.yaml routing at runtime).
- **§f docs fixes:** all dispatched via V2 waves — recipes §2.44 (+ write-side
  fence, depth semantics), advanced.md §6.13 modern-first rewrite, COOKBOOK
  graph chapter, graphadapter README (new), system/README graph + declaration
  sections, FAQ entries. T19 consumer dry-run GREEN (2 iterations).
- **§e/§f pre-existing red (core/v5 links, cqrs-bench, golangci, etc.):** NOT
  this wave's scope — left to the owning sessions, tracked in the 07-37 report §f.
