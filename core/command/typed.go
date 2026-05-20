package command

import (
	"context"
	"fmt"
)

// TypedHandler processes a typed command.
// The concrete command type T is extracted from the Command interface,
// eliminating the need for manual type assertions in handlers.
type TypedHandler[T Command] func(ctx context.Context, cmd T) error

// RegisterTyped binds a typed handler to a command type.
// The handler receives the concrete command type T directly,
// providing compile-time type safety without manual type assertions.
func RegisterTyped[T Command](d *Dispatcher, cmdType Type, handler TypedHandler[T]) error {
	return d.Register(cmdType, func(ctx context.Context, cmd Command) error {
		typed, ok := cmd.(T)
		if !ok {
			return fmt.Errorf("command type assertion failed: expected %T, got %T", typed, cmd) //nolint:err113
		}

		return handler(ctx, typed)
	})
}
