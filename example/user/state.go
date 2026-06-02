package main

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

type UserState struct {
	Email        string
	Name         string
	Deleted      bool
	DeleteReason string
}

func unmarshalAs[T any](evt event.Event) (T, error) {
	return event.DecodePayload[T](evt, codec.JSONCodec{})
}

func foldUser(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case eventUserCreated:
		p, err := unmarshalAs[UserCreatedPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{Email: p.Email, Name: p.Name}, nil
	case eventUserNameChanged:
		p, err := unmarshalAs[UserNameChangedPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{
			Email:        state.Email,
			Name:         p.Name,
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
			DeleteReason: p.Reason,
		}, nil
	case eventUserReborn:
		p, err := unmarshalAs[UserRebornPayload](evt)
		if err != nil {
			return state, err
		}

		return UserState{Email: p.Email, Name: p.Name, Deleted: false}, nil
	default:
		return state, nil
	}
}
