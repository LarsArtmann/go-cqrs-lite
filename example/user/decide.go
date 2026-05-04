package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func decideCreateUser(
	aggID id.AggregateID,
	email, name string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Email != "" {
			return nil, event.NewConflict("user.already_exists",
				"user with this ID already exists")
		}

		if email == "" {
			return nil, event.NewRejection("user.create.email_required",
				"email is required")
		}

		evt, err := event.NewEvent(
			eventUserCreated, aggID, aggregateType, version.Int()+1,
			mustMarshal(UserCreatedPayload{Email: email, Name: name}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserCreated event: %w", err)
		}

		return []event.Event{evt}, nil
	}
}

func decideChangeName(
	aggID id.AggregateID,
	name string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Email == "" {
			return nil, event.NewRejection("user.change_name.not_found",
				"user does not exist")
		}

		if name == "" {
			return nil, event.NewRejection("user.change_name.name_required",
				"name is required")
		}

		evt, err := event.NewEvent(
			eventUserNameChanged, aggID, aggregateType, version.Int()+1,
			mustMarshal(UserNameChangedPayload{Name: name}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserNameChanged event: %w", err)
		}

		return []event.Event{evt}, nil
	}
}
