# Cordis → go-cqrs-lite: Mapping Spatiotemporal Composability

> **Date:** 2026-09-10
> **Kind:** Point-in-time concept-mapping analysis (no code changes)
> **Sources:**
>
> - Shi, Zhang, Cui — *A Programming Paradigm for Spatiotemporal Composability* (arXiv:2608.25512, cs.PL, Aug 2026; Peking University + DeepSeek-AI). Full text has no arXiv HTML rendering; grounding = the detailed abstract + the DeepSeek Harness Cordis primer (`deepseek-harness.github.io/deepseek-harness/reference/cordis-primer`), whose authors overlap with the paper.
> - Repo state as of `master` @ 23ffe380a; every cited mechanism spot-verified with `rg` before writing.

---

## 1. The paradigm in brief

Cordis formalizes **dynamic composition** along two orthogonal axes:

- **Temporal composability** — *revertible effects*: every context transformation carries an inverse that the runtime holds; removing a component completely undoes its footprint.
- **Spatial composability** — *reactive coeffects*: components declare what they need (a coeffect specification); every context change is classified against that spec to drive the component's activation and deactivation.
- Both are unified through a single **context type** (the *context paradigm*), mediating every effect and coeffect — inducing an observational equivalence under which the effects of distinct components interleave without disturbing one another.
- Implementation: a core library (effect tracking + coeffect resolution) plus a declarative component loader (config reconciliation + hot module replacement).

The DeepSeek Harness primer translates this to practice: plugins implement `Service`; the context is a service container (`ctx.tools`, `ctx.llm`); `inject` declares dependencies and waits for readiness; typed events dispatch as `emit` / `waterfall` / `parallel` / `serial` / `bail`; every registration is reversible (`ctx.effect()` / `ctx.on()` return disposers, undone on reload/teardown).

## 2. Thesis: one paradigm, three levels of dynamism

Go cannot — and should not — host this at runtime the way TypeScript plugin systems do. The mapping therefore lands on three levels, and go-cqrs-lite is notable because it already lives at the third:

| Level | Where dynamism lives | Temporal axis | Spatial axis |
| --- | --- | --- | --- |
| Cordis | Runtime (loader, HMR) | `ctx.effect()` disposers | `inject` + reactive activation |
| **go-modularize skill** | Compile time (go.mod DAG, versions) | Versioned deprecation waves; revertible migration steps | Import DAG, interface seams, composability payoff |
| **go-cqrs-lite** | Data (event journal) | Append-only log, rebuild-by-replay, tombstone/rebirth | `EventTypes()` coeffect specs, cost-planned reactive routing |

## 3. Concept-by-concept mapping (all sites verified)

