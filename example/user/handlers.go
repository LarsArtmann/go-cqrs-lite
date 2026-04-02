package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

type Repository struct {
	store *event.MemoryStore
	bus   *event.MemoryBus
}

func NewRepository(store *event.MemoryStore, bus *event.MemoryBus) *Repository {
	return &Repository{store: store, bus: bus}
}

func (r *Repository) Save(ctx context.Context, user *User) error {
	changes := user.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID := id.MustParseAggregateID(user.ID())
	if err := r.store.Save(ctx, user.Type(), aggregateID, changes, event.Version(user.Version()-len(changes))); err != nil {
		return fmt.Errorf("save events: %w", err)
	}

	if err := r.bus.Publish(ctx, changes...); err != nil {
		return fmt.Errorf("publish events: %w", err)
	}

	user.MarkChangesAsCommitted()
	return nil
}

func (r *Repository) Load(ctx context.Context, userID id.AggregateID) (*User, error) {
	user := NewUser(userID)

	events, err := r.store.Load(ctx, user.Type(), userID)
	if err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	for _, evt := range events {
		if err := user.Apply(evt); err != nil {
			return nil, fmt.Errorf("apply event %s: %w", evt.Type(), err)
		}
	}

	if err := user.LoadFromHistory(events); err != nil {
		return nil, fmt.Errorf("load from history: %w", err)
	}

	return user, nil
}

func RegisterHandlers(dispatcher *command.Dispatcher, repo *Repository) {
	dispatcher.Register(CommandCreateUser, func(ctx context.Context, cmd command.Command) error {
		createCmd := cmd.(*CreateUser)
		user := NewUser(id.MustParseAggregateID(cmd.AggregateID()))
		if err := user.Create(ctx, createCmd.Name, createCmd.Email); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		return repo.Save(ctx, user)
	})

	dispatcher.Register(CommandChangeUserEmail, func(ctx context.Context, cmd command.Command) error {
		changeCmd := cmd.(*ChangeUserEmail)
		user, err := repo.Load(ctx, id.MustParseAggregateID(cmd.AggregateID()))
		if err != nil {
			return fmt.Errorf("load user: %w", err)
		}
		if err := user.ChangeEmail(ctx, changeCmd.NewEmail); err != nil {
			return fmt.Errorf("change email: %w", err)
		}
		return repo.Save(ctx, user)
	})
}
