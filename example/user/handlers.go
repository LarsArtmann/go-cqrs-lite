package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/decider"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

var errUnexpectedQueryType = errors.New("unexpected query type")

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
		func(_ context.Context, q query.Query) (ReadModel, error) {
			getUserQuery, ok := q.(*GetUserQuery)
			if !ok {
				return ReadModel{}, fmt.Errorf(
					"unexpected query type %T: %w",
					q,
					errUnexpectedQueryType,
				)
			}

			readModelResult, ok := readModel.Get(getUserQuery.aggregateID)
			if !ok {
				return readModelResult, fmt.Errorf(
					"user %s: %w",
					getUserQuery.aggregateID,
					event.ErrAggregateNotFound,
				)
			}

			return readModelResult, nil
		},
	)

	_ = query.RegisterTyped(
		dispatcher, queryListUsers,
		func(_ context.Context, _ query.Query) ([]ReadModel, error) {
			return readModel.List(), nil
		},
	)
}

func registerBusHandlers(bus event.Bus, readModel *ReadModelStore, published *[]event.Event) {
	_ = bus.SubscribeAll(func(_ context.Context, evt event.Event) error {
		*published = append(*published, evt)
		_ = readModel.Handle( //nolint:contextcheck // no parent context in bus handler
			context.Background(),
			evt,
		)

		return nil
	})
}

func newUserCmd(aggID id.AggregateID, email, name string) *CreateUserCmd {
	return &CreateUserCmd{
		aggregateID: aggID,
		email:       email,
		name:        name,
	}
}
