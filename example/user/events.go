package main

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

const (
	AggregateType event.AggregateType = "user"

	EventUserCreated      event.Type = "user.created"
	EventUserEmailChanged event.Type = "user.email_changed"
)

type UserCreatedPayload struct {
	Name  string `json:"name"  doc:"Full name of the user"`
	Email string `json:"email" doc:"Email address of the user"`
}

type UserEmailChangedPayload struct {
	OldEmail string `json:"old_email" doc:"Previous email address"`
	NewEmail string `json:"new_email" doc:"New email address"`
}

type UserCreated struct {
	*event.EventCatalogCore
}

func NewUserCreated(
	aggregateID id.AggregateID,
	payload UserCreatedPayload,
) (*UserCreated, error) {
	evt, err := event.NewEventCatalogCore(
		EventUserCreated,
		aggregateID,
		AggregateType,
		1,
		marshalPayload(payload),
		event.EventCatalogMeta{
			Name:          "UserCreated",
			Version:       "1.0.0",
			Summary:       "Emitted when a new user is created",
			AggregateType: AggregateType,
		},
	)
	if err != nil {
		return nil, err
	}

	return &UserCreated{EventCatalogCore: evt}, nil
}

type UserEmailChanged struct {
	*event.EventCatalogCore
}

func NewUserEmailChanged(
	aggregateID id.AggregateID,
	payload UserEmailChangedPayload,
) (*UserEmailChanged, error) {
	evt, err := event.NewEventCatalogCore(
		EventUserEmailChanged,
		aggregateID,
		AggregateType,
		1,
		marshalPayload(payload),
		event.EventCatalogMeta{
			Name:          "UserEmailChanged",
			Version:       "1.0.0",
			Summary:       "Emitted when a user's email is changed",
			AggregateType: AggregateType,
		},
	)
	if err != nil {
		return nil, err
	}

	return &UserEmailChanged{EventCatalogCore: evt}, nil
}

func marshalPayload(v any) []byte {
	data, _ := json.Marshal(v) //nolint:errcheck // safe for catalog payloads
	return data
}
