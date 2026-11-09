package decider_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

type UserState struct {
	Name  string
	Email string
}

func ExampleRepository_Execute() {
	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[UserState]{
		Initial: UserState{},
		Apply: func(state UserState, evt event.Event) (UserState, error) {
			switch evt.Type() {
			case "UserCreated":
				state.Email = string(evt.Payload())
			case "UserNameChanged":
				state.Name = string(evt.Payload())
			}

			return state, nil
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	streamID := id.NewStreamID()

	err = repo.Execute(
		context.Background(),
		streamID,
		"User",
		func(state UserState, version event.Version) ([]event.Event, error) {
			evt, evtErr := event.NewEvent(
				"UserCreated",
				streamID,
				"User",
				version+1,
				[]byte("alice@example.com"),
			)

			return []event.Event{evt}, evtErr
		},
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println("executed: ok")

	// Output:
	// executed: ok
}

// renameUserCommand is a consumer command: it satisfies decider.CausedCommand
// structurally through its ID method (and the optional Type capability that
// enriches the causation stamp with the command type).
type renameUserCommand struct {
	id      id.CommandID
	cmdType record.Type
}

func (c *renameUserCommand) ID() id.CommandID  { return c.id }
func (c *renameUserCommand) Type() record.Type { return c.cmdType }

func ExampleExecuteCommandRef() {
	store := memory.NewMemoryStore()
	bus := eventtest.NewFakeBus()

	d := decider.Decider[UserState]{
		Initial: UserState{},
		Apply: func(state UserState, evt event.Event) (UserState, error) {
			if evt.Type() == "UserRenamed" {
				state.Name = string(evt.Payload())
			}

			return state, nil
		},
	}

	repo, err := decider.NewRepository(store, bus, d)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	ref := id.NewStreamRef("User", id.NewStreamID())
	cmd := &renameUserCommand{id: id.NewCommandID(), cmdType: "RenameUser"}

	err = decider.ExecuteCommandRef(
		context.Background(),
		repo,
		ref,
		cmd,
		func(state UserState, version event.Version, cmd *renameUserCommand) ([]event.Event, error) {
			evt, evtErr := event.NewEvent(
				"UserRenamed",
				ref.ID,
				"User",
				version+1,
				[]byte("grace"),
			)

			return []event.Event{evt}, evtErr
		},
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	evts, err := store.Load(context.Background(), ref)
	if err != nil || len(evts) == 0 {
		fmt.Println("error: no events")

		return
	}

	if cause := evts[0].Metadata().Causation; cause != nil {
		fmt.Println("caused by:", cause.CommandType, cause.CommandID != cmd.ID())
	}

	// Output:
	// caused by: RenameUser false
}
