package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func registerCommandHandlers(
	dispatcher *command.Dispatcher,
	deciderRepo *decider.Repository[UserState],
) {
	_ = dispatcher.Register(
		cmdCreateUser,
		func(ctx context.Context, cmd command.Command) error {
			c := cmd.(*CreateUserCmd)
			return deciderRepo.Execute(
				ctx, c.AggregateID(), "User",
				decideCreateUser(c.AggregateID(), c.email, c.name),
			)
		},
	)

	_ = dispatcher.Register(
		cmdChangeUserName,
		func(ctx context.Context, cmd command.Command) error {
			c := cmd.(*ChangeUserNameCmd)
			return deciderRepo.Execute(
				ctx, c.AggregateID(), aggregateType,
				decideChangeName(c.AggregateID(), c.name),
			)
		},
	)
}

func registerQueryHandlers(
	dispatcher *query.Dispatcher,
	readModel *ReadModelStore,
) {
	_ = dispatcher.Register(
		queryGetUser,
		func(_ context.Context, q query.Query) (any, error) {
			gq := q.(*GetUserQuery)
			rm, ok := readModel.Get(gq.aggregateID)
			if !ok {
				return nil, fmt.Errorf("user %s: %w", gq.aggregateID, event.ErrAggregateNotFound)
			}

			return rm, nil
		},
	)

	_ = dispatcher.Register(
		queryListUsers,
		func(_ context.Context, _ query.Query) (any, error) {
			return readModel.List(), nil
		},
	)
}

func registerBusHandlers(bus event.Bus, readModel *ReadModelStore, published *[]event.Event) {
	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		*published = append(*published, evt)
		readModel.Handle(context.Background(), evt)

		return nil
	})
}

func newUserCmd(aggID id.AggregateID, email, name string) *CreateUserCmd {
	return &CreateUserCmd{
		aggregateID: aggID,
		email:       email,
		name:        name,
		idempotency: aggID.String(),
	}
}
