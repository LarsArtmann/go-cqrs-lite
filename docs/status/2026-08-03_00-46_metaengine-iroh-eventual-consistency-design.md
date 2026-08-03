# Status: Metaengine Eventual Consistency & Iroh Design Session

> **Date:** 2026-08-03 00:46
> **Session scope:** Iroh integration analysis → eventual consistency design → visibility model implementation → naming crisis → model redesign
> **Verdict:** ~~Design exploration valuable; implementation half-committed then proven wrong; needs full rewrite~~ **Rewritten.** The wrong `Visibility` model was replaced by `Replication` (`72818e88`, ADR-0093). Phase 2 polish (`f25e1d21`), Universal ADT Phase 3 (`8b41f658`, ADR-0094), and Iroh evaluation (ADR-0096) all completed.

---

## Context

The session started with a question: "How could this project benefit from [Iroh](https://github.com/n0-computer/iroh)?" This led through three stages:

1. **Iroh analysis** — mapped Iroh's capabilities (P2P QUIC, iroh-docs CRDT KV store, iroh-blobs content-addressed storage) onto the go-cqrs-lite architecture
2. **Eventual consistency reframing** — realized all CQRS read models are already eventual (projection lag), so the planner should model this honestly instead of pretending engines are "strong"
3. **Implementation** — started coding a "visibility model" that was progressively revealed to be the wrong abstraction through Socratic questioning

---

## a) FULLY DONE

### Design document

- **File:** `docs/planning/meta-engine-eventual-consistency-and-iroh.md` (committed in `31f26b8c`)
- Covers: the "all read models are eventual" insight, visibility vs consistency, CALM theorem connection, Iroh backend mapping, PN-Counter killer feature, Level 1 (standalone engine) vs Level 2 (replication wrapper), materialize-vs-replay cost model update, CGo/Rust bridging options
- **Caveat:** The document uses the `Visibility` naming that was subsequently rejected. It needs updating to the final `Replication` model once we settle the naming.

### Iroh + metaengine fit analysis

- Identified that iroh-docs maps onto 5 of 10 ADT backends (Map, Set, Counter, Multimap, Log)
- Identified that the Counter ADT (PN-Counter) is Iroh's killer feature — conflict-free distributed counting without coordination
- Identified two integration levels: standalone engine (limited) vs replication wrapper (strategic, recommended)
- Identified the Rust/Go bridge challenge (CGo FFI matches the stack/duckdb precedent)

---

## b) PARTIALLY DONE

### Core EngineProfile changes

- **Committed in `31f26b8c`** to `metaengine/engine.go`: Added `Visibility VisibilityModel` and `TypicalLag time.Duration` fields
- **PROBLEM:** The naming is wrong. Through discussion, we evolved from `Visibility` → `Replication` + `NetworkRTT` + `ReplicationLag`. The committed code uses the rejected naming.

### QueryConfig changes

- **Committed in `31f26b8c`** to `metaengine/query.go`: Added `visibility VisibilityModel` field
- **PROBLEM:** This field should NOT EXIST AT ALL. Through discussion, we determined that replication topology is an engine property, not a query property. Queries declare what to compute, not where data lives. This was a fundamental design error.

### Cost estimator changes

- **Committed in `31f26b8c`** to `metaengine/cost.go`: Added `estimateCostWithLag` function
- **PARTIALLY VALID:** The concept of lag as an additive cost component is correct. But the implementation conflates replication lag and network RTT into one "lag" value, which are orthogonal (lag scales with replication mode; RTT is a fixed per-query overhead).

### New file: visibility.go

- **Committed in `31f26b8c`** to `metaengine/visibility.go`: Defines `VisibilityModel`, `VisibilityLocal`, `VisibilityGlobal`, `EffectiveVisibility()`, `EffectiveTypicalLag()`
- **PROBLEM:** The entire file uses the rejected naming and concept. Needs to be rewritten as `replication.go` with the correct model.

---

## c) NOT STARTED

