package main

import (
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

type UserState struct {
	Email string
	Name  string
}

func unmarshalAs[T any](evt event.Event) (T, error) {
	var payload T
	if err := json.Unmarshal(evt.Payload(), &payload); err != nil {
		return payload, fmt.Errorf("unmarshal %s: %w", evt.Type(), err)
	}

	return payload, nil
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

		return UserState{Email: state.Email, Name: p.Name}, nil
	default:
		return state, nil
	}
}
