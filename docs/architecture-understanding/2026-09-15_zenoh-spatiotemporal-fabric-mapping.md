# Zenoh → go-cqrs-lite: Mapping the Spatiotemporal Fabric

> **Date:** 2026-09-15
> **Kind:** Point-in-time external-technology research (no code changes)
> **Sources (fetched & verified 2026-09-15):**
>
> - zenoh.io — What is Zenoh, Abstractions manual, Storage manager plugin, Regions migration guide (v1.8→v1.9), blog index, Mucalinda (1.10) release notes, NTU performance comparison (2023)
> - github.com/eclipse-zenoh/zenoh — README (v1.5.1 crate ref shown; releases up to **1.10.1**, 2026-09-07)
> - github.com/eclipse-zenoh/zenoh-go — README, go.mod, repo tree, releases (first release v1.9.0, 2026-04-13)
> - Repo state: `master` @ 526b9e8c0; every cited mechanism spot-verified with `grep` before writing.
>
> Sibling analysis: [`2026-09-10_cordis-spatiotemporal-composability-mapping.md`](2026-09-10_cordis-spatiotemporal-composability-mapping.md). This doc extends the same "spatial axis" story one level down — from data placement _within_ a process to data movement _across_ a network.

---

## 1. What Zenoh is (verified, as of 2026-09-15)

**Eclipse Zenoh** (dual-licensed EPL-2.0 / Apache-2.0, Eclipse _incubating_, designed by Angelo Corsaro — former OMG DDS spec co-chair; primary maintenance by ZettaScale + community) is a **pub/sub/query protocol that "unifies data in motion, data at rest, and computations"**.

Current version: **1.10.1 "Mucalinda"** (2026-09-07). Release cadence is healthy: 1.5 → 1.10 in ~14 months.

| Dimension   | Verified facts                                                                                                                                                                                                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Positioning | Location transparency for data **at rest** (queries routed to whichever geo-distributed storage holds the key), plus classic pub/sub location transparency for data in motion; computations via **queryables**                                                                                   |
| Scope       | One protocol from microcontroller (zenoh-pico, pure C, UDP multicast) to data-center; 4–6 bytes wire overhead                                                                                                                                                                                    |
| Topology    | client / peer / router modes, mixed at runtime; multicast + gossip scouting (auto-discovery); **regions** with gateway filters since 1.9 (topology-as-deployment-config)                                                                                                                         |
| Transports  | TCP, UDP, QUIC, TLS, WebSocket, serial, Bluetooth; Linux **io_uring** RX (~20% RTT cut, 1.10); selective **shared-memory zero-copy** per message category                                                                                                                                        |
| Storage     | Router-side **storages** = subscriber + queryable over pluggable **volumes** (memory, RocksDB, S3, InfluxDB, filesystem, ReductStore) — **latest-value KV semantics**                                                                                                                            |
| Ops surface | **Admin key space** `@/router/<id>/**`: runtime config read/write over the protocol itself (PUT/DELETE volumes & storages via REST plugin) — config-as-data                                                                                                                                      |
| QoS         | Congestion control (block/drop), priorities, express (skip batching), reliability per-sub; HLC timestamps on every value                                                                                                                                                                         |
| Ecosystem   | zenohd plugins: REST/HTTP, DDS bridge, ROS2, MQTT bridge, webserver; Wireshark dissector; adopters incl. Google, Amazon, Meta, Bosch, GM, SoftBank, NXP, Foxconn, ROS/Open Robotics, AI Institute, Stalwart                                                                                      |
| Performance | NTU/arXiv:2303.09419 (2023, v0.7 era, vendor-published): P2P ~4M msg/s small payloads, ~67 Gbps single-machine, 10 µs (64 B) single-machine RTT/2 vs Kafka 27 µs / MQTT 73 µs; DDS competitive on latency via UDP multicast. **Treat as directional, not gospel — old version, vendor-curated.** |

### The abstractions that matter here

- **Key expressions** — hierarchical `/`-separated wildcard language (`user/*/created`, `tenantA/**`) with a _canonical form guarantee_ (two expressions match the same key set ⇔ same string). This is a routing-grade topic language, not string matching.
- **Selectors** — key expression + query parameters (`path/**/x?filter=...`); the params half is application-defined.
- **Queryable** — a computation registered at a key expression, triggered by `get(selector)`. Zenoh routes the query to matching queryables anywhere in the fabric.
- **Query consolidation** — `LATEST` (one newest reply per key across replicas), `MONOTONIC`, `NONE`, `AUTO`.
- **Liveliness tokens** — cheap declarations whose lifetime is tied to the session; queryable/ subscribable (`z_liveliness`, `z_sub_liveliness` examples exist in zenoh-go).
- **Matching listeners** — notification when a remote subscriber's key expression matches your publisher (interest negotiation).
- **UHLC timestamps** — Hybrid Logical Clock: unique, happens-before-preserving, totally orderable **without consensus**. Every value gets one at the first hop.
- **1.10 timestamp instrumentation** — opt-in per-message **timestamp stack** recorded at Send / per-Route-hop / Receive: wire-level per-hop latency tracing, with custom callback hooks (pluggable clock).
- **zenoh-ext AdvancedPub/Sub** — publication cache (last-value cache on the publisher side) + querying subscriber with gap detection/recovery via per-source sequence numbers.

