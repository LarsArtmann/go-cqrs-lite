package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/command"
)

func RegisterHandlers(dispatcher *command.Dispatcher, repo *aggregate.EventSourcedRepository) {
	dispatcher.Register(CommandCreateUser, func(ctx context.Context, cmd command.Command) error {
		createCmd := cmd.(*CreateUser)
		user := NewUser(cmd.AggregateID())
		if err := user.Create(ctx, createCmd.Name, createCmd.Email); err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		return repo.Save(ctx, user)
	})

	dispatcher.Register(
		CommandChangeUserEmail,
		func(ctx context.Context, cmd command.Command) error {
			changeCmd := cmd.(*ChangeUserEmail)
			user := NewUser(cmd.AggregateID())
			if err := repo.Load(ctx, user); err != nil {
				return fmt.Errorf("load user: %w", err)
			}

			if err := user.ChangeEmail(ctx, changeCmd.NewEmail); err != nil {
				return fmt.Errorf("change email: %w", err)
			}

			return repo.Save(ctx, user)
		},
	)
}
