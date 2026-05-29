package command

import (
	ro "github.com/samber/ro"
)

type CommandBus = ro.Subject[Command]

func NewCommandBus() ro.Subject[Command] {
	return ro.NewPublishSubject[Command]()
}

func FilterCommandType(cmdType Type) func(ro.Observable[Command]) ro.Observable[Command] {
	return ro.Filter[Command](func(c Command) bool {
		return c.Type() == cmdType
	})
}
