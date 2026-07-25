# Session Status: Meta-Engine Design & Event-Query Model

**Date:** 2026-07-23 20:30
**Session focus:** How to separate the storage layer from the domain/compute layers so operators choose infrastructure at deployment time, leading to the discovery of the Event-Query Model and the Meta-Engine concept.

---

## Session Journey (Chronological)

This session started with a practical problem and evolved into a research-grade architecture discovery:

1. **"Apps going down on the DB layer"** — Consumer code imports `database/sql`, touches `*sql.DB`, writes SQL DDL. The storage choice is baked into the app at build time.
2. **Leak analysis** — Traced 4 exact leak points in go-cqrs-lite where storage internals escape into consumer code.
3. **Plugin exploration** — Explored Go's plugin mechanisms (registration pattern, `plugin` package, `hashicorp/go-plugin`, WASM). Concluded: `database/sql` registration pattern is the only correct answer for a library.
4. **`go-plugin-mvp` analysis** — Studied the sibling project. Found the closed-enum anti-pattern (`if backend == SQLite { ... } else { memory }`) that we must avoid.
5. **Data structure research** — Wikipedia's "Data Structure" article revealed the ADT vs Data Structure distinction. Mapped 7 ADTs (Map, SortedMap, Multimap, Counter, Set, Graph, Log) to query patterns and engines.
6. **"Entity" rejection** — User correctly rejected entity-based thinking. Events are the source of truth. State is derived. Reframed everything as projections.
7. **Cost-based optimizer insight** — Every query CAN be served by every engine. The question is cost given hardware constraints (RAM, disk, CPU, time, network-when-remote). This makes the meta-engine a cost-based optimizer, not a capability matcher.
8. **Scale-dependent structure selection** — Bloom filter vs hash set, B-tree vs sorted slice, counter vs scan. The optimal structure is a function of cardinality (N).
9. **Event + Query sufficiency** — The breakthrough. Two primitives are enough. The fold return type IS the ADT. The query input type IS the read pattern. No View, no Store, no Entity, no stringly-typed declarations.
10. **Graph-at-three-levels** — User's insight: data is a graph at Runtime (RAM), Disk (projections), and Time (event log). The meta-engine keeps these in sync.
11. **Command-Event-Query symmetry** — Three temporal roles on one graph: should (command), did (event), is (query). Commands produce events, queries consume them.
12. **Metadata, audit, sessions** — Metadata travels with events as first-class fields. Commands and queries are themselves event streams (for audit/analytics). Sessions can be event-streamed too.
13. **Auth is upstream's concern** — The meta-engine stores identity projections like any other data. Auth enforcement is a transport/policy concern, not a storage concern.

---

> **Update 2026-07-25:** This research session **became reality.** The
> `metaengine/v4` module now exists with `types.go`, `query.go`, `engine.go`,
> `sqlite_engine.go`, `memory_engine.go`, `planner.go`, `cost.go`, and
> `metaengine/projectionadapter/`. Section b) "PARTIALLY DONE" items 2–5 and
> section c) "NOT STARTED" items 1, 3–10, 15, 17, 18 are **DONE** — 174 BDD
> specs, 87.7% coverage, SQLite engine, cost calibration, projection adapter
> (ADRs 0061–0063). Still genuinely OPEN: the 4 leak fixes (b.1), hot-reload
> (c.11), `iter.Seq2` streaming (c.12), schema migration (c.13), formal paper
> (c.16), visual plan output (c.19), Bloom/HyperLogLog primitives (c.20).

## a) FULLY DONE

