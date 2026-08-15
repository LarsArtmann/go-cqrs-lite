package middleware

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// commandMetadataProvider is the optional interface a command implements to
// expose its metadata. *command.BasicCommand satisfies this.
type commandMetadataProvider interface {
	Metadata() command.Metadata
}

// CommandActorContext is a command middleware that extracts the actor from
// the incoming command's metadata and stores it in the handler context via
// [event.WithActorContext]. Pair with [event.ActorEnricher] on the decider
// repository so events emitted while handling the command record the same
// actor that issued it:
//
//	dispatcher.Use(middleware.CommandActorContext())
//	repo, _ := decider.NewRepository[State](store, bus, d,
//	    decider.WithEnricher(event.ActorEnricher))
//
// Commands without an actor (or custom Command implementations that do not
// expose metadata) pass through unchanged.
func CommandActorContext() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			if mp, ok := cmd.(commandMetadataProvider); ok {
				if actor := mp.Metadata().ActorID; !actor.IsZero() {
					ctx = event.WithActorContext(ctx, actor)
				}
			}

			return next(ctx, cmd)
		}
	}
}
