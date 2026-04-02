package main

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

const (
	CommandCreateUser      command.Type = "user.create"
	CommandChangeUserEmail command.Type = "user.change_email"
)

type CreateUser struct {
	*command.Core
	Name  string
	Email string
}

func NewCreateUser(aggregateID id.AggregateID, name, email string) *CreateUser {
	return &CreateUser{
		Core:  command.New(CommandCreateUser, aggregateID),
		Name:  name,
		Email: email,
	}
}

type ChangeUserEmail struct {
	*command.Core
	NewEmail string
}

func NewChangeUserEmail(aggregateID id.AggregateID, newEmail string) *ChangeUserEmail {
	return &ChangeUserEmail{
		Core:     command.New(CommandChangeUserEmail, aggregateID),
		NewEmail: newEmail,
	}
}
