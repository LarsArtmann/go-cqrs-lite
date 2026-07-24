# scenario — Fluent BDD Test DSL for Deciders and Projections

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/scenario/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/scenario/v4)

Test CQRS deciders and projections with a fluent Given/When/Then DSL. Minimal dependencies: imports `event/` and `projection/` only, keeping the test surface lightweight.

```bash
go get github.com/larsartmann/go-cqrs-lite/scenario/v4
```

## Quick Start

### Decider Testing

```go
func TestIncrement(t *testing.T) {
    scenario.Given[t, CounterState](t, foldCounter, CounterState{},
        mustEvent(evtIncremented),
    ).
        When(incrementCmd{Amount: 5}, decideIncrement).
        Then(event.Type("counter.incremented"))
}
```

Assert on errors or final state:

```go
// Assert the decide function returns a specific error:
scenario.Given(incrementFold, CounterState{}, ...).
    When(cmd, decide).
    ThenError(ErrAlreadyExists)

// Assert the final state after folding produced events:
scenario.Given(incrementFold, CounterState{}, ...).
    When(cmd, decide).
    ThenState(incrementFold, CounterState{}, expectedState)
```

### Projection Testing

```go
func TestUserProjection(t *testing.T) {
    scenario.GivenProjection(t, userProj, evt1, evt2, evt3).
        ThenNoError()

    // Or assert that at least one handler error occurred:
    scenario.GivenProjection(t, proj, badEvent).
        ThenError()
}
```

## API

### Decider Scenario

| Symbol                              | Kind   | Description                                                      |
| ----------------------------------- | ------ | ---------------------------------------------------------------- |
| `Given[Cmd, State](...)`            | Func   | Pre-existing events folded into initial state. Returns scenario. |
| `GivenState[State](...)`            | Func   | Convenience: pins `Cmd` to `any` for state-only tests.           |
| `.When(cmd, decide)`                | Method | Runs the decide function against the folded state.               |
| `.Then(types...)`                   | Method | Asserts the emitted event types match (order-sensitive).         |
| `.ThenError(target)`                | Method | Asserts the decide function returned an error matching `target`. |
| `.ThenState(apply, init, expected)` | Method | Folds produced events, asserts final state via `DeepEqual`.      |

### Projection Scenario

| Symbol                                | Kind   | Description                                  |
| ------------------------------------- | ------ | -------------------------------------------- |
| `GivenProjection(t, proj, events...)` | Func   | Feeds events to `proj.Handle` immediately.   |
| `.ThenNoError()`                      | Method | Asserts zero handler errors.                 |
| `.ThenError()`                        | Method | Asserts at least one handler error occurred. |

## Design

- **Pure functional approach**: Each `Then*` re-folds given events from scratch, ensuring test isolation.
- **Type parameters**: `Given[Cmd, State]` binds the command and state types at compile time. `GivenState[State]` is a convenience for state-only scenarios.
- **Minimal dependencies**: `DecideFunc` is intentionally separate from `decider.DecideFunc` (command-first vs version-first) to avoid importing `snapshot`, `otel`, and `storage/memory`.
- **Event types, not payloads**: `.Then()` compares event types only. For payload assertions, inspect the events directly.

## Related Modules

- [**decider**](../decider/README.md) — The `Decider[State]` type under test
- [**projection**](../projection/README.md) — The `Projection` interface under test
- [**event**](../event/README.md) — Event types used in scenarios
