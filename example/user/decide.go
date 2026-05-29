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
			eventUserCreated, aggID, aggregateType, version.Increment(),
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

		if state.Deleted {
			return nil, event.NewRejection("user.change_name.deleted",
				"cannot change name of deleted user")
		}

		if name == "" {
			return nil, event.NewRejection("user.change_name.name_required",
				"name is required")
		}

		evt, err := event.NewEvent(
			eventUserNameChanged, aggID, aggregateType, version.Increment(),
			mustMarshal(UserNameChangedPayload{Name: name}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserNameChanged event: %w", err)
		}

		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if state.Email == "" {
			return nil, event.NewRejection("user.delete.not_found",
				"user does not exist")
		}

		if state.Deleted {
			return nil, event.NewRejection("user.delete.already_deleted",
				"user is already deleted")
		}

		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateType, version.Increment(),
			mustMarshal(UserDeletedPayload{Reason: reason}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserDeleted event: %w", err)
		}

		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, fmt.Errorf("mark tombstone: %w", markErr)
		}

		return []event.Event{marked}, nil
	}
}

func decideRebirthUser(
	aggID id.AggregateID,
	email, name string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if !state.Deleted {
			return nil, event.NewRejection("user.rebirth.not_deleted",
				"user must be deleted before rebirth")
		}

		if email == "" {
			return nil, event.NewRejection("user.rebirth.email_required",
				"email is required for rebirth")
		}

		evt, err := event.NewEvent(
			eventUserReborn, aggID, aggregateType, version.Increment(),
			mustMarshal(UserRebornPayload{Email: email, Name: name}),
		)
		if err != nil {
			return nil, fmt.Errorf("create UserReborn event: %w", err)
		}

		return []event.Event{evt}, nil
	}
}
