# CQRS & Event Sourcing: Comprehensive Academic Literature Review & Project Implications

> **Status:** Informational deep-research report — 6/29/2026
> **Method:** Systematic survey of the peer-reviewed academic literature (IEEE, ACM, Springer, CIDR, DEBS, SIGMOD, arXiv) on CQRS and Event Sourcing, extraction of empirical findings, design principles, and benchmark data, mapped against go-cqrs-lite's verified implementation state.
> **Scope:** ~50 academic papers, theses, foundational works, and production-engineering references, spanning 1987–2026. Coverage includes concurrency control, consistency models, formal verification, saga/process-manager theory, performance benchmarks, snapshotting, schema evolution, read-your-writes, bitemporal modeling, and real-world framework architectures.
> **Companion:** `docs/research/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md` covers industry innovation trends; this report focuses on **academic findings and their engineering implications**.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The Literature (Annotated Bibliography)](#2-the-literature-annotated-bibliography)
3. [Deep Findings from Key Papers](#3-deep-findings-from-key-papers)
4. [Gap Analysis: Research vs. go-cqrs-lite State](#4-gap-analysis-research-vs-go-cqrs-lite-state)
5. [Actionable Recommendations](#5-actionable-recommendations)
6. [Anti-Patterns the Research Warns Against](#6-anti-patterns-the-research-warns-against)
7. [Cross-Cutting Insights](#7-cross-cutting-insights)
8. [Decision Matrix](#8-decision-matrix)

---

## 1. Executive Summary

The academic literature on CQRS/ES is **thinner than its industry popularity suggests** — a finding repeatedly made by the researchers themselves (Overeem 2021, Laigner 2024, Hakim 2012). The most rigorous work comes from five research streams:

1. **Utrecht University (Overeem, Jansen, Brinkkemper, Maddodi, Spoor)** — the definitive empirical ES study (25 engineers, 19 systems), schema-evolution tactics, and performance modeling.
2. **KPI Dnipro (Lytvynov & Hruzin, 2024–2026)** — the only sustained CQRS+ES architectural-variation research program: snapshot-centric CQRS, critical causal events, decision frameworks.
3. **Pat Helland (Microsoft/Amazon)** — the foundational theoretical lens: inside/outside data, entities/activities, immutability, idempotence.
4. **KTH / Formal Methods (Hakim 2012; Gomes/Kleppmann 2017; Kingsbury/Jepsen)** — the sparse but important work on formal verification and correctness of CQRS/ES and eventually-consistent systems.
5. **Distributed Systems Classics (Garcia-Molina & Salem 1987; Terry et al. 1994; Thompson/LMAX 2011; Kreps 2014)** — the foundational papers that CQRS/ES builds upon: sagas, session guarantees, mechanical sympathy, log compaction.

**Bottom line for go-cqrs-lite:** The core architecture is **strongly validated** — the Decider pattern, OCC version handling, immutability, idempotency, upcasting, tombstones, the projection-host framework, and the decider singleflight all map directly to research recommendations. However, the literature exposes **twelve concrete capability gaps**, prioritized below. The highest-leverage gaps the new research surfaced are:

- **Read-your-writes mitigation** (Terry 1994) — currently zero implementation; the #1 UX complaint in CQRS production systems.
- **Projection lag metrics** (Laigner 2024; production incident reports) — the library cannot currently tell you _how stale_ a read model is.
- **Multi-stream atomic append** (KurrentDB 25.1+) — eliminates entire classes of saga complexity.
- **Property-based verification of the store itself** (Hakim 2012; Porcupine/Jepsen lineage) — the storage layer has no rapid/linearizability tests despite the event layer having them.
- **Bitemporal modeling** (valid_time vs transaction_time) — missing entirely; needed for accounting/insurance/compliance domains.

The report also surfaces **performance benchmarks with real numbers** (the LMAX Disruptor at 6M TPS; Pececnik's 146× latency improvement at 500 users; lock-vs-CAS-vs-volatile contention costs), **saga/process-manager theory** (the Garcia-Molina original vs the modern confusion), and **consistency-model taxonomy** (CAP/PACELC, session guarantees, causal consistency) that should inform how the library documents its guarantees.

---

## 2. The Literature (Annotated Bibliography)

### 2.1 Foundational Works (Industry-Origin, Universally Cited)

| Work                                    | Author(s)                                     | Year           | Significance                                                                               |
| --------------------------------------- | --------------------------------------------- | -------------- | ------------------------------------------------------------------------------------------ |
| _CQRS Documents_                        | Greg Young                                    | 2010           | Originated CQRS; extended Meyer's CQS to architecture                                      |
| _CQRS_ (bliki)                          | Martin Fowler                                 | 2011           | Popularized CQRS; **cautionary** — "for most systems CQRS adds risky complexity"           |
| _Event Sourcing_                        | Martin Fowler                                 | 2005           | Coined "Event Sourcing"; defined rebuild, temporal query, event replay, retroactive events |
| _Clarified CQRS_                        | Udi Dahan                                     | 2009           | Precursor of CQRS (predates Young); SOA+DDD framing                                        |
| _Exploring CQRS and Event Sourcing_     | Betts et al. (Microsoft)                      | 2013           | Reference implementation on Azure; most-cited industry resource                            |
| _Disruptor_ whitepaper                  | Thompson, Farley, Barker, Gee, Stewart (LMAX) | 2011           | Real-world CQRS+ES at **6 million TPS**; mechanical sympathy                               |
| _The Log_                               | Jay Kreps                                     | 2014           | Append-only log as unifying data infrastructure; log compaction                            |
| _Versioning in an Event Sourced System_ | Greg Young                                    | 2017 (Leanpub) | Canonical versioning/upcasting reference                                                   |

### 2.2 Peer-Reviewed CQRS Papers (ACM/IEEE/Springer)

| Paper                                                           | Venue          | Year | Key Contribution                                   |
| --------------------------------------------------------------- | -------------- | ---- | -------------------------------------------------- |
| Kabbedijk et al. — _Variability Consequences of CQRS_           | EuroPLoP (ACM) | 2012 | First peer-reviewed CQRS case study                |
| Overeem et al. — _Dark Side of Event Sourcing_                  | IEEE SANER     | 2017 | Data conversion as adoption barrier                |
| Debski et al. — _Scalable Reactive Architecture_                | IEEE Software  | 2018 | Real cloud CQRS+ES (Akka/Cassandra/Kafka/Neo4j)    |
| Long — _High Performance CQRS_                                  | IEEE ICRIS     | 2017 | CQRS performance characteristics                   |
| Rajkovic — _CQRS in Medical Information Systems_                | Conf.          | 2013 | Denormalized read DBs → measurable perf gains      |
| Maddodi et al. — _Aggregate Architecture Simulation_            | ACM/SPEC ICPE  | 2020 | Layered queuing networks for CQRS perf modeling    |
| Munonye & Martinek — _Data Storage in Microservices_            | IEEE ICCSE     | 2020 | CQRS+ES standardization for microservices          |
| Laigner et al. — _Event Management Challenges in Microservices_ | **ACM TOSEM**  | 2024 | 8,000+ SO questions; 16 challenges in 5 categories |
| Mohammad — _Kafka Event-Streaming Design Patterns_              | IEEE ICICyTA   | 2025 | Synthesis of 42 studies; CQRS bus taxonomy         |

### 2.3 Peer-Reviewed Event Sourcing Papers

| Paper                                                                      | Venue                | Year | Key Contribution                                       |
| -------------------------------------------------------------------------- | -------------------- | ---- | ------------------------------------------------------ |
| **Overeem et al. — _Empirical Characterization of ES & Schema Evolution_** | **JSS**              | 2021 | **THE definitive ES study** — 25 engineers, 19 systems |
| Erb et al. — _Consistent Retrospective Snapshots_                          | NetSys               | 2017 | Snapshotting across distributed streams                |
| Meißner — _Time Travel in Distributed ES_                                  | ACM DEBS             | 2018 | Globally-consistent temporal queries                   |
| Meißner et al. — _Retro-λ_                                                 | ACM DEBS             | 2018 | Retroactive computing in serverless ES                 |
| Erb et al. — _Log Pruning in Distributed ES_                               | ACM DEBS             | 2018 | Bounded logs via pruning                               |
| Erb et al. — _Chronograph_                                                 | ACM DEBS             | 2017 | Event-sourced graphs (online + batch)                  |
| Erb — _Distributed Computing on Event-Sourced Graphs_                      | Ulm PhD              | 2020 | Synthesizes the Ulm research program                   |
| Alongi et al. — _Event-Sourced Observable Architectures_                   | Softw. Pract. Exp.   | 2022 | ES + observability experience report                   |
| Kleppmann & Kreps — _Kafka, Samza and Unix Philosophy_                     | IEEE Data Eng. Bull. | 2015 | Append-only log as general-purpose event store         |

### 2.4 CQRS/ES Theses & Dissertations

| Thesis                                                  | Institution      | Year | Key Contribution                                                 |
| ------------------------------------------------------- | ---------------- | ---- | ---------------------------------------------------------------- |
| **Hakim — _Correctness for CQRS Systems_**              | KTH Stockholm    | 2012 | **Only** dedicated CQRS formal-verification study (Promela/SPIN) |
| Nilsson & Korkmaz — _Practitioners' View on CQRS_       | Lund             | 2014 | Qualitative study; 5 adoption themes                             |
| Pececnik Almlöf & Karlsson — _CQRS vs CRUD Performance_ | KTH              | 2026 | **Concrete benchmark numbers** (146× latency improvement)        |
| Richter — _CQRS Performance in C# Web APIs_             | Halle-Wittenberg | 2024 | Quantitative load-test analysis                                  |
| Léonard — _CRUD vs DDD/CQRS/ES_                         | Liège            | 2024 | Comparative performance + qualitative study                      |
| Lytvynov & Hruzin — _Snapshot-Centric CQRS+ES_          | KPI Dnipro       | 2026 | mCQRS variant (−32% complexity)                                  |

### 2.5 Distributed Systems Classics (Foundational Theory)

| Paper                                                                         | Venue          | Year      | Relevance to CQRS/ES                                       |
| ----------------------------------------------------------------------------- | -------------- | --------- | ---------------------------------------------------------- |
| **Garcia-Molina & Salem — _Sagas_**                                           | **ACM SIGMOD** | **1987**  | The original saga paper; compensation semantics            |
| **Terry et al. — _Session Guarantees for Weakly Consistent Replicated Data_** | **PDIS**       | **1994**  | Read-your-writes, monotonic reads (Bayou/Xerox PARC)       |
| Gray, Helland, O'Neil, Shasha — _Dangers of Replication_                      | ACM SIGMOD     | 1996      | Primary-copy replication foundation                        |
| Helland — _Data on the Outside vs. Inside_                                    | CIDR           | 2005      | Inside (ACID) vs. outside (immutable) data                 |
| Helland — _Life beyond Distributed Transactions_                              | CIDR/CACM      | 2007/2017 | Entities/activities; idempotence; tentative operations     |
| Helland & Campbell — _Building on Quicksand_                                  | CIDR           | 2009      | Embrace uncertainty; semantic reconciliation               |
| Helland — _Immutability Changes Everything_                                   | CIDR/CACM      | 2015/2016 | "The truth is the log; the DB is a cache of the log"       |
| Helland — _Don't Get Stuck in the "Con" Game_                                 | CACM           | 2021      | Consistency vs. convergence vs. confluence                 |
| Kung & Robinson — _On Optimistic Methods for Concurrency Control_             | ACM VLDB       | 1981      | Theoretical basis for OCC (the "expected version" pattern) |

### 2.6 Correctness, Verification & Testing

| Work                                                                            | Venue         | Year  | Key Contribution                                     |
| ------------------------------------------------------------------------------- | ------------- | ----- | ---------------------------------------------------- |
| Gomes, Kleppmann, Mulligan, Beresford — _Verifying Strong Eventual Consistency_ | PACMPL/OOPSLA | 2017  | Isabelle/HOL machine-checked proofs for CRDTs        |
| Kingsbury & Alvaro — _Elle: Inferring Isolation Anomalies_                      | PODC/VLDB     | 2021  | Opaque-box serializability checker (Jepsen)          |
| Athalye — _Porcupine_                                                           | (engineering) | 2017+ | Fast linearizability checker in Go (etcd/TiDB)       |
| Djerou & Tibermacine — _SAGA Verification Using Maude_                          | IEEE          | 2023  | Formal verification of saga compensation correctness |

---

## 3. Deep Findings from Key Papers

### 3.1 Overeem et al. (2021) — The Definitive Empirical Study

**Method:** Constructivist grounded theory; 22 interviews, 25 engineers (103 combined years), 19 production systems (thousands to billions of events).

#### Three Degrees of Immutability

Immutability in ES is **self-imposed, not enforced** (unlike blockchain). The study observed:

- **Strict immutability** (8/19) — events never change.
- **Cut-off immutability** (3/19) — store changed at defined cut-offs; backups preserve audit trails.
- **Mutable** (8/19) — events can change; no permanent audit guarantee.

**The degree chosen determines which schema-evolution techniques are available.**

#### The Five Schema-Evolution Tactics (by adoption rate)

| Tactic                      | Engineers | Mechanism                                           | Key Liability                                  |
| --------------------------- | --------- | --------------------------------------------------- | ---------------------------------------------- |
| **Copy-and-Transform**      | **14/25** | ETL old streams → new streams; source untouched     | Slow for large stores                          |
| **Upcasting**               | **12/25** | Transform-on-read; `upcast` runs before `project`   | Accumulated chain slows loading                |
| **Weak Schema**             | **11/25** | Permissive schema; tolerant readers (Protobuf/AVRO) | Logic pollution; can't express complex changes |
| **In-Place Transformation** | 5/25      | Directly mutate stored events                       | Destroys audit trail; data-loss risk           |
| **Versioned Events**        | 2/25      | New event types; handle old+new in projection       | Logic pollution                                |

**Tactics are combined in practice.** Upcasters + copy-and-transform is the most common pairing.

#### Recommended Progression (framework guidance)

> 1. **Versioned events + weak schema** (simplest) → 2. **Upcasting** (preserves immutability) → 3. **Copy-and-transform** (when upcaster chains hurt performance) → 4. **In-place transformation** only for systems not requiring an audit log.

#### The Five Key Challenges

1. **Event system evolution** — engineers "dreaded the upgrading, had fear in advance."
2. **Steep learning curve** — 13/25: events/state-transfers thinking is fundamentally different; eventual consistency is "the biggest hurdle."
3. **Lack of available technology** — 8/25: immature frameworks; tension between "no framework needed" vs. "want mature libraries."
4. **Rebuilding projections** — slow full-history replay. Mitigations: weekend rebuilds, domain-aware skipping, targeted/partial rebuilds.
5. **Data privacy (GDPR)** — conflicts with retain-forever. Mitigations: separate personal-data store, anonymization, crypto-shredding.

#### Framework-Designer Implications

- **Upcasters as first-class** — most popular immutability-preserving technique; its main liability (accumulated perf cost) is a _framework-level_ problem solvable via caching or compiler-like chaining.
- **Copy-and-transform tooling built in** — most widely used; frameworks should provide native stream/store-level ETL.
- **Weak schema in serialization layers** — tolerant readers (Protobuf/AVRO-style).
- **Projection rebuild infrastructure** — independent, autonomous projectors; targeted/partial rebuilds; domain-aware filtering.

---

### 3.2 Lytvynov & Hruzin (2024–2026) — Architectural Variations

#### Finding A: Snapshot-Centric CQRS (mCQRS)

Making **snapshots the source of truth** (events still stored for audit) reduces component complexity:

| Metric                          | Classical CQRS | mCQRS  | Change      |
| ------------------------------- | -------------- | ------ | ----------- |
| Cyclomatic complexity (command) | 120            | 82     | **−31.67%** |
| Query response time             | 44 ms          | 44 ms  | Identical   |
| End-to-end consistency          | 268 ms         | 347 ms | +22.76%     |

**Implication:** Snapshot-as-source-of-truth is a legitimate lower-complexity entry point. Read performance is unaffected.

#### Finding B: Critical Causal Events

In async CQRS+ES with pub/sub, **events are not guaranteed in-order**. When event A _causes_ event B and they arrive out of order, the system enters intermittent, hard-to-reproduce inconsistent states. Likelihood depends on entity interdependencies.

**Best solution: "Container of Events"** — each event carries metadata about its causal predecessors (full causality history). Evaluated against synchronous queues and causal barriers; Container of Events won on integrated performance/complexity assessment.

#### Finding C: CQRS+ES Is a Family of Variations

Not one architecture but a family differing in complexity, performance, and expertise required. Migration between variations is costly if unplanned. Metrics-based decision-making replaces subjective judgment.

---

### 3.3 Laigner et al. (ACM TOSEM 2024) — Microservices Event Management

**Method:** Mined 8,000+ Stack Overflow questions; manually coded 628.

**The paradox:** Events are adopted to maximize decoupling, yet the most common challenges involve re-coupling (ordering, dependencies, synchronization).

#### The 5 Challenge Categories (16 challenges)

| Category              | Top Challenge                              | % of Total    | go-cqrs-lite Relevance |
| --------------------- | ------------------------------------------ | ------------- | ---------------------- |
| **Safety & Liveness** | CAS7: Weak delivery semantics (duplicates) | **17.20%**    | Idempotency ✅         |
|                       | CAS3: Event dependencies (cross-stream)    | 12.54%        | Causal ordering ❌     |
|                       | CAS4: In-stream ordering                   | 10.39%        | Per-stream ✅          |
|                       | CAS5: State sync via replay                | 8.96%         | ProjectionHost ✅      |
|                       | CAS2: Rollback (compensations)             | 8.24%         | Saga deferred          |
|                       | CAS1: Safe publishing (atomicity)          | 8.60%         | Outbox deferred        |
| **Schema**            | CEM1: Modeling events (fat vs thin)        | **19.71%**    | Catalog ✅             |
|                       | CEM2: Schema evolution                     | 5.02%         | Upcasting ✅           |
| **Performance**       | CP1: Processing overhead                   | 5.38%         | Snapshots ✅           |
|                       | CP2: Large payloads                        | 1.79%         | —                      |
|                       | CP3: Fluctuating rate (no backpressure)    | 2.87%         | ❌                     |
| **Observability**     | CO1: Event flow observability              | (89% of obs.) | OTel ✅                |
|                       | CO2: Auditing via replay                   | —             | Journal ✅             |
| **Security**          | CS1: Async auth                            | —             | —                      |
|                       | CS2: Data privacy (GDPR)                   | —             | Redaction ❌           |

#### Framework-Designer Recommendations

1. Provide an API to **wait for multiple dependent events** — addresses CAS3.
2. Make **delivery/publishing semantics explicit** in docs.
3. **Native observability integration** — metrics for event-processing delays.
4. **Efficient backpressure** for fast-producer/slow-consumer.

---

### 3.4 Hakim (2012) — The Only CQRS Formal-Verification Study

**This is the only dedicated academic study of CQRS correctness, and it's sobering.**

Hakim modeled a CQRS system (command bus → handler → event store → event bus → projection → read model) in **Promela** and verified it with the **SPIN model checker**.

#### Key Findings

- **The system was too complex for exhaustive verification** — state-space explosion forced **bitstate hashing** (an approximation that may miss states).
- SPIN demonstrated the _presence_ of correctness properties (deadlock-freedom, message-delivery, eventual convergence) but could not provide a mathematical _proof_.
- Hakim proposed **probabilistic bounded staleness (PBS)** as a metric for quantifying eventual consistency: `P(read_stale(Δt)) < ε`.
- The reusable approach: **specification-implementation conformance** — if a real implementation conforms to the verified Promela message-passing protocol, it inherits the properties.

#### Consistency Conditions Identified

| Condition            | LTL Expression                               | Meaning                                                               |
| -------------------- | -------------------------------------------- | --------------------------------------------------------------------- |
| Eventual convergence | `□(event_published → ◇(read_model_updated))` | "Always, if an event is published, eventually the read model updates" |
| Deadlock-freedom     | SPIN invalid-end-state check                 | No terminal states except intended quiescent                          |
| Message fidelity     | Channel semantics                            | No dropped/duplicated messages                                        |

#### Why This Matters for go-cqrs-lite

- **No direct follow-up work exists** — CQRS verification remains an open problem 14 years later.
- **The storage layer has no property-based/linearizability tests** — the `event/` module has rapid tests, but `storage/` does not. This is exactly the gap Hakim's framework would address empirically.
- Tools like **Porcupine** (Go linearizability checker, used by etcd/TiDB) make Jepsen-style testing practical for a Go library.

---

### 3.5 Garcia-Molina & Salem (SIGMOD 1987) — The Original Saga Paper

**The most important distinction the academic literature makes that the industry has forgotten:**

A **saga** (in the original 1987 sense) is a **compensation strategy** — break a long-lived transaction into sub-transactions, each with a compensating transaction. The guarantee: all complete, or all are compensated.

What most modern frameworks (Axon, NServiceBus, MassTransit) call a "saga" is actually a **process manager** (Enterprise Integration Patterns) — a stateful component that routes messages based on current state.

| Feature             | Saga (Garcia-Molina)       | Process Manager (EIP)       |
| ------------------- | -------------------------- | --------------------------- |
| **State**           | Stateless or minimal       | Stateful state machine      |
| **Decision basis**  | Event data only            | Event data + current state  |
| **Coordination**    | Choreography (distributed) | Orchestration (centralized) |
| **Primary purpose** | Compensation strategy      | Workflow coordination       |

**go-cqrs-lite's decision to remove the saga module (ADR-0004 superseded) and favor emergent orchestration (projection + command dispatch + `deriver/`) aligns with the choreography/saga-original-meaning approach.** This is validated by the literature — the project should document _why_ explicitly.

#### Compensation Semantics (Critical)

> "The compensating transaction undoes, from a semantic point of view, any of the actions performed by Tᵢ, but **does not necessarily return the database to the state that existed when the execution of Tᵢ began**."

Compensation **adds a new forward-moving action** rather than reverting. This is exactly how event sourcing handles it: `OrderCancelled` is a new event, not a deletion of `OrderConfirmed`.

---

### 3.6 Terry et al. (PDIS 1994) — Session Guarantees

The foundational paper for the **read-your-writes problem** that plagues every CQRS system. From the Bayou project at Xerox PARC.

#### The Four Session Guarantees

| Guarantee               | Definition                                                           | CQRS Relevance                                                   |
| ----------------------- | -------------------------------------------------------------------- | ---------------------------------------------------------------- |
| **Read-Your-Writes**    | If Read R follows Write W in a session, W is included in what R sees | **The #1 UX problem in CQRS** — users don't see their own writes |
| **Monotonic Reads**     | Once you see version N, you never see an older version               | Prevents data "coming and going"                                 |
| **Monotonic Writes**    | Writes propagate in order                                            | Prevents version N+1 before N                                    |
| **Writes-Follow-Reads** | Writes ordered after writes whose effects were seen                  | Causal dependency preservation                                   |

#### Implementation Mechanism

Implemented via **version vectors** — a per-session read-vector and write-vector. Before each read, check the server's version vector dominates the session's write-vector (for RYW).

> "The main cost of requesting session guarantees is a **potential reduction in availability** compared to a basic read-any/write-any replication scheme."

**go-cqrs-lite has zero read-your-writes mitigation** — no version tokens, no projection-catch-up waiting, no session-stickiness. This is the most impactful gap for user-facing systems.

---

### 3.7 LMAX Disruptor (2011) — Performance Engineering at the Limit

The Disruptor is the proof that event-sourced systems can achieve extreme performance. Key numbers:

#### Concrete Performance Data

| Metric                          | ArrayBlockingQueue | Disruptor       | Improvement |
| ------------------------------- | ------------------ | --------------- | ----------- |
| Throughput (1P-1C)              | 5.3M msg/s         | **26M msg/s**   | ~5×         |
| Throughput (1P-3C pipeline)     | 2.1M msg/s         | **16.8M msg/s** | ~8×         |
| Mean latency (3-stage pipeline) | 32,757 ns          | **52 ns**       | **~630×**   |
| P99 latency                     | 2,097,152 ns       | **128 ns**      | ~16,000×    |

LMAX production: **6 million orders/second** on a single thread (3 GHz Nehalem, 32GB RAM).

#### The Contention Cost Hierarchy (critical for ES library design)

Incrementing a 64-bit counter 500M times:

| Method                         | Time (ms)   | Relative |
| ------------------------------ | ----------- | -------- |
| Single thread (no sync)        | 300         | 1×       |
| Single thread (volatile write) | 4,700       | 16×      |
| Single thread (CAS)            | 5,700       | 19×      |
| Single thread (lock)           | 10,000      | 33×      |
| Two threads (CAS)              | 30,000      | 100×     |
| **Two threads (lock)**         | **224,000** | **747×** |

**Implication for go-cqrs-lite:** The library already uses `atomic.Int32` for circuit-breaker fast paths and pre-computed middleware chains. The Disruptor data validates this approach and shows the cost ceiling of lock-based contention. The `decider.Repository` singleflight and lock-free EventBus publish path are the right patterns.

#### Cache Line / False Sharing

CPUs access memory in **64-byte cache lines**. If two independent variables share a cache line and are written by different cores, **false sharing** causes each write to invalidate the other's cache. The Disruptor **pads sequence counters to occupy their own cache line**.

**Implication:** The project's ADR-0020 (performance optimization) and the `CQRSHistogramBoundaries` are aligned. Any hot struct with concurrent fields should consider padding — though Go's escape analysis and the garbage collector make this less critical than in Java.

---

### 3.8 Performance Benchmarks — Concrete Numbers

Extracted from peer-reviewed studies and theses:

| Study                               | Metric       | CRUD             | CQRS                    | Improvement        |
| ----------------------------------- | ------------ | ---------------- | ----------------------- | ------------------ |
| **Pececnik (500 users)**            | Avg latency  | 768.64 ms        | 5.26 ms (separated DBs) | **~146×**          |
| **Pececnik (500 users)**            | P99 latency  | 1,885 ms         | 12 ms                   | **~157×**          |
| **Pececnik (500 users)**            | Throughput   | 197.95 TPS       | 298.92 TPS              | **~51%**           |
| **Pececnik (350 users)**            | Avg latency  | 142.87 ms        | 5.52 ms                 | ~26×               |
| **Richter (GetChildrenOrTopLevel)** | p95 latency  | ~150 ms          | ~20 ms                  | ~7.5×              |
| **Richter (GetCategoryMapping)**    | p95 latency  | ~9.75 ms         | ~10.25 ms               | CQRS **5% slower** |
| **Adiputra**                        | Bulk op time | 0.073 s          | 0.053 s                 | ~27% faster        |
| **Adiputra**                        | Read latency | 0.089 s (direct) | 0.101 s (ES read model) | ES **13% slower**  |

#### Critical Nuance from the Benchmarks

- **Logical CQRS separation (shared DB) shows negligible improvement** over CRUD (Pececnik). The win comes from **physically separated read/write stores** — confirming the value of the project's multi-DB presets (`WithEventDB`, `WithViewDB`).
- **For 6 of 9 endpoints, CQRS showed no performance difference** (Richter). CQRS is not a universal performance win — it's a **read-scalability and domain-complexity** win.
- **ES read models are ~13% slower than direct DB queries** (Adiputra) — the projection overhead is real but small.

---

### 3.9 Kafka Log Compaction — The Pruning Solution

Kreps's "turning the log upside-down": retain **at least the latest value for every key** while discarding superseded updates. Enables Kafka to serve as a **persistent event store with bounded storage**.

| Aspect           | Retention (delete)           | Compaction (compact)                           |
| ---------------- | ---------------------------- | ---------------------------------------------- |
| What's removed   | Whole segments past age/size | Superseded records (older values for same key) |
| Restorable state | Only within retention window | **Full current state (latest per key)**        |
| Best for         | Transient event streams      | Event sourcing, state stores                   |
| Tombstones       | N/A                          | Null payload = mark for deletion               |

**Implication for go-cqrs-lite:** This is the academically-validated model for the project's deferred log-pruning feature (G6). Rather than deleting events (which destroys time-travel), **compaction** (Kafka-style) preserves the latest state per aggregate while bounding storage. The project's snapshot store is conceptually similar but not yet wired as a compaction mechanism.

---

### 3.10 Consistency Models — CAP, PACELC, and the Trade-off Space

**PACELC** (Abadi 2010) extends CAP: _"During Partition, choose A vs C; Else, choose Latency vs Consistency."_

| System             | On Partition | Normal (Else)           |
| ------------------ | ------------ | ----------------------- |
| DynamoDB (default) | PA           | EL (low latency, weak)  |
| Cassandra          | PA           | EL                      |
| MongoDB            | PA/PC        | EC                      |
| Spanner            | PC           | EC (TrueTime offsets L) |

**A CQRS system with separate write/read models is fundamentally an EL architecture** under normal operation — it trades consistency for latency. The read-your-writes problem is the direct consequence.

**Implication:** The library should **document its consistency guarantees explicitly** — it provides per-stream ordering (FIFO within an aggregate) but no cross-stream ordering guarantee and no read-your-writes. Consumers need to know this, per Laigner's recommendation #2 ("make delivery/publishing semantics explicit").

---

### 3.11 Bitemporal Event Sourcing — Valid-Time vs Transaction-Time

Two orthogonal time axes:

- **Valid Time** (business time): when the fact _held_ in reality.
- **Transaction Time** (system time): when the system _recorded_ the fact.

This enables:

- "What did we _believe_ the state was on date X?" (as-of transaction time)
- "What _was_ the true state on date X?" (as-of valid time, including corrections)
- "What will the state be on future date X?" (future-dated events)

Events carry `valid_at` metadata distinct from their recording `timestamp`. Corrections are **new events** with `valid_at` pointing backward — the original is never mutated.

**go-cqrs-lite has zero bitemporal support** — the only time dimension is `OccurredAt` (transaction time). This matters for accounting, insurance, and compliance domains where "when was this true?" differs from "when did we learn it?"

---

### 3.12 Concurrency Control in Event Stores — The OCC Foundation

All major event stores (EventStoreDB, Axon, NEventStore, Eventuous) use **optimistic concurrency control (OCC)** at the stream level, rooted in Kung & Robinson (1981).

#### The "Expected Version" Pattern

1. **Read phase**: client reads stream at version N.
2. **Compute phase**: computes new events — no locks held.
3. **Commit phase**: appends with `expectedVersion=N`. Store atomically checks; if still N, append succeeds. If N+1, **rejected**.

#### Special Version Values (KurrentDB model)

| Value             | Meaning                         |
| ----------------- | ------------------------------- |
| `Any`             | Never conflict — always succeed |
| `NoStream`        | Stream must not exist yet       |
| Specific revision | Must be at exactly that version |

#### First-Writer-Wins + Retry

When two writers race, **first writer wins**; the loser receives `ErrVersionConflict` and must retry (re-load, re-evaluate, re-append). This is exactly the pattern the project implements in `decider.Repository.Execute`.

**go-cqrs-lite validates this correctly**: `Save(ctx, ref, events, expectedVersion)` with `SharedCheckVersion` and `ErrVersionConflict` (`storage/sql/helpers.go:178`). The decider's load-fold-decide-save loop IS the OCC retry loop.

#### NEventStore's Commit Model (alternative)

NEventStore shifts the atomic unit from individual event to **commit** (batch), using a monotonically-increasing `CommitSequence`. This enables storage-engine portability (any SQL/NoSQL) and is simpler than per-event versioning. The project's `AppendBatch` is conceptually similar.

---

### 3.13 Real-World Framework Comparison — Validated Patterns

Synthesized from EventStoreDB/KurrentDB, Axon, Eventuous, NEventStore, RailsEventStore, and Marten:

#### Design Choices Validated by Production Use

| Principle                                                | Validated By                                            | go-cqrs-lite Alignment      |
| -------------------------------------------------------- | ------------------------------------------------------- | --------------------------- |
| **Avoid 2PC at all costs** — co-locate or async-dispatch | NEventStore, Eventuous, Marten                          | ✅ ADR-0016 declined outbox |
| **Upcasting at read time, never modify stored events**   | Axon (IntermediateEventRepresentation), all frameworks  | ✅ `schema/` module         |
| **Separate processing infra from handler logic**         | Axon (3-tier), Eventuous, Marten                        | ✅ `projectionhost/`        |
| **Claim-based checkpoint for distributed processing**    | Axon (token claim/steal), Marten (advisory locks)       | ❌ Single-process only      |
| **Immutable aggregate state**                            | Eventuous                                               | ✅ Decider pure functions   |
| **Configurable error handling per mode**                 | Marten (SkipApplyErrors), Axon                          | ✅ DLQ + retry config       |
| **Storage-engine abstraction**                           | NEventStore (commit model), Eventuous (IReader/IWriter) | ✅ Sink/Source split        |

#### Marten's Three Projection Lifecycles (innovative)

| Lifecycle  | When                              | Consistency    | go-cqrs-lite Equivalent |
| ---------- | --------------------------------- | -------------- | ----------------------- |
| **Inline** | Same transaction as event capture | Strong         | ❌ Not supported        |
| **Live**   | On-demand, in-memory              | Always current | ✅ `decider.Load`       |
| **Async**  | Background daemon                 | Eventual       | ✅ `projectionhost/`    |

**Marten's Inline projections are notable** — they update the read model in the _same transaction_ as the event append, eliminating projection lag entirely for those projections. This is the strongest possible read-your-writes guarantee. go-cqrs-lite's storage layer supports this (same `*sql.DB`), but no projection primitive exploits it.

---

## 4. Gap Analysis: Research vs. go-cqrs-lite State

I verified the project's implementation state against all research findings. Below: **what's validated** and **what's missing**.

### 4.1 What the Project Does RIGHT (validated by research)

| Research Recommendation               | Implementation                                                  | Source                               |
| ------------------------------------- | --------------------------------------------------------------- | ------------------------------------ |
| Pure functions (Decider)              | `decider.Decider[State]`, ADR-0001                              | Overeem; Erb/Meißner                 |
| Upcasting as first-class              | `schema/` module                                                | Overeem: 12/25 engineers             |
| OCC version handling                  | `Save(ctx, ref, events, expectedVersion)`, `ErrVersionConflict` | Kung & Robinson 1981; all frameworks |
| Idempotency for at-least-once         | `idempotency/` module                                           | Helland #4; Laigner CAS7             |
| Aggregate = transaction boundary      | Decider + Repository                                            | Helland #2                           |
| Tombstone soft-delete                 | ADR-0005                                                        | Laigner CS2                          |
| Sink/Source ISP split                 | ADR-0006                                                        | Helland inside/outside               |
| Replay mode flag                      | `ProcessingMode`, `IsReplay(ctx)`                               | Helland #10; Fowler                  |
| Command→event causation               | `Causation`, `CommandCausalityEnricher`                         | Lytvynov                             |
| Immutable events + defensive clones   | `*ImmutableEvent`                                               | Helland #1, #3                       |
| Branded IDs (stable identity)         | `id.Of[T]`                                                      | Helland: stable outside data         |
| Event catalog                         | `catalog/` AsyncAPI/OpenAPI                                     | Laigner CEM1                         |
| Managed projection host               | `projectionhost/`                                               | Overeem Challenge 4                  |
| Scheduled commands                    | `scheduling/`                                                   | Garcia-Molina activities             |
| OTel observability                    | `otel/`, `prometheus/`                                          | Laigner CO1                          |
| Scenario-testing DSL                  | `scenario/`                                                     | Overeem Challenge 2                  |
| Library-not-framework                 | 52 modules, go.work                                             | Overeem Challenge 3                  |
| Lock-free hot paths                   | `atomic.Int32`, pre-computed chains                             | LMAX contention hierarchy            |
| Saga removal (emergent orchestration) | ADR-0004 superseded                                             | Garcia-Molina original meaning       |

**Verdict:** The core architecture is _exemplary_ against the literature.

---

### 4.2 What the Research Exposes as MISSING (12 actionable gaps)

| #       | Research Finding                                                             | Current State                                     | Effort   | Impact                       |
| ------- | ---------------------------------------------------------------------------- | ------------------------------------------------- | -------- | ---------------------------- |
| **G1**  | **Copy-and-Transform (event-store ETL)** — 14/25 engineers                   | MISSING                                           | Medium   | **HIGH**                     |
| **G2**  | **Event→event causal tracking + causal barriers** — "Container of Events"    | PARTIAL (cmd→event only)                          | Med-High | **HIGH**                     |
| **G3**  | **GDPR redaction / crypto-shredding**                                        | MISSING (design only)                             | Low-Med  | **MED-HIGH**                 |
| **G4**  | **Auto side-effect suppression during replay**                               | PARTIAL (flag only)                               | Low      | **MEDIUM**                   |
| **G5**  | **Snapshot strategy diversity** (read-pressure, time-based)                  | PARTIAL (EveryNEvents only)                       | Low-Med  | **MEDIUM**                   |
| **G6**  | **Log pruning / compaction** (Kafka-style)                                   | MISSING (deferred)                                | High     | MEDIUM                       |
| **G7**  | **Projection rebuild primitives** (domain-aware skip, parallel)              | PARTIAL                                           | Medium   | **MEDIUM**                   |
| **G8**  | **Read-your-writes mitigation** (version tokens, catch-up wait) — Terry 1994 | **MISSING**                                       | Medium   | **HIGH**                     |
| **G9**  | **Projection lag metrics** (how stale is the read model?)                    | **MISSING**                                       | Low      | **HIGH**                     |
| **G10** | **Multi-stream atomic append** (KurrentDB 25.1+)                             | **MISSING**                                       | Medium   | **HIGH**                     |
| **G11** | **Property-based store verification** (Porcupine-style linearizability)      | **MISSING** (event/ has rapid; storage/ does not) | Medium   | **MEDIUM**                   |
| **G12** | **Bitemporal modeling** (valid_at vs transaction_time)                       | **MISSING**                                       | Medium   | **MEDIUM** (domain-specific) |

---

## 5. Actionable Recommendations

Ordered by impact-to-effort ratio, incorporating the new research.

### P1 — Read-Your-Writes Mitigation (closes G8) — _Highest UX impact_

**Why:** Terry et al. (1994) is the foundational paper, and Laigner (2024) confirms this is a top production pain. Every CQRS system without RYW mitigation generates "the action didn't work" support tickets and double-submits. The library currently has **zero** mitigation.

**Proposal (layered):**

```go
// Option A: Version-token check (Terry's write-vector approach)
result, _ := repo.Execute(ctx, aggID, decider, cmd)
// result.EventVersion = 42 (the version after the write)

// On subsequent read, check the projection has caught up:
ok := readModel.IsCaughtUp(ctx, aggID, result.EventVersion)
if !ok {
    // poll, wait, or return optimistic data
}

// Option B: WaitForProjection helper
view, _ := projectionHost.WaitForProjection(ctx, aggID, result.EventVersion, 2*time.Second)

// Option C: Inline projection (Marten-style, strongest guarantee)
// Project within the same transaction as the event append:
store.SaveWithProjection(ctx, ref, events, expectedVersion, inlineProjection)
```

**Research backing:** Terry et al. (1994) RYW; Marten Inline projections; production incident reports (5-second projection lag causing double-spend).

---

### P2 — Projection Lag Metrics (closes G9) — _Quick win, high impact_

**Why:** The library cannot currently tell you _how stale_ a read model is. Laigner's CO1 (89% of observability challenges) demands exactly this. Production incident reports show silent drift from seconds → hours. Already listed as a TODO in the project's own brutal self-review.

**Proposal:**

```go
// projectionhost exposes lag via Status() and OTel metrics
status := host.Status()
for _, s := range status {
    fmt.Printf("%s: lag=%d events behind\n", s.Name, s.Lag)  // journal_tail - checkpoint
}
// OTel: cqrs.projection.lag (Int64ObservableGauge), cqrs.projection.dlq.depth
```

Implementation: compare checkpoint position against journal tail (`ReadAll` count). ~50 lines.

**Research backing:** Laigner (2024) CO1; production SLO thresholds (dashboards <100ms, search <1s, analytics <5min).

---

### P3 — Copy-and-Transform Tooling (closes G1) — _Highest leverage for schema evolution_

**Why:** The single most-used schema-evolution technique in industry (14/25 engineers). Upcasting is the right day-to-day tool (already shipped), but teams periodically need to "clean up" accumulated upcaster chains by rewriting the store.

**Proposal:** A `migrate/` module:

```go
migrator := migrate.New(sourceStore, targetStore,
    migrate.WithBatchSize(1000),
    migrate.WithTransform(upcasterChain),
    migrate.WithFilter(skipObsoleteEvents),
    migrate.WithProgress(reporter),
)
err := migrator.Run(ctx)
```

Composes existing primitives (`schema.Upcaster`, `event.Journal.ReadAll`, `event.Store.AppendBatch`).

**Research backing:** Overeem (2021) §7 Tactic 5.

---

### P4 — Multi-Stream Atomic Append (closes G10) — _Eliminates saga complexity_

**Why:** KurrentDB 25.1+ ships this. It eliminates the need for sagas/process-managers in many cross-aggregate scenarios (write events to multiple streams atomically). The project's SQL backend already uses a single `*sql.DB` — the transaction boundary already exists.

**Proposal:**

```go
// Append events to multiple aggregate streams in one transaction.
err := store.AppendMulti(ctx, []event.StreamAppend{
    {Ref: orderRef, Events: orderEvents, ExpectedVersion: 5},
    {Ref: inventoryRef, Events: inventoryEvents, ExpectedVersion: 12},
})
```

**Research backing:** KurrentDB multi-stream appends; Helland entities/activities; the aggregateless ES research (CTE-based atomic check-and-insert).

---

### P5 — Event→Event Causality + Causal Barriers (closes G2) — _Highest research signal_

**Why:** Lytvynov (2024) shows out-of-order causal events cause intermittent, hard-to-reproduce bugs. Laigner CAS3 (12.54%) and CAS4 (10.39%) are top-5 industry pain points.

**Proposal (phased):**

1. Extend `event.Metadata.Causation` with optional `CausedByEventID id.EventID`.
2. A `causality.Barrier` projection wrapper — buffers events until causal predecessor processed.
3. Optional `graph/`-backed causation DAG viewer.

**Research backing:** Lytvynov & Hruzin (2024); Laigner CAS3/CAS4.

---

### P6 — Replay Side-Effect Gating (closes G4) — _Quick safety win_

**Why:** Helland #10 and Fowler insist side effects MUST be suppressed during replay. The library exposes `IsReplay(ctx)` but consumers must remember to check.

**Proposal:**

```go
sendEmail := replay.Gate(ctx, func(ctx context.Context) error {
    return emailer.Send(ctx, msg)
})
```

**Research backing:** Helland #10; Fowler §External Updates.

---

### P7 — GDPR Redaction / Crypto-Shredding (closes G3) — _Regulatory necessity_

**Why:** Overeem Challenge 5; Laigner CS2. Crypto-shredding extends existing `encryption/`.

**Proposal:**

```go
redactor := redaction.NewCryptoShredder(keyStore)
bus.UsePublish(redaction.RedactMiddleware(redactor, redaction.WithFields("ssn", "email")))
// "Forget" a subject: keyStore.Destroy(subjectID) → PII becomes unreadable ciphertext.
```

**Research backing:** Overeem (2021) Challenge 5; Laigner (2024) CS2.

---

### P8 — More Snapshot Strategies (closes G5) — _Low-effort breadth_

**Proposal:** Add `TimeBasedStrategy`, `ReadPressureStrategy` (design exists), `CompositeStrategy`. Each ~30 lines.

**Research backing:** Lytvynov mCQRS; Maddodi ICPE 2020.

---

### P9 — Property-Based Store Verification (closes G11) — _Correctness assurance_

**Why:** Hakim (2012) shows CQRS verification is possible but hard. The `event/` module has rapid tests; `storage/` does not. Porcupine (Go) makes linearizability checking practical.

**Proposal:** Add rapid-based property tests to `storage/`:

```go
func TestStoreLinearizability(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random command sequences: Save, Load, AppendBatch
        // Verify: concurrent Saves produce consistent versions
        // Verify: OCC conflicts are correctly detected
        // Verify: no lost appends under -race
    })
}
```

**Research backing:** Hakim (2012); Porcupine; Jepsen lineage.

---

### P10 — Projection Rebuild Primitives (closes G7)

**Proposal:** `WithEventFilter`, `WithParallelism`, `Rebuild(ctx, name)` with progress reporting.

**Research backing:** Overeem Challenge 4.

---

### P11 — Bitemporal Modeling (closes G12) — _Domain-specific_

**Why:** Accounting, insurance, compliance domains need "when was this true?" vs "when did we learn it?"

**Proposal:** Add `valid_at` to `event.Metadata`; extend `LoadToTimestamp` to filter on valid-time axis. Events are never mutated — corrections are new events with backward-pointing `valid_at`.

**Research backing:** Bitemporal modeling literature; Arkency RailsEventStore implementation.

---

### P12 — Log Pruning / Compaction (closes G6) — _Defer until scale_

**Recommendation:** **Keep deferred.** Kafka-style compaction (retain latest per key) is the right model if ever needed — it bounds storage without destroying current-state reconstruction. Conflicts with full time-travel; defer until scale demands.

**Research backing:** Erb et al. (2018); Kreps log compaction; Overeem cut-off immutability.

---

## 6. Anti-Patterns the Research Warns Against

The literature is rich with cautions. These should inform both the library's design and its documentation:

| Anti-Pattern                                                           | Source                               | Consequence                                           |
| ---------------------------------------------------------------------- | ------------------------------------ | ----------------------------------------------------- |
| **Assuming exactly-once delivery**                                     | Helland; Laigner CAS7 (17.2%)        | Duplicates, double-charges                            |
| **Hiding every inconsistency behind "eventual consistency"**           | Production incident reports          | Users don't understand "eventual"; double-submits     |
| **Sending side effects during replay**                                 | Helland #10; Fowler                  | Duplicate emails/payments on every projection rebuild |
| **Mutating stored events** (in-place transform without opt-in)         | Overeem                              | Destroyed audit trail; data-loss risk                 |
| **Assuming cross-stream ordering**                                     | Lytvynov; project's own docs         | Intermittent, unreproducible bugs                     |
| **Using locks for hot-path coordination**                              | LMAX contention hierarchy            | 747× slower than uncontended                          |
| **Building CQRS without measuring projection lag**                     | Laigner CO1; production reports      | Silent drift from seconds → hours                     |
| **Confusing saga (compensation) with process manager (orchestration)** | Garcia-Molina 1987 vs EIP            | Wrong abstraction, over-engineering                   |
| **Logical-only CQRS separation (shared DB) expecting perf wins**       | Pececnik benchmarks                  | Negligible improvement; need physical separation      |
| **Treating immutability as boolean**                                   | Overeem 3-tier model                 | Wrong constraints on schema evolution options         |
| **2PC across services**                                                | Helland; NEventStore; all frameworks | Fragility, availability loss                          |

---

## 7. Cross-Cutting Insights

1. **Fowler's caution is empirically validated.** CQRS "adds risky complexity." **The library's job is to absorb that complexity** — every "consumer must hand-roll X" is a failure. The gaps (especially G8, G9) are places complexity still leaks.

2. **Immutability is a spectrum.** Overeem shows only 8/19 systems are strict-immutable. Make the choice explicit — strict by default, with documented opt-in to cut-off/mutable modes.

3. **The decoupling paradox (Laigner).** Events decouple, but top challenges are re-coupling (ordering, dependencies). Causal-barrier work (P5) serves the most painful part.

4. **Performance is modelable.** Maddodi (ICPE 2020) shows CQRS performance is predictable via queuing networks. The project could ship a benchmark harness predicting rebuild costs.

5. **The LMAX lesson: mechanical sympathy matters.** Lock-free fast paths, cache-line awareness, pre-computed chains — the project already does this (ADR-0020). The contention cost hierarchy (747× for contended locks) validates it.

6. **Sagas are compensation, process managers are orchestration.** The project's decision to remove the saga module and favor emergent orchestration is _aligned with the original Garcia-Molina definition_. Document this explicitly.

7. **Read-your-writes is not optional for user-facing systems.** Terry's 1994 paper is 30 years old and still the definitive reference. The library should ship at least one mitigation.

8. **Verification is possible but hard.** Hakim showed CQRS can be model-checked but state-space explosion forces approximation. Property-based testing (rapid) is the pragmatic middle ground the project should adopt for storage/.

---

## 8. Decision Matrix

| Recommendation                          | Effort   | Impact   | Research Consensus                  | Verdict              |
| --------------------------------------- | -------- | -------- | ----------------------------------- | -------------------- |
| **P1** Read-your-writes mitigation      | Medium   | **HIGH** | Strong (Terry, Laigner, production) | **DO**               |
| **P2** Projection lag metrics           | Low      | **HIGH** | Strong (Laigner CO1)                | **DO first**         |
| **P3** Copy-and-Transform               | Medium   | **HIGH** | Strong (Overeem 14/25)              | **DO**               |
| **P4** Multi-stream atomic append       | Medium   | **HIGH** | Strong (KurrentDB, Helland)         | **DO**               |
| **P5** Event→event causality + barriers | Med-High | **HIGH** | Strong (Lytvynov, Laigner)          | **DO (phased)**      |
| **P6** Replay side-effect gating        | Low      | MEDIUM   | Strong (Helland #10)                | **DO**               |
| **P7** GDPR crypto-shredding            | Low-Med  | MED-HIGH | Strong (Overeem, Laigner)           | **DO**               |
| **P8** More snapshot strategies         | Low      | MEDIUM   | Moderate                            | **DO**               |
| **P9** Property-based store tests       | Medium   | MEDIUM   | Moderate (Hakim)                    | **DO**               |
| **P10** Projection rebuild primitives   | Medium   | MEDIUM   | Moderate (Overeem)                  | **Do when capacity** |
| **P11** Bitemporal modeling             | Medium   | MEDIUM   | Moderate (domain-specific)          | **Do when needed**   |
| **P12** Log pruning / compaction        | High     | MEDIUM   | Moderate (Erb, Kreps)               | **DEFER**            |

**Net:** Eight "DO" items (P1–P8), two "do when capacity" (P9–P10), one "do when needed" (P11), one "defer" (P12). **P2 (projection lag metrics) is the highest impact-to-effort** and should ship first — it's ~50 lines, already a known TODO, and addresses the #1 observability gap.

---

## References (Selected, by importance)

### Primary Academic Sources

1. Overeem et al. _Empirical Characterization of ES & Schema Evolution._ JSS 178, 2021. arXiv:2104.01146
2. Lytvynov & Hruzin. _Snapshot-Centric CQRS+ES._ KPI Science News 142(1), 2026.
3. Lytvynov & Hruzin. _Critical Causal Events in CQRS+ES._ Radio Elec. CS Control 3, 2024.
4. Laigner et al. _Challenges of Event Management in Microservices._ ACM TOSEM, 2024.
5. Hakim. _Correctness for CQRS Systems._ KTH, 2012. DiVA diva2:654390
6. Garcia-Molina & Salem. _Sagas._ ACM SIGMOD, 1987. DOI 10.1145/38713.38742
7. Terry et al. _Session Guarantees for Weakly Consistent Replicated Data._ PDIS, 1994.
8. Thompson et al. _Disruptor: High Performance Alternative to Bounded Queues._ LMAX, 2011.
9. Helland. _Immutability Changes Everything._ CIDR 2015 / CACM 59(1), 2016.
10. Helland. _Life beyond Distributed Transactions._ CIDR 2007 / CACM 60(2), 2017.
11. Helland. _Data on the Outside Versus Data on the Inside._ CIDR 2005.
12. Erb et al. _Log Pruning in Distributed Event-sourced Systems._ ACM DEBS, 2018.
13. Gomes, Kleppmann et al. _Verifying Strong Eventual Consistency._ PACMPL/OOPSLA, 2017.
14. Kabbedijk et al. _Variability Consequences of CQRS._ EuroPLoP (ACM), 2012.
15. Maddodi et al. _Aggregate Architecture Simulation in ES._ ACM/SPEC ICPE, 2020.
16. Kung & Robinson. _On Optimistic Methods for Concurrency Control._ ACM VLDB, 1981.
17. Djerou & Tibermacine. _SAGA Verification Using Maude._ IEEE, 2023.
18. Kingsbury & Alvaro. _Elle: Inferring Isolation Anomalies._ PODC/VLDB, 2021.

### Performance Benchmark Sources

19. Pececnik Almlöf & Karlsson. _CQRS vs CRUD Performance._ KTH, 2026. DiVA diva2:2067066
20. Richter. _CQRS Performance in C# Web APIs._ Halle-Wittenberg, 2024.
21. Adiputra et al. _Property Management with CQRS/ES._ Jurnal BT, 2025.

### Foundational Industry Works

22. Young. _CQRS Documents._ 2010.
23. Fowler. _CQRS / Event Sourcing._ martinfowler.com, 2011/2005.
24. Betts et al. _Exploring CQRS and Event Sourcing (CQRS Journey)._ Microsoft, 2013.
25. Kreps. _The Log: What every software engineer should know about real-time data's unifying._ 2014.
26. Young. _Versioning in an Event Sourced System._ Leanpub, 2017.

Full annotated bibliography (~50 entries) in §2.
