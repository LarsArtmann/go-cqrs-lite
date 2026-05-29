package main

import (
	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/query"
)

type GetUserQuery struct {
	aggregateID id.AggregateID
}

func (q *GetUserQuery) Type() query.Type { return queryGetUser }

type ListUsersQuery struct{}

func (q *ListUsersQuery) Type() query.Type { return queryListUsers }
