package main

import (
	"encoding/json"
	"fmt"
)

type UserCreatedPayload struct {
	Email string `description:"The user's email address" json:"email"`
	Name  string `description:"The user's display name"  json:"name"`
}

type UserNameChangedPayload struct {
	Name string `description:"The new display name" json:"name"`
}

type UserDeletedPayload struct {
	Reason string `description:"Reason for deletion" json:"reason"`
}

type UserRebornPayload struct {
	Email string `description:"New email address" json:"email"`
	Name  string `description:"New display name"  json:"name"`
}

func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return b, nil
}

type CreateUserPayload struct {
	Email string `description:"The user's email address" json:"email"`
	Name  string `description:"The user's display name"  json:"name"`
}

type ChangeUserNamePayload struct {
	Name string `description:"The new display name" json:"name"`
}