### Go story

`github.com/eclipse-zenoh/zenoh-go` — official, **CGo bindings over zenoh-c** (must be built with `-DZENOHC_BUILD_WITH_UNSTABLE_API=ON` for parts of the surface). First release **v1.9.0 (2026-04-13)**; in lockstep since (v1.10.1, 2026-09-07). API surface (repo tree, verified): session, config, keyexpr, link/transport, publisher, subscriber, get/query/querier/reply, **liveliness**, **matching**, source_info, scout, plus `zenohext` (AdvancedPublisher/AdvancedSubscriber, serialization). Runtime Go deps: **zero** (one Option lib). **There is no pure-Go implementation of the protocol.**

---

## 2. The thesis: Zenoh is the network-level twin of this repo's north star

> _"Developers declare ONLY Commands + Events + Queries and their relationships … while where data lives is up to operators at DEPLOYMENT time."_

Zenoh's founding pitch is literally the second half of that sentence, applied to **the network**: _"Zenoh is the first technology to bring location transparency for data at rest, allowing queries to be expressed without any concern for the actual location of data … It is Zenoh that takes care of identifying the optimal set of databases where the query should be executed."_

The Cordis mapping established that go-cqrs-lite is the **context paradigm for data placement**: the metaengine planner mediates between developer coeffect specs (query shapes) and operator engine reconciliation. Zenoh is the same paradigm at the **fabric layer**:

| Axis     | go-cqrs-lite (in-process)                                                                           | Zenoh (across processes/sites)                                                                          |
| -------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Spatial  | `DeploymentConfig.Engines` + cost planner pick engine per collection (`system/config_types.go:127`) | Topology config (client/peer/router/regions) + key-expression routing decide where data lives and flows |
| Temporal | Append-only journal + replay + tombstone/rebirth                                                    | HLC-ordered values; storages hold latest state; history via backend choice (ReductStore/RocksDB)        |
| Mediator | Cost-based planner (`metaengine/planner.go`)                                                        | The zenoh network itself (interest routing, consolidation)                                              |

A consumer composing both gets the full "spatiotemporal" stack: CQRS/event-sourcing semantics from go-cqrs-lite, geo-distribution and edge reach from Zenoh — **without the library ever growing a transport**, which ADR-0127 already forbids.

---

## 3. Concept-by-concept mapping (repo sites verified)

| Zenoh concept                          | go-cqrs-lite mechanism                                                                                                                                                                                   | Fit / verified site                                                                                                                                                                                                               |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Key expression (`user/*/created`)      | `record.StreamRef` = `"Type/EntityID"`; `event.Bus.Subscribe(eventType, …)` is **exact-type only** (`event/bus.go:27`, `watermill/event_bus.go:120`) — no wildcard tier between exact and `SubscribeAll` | **Gap-filler.** A zenoh backend gives `user.created/*`-style family subscriptions for free; `record.StreamRef.Split()` round-trips into key chunks cleanly                                                                        |
| Pub/sub (peer or routed)               | `watermill.EventBus` + `WithBackend(pub, sub, closer)` — any Watermill backend (`watermill/README.md`, ADR-0127 canonical path)                                                                          | **The natural integration seam.** Redis Streams is the only broker with a verified roundtrip suite today; NATS JetStream has _no maintained plugin_ (watermill README) — zenoh slots into exactly that hole                       |
| Queryable + consolidation `LATEST`     | `query.Dispatcher` (in-process); `metaengine.ServeSSE[V]` + `Watcher[V]` (in-process push)                                                                                                               | Speculative: serve read-model state cross-site; consolidation gives newest-replica-wins without adding consensus — matches the planner's "same answer whichever engine serves it" observational-equivalence clause                |
| Liveliness tokens                      | `ProbeEngine` / health-driven quarantine (ADR-0137): N consecutive failures → quarantine, reprobe to reactivate                                                                                          | Liveliness = **free, session-tied failure signal** that could feed/supplement probes. NOT a `claiming/` replacement: no fencing tokens/epochs (`ErrLeaseNotHeld` + `RowsAffected` fencing stays)                                  |
| UHLC timestamps                        | `record.NewStamp` (known-vs-inferred); `irohengine` LWW resolution needs cross-node timestamps                                                                                                           | HLC gives happens-before ordering without consensus — the exact primitive a `zenoh`-replicated-engine (irohengine analog) would need. Per-stream **linearization still requires** the version check (`AtomicAppender`); HLC ≠ OCC |
| 1.10 timestamp instrumentation         | `LatencyTracker` (EWMA P50/P95/P99) + `ProbeEngine` calibration (`METAENGINE-LIVE-LATENCY-MODEL.md`); `metaengine/otelobserver`                                                                          | Would measure the **fabric** (per-hop) instead of only engine ops; custom-timestamp callbacks could emit OTel-aligned records                                                                                                     |
| zenoh-ext publication cache / recovery | `watermill.CatchUpSubscriber` (journal replay + checkpoint, durable)                                                                                                                                     | Different durability tier (router RAM vs journal). Complements, does not replace: zenext for hot late-joiners, CatchUp for crash recovery                                                                                         |
| Storage volumes (latest-value KV)      | `metaengine` engines (planned tables, pushdown, vectors)                                                                                                                                                 | **Not a metaengine engine** — no scans/secondary indexes/transactions. Correct use: edge-side materialized-view cache (storage = subscriber + queryable ⇒ a projection that publishes state gets replication + serving for free)  |
| Admin key space                        | `DeploymentConfig` + `ManifestPath` plan-drift scream                                                                                                                                                    | Kindred "config reconciliation" spirit; zenoh's is runtime-mutable over the wire                                                                                                                                                  |
| Regions (1.9)                          | `EngineProfile.NetworkRTT` priors → live measurement                                                                                                                                                     | Regions are the deployment-time analog of the planner's RTT-aware geo routing; together: region topology _outside_, engine placement _inside_                                                                                     |
| MQTT / DDS / ROS2 plugins              | none (by doctrine)                                                                                                                                                                                       | Free bridges to legacy field systems on the _fabric_ side — invisible to the library                                                                                                                                              |