| #   | Deliverable                                                                                      | File                                                                           |
| --- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| 1   | Initial storage-domain separation analysis (3 paths, leak tracing)                               | `docs/planning/storage-domain-separation.md`                                   |
| 2   | Storage plugin system design (database/sql registration pattern)                                 | `docs/planning/storage-plugin-system.md`                                       |
| 3   | Consolidated "keep apps off DB layer" doc (4 leaks + action plan)                                | `docs/planning/keep-apps-off-db-layer.md`                                      |
| 4   | Meta-engine vision doc (cost-based optimizer, 4 deployment scenarios)                            | `docs/planning/meta-engine-design.md`                                          |
| 5   | Meta-engine assumptions & query planning (cost model, scale thresholds, "don't be stupid" rules) | `docs/planning/meta-engine-assumptions-and-query-planning.md`                  |
| 6   | Meta-engine project definition (separate project, formal model, build phases)                    | `docs/planning/meta-engine-project-definition.md`                              |
| 7   | **The Event-Query Model** — the definitive core abstraction                                      | `docs/planning/event-query-model.md`                                           |
| 8   | Analysis of `go-plugin-mvp` (WASM plugin marketplace, closed-enum anti-pattern)                  | (inline, not a separate doc)                                                   |
| 9   | Analysis of `cqrs-htmx/identity-model` (identity facts vs authz decisions)                       | (inline, not a separate doc)                                                   |
| 10  | Review of "100 Things I Hate" and "Perfect Software Architecture" requirement docs               | (inline, applied to design decisions)                                          |
| 11  | Network cost correction (local engines = zero network cost, not "cross-engine = expensive")      | Applied across `meta-engine-design.md` and `meta-engine-project-definition.md` |

## b) PARTIALLY DONE

| #   | What                          | What's missing                                                                                                                                            |
| --- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **The 4 leak fixes (Path A)** | Designed in detail with exact `file:line` references. Zero implementation. No Go code written.                                                            |
| 2   | **The Event-Query Model API** | The fold/On/Query/FilterOn/SortOn API is designed in pseudo-code. No concrete Go types. No prototype. No proof it compiles.                               |
| 3   | **The planner algorithm**     | 7-step algorithm designed in prose. No implementation. No validation that the greedy heuristic produces good plans.                                       |
| 4   | **Scale threshold tables**    | Cardinality thresholds documented (Bloom at N>100K, B-tree at N>10K, etc.). Not validated empirically. Not parameterized for different hardware profiles. |
| 5   | **Cost model**                | Formal ILP formulation written. Greedy approximation described. No implementation. No calibration against real benchmarks.                                |
| 6   | **Hot-reload design**         | The flow is described (add engine → re-plan → background replay → cutover). No implementation. No detail on how dual-read cutover works atomically.       |
| 7   | **Engine plugin interface**   | `Plugin` interface designed with `Profile()`, `Capabilities()`, `Build()`. No actual plugin files created.                                                |

## c) NOT STARTED

| #   | What                                                                |
| --- | ------------------------------------------------------------------- |
| 1   | Any actual Go code for ANY proposed design                          |
| 2   | Closing the 4 leaks in actual source files                          |
| 3   | A prototype of the planner (even 200 lines proving the concept)     |
| 4   | `metaengine.Query[Q, R]()` — the core API type                      |
| 5   | `metaengine.On()` — the fold registration mechanism                 |
| 6   | `metaengine.Plan(engines, queries...)` — the planner entry point    |
| 7   | Engine cost profiles (EngineProfile, Performance, Complexity types) |
| 8   | Auto-generated projection handlers (event → engine writes)          |
| 9   | Auto-generated read handlers (query → engine reads → result)        |
| 10  | FilterOn/SortOn typed accessor mechanism                            |
| 11  | Hot-reload (re-planning, background replay, live cutover)           |
| 12  | Streaming/pagination support (iter.Seq2, cursor-based)              |
| 13  | Schema migration across engines                                     |
| 14  | Degradation detection and warning system                            |
| 15  | Benchmark suite comparing planned vs hand-tuned layouts             |
| 16  | Formal model paper / thesis writeup                                 |
| 17  | The new project repo (go.mod, package structure, README)            |
| 18  | Tests for ANY of the above                                          |
| 19  | Visual plan output (D2/Mermaid diagram of the optimization plan)    |
| 20  | Bloom filter / HyperLogLog / Count-min sketch primitive modules     |

## d) TOTALLY FUCKED UP

