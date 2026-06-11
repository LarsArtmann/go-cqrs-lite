package main

import (
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

type CreateUserCmd struct {
	aggregateID id.AggregateID
	email       Email
	name        DisplayName
}

func (c *CreateUserCmd) Type() command.Type          { return cmdCreateUser }
func (c *CreateUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type ChangeUserNameCmd struct {
	aggregateID id.AggregateID
	name        DisplayName
}

func (c *ChangeUserNameCmd) Type() command.Type          { return cmdChangeUserName }
func (c *ChangeUserNameCmd) AggregateID() id.AggregateID { return c.aggregateID }

type DeleteUserCmd struct {
	aggregateID id.AggregateID
	reason      Reason
}

func (c *DeleteUserCmd) Type() command.Type          { return cmdDeleteUser }
func (c *DeleteUserCmd) AggregateID() id.AggregateID { return c.aggregateID }

type RebirthUserCmd struct {
	aggregateID id.AggregateID
	email       Email
	name        DisplayName
}

func (c *RebirthUserCmd) Type() command.Type          { return cmdRebirthUser }
func (c *RebirthUserCmd) AggregateID() id.AggregateID { return c.aggregateID }
