# Event History → Aggregate State Visualization Tools

> **The core question:** Is there a production-ready tool that shows how an aggregate's state evolves event-by-event with a timeline/slider — like Redux DevTools does for Redux, but for server-side event sourcing?

**Short answer: No. This is a genuine gap in the ecosystem.**

---

## Comparison Table

| Tool | Type | Shows Aggregate State Over Time? | How | Availability | Maturity |
|---|---|---|---|---|---|
| **KurrentDB Visualize Tab** | Runtime debugger | ✅ Causal graph of events linked by correlation/causation IDs, showing full entity lifecycle | Graph/tree view with event nodes + causal links. Branches where multiple events share a causation ID. | **Enterprise/Cloud only** — not in OSS | Production |
| **KurrentDB Stream Browser** | Runtime browser | ⚠️ Partial — browse individual streams (one aggregate), see events in order, but no state reconstruction | Paginated event list per stream. You see the raw events, not the folded state. | **Free (OSS)** | Production |
| **Prooph Board** | Design-time modeling | ✅ Event modeling with timeline showing commands→events→state over time | Interactive timeline: commands (blue), events (orange), read models (green). Design-phase, not runtime. | Free (3 seats), €6/seat/mo | Production |
| **Redux DevTools** | Runtime debugger | ✅ Action-by-action state diffs with time travel | Timeline of actions, state tree inspector, diff highlight, jump to any point | Free (browser ext) | Production (20k+ ⭐) |
| **XState Stately** | Design + simulation | ✅ Event-triggered state transitions visualized on a statechart | Interactive statechart: send events, watch transitions, inspect context | Free (basic), paid (teams) | Production |
| **Avala State Machine Panel** | Runtime monitor | ✅ State machine history with timeline scrubbing + analytics | Horizontal bar chart of state durations, transition history, frequency stats | — | Early |
| **Replay.io** | Runtime recorder | ⚠️ Partial — auto-discovers state machines from recorded sessions, but not ES-specific | Record→replay browser sessions, auto-extract flow maps | Paid | Production |
| **Marten** | Event store library | ❌ PostgreSQL-based, no built-in UI | Would need custom Grafana/superset dashboard | Free (OSS, 1.8k ⭐) | Production |
| **Axon Server** | Event store + command bus | ⚠️ Partial — Grafana dashboards for metrics, not aggregate state | Operational monitoring only, not state timeline | Free (standard), paid (enterprise) | Production |

---

## Detailed Breakdown

### KurrentDB (ex-EventStoreDB)

- **Repo:** <https://github.com/kurrent-io/KurrentDB> (~5,800 ⭐)
- **UI Repo:** <https://github.com/kurrent-io/EventStore.UI> (59 ⭐)
- **Language:** C#, JS/Angular
- **License:** BSD-3 / Commercial

**Stream Browser (OSS):** Browse individual aggregate streams, see events paginated in order. You see raw events, not the folded aggregate state.

**Visualize Tab (Enterprise/Cloud only):** The closest thing to what we want. Uses `$correlationId` and `$causationId` metadata to build a causal graph of events — showing the full lifecycle of a business entity (e.g., an order from placement through payment, shipping, delivery). Events are nodes, causal relationships are edges. Branches where multiple events share a causation ID. Requires the `$by_correlation_id` projection and events must carry proper metadata.

**Gap:** Shows event relationships, not the folded state. No slider/timeline to scrub through aggregate history. Enterprise-only for the visualize tab.

---

### Prooph Board

- **Website:** <https://prooph-board.com/>
- **GitHub:** <https://github.com/proophboard>
- **License:** Free (3 seats), Business €6/seat/mo, Enterprise custom

Event modeling tool with an interactive timeline showing commands (blue), events (orange), read models (green), UI elements (gray), and automation (purple). Real-time collaboration, AI integration via Model-Context-Protocol.

**Gap:** Design-phase only. No runtime data — it models the system you *intend* to build, not the events that actually happened. You can't point it at a running event store and scrub through aggregate history.

---

### Redux DevTools

