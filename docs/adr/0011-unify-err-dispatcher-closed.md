# ADR 0011: Unify ErrDispatcherClosed Across Packages

## Status

Proposed

## Context

Three packages define `ErrDispatcherClosed` with identical purpose but separate instances:

| Package      | Error Code                     | Type           |
| ------------ | ------------------------------ | -------------- |
| `dispatcher` | `dispatcher.dispatcher_closed` | Infrastructure |
| `command`    | `command.dispatcher_closed`    | Infrastructure |
| `query`      | `query.dispatcher_closed`      | Infrastructure |

Consumers who dispatch both commands and queries cannot write a single `errors.Is(err, dispatcher.ErrDispatcherClosed)` check that covers both. The same logical condition ("dispatcher is closed") produces different error instances depending on which dispatcher type is used.

## Decision

For v3, define a single sentinel in `dispatcher/`:

```go
// dispatcher/errors.go
var ErrDispatcherClosed = errorfamily.NewInfrastructure(
    "dispatcher.closed",
    "dispatcher is closed",
)
```

Re-export from `command` and `query`:

```go
// command/errors.go
var ErrDispatcherClosed = dispatcher.ErrDispatcherClosed

// query/errors.go
var ErrDispatcherClosed = dispatcher.ErrDispatcherClosed
```

## Consequences

- **Breaking change** — `errors.Is(err, command.ErrDispatcherClosed)` still works, but the error code changes from `command.dispatcher_closed` to `dispatcher.closed`
- Single `errors.Is` check covers all dispatcher types
- `command` and `query` gain a dependency on `dispatcher` (they already have it)
- Error codes become platform-level rather than module-level
