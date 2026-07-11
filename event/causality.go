package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Causation is the typed representation of command causation on an event
// (ADR-0031). It records which command produced this event, replacing the
// stringly-typed Custom[MetadataKeyCommandType]/Custom[MetadataKeyCommandID]
// pattern while keeping those entries for v2 backward compatibility.
type Causation struct {
	CommandType string
	CommandID   id.CommandID
}

type ctxKeyCausality struct{}

type causalityCtx struct {
	commandType string
	commandID   id.CommandID
}

// WithCommandCausality stores command type and ID in the context so that
// events created during this command's execution automatically trace back
// to the command that caused them.
//
// Usage in a command handler:
//
//	ctx = event.WithCommandCausality(ctx, "create_user", cmdID)
//	// pass ctx to decider.Execute — events will carry command metadata
func WithCommandCausality(
	ctx context.Context,
	commandType string,
	commandID id.CommandID,
) context.Context {
	return context.WithValue(ctx, ctxKeyCausality{}, causalityCtx{
		commandType: commandType,
		commandID:   commandID,
	})
}

// CommandCausalityFromContext returns the command type and ID stored in the
// context, if any. Returns zero values if no command causality was set.
func CommandCausalityFromContext(
	ctx context.Context,
) (commandType string, commandID id.CommandID, ok bool) {
	v, exists := ctx.Value(ctxKeyCausality{}).(causalityCtx)
	if !exists {
		return "", id.CommandID{}, false
	}

	return v.commandType, v.commandID, true
}

// CommandCausalityEnricher creates a ContextEnricher that propagates command
// type and ID from the context into event metadata. Use with decider's
// WithEnricher option to automatically trace every event back to the command
// that produced it.
//
//	repo, _ := decider.NewRepository[State](store, bus, d,
//	    decider.WithEnricher(event.CommandCausalityEnricher))
func CommandCausalityEnricher(ctx context.Context) []Option {
	cmdType, cmdID, ok := CommandCausalityFromContext(ctx)
	if !ok {
		return nil
	}

	return []Option{
		WithCausation(cmdType, cmdID),
		WithCustom(MetadataKeyCommandType, cmdType),
		WithCustom(MetadataKeyCommandID, cmdID.String()),
	}
}