---

## 4. Ranked integration surfaces (Pareto)

### W1 — `watermill-zenoh` external plugin (do first)

An external sibling repo (the `go-sse` pattern, NOT in-repo — ADR-0127): construct zenoh publisher/subscriber, pass to `watermill.WithBackend` / `WithCommandBackend`. Zero go-cqrs-lite surface changes.

- Deliverable: pub/sub roundtrip test mirroring `TestRedisStreamRoundtrip`; a key-expression mapping spec (`event.Type` → key chunk, `**` for `SubscribeAll`, wildcards for stream families).
- Wins: brokerless dev topology (peer mode), geo fan-out (router mode), MQTT/DDS/ROS2 bridging, fills the NATS-JetStream-shaped hole.
- Cost: CGo isolated in the plugin repo (duckdb / `irohengine/quic` precedent); zenoh-c must build with unstable API; Nix packaging for CI.

### W2 — Command dispatch to the edge + liveliness health

CommandBus over the same fabric (commands to nodes that may be offline-ish); projection hosts announce via liveliness token; `otelobserver`-style counters on liveliness transitions. Modest, high-synergy with ADR-0137.

### W3 — Research spikes (vision-aligned, defer until W1 proves out)

1. **Queryable-served read models**: `Watcher` → queryable + `LATEST` consolidation across sites; compare vs `ServeSSE` reach.
2. **`zenohengine`**: CRDT-replicated engine wrapper over zenoh (the irohengine three-tier transport pyramid, reusing HLC instead of hand-rolled LWW clocks). Note iroh gives NAT traversal + direct P2P; zenoh gives routing topology + storages + richer query semantics — they are **complements**, pick per deployment (operator choice again).
3. **Fabric latency → planner**: timestamp-instrumentation stacks feeding `LatencyTracker`; a `Prober` implemented over zenoh round-trips.

### Non-goals (explicitly declined)

- **No in-repo `transport/zenoh` module** — ADR-0127 doctrine; the fabric is plugin territory.
- **No zenoh storage as event store or metaengine engine** — latest-value KV, no OCC, no scans; the journal and planner keep their jobs.
- **No liveliness-based leadership** — no fencing; `claiming/` + `queue/*` leases remain the coordination truth.
- **No perf-motivated migration** — Redis Streams is sufficient in-DC; zenoh's case is topology (edge/geo/brokerless), not benchmark numbers.

---

## 5. Risks & honest caveats

1. **CGo-only Go story.** zenoh-go is young (Apr 2026 first release), requires self-built zenoh-c with unstable API. Cross-compilation and Nix plumbing are real costs. Everything stays in isolated external modules per the duckdb/quic precedent.
2. **Eclipse _incubating_ + ZettaScale-centred maintenance** — governance is proper Eclipse, but bus-factor risk is worth noting before betting an edge strategy on it.
3. **Vendor-curated benchmarks** (2023, v0.7). Methodology is published (arXiv) and directionally credible, but re-verify on current versions if throughput is ever the justification.
4. **Peer-mode constraints tightened in 1.9** (peers are clique-only; meshes need routers) — topology design is router-centric for scale.
5. **Unstable-API dependence** in the Go binding (`Reliability`, source-info surfaces are flagged unstable) — pin versions; expect churn across minors.

---

## 6. Bottom line

Zenoh is the most conceptually-aligned external fabric this repo could sit on: it pushes _"where data lives is up to operators"_ from the process boundary down to the network, with abstractions (key expressions, queryables, liveliness, HLC) that map almost one-to-one onto `StreamRef`, `query.Dispatcher`, probe-health, and replication-clock needs — while its storage layer correctly stays out of the event-store/planner's way. The doctrine-compliant move is small and external: a `watermill-zenoh` plugin (W1). Everything deeper (read models over queryables, a `zenohengine`) is a natural but optional second act on the metaengine roadmap.
