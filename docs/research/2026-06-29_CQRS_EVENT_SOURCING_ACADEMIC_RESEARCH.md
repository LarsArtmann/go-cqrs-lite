# CQRS & Event-Sourcing Academic Literature: Deep Research & Project Implications

> **Status:** Informational research report — 6/29/2026
> **Method:** Surveyed the peer-reviewed academic literature (IEEE, ACM, Springer, CIDR, arXiv) on CQRS and Event Sourcing, extracted the empirical findings and design principles, and mapped them to go-cqrs-lite's current state.
> **Scope:** ~35 academic papers, theses, and foundational works, spanning 1996–2026.

---

## Executive Summary

The academic literature on CQRS/ES is **surprisingly thin** compared to its industry popularity — a recurring finding in the research itself (Overeem 2021, Laigner 2024). The most rigorous peer-reviewed work comes from three research streams:

1. **Utrecht University (Overeem, Jansen, Brinkkemper, Maddodi, Spoor)** — empirical studies of real ES systems, schema evolution, and performance modeling.
2. **KPI Dnipro (Lytvynov & Hruzin, 2024–2026)** — the only sustained research program on CQRS+ES _architectural variations_, including snapshot-centric CQRS and critical causal events.
3. **Pat Helland (Microsoft/Amazon)** — the foundational theoretical lens for distributed, event-driven systems (inside vs. outside data, entities/activities, immutability).

**Bottom line for go-cqrs-lite:** The project's architecture is **strongly validated** by the literature — the Decider pattern, immutability, idempotency, upcasting, tombstones, and the projection-host framework all map directly to research recommendations. However, the literature exposes **seven concrete capability gaps** that this library should consider closing, prioritized by how often the research flags them. The highest-impact gap is **copy-and-transform (event-store rewriting) tooling**, which is the single most widely used schema-evolution technique in industry (14/25 engineers in Overeem's study) and currently has **zero** implementation in this codebase.

---

## Part 1: The Literature Surveyed

### 1.1 Foundational Works (Industry-Origin, Universally Cited)

| Work                                               | Author(s)                                        | Year | Significance                                                                             |
| -------------------------------------------------- | ------------------------------------------------ | ---- | ---------------------------------------------------------------------------------------- |
| _CQRS Documents_                                   | Greg Young                                       | 2010 | Originated CQRS; extended Meyer's CQS to architecture; linked CQRS+ES+DDD+task-based UIs |
| _CQRS_ (bliki)                                     | Martin Fowler                                    | 2011 | Popularized CQRS; **cautionary** — "for most systems CQRS adds risky complexity"         |
| _Clarified CQRS_                                   | Udi Dahan                                        | 2009 | Precursor of CQRS (predates Young); SOA+DDD framing                                      |
| _Exploring CQRS and Event Sourcing_ (CQRS Journey) | Betts, Domínguez, Melnik, Simonazzi, Subramanian | 2013 | Microsoft reference implementation on Azure; most-cited industry resource                |
| _Event Sourcing_                                   | Martin Fowler                                    | 2005 | Coined "Event Sourcing"; defined rebuild, temporal query, event replay                   |
| _Disruptor_ whitepaper                             | Thompson, Farley, Barker, Gee, Stewart (LMAX)    | 2011 | Real-world CQRS+ES at millions of TPS via lock-free ring buffer                          |
| _Kappa Architecture_                               | Jay Kreps                                        | 2014 | Single immutable log replaces lambda batch+speed layers                                  |

### 1.2 Peer-Reviewed Academic Papers — CQRS (ACM/IEEE/Springer)

