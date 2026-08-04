# Status Report: Metaengine Redesign — Session 2026-08-04 07:48

> **Scope:** This report covers the work done in the current session only.
> It is brutally honest about what went well, what went wrong, and what was
> forgotten. It does not research new topics — it audits the session's output.

---

## Executive Summary

The session advanced the `metaengine-redesign.md` design document from 1094 to
1568 lines (+474 lines). Four new decisions were recorded (D6-D9). Two major
architectural concepts were added (Cache Tier, Time as 4th Dimension). Seven of
ten open questions were resolved. However, the document has **internal
inconsistencies** — several decisions were recorded in §4 but NOT propagated to
the diagrams, code examples, and comparison tables they affect.

**Bottom line:** The design is richer but less coherent than when the session
started. The vision is correct; the execution has integration gaps.

---

## A) FULLY DONE ✅

| # | Work item | Evidence |
|---|---|---|
| 1 | Cache Tier concept (§5.5) — ZFS-ARC parallel, immutability argument, when it helps vs doesn't, otter library justification | §5.5: lines 665-790, with comparison table vs samber/hot |
| 2 | Named Engines refactor — all config examples converted from inline strings to named declarations | §6.2, §7.1, §7.4 all use `engines: map[string]EngineConfig` |
| 3 | Time as 4th Dimension concept (§5.6) — three layers, engine-native temporal support table, StreamReadAsOf interface | §5.6: lines 792-870 |
| 4 | Decision D6: System scope (layered-full) — owns all infrastructure, consumer provides domain-only | §4.6: lines 450-490 |
| 5 | Decision D7: Config via koanf (Go struct + YAML + env) | §4.7 + §7.2 with YAML and Go examples |
| 6 | Decision D8: Gradual migration (new system/ module, sqlite+memory first) | §4.8 with comparison table |
| 7 | Decision D9: Multi-bus support (operator-configured, multiple simultaneous) | §4.9 |
| 8 | StreamLogBackend interface (§10.1 resolved) — one interface for events+commands+queries, three adapters | §10.1 with full Go code |
| 9 | Open questions resolved: §10.1, §10.2, §10.3, §10.6, §10.7, §10.9 | 7 of 10 resolved |
| 10 | Topology type updated — BusTopology, CacheTierInfo structs | §8.2 |
| 11 | Glossary updated — Cache Tier, Named Engine, Temporal/Time-Aware, VersionedStorage, StreamLogBackend | §11 |
| 12 | otter vs samber/hot research — evaluated both libraries, justified otter choice | §5.5 comparison table |
| 13 | Existing time-based capabilities audited — LoadToTimestamp, ExecuteAsOf, VersionedStorage | §5.6 "What already exists" table |

---

## B) PARTIALLY DONE ⚠️

| # | Work item | What's done | What's missing |
|---|---|---|---|
| 1 | **Multi-bus in architecture** | D9 recorded, BusTopology type added to §8.2, YAML config in §7.2 shows buses with publish/subscribe | Topology diagrams (§6.1) show ZERO buses. Go code examples (§6.2) have NO bus config. No bus driver registry section (equivalent to §7.1 storage drivers). No consumer code showing how to subscribe to specific buses. |
| 2 | **Cache tier in diagrams** | §5.5 concept + config example | None of the 3 topology diagrams in §6.1 show cache tiers. The comparison tables (§6.3, §6.4) don't mention cache support. |
| 3 | **Time dimension in comparison tables** | §5.6 concept + engine capability table | §6.3 comparison table (current vs target) doesn't mention temporal query support. §6.4 LiveStore comparison doesn't mention time. |
| 4 | **Consumer middleware injection** | D6 states "consumer must inject domain middleware" | No Config field for middleware. No code example showing how a consumer registers handlers/projections/middleware with the System. The Config struct only has Queries + Instances + Engines. |
| 5 | **§5.2 "What's missing" list** | §10.1 resolved with StreamLogBackend | §5.2 still says "What's missing: Richer LogBackend operations" referencing the OLD LogBackend. Should cross-reference §10.1 StreamLogBackend. |
| 6 | **InstanceConfig Go struct** | Go code in §6.2 shows Role/Engine/Collections/Durability | Missing Publish/Subscribe bus fields (shown in YAML §7.2 but not in Go struct). Missing Cache field. |
| 7 | **Open question §10.1b (defaults)** | Listed as "probably (b) one per layer" | Not formally resolved as a decision. |

