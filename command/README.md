# command — CQRS Command Dispatch

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/command/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/command/v2)

Typed command dispatch with middleware chains and lifecycle management.

```bash
go get github.com/larsartmann/go-cqrs-lite/command/v2
```

## Quick Start

```go
cmds := command.NewDispatcher()
cmds.Register("user.create", handler)
err := cmds.Dispatch(ctx, cmd)
```

## Typed Handlers

```go
command.RegisterTyped[CreateUserCmd](cmds, "user.create",
    func(ctx context.Context, cmd *CreateUserCmd) error {
        return handleCreate(cmd)
    },
)
```

## Key Types

| Type              | Purpose                                                     |
| ----------------- | ----------------------------------------------------------- |
| `Dispatcher`      | Command dispatcher with handler registry + middleware chain |
| `Command`         | Interface: Type(), AggregateID(), IdempotencyKey()          |
| `BasicCommand`    | Embed in command structs for interface satisfaction         |
| `TypedHandler[T]` | Type-safe handler receiving T, not Command                  |
| `Middleware`      | func(Handler) Handler — wraps handlers in a chain           |
