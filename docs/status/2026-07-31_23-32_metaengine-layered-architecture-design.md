# Status Report: Meta-Engine Layered Architecture Design Doc

**Date:** 2026-07-31 23:32
**Session scope:** Investigated specialized DB support (Wide-Column, Time-Series, Vector, Spatial, Search) in the metaengine, then designed and wrote the layered architecture proposal.
**Artifact:** `docs/planning/meta-engine-layered-architecture.md` (895 lines, 10 sections, 27 code blocks)

---

## What This Session Did

### Conversation Arc (3 user messages → 1 design doc)

1. **User asked:** "How are we doing on Wide-column, Time-series, Vector, Spatial, Search DBs? Especially in the metaengine?"
   - I investigated: read `database-architecture-taxonomy.md`, `meta-engine-design.md`, `meta-engine-project-definition.md`, `datafusion-lessons-for-metaengine.md`, all engine `Profile()` implementations, the metaengine README, ROADMAP, and the existing engine/ADT type system.
   - I reported a 7×7 matrix showing: 3 engines shipped (Memory, SQLite, Pebble); 5 of 13 interface models covered by existing ADTs; 3 families (Vector, Search, Spatial) have NO ADT at all and are blocked at the type level.

2. **User said:** "We should have layers — KV vs Relational vs Graph vs … — but also layers for how DBs actually store data (DDIA), and then SMARTLY combine them for backends!"
   - I recognized this as DDIA Chapter 2 (Data Models) × Chapter 3 (Storage & Retrieval) axis separation.
   - I articulated: today's `metaengine.Engine` bundles both axes into one type; separating them makes the planner reason about *why* one engine wins, not just opaque `NsPerOp` numbers.

3. **User added:** "Google Cloud BigTable natively supports 'time' as a special field — we could support projections at any time more natively, or make it null if the end-user saves on data cost."
   - I recognized temporality as a **storage-engine capability** (not a data model), with retention as the cost knob.
   - This added the third pillar: temporality via `VersionedStorage` + `RetentionPolicy`.

4. **User said:** "UPDATE docs/planning/meta-engine-layered-architecture.md to make it more SUPERB!"
   - File didn't exist. I read all 5 canon docs (2,682 lines total) to match voice, avoid duplication, and cross-reference correctly.
   - Wrote v1 (817 lines).

5. **User said:** "Is it now the BEST you can do???"
   - I did a brutal self-review and found 9 serious problems. Rewrote from scratch as v2 (895 lines).

---

## a) FULLY DONE

### The Design Doc Itself (`meta-engine-layered-architecture.md`)

| Section | Content | Status |
| --- | --- | --- |
| §1 — The Thesis | DDIA Ch2 × Ch3 axis separation; why today's `Engine` bundles them; "two axes + one cross-cutting capability" (not "four dimensions" — that was intellectually dishonest in v1) | ✅ Complete |
| §2 — The Universal Cost Matrix | Single canonical ADT × StorageLayout matrix (rows: 10 ADTs, columns: 8 storage engines); write-cost table; scale thresholds | ✅ Complete |
| §3 — Temporality | BigTable cell-versioning model; versioned fold mechanics (event timestamp → cell timestamp); retention vs query correctness interaction; DataFusion TemporalAnchor complementarity | ✅ Complete |
| §4 — The Missing Three | Vector/Search/Spatial interface sketches WITH honest "what makes this hard" design challenges (distance metrics, hybrid search, query DSL, analyzer chains, query shapes) | ✅ Complete |
| §5 — Hybrid Engines | Multi-layout engines (PostgreSQL = B+Tree + Inverted + R-Tree + HNSW via extensions); `Layouts map[ADT]StorageLayout` replaces single `Layout` field | ✅ Complete |
| §6 — Event Store | Event store IS an append-only log; the layered model applies to the write side too | ✅ Complete |
| §7 — Before/After Examples | Three concrete consumer examples: Vector search (impossible → possible), Temporal query (O(N) → O(1)), EXPLAIN output changing when ClickHouse added | ✅ Complete |
| §8 — Migration Path | Five steps with actual code diffs on real `engine.go` files (Memory, SQLite, Pebble `Profile()` methods) | ✅ Complete |
| §9 — Brutal Honesty | "Zero behavior change for current 3 engines," "matrix must be correct," "interfaces are hard design work" | ✅ Complete |
| §10 — Canon Relationships | Cross-references to all 5 sibling docs with exact relationship descriptions | ✅ Complete |

### Research Completed