- **Repo:** <https://github.com/reduxjs/redux-devtools> (20k+ ⭐)
- **License:** Free (browser extension)

The gold standard for event→state visualization. Shows every dispatched action, the state before/after, a diff highlight of what changed, and allows jumping to any point in history (time travel). Proves the pattern works and is incredibly useful.

**Gap:** Only works for Redux (client-side JavaScript). Not applicable to server-side event sourcing. The pattern of "action → state diff → time travel" is exactly what we want, but nobody has built the server-side equivalent.

---

### XState Stately

- **Website:** <https://stately.ai/viz>
- **License:** Free (basic), paid (teams)

Interactive statechart visualization. Send events to a machine, watch transitions happen on the diagram, inspect context/state. Excellent for finite state machines with explicit transitions.

**Gap:** Works for finite-state machines, not for event-sourced aggregates with arbitrary fold functions. An aggregate's state isn't limited to a finite set of named states — it's the cumulative result of folding every event. XState can't express that.

---

### Avala State Machine Panel

- **Website:** <https://avala.ai/docs/visualization/panels/state-machine-panel>

Runtime state machine monitoring with timeline scrubbing. Shows horizontal bar charts of state durations, full transition history, frequency stats. Multiple machines simultaneously.

**Gap:** Early stage, focused on robotics/embedded systems, not event sourcing.

---

### Replay.io

- **Website:** <https://www.replay.build/>

Records browser sessions, then auto-discovers state machines from the recording by analyzing user interactions. Generates flow maps.

**Gap:** Browser-only, not server-side ES. Discovers state machines post-hoc rather than visualizing known event streams.

---

### Marten

- **Repo:** <https://github.com/JasperFx/marten> (1.8k ⭐)
- **Language:** F#/C#, PostgreSQL

PostgreSQL-based event store and document store for .NET. Projections, async daemon, multi-tenancy.

**Gap:** No built-in visualization. Would need a custom dashboard built on top of the PostgreSQL event tables.

---

### Axon Server

- **Website:** <https://www.axoniq.io/server>
- **Grafana Dashboard:** ID 22622

Combined event store and command/event message router for JVM systems. Grafana dashboards for cluster health, event store metrics, message processing stats.

**Gap:** Dashboards show operational metrics, not aggregate state evolution. No timeline/scrubber for individual aggregates.

---

### Tools Rejected (Too Immature)

| Tool | Stars | Reason |
|---|---|---|
| TimeTravelDebugger | 0 | 1 commit, 0 ⭐, React/Three.js demo — toy project |
| CQRS Event Streaming | 0 | 6 commits, template only |
| Event-Sourcing-Visualizer (wbbrick) | 0 | 2016-era Elm, abandoned |
| EventSourcingDB (thenativeweb) | N/A | No built-in UI, commercial license (free ≤25k events) |
| EventSourcing.NetCore | 3.7k ⭐ | Educational examples only, no visualization |

---

## The Gap

**No production-ready tool exists that does the following:**

1. Connect to an event store
2. Select an aggregate stream
3. Replay events through the fold function
4. Show the aggregate state evolving event-by-event with a timeline/slider
5. Display state diffs between events
6. Allow jumping to any point in history (time travel)

This is exactly what Redux DevTools does for Redux, but translated to server-side event sourcing. The pattern is proven. The value is obvious (debugging, auditing, understanding). Nobody has built it.

**The closest approximations:**
- **KurrentDB Visualize Tab** — causal event graph, but no state folding, enterprise-only
- **Prooph Board** — design-time timeline, but no runtime data
- **Redux DevTools** — the right UX pattern, but client-side only

---

## Opportunity

A tool that does `event-store → select aggregate → replay through fold → timeline with state diffs` would fill a real gap. The building blocks exist:

- Event stores expose stream read APIs (KurrentDB, Marten, go-cqrs-lite `event.Store.Load`)
- Fold functions are deterministic and replayable by design
- Redux DevTools proves the UX pattern works

What's missing is the glue: a generic visualizer that takes a stream reader + a fold function and produces an interactive timeline.

---

*Researched 2026-05-03*
