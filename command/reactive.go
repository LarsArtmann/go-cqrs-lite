package command

import (
	ro "github.com/samber/ro"
)

// CommandBus is a reactive subject for command streams.
// Use NewCommandBus() to create one. Subscribe with ro.Observer, emit with Next.
type CommandBus = ro.Subject[Command]

// NewCommandBus creates a new PublishSubject-backed CommandBus for broadcasting commands.
func NewCommandBus() ro.Subject[Command] {
	return ro.NewPublishSubject[Command]()
}

// FilterCommandType returns an operator that filters an Observable[Command] to only commands of the given type.
func FilterCommandType(cmdType Type) func(ro.Observable[Command]) ro.Observable[Command] {
	return ro.Filter(func(c Command) bool {
		return c.Type() == cmdType
	})
}

// Observable is a named type for command observables.
type Observable = ro.Observable[Command]
