package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func decideCreateUser(
	aggID id.AggregateID,
	email Email, name DisplayName,
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

		payload, mErr := marshalPayload(
			UserCreatedPayload{Email: string(email), Name: string(name)},
		)
		if mErr != nil {
			return nil, event.Newf(event.Infrastructure, "user.decide.payload", "%v", mErr)
		}

		evt, err := event.NewEvent(
			eventUserCreated, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"user.decide.1",
				"create UserCreated event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

func decideChangeName(
	aggID id.AggregateID,
	name DisplayName,
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

		payload, mErr := marshalPayload(UserNameChangedPayload{Name: string(name)})
		if mErr != nil {
			return nil, event.Newf(event.Infrastructure, "user.decide.payload", "%v", mErr)
		}

		evt, err := event.NewEvent(
			eventUserNameChanged, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"user.decide.2",
				"create UserNameChanged event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}

func decideDeleteUser(
	aggID id.AggregateID,
	reason Reason,
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

		payload, mErr := marshalPayload(UserDeletedPayload{Reason: string(reason)})
		if mErr != nil {
			return nil, event.Newf(event.Infrastructure, "user.decide.payload", "%v", mErr)
		}

		evt, err := event.NewEvent(
			eventUserDeleted, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"user.decide.3",
				"create UserDeleted event: %v",
				err,
			)
		}

		marked, markErr := event.MarkTombstone(evt)
		if markErr != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"user.decide.4",
				"mark tombstone: %v",
				markErr,
			)
		}

		return []event.Event{marked}, nil
	}
}

func decideRebirthUser(
	aggID id.AggregateID,
	email Email, name DisplayName,
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

		payload, mErr := marshalPayload(UserRebornPayload{Email: string(email), Name: string(name)})
		if mErr != nil {
			return nil, event.Newf(event.Infrastructure, "user.decide.payload", "%v", mErr)
		}

		evt, err := event.NewEvent(
			eventUserReborn, aggID, aggregateType, version.Increment(),
			payload,
		)
		if err != nil {
			return nil, event.Newf(
				event.Infrastructure,
				"user.decide.5",
				"create UserReborn event: %v",
				err,
			)
		}

		return []event.Event{evt}, nil
	}
}
