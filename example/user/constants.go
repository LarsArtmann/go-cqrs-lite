package main

import (
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

const (
	aggregateType event.AggregateType = "User"

	eventUserCreated     event.Type = "UserCreated"
	eventUserNameChanged event.Type = "UserNameChanged"

	cmdCreateUser     command.Type = "CreateUser"
	cmdChangeUserName command.Type = "ChangeUserName"

	queryGetUser  query.Type = "GetUser"
	queryListUsers query.Type = "ListUsers"
)
