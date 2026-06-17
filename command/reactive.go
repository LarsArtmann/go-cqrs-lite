package command

import (
	"context"

	ro "github.com/samber/ro"
)

// CommandBus is a reactive subject for command streams.
// Use NewCommandBus() to create one. Subscribe with ro.Observer, emit with Next.
//
// This mirrors event.EventBus for command-side reactive dispatch. Commands
// published via ro.Subject.Next(cmd) are broadcast to all subscribers.
//
// Example:
//
//	bus := command.NewCommandBus()
//	filtered := ro.Pipe1(bus, command.FilterCommandType("user.create"))
//	filtered.Subscribe(command.HandlerToObserver(myHandler))
//	bus.Next(createCmd)
type CommandBus = ro.Subject[Command]

// NewCommandBus creates a new PublishSubject-backed CommandBus for
// broadcasting commands to multiple subscribers.
func NewCommandBus() ro.Subject[Command] {
	return ro.NewPublishSubject[Command]()
}

// NewReplayCommandBus creates a new ReplaySubject-backed CommandBus that
// replays the last n commands to new subscribers.
func NewReplayCommandBus(n int) ro.Subject[Command] {
	return ro.NewReplaySubject[Command](n)
}

// NewBehaviorCommandBus creates a new BehaviorSubject-backed CommandBus that
// replays the latest command to new subscribers.
func NewBehaviorCommandBus(initial Command) ro.Subject[Command] {
	return ro.NewBehaviorSubject(initial)
}

// FilterCommandType returns an operator that filters an Observable[Command]
// to only commands of the given type.
func FilterCommandType(cmdType Type) func(ro.Observable[Command]) ro.Observable[Command] {
	return ro.Filter(func(c Command) bool {
		return c.Type() == cmdType
	})
}

// FilterCommandTypes returns an operator that filters an Observable[Command]
// to only commands of the given types.
func FilterCommandTypes(cmdTypes ...Type) func(ro.Observable[Command]) ro.Observable[Command] {
	types := newCmdTypeSet(cmdTypes)

	return ro.Filter(func(c Command) bool {
		return types.has(c.Type())
	})
}

// HandlerToObserver converts a command Handler into a ro.Observer[Command].
// The handler receives the context from the stream (via NextWithContext/SubscribeWithContext).
// If the handler returns an error, the error is forwarded through the observer's error channel.
func HandlerToObserver(handler Handler) ro.Observer[Command] {
	var obs ro.Observer[Command]

	obs = ro.NewObserverWithContext(
		func(ctx context.Context, c Command) {
			err := handler(ctx, c)
			if err != nil {
				obs.ErrorWithContext(ctx, err)
			}
		},
		func(_ context.Context, _ error) {},
		func(_ context.Context) {},
	)

	return obs
}

// Observable is a named type for command observables, improving discoverability
// over the raw ro.Observable[Command].
type Observable = ro.Observable[Command]

type cmdTypeSet map[Type]struct{}

func newCmdTypeSet(types []Type) cmdTypeSet {
	if len(types) == 0 {
		return nil
	}

	s := make(cmdTypeSet, len(types))
	for _, t := range types {
		s[t] = struct{}{}
	}

	return s
}

func (s cmdTypeSet) has(t Type) bool {
	if s == nil {
		return true
	}

	_, ok := s[t]

	return ok
}
