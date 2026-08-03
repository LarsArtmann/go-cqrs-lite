# Session: Full ADR Review + SSE Architecture Investigation

**Date:** 2026-08-03
**Scope:** Read all 91 ADRs (0001-0093), synthesized patterns, gathered decisions via Q&A, then investigated the SSE architecture across three repos (`go-cqrs-lite`, `go-sse`, `cqrs-htmx`).

---

## Part 1: ADR Review Synthesis

All 91 ADRs were read and analyzed. Three big arcs span the collection:

### Arc 1: Identity/Naming Honesty (0001 → 0024 → 0058)

| ADR  | Step                                                                                                                                       |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 0001 | Kill OO aggregate (behavioral): Decider pattern replaces 9-method Aggregate Root                                                           |
| 0024 | Export all 8 phantom marker types (consistency)                                                                                            |
| 0058 | Rename `Aggregate*` → `Stream*` (identity): the library dismantled the concept, but the name survived on the types. 1:1 mechanical rename. |

### Arc 2: Codec Safety Net (0015 → 0044 → 0050 → 0051 → 0052 → 0053 → 0054)

A multi-step journey from JSON default → CBOR default, with a safety net built first:

| ADR  | Step                                                                                           |
| ---- | ---------------------------------------------------------------------------------------------- |
| 0015 | Introduce CBOR codec (opt-in, JSON stays default)                                              |
| 0044 | Envelope wrapping for blind stores — every write stamps encoding, every read auto-detects      |
| 0050 | Envelope JSON fallback — keep forever (not a transitional measure)                             |
| 0051 | Flip `event.DefaultCodec` to CBOR                                                              |
| 0052 | Transport boundary: SSE does NOT auto-convert; consumers use `WithPayloadTransform` explicitly |
| 0053 | Unified codec default flip — all blind store defaults are now CBOR                             |
| 0054 | `json/v2` case-insensitive decode (silent zero-value bug prevention)                           |

### Arc 3: Metaengine Bet (0061 → 0093)

33 ADRs consumed by the metaengine — the strategic future of the project:

| ADR Range | Theme                                                                                                                               |
| --------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| 0061-0068 | SQLite engine, dependency boundary, pushdown, tx-atomic MapUpdate, multimap seq-seed                                                |
| 0069-0070 | Error wrapping helpers, transform fallback observability                                                                            |
| 0071      | DuckDB CGo introduction (first CGo in the project)                                                                                  |
| 0072-0076 | Pushdown, layout planning, Pebble engine, ADT test harness, raw readers                                                             |
| 0077-0081 | Graph reconciliation, kv coexistence, SSE consolidation, dialect upsert, runtime casts                                              |
| 0082      | Store redesign analysis (4 alternatives to eliminate casts — all rejected)                                                          |
| 0083-0085 | Planner rule pipeline, layered architecture (StorageLayout, cost matrix, materialize-vs-replay), new ADTs (Vector, Search, Spatial) |
| 0086-0087 | DuckDB engine, Postgres engine                                                                                                      |
| 0088-0090 | Block-level suppression, flight recorder, benchkit evidence metrics                                                                 |
| 0091-0093 | SSE consolidation decision, DuckDB columnar-native, replication model (DDIA Ch5)                                                    |

### Cross-Cutting Patterns Identified

**1. "Keep both" is a recurring verdict:**

| ADR       | Case                            | Verdict                                                                 |
| --------- | ------------------------------- | ----------------------------------------------------------------------- |
| 0077      | GraphBackend vs `graph/` module | Keep both — different layers (planner ADT vs projection tier)           |
| 0078      | kv.ViewStore vs metaengine      | Keep both — different complexity tiers                                  |
| 0079/0091 | Two SSE implementations         | Keep both — different push semantics (raw events vs read-model results) |
| 0043      | Two DLQ types                   | Keep both — different lifecycles (dispatch-retry vs projection-poison)  |

Each is individually justified. Collectively they signal a tolerance for parallel abstractions.

**2. Deferred cleanup debt is real and tracked:**