- The correct `Replication` type definition (replacing `VisibilityModel`)
- `NetworkRTT` field on EngineProfile (never written)
- Updating the 5 engine Profile() constructors (Memory, SQLite, Pebble, DuckDB, Postgres) with correct fields
- The planner routing/filtering logic (never written — `planQuery` in planner.go was never touched)
- The cost estimator integration (estimateCostWithLag exists but is never called by the planner)
- `WithVisibility` / `WithReplicationMode` query option — discussed, then determined to be **wrong** (shouldn't exist)
- Tests for the visibility/replication model
- Engine profiles for external engines (pebbleengine, duckdbengine, pgengine) — never updated
- Design doc update with the corrected model and the Q4 "universal ADT support" insight

---

## d) TOTALLY FUCKED UP

### The naming journey: Visibility → Replication + Network + Lag

**What happened:** I proposed `VisibilityModel` with `VisibilityLocal` / `VisibilityGlobal` as the central dimension. Through four rounds of questioning, this was progressively dismantled:

1. **"VisibilityGlobal is a bad name"** — Correct. "Global" is vague.
2. **"Is Visibility the right prefix, and why?"** — This forced me to research what "visibility" actually means in distributed systems. It's a _temporal_ concept (MVCC visibility, visibility lag), but I was using it for a _spatial/topological_ question (does data cross process boundaries?). The conflation was wrong.
3. **"Does this query need network access and how far away is it?"** — This exposed that I was conflating TWO orthogonal dimensions into one: (a) replication topology and (b) network latency. These scale differently: replication lag affects staleness; network RTT is a fixed per-query cost overhead.
4. **"WithReplicationMode(ReplicationLeaderless) — why is this on a query?"** — This exposed the biggest design error. I put a routing concern on the query declaration. But replication is an engine property. Queries don't know or care about topology. The `visibility` field on `QueryConfig` was fundamentally wrong.

**The correct model we arrived at (not yet implemented):**

```go
type Replication string
const (
    ReplicationNone         Replication = "none"
    ReplicationSingleLeader Replication = "single-leader"
    ReplicationMultiLeader  Replication = "multi-leader"
    ReplicationLeaderless   Replication = "leaderless"
)

type EngineProfile struct {
    // ...
    Replication    Replication     // DDIA Ch5: replication mode
    ReplicationLag time.Duration   // DDIA Ch5: replication lag
    NetworkRTT     time.Duration   // DDIA Ch1: round-trip time
}
// QueryConfig gets ZERO new fields — replication is engine-only.
```

### The auto-commit problem

The auto-commit daemon committed the wrong/half-baked code as commit `31f26b8c` with a detailed commit message describing the `Visibility` model. This means the rejected naming is now in git history. The code compiles and tests pass (because the new fields default to zero values), but the concept is wrong.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Research terminology BEFORE naming types.** I should have checked what "visibility" means in DDIA and distributed systems literature before proposing it as a type name. The naming crisis consumed 5+ question rounds that could have been avoided by 10 minutes of research upfront.

2. **Separate orthogonal dimensions from the start.** I collapsed replication topology, replication lag, and network latency into a single "visibility" concept. DDIA keeps these separate (Ch1 for latency, Ch5 for replication). I should have started from DDIA's established vocabulary.

3. **Don't put engine properties on queries.** This was a category error. Queries declare what to compute (folds, ADTs, read patterns). Engines declare how data is stored (layout, cost, replication). I conflated routing concerns with query declaration.

4. **Don't start implementing until the design survives questioning.** I wrote `visibility.go`, modified `engine.go`, `query.go`, and `cost.go` before the naming was settled. The auto-commit daemon then locked the wrong code into history.

5. **The "READ, UNDERSTAND, RESEARCH, REFLECT" prompt was the right call.** When challenged on naming, my first instinct was to offer alternatives rather than research the actual concept. I needed to step back and ground the terminology in established literature.

### Design improvements

6. **The Q4 "universal ADT support" insight is not captured in code or docs.** The user proposed that every engine should support every ADT (with cost warnings/screams), rather than engines being partial (DuckDB doesn't implement GraphBackend, etc.). This is a significant design direction change for the metaengine that deserves its own exploration. Currently engines silently skip unsupported ADTs — the user wants honest cost signals instead.

7. **The design doc needs a major update.** It uses `Visibility` throughout, doesn't mention `NetworkRTT` or `ReplicationLag` as separate dimensions, includes a `WithVisibility()` query option that was determined wrong, and doesn't capture the "replication is an engine property, not a query property" insight.

---

## f) Up to 50 Things to Do Next

### Fix the broken code (P0 — the committed code uses rejected naming)

1. **Rewrite `metaengine/visibility.go` → `metaengine/replication.go`** with `Replication` type and `ReplicationNone`/`ReplicationSingleLeader`/`ReplicationMultiLeader`/`ReplicationLeaderless` constants
2. **Remove `Visibility` and `TypicalLag` from `EngineProfile`** in `engine.go`
3. **Add `Replication Replication`, `ReplicationLag time.Duration`, `NetworkRTT time.Duration`** to `EngineProfile` in `engine.go`
4. **Add helper methods:** `EffectiveReplication()`, `EffectiveReplicationLag()`, `EffectiveNetworkRTT()` on EngineProfile
5. **Remove `visibility VisibilityModel` from `QueryConfig`** in `query.go` — this field should not exist
6. **Fix `estimateCostWithLag` in `cost.go`** — separate `ReplicationLag` from `NetworkRTT`; RTT is additive fixed, lag scales differently
7. **Add `time` import** to `engine.go` (already done in committed code, verify after rewrite)
8. **Verify build compiles** with `go build -tags "goexperiment.jsonv2" ./...`

### Update engine profiles (P1)

9. **Memory engine** (`memory_engine.go`): Set `Replication: ReplicationNone`, leave lag/RTT as zero defaults
10. **SQLite engine** (`engine.go` `SQLiteEngineProfile()`): Same
11. **Pebble engine** (`pebbleengine/engine.go`): Same
12. **DuckDB engine** (`duckdbengine/engine.go`): Same
13. **Postgres engine** (`pgengine/engine.go`): Same — but could set `NetworkRTT` for shared remote Postgres (design decision)

### Planner integration (P1)

14. **Decide if the planner needs a replication filter at all.** If replication is engine-only and queries don't declare visibility requirements, the planner may not need a filter rule — it just routes by cost, and global engines lose on cost.
15. **If a filter IS needed,** add a `replicationRule` to the rule pipeline (but based on what query property? This needs design)
16. **Update `estimateCost` call in `planQuery`** (planner.go:167) to pass `NetworkRTT` from the engine profile
17. **Consider whether replication lag should warn the user** via diagnostics (e.g., "query routed to engine with 5s replication lag")

### Cost model correctness (P1)

18. **Separate compute cost from network cost.** `latency = (ops × nsPerRead / 1e6) + networkRTTMs`. RTT doesn't multiply with volume.
19. **Decide how ReplicationLag factors into cost.** Is it additive (like RTT) or does it gate (reject engines above a threshold)? The user's Q2 answer suggested `WithMaxLag()` might not work for many engines but could be interesting for Spanner.
20. **Write tests for the cost model** verifying that NetworkRTT is additive (not multiplied by volume)

### Design doc update (P2)

21. **Rewrite `docs/planning/meta-engine-eventual-consistency-and-iroh.md`** to use `Replication` / `ReplicationLag` / `NetworkRTT` naming throughout
22. **Remove the `WithVisibility()` query option from the doc** — explain that replication is engine-only
23. **Add a section on the three orthogonal dimensions** (replication mode, replication lag, network RTT) with DDIA references
24. **Add the Q4 "universal ADT support" insight** as a separate section or companion doc
25. **Update the EngineProfile code examples** in the doc
26. **Update the materialize-vs-replay cost model** to show compute/network/lag separation
27. **Add the "replication is engine property, not query property" principle** to the design rationale

### Universal ADT support (P2 — separate but related)

28. **Write a design exploration doc** for "every engine supports every ADT with honest cost warnings"
29. **Audit which engines skip which ADTs** (DuckDB: no Graph/Vector/Search/Spatial/Set/Multimap/Log; Pebble: no Vector/Search/Spatial; etc.)
30. **Design the "SCREAM" diagnostic system** — what does the planner emit when a query is routed to a non-native engine? (e.g., "3 Graph queries routed to Pebble: O(N) scan, no native graph support")
31. **Decide if fallback implementations are needed** (brute-force graph in Pebble? In-memory adjacency in DuckDB?) or if the Memory engine's brute-force approach is the universal fallback

### Testing (P2)

32. **Write tests verifying all engines default to `ReplicationNone`**
33. **Write tests for the cost estimator with non-zero `NetworkRTT`**
34. **Write tests for the cost estimator with non-zero `ReplicationLag`**
35. **Write a test verifying that a local engine beats a global engine on cost** (RTT + lag make global engines lose)
36. **Update `adttest.RunMatrix`** — no changes needed (replication doesn't affect ADT correctness)

### Iroh integration prep (P3 — future work)

37. **Evaluate Iroh's C binding maturity** — does `iroh-net` / `iroh-docs` expose a stable C API?
38. **Check for community Go bindings** for Iroh
39. **Prototype `metaengine/irohengine/` module skeleton** with `//go:build cgo` isolation
40. **Design the `iroh.Replicated(engine)` wrapper API** (Level 2 integration)
41. **Map CRDT-safe operations** (MapSet, SetAdd, CounterIncrement, MultiAdd, LogAppend) vs non-CRDT (MapUpdate, MapDelete)
42. **Design the PN-Counter implementation** for the Counter ADT via iroh-docs authors

### Codebase hygiene (P3)

43. **Update `AGENTS.md` metaengine section** with the replication model once implemented
44. **Add `Replication` to the `EngineProfile.String()` method** output
45. **Update `CollectionInfo` struct** — should it expose the engine's replication mode to consumers?
46. **Consider whether `PlanResult` / `QueryAssignment` should include replication diagnostics**
47. **Update `cmd/api-stability` golden** if any exported symbols change
48. **Update the metaengine design docs** (`meta-engine-design.md`, `meta-engine-assumptions-and-query-planning.md`) with the replication dimension

### Broader architecture (P3)

49. **Consider how replication interacts with `projectionhost`** — if the engine replicates, does the projection host need to be replication-aware?
50. **Consider how replication interacts with `EventLog`** — can the event log itself be replicated, or only projections?

---

## g) Questions I Cannot Answer Myself

### Q1: Should the committed `31f26b8c` be reverted/rebased, or should we write a follow-up commit that replaces the code?

The auto-commit daemon committed the half-baked `Visibility` model as `31f26b8c`. Options:

- **(a)** Leave it and write a corrective commit (history preserves the mistake, but no force-push needed)
- **(b)** Rebase/revert it (cleaner history, but requires rewriting a daemon commit)

The project rules say "NEVER git reset" and "NEVER revert changes you didn't author" — but this IS my change. I lean toward (a) — a follow-up "fix" commit is safer and the project convention seems to favor forward-only history.

### Q2: Should NetworkRTT be on EngineProfile (fixed per-engine) or on the Store/connection (variable per deployment)?

A Pebble engine is in-process (RTT=0), but a shared Postgres might be on localhost (RTT=0.1ms) or across the country (RTT=50ms). The same engine binary has wildly different RTT depending on deployment. Should `NetworkRTT` be:

- **(a)** A field on `EngineProfile` set at construction time (consumer calibrates it)
- **(b)** A field on the `Store` or passed as a planner option
- **(c)** Auto-calibrated (like the existing `Calibrate()` method in `reliability.go`)

### Q3: Do you want me to capture the "universal ADT support with cost screams" insight as a formal design doc, or just leave it as a note for now?

Your Q4 answer proposed a significant design direction change: every engine should support every ADT (with honest cost warnings) rather than silently skipping. This affects the planner's routing logic, the engine interface contract, and every engine's Profile() implementation. It's big enough to warrant its own doc, but it's also a separate concern from the replication model.

---

## Resolution (2026-08-03)

The design exploration was valuable. The wrong `Visibility` model was replaced by the `Replication` model (`72818e88`, ADR-0093). All P0-P2 code/doc tasks resolved by later sessions. The PN-Counter insight informed the replication design. Iroh integration evaluated in ADR-0096 (deferred — requires CRDT expertise and iroh-docs authors). The `metaengine/irohengine/` module skeleton was NOT created (Iroh deferred).
