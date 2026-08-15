package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type ctxKeyActor struct{}

// WithActorContext stores the acting [id.ActorID] in the context so events
// created during this execution automatically record who initiated them.
// This is the actor counterpart to [WithCommandCausality], which records
// what command caused the events.
//
// Usage in a command handler:
//
//	ctx = event.WithActorContext(ctx, actor)
//	// pass ctx to decider.Execute with decider.WithEnricher(event.ActorEnricher)
func WithActorContext(ctx context.Context, actor id.ActorID) context.Context {
	return context.WithValue(ctx, ctxKeyActor{}, actor)
}

// ActorFromContext returns the [id.ActorID] stored in the context, if any.
// The second return is false when no actor was set or the actor is zero.
func ActorFromContext(ctx context.Context) (id.ActorID, bool) {
	v, ok := ctx.Value(ctxKeyActor{}).(id.ActorID)
	if !ok || v.IsZero() {
		return id.ActorID{}, false
	}

	return v, true
}

// ActorEnricher is a [ContextEnricher] that propagates the actor from the
// context into event metadata. Use with decider's WithEnricher option so
// every event records who initiated it — the audit-trail counterpart to
// [CommandCausalityEnricher], which records which command produced the event.
//
//	repo, _ := decider.NewRepository[State](store, bus, d,
//	    decider.WithEnricher(event.ActorEnricher))
func ActorEnricher(ctx context.Context) []Option {
	actor, ok := ActorFromContext(ctx)
	if !ok {
		return nil
	}

	return []Option{WithActor(actor)}
}