| ADR  | What's deferred                                                       | Status                                         |
| ---- | --------------------------------------------------------------------- | ---------------------------------------------- |
| 0028 | Ghost bus deletion (memory bus, PostgresBus, event.Bus reactive seam) | "not yet executed"                             |
| 0027 | PostgresBus deprecated                                                | Still present for backward compat              |
| 0031 | Metadata alias-to-struct conversion                                   | Incomplete — aliases repointed but not removed |
| 0064 | retry/ extraction to go-retry repo                                    | Proposed                                       |
| 0065 | idempotency/ extraction to go-idempotency repo                        | Proposed                                       |
| 0059 | DLQ unification proposal                                              | Proposed                                       |

**3. Process lessons baked into ADRs (institutional memory):**

- **ADR-0074**: `slices.Backward` yields copies — auto-commit daemon reverted the fix TWICE while a status report claimed GREEN on stale evidence
- **ADR-0090**: Benchmarks silently measured EMPTY stores for multiple sessions (event type casing mismatch: `"MeBenchItemCreated"` vs `meBenchItemCreated`) — "benchmarks without assertions are theater"
- **ADR-0083**: Diagnostics overwrite bug found mid-extraction — `plan.Diagnostics = checkWriteAmplification(...)` silently destroyed schema enforcement diagnostics
- **ADR-0093**: 5-round naming crisis ("Visibility" → "Replication") before landing on DDIA-canonical terms
- **AGENTS.md "Stale GREEN" anti-pattern**: documented as recurring across 4+ sessions

**4. Genuine architectural tensions documented honestly:**

- ADR-0020 vs ADR-0049: opposite middleware rebuild strategies (pre-compute on `Use()` vs rebuild on every `Dispatch()`), both "correct" for their respective call sites
- ADR-0081/0082: runtime casts are "structurally necessary" — Go can't express heterogeneous generic containers until ~1.29+ (existential types, associated types)

---

## Part 2: Decisions Gathered via Q&A

Five questions were asked. Answers and implications:

### Q1: Metaengine Future — What's the endgame?

**Answer: Genuinely undecided — let real consumer demand decide the boundary.**

The metaengine has consumed 33 of the last 33 ADRs and AGENTS.md flags it as "possibly a future dedicated project." Scope now spans DDIA replication theory, CALM theorem reasoning (5 of 10 ADTs are monotonic → CRDT-convergent), columnar layout planning, and 10 ADTs across 5 engines.

**Implication:** No premature spinoff. Let it grow inside the monorepo until a real non-CQRS consumer forces the boundary call.

### Q2: Deferred Debt — Which items to execute?

**Answer: Execute ALL FOUR debt items.**

