# testutil — Shared Test Helpers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/testutil/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/testutil/v3)

Cross-module test utilities: test-friendly command constructors (tb.Fatalf on error, zero panics) and no-op handlers for use in test suites across the go-cqrs-lite ecosystem.

```bash
go get github.com/larsartmann/go-cqrs-lite/testutil/v3
```

## Quick Start

```go
import (
    "testing"

    "github.com/larsartmann/go-cqrs-lite/command/v3"
    "github.com/larsartmann/go-cqrs-lite/testutil/v3"
)

// Build a command in tests (calls t.Fatalf on invalid input — no panics)
cmd := testutil.NewCmd(t, "user.create", aggID)

// Use in test setup
dispatcher := command.NewDispatcher()
dispatcher.Register("ping", testutil.NoopCommandHandler())
```

## API

| Function                              | Purpose                                               |
| ------------------------------------- | ----------------------------------------------------- |
| `NewCmd(tb, cmdType, aggID, opts...)` | Wraps `command.New`, calls `tb.Fatalf` on error       |
| `NoopCommandHandler()`                | Returns a `command.Handler` that always returns `nil` |

## Dependencies

| Dependency                         | Purpose                          |
| ---------------------------------- | -------------------------------- |
| [command/v3](../command/README.md) | Command types and `BasicCommand` |
| [id/v3](../id/README.md)           | `AggregateID` type               |

## Related Modules

- [command/v3](../command/README.md) — Command dispatch and typed handlers
- [id/v3](../id/README.md) — Branded IDs
