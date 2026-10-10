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

## Deployment presets

One-liner `DeploymentConfig`s for tests that do not need to exercise a
specific engine topology:

| Preset                     | Layout                                                                        | Use when                                                |
| -------------------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------- |
| `systemscenario.Memory()`  | memory primary (journal + projections) + dedicated memory `timers` engine     | default fast path; `TimeAdvances` works out of the box  |
| `systemscenario.SQLite(t)` | file-backed SQLite primary under `t.TempDir()` (WAL) + memory `timers` engine | exercising real SQL planning, pragmas, file-backed data |

`SQLite(t)` uses a file DSN, not shared-cache in-memory: engines own and
close their `*sql.DB`, and shared-cache in-memory databases die with the
last connection — file DSNs keep reopen-style assertions honest.

## Phases

| Phase | Methods                                                                                                                                                                                                                                                                                                                                    | Axon analog                                          |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------- |
| Given | `Given(events...)`, `sc.Event(type, ref, payload)` (auto-versioned), `Given().Command(cmds...)` (seed by intent)                                                                                                                                                                                                                           | `given().events()/commands()`                        |
| When  | `When(cmd)`, `WhenEvent(events...)` (journaled + published external events), `WhenQuery(q)`, `TimeAdvances(d)`                                                                                                                                                                                                                             | `when().command()/event()`, Axon 4 `whenTimeElapses` |
| Then  | `Then(types...)`, `ThenEvents`, `ThenEventsSatisfy`, `ThenPayload` (generic), `ThenMetadata`, `ThenQuery`, `ThenQueryFunc`, `ThenQueryFails`, `ThenQueryEventuallyFails`, `ThenResult`, `ThenSuccess`, `ThenError`, `ThenErrorFamily`, `ThenCommands`, `ThenCommandsSatisfy`, `ThenCommandsSatisfyAwait`, `ThenNoEvents`, `ThenNoCommands` | `then().events()/commands()/exception()/success()`   |
| Modes | `Await()` (poll mode for async bus outcomes), options `WithAwaitTimeout`, `WithQuietWindow` (bounds `ThenNoEvents` silence under await), `WithCommandCaptureFilter` (capture noise control), `WithClock`                                                                                                                                   | —                                                    |

**Determinism contract:** journal/command assertions are synchronous
(dispatch writes the journal before returning); read-model assertions poll
(`ThenQuery`) because projection folding is asynchronous. `TimeAdvances`
advances the harness `ManualClock` (frozen at 2026-01-01Z) and flips into
poll mode — deadline timers fire without sleeping when your `Timers` closure
wires `scheduling.WithClock(sys.Clock().Now)`.

**Timeout honesty:** every polling assertion reports the LAST probe outcome
on timeout, and query assertions additionally carry the last query error
alongside the check mismatch — a projection that kept erroring before the
data settled is distinguishable from wrong data. `ThenQueryEventuallyFails`
is the eventual negative (rows that vanish once the projection catches up —
the first-class `awaitNotFound`); `ThenQueryFails` stays immediate.

**Negative-event windows:** under await mode `ThenNoEvents` watches for
silence for the quiet window — the full await timeout by default, shorter
via `WithQuietWindow` — failing the moment any event lands inside it.

**Vacuous guard:** a scenario that never runs a `Then*` fails the test.

**Saga capture:** the harness installs an always-on command-capture
middleware; `ThenCommands` diffs dispatched commands against the act
baseline — the event→command→event chain of a deriver is assertable
end-to-end (`Await()` first: bus delivery is asynchronous).

> **Known constraint (found by the harness saga test, 2026-10-09; ruled by
> [ADR-0154](../docs/adr/0154-deriver-async-dispatch-and-journal-tailed-host.md)):**
> a deriver subscribed via `sys.Bus().Subscribe` that dispatches derived
> commands **synchronously** deadlocks — the default event bus publishes with
> `BlockPublishUntilSubscriberAck`, so the derived dispatch re-publishes from
> inside the handler the publisher is waiting on (live stack evidence:
> [docs/evidence/2026-10-09_deriver-bus-deadlock.md](../docs/evidence/2026-10-09_deriver-bus-deadlock.md)).
> Wire sagas with `deriver.WithAsyncDispatch` (per-event goroutine, error
> callback) and assert outcomes via `Await()`; the saga fixture in this
> module's tests is the reference wiring.

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

## Cost of a scenario

`BenchmarkScenarioBoot` measures the full per-scenario cost a harness test
pays: `system.New` + middleware + Start + one given + one act + one Then +
GracefulClose, on `Memory()` engines. Snapshot (2026-10-10, 32-vCPU host,
`go test -bench BenchmarkScenarioBoot -benchmem`):

```
BenchmarkScenarioBoot-32  ~1.2–1.8 ms/op   ~9.2 MB/op   ~1371 allocs/op
```

The 9 MB is the composition root itself (engines, dispatcher, bus,
projection host, middleware chains) — one scenario is one real boot, which
is the point. Suites with hundreds of scenarios stay in the seconds; use
`scenario` (decider-pure) when you need microsecond loops instead.

Correctness depth: `TestProperty_ReadModelMatchesFold` (rapid) drives random
create/rename/complete sequences and asserts the projected read-model row
equals the pure fold over the journal — the fold-equivalence oracle; and
`TestProperty_RandomCommandSequencesKeepJournalOrdered` pins journal
versioning under adversarial dispatch order.

## Chaos and SSE

Two legs that take a scenario beyond the happy path:

- **`DelayedDriver(t, base, delay)`** — journal-latency chaos. Wraps any
  registered base driver (e.g. `"memory"`) with a `delay` sleep before every
  journal-family operation, registers it under a process-unique name, and
  returns that name for `EngineConfig{Driver: name}`:

  ```go
  deploy := systemscenario.Memory()
  deploy.Engines["primary"] = system.EngineConfig{
      Driver: systemscenario.DelayedDriver(t, "memory", 2*time.Millisecond),
  }
  ```

  Map operations pass through undelayed, so read models stay fast — the
  chaos targets the journal I/O path. Booting a delayed deployment at all
  proves capability forwarding (system.New's atomicity gate asserts
  `AtomicAppender`/`Transactional` on the wrapper; Go does not tunnel type
  assertions through embedding, so the wrapper forwards them by hand), and
  a concurrent-save race proves optimistic concurrency still serializes
  under latency (`chaos_test.go`).

- **`SubscribeSSE[V](t, ctx, sc, collection)`** — SSE assertions over the
  production wire path. Serves a projection collection from the scenario's
  metaengine store via `metaengine.ServeSSE` on an in-process HTTP server
  and connects as the first client, decoding events as `V`:

  ```go
  sub := systemscenario.SubscribeSSE[TaskView](t, ctx, sc, "task_views")
  // ... Given / When acts ...
  sub.Await(t, 5*time.Second, "renamed task view", func(v TaskView) bool {
      return v.Title == "renamed"
  })
  ```

  Subscribe BEFORE the acts whose projection changes you want to observe —
  notifications fire on writes. An `Await` passing here means a browser
  EventSource on the same endpoint sees the value too.

## Status

Experimental (rides `system`'s experimental status, FEATURES.md). Registered
in the module gates (layers, dep budget, api-stability golden).
