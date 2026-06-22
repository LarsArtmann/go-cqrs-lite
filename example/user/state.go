package main

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type UserState struct {
	Email        Email
	Name         DisplayName
	Deleted      bool
	DeleteReason Reason
}

func unmarshalAs[T any](evt event.Event) (T, error) {
	return event.DecodePayload[T](evt, codec.JSONCodec{})
}

func applyUser(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case eventUserCreated:
		p, err := unmarshalAs[UserCreatedPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{Email: Email(p.Email), Name: DisplayName(p.Name)}, nil
	case eventUserNameChanged:
		p, err := unmarshalAs[UserNameChangedPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{
			Email:        state.Email,
			Name:         DisplayName(p.Name),
			Deleted:      state.Deleted,
			DeleteReason: state.DeleteReason,
		}, nil
	case eventUserDeleted:
		p, err := unmarshalAs[UserDeletedPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{
			Email:        state.Email,
			Name:         state.Name,
			Deleted:      true,
			DeleteReason: Reason(p.Reason),
		}, nil
	case eventUserReborn:
		p, err := unmarshalAs[UserRebornPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{Email: Email(p.Email), Name: DisplayName(p.Name), Deleted: false}, nil
	default:
		return state, nil
	}
}
