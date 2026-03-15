package command

import "context"

// Handler processes a command and returns any error
type Handler func(ctx context.Context, cmd Command) error

// Middleware wraps command handlers for cross-cutting concerns
type Middleware func(Handler) Handler
