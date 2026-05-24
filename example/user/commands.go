package main

import (
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

type CreateUserCmd struct {
	aggregateID id.AggregateID
	email       string
	name        string
}

func (c *CreateUserCmd) Type() command.Type          { return cmdCreateUser }
func (c *CreateUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeUserNameCmd struct {
	aggregateID id.AggregateID
	name        string
}

func (c *ChangeUserNameCmd) Type() command.Type          { return cmdChangeUserName }
func (c *ChangeUserNameCmd) AggregateID() id.AggregateID { return c.aggregateID }
