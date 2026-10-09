# ADR-0153: System-Level BDD Testing Harness (`systemscenario`)

- Status: Accepted (owner granted Full Execution Mode 2026-10-09, approving the three
  embedded recommendations of
  [`docs/planning/2026-10-09_04-04_SUPERB-bdd-testing-harness-pareto-plan.md`](../planning/2026-10-09_04-04_SUPERB-bdd-testing-harness-pareto-plan.md) as recommended)
- Date: 2026-10-09
- Deciders: owner (Full Execution Mode grant), via the BDD-harness Pareto plan
- Related: ADR-0123 (v5 unification, single composition root — the enabling precondition),
  ADR-0152 (fleet-first topology, v5 dual-support), ADR-0136 (temporal composability),
  ADR-0142 (engine-backed timers), ADR-0114 (tombstone as domain event)

## Context

`scenario/v4` (Production, FEATURES.md) covers deciders + projections as a pure
functional core — it imports only `event` + `projection` and never boots
infrastructure. Nothing in the repo dispatches a command through a real
`system.New` boot and asserts across the full loop
(command → dispatcher → decider → store → bus → projection → query).

Companion evidence (2026-10-09 surveys):

- **cqrs-htmx** (`systemadapter/declarative_test.go`, 1242 lines): 51 raw
  `Dispatch` sites, ~50 hand-rolled `eventually(...)` poll blocks (~300 lines of
  assertion scaffolding with string-only failure messages), the
  `RegisterUserCmd` given-prerequisite re-derived 22× across files, and ~100
  lines of duplicated boot/wiring. It never asserts raw events; every assert is
  a polling query.
- **go-appkit** (thin `system` wrapper, 5 production files in `cqrs/`): zero
  `scenario/v4` usage anywhere; tests hand-roll boot + `waitFor` polling
  (`domain_test.go:561-565`).
- **deriver** has 16 unit tests covering composition/filtering/idempotency
  against stub dispatchers, but no end-to-end event→command→event saga story.

### Verified Axon Framework 5 lessons (primary sources, 2026-10-09)

Re-verified against docs.axoniq.io 5.3 + apidocs.axoniq.io (the earlier
session's saved pages were garbage-collected; two session claims were WRONG and
are corrected here):

| Claim | Verdict | Source |
|---|---|---|
| `AxonTestFixture.with(configurer)` replaces `AggregateTestFixture`/`SagaTestFixture` | ✅ verified (single unified fixture; official migration page) | docs.axoniq.io/…/migration/paths/test-fixtures/ |
| Any message in any given/when phase (`given().event/command`, `when().event/command/events`) | ✅ verified for commands+events | apidocs `AxonTestPhase.Given`/`.When` |
| Then surface: `events`, `eventsSatisfy`, `noEvents`, `commands`, `commandsSatisfy`, `noCommands`, `exception`, `success`, `resultMessagePayload`, `await`, `expect` | ✅ verified | apidocs `AxonTestPhase.Then` equivalents |
| `when().timeElapses(...)` on the Axon 5 fixture | ❌ **does not exist** — `whenTimeElapses`/`whenTimeAdvancesTo` are Axon 4 `AggregateTestFixture` APIs | apidocs (absent), Axon 4 docs |
| `whenQuery(...)` phase | ❌ **does not exist** — Axon tests queries only via `then().expect(config -> gateway.query(...))` | docs.axoniq.io 5.3 testing pages |
| Fixture built from the production `ApplicationConfigurer` (same config in prod and test) | ✅ verified — the load-bearing idea | docs.axoniq.io 5.3 basic-testing |

The lesson set we adopt: **fixture-from-production-config**, **any-message
phases**, **command-capture assertions**, **time control**. Two deliberate
divergences: we make queries a first-class `When().Query(...)` phase (queries
are first-class in go-cqrs-lite), and our time control follows the Axon 4
`whenTimeElapses` shape because Axon 5 dropped it.

## Decision

### D1 — New module `systemscenario/v4`

Module `github.com/larsartmann/go-cqrs-lite/systemscenario/v4`, package
`systemscenario`, registered as test-infrastructure (LAYER 7 in
`check-module-layers.sh`, the `systemtest`/`event/v4/eventtest` tier).
Direct deps: `system`, `event`, `command`, `query`, `id` (all in-repo) +
`go-error-family` (fleet-standard, already a dep of every core module).
Dep budget 6, mirroring `systemtest`.

Alternatives rejected:

- **Grow `scenario/v4` with a `system` dependency** — layer violation (L3
  aggregation importing L5 composition) and it would drag the full system dep
  tree into every functional-core scenario consumer.
