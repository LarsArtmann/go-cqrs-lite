package main

import "encoding/json"

type UserCreatedPayload struct {
	Email string `description:"The user's email address" json:"email"`
	Name  string `description:"The user's display name"  json:"name"`
}

type UserNameChangedPayload struct {
	Name string `description:"The new display name" json:"name"`
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return b
}
