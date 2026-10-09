# systemscenario

System-level BDD testing harness for [go-cqrs-lite](../) — Given/When/Then
chains over a **real `system.New` boot** (ADR-0153).

```go
import "github.com/larsartmann/go-cqrs-lite/systemscenario/v4"

func TestTaskCompletion(t *testing.T) {
	ctx := context.Background()
	sc := systemscenario.System(t, ctx, taskDomain(), memoryDeployment())

	sc.Given(
		sc.Event("task.created", ref, TaskCreated{Title: "ship it"}),
	).When(cmdComplete).
		Then("task.updated").
		ThenQuery(viewQuery(sc, ctx, ref), TaskView{Title: "ship it", Status: "completed"})
}
```

The harness boots the SAME `DomainConfig`/`DeploymentConfig` your production
binary uses — fixture-from-production-config — so harness tests cannot drift
from production wiring. There is nothing to re-declare.

## Phases

| Phase | Methods                                                                                                                                                                                                                                                          | Axon analog                                          |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| Given | `Given(events...)`, `sc.Event(type, ref, payload)` (auto-versioned), `Given().Command(cmds...)` (seed by intent)                                                                                                                                                 | `given().events()/commands()`                        |
| When  | `When(cmd)`, `WhenEvent(events...)` (journaled + published external events), `WhenQuery(q)`, `TimeAdvances(d)`                                                                                                                                                   | `when().command()/event()`, Axon 4 `whenTimeElapses` |
| Then  | `Then(types...)`, `ThenEvents`, `ThenEventsSatisfy`, `ThenPayload` (generic), `ThenMetadata`, `ThenQuery`, `ThenQueryFunc`, `ThenResult`, `ThenSuccess`, `ThenError`, `ThenErrorFamily`, `ThenCommands`, `ThenCommandsSatisfy`, `ThenNoEvents`, `ThenNoCommands` | `then().events()/commands()/exception()/success()`   |
| Modes | `Await()` (poll mode for async bus outcomes), options `WithAwaitTimeout`, `WithClock`                                                                                                                                                                            | —                                                    |

**Determinism contract:** journal/command assertions are synchronous
(dispatch writes the journal before returning); read-model assertions poll
(`ThenQuery`) because projection folding is asynchronous. `TimeAdvances`
advances the harness `ManualClock` (frozen at 2026-01-01Z) and flips into
poll mode — deadline timers fire without sleeping when your `Timers` closure
wires `scheduling.WithClock(sys.Clock().Now)`.

**Vacuous guard:** a scenario that never runs a `Then*` fails the test.

**Saga capture:** the harness installs an always-on command-capture
middleware; `ThenCommands` diffs dispatched commands against the act
baseline — the event→command→event chain of a deriver is assertable
end-to-end (`Await()` first: bus delivery is asynchronous).

> **Known constraint (found by the harness saga test, 2026-10-09):** a
> deriver subscribed via `sys.Bus().Subscribe` that dispatches derived
> commands **synchronously** deadlocks — the default event bus publishes with
> `BlockPublishUntilSubscriberAck`, so the derived dispatch re-publishes from
> inside the handler the publisher is waiting on. Derive asynchronously (go
> routine) until the product fix lands.

## Relationship to scenario/v4 (parity)

`scenario/v4` stays the **functional-core** tier: pure decider/projection
testing with zero infrastructure (imports only `event` + `projection`).
`systemscenario` is the **system** tier: full composition root, real
dispatch, journal, bus, projections, timers. Neither replaces the other:

| scenario/v4 (unchanged, Production)                                 | systemscenario                                                             |
| ------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `Given[Cmd,State](t, apply, initial, events)` folds state in memory | `Given(events)` seeds the real journal + bus                               |
| `When(cmd, decide)` calls your decide func directly                 | `When(cmd)` dispatches through the system dispatcher                       |
| `Then(types)` on decide's return value                              | `Then(types)` journal diff since the act baseline                          |
| `ThenEvents(inspect)`                                               | `ThenEvents(inspect)` + `ThenEventsSatisfy` (polling)                      |
| `GivenProjection(t, proj, events)` + `ThenQueryResult(fn, want)`    | `Given(...)` + `When(cmd)` + `ThenQuery(fn, want)` (awaits the async host) |
| `ThenError(target)`                                                 | `ThenError(target)` + `ThenErrorFamily(family)`                            |
| vacuous guard (`dsl.go`)                                            | vacuous guard (ported)                                                     |
| —                                                                   | `WhenEvent`, `WhenQuery`, `ThenCommands`, `TimeAdvances`, `Await`          |

Use `scenario` for fast decider-pure unit loops; use `systemscenario` when
the wiring, projections, sagas, or timers are under test.

## Status

Experimental (rides `system`'s experimental status, FEATURES.md). Registered
in the module gates (layers, dep budget, api-stability golden).
