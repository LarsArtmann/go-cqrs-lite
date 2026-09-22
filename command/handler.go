package command

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
)

// Handler processes a command and returns any error.
type Handler func(ctx context.Context, cmd Command) error

// Middleware wraps command handlers for cross-cutting concerns. It is an
// alias of the shared [dispatcher.Middleware] shape (E15 unification): one
// function value composes with every dispatcher and bus.
type Middleware = dispatcher.Middleware[Handler]