1. **Ghost bus removal** (ADR-0028): Execute the deferred deletion — `memory/bus.go`, `memory/command_bus.go`, `storage/pg_bus.go`, `event.Bus`, `event.Subscriber`, `event.Middleware`, reactive `EventBus`. The "five buses" anti-pattern cleanup.
2. **Metadata aliases** (ADR-0031): Complete the alias-to-struct conversion for `command.Metadata` / `query.Metadata`. They were repointed from `event.Metadata` to `metadata.CustomData[MetadataKey]` but are still aliases, not standalone structs.
3. **Extract retry/** (ADR-0064): Create `github.com/larsartmann/go-retry` repo + re-export alias in `go-cqrs-lite/retry/`.
4. **Extract idempotency/** (ADR-0065): Create `github.com/larsartmann/go-idempotency` repo + re-export alias in `go-cqrs-lite/idempotency/`.

**Implication:** This is the next real roadmap. These aren't "someday" items — they're committed tasks.

### Q3: Keep-Both Philosophy — When does tolerance tip into confusion?

**Answer: Leave it case-by-case internally + add a decision matrix in SKILL.md for consumers.**

Internal ADRs stay ad-hoc (each case is genuinely unique). But consumers get explicit routing: "if you need X, use Y; if you need Z, use W." Turn the overlap into a feature by making the choice explicit in consumer-facing documentation.

**Implication:** No codified rule for internal "keep both" decisions, but SKILL.md gains a decision matrix so consumers aren't left choosing between equally-valid options with no guidance.

### Q4: Benchmark Trust — Are the cost constants real?

**Answer: Not very confident yet.**

Given ADR-0090 (benchmarks measured empty stores for sessions due to casing mismatch) and ADR-0074 (auto-commit daemon reverted a fix twice while GREEN was claimed), the recurring "stale GREEN" pattern documented across 4+ sessions means current trust in metaengine cost constants (`PebbleNsPerRead=708`, `DuckDBNsPerRead=3000`, etc.) is low.

**Implication:** The cost constants need re-verification with the ADR-0090 correctness assertions enabled before being trusted for planner routing. The planner's engine-selection decisions are only as trustworthy as the numbers they're based on. **Highest-leverage next move: re-run metaengine cost-model benchmarks with empty-store assertions across all five engines, either pin the constants with evidence or fix what they reveal.**

### Q5: Runtime Casts — Comfortable living with them?

**Answer: Revisit case-by-case (per-query).**

No proactive codegen investment. The casts (ADR-0081/0082) are bounded, fail loudly on mismatch, and pay zero cost when not hit. `cqrs-gen` typed-Store generation earns its place only when profiling demands it on a specific hot query set — not as a blanket solution.

**Implication:** Runtime API stays as-is. `ExecuteTyped[I,V]` + reify DX tax is acceptable for the foreseeable future. Codegen is a targeted tool, not a foundational investment.

### Decision Summary Table

| Theme                | Stance                         | Implication                                                              |
| -------------------- | ------------------------------ | ------------------------------------------------------------------------ |
| Metaengine scope     | Undecided, demand-driven       | No premature spinoff; let it grow until forced                           |
| Deferred debt        | Execute **all four**           | This is the next real roadmap                                            |
| Keep-both philosophy | Case-by-case + SKILL.md matrix | Internal stays ad-hoc; consumers get routing guidance                    |
| Benchmark confidence | Low — "not very confident yet" | Re-verify cost constants with assertions before trusting planner routing |
| Runtime casts        | Revisit per-query              | No proactive codegen; cqrs-gen earns its place per hot query             |

---

## Part 3: SSE Architecture Investigation

### The Question That Started It

"Why do we have 2 SSE?" — led to a three-repo investigation that uncovered a significant architectural inconsistency.

### The Two SSE Implementations in go-cqrs-lite

|                 | `transport/http.SSEBroker`                                              | `metaengine.ServeSSE`                               |
| --------------- | ----------------------------------------------------------------------- | --------------------------------------------------- |
| **Pushes**      | Raw domain events (`event.Event` bytes)                                 | Materialized query results (typed `V` values)       |
| **Source**      | `event.Bus` (event-sourcing layer)                                      | `Watcher[V]` (read-model layer)                     |
| **Example**     | "user.created" event                                                    | "user count is 42"                                  |
| **Replay**      | `event.SeekableJournal` (durable)                                       | In-memory ring buffer (cheap, recent only)          |
| **Features**    | Event filter, payload transform, byte budget, REST backfill, OTel spans | Heartbeat keepalive, collection-scoped subscription |
| **Module tier** | Tier 4 (`transport/http`)                                               | Tier 0 (`metaengine`)                               |

ADR-0091 documented the rationale: they serve different layers (event-sourcing vs read-model), depend on different types, and merging would break the dependency boundary that keeps `metaengine` zero-dep (ADR-0062).

### The Discovery: go-sse Already Exists

`/home/lars/projects/go-sse` is a standalone SSE transport library extracted from production use. It provides exactly the four primitives that both `go-cqrs-lite` SSE implementations reimplement from scratch:

| Component                        | go-sse file                    | What it does                                                                                            |
| -------------------------------- | ------------------------------ | ------------------------------------------------------------------------------------------------------- |
| `Event`, `EventID`, `WriteEvent` | `event.go`                     | SSE wire-format types + allocation-minimized serializer                                                 |
| `Stream`                         | `stream.go`                    | Single SSE connection — headers, mutex-guarded send, heartbeat, disconnect hooks, `Last-Event-ID`       |
| `Broadcaster[T]` + `fanOut[T]`   | `broadcaster.go` + `fanout.go` | Generic subscriber fan-out — subscribe, broadcast, close, hooks. Non-blocking (drops to slow consumers) |
| `EventStore` + `Replay`          | `replay.go`                    | Reconnection replay — missed events sent on reconnect via `EventsAfter(lastID)`                         |

**Critical finding: `go-cqrs-lite` imports `go-sse` ZERO times.** Both SSE implementations reimplement the wire format (`fmt.Fprintf`), fan-out (`map[clientID]chan`), and replay (`dedup.Ring` / ring buffer) from scratch.

### The Smoking Gun: cqrs-htmx Is the Model Consumer

`/home/lars/projects/cqrs-htmx` consumes `go-sse` v0.3.0 correctly — exactly the thin-adapter pattern that `go-cqrs-lite` should follow:

| cqrs-htmx file       | Role                                                                                                    | Lines of real value                  |
| -------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `sse_event.go`       | Pure type aliases: `SSEEvent = sse.Event`, `SSEStream = sse.Stream`, `WriteSSEEvent = sse.WriteEvent`   | ~60 lines of delegation              |
| `sse_store.go`       | `SSEEventStore = sse.EventStore`, `ReplayEvents = sse.Replay` (verbatim delegation)                     | ~25 lines of delegation              |
| `sse_broadcaster.go` | Embeds `*sse.Broadcaster[sse.Event]`, adds CQRS hooks (`BroadcastOnSuccess`, `BroadcastOnError`)        | ~110 lines — the CQRS-specific value |
| `event_store_sse.go` | `JournalSSEStore` — adapts `event.Journal` → `sse.EventStore` with consumer-provided `EventToSSEMapper` | ~200 lines — the CQRS bridge         |

cqrs-htmx adds ~200 lines of real CQRS value (dispatch hooks + journal-backed replay adapter) on top of `go-sse`'s ~600 lines of primitives. The `//nolint:wrapcache // pure delegation` comments throughout are the signature of an adapter that delegates cleanly to a well-scoped extraction.

### The Three-Repo Comparison

| Repo                       | SSE strategy                  | Consumes `go-sse`?               | reimplements wire/fanout/replay?      |
| -------------------------- | ----------------------------- | -------------------------------- | ------------------------------------- |
| `go-sse`                   | The extraction itself         | —                                | N/A (IS the primitives)               |
| `cqrs-htmx`                | Thin CQRS adapter on `go-sse` | ✅ Yes (v0.3.0, 12 go.mod files) | ❌ No                                 |
| `go-cqrs-lite` `SSEBroker` | Full reimplementation         | ❌ No                            | ✅ Yes (wire format, fan-out, replay) |
| `go-cqrs-lite` `ServeSSE`  | Full reimplementation         | ❌ No                            | ✅ Yes (wire format, fan-out, replay) |

### The Contradiction with ADR-0091

ADR-0091 explicitly rejected a "shared SSE utility package" with the rationale:

> _"the shared code is trivial Go stdlib (`http.Flusher`, `fmt.Fprintf`); extracting it adds coupling without meaningful deduplication"_

This ADR was deciding between merging the **two go-cqrs-lite** SSE implementations. It never considered that `go-sse` — which IS the extracted shared utility — already existed and was already being consumed correctly by cqrs-htmx.

**The ADR-0091 rationale was written as if `go-sse` didn't exist.** Meanwhile cqrs-htmx was already proving the extraction was worth it.

### The Honest Read

`go-sse` was extracted from cqrs-htmx's production code. cqrs-htmx consumes it correctly. `go-cqrs-lite` was never refactored to follow. The ADR-0091 "keep both, don't extract" decision was rationalizing a status quo that a sibling project had already moved past.

The two `go-cqrs-lite` SSE implementations are the **debt**, not the architecture. cqrs-htmx is the reference for what the cleanup looks like:

- `transport/http.SSEBroker` → becomes a thin `event.Bus` → `sse.Broadcaster` adapter (mirroring cqrs-htmx's `Broadcaster` + `JournalSSEStore`)
- `metaengine.ServeSSE` → becomes a thin `Watcher[V]` → `sse.Broadcaster[V]` adapter

This is a new debt item not captured in any existing ADR — it was discovered during this session's investigation.

---

## Part 4: Key Findings & Recommended Next Steps

### Findings Ranked by Impact

1. **SSE three-repo inconsistency (NEW)**: `go-sse` exists and is consumed by cqrs-htmx, but `go-cqrs-lite` ignores it entirely — two reimplementations of primitives that were already extracted. ADR-0091's rationale is contradicted by the existence of `go-sse`. This is undocumented debt discovered this session.

2. **Benchmark trust deficit**: Low confidence in cost constants means the metaengine planner's routing decisions are currently untrustworthy. The ADR-0090 correctness assertions exist but haven't been systematically applied to verify the constants that drive the planner.

3. **Deferred debt backlog**: Four committed items (ghost buses, metadata aliases, retry extraction, idempotency extraction) form a concrete roadmap but have no execution timeline.

4. **Metaengine scope creep**: 33 ADRs, undecided on spinoff. The boundary between "CQRS library feature" and "standalone storage-planner product" is dissolving with no forcing function yet.

### Recommended Execution Order

1. **Verify benchmark constants** (unblocks trust in the planner): re-run metaengine cost-model benchmarks with ADR-0090 empty-store assertions across all 5 engines. Either pin constants with evidence or fix what they reveal.

2. **Write an ADR for the SSE finding**: Document that `go-sse` exists, `go-cqrs-lite` should consume it, and ADR-0091's rationale needs revisiting. Propose `transport/http.SSEBroker` and `metaengine.ServeSSE` as thin adapters over `go-sse` primitives (following the cqrs-htmx pattern).

3. **Execute the four deferred debt items** (in priority order):
   - Ghost bus removal (ADR-0028) — largest blast radius, do first
   - Metadata alias completion (ADR-0031) — finish the half-done work
   - retry/ extraction (ADR-0064) — straightforward, proven pattern
   - idempotency/ extraction (ADR-0065) — companion to retry

4. **Add SSE decision matrix to SKILL.md**: Route consumers between `transport/http.SSEBroker` (raw events) and `metaengine.ServeSSE` (read-model results) with explicit guidance.

---

## Appendix: All ADRs Read

91 ADRs total (0001-0093, with 0036 and 0041 as numbering gaps):

- **0001-0010**: Foundational decisions (Decider, error taxonomy, multi-module, saga removal, tombstone, ISP split, gopls, typed handlers, Pebble scope, io.Closer)
- **0011-0020**: Interface cleanup (ErrDispatcherClosed, catalog split, zero-copy, test deps, CBOR, outbox declined, schema registry, distributed checkpointing, CBOR envelope, perf patterns)
- **0021-0030**: Storage evolution (close semantics, KV abstraction, Pebble KV adapter, ID markers, transport strategy, experimental features, Postgres LISTEN/NOTIFY, Watermill, storage consolidation, projection dissolution)
- **0031-0040**: Domain refinement (metadata split, readmodel→kv merge, multi-DB split, session store boundary, branded DSN rejected, projection extraction, graph tier, graph schema, deriver)
- **0042-0055**: Reliability + codec (pure replay DLQ, DLQ unification, blind store encoding, eventtest path fix, seven-tier model, COSE, deterministic JSON, dispatch-time middleware, envelope fallback, CBOR default, transport codec, unified flip, json/v2, cqrs-lint loader)
- **0056-0065**: Polish + extraction (timezone types, catalog REST, aggregate→stream rename, DLQ proposal, benchkit, metaengine SQLite, dep boundary, pushdown, retry extraction, idempotency extraction)
- **0066-0080**: Metaengine deep dive (reify, tx-MapUpdate, multimap seq-seed, error helpers, transform observability, DuckDB CGo, pushdown impl, layout planning, Pebble engine, ADT test harness, raw readers, graph reconciliation, kv coexistence, SSE consolidation, dialect upsert)
- **0081-0093**: Metaengine advanced (runtime casts, store redesign analysis, planner pipeline, layered architecture, new ADTs, DuckDB engine, Postgres engine, block suppression, flight recorder, benchkit evidence, SSE decision, columnar-native, replication model)

---

## Resolution (2026-08-03)

This session's 4 recommended next steps have been routed:

1. **Verify benchmark constants** → TODO_LIST "Benchmark Trust" (correctness assertions for 29 unasserted benchmarks + DuckDB/Postgres engine benchmarks)
2. **Write ADR for SSE finding** → TODO_LIST "SSE Consolidation" (ADR for go-sse consumption; ADR-0091's rationale needs revisiting)
3. **Execute four deferred debt items** → TODO_LIST "Deferred Debt" (ghost bus removal ADR-0028, metadata aliases ADR-0031, retry/ extraction ADR-0064, idempotency/ extraction ADR-0065)
4. **Add SSE decision matrix to SKILL.md** → TODO_LIST "SSE Consolidation"

The go-sse finding (go-sse exists as standalone library, go-cqrs-lite reimplements SSE) was the most significant discovery. ADR-0096 (Iroh evaluation) was also written from this session's work.
