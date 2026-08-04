# Status Report: Metaengine Redesign — Session 2026-08-04 10:00

> **Scope:** This report covers the second work round of this session only
> (after the first status report at 07:48). It audits the quality of the fixes
> applied in response to the first report's findings.
>
> **Document under review:** `docs/planning/metaengine-redesign.md` (1850 lines)

---

## Executive Summary

The second round addressed most Tier 1 issues from the first report, but
**introduced a new corruption** during the fix. The §4 intro edit merged the
section header with the intro paragraph, destroying the §4.1 heading. The
document has **0 open questions** and is architecturally complete, but has
**2 active corruptions** that need immediate fixing.

**Bottom line:** The design is now comprehensive (cache, time, multi-bus,
consumer API, bus registry, scream store API, all questions resolved) but
sloppy editing created new defects while fixing old ones.

---

## A) FULLY DONE ✅

| # | Work item | Evidence |
|---|---|---|
| 1 | Duplicate TOC entry removed | Line 35 duplicate of line 32 — gone |
| 2 | §5.2 updated to reference StreamLogBackend | Now cross-references §10.1 |
| 3 | Topology diagrams redrawn with buses + cache + System scope | §6.1 all 3 topologies show Publish/Subscribe/Cache/Buses |
| 4 | InstanceConfig Go struct has Publish/Subscribe/Cache fields | §6.2 operator code shows full struct |
| 5 | Consumer-facing API section added | §6.2 shows sys.Command(), sys.Query(), sys.Decider(), sys.UseCommandMiddleware() |
| 6 | Bus driver registry section added | §7.5 with gochannel/nats/redis, sync/async modes table |
| 7 | Cache tier wrapper code written | §7.6 CachedEventStore with otter read-through |
| 8 | Scream store operator-facing API added | §9.6 with startup check, ACK flag, tier boundary table with SQLite/Postgres/Pebble research |
| 9 | Comparison tables updated (§6.3, §6.4) | Added cache, temporal, multi-bus rows |
| 10 | Multi-bus publish model documented | §6.2 publish model table (sync local, async remote) |
| 11 | ProjectionHost consumption model documented | §6.2 explains pull (journal) vs push (bus) |
| 12 | All 10 open questions resolved | 0 `**Question:**` lines remain in §10 |
| 13 | Deep research: multi-bus ordering | EventBus internals, BlockPublishUntilSubscriberAck, dispatchLocal sequential model |
| 14 | Deep research: decider ownership | Current Repository API, taskmanager wiring, minimum consumer surface |
| 15 | Deep research: scream store boundaries | SQLite PRAGMA semantics, WAL+NORMAL safety, synchronous=OFF corruption risk |
| 16 | D10 driver registration resolved | init()-based — operator doesn't code |
| 17 | D11 scope model resolved | Hybrid: per-instance projections, per-layer source-of-truth |
| 18 | D12 codec resolved | CBOR default + per-instance override |
| 19 | D13 engine sharing resolved | Shared pool as named samber/do service |
| 20 | Scream store tier boundary table | Grounded in actual SQLite/Postgres PRAGMA research — Normal is WAL-safe, OFF corrupts |

---

## B) PARTIALLY DONE ⚠️

| # | Item | Done | Missing |
|---|---|---|---|
| 1 | **§4 intro cleanup** | Paragraph updated to "Nine architectural decisions" | The edit MERGED the intro line with the §4.1 header text, destroying it. See D-1. |
| 2 | **Decider ownership API** | §6.2 shows `sys.Decider(ctx, streamID, ...)` and `sys.Command()` | The Config struct shows `Decider: TaskDecider` but doesn't show multi-deider support (multiple aggregates). Real apps have many deciders, not one. |
| 3 | **Multi-decider registration** | Not addressed at all. The current `taskmanager` example has one decider (Task). Real systems have many (Order, User, Inventory...). The Config has a single `Decider` field, not `[]Decider` or a registration API. | Need either `Deciders: map[string]Decider` or `sys.RegisterDecider(streamType, deciderDef)`. |
| 4 | **Connection pool as samber/do named service** | §10.10 text says "Connection pools are named services in samber/do" | No design for HOW pools are configured, sized, or hot-reloaded. Just states "samber/do should help." |
| 5 | **§7.5 bus driver registry** | Section exists with 3 drivers, sync/async table | No code for how a bus driver registers (the init() function). Storage has full import examples; buses only have import stubs. |
| 6 | **Glossary** | StreamLogBackend, Cache Tier, Named Engine, Temporal added | Missing: Bus Topology, Multi-Bus, CachedEventStore, InstanceConfig, Publish Model |

---

## C) NOT STARTED ❌