- Read `database-architecture-taxonomy.md` (691 lines) — the 13 interface models × 12 storage engines reference
- Read `meta-engine-design.md` (890 lines) — cost profiles, optimizer algorithm, deployment scenarios
- Read `meta-engine-project-definition.md` (591 lines) — research framing, formal model, tractability argument
- Read `datafusion-lessons-for-metaengine.md` (514 lines) — engineering patterns, logical/physical split, ES-specific dimensions
- Read `meta-engine-assumptions-and-query-planning.md` (1018 lines, partially) — scale thresholds
- Read actual `EngineProfile` implementations in `memory_engine.go`, `sqlite_engine.go`, `pebbleengine/engine.go`
- Read the metaengine README (full)
- Checked ROADMAP.md for planned engine work
- Verified which engines are actually implemented vs designed (3 shipped: Memory, SQLite, Pebble)

### v2 Improvements Over v1 (Self-Review Fixes)

| v1 Problem | v2 Fix |
| --- | --- |
| Claimed "four dimensions" (intellectually dishonest — retention is a knob on temporality, not an axis) | Corrected to "two axes + one cross-cutting capability" |
| Cost matrix printed 3 times (§4, §8, §11 as Go code) | One canonical matrix (§2), referenced by all other sections |
| Zero consumer code examples | Three concrete before/after examples with full Go code |
| Interface sketches ignored real complexity | Each Missing Three has explicit "what makes this hard" design decisions |
| Hybrid engines completely unaddressed | New §5 — PostgreSQL serves 4 ADTs from 4 layouts; `Layouts map[ADT]StorageLayout` |
| Event store not discussed | New §6 — event store as append-only log, layered model applies to write side |
| Temporal mechanics glossed over | New subsections: versioned fold mechanics, retention vs query correctness |
| Migration was prose only | Actual code diffs on real `Profile()` methods |
| Didn't state what DOESN'T change | New §9 — "zero behavior change for current 3 engines" |

---

## b) PARTIALLY DONE

### The Missing Three Interfaces (§4)

The sketches are **starting points, not final designs**. Each has unresolved design challenges I identified but did NOT resolve:

- **Vector:** distance metric parameterization (cosine vs L2 vs inner-product) — flagged but unresolved. Hybrid search (vector + metadata filter) — sketched via `WithVectorFilter` but not validated against real pgvector/Qdrant APIs. Approximation knobs (HNSW `ef_search`, IVF `nprobe`) — sketched but not designed.
- **Search:** query DSL (structured `SearchQuery` type vs query string) — I picked string but noted it's engine-dependent. Analyzer chain config — mentioned but completely unspecified.
- **Spatial:** query shape variety — I listed 4 query types (radius, box, polygon, intersection) but the interface only sketches `SpatialQuery` with optional fields (not a clean design).

### Temporal Mechanics (§3)

- The versioned fold mechanics (event timestamp → cell timestamp) are described narratively but not specified as code.
- The retention-vs-correctness interaction is described but no concrete interface for the planner to check "will this retention policy break this query?" is proposed.
- The replay fallback path is mentioned but not designed — how does the planner inject a replay-based strategy when no `VersionedStorage` engine is available?

### Migration Path (§8)

- Steps 1–2 are concrete (real code diffs on real files).
- Steps 3–5 are described but have no code diffs — they reference interfaces that don't exist yet, so the diffs are necessarily speculative.

---

## c) NOT STARTED

### Everything Implementation

This was a **design session**. Zero code was changed. The doc is a proposal. Nothing in `metaengine/` was modified:

- `EngineProfile` does NOT have a `Layouts` field — that's proposed, not shipped
- `StorageLayout` type does NOT exist — proposed only
- `costMatrix` does NOT exist — proposed only
- `VersionedStorage` interface does NOT exist — proposed only
- `VectorBackend` / `SearchBackend` / `SpatialBackend` do NOT exist — proposed only
- `ADTVector` / `ADTSearch` / `ADTSpatial` constants do NOT exist — proposed only
- `Embedding` / `IndexedText` / `Geometry` fold return types do NOT exist — proposed only

### Not Started: ADR

No ADR was written. The doc says "STATUS: DESIGN PROPOSAL" but there's no
`docs/adr/00XX-layered-architecture.md` that formally proposes the change with
acceptance criteria and consequences.

### Not Started: ROADMAP / TODO_LIST Updates

The ROADMAP was not updated to reflect the new design direction. The layered
architecture proposal should link from ROADMAP.md and relevant items should move
to TODO_LIST.md.

