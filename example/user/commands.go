package main

import (
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type CreateUserCmd struct {
	aggregateID id.AggregateID
	email       string
	name        string
	idempotency string
}

func (c *CreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *CreateUserCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *CreateUserCmd) IdempotencyKey() string      { return c.idempotency }

type ChangeUserNameCmd struct {
	aggregateID id.AggregateID
	name        string
	idempotency string
}

func (c *ChangeUserNameCmd) Type() command.Type          { return "ChangeUserName" }
func (c *ChangeUserNameCmd) AggregateID() id.AggregateID { return c.aggregateID }
func (c *ChangeUserNameCmd) IdempotencyKey() string      { return c.idempotency }
