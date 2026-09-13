package deriver

import (
	"context"
	"fmt"
	"strconv"

	cqrscommand "github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsid "github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilDispatcher is returned when AsHandler is called with a nil dispatcher.
var ErrNilDispatcher = errorfamily.NewRejection(
	"deriver.nil_dispatcher",
	"deriver: dispatcher must not be nil",
)

// ErrDepthExceeded is returned when [Deriver.AsHandler] is configured with
// [WithMaxDepth] and a synchronous derivation chain exceeds the bound.
var ErrDepthExceeded = errorfamily.NewOrchestration(
	"deriver.depth_exceeded",
	"deriver: synchronous derivation depth exceeded",
)

// DefaultMaxDepth is the depth bound used by [WithMaxDepth] when called
// without an explicit limit.
const DefaultMaxDepth = 16

type depthCtxKey struct{}

// HandlerOption configures the handler produced by [Deriver.AsHandler].
type HandlerOption func(*asHandlerConfig)

type asHandlerConfig struct {
	maxDepth int
}

// WithMaxDepth bounds synchronous derivation chains. Derived dispatches run
// with a depth counter in their context; when a derived command's handler
// synchronously emits an event that re-enters this handler, the counter
// increments, and exceeding maxDepth fails with [ErrDepthExceeded] instead of
// looping forever. Use [DefaultMaxDepth] for the standard bound.
//
// This guards the in-process path only (context flows through synchronous
// dispatch). Fully asynchronous loops (events re-published through a broker)
// are NOT caught here — break those cycles in the domain, or key idempotency
// on something that does not change every round.
func WithMaxDepth(depth int) HandlerOption {
	return func(c *asHandlerConfig) {
		if depth <= 0 {
			depth = DefaultMaxDepth
		}

		c.maxDepth = depth
	}
}

// Deriver transforms an event into zero or more derived commands.
//
// Contract:
//   - Deterministic: the same event always produces the same commands.
//   - Pure: no side effects beyond returning commands (no I/O, no state mutation).
//   - Non-terminating errors propagate; partial results are NOT dispatched.
//
// A Deriver that returns (nil, nil) produces no commands — this is valid.
type Deriver func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error)

// Then returns a Deriver that runs d, then runs next, and concatenates their
// outputs. Both Derivers see the same input event. If d errors, next is not
// called and the error propagates.
//
// This is fan-out composition: one event triggers multiple independent
// derivations. For sequencing (deriver A's commands produce events that
// deriver B processes), wire both into the bus — the event loop handles it.
func (d Deriver) Then(next Deriver) Deriver {
	return func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmds, err := d(ctx, evt)
		if err != nil {
			return nil, err
		}

		more, err := next(ctx, evt)
		if err != nil {
			return nil, err
		}

		return append(cmds, more...), nil
	}
}

// Filter returns a Deriver that only processes events of the given types.
// Events of other types produce no commands (nil, nil).
func (d Deriver) Filter(types ...cqrsevent.Type) Deriver {
	allowed := make(map[cqrsevent.Type]struct{}, len(types))

	for _, t := range types {
		allowed[t] = struct{}{}
	}

	return func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
		if _, ok := allowed[evt.Type()]; !ok {
			return nil, nil
		}

		return d(ctx, evt)
	}
}

// SourceEventIDKey is the custom metadata key stamped on derived commands
// recording the ID of the source event. Use this to trace a command back to
// the event that produced it.
const SourceEventIDKey cqrscommand.MetadataKey = "deriver.source_event_id"

// AsHandler converts a Deriver into an [cqrsevent.Handler] that dispatches
// derived commands via the given command.Dispatcher. Each command is dispatched
// sequentially; the first dispatch error stops processing and propagates.
//
// Derivation cycles (A's commands produce events that re-trigger A) do not
// self-terminate: deterministic command IDs key on the source event ID, which
// changes every round. Pass [WithMaxDepth] to bound synchronous chains.
//
// For at-least-once delivery safety, chain [Deriver.Idempotent] before
// AsHandler so that re-processing the same event yields the same command IDs:
//
//	myDeriver.Idempotent().AsHandler(dispatcher, deriver.WithMaxDepth(deriver.DefaultMaxDepth))
func (d Deriver) AsHandler(
	dispatcher *cqrscommand.Dispatcher,
	opts ...HandlerOption,
) cqrsevent.Handler {
	if dispatcher == nil {
		return func(_ context.Context, _ cqrsevent.Event) error {
			return ErrNilDispatcher
		}
	}

	var cfg asHandlerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(ctx context.Context, evt cqrsevent.Event) error {
		depth := 0
		if v, ok := ctx.Value(depthCtxKey{}).(int); ok {
			depth = v
		}

		if cfg.maxDepth > 0 && depth >= cfg.maxDepth {
			return fmt.Errorf("deriver depth %d exceeds %d: %w",
				depth, cfg.maxDepth, ErrDepthExceeded)
		}

		cmds, err := d(ctx, evt)
		if err != nil {
			return fmt.Errorf("deriver for %s: %w", evt.Type(), err)
		}

		dispatchCtx := ctx
		if cfg.maxDepth > 0 {
			dispatchCtx = context.WithValue(ctx, depthCtxKey{}, depth+1)
		}

		for _, cmd := range cmds {
			if err := dispatcher.Dispatch(dispatchCtx, cmd); err != nil {
				return fmt.Errorf("deriver dispatch %s (from %s): %w",
					cmd.Type(), evt.Type(), err)
			}
		}

		return nil
	}
}

// Idempotent returns a Deriver that re-stamps each derived command with a
// deterministic [id.CommandID] derived from the source event's ID and the
// command's position in the output slice. Same event re-processed → same
// command IDs, so an idempotency store keyed on the command ID (see
// middleware.CommandIdempotency with a nil keyExtractor) deduplicates
// at-least-once redeliveries.
//
// The source event's ID is also stamped as custom metadata
// (SourceEventIDKey) on each command for traceability.
//
// Commands must be *command.BasicCommand (produced by command.New) for the
// re-stamp to take effect. Non-BasicCommand implementations are passed through
// unchanged — the ID and metadata remain as the Deriver author set them.
//
// This is a no-op if the Deriver returns no commands.
func (d Deriver) Idempotent() Deriver {
	return func(ctx context.Context, evt cqrsevent.Event) ([]cqrscommand.Command, error) {
		cmds, err := d(ctx, evt)
		if err != nil {
			return nil, err
		}

		for i := range cmds {
			basicCmd, ok := cmds[i].(*cqrscommand.BasicCommand)
			if !ok {
				continue
			}

			derivedID := cqrsid.DeriveCommandID("deriver", evt.ID().String(), strconv.Itoa(i))
			cqrscommand.WithCommandID(derivedID)(basicCmd)
			cqrscommand.WithCustomMetadata(string(SourceEventIDKey), evt.ID().String())(basicCmd)
		}

		return cmds, nil
	}
}

// Noop is a terminal Deriver that produces no commands. Useful as a placeholder
// or default in optional composition chains.
func Noop() Deriver {
	return func(_ context.Context, _ cqrsevent.Event) ([]cqrscommand.Command, error) {
		return nil, nil
	}
}
