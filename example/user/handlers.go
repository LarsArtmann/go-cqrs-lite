package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func registerCommandHandlers(
	dispatcher *command.Dispatcher,
	deciderRepo *decider.Repository[UserState],
) {
	_ = command.RegisterTyped(
		dispatcher, cmdCreateUser,
		func(ctx context.Context, c *CreateUserCmd) error {
			return deciderRepo.Execute(
				ctx, c.AggregateID(), aggregateType,
				decideCreateUser(c.AggregateID(), c.email, c.name),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdChangeUserName,
		func(ctx context.Context, c *ChangeUserNameCmd) error {
			return deciderRepo.Execute(
				ctx, c.AggregateID(), aggregateType,
				decideChangeName(c.AggregateID(), c.name),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdDeleteUser,
		func(ctx context.Context, c *DeleteUserCmd) error {
			return deciderRepo.Execute(
				ctx, c.AggregateID(), aggregateType,
				decideDeleteUser(c.AggregateID(), c.reason),
			)
		},
	)

	_ = command.RegisterTyped(
		dispatcher, cmdRebirthUser,
		func(ctx context.Context, c *RebirthUserCmd) error {
			return deciderRepo.Execute(
				ctx, c.AggregateID(), aggregateType,
				decideRebirthUser(c.AggregateID(), c.email, c.name),
			)
		},
	)
}

func registerQueryHandlers(
	dispatcher *query.Dispatcher,
	readModel *ReadModelStore,
) {
	_ = query.RegisterTyped(
		dispatcher, queryGetUser,
		func(_ context.Context, q *GetUserQuery) (ReadModel, error) {
			readModelResult, ok := readModel.Get(q.aggregateID)
			if !ok {
				return readModelResult, event.Newf(
					event.Infrastructure, "user.handlers.1",
					"user %s: %v",
					q.aggregateID,
					event.ErrAggregateNotFound,
				)
			}

			return readModelResult, nil
		},
	)

	_ = query.RegisterTyped(
		dispatcher, queryListUsers,
		func(_ context.Context, _ *ListUsersQuery) ([]ReadModel, error) {
			return readModel.List(), nil
		},
	)
}

func registerProjection(
	bus event.Bus,
	readModel *ReadModelStore,
) func() {
	_ = bus.SubscribeAll(func(ctx context.Context, evt event.Event) error {
		return readModel.Handle(ctx, evt)
	})

	return func() {}
}

func trackPublishedEvents(bus event.Bus, published *[]event.Event) {
	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		*published = append(*published, evt)

		return nil
	})
}

func newUserCmd(aggID id.AggregateID, email, name string) *CreateUserCmd {
	return &CreateUserCmd{
		commandID:   id.NewCommandID(),
		aggregateID: aggID,
		email:       Email(email),
		name:        DisplayName(name),
	}
}