| Paper                                                                                 | Venue                      | Year | Key Contribution                                                     |
| ------------------------------------------------------------------------------------- | -------------------------- | ---- | -------------------------------------------------------------------- |
| Kabbedijk, Jansen, Brinkkemper — _Variability Consequences of CQRS_                   | EuroPLoP (ACM)             | 2012 | First peer-reviewed CQRS case study; variability/complexity tension  |
| Overeem, Spoor, Jansen — _Dark Side of Event Sourcing: Data Conversion_               | IEEE SANER                 | 2017 | Data conversion (CRUD→ES) as an adoption barrier                     |
| Debski, Szczepanik, Malawski, Spahr — _Scalable Reactive Architecture_                | IEEE Software              | 2017 | Real cloud CQRS+ES design (61 citations)                             |
| Long — _High Performance CQRS_                                                        | IEEE ICRIS                 | 2017 | CQRS performance characteristics                                     |
| Rajkovic — _CQRS for Medical Information Systems_                                     | Conf.                      | 2013 | Denormalized read DBs → measurable perf gains (22 citations)         |
| Maddodi, Jansen, Overeem — _Aggregate Architecture Simulation_                        | ACM/SPEC ICPE              | 2020 | Layered queuing networks for CQRS performance modeling               |
| Munonye, Martinek — _Data Storage Patterns in Microservices_                          | IEEE ICCSE                 | 2020 | CQRS+ES standardization could mitigate microservice data challenges  |
| Laigner, Almeida, Assunção et al. — _Challenges of Event Management in Microservices_ | **ACM TOSEM**              | 2024 | First empirical characterization; 8,000+ SO questions; 16 challenges |
| Mohammad — _Kafka Event-Streaming Design Patterns_                                    | IEEE ICICyTA               | 2025 | Synthesis of 42 studies; CQRS bus pattern taxonomy                   |
| Hakim — _Correctness for CQRS Systems_                                                | KTH thesis                 | 2012 | **Only** dedicated CQRS correctness/formal-verification study        |
| Nilsson & Korkmaz — _Practitioners' View on CQRS_                                     | Lund thesis                | 2014 | Qualitative study; 5 adoption themes                                 |
| Lytvynov & Hruzin — _Snapshot-Centric CQRS+ES_                                        | KPI Science News           | 2026 | mCQRS variant; −32% complexity at +23% write latency                 |
| Lytvynov & Hruzin — _Critical Causal Events in CQRS+ES_                               | Radio Elec. CS Control     | 2024 | Out-of-order causal events; "Container of Events" solution           |
| Lytvynov & Hruzin — _CQRS+ES Architectural Variations_                                | Tech. Audit Prod. Reserves | 2025 | Decision-making framework for variation selection                    |

### 1.3 Peer-Reviewed Academic Papers — Event Sourcing

| Paper                                                                      | Venue                      | Year | Key Contribution                                                |
| -------------------------------------------------------------------------- | -------------------------- | ---- | --------------------------------------------------------------- |
| **Overeem et al. — _Empirical Characterization of ES & Schema Evolution_** | **JSS** (arXiv 2104.01146) | 2021 | **THE definitive academic ES study** — 25 engineers, 19 systems |
| Erb, Meißner et al. — _Consistent Retrospective Snapshots_                 | NetSys                     | 2017 | Snapshotting across distributed streams                         |
| Meißner — _Time Travel in Distributed ES_                                  | ACM DEBS                   | 2018 | Globally-consistent temporal queries                            |
| Meißner, Erb, Kargl, Tichy — _Retro-λ_                                     | ACM DEBS                   | 2018 | Retroactive computing in serverless ES                          |
| Erb, Meißner, Ogger, Kargl — _Log Pruning in Distributed ES_               | ACM DEBS                   | 2018 | Bounded log growth via pruning                                  |
| Erb, Meißner, Pietron, Kargl — _Chronograph_                               | ACM DEBS                   | 2017 | Event-sourced graphs (online + batch)                           |
| Erb — _Distributed Computing on Event-Sourced Graphs_                      | Ulm PhD thesis             | 2020 | Synthesizes the Ulm research program                            |
| Alongi et al. — _Event-Sourced Observable Architectures_                   | Softw. Pract. Exp.         | 2022 | ES + observability experience report                            |
| Kleppmann & Kreps — _Kafka, Samza and Unix Philosophy_                     | IEEE Data Eng. Bull.       | 2015 | Append-only log as general-purpose event store                  |

### 1.4 Pat Helland — The Theoretical Foundation

