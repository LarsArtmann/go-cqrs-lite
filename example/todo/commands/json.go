package commands

type CommandType string

const (
	CommandTypeCreate       CommandType = "create"
	CommandTypeUpdate       CommandType = "update"
	CommandTypeDelete       CommandType = "delete"
	CommandTypeChangeStatus CommandType = "change_status"
)
