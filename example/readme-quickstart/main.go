package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

type (
	UserState   struct{ Name string }
	UserCreated struct{ Name string }
)

// CreateUser implements command.Command via embedded *BasicCommand.
type CreateUser struct {
	*command.BasicCommand

	Name string
}

func main() {
	ctx := context.Background()
	store := memory.NewMemoryStore()
	bus := cqrswatermill.NewEventBus()

	d := decider.Decider[UserState]{
		Initial: UserState{},
		Apply: func(s UserState, e event.Event) (UserState, error) {
			p, _ := event.DecodePayloadAuto[UserCreated](e)
			s.Name = p.Name

			return s, nil
		},
	}
	repo, _ := decider.NewRepository(store, bus, d)

	cmds := command.NewDispatcher()
	aggID := id.NewStreamID()
	_ = command.RegisterTyped(cmds, "user.create",
		func(ctx context.Context, cmd *CreateUser) error {
			return repo.Execute(
				ctx,
				cmd.StreamID(),
				"User",
				func(_ UserState, v event.Version) ([]event.Event, error) {
					return event.NewEvents(cmd.StreamID(), "User", v,
						[]event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
				},
			)
		})

	basic, _ := command.New("user.create", aggID)
	_ = cmds.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"})

	state, _, _ := repo.Load(ctx, aggID, "User")
	fmt.Printf("User: %s\n", state.Name) // User: Alice
}