- **Export from `systemtest`** — that module owns the repo's own real-engine
  suites (Feedback-#4 split); consumers importing it inherit suite-only deps.
- **A `system` subpackage** — the composition root must not grow test-tooling
  surface; module isolation keeps budgets honest.
- **A `scenario/v5` module path** — premature: no v5 paths exist yet
  (ADR-0152), and the pilot consumers (cqrs-htmx, go-appkit) are v4 codebases
  that must import the harness today. The v5 cut can re-home it with the rest
  of the wave.

### D2 — Phase-object Given/When/Then API on plain `testing.T`

```go
// skip-validate
sc := systemscenario.System(t, ctx, domainCfg, deployCfg) // boots system.New + Start + t.Cleanup

sc.Given(evtCreated).                     // seed journal (AppendBatch + bus publish)
    When(cmdComplete).                    // dispatch through the real dispatcher
    Then("task.completed")                // journal diff since the When baseline

sc.Given().Command(cmdRegister).          // seed by intent (dispatch-based given)
    When(cmdRename).
    ThenError(system.ErrNoDecider)        // errors.Is on the captured outcome

sc.Given(...).WhenEvent(evtExternal).ThenQuery(func() (any, error) {
    return metaengine.ExecuteTyped[...](ctx, sc.System().MetaEngine(), in)
}, wantView)                              // poll-await until projections settle

sc.When().Query(q).ThenResult(want)       // query as the act under test
sc.When().TimeAdvances(time.Hour).Then("order.cancelled") // timer firing (Axon 4 timeElapses analog)
```

- Style ruling (Q2): plain `testing.T` fluent chains, consistent with
  `scenario/v4`; Ginkgo/Gomega stays a consumer choice, never a library dep.
- Vacuous-assertion guard ported from `scenario/dsl.go:81`: a scenario whose
  chain never ran a `Then*` fails the test.
- `Given` seeds the journal via `AppendBatch` (no concurrency checks) AND
  publishes each event to the bus, mirroring the decider repository's
  save→publish order, so projections and saga subscribers observe given events.
- Event versions: callers construct events via `event.New`; the harness offers
  `sc.Event(type, payload, ref)` which auto-versions per stream
  (harness-tracked per-stream counters) so givens compose without manual
  version bookkeeping.

### D3 — Determinism contract

- **Journal asserts are synchronous**: dispatch saves events before it
  returns, so `Then`/`ThenEvents`/`ThenNoEvents`/`ThenCommands` diff the
  journal against a baseline captured when `When*` began — no polling.
- **Read-model asserts poll-await**: `ThenQuery` polls the query closure until
  the result deep-equals the want (or an `AwaitTimeout`, default 5 s, option
  `WithAwaitTimeout`), because bus delivery and projection folding are
  asynchronous (watermill GoChannel).
- **`TimeAdvances(d)`** advances the harness `ManualClock` and flips the
  scenario into await mode: subsequent `Then*` assertions poll until match
  (timer firing is scheduler-tick asynchronous).

### D4 — Clock seam: `system.Clock` + `WithClock` (Q3) with one documented deviation

- `system` gains `type Clock interface { Now() time.Time }`, `RealClock`,
  `ManualClock` (freeze/advance/set), a variadic `Option` on `system.New`
  (`system.WithClock(c)`), and `sys.Clock()`. The system surface is
  experimental (FEATURES.md), so additive options are stability-safe.
- The harness boots every scenario with a `ManualClock` seeded at a fixed
  epoch, exposing it via `sc.Clock()` so consumer `DomainConfig.Timers`
  closures can compute deterministic `FireAt` values and wire
  `scheduling.WithClock(sys.Clock().Now)`.
- **Documented deviation from plan guardrail 6** ("`scheduling/` untouched
  until v5"): `scheduling` gains a purely additive `WithClock(now
  func() time.Time)` option consumed by `Scheduler.tick` (scheduler.go:150
  reads `time.Now()` directly today). Rule broken: no scheduling changes until
  v5. Why: without the seam, `TimeAdvances` cannot make timers fire
  deterministically — it would be a fake feature (worse than no feature). The
  option is additive and non-breaking (api-stability golden grows one
  symbol); `scheduling` cannot import `system.Clock` (layering), hence the
  `func() time.Time` shape. Tradeoff recorded here and in the CHANGELOG.

### D5 — Command capture (saga story)

The harness installs a capture middleware on the command dispatcher before
`Start` (always on; negligible cost). `ThenCommands(types...)` /
`ThenCommandsSatisfy(inspect)` diff captured commands against the When
baseline — the first end-to-end deriver/saga assertion story (an event
published on the bus triggers a deriver whose derived command is captured).

### D6 — No duplication of the schema-evolution axis (T05)

Upcast-in-given and snapshot round-trip asserts are NOT built here; they ride
TODO_LIST §"V5 declarative schema evolution" T2 (named upcast ops) and T4
(snapshot state-shape stamp). This ADR's scope is dispatch-level BDD only.

## Consequences

- Consumers get Axon-style fixtures from the SAME `DomainConfig` their
  production binary boots — config drift between prod and tests becomes
  structurally impossible for harness-written tests.
- cqrs-htmx pilot (Train A: `declarative_test.go` user-lifecycle sub-train,
  ~330 lines / 8 tests) and go-appkit pilot (`cqrs/scenario_pilot_test.go`
  over the memory-driver facade) validate DX and feed API fixes before the
  docs wave claims wins.
- `scenario/v4` stays untouched-green throughout (parity proven, zero-cost
  delegations documented where free).
- Guardrails: zero v4 API breaks; no new production deps beyond in-repo
  modules + `go-error-family`; 350-line/30-line rules checked per phase; every
  phase exits green (`nix run .#check-arch`, `#check-file-size`,
  api-stability regen at each exported-symbol change).
- v5 note: when ADR-0152's v5 wave lands, `systemscenario` re-homes with the
  wave and may absorb `scenario/v4` (the decider-level DSL becomes the
  functional-core tier beneath the system tier).
