package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type User struct {
	*aggregate.Core
	name  string
	email string
}

var _ aggregate.Root = (*User)(nil)

func NewUser(userID id.AggregateID) *User {
	return &User{
		Core: aggregate.NewCore(userID, AggregateType),
	}
}

func (u *User) Name() string  { return u.name }
func (u *User) Email() string { return u.email }

func (u *User) Apply(evt event.Event) error {
	switch evt.Type() {
	case EventUserCreated:
		var p UserCreatedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal %s: %w", EventUserCreated, err)
		}
		u.name = p.Name
		u.email = p.Email
	case EventUserEmailChanged:
		var p UserEmailChangedPayload
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return fmt.Errorf("unmarshal %s: %w", EventUserEmailChanged, err)
		}
		u.email = p.NewEmail
	}
	return nil
}

func (u *User) Create(ctx context.Context, name, email string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}

	catEvt, err := NewUserCreated(
		id.MustParseAggregateID(u.ID()),
		UserCreatedPayload{Name: name, Email: email},
	)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	u.name = name
	u.email = email
	u.RecordEvent(ctx, catEvt)
	return nil
}

func (u *User) ChangeEmail(ctx context.Context, newEmail string) error {
	if newEmail == "" {
		return fmt.Errorf("email is required")
	}
	if newEmail == u.email {
		return fmt.Errorf("new email must differ from current email")
	}

	catEvt, err := NewUserEmailChanged(id.MustParseAggregateID(u.ID()), UserEmailChangedPayload{
		OldEmail: u.email,
		NewEmail: newEmail,
	})
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	u.email = newEmail
	u.RecordEvent(ctx, catEvt)
	return nil
}
