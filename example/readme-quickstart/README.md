# readme-quickstart — Minimal Command + Decider Example

The absolute minimum: define a command, a decider, execute it, and read back the state. No projections, no bus, no stack layer. Perfect for understanding the core building blocks.

## What It Demonstrates

- Defining a command type via embedded `*command.BasicCommand`
- Wiring a typed command handler via `command.RegisterTyped`
- Creating a `decider.Repository` with a fold function
- Executing a command that produces an event
- Loading the aggregate state from events

## Run It

```bash
cd example/readme-quickstart
GOWORK=off go run main.go
```

Output:

```
User: Alice
```

## How It Works

```go
// 1. Define domain types
type UserState   struct{ Name string }
type UserCreated struct{ Name string }

// 2. Command embeds *BasicCommand for the Command interface
type CreateUser struct {
    *command.BasicCommand
    Name string
}

// 3. Wire: store + bus + decider → repository
store := memory.NewMemoryStore()
bus   := cqrswatermill.NewEventBus()
repo, _ := decider.NewRepository(store, bus, d)

// 4. Register a typed command handler
command.RegisterTyped(cmds, "user.create", func(ctx, cmd *CreateUser) error {
    return repo.Execute(ctx, cmd.StreamID(), "User", decideFunc)
})

// 5. Dispatch → event sourced → state readable
cmds.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"})
state, _, _ := repo.Load(ctx, aggID, "User")
fmt.Printf("User: %s\n", state.Name) // "Alice"
```

## Related

- [**getting-started**](../getting-started/) — Full pipeline with projections and read models
- [**taskmanager**](../taskmanager/) — Flagship example: full HTTP service