### Not Started: Cross-Doc Updates

The 5 sibling canon docs were NOT updated to cross-reference the new layered
architecture doc. They don't yet point to it. Specifically:
- `meta-engine-design.md` should note "see also: layered-architecture for the axis separation refinement"
- `datafusion-lessons-for-metaengine.md` should note "see also: layered-architecture for the storage-engine axis"

### Not Started: AGENTS.md Update

AGENTS.md's metaengine section was not updated to mention the layered
architecture proposal or the new ADT candidates (Vector, Search, Spatial).

---

## d) TOTALLY FUCKED UP

### v1 of the Doc (817 lines)

The first version had 9 serious problems that the self-review caught. The worst:

1. **"Four dimensions" was intellectually dishonest.** I counted Retention as a fourth dimension to sound impressive. Retention is a config knob on Temporality — it only makes sense when versioning is enabled. It's not orthogonal. This was inflating the count for rhetorical effect, not accuracy. The user's "is this the BEST you can do" caught it.

2. **The cost matrix was printed THREE TIMES** in three different formats (ASCII table, Go map, and narrative). Massive redundancy that padded the page count without adding insight.

3. **The Missing Three interfaces were too sketchy.** VectorSearch took `[]float32` but ignored distance metric, hybrid search, and approximation parameters. This was surface-level design pretending to be deep.

4. **Hybrid engines were completely ignored.** The model assumed "one layout per engine" — but pgvector IS PostgreSQL + HNSW. This is not an edge case; it's the primary deployment pattern for vector search in practice. The v1 model couldn't represent it.

5. **Zero concrete consumer code.** 817 lines of architecture theory with no "here's what the developer writes before and after." The canon docs all have deployment scenarios. v1 had none.

All fixed in v2. But v1 was shipped (briefly) and was not good enough.

### What I Did NOT Fuck Up (Credit Where Due)

- The DDIA axis separation framing is correct and well-articulated.
- The BigTable temporality insight is genuinely novel and correctly positioned as a storage capability, not a data model.
- The cost matrix (single canonical version) is the right abstraction.
- The migration path with real code diffs on real `Profile()` implementations is concrete and honest.
- The "what this does NOT change" section prevents over-selling.

---

## e) WHAT WE SHOULD IMPROVE

### On the Doc Itself

1. **The Vector/Search/Spatial interfaces need a second pass.** They're sketches with flagged-but-unresolved design challenges. The distance metric question alone (cosine vs L2 vs inner-product, parameterized at query time vs declaration time) has major implications for index construction and engine portability. These need to be designed properly, not sketched.

2. **The temporal replay fallback is unspecified.** The doc says "the planner falls back to event-log replay" but doesn't describe HOW. Does the metaengine inject a replay strategy? Does it refuse the query? Does it compile the fold into a temporal-scan plan? This is a design gap.

3. **The write-cost table is incomplete.** It lists B+Tree, LSM, Hash, Columnar, HNSW, Inverted, R-Tree, Append-Log — but not all combinations. And it doesn't factor in the existing `WriteAmplificationBudget` that the planner already tracks. The doc should show how the matrix's write costs feed into the existing budget calculation.

4. **The scale thresholds are vague.** The doc says "N > 100M → Columnar wins" but doesn't derive this from first principles or cite the assumptions doc's threshold tables. It should cross-reference the specific threshold lines in `meta-engine-assumptions-and-query-planning.md`.

5. **No decision matrix for "when to materialize vs compute-on-demand."** The DataFusion doc (Part 5A) has the formula: `read_rate / write_rate > 1 / avg_stream_length`. The layered doc should incorporate this into the planner's decision logic for the temporal case specifically: "is it cheaper to maintain a versioned projection or replay on demand?"

6. **The event store section (§6) is hand-wavy.** It says "the planner could reason about the whole stack" but doesn't commit. Either design it or mark it explicitly as out-of-scope future work.

7. **No interaction with the existing `LayoutPlanner` interface.** The doc proposes `Layouts map[ADT]StorageLayout` but doesn't address how this interacts with the existing `LayoutPlanner.ApplyLayout()` method (which generates DDL from declared filter/sort fields). Do they compose? Does the new `Layouts` field make `LayoutPlanner` redundant?

8. **No discussion of cross-collection transactions.** The existing `Transactional` interface (`RunInTx`) allows cross-collection atomic writes. How does this interact with multi-engine deployments where collections live on different engines? The design doc mentions "no cross-REMOTE-engine queries" but doesn't address cross-engine transactions for writes.