| #   | What                                                             | Why it's fucked                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| --- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **7 overlapping design docs that partially contradict**          | `storage-domain-separation.md` proposes Path C (Declare+Bind with stringly-typed `.Filter("status")`). `keep-apps-off-db-layer.md` says "Path A is enough." `meta-engine-design.md` says "actually it's a cost-based optimizer." `event-query-model.md` says "actually Filter/Sort declarations are wrong — use fold return types." A reader hitting any one doc gets a different recommendation. No consolidation. No clear "read this one first" guide. |
| 2   | **Proposed `model.Entity` and `UserStore`**                      | User explicitly rejected both. Entity is CRUD thinking. UserStore violates ISP. Had to be told twice. The event-query model is the CORRECT framing, but the old docs still contain the rejected ideas.                                                                                                                                                                                                                                                    |
| 3   | **Stringly-typed `Filter("status")` in early docs**              | User called this out as "bad example." Column names as strings violate everything the project stands for. Fixed in `event-query-model.md` with typed accessors, but old docs still have the strings.                                                                                                                                                                                                                                                      |
| 4   | **Overstated network cost across engines**                       | Said "cross-engine queries are expensive, denormalize to avoid them." User corrected: local engines (SQLite file, Pebble dir) have zero network cost. Fixed in meta-engine-design.md and project-definition.md, but the original framing was wrong.                                                                                                                                                                                                       |
| 5   | **Auth section in event-query-model.md**                         | Wrote a full section on auth (9 layers, multi-tenant, RBAC). User said "Auth is not our problem, upstream should deal with that." The section should be removed. The identity-model analysis confirmed: auth is runtime policy (Casbin), not a storage concern.                                                                                                                                                                                           |
| 6   | **Never addressed sessions as event streams**                    | User asked "Why don't you want to Event Stream Sessions? This way we can do analytics on Sessions too." The identity-model treats sessions as ephemeral runtime objects. But the user is RIGHT — sessions CAN be events (SessionStarted, SessionEnded) and would benefit from the same projection machinery for analytics. Not addressed in any doc.                                                                                                      |
| 7   | **Never wrote concrete Go types**                                | Every design doc has pseudo-code Go. Not a single compilable type definition. Can't reason about ergonomics or feasibility from prose. The `FilterOn` accessor mechanism, the `metaengine.On` fold registration, the `Plan` return type — all hand-wavy.                                                                                                                                                                                                  |
| 8   | **Meta-engine-project-definition.md has LaTeX rendering issues** | The formal model section has broken LaTeX (`\mathformal{A}`, `\math meta-engine-design.md`, `budget_{\text{brain}}`). The math is also half-baked — it's an ILP formulation that nobody will solve as an ILP.                                                                                                                                                                                                                                             |

## e) WHAT WE SHOULD IMPROVE

| #   | Issue                                                | Fix                                                                                                                                                                                                                                                                                                                                                               |
| --- | ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Consolidate 7 docs into a clear narrative**        | Archive superseded docs. Create an index. Mark `event-query-model.md` as THE model, others as historical context.                                                                                                                                                                                                                                                 |
| 2   | **Build a minimal prototype**                        | 300 lines: define 2 queries, 2 engines, run the planner, verify the assignment. Without code, this is architecture astronautics.                                                                                                                                                                                                                                  |
| 3   | **The fold return type → ADT inference needs proof** | Claim: `(Key, Value)` return → Map, `Key` return → Set, `Delta` return → Counter, `Edge` return → Graph. This needs to actually work with Go generics. Write the type constraints and verify they compile.                                                                                                                                                        |
| 4   | **FilterOn typed accessor mechanism is unproven**    | `FilterOn(func(r FindUserResult) string { return r.Status })` — how does the planner extract "Status" from this? Reflection can't inspect closure bodies. The "named field descriptor" alternative (`r.Field("Status")`) is stringly-typed. This is an unsolved problem.                                                                                          |
| 5   | **Hot-reload is underdesigned**                      | The flow is 7 steps. But: how does dual-read work? What if the new projection errors during replay? How does atomic cutover happen without a distributed transaction? These are hard problems.                                                                                                                                                                    |
| 6   | **No observability story**                           | With 3+ engines, the operator needs: lag per projection, write amplification, query latency per engine, degradation status. Not in any doc.                                                                                                                                                                                                                       |
| 7   | **No testing strategy for the planner**              | How do you unit-test a cost-based planner? Property-based testing (rapid) on the planner: given random engines + patterns, verify every pattern is served.                                                                                                                                                                                                        |
| 8   | **Scale thresholds are unvalidated**                 | The cardinality tables (Bloom at N>100K, sorted slice at N<10K) are educated guesses. They need empirical validation on real hardware. The thresholds differ dramatically between an M.2 SSD and a spinning HDD.                                                                                                                                                  |
| 9   | **Sessions should be event-streamed**                | User's insight: `SessionStarted{ActorID, Token, Origin, At}` and `SessionEnded{ActorID, At}` as events. Benefits: analytics ("how many active sessions?"), audit ("who was logged in when X happened?"), security ("revoke all sessions for suspended user"). The identity-model treats sessions as ephemeral — but event-streaming them is strictly more useful. |
| 10  | **The meta-engine needs a name**                     | "Meta-engine" is generic. The project needs a real name for branding, repo creation, and documentation.                                                                                                                                                                                                                                                           |
| 11  | **Visual plan output**                               | Complaint #40 from "100 Things": "Do you have any Visualizations?" The planner should output a D2/Mermaid diagram showing: projections → engines → data structures → costs. Massive DX.                                                                                                                                                                           |
| 12  | **The formal model LaTeX is broken**                 | `meta-engine-project-definition.md` has rendering errors. Either fix the LaTeX or replace with a simpler notation that renders correctly in Markdown.                                                                                                                                                                                                             |

