package main

import (
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

type GetUserQuery struct {
	aggregateID id.AggregateID
}

func (q *GetUserQuery) Type() query.Type { return queryGetUser }

type ListUsersQuery struct{}

func (q *ListUsersQuery) Type() query.Type { return queryListUsers }