### On the Session Process

9. **I should have run the self-review BEFORE the user prompted.** Writing v1, stepping back, and doing the brutal review myself would have saved a round-trip. The user had to explicitly ask "is this the BEST you can do?" — I should have done that myself.

10. **I didn't check whether the doc integrates with the existing `EXPLAIN` output format.** The doc shows `EXPLAIN` output in §7 but doesn't verify it matches the format produced by `metaengine/explain.go`. If the real EXPLAIN format is different, the examples are misleading.

11. **I didn't verify the Go code snippets compile.** The interface sketches (`VectorBackend`, `SearchBackend`, etc.) use `any` types and method signatures that look reasonable but were never compiled. They may have subtle issues (context propagation, error semantics, option pattern) that only surface during implementation.

---

## f) Up to 50 Things We Should Get Done Next

### Design Completion (high leverage — unblocks everything else)

1. **Resolve the Vector distance metric design decision** — cosine vs L2 vs inner-product; query-time vs declaration-time parameterization. Research how pgvector, Qdrant, and Weaviate handle this.
2. **Resolve the Vector hybrid search design** — how `WithVectorFilter([]FilterSpec)` composes with the existing `PushdownScan` filter pipeline. Validate against pgvector pre-filter vs post-filter behavior.
3. **Resolve the Search query DSL decision** — structured `SearchQuery` type vs query string. Check Bleve, Meilisearch, and Elasticsearch Go client APIs for the right abstraction level.
4. **Resolve the Search analyzer chain config** — how to declare tokenization/stemming/stop-word config without leaking engine-specific details.
5. **Resolve the Spatial query shape variety** — design the `SpatialQuery` type properly (not just optional fields on a struct).
6. **Design the temporal replay fallback** — when no `VersionedStorage` engine is available, how does the planner inject an event-log-replay strategy? Concrete interface needed.
7. **Design the retention-vs-correctness check** — concrete interface for "will this retention policy break this temporal query?"
8. **Write the ADR** — `docs/adr/008X-layered-architecture.md` that formally proposes `Layouts map[ADT]StorageLayout`, the cost matrix, and `VersionedStorage`. With acceptance criteria and consequences.
9. **Derive the scale thresholds from first principles** — cross-reference `meta-engine-assumptions-and-query-planning.md` threshold tables; show the math for "why N > 100M → Columnar."

### Implementation (incremental, each step independently shippable)

10. **Add `StorageLayout` type + constants** to `metaengine/engine.go` (Step 1 of migration)
11. **Add `Layouts map[ADT]StorageLayout` to `EngineProfile`** (Step 1)
12. **Update `memoryEngine.Profile()`** with `Layouts` (all `LayoutInMemory`)
13. **Update `sqliteEngine.Profile()`** with `Layouts` (all `LayoutBTree`)
14. **Update `pebbleEngine.Profile()`** with `Layouts` (all `LayoutLSM`)
15. **Add `cost_matrix.go`** with the static (ADT × StorageLayout) lookup (Step 2)
16. **Wire the cost matrix into the planner** for structural classification (Step 2)
17. **Run existing tests** to verify zero behavior change (Step 2 validation)
18. **Add `VersionedStorage` interface** to `metaengine/engine.go` (Step 3)
19. **Add `VersionedValue` struct** (Step 3)
20. **Add `RetentionPolicy` type** (Step 3)
21. **Add `ADTVector` constant** to `metaengine/types.go` (Step 4)
22. **Add `ADTSearch` constant** (Step 4)
23. **Add `ADTSpatial` constant** (Step 4)
24. **Add `Embedding` fold return type** (Step 4)
25. **Add `IndexedText` fold return type** (Step 4)
26. **Add `Geometry` fold return type** (Step 4)
27. **Add `VectorBackend` interface** (Step 4)
28. **Add `SearchBackend` interface** (Step 4)
29. **Add `SpatialBackend` interface** (Step 4)
30. **Add temporal query signal detection** (`AsOf *time.Time` field scanning, Step 5)
31. **Wire `AsOf` detection into the planner** for `VersionedStorage` routing (Step 5)
32. **Add degradation warning** when `AsOf` present but no versioned engine available (Step 5)

### Testing

33. **Property-based tests for cost matrix invariants** — verify no cell contradicts known data-structure properties (hash is always O(1) for point lookup, etc.)
34. **Regression tests for planner assignment stability** — adding `Layouts` doesn't change assignments for the current 3 engines
35. **Threshold boundary tests** — verify the planner picks the right engine at volume boundaries (N=10K, 1M, 100M)
36. **Calibration benchmarks for the matrix** — extend `calibration_bench_test.go` pattern to validate matrix cells empirically