## f) Up to 50 Things to Do Next

### Immediate: Document Cleanup

1. Remove auth section from `event-query-model.md`, replace with one-liner
2. Add sessions-as-events insight to `event-query-model.md`
3. Mark `storage-domain-separation.md` as superseded by `event-query-model.md`
4. Mark `meta-engine-design.md` as "vision — see event-query-model.md for the actual API"
5. Create `docs/planning/README.md` index with reading order: event-query-model first, then assumptions, then design
6. Fix broken LaTeX in `meta-engine-project-definition.md`
7. Archive `storage-plugin-system.md` (registration pattern absorbed into event-query-model)

### Immediate: Path A (close the 4 leaks in go-cqrs-lite)

8. Add `sqlite.WithMaxOpenConns(n)` option to sqlite preset
9. Deprecate `bundle.Database()` with doc comment
10. Define `stack.ColumnType` enum (TypeString, TypeInt, TypeBool, TypeReal, TypeBytes)
11. Update `ViewMapper.Type` to use `ColumnType` instead of `string`
12. Change `SQLViewModel` return type to `kv.ViewStore[V,K]`
13. Add `bundle.RelationalProjection(name, schema, handler, types)` method
14. Update taskmanager example to remove all `database/sql` imports
15. Verify zero storage imports in taskmanager via grep

### Meta-Engine: Prototype (prove the concept)

16. Create the new project repo (go.mod, package structure)
17. Write `metaengine.Query[Q, R]()` type definition
18. Write `metaengine.On[E, V]()` fold registration
19. Write `metaengine.Delta`, `metaengine.Edge`, `metaengine.Remove`, `metaengine.Skip` types
20. Write `metaengine.Plan(engines, queries...)` entry point
21. Define `EngineProfile` with `Provides map[ADT]Performance`
22. Define `ADT` enum (Map, SortedMap, Counter, Set, Graph, Log, Multimap)
23. Write the fold-return-type → ADT classifier
24. Write the greedy engine assignment algorithm
25. Write degradation detection + warning output
26. Test: 2 queries + 2 engines → verify correct assignment
27. Test: memory-only + filter pattern → verify degradation warning
28. Test: graph pattern + no graph engine → verify hard error

### Meta-Engine: Filter/Sort Mechanism

29. Prototype `FilterOn` / `SortOn` typed accessor
30. Solve the field-path extraction problem (the hardest API design question)
31. Test: planner creates correct indexes from accessor declarations

### Meta-Engine: Engine Plugins

32. Write SQLite engine plugin with cost profile
33. Write Pebble engine plugin with cost profile
34. Write Memory engine plugin with cost profile
35. Write Neo4j engine plugin stub (for graph ADT testing)

### Meta-Engine: Auto-Generated Handlers

36. Write auto-generator for Map ADT (event → Set/Get/Delete)
37. Write auto-generator for Counter ADT (event → Increment)
38. Write auto-generator for Graph ADT (event → MergeEdge)
39. Define the boundary: auto-generate for single-document, custom handler for multi-table

