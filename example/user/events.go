package main

import "github.com/larsartmann/go-cqrs-lite/event"

const (
	AggregateType event.AggregateType = "user"

	EventUserCreated      event.Type = "user.created"
	EventUserEmailChanged event.Type = "user.email_changed"
)

type UserCreatedPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserEmailChangedPayload struct {
	OldEmail string `json:"old_email"`
	NewEmail string `json:"new_email"`
}