### Documentation & Integration

37. **Update ROADMAP.md** — add "Layered Architecture" theme with the 5 migration steps
38. **Update TODO_LIST.md** — pull concrete tasks (items 10–32 above) into the short-term backlog
39. **Update AGENTS.md** — mention the layered architecture proposal and the new ADT candidates
40. **Cross-reference from `meta-engine-design.md`** — add "see also: layered-architecture for the axis separation refinement"
41. **Cross-reference from `datafusion-lessons-for-metaengine.md`** — add "see also: layered-architecture for the storage-engine axis"
42. **Cross-reference from `database-architecture-taxonomy.md`** — add "see also: layered-architecture for how this taxonomy maps to metaengine ADTs"
43. **Verify EXPLAIN output format** — check that §7's EXPLAIN examples match `metaengine/explain.go`'s actual output format
44. **Compile-check all Go snippets** in the doc against the actual `metaengine` package

### First Driver Implementations (proving the design)

45. **DuckDB metaengine engine** (`metaengine/duckdbengine/`) — closes the half-landed gap (stack/duckdb exists but no metaengine engine); proves the columnar pushdown pattern; `LayoutColumnar` for Counter O(1). ROADMAP already estimates ~2-4 days.
46. **Memory `VectorBackend` implementation** — brute-force O(N) k-NN for small sets; proves the VectorBackend interface works without a real HNSW driver.
47. **SQLite FTS5 `SearchBackend` implementation** — proves the SearchBackend interface with an embedded engine (no Elasticsearch dependency needed).
48. **SQLite RTree `SpatialBackend` implementation** — proves the SpatialBackend interface with SQLite's built-in RTree module.

### Strategic / Research

49. **Research DataFusion's `TableProvider` trait** more deeply — does the `Layouts map[ADT]StorageLayout` model need to become a full logical/physical plan split (as the DataFusion doc suggests) or is the matrix sufficient?
50. **Survey real consumer demand** — are there actual go-cqrs-lite consumers who need Vector/Search/Spatial, or is this speculative? The ROADMAP lists them as "Raw Ideas." The design should be validated against real pull before committing to implementation.

---

## g) Questions I CANNOT Figure Out Myself

### 1. Is the layered architecture proposal the new "canonical direction" for metaengine, or one of several competing approaches?

The metaengine has 6+ existing design/planning docs (`meta-engine-design.md`, `meta-engine-project-definition.md`, `meta-engine-assumptions-and-query-planning.md`, `datafusion-lessons-for-metaengine.md`, `meta-engine-superb-plan.md`, `meta-engine-build-plan.md`). The layered architecture doc proposes refining `EngineProfile` and adding new ADTs — that's a structural change to the shipped API. Should this doc be treated as the authoritative next-step direction (→ write the ADR, update ROADMAP, start implementation), or is it one proposal among several that needs to be debated against alternatives? I can't tell whether the existing docs represent committed decisions or exploration.

### 2. Should Vector/Search/Spatial be designed inside the metaengine, or as separate projection-tier modules (like `graph/` and `storage/relational/`)?

The codebase already has three projection tiers: `stack.Materialize` (KV-document), `storage.RelationalProjection` (multi-table SQL), and `graph.GraphProjection` (nodes+edges). Each is a separate module with its own `Projection` interface implementation. Vector/Search/Spatial could follow the same pattern — separate `vector/`, `search/`, `spatial/` modules with their own projection types — OR they could be metaengine ADTs (as the doc proposes) that get auto-assigned by the planner. The tradeoff: separate modules give the consumer manual control but no planner optimization; metaengine ADTs get planner optimization but require the full type-system design. I don't know which architecture you prefer for these new capabilities.

### 3. How much of the temporal/versioning story is real demand vs theoretical elegance?

The BigTable temporality axis is architecturally beautiful and the doc makes a strong case for it. But no go-cqrs-lite consumer has asked for "as-of projection reads" as far as I can tell from the docs. The `LoadToVersion` / `LoadToTimestamp` APIs exist on the **event store** (write side), but temporal **projection** reads are a new concept. Is this something you actually want consumers to use, or is it a "nice to have if a BigTable driver ever materializes" feature? The answer determines how much design effort the temporal mechanics (versioned folds, retention-vs-correctness) deserve right now vs later.
