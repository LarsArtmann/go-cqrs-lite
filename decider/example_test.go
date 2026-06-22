package decider_test

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
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

	aggID := id.NewAggregateID()

	err = repo.Execute(
		context.Background(),
		aggID,
		"User",
		func(state UserState, version event.Version) ([]event.Event, error) {
			evt, evtErr := event.NewEvent(
				"UserCreated",
				aggID,
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
