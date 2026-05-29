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
	eventUserDeleted     event.Type = "UserDeleted"
	eventUserReborn      event.Type = "UserReborn"

	cmdCreateUser     command.Type = "CreateUser"
	cmdChangeUserName command.Type = "ChangeUserName"
	cmdDeleteUser     command.Type = "DeleteUser"
	cmdRebirthUser    command.Type = "RebirthUser"

	queryGetUser   query.Type = "GetUser"
	queryListUsers query.Type = "ListUsers"
)