| # | Item | Why it matters |
|---|---|---|
| 1 | **ADRs** | 9+ decisions (D1-D13), zero ADRs cut |
| 2 | **Implementation** | Entire `system/` module — zero code written |
| 3 | **Multi-decider design** | How does the System handle multiple aggregate types? Each has its own Decider[State]. |
| 4 | **Operator config-driven decider routing** | The operator doesn't pick deciders (that's domain logic), but the System must route commands to the right decider. How? |
| 5 | **Connection pool lifecycle** | Pools need max connections, idle timeout, hot-reload. No design. |
| 6 | **Lua scripting for operator** | Q1 answer mentioned "max LUA kind of scripting" for operators. No design for how this would work. |
| 7 | **koanf config schema** | D7 says koanf but no formal schema document (env var prefix, YAML structure validation) |
| 8 | **Snapshot-to-instance mapping** | Snapshots are mentioned in topologies but the ADT/instance model for them isn't specified. Map ADT? Separate instance? |

---

## D) TOTALLY FUCKED UP 💥

| # | What | Evidence | Fix |
|---|---|---|---|
| 1 | **§4.1 header destroyed** | The edit to clean up the §4 intro merged "Nine architectural decisions recorded below." with the §4.1 title text, producing: `Nine architectural decisions recorded below.: Hybrid (registry + config), leaning runtime`. The `### 4.1 Backend selection: Hybrid (registry + config), leaning runtime` heading is GONE. Grep for `### 4.1` returns nothing. | Reconstruct the §4.1 header |
| 2 | **§4 intro runs into §4.1 body** | Line 358: `Nine architectural decisions recorded below.: Hybrid (registry + config), leaning runtime` — the intro sentence runs directly into the first sentence of D1 with a colon. The section break is lost. | Add newline + `### 4.1` header between intro and D1 body |

---

## E) WHAT WE SHOULD IMPROVE

### Process

1. **The multiedit that broke §4.1** — I tried to simultaneously clean the §4 intro AND update the §4.1 sub-header in one edit. The old_string matched too broadly and swallowed the header. Should have done it in two separate, targeted edits. Lesson: when an edit changes text near a section header, verify the header survives.

2. **No post-edit verification** — After the §4 intro edit, I didn't run a single grep to verify `### 4.1` still existed. The corruption was invisible to me because I moved on to the next task immediately. A 1-second grep would have caught it.

3. **Config struct has a single Decider field** — This is a design oversight, not an editing error. The consumer API section shows `Decider: TaskDecider` as if there's only one aggregate type. Real systems have many. The design needs multi-decider support before implementation.

### Design gaps

4. **Multi-decider is the biggest missing concept.** The current taskmanager example has one decider. The design assumes one. But a real system has Order, User, Inventory, Task — each with its own `Decider[State]`. The Config needs either `Deciders: map[string]Decider` (keyed by stream type) or a `sys.RegisterDecider(streamType, deciderDef)` method. Commands route to deciders by stream type. This is fundamental and it's missing.

5. **Connection pool lifecycle is hand-waved.** §10.10 says "pools are named services in samber/do" but doesn't design the lifecycle: pool sizing, health checks, hot-reload of pool config, connection draining on shutdown. This is a production-critical concern.

6. **Snapshot instance model is unclear.** Topology diagrams show "snapshots" as a Map ADT collection inside a source-of-truth instance. But snapshots need `LoadAtVersion` (point-in-time read). The Map ADT doesn't support that. Either snapshots need a different ADT, or they need a specialized adapter.

7. **Operator Lua scripting** mentioned in Q1 answer but not designed. This is a significant scope item — embedding a scripting engine for operator hot-reload. Should at minimum be an ADR-pending note.

---

## F) NEXT 50 THINGS TO DO

### Tier 0: Fix what's broken (IMMEDIATE)

1. **Reconstruct §4.1 header** — restore `### 4.1 Backend selection: Hybrid (registry + config), leaning runtime`
2. **Fix §4 intro** — separate the intro paragraph from the D1 body with proper header
3. **Verify post-fix** — grep `### 4.` to confirm all 9 subsection headers exist

### Tier 1: Fill remaining design gaps

4. **Design multi-decider API** — Config needs `Deciders: map[string]Decider[State]` or `sys.RegisterDecider(streamType, deciderDef)`. Commands route by stream type.
5. **Update §6.2 consumer code** to show multi-decider registration (Task + Order example)
6. **Design connection pool lifecycle** — sizing, health, hot-reload, drain
7. **Design snapshot-to-instance mapping** — what ADT? how does LoadAtVersion work?
8. **Add Lua scripting ADR-pending note** — scope item for future, not v1
9. **Complete bus driver registration code** — init() function sketch
10. **Complete glossary** — BusTopology, CachedEventStore, InstanceConfig, PublishModel
11. **Add Config struct field reference** — document every field with type and purpose