| Paper                                                                       | Venue       | Year      | Key Contribution                                                      |
| --------------------------------------------------------------------------- | ----------- | --------- | --------------------------------------------------------------------- |
| _Data on the Outside vs. Data on the Inside_                                | CIDR        | 2005      | Inside (ACID "now") vs. outside (immutable "past") data               |
| _Life beyond Distributed Transactions_                                      | CIDR / CACM | 2007/2017 | Entities/activities model; at-least-once + idempotence; tentative ops |
| _The Dangers of Replication and a Solution_ (Gray, Helland, O'Neil, Shasha) | ACM SIGMOD  | 1996      | Primary-copy replication foundation                                   |
| _Building on Quicksand_ (Helland & Campbell)                                | CIDR        | 2009      | Embrace uncertainty; semantic reconciliation                          |
| _Immutability Changes Everything_                                           | CIDR / CACM | 2015/2016 | "The truth is the log; the database is a cache of the log"            |
| _Don't Get Stuck in the "Con" Game_                                         | CACM        | 2021      | Consistency vs. convergence vs. confluence                            |
| _Idempotence Is Not a Medical Condition_                                    | CACM        | 2012      | Idempotence as essential for reliable event processing                |

---

## Part 2: Key Findings From the Most Important Papers

### 2.1 Overeem et al. (2021) — The Definitive Empirical Study

**Method:** Constructivist grounded theory; 22 interviews with 25 engineers (103 combined years of experience) across 19 production event-sourced systems (thousands to billions of events).

#### Three Degrees of Immutability (critical insight)

Immutability in ES is **self-imposed, not enforced** (unlike blockchain). The study observed three levels:

- **Strict immutability** (8/19 systems) — events never change.
- **Cut-off immutability** (3/19) — store changed at defined cut-off moments; backups preserve audit trails.
- **Mutable** (8/19) — events can change; no permanent audit guarantee.

**The degree chosen directly determines which schema-evolution techniques are available.**

#### The Five Schema-Evolution Tactics (by adoption rate)

| Tactic                      | Engineers Using       | How It Works                                                                | Key Liability                                              |
| --------------------------- | --------------------- | --------------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Copy-and-Transform**      | **14/25** (most-used) | ETL old streams → new streams with new schema; source untouched             | Slow for large stores; new store is effectively mutable    |
| **Upcasting**               | **12/25**             | Transform-on-read; `upcast` function runs before `project`; store untouched | Accumulated chain slows loading over time                  |
| **Weak Schema**             | **11/25**             | Permissive/minimal schema; tolerant readers (Protobuf/AVRO-style)           | Application logic pollution; can't express complex changes |
| **In-Place Transformation** | 5/25                  | Directly mutate stored events to new schema                                 | Destroys audit trail; data-loss risk                       |
| **Versioned Events**        | 2/25                  | Introduce new event types; handle both old+new in projection                | Application logic pollution                                |

**Tactics are combined in practice**, never used in isolation. Upcasters + copy-and-transform is the most common pairing (upcasters day-to-day, copy-and-transform to "clean up" periodically).

#### Recommended Progression (framework designers' guidance)

> 1. Start with **versioned events + weak schema** (simplest).
> 2. When those can't handle the change, add **upcasting** (preserves immutability).
> 3. Only when upcaster chains hurt performance/maintainability, apply **copy-and-transform**.
> 4. **In-place transformation** only for systems that don't require an audit log.

#### The Five Key Challenges

1. **Event system evolution** — engineers "dreaded the upgrading, had fear in advance."
2. **Steep learning curve** — 13/25 reported events/state-transfers thinking is fundamentally different; eventual consistency is "the biggest hurdle."
3. **Lack of available technology** — 8/25 reported immature frameworks; tension between "no framework needed" vs "want mature libraries."
4. **Rebuilding projections** — slow; full-history replay. Mitigations: weekend rebuilds, domain-aware skipping, targeted/partial rebuilds, in-memory-first.
5. **Data privacy (GDPR)** — conflicts with retain-forever. Mitigations: separate personal-data store, anonymization, removal (breaks immutability).

#### Implications for Framework Designers

- **Upcasters as a first-class concept** — most popular immutability-preserving technique; its main liability (accumulated perf cost) is a _framework-level_ problem solvable via intelligent caching or compiler-like upcaster chaining.
- **Copy-and-transform tooling should be built in** — most widely used technique overall; frameworks should provide native stream/store-level ETL with batch processing.
- **Weak schema support in serialization layers** — tolerant readers (Protobuf/AVRO-style).
- **In-place transformation should be opt-in with clear warnings.**
- **Projection rebuild infrastructure is essential** — independent, autonomous projector processes; targeted/partial rebuilds; domain-aware event filtering.

---

### 2.2 Lytvynov & Hruzin (2024–2026) — CQRS+ES Architectural Variations

#### Finding A: Snapshot-Centric CQRS (mCQRS)

The classical event-log-as-source-of-truth model has high component complexity. The **mCQRS** variant makes relational **snapshots the source of truth** (events still stored for audit/temporal reconstruction):

| Metric                                   | Classical CQRS | mCQRS  | Change                  |
| ---------------------------------------- | -------------- | ------ | ----------------------- |
| Cyclomatic complexity (command workflow) | 120            | 82     | **−31.67%**             |
| Query response time                      | 44 ms          | 44 ms  | Identical               |
| End-to-end consistency time              | 268 ms         | 347 ms | +22.76% (slower writes) |

**Implication:** Snapshot-as-source-of-truth is a legitimate, lower-complexity entry point for systems where write performance is not critical. Read performance is unaffected.

#### Finding B: Critical Causal Events

In async CQRS+ES with pub/sub, **events are not guaranteed in-order**. When event A _causes_ event B and they arrive out of order, the system enters intermittent, hard-to-reproduce inconsistent states ("critical causal events"). The likelihood depends on entity interdependencies and use-case complexity.

**Best solution identified: "Container of Events"** — each event carries metadata about its causal predecessors (full causality history), adapted for CQRS+ES. Requires modifications to the Event Delivery Subsystem. Evaluated against synchronous queues and causal barriers; Container of Events won on the integrated performance/complexity assessment.

#### Finding C: CQRS+ES Is a Family of Variations

Not one architecture but a family differing in complexity, performance, development time, and required expertise. Migration between variations is costly if unplanned. They propose metrics-based decision-making (replacing subjective expert judgment).

---

### 2.3 Laigner et al. (ACM TOSEM 2024) — Microservices Event Management

**Method:** Mined 8,000+ Stack Overflow questions; manually coded 628 in depth.

**The paradox:** Events are adopted to maximize decoupling, yet the most common challenges involve developers needing to **synchronize, order, and coordinate** events — effectively re-coupling their systems.

#### The 5 Challenge Categories (16 challenges)

| Category              | Top Challenge                                              | % of Total    | Relevance          |
| --------------------- | ---------------------------------------------------------- | ------------- | ------------------ |
| **Safety & Liveness** | CAS7: Weak delivery semantics (at-least-once → duplicates) | **17.20%**    | Idempotency        |
|                       | CAS3: Processing event dependencies (cross-stream)         | 12.54%        | Causal ordering    |
|                       | CAS4: Processing events in order (within-stream)           | 10.39%        | Ordering           |
|                       | CAS5: Synchronizing states via event replay                | 8.96%         | Projection rebuild |
|                       | CAS2: Rolling back states (compensations)                  | 8.24%         | Sagas              |
|                       | CAS1: Publishing events safely (atomicity)                 | 8.60%         | Outbox             |
|                       | CAS6: Continuous queries over events                       | 7.53%         | Streaming          |
| **Schema Management** | CEM1: Modeling event schemas (fat vs. thin events)         | **19.71%**    | Event design       |
|                       | CEM2: Evolving event schemas                               | 5.02%         | Schema evolution   |
| **Performance**       | CP1: Event-processing overhead                             | 5.38%         | Replay perf        |
|                       | CP2: Large event payloads                                  | 1.79%         | Payload design     |
|                       | CP3: Fluctuating event rate (no backpressure)              | 2.87%         | Backpressure       |
| **Observability**     | CO1: Event flow observability (tracing gaps)               | (89% of obs.) | OTel               |
|                       | CO2: Auditing via event replay                             | —             | Audit              |
| **Security**          | CS1: Authentication with async events                      | —             | Auth               |
|                       | CS2: Data privacy (GDPR)                                   | —             | Redaction          |

#### Framework-Designer Recommendations

1. Provide an API to **wait for multiple dependent events** (not just individual events) — addresses CAS3.
2. Make **delivery/publishing semantics explicit** in docs.
3. Provide guidelines for **rolling out schema changes** without breaking consumers.
4. **Native observability integration** — metrics for event-processing delays and consumer-ack delays.
5. **Efficient backpressure** for fast-producer/slow-consumer.

---

### 2.4 Pat Helland — Ten Design Principles for ES Libraries

Synthesized across the three seminal papers:

1. **The event log is the system of record; all state is a derived projection.** "The truth is the log; the database is a cache of the log."
2. **The aggregate/entity is the transaction boundary.** No transaction spans aggregates; coordination is via messages only.
3. **Events are immutable, versioned, self-describing.** Carry version-dependent identifiers for referenced data (e.g., the price _at the time_, not a mutable product ID).
4. **Design for at-least-once delivery with idempotence.** Never assume exactly-once. "If I run this handler twice, do I get the same result?"
5. **Treat inside vs. outside data differently.** Inside: ACID, mutable, "now." Outside: immutable, "past," eventual consistency. Commands are "hopeful for the future"; incoming events are "from the past."
6. **Model cross-aggregate workflows as activities with tentative operations** (cancelled/confirmed), not distributed transactions.
7. **Accept uncertainty and reconcile.** Background jobs scan for "pending too long"; explicit intermediate states.
8. **Separate commands from events (CQRS).** Commands = intent, may be rejected. Events = facts, already happened.
9. **Leverage immutability for replication and parallelism.** Immutable events replicate without locks; projections build independently.
10. **Never send side effects during replay.** The processor must know live vs. replay mode and gate external systems.

---

## Part 3: Gap Analysis — Research Findings vs. go-cqrs-lite State

I verified the project's implementation state for each research finding. Below: **what's validated** (the research confirms decisions already made) and **what's missing** (the research exposes gaps).

### 3.1 What the Project Already Does RIGHT (validated by research)

| Research Recommendation                         | Project Implementation                                                                                       | Source                                                       |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------ |
| Pure functions (Decider pattern)                | `decider.Decider[State]`, ADR-0001                                                                           | Overeem: functional core/imperative shell shift; Erb/Meißner |
| Upcasting as first-class                        | `schema.Upcaster`, `VersionedStore`, `upcasterRegistry` (`schema/upcaster.go`, `schema/versioned_source.go`) | Overeem: 12/25 engineers; recommended framework primitive    |
| Idempotency for at-least-once delivery          | `idempotency/` (MemoryStore, KVStore, middleware, `ErrDuplicate`)                                            | Helland #4; Laigner CAS7 (17.2% of challenges)               |
| Aggregate = transaction boundary                | Decider + Repository, per-aggregate `version`                                                                | Helland #2                                                   |
| Tombstone soft-delete (no Delete)               | ADR-0005, `event.DetectTombstone`                                                                            | Laigner CS2 (GDPR via tombstone events)                      |
| Sink/Source ISP split                           | ADR-0006                                                                                                     | Helland inside/outside                                       |
| Replay mode flag                                | `event.ProcessingMode`, `ModeReplay`, `IsReplay(ctx)` (`event/replay.go`)                                    | Helland #10; Fowler                                          |
| Command→event causation                         | `Causation`, `WithCommandCausality`, `CommandCausalityEnricher` (`event/causality.go`)                       | Lytvynov critical causal events                              |
| Immutable events                                | `*ImmutableEvent`, defensive clones on all accessors                                                         | Helland #1, #3                                               |
| Branded IDs (version-independent identity)      | `id.Of[T]`, ADR-0024                                                                                         | Helland: stable, identifiable outside data                   |
| Event catalog / self-describing                 | `catalog/` AsyncAPI, OpenAPI, EventCatalog, D2                                                               | Helland #3; Laigner CEM1 (event schema modeling)             |
| Managed projection host (the "last loop")       | `projectionhost/` with DLQ, checkpoint, backoff                                                              | Overeem Challenge 4; Laigner CAS5                            |
| Scheduled commands (durable deadlines)          | `scheduling/`                                                                                                | Activities/tentative operations (Helland #6)                 |
| OTel observability                              | `otel/`, `prometheus/`, tracing/metrics middleware                                                           | Laigner CO1 (89% of observability challenges)                |
| Scenario-testing DSL                            | `scenario/` Given/When/Then                                                                                  | Overeem Challenge 2 (learning curve)                         |
| Multi-module isolation (library, not framework) | 52 `go.mod` files, go.work                                                                                   | Overeem Challenge 3 (lack of mature tech)                    |

**Verdict:** The core architecture is _exemplary_ against the literature. Every major research recommendation has a corresponding implementation.

---

### 3.2 What the Research Exposes as MISSING (7 actionable gaps)

| #      | Research Finding                                                                                                                                  | Current Project State                                                                                                                                                  | Effort      | Impact                                    |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ----------------------------------------- |
| **G1** | **Copy-and-Transform (event-store ETL)** — most-used schema-evolution technique (14/25 engineers); Overeem explicitly recommends built-in tooling | **MISSING** — only upcasting (transform-on-read) exists. No tool to copy streams into a new store with transformed payloads.                                           | Medium      | **HIGH**                                  |
| **G2** | **Event→event causal dependency tracking + causal barriers** — Lytvynov's "Container of Events" solves out-of-order causal events                 | **PARTIAL** — only command→event causation (`event/causality.go`). No event→event causal graph, no causal-barrier/wait-for-dependency.                                 | Medium-High | **HIGH**                                  |
| **G3** | **GDPR redaction / crypto-shredding** — Overeem Challenge 5; Laigner CS2; separation of personal data                                             | **MISSING** — only a design-review HTML (`docs/brainstorming/event-redaction-design-review.html`), no code. `encryption/` is encrypt-at-rest, not deletion.            | Low-Medium  | **MEDIUM-HIGH**                           |
| **G4** | **Auto side-effect suppression during replay** — Helland #10; Fowler                                                                              | **PARTIAL** — `ProcessingMode` flag exists but consumers must manually check `IsReplay(ctx)`. No `SideEffect` abstraction or auto-gating middleware.                   | Low         | **MEDIUM**                                |
| **G5** | **Snapshot strategy diversity** (read-pressure, time-based, adaptive) — Lytvynov snapshot-centric; Maddodi performance modeling                   | **PARTIAL** — only `EveryNEvents` ships. `ReadPressureStrategy` is design-only (`docs/design/read-pressure-snapshots.md`).                                             | Low-Medium  | **MEDIUM**                                |
| **G6** | **Log pruning / compaction / tiered storage** — Erb et al. (DEBS 2018); Overeem cut-off immutability                                              | **MISSING** — design spikes only (`docs/design/event-compaction.md`, `event-archival.md`), deferred.                                                                   | High        | **MEDIUM** (defer OK until scale demands) |
| **G7** | **Projection rebuild optimization** (domain-aware skipping, targeted/partial, in-memory-first) — Overeem Challenge 4                              | **PARTIAL** — `projectionhost/` handles crash-restart + checkpoint, but no domain-aware event filtering or parallel rebuild primitives beyond the catch-up subscriber. | Medium      | **MEDIUM**                                |

---

## Part 4: Actionable Recommendations (Pareto-Prioritized)

Ordered by impact-to-effort ratio, informed by how frequently the research flags each gap.

### P1 — Add Copy-and-Transform Tooling (closes G1) — _Highest leverage_

**Why:** The single most-used schema-evolution technique in industry (14/25 engineers). Upcasting is the right _day-to-day_ tool (already shipped), but Overeem's research shows teams periodically need to "clean up" accumulated upcaster chains by rewriting the store. Currently a consumer must hand-roll this ETL.

**Proposal:** A `migrate/` (or `transform/`) module providing:

```go
// Copy event streams from source store to target store with per-event transformation.
// Preserves source immutability; produces a new store conforming to the new schema.
migrator := migrate.New(sourceStore, targetStore,
    migrate.WithBatchSize(1000),               // Overeem: batch for performance
    migrate.WithTransform(upcasterChain),        // reuse schema.Upcaster chain
    migrate.WithFilter(skipObsoleteEvents),      // domain-aware skipping (Overeem Challenge 4)
    migrate.WithProgress(reporter),              // large stores take time (Overeem)
)
err := migrator.Run(ctx, event.NewAggregateRef("User", userID))  // per-stream or global
```

This composes existing primitives (`schema.Upcaster`, `event.Journal.ReadAll`, `event.Store.AppendBatch`). **No new abstractions needed** — it's an opinionated orchestration of existing pieces.

**Research backing:** Overeem (2021) §7 Tactic 5; recommended explicitly for framework designers.

---

### P2 — Add a `replay` Middleware Namespace for Side-Effect Gating (closes G4) — _Quick win_

**Why:** Helland #10 and Fowler both insist side effects (emails, payments, webhooks) MUST be suppressed during replay. The library exposes `event.IsReplay(ctx)` but leaves every consumer to remember to check it. A reusable middleware removes this footgun.

**Proposal:** A thin `middleware` (or `replay/`) helper:

```go
// Gate wraps a side-effect function; it no-ops during replay.
sendEmail := replay.Gate(ctx, func(ctx context.Context) error {
    return emailer.Send(ctx, msg)
})
// or middleware form on a command/event handler
cmdDisp.Use(replay.SuppressSideEffects())  // marks handlers as side-effect-safe-to-skip
```

Low effort: it's a context check + a decorator. High safety payoff.

**Research backing:** Helland #10; Fowler _Event Sourcing_ §External Updates.

---

### P3 — Enrich Causality to Event→Event + Causal Barriers (closes G2) — _Highest research signal_

**Why:** Lytvynov (2024) shows out-of-order causal events cause intermittent, hard-to-reproduce bugs, and that "Container of Events" (full causal history in metadata) is the best solution. Laigner CAS3 (cross-stream dependencies, 12.54% of all challenges) and CAS4 (in-stream ordering, 10.39%) are both top-5 industry pain points. The library currently tracks only command→event causation.

**Proposal (phased):**

1. **Phase 1 (low effort):** Extend `event.Metadata.Causation` to carry an optional `CausedByEventID id.EventID` (event→event causation). Stamp it when a projection/saga emits a command that produces an event.
2. **Phase 2 (medium effort):** A `causality.Barrier` — a projection wrapper that buffers events until their causal predecessor has been processed, then releases in causal order. Composes with `projectionhost/`.
3. **Phase 3 (optional):** A `graph/`-backed causation DAG viewer (already supported by `graph.GraphProjection` — just needs the data).

**Research backing:** Lytvynov & Hruzin (2024); Laigner et al. (2024) CAS3/CAS4.

---

### P4 — Add GDPR Redaction / Crypto-Shredding (closes G3) — _Regulatory necessity_

**Why:** Overeem Challenge 5 and Laigner CS2 both flag GDPR as a fundamental tension with retain-forever event logs. Three mitigation patterns emerge from the research: (a) separate personal-data store, (b) anonymization, (c) crypto-shredding (delete the key). The library already has `encryption/` — crypto-shredding is a natural extension.

**Proposal:**

```go
// Crypto-shredding: encrypt PII fields per-subject; deletion = destroy the key.
redactor := redaction.NewCryptoShredder(keyStore)
bus.UsePublish(redaction.RedactMiddleware(redactor,
    redaction.WithFields("ssn", "email", "address"),
))
// To "forget" a subject: keyStore.Destroy(subjectID) → all their PII becomes unreadable ciphertext.
```

This preserves immutability (events stay; only keys are destroyed) and composes with existing `encryption/`. The design review already exists — it needs code.

**Research backing:** Overeem (2021) Challenge 5; Laigner (2024) CS2.

---

### P5 — Expand Snapshot Strategies (closes G5) — _Low-effort breadth_

**Why:** Only `EveryNEvents` ships. Lytvynov's snapshot-centric mCQRS research and Maddodi's performance-modeling work both show snapshot strategy materially affects performance and complexity. The `SnapshotStrategy` interface is already extensible — just ship more strategies.

**Proposal:** Add to `snapshot/`:

- `TimeBasedStrategy(maxAge time.Duration)` — snapshot if last snapshot older than threshold (good for read-heavy aggregates).
- `ReadPressureStrategy(threshold int)` — the design spike in `docs/design/read-pressure-snapshots.md`, promoted to code. Snapshot when read count exceeds threshold.
- `CompositeStrategy(strategies...)` — first-match-wins composition.

Each is ~30 lines. The `SnapshotStrategy` interface (`snapshot/strategy.go:5`) already supports this.

**Research backing:** Lytvynov & Hruzin (2026) mCQRS; Maddodi et al. (2020) ICPE.

---

### P6 — Projection Rebuild Primitives (closes G7) — _Operational quality_

**Why:** Overeem Challenge 4 (rebuilding projections is slow) is a top-5 industry pain. Mitigations the research identifies: domain-aware skipping, targeted/partial rebuilds, in-memory-first. `projectionhost/` handles lifecycle but not rebuild optimization.

**Proposal:** Add to `projectionhost/`:

- `WithEventFilter(filter func(event.Event) bool)` — skip events a projection doesn't care about during rebuild (Overeem: "older events may no longer be relevant").
- `WithParallelism(n int)` — rebuild independent projections concurrently.
- A `Rebuild(ctx, projectionName)` method that resets the checkpoint and replays from zero, with progress reporting.

**Research backing:** Overeem (2021) Challenge 4.

---

### P7 — Log Pruning / Tiered Storage (closes G6) — _Defer until scale demands_

**Why:** Erb et al. (DEBS 2018) address unbounded log growth. Overeem's "cut-off immutability" tier (3/19 systems) explicitly prunes at defined moments. However, this destroys time-travel ability (the project's own research notes flag this — `docs/research/time-travel-options.md:374`).

**Recommendation:** **Keep deferred.** The research supports the current decision: pruning is only justified at scale (billions of events) and conflicts with temporal-query capabilities the library deliberately preserves. When needed, implement tiered storage (hot → warm → S3 archive via `docs/design/event-archival.md`) rather than deletion, mirroring KurrentDB's archiving approach.

**Research backing:** Erb et al. (2018); Overeem (2021) cut-off immutability tier.

---

## Part 5: Cross-Cutting Insights Worth Internalizing

These don't map to single features but shape how the library should evolve:

1. **Fowler's caution is empirically validated.** CQRS "adds risky complexity" (Fowler 2011). Kabbedijk (2012) measured the variability consequences. **The library's job is to absorb that complexity** — every "consumer must hand-roll X" is a failure of this mission. The projectionhost, scenario DSL, and stack presets all serve this; the gaps above (especially G1, G4) are places complexity still leaks to consumers.

2. **Immutability is a spectrum, not a boolean.** Overeem shows only 8/19 systems are strict-immutable. The library should make the choice _explicit_ — strict by default, with clear opt-in to cut-off/mutable modes (this informs whether in-place transformation should ever be offered).

3. **The "decoupling paradox" (Laigner).** Events are meant to decouple, but the top challenges are re-coupling (ordering, dependencies, synchronization). The library's causal-barrier work (P3) and any future "wait for N events" primitive directly serve the most painful part of this paradox.

4. **Events need stable business meaning.** Overeem: "Doing DDD leads to a less fragile design" because events with real business equivalents change less. The `catalog/` module's auto-documentation and the DOMAIN_LANGUAGE.md convention directly support this — they're not just docs, they're fragility-reduction tools.

5. **Performance modeling is possible.** Maddodi (ICPE 2020) shows CQRS performance is modelable via queuing networks. The library could eventually ship a benchmark/simulation harness that predicts aggregate-rebuild and projection-rebuild costs — turning "rebuilding is slow" from a surprise into a planned operation.

---

## Part 6: Summary Decision Matrix

| Recommendation                           | Effort      | Impact      | Research Consensus           | Verdict                                |
| ---------------------------------------- | ----------- | ----------- | ---------------------------- | -------------------------------------- |
| **P1** Copy-and-Transform                | Medium      | HIGH        | Strong (14/25 engineers)     | **DO**                                 |
| **P2** Replay side-effect gating         | Low         | MEDIUM      | Strong (Helland #10)         | **DO**                                 |
| **P3** Event→event causality + barriers  | Medium-High | HIGH        | Strong (Lytvynov + Laigner)  | **DO (phased)**                        |
| **P4** GDPR redaction / crypto-shredding | Low-Medium  | MEDIUM-HIGH | Strong (Overeem + Laigner)   | **DO**                                 |
| **P5** More snapshot strategies          | Low         | MEDIUM      | Moderate (Lytvynov, Maddodi) | **DO**                                 |
| **P6** Projection rebuild primitives     | Medium      | MEDIUM      | Moderate (Overeem)           | **DO when capacity**                   |
| **P7** Log pruning / tiered storage      | High        | MEDIUM      | Moderate (Erb)               | **DEFER** (conflicts with time-travel) |

**Net:** Five "DO" items (P1–P5), one "do when capacity" (P6), one "defer" (P7). P1, P2, and P4 are the highest impact-to-effort and should be the first to execute.

---

## References (Selected, by importance to this report)

1. Overeem, Spoor, Jansen, Brinkkemper. _An Empirical Characterization of Event Sourced Systems and Their Schema Evolution — Lessons from Industry._ JSS 178, 2021. arXiv:2104.01146 — **primary source for G1, G3, G7 and §2.1.**
2. Lytvynov & Hruzin. _Engineering of Software Systems Based on Snapshot-Centric CQRS with Event Sourcing Architecture._ KPI Science News 142(1), 2026. DOI 10.20535/kpisn.2026.1.350992 — **primary source for G5 and §2.2A.**
3. Lytvynov & Hruzin. _Critical Causal Events in Systems Based on CQRS with Event Sourcing Architecture._ Radio Elec. CS Control 3, 2024. DOI 10.15588/1607-3274-2024-3-11 — **primary source for G2 and §2.2B.**
4. Laigner, Almeida, Assunção et al. _An Empirical Study on Challenges of Event Management in Microservice Architectures._ ACM TOSEM, 2024. DOI 10.1145/3776581 — **primary source for §2.3.**
5. Helland. _Immutability Changes Everything._ CIDR 2015 / CACM 59(1), 2016. — **§2.4 principle #1.**
6. Helland. _Life beyond Distributed Transactions._ CIDR 2007 / CACM 60(2), 2017. — **§2.4 principles #2, #6.**
7. Helland. _Data on the Outside Versus Data on the Inside._ CIDR 2005. — **§2.4 principle #5.**
8. Helland. _Don't Get Stuck in the "Con" Game._ CACM 19(3), 2021.
9. Erb, Meißner, Ogger, Kargl. _Log Pruning in Distributed Event-sourced Systems._ ACM DEBS 2018. — **G7/P7.**
10. Meißner, Erb, Kargl, Tichy. _Retro-λ._ ACM DEBS 2018.
11. Fowler. _Event Sourcing._ martinfowler.com, 2005. — **§2.4 principle #10.**
12. Betts et al. _Exploring CQRS and Event Sourcing (CQRS Journey)._ Microsoft, 2013.
13. Kabbedijk, Jansen, Brinkkemper. _Variability Consequences of CQRS._ EuroPLoP (ACM), 2012. DOI 10.1145/2602928.2603078.
14. Maddodi, Jansen, Overeem. _Aggregate Architecture Simulation in ES._ ACM/SPEC ICPE, 2020. DOI 10.1145/3358960.3375797.
15. Hakim. _Correctness for CQRS Systems._ KTH, 2012.

Full bibliography (~35 entries) in Part 1.
