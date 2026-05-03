package main

import (
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

type GetUserQuery struct {
	aggregateID id.AggregateID
}

func (q *GetUserQuery) Type() query.Type { return "GetUser" }

type ListUsersQuery struct{}

func (q *ListUsersQuery) Type() query.Type { return "ListUsers" }