### Tier 2: Cut ADRs

12. ADR: N-instance metaengine (D3)
13. ADR: StreamLogBackend (§10.1)
14. ADR: Cache tier — otter read-through (§5.5)
15. ADR: Time as first-class dimension (§5.6)
16. ADR: System owns all infrastructure (D6)
17. ADR: Multi-bus support (D9)
18. ADR: Gradual migration (D8)
19. ADR: koanf config (D7)
20. ADR: Scream store tiered enforcement (D5)
21. ADR: Hybrid scope model (D11)
22. ADR: CBOR default + per-instance override (D12)
23. ADR: init()-based driver registration (D10)
24. ADR: Shared connection pools as named services (D13)

### Tier 3: Formal design documents

25. **koanf config schema** — YAML structure, env var naming convention, defaults
26. **StreamLogBackend SQL schema** — table structure for stream-keyed log storage
27. **Scream store plan diff algorithm** — PlanDiff, PlanFingerprint, comparison rules
28. **Multi-bus fan-out ordering contract** — formal spec for sync vs async delivery
29. **Hot-reload mechanism** — which fields are hot-reloadable, how config-swap works
30. **Instance lifecycle state machine** — boot → running → draining → stopped

### Tier 4: Implementation prep

31. Create `system/` Go module skeleton
32. Implement driver registry (init()-based)
33. Implement bus driver registry
34. Implement StreamLogBackend on Memory engine
35. Implement StreamLogBackend on SQLite engine
36. Implement EventAdapter (wraps StreamLogBackend as event.Store)
37. Implement CommandAdapter
38. Implement QueryAdapter
39. Implement CachedEventStore (otter wrapper)
40. Implement System.New() constructor
41. Implement multi-decider routing (command → decider by stream type)
42. Implement koanf config loader
43. Implement scream store: PlanDiff + PlanFingerprint
44. Implement scream store: pinned manifest
45. Implement scream store: safety rules engine
46. Implement System.Snapshot() / Health() / Plan() / Explain() / Verify()
47. Implement connection pool lifecycle (samber/do named services)
48. Write system/ contract tests
49. Write system/ integration tests
50. Migrate example/taskmanager to System (validation)

---

## G) Questions I CANNOT Answer Myself

### Q1: Multi-decider routing model

The System owns the decider (D6), and the consumer provides `Decider[State]`
definitions. But real systems have multiple aggregate types (Task, Order, User).
How should commands route to the right decider?

- **Option A:** The consumer registers `sys.RegisterDecider("Task", taskDecider)`
  and commands carry their stream type. The System routes by stream type.
- **Option B:** The consumer provides a single `DecideFunc` that switches on
  command type — basically what taskmanager/handlers.go does today.
- **Option C:** The System doesn't route — the consumer's command handler calls
  `sys.Decider(ctx, streamID, streamType, decideFn)` explicitly, passing the
  right decide function each time.

This determines the command handler API shape. I lean A (registration by stream
type) but it's a domain-modeling decision.

### Q2: Should the Config struct separate consumer and operator concerns?

Currently `system.Config` mixes both:
```go
Config{
    Decider:   TaskDecider,          // consumer
    Commands:  registerCommands,     // consumer
    Engines:   ...,                  // operator
    Buses:     ...,                  // operator
    Instances: ...,                  // operator
}
```

Should these be separate types?
```go
system.New(ctx, system.DomainConfig{...}, system.DeploymentConfig{...})
```

This is a Go API design question that affects the import boundary.

### Q3: Snapshot storage — what ADT?

The topology diagrams show snapshots as a "Map ADT" inside a source-of-truth
instance. But snapshots need `LoadAtVersion(streamID, version)` — find the
snapshot at or below a given version. The Map ADT is key-value (streamID →
snapshot), which works for "latest snapshot" but not for "snapshot at version N."

Options:
- Map ADT with versioned keys (key = `streamID:version`) — scan backward
- A new SnapshotBackend interface (like StreamLogBackend)
- Store all snapshots, query by version range (SQL: WHERE version <= N ORDER BY version DESC LIMIT 1)

This is a storage model question I can't answer without knowing how snapshot
versioning should work in the new instance model.

---

## Document Metrics

| Metric | Value |
|---|---|
| Total lines | 1850 |
| Decisions recorded | 13 (D1-D13) |
| Open questions | 0 |
| Resolved questions | 13 |
| Active corruptions | 2 (§4.1 header destroyed, intro/body merged) |
| Design gaps | 4 (multi-decider, pool lifecycle, snapshot ADT, Lua scripting) |
| ADRs cut | 0 |
| Lines of implementation code | 0 |
| Sections | 11 |
| TOC entries | 11 (matches sections) |

---

_End of report. Fix Tier 0 items immediately — the §4.1 corruption is embarrassing._
