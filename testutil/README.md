# testutil — Shared Test Helpers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/testutil/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/testutil/v2)

Cross-module test utilities: panic-on-error command constructors and no-op handlers for use in test suites across the go-cqrs-lite ecosystem.

```bash
go get github.com/larsartmann/go-cqrs-lite/testutil/v2
```

## Quick Start

```go
import (
    "github.com/larsartmann/go-cqrs-lite/command/v2"
    "github.com/larsartmann/go-cqrs-lite/testutil/v2"
)

// Build a command without handling the error (panics on invalid input)
aggID := testutil.ParseAggID("01HXYZ...")
cmd := testutil.MustNewCmd("user.create", aggID)

// Use in test setup
dispatcher := command.NewDispatcher()
dispatcher.Register("ping", testutil.NoopCommandHandler())
```

## API

| Function                              | Purpose                                               |
| ------------------------------------- | ----------------------------------------------------- |
| `MustNewCmd(cmdType, aggID, opts...)` | Wraps `command.New`, panics on error                  |
| `ParseAggID(s)`                       | Wraps `id.ParseAggregateID`, panics on error          |
| `NoopCommandHandler()`                | Returns a `command.Handler` that always returns `nil` |

## Dependencies

| Dependency                         | Purpose                          |
| ---------------------------------- | -------------------------------- |
| [command/v2](../command/README.md) | Command types and `BasicCommand` |
| [id/v2](../id/README.md)           | `AggregateID` parsing            |

## Related Modules

- [command/v2](../command/README.md) — Command dispatch and typed handlers
- [id/v2](../id/README.md) — Branded IDs