| Cordis concept | go-cqrs-lite mechanism | Verified site |
| --- | --- | --- |
| Component / `Service` | Independently versioned Go module; consumer composes via `system.New` | 82 `go.mod` files |
| Context = service container (`ctx.<key>`) | **Interfaces are the compile-time service lookup**; capability interfaces (`MultiSink`, `StreamingJournal`, …) discovered by type assertion and never dropped by wrappers | `event/store_middleware.go:64,123` (ADR-0126) |
| `inject` (wait-for-readiness deps) | Constructor params + `DomainConfig` / `DeploymentConfig` split; `validateShutdownDependencies` edges | `system/shutdown.go:37` |
| Reactive activation/deactivation | `CheckRouting` + `ReplanLayout` re-route on live latency drift (hysteresis-gated); `System.Close` deactivates in dependency order | `metaengine/store_routing.go:40`, `metaengine/relayout.go:64` |
| `ctx.effect()` reversible registration | Disposer discipline: `Closer` / `DeferClose` (47 production sites), `RegisterCloser`, error-joined `Close`, drain-then-close `GracefulClose` | `metaengine/engine.go:593,600`; `system/system.go:260,302` |
| Typed events (`emit`/`waterfall`/`parallel`/`serial`/`bail`) | Dispatcher middleware chain = **waterfall** (each middleware wraps `next`; chain rebuilt on `Use`); bus publish = emit; projection fan-out = parallel/serial; circuit breaker = bail | `dispatcher/dispatcher.go:72`; `middleware/` |
| Observational equivalence | Stream isolation (per-aggregate journals) + `ImmutableEvent` + defensive clones + decider purity (`decide` is pure) | `event/`, `decider/` |
| Declarative loader + config reconciliation | `metaengine.RegisterDriver` — database/sql-style `init()` self-registration = Go's canonical plugin registry (per-module `register.go` clones are intentional, ADR-governed) | `metaengine/registry.go:61` |
| HMR | Impossible / undesired in Go → shifted to **data-level evolution**: `UpcastSourceTransform` (old components' events stay readable by new code); `RebuildThreshold` / `ConfirmRebuild` re-materializes projections without downtime | `schema/versioned_source.go:20`; `metaengine/relayout.go:171` |

**Negative finding (hygiene):** `samber/do` appears in the `go.mod` of five `cmd/*` modules (`doc-check`, `cqrs-gen`, `api-stability`, `cqrs-bench`, `cqrs-lint`) with **zero** `.go` importers — stale requires, not DI usage. `go mod tidy` per module would clear them.

## 4. The deep correspondence

**go-cqrs-lite is a context paradigm for data, already.** The journal + bus *is* the unified context: every write is mediated through the store (effect side); every observer subscribes through projections (coeffect side). Concretely:

1. **Event sourcing = temporal composability at the domain level.** A projection's entire side-effect footprint is revertible *by replay* — drop the read model, rebuild from the journal. ADR-0114 goes further: deletion itself is a domain event (`user.deleted` tombstone, rebirth as inverse) — literally "an inverse the runtime holds," expressed inside the domain rather than in a plugin disposer.
2. **`projection.Projection` is a coeffect specification.** The interface is `Name() + Handle(ctx, evt) + EventTypes()` (`projection/projection.go:23-27`). The projection *declares what it needs*; `projectionhost` builds a type set (`projectionhost/host.go:127`) and classifies every incoming event against it to decide activation. That is, almost word-for-word, the paper's reactive coeffects. Likewise `metaengine.Watcher[V].Watch(ctx, key) <-chan V` (`metaengine/dx.go:29,150`) is reactive push on context change.
3. **The go-modularize skill is the compile-time projection of both axes.**
   - Its Phase 5 rule "each step independently revertible" is revertible effects applied to the *development process* itself.
   - DAG enforcement and the composability-payoff litmus test ("does any consumer import A without B?") are static coeffect specifications.
   - Failure mode FM#7 (contract errors live in the interface module, not implementations) makes error identity part of the coeffect spec.
   - Phase 1.5 deprecation planning (deprecate → sunset → remove) = inverse-carrying component removal — exactly what the repo's v5 wave (ADR-0123 single composition root, ADR-0126 transform-canonical forms, ADR-0127 transport removal) is executing now.

## 5. Honest gaps — where Cordis exceeds the repo

Mostly platform trades, not defects:

- **Runtime component removal/reload**: no HMR; "removing a component" is a versioned migration (deprecated shells → v5 removal), not `ctx.dispose()`. Go's answer: redeploy + upcasts. Deliberate.
- **Reversibility asymmetry**: `Closer` covers resources, not in-flight semantic effects. True inversion exists only where data is evented (replay, snapshots, upcasts); external side effects (HTTP calls, emails) need compensating events — the `deriver` saga pattern is the honest answer, not the disposer.
- **Reactivity is routing-scoped**: `AutoReplan` reacts to live latency drift, but components do not re-activate when a *service* appears later (Cordis's `inject` waits and re-evaluates). In Go this resolves at construction time — acceptable, but it means the "coeffect paradox" (missing dependency) must fail loudly at build/compose time, which the skill's *fail noisily* principle already mandates.
- **Stale `samber/do` requires** in 5 `cmd/*` go.mod files (see §3).

## 6. Pareto-ranked implications (observations, not executed)

1. **Codify the temporal contract** (~zero cost, high clarity): a short ADR stating which effects are runtime-invertible (replayable projections, snapshots, upcasts) vs compensation-only (external effects → `deriver`). The paper supplies the vocabulary; it names what ADR-0114 already gestures at.
2. **Name `EventTypes()` what it is**: docs/comments calling it the projection's *coeffect specification* would connect `projectionhost`, metaengine routing, and `Watcher` under one mental model — likely improving future planner-level design discussions.
3. **`go mod tidy` the five `cmd/*` modules** to drop the phantom `samber/do` requires.

## 7. The metaengine: where the paradigm runs at runtime

The metaengine is the point of **strongest convergence** between this repo and the paper — the one place where the paradigm operates at *runtime* on *persistent state*, not just at compile time.

### The vision statement is the context paradigm, verbatim

> *"Developers declare ONLY Commands + Events + Queries and their relationships … while where data lives is up to operators at DEPLOYMENT time."* (AGENTS.md, north star)

That is Cordis's spatial composability applied to **data placement**: the developer writes a coeffect specification (query shapes + relationships), the operator performs config reconciliation (which engines exist), and the cost-based planner is the **unified context** mediating between them. The paper's observational-equivalence clause has a direct analog: a query returns the same answer whichever engine serves it — routing is transparent to the developer.

### The five primer concepts, mapped to metaengine (verified)

| Cordis concept | Metaengine mechanism | Site |
| --- | --- | --- |
| Plugin / `Service` | An engine is literally **`Profile() + Closer`** — a capability declaration (coeffect side) fused with a disposer (temporal side) in one 3-line interface | `metaengine/engine.go:616-619` |
| Loader / registry | Dep-isolated `metaengine/*engine` modules self-register via `init()` → `RegisterDriver` (database/sql pattern); conformance harnesses validate "plugins" (`adttest.RunMatrix`, `RunCapabilityConformance`) | `metaengine/registry.go:61` |
| `ctx.<key>` + optional capabilities | `Store` + collections; planner discovers optional capability interfaces per engine — `MapUpdater`, `PushdownScan`, `StreamingScan`, `LayoutPlanner`, `RawValueReader`, `RawScanReader` — and degrades plans gracefully | `metaengine/engine.go:267-437` |
| Reactive activation | `ProbeEngine` background loop → `LatencyTracker` (EWMA, 512-sample window, 30s staleness) → calibration precedence (compile-time priors → calibration → live measurement) → `CheckRouting` diagnostics → reroute, debounced by `WithRoutingHysteresis` / `WithRoutingMinDelta` | `metaengine/probe.go:209`, `store_routing.go:40`, `planner.go:106,119` |
| Reversible registration (`ctx.effect()`) | **Two-phase layout migration**: `ReplanLayout` → `[]LayoutDiff{From, To, Reason, EstimatedRebuildEvents}` → gated by `RebuildThreshold` (≤100K events / ≤1GB = automatic; larger = operator `ConfirmRebuild`, ADR-0124 §11) | `metaengine/relayout.go:25,64,171` |

### The two axes, metaengine-specifically

- **Spatial (reactive coeffects):** "which engine serves which collection" is never hard-wired. Context changes — live latency drift — are *classified against* each engine's profile to re-route. This is genuine runtime reactivity, the thing the rest of the repo delegates to compile time.
- **Temporal (revertible effects):** layouts are invertible *because projections are derived data*. Moving placement = drop the materialized view (effect undone) + rebuild from the immutable journal elsewhere. `RebuildThreshold` / `ConfirmRebuild` is "the runtime holds the inverse," made explicit and safety-gated — `LayoutDiff.EstimatedRebuildEvents` even *prices* the inverse before committing to it.

### Where it deliberately diverges from Cordis

Cordis's `inject` re-evaluates continuously and components hot-swap; the metaengine's reactivity is **routing-scoped, not engine-scoped** — engines open at composition time, and drift detection is advisory and heavily debounced. That is correct for the domain: flapping *plugin wiring* costs a reload; flapping *data placement* costs a rebuild. Hysteresis, min-delta floors, and operator confirmation are the data-grade versions of Cordis's reactivity.

**One line:** the metaengine = the context paradigm where the context is *persistent data placement*, effects are *materialized layouts* (invertible via rebuild from the journal), and coeffects are *query/relationship declarations* classified against capability profiles with live-measured costs.

## 8. Bottom line

The paper supplies the formal vocabulary this architecture already speaks: revertible effects = event sourcing + disposers; reactive coeffects = subscription specs + replan; the unified context = the journal mediated through the composition root — and in the metaengine, the paradigm runs at runtime over data placement itself. The go-modularize skill is the compile-time twin of the same two axes. The repo sits — unusually comfortably — at their intersection; the only true divergences (HMR, runtime re-activation) are Go-platform trades already compensated by versioning discipline and data-level evolution.

## 9. Where this went next (cross-links, appended 2026-09-10)

This report is a point-in-time artifact; the observations in §6 were operationalized without editing the body above. Follow the work here:

- **Execution plan:** [`docs/planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md`](../planning/2026-09-10_08-10_SUPERB-cordis-paradigm-pareto-execution.md) — ranks every §6/§7 implication into Pareto waves (W0 truth+durability → W1 temporal-correctness core → W2 paradigm operationalized → W3 final 20%), 27 medium tasks (M-01..M-27) + an 85-task ≤12-minute micro breakdown, with a mermaid execution graph and per-wave verification gates.
- **Living backlog:** [`TODO_LIST.md`](../../TODO_LIST.md) §"Cordis spatiotemporal-composability follow-ups" — the durable, harvestable form of the remaining (Wave 1-3) work; completed items move to CHANGELOG per repo policy.
- **Session status:** [`docs/status/2026-09-10_08-04_cordis-paradigm-mapping-session.md`](../status/2026-09-10_08-04_cordis-paradigm-mapping-session.md) — the a-g status snapshot that seeded the plan.
- **Guardrail (unchanged):** every behavior change derived here is warn-first in v4.x and hard-errors only at v5, riding the existing ADR-0123 composition-root wave. HMR, service-locator `ctx.<key>`, and implementing the paper's formal calculus remain explicitly declined as paradigm-envy (§5).