---

## C) NOT STARTED ❌

| # | Work item | Why it matters |
|---|---|---|
| 1 | **ADRs cut from decisions** | D1-D9 should become numbered ADRs. None exist yet. |
| 2 | **Implementation of system/ module** | The entire System type, driver registry, StreamLogBackend, adapters — zero code written. |
| 3 | **Bus driver registry design** | §7.1 covers storage drivers in detail. No equivalent for bus drivers (gochannel, nats, redis). How does a bus driver register? What's the interface? |
| 4 | **Consumer-facing API design** | The Config struct shows Instances + Engines but not how a consumer registers projections, handlers, deciders, or middleware with the System. |
| 5 | **Scream store implementation design** | §9 has the concept + architecture diagram + unsafe changes table. But no concrete API (how does the operator interact with it? what's the override mechanism?). |
| 6 | **Hot-reload mechanism design** | §7.3 says "hot-reload for additive, graceful restart for structural" but no design for the config-swap mechanism. |
| 7 | **koanf integration design** | D7 says koanf but no design for how config keys map to Go struct fields, env var naming convention, or YAML schema. |
| 8 | **Migration tooling** | D8 says gradual but no tooling design for Bundle→System migration. |

---

## D) TOTALLY FUCKED UP 💥

| # | What | Why it's wrong | Fix |
|---|---|---|---|
| 1 | **TOC has duplicate §5 entry** | Lines 32 AND 35 both list "5. The Key Insight: Multi-Instance Metaengine" — copy-paste error when adding §5.5/§5.6 sub-entries | Remove the duplicate line 35 |
| 2 | **§5.2 is stale** | Still says "What's missing: Richer LogBackend operations" referencing old LogBackend interface. §10.1 was resolved with StreamLogBackend — a completely different interface. §5.2 should point to §10.1. | Update §5.2 to cross-reference §10.1 StreamLogBackend |
| 3 | **D6-D9 NOT propagated to diagrams** | The topology diagrams (§6.1) show NO buses, NO cache tiers, NO time support — despite D6 (full infrastructure ownership), D9 (multi-bus), §5.5 (cache), §5.6 (time). The diagrams are frozen at the pre-D6 state. | Redraw §6.1 topology diagrams with buses + cache tiers |
| 4 | **Go Config struct inconsistent with YAML** | §7.2 YAML shows `publish: [local, cross-service]` and `subscribe: local` on instances. §6.2 Go Config struct has NO Publish/Subscribe fields. Two different shapes for the same concept. | Add Bus/Publish/Subscribe to the Go InstanceConfig |
| 5 | **No consumer API for the thing D6 says consumer must do** | D6 says "consumer must inject domain middleware and register handlers/projections." But there is NO code example showing this. The Config struct has no Middleware/Handlers/Projections field. The consumer experience section (§6.2) shows query declarations but not handler/middleware registration. | Add consumer-facing API section |
| 6 | **§4 intro text says "Nine architectural decisions"** | There are 9 decisions (D1-D9 in §4.1-§4.9), but the intro paragraph still references the old 5-question framing context. | Clean up §4 intro |

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements (how we worked)

1. **Edit-then-verify discipline** — I made rapid sequential edits without running a consistency check between them. Each edit should have been followed by a targeted grep/view to verify it didn't contradict an earlier section. The TOC duplication and §5.2 staleness are direct consequences.

2. **Decision propagation** — Recording a decision in §4 is only 20% of the work. The other 80% is propagating it to every diagram, code example, comparison table, and open question it affects. I consistently did the 20% and skipped the 80%.

3. **Research → design → code examples** — I researched otter/hot and time capabilities thoroughly, but didn't translate that research into concrete code examples. The cache tier has no wrapper code. The time dimension has no adapter code. The StreamLogBackend has interface code but no implementation sketch.

### Design improvements (what the document is missing)

4. **No bus driver registry** — Storage has a full driver registry section (§7.1). Buses have nothing equivalent. The bus is equally infrastructure — it needs the same `database/sql`-style registration pattern.

5. **No consumer registration API** — The System owns everything (D6), but the document never shows how the consumer registers their domain code (handlers, projections, deciders, middleware) with the System. This is the MOST IMPORTANT consumer-facing API surface and it's completely absent.

6. **No bus-to-instance wiring diagram** — How does an event instance know which bus(es) to publish to? How does a projectionhost know which bus to subscribe to? The YAML shows `publish: [local]` but the wiring logic isn't designed.

7. **No cache wrapper code** — §5.5 says "read-through wrapper" but shows zero Go code for the wrapper. The design claims it's "transparent" but doesn't show how it intercepts reads.

8. **Scream store has no operator-facing API** — §9 has the architecture and the table of unsafe changes. But how does the operator ACK a WARN+OVERRIDE? What flag? What endpoint? No design.

### Architecture improvements (what the design is missing)

9. **Multi-bus event fan-out semantics** — If events publish to both `local` (GoChannel) and `cross-service` (NATS), what guarantees ordering? What happens if NATS is down? Is the event.Store.Save blocked on bus publish?

10. **Projection-to-bus subscription model** — D6 says System owns projectionhost. But which bus does the projectionhost consume from? If there are multiple buses, does it consume from all? Just the local one? This is unspecified.

11. **Cache tier and the write path** — §5.5 says "writes go to authoritative, reads check cache first." But what about projectionhost? When the projectionhost reads events for replay, does it hit the cache? Should it? (Yes — parallel rebuilds benefit enormously, but this isn't stated.)

12. **Time dimension and snapshot interaction** — §5.6 mentions `snapshot.LoadAtVersion` exists. But the design doesn't address how snapshots interact with the new System. Are snapshots just another source-of-truth instance (Map ADT)? Do they need the StreamLogBackend?

---

## F) NEXT 50 THINGS TO DO

### Tier 1: Fix what's broken (do these FIRST)

1. Fix duplicate TOC entry (line 35 — remove duplicate §5)
2. Update §5.2 "What's missing" to reference StreamLogBackend (§10.1), not old LogBackend
3. Redraw §6.1 topology diagrams to show buses + cache tiers
4. Add Publish/Subscribe/Cache fields to the Go InstanceConfig struct in §6.2
5. Add consumer-facing API section: how consumers register handlers, projections, deciders, middleware
6. Clean up §4 intro paragraph
7. Run a full consistency audit: grep every decision (D1-D9) and verify it's reflected in every section it affects

### Tier 2: Fill design gaps

8. Design the bus driver registry (equivalent to §7.1 for buses: gochannel, nats, redis)
9. Design multi-bus event fan-out semantics (ordering, failure handling)
10. Design projectionhost bus subscription model
11. Write cache tier wrapper code (the read-through adapter)
12. Design the scream store operator-facing API (ACK mechanism, override flag)
13. Design hot-reload config-swap mechanism
14. Design koanf config key mapping (env var naming, YAML schema)
15. Add cache tier to §6.3 comparison table
16. Add temporal support to §6.3 comparison table
17. Add multi-bus to §6.3 comparison table
18. Update §6.4 LiveStore comparison with cache + time dimensions
19. Add InstanceConfig full Go struct (with ALL fields: buses, cache, durability, collections)
20. Design how snapshots fit into the instance model (separate instance? Map ADT?)

### Tier 3: Resolve remaining open questions

21. Resolve §10.1b (default instance grouping) — formally record as D10
22. Resolve §10.4 (driver registration: init() vs explicit Register())
23. Resolve §10.5 (samber/do scope boundaries — per layer? per instance?)
24. Resolve §10.8 (codec defaults — unify or per-layer?)
25. Resolve §10.10 (named engine sharing — shared pool vs isolated)

### Tier 4: Cut ADRs from decisions

26. ADR: N-instance metaengine with operator-configured grouping (from D3 + §5)
27. ADR: StreamLogBackend — one interface for all source-of-truth storage (from §10.1)
28. ADR: Cache tier — otter-based read-through for immutable events (from §5.5)
29. ADR: Time as first-class dimension (from §5.6)
30. ADR: System owns all infrastructure (from D6)
31. ADR: Multi-bus support (from D9)
32. ADR: Gradual migration path (from D8)
33. ADR: koanf config format (from D7)
34. ADR: Scream store tiered enforcement (from D5)

### Tier 5: Begin implementation (after design gaps filled)

35. Create `system/` Go module skeleton
36. Implement driver registry (storage drivers)
37. Implement bus driver registry
38. Implement StreamLogBackend on Memory engine (simplest)
39. Implement StreamLogBackend on SQLite engine
40. Implement EventAdapter (wraps StreamLogBackend as event.Store)
41. Implement CommandAdapter (wraps StreamLogBackend as command.Store)
42. Implement QueryAdapter (wraps StreamLogBackend as query.Store)
43. Implement cache tier wrapper (otter-based read-through)
44. Implement System.New() constructor
45. Implement System.Snapshot() / Health() / Plan() / Explain() / Verify()
46. Implement koanf config loader
47. Implement scream store: PlanDiff + PlanFingerprint
48. Implement scream store: pinned manifest (plan.pin.json)
49. Implement scream store: safety rules engine
50. Write system/ contract tests (integration test suite)

---

## G) Questions I CANNOT Answer Myself

### Q1: Multi-bus ordering semantics

When events fan-out to multiple buses (e.g., GoChannel + NATS), what is the
ordering guarantee?

- **Option A:** Strong ordering — event.Save blocks until ALL buses accept.
  Consistent but slow (NATS latency on every write).
- **Option B:** Local-first — Save returns after local bus publish; remote
  buses are fire-and-forget (async). Fast but remote consumers may see lag.
- **Option C:** Configurable per-instance — operator picks the consistency
  level per bus (local = sync, remote = async).

This is a business requirement, not a technical decision. I cannot determine
the right tradeoff without knowing your deployment scenarios.

### Q2: Should the System own the decider, or does the consumer?

D6 says "System owns all infrastructure." But the decider contains domain
logic (decide + fold functions) — it's the consumer's core IP. The decider
imports `event.Store` (which the System provides) but the decider itself is
domain code.

- **Option A:** Consumer constructs the decider, passes it to
  `system.RegisterDecider(deciderDef)`. System wires it to the event.Store.
- **Option B:** Consumer provides only decide + fold functions as closures.
  System constructs the decider.Repository internally.
- **Option C:** Consumer owns the decider entirely, gets the event.Store from
  the System via `system.EventStore()`.

This determines whether the System is a "full framework" or a "service
locator." I lean B (maximum ownership, minimum consumer boilerplate) but
this is a philosophical call.

### Q3: How much of the scream store should block startup vs warn?

D5 says tiered (SCREAM / WARN+OVERRIDE / ADVISORY). But the boundary between
SCREAM (hard block) and WARN+OVERRIDE is unclear for some cases:

- Removing a persistent engine: clearly SCREAM (data loss)
- Durability downgrade (Strict → Normal): SCREAM or WARN?
  Normal is still crash-safe (SQLite synchronous=NORMAL).
- Adding a volatile cache tier to a source-of-truth instance: SCREAM or WARN?
  The cache is read-only, so data is safe — but it changes the read path.

Where exactly is the line? This depends on your risk tolerance for production
deployments.

---

## Document Metrics

| Metric | Value |
|---|---|
| Total lines | 1568 |
| Decisions recorded | 9 (D1-D9) |
| Open questions | 3 unresolved (§10.1b, §10.4, §10.5, §10.8, §10.10 — wait, that's 5, not 3. I miscounted earlier.) |
| Resolved questions | 7 (§10.1, §10.2, §10.3, §10.6, §10.7, §10.9 — and §10.1b is semi-resolved) |
| Internal inconsistencies found | 6 (listed in section D) |
| ADRs cut | 0 |
| Lines of implementation code | 0 |

---

_End of report. The design document is richer but has coherence gaps. Fix Tier 1 items before continuing._


---

## Annotation (2026-08-04)

Items marked `done at <hash>` were resolved by subsequent commits. Items without markers remain open. See TODO_LIST.md for current status.
