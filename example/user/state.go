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

func foldUser(state UserState, evt event.Event) (UserState, error) {
	switch evt.Type() {
	case eventUserCreated:
		var p UserCreatedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return state, fmt.Errorf("unmarshal UserCreated: %w", err)
		}

		return UserState{Email: p.Email, Name: p.Name}, nil
	case eventUserNameChanged:
		var p UserNameChangedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return state, fmt.Errorf("unmarshal UserNameChanged: %w", err)
		}

		return UserState{Email: state.Email, Name: p.Name}, nil
	default:
		return state, nil
	}
}
