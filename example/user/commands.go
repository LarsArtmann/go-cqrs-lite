package main

import (
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

const (
	CommandCreateUser      command.Type = "user.create"
	CommandChangeUserEmail command.Type = "user.change_email"
)

type CreateUser struct {
	*command.CatalogCore

	Name  string `json:"name"  doc:"Full name of the user"`
	Email string `json:"email" doc:"Email address of the user"`
}

func NewCreateUser(aggregateID id.AggregateID, name, email string) *CreateUser {
	return &CreateUser{
		CatalogCore: command.NewCatalogCore(
			CommandCreateUser,
			aggregateID,
			command.CatalogMeta{
				Name:    "CreateUser",
				Version: "1.0.0",
				Summary: "Creates a new user in the system",
			},
		),
		Name:  name,
		Email: email,
	}
}

type ChangeUserEmail struct {
	*command.CatalogCore

	NewEmail string `json:"newEmail" doc:"The new email address"`
}

func NewChangeUserEmail(aggregateID id.AggregateID, newEmail string) *ChangeUserEmail {
	return &ChangeUserEmail{
		CatalogCore: command.NewCatalogCore(
			CommandChangeUserEmail,
			aggregateID,
			command.CatalogMeta{
				Name:    "ChangeUserEmail",
				Version: "1.0.0",
				Summary: "Updates a user's email address",
			},
		),
		NewEmail: newEmail,
	}
}