### Meta-Engine: Read API

40. Write runtime typed store assembly (`store.Execute(ctx, query)`)
41. Wire each query to its assigned engine
42. Add streaming support (`store.Stream(ctx, query, fn)`)
43. Add cursor pagination (`metaengine.Cursor`)

### Meta-Engine: Advanced

44. Write hot-reload: re-plan on engine add/remove
45. Write background replay with checkpoint tracking
46. Write atomic cutover (dual-read during transition)
47. Write D2/Mermaid plan visualizer
48. Write benchmark suite (planned vs hand-tuned)
49. Validate scale thresholds empirically
50. Write the formal model paper

### Sessions as Events

51. Define `SessionStarted` / `SessionEnded` event types
52. Define session analytics queries (active sessions, session duration, concurrent users)
53. Define session revocation as a projection (suspend user → revoke sessions)

## g) Questions I CANNOT Answer Myself

**Q1: The `FilterOn` field-path extraction problem.**
`FilterOn(func(r FindUserResult) string { return r.Status })` — Go reflection cannot inspect closure bodies to extract "Status." The alternative `r.Field("Status")` is stringly-typed. Code generation (go generate) works but adds a build step. **Which mechanism do you want?** Options: (a) code generation, (b) stringly-typed field names but only inside the accessor (not in declarations), (c) a different API shape I'm not seeing, (d) something else entirely.

**Q2: Should the meta-engine be a separate repo or a monorepo module?**
I recommended separate repo (it's research-grade, different audience, different release cadence, the "lite" in go-cqrs-lite). But you might prefer monorepo for easier cross-refactoring and the existing CI/tooling. **What's your preference?**

**Q3: What should the project be named?**
"Meta-engine" is generic and doesn't communicate the value. The project does: cost-based storage optimization for event-sourced data, across any combination of engines. **What name do you want?** This determines the repo, the go.mod path, and all branding.

---

## Sessions as Events (Addressing the User's Question)

The user asked: "Why don't you want to Event Stream Sessions? This way we can do analytics on Sessions too."

The identity-model treats sessions as ephemeral runtime objects (not event-sourced). But the user is RIGHT that event-streaming sessions is strictly more useful:

```
SessionStarted { ActorID, Token, Origin, IPAddress, At }
SessionEnded   { ActorID, Token, At, Reason }
SessionRevoked { ActorID, Token, At, By }
```

Benefits of event-streaming sessions:

- **Analytics:** "How many concurrent sessions right now?" (Counter projection)
- **Audit:** "Who was logged in when the data breach happened?" (Time-range query)
- **Security:** "Revoke all sessions for suspended user" (projection reads SessionRevoked events)
- **Patterns:** "User logs in from 2 countries simultaneously" (Graph/Set projection)

These are ALL just Event + Query — the same machinery serves them. Sessions are not special; they're another event stream with its own projections.

The identity-model chose ephemeral sessions for simplicity. The meta-engine makes event-streaming sessions trivial because the projection cost is near-zero (it's just another fold). **This should be documented in the event-query-model.**

---

## Summary: What This Session Produced

The session evolved from a practical bug report ("apps go down on the DB layer") into the discovery of a genuinely novel architecture:

**The Event-Query Model:** Two primitives (Events + Queries) are sufficient to derive all storage decisions. The fold return type declares the ADT. The query input type declares the read pattern. Each query gets its own independent projection on its own optimal engine. The planner is a cost-based optimizer that maps declared patterns to the best available data structures, respecting cardinality, hardware constraints, and engine capabilities.

**The Meta-Engine:** A deployment-time, cost-based storage optimizer that takes event-sourced data + declared query patterns + whatever engines the operator provides, and automatically distributes projections across engines optimally. Run on a single SQLite, or ScyllaDB+ClickHouse+Neo4j — same developer code.

**The immediate fix (Path A):** Close 4 specific leak points in go-cqrs-lite so app code never imports storage packages. This is ~1 day of work and is the foundation everything else builds on.

**What's missing:** A single line of Go code. Everything is design docs. The concept needs a prototype to prove it's buildable.
