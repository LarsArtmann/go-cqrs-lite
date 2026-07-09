package middleware

import (
	"context"
	"errors"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// NewIdempotency returns a generic middleware that rejects duplicate messages
// using the provided idempotency.Store. On the first occurrence of a key, it
// records the key with the given TTL and passes the message to the next
// handler. On subsequent occurrences within the TTL, it returns
// idempotency.ErrDuplicate without calling the next handler.
//
// An empty key ("") skips dedup for that message (pass-through).
func NewIdempotency[M any](
	adapter MessageAdapter[M],
	store idempotency.Store,
	ttl time.Duration,
	keyExtractor func(M) string,
) Middleware[M] {
	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) error {
			key := keyExtractor(msg)
			if key == "" {
				return next(ctx, msg)
			}

			if err := store.CheckAndRecord(ctx, key, ttl); err != nil {
				if errors.Is(err, idempotency.ErrDuplicate) {
					return err //nolint:wrapcheck // sentinel error
				}

				return errorfamily.Wrapf(
					err, errorfamily.Transient,
					"middleware."+adapter.Kind+"_idempotency",
					"check-and-record failed for %s %s",
					adapter.Kind, adapter.ExtractType(msg),
				)
			}

			return next(ctx, msg)
		}
	}
}

// CommandIdempotency wires the Store into a command.Dispatcher middleware
// chain. Pass nil for keyExtractor to use the command's minted ID
// (cmd.ID().String()).
func CommandIdempotency(
	store idempotency.Store,
	ttl time.Duration,
	keyExtractor func(command.Command) string,
) command.Middleware {
	if keyExtractor == nil {
		keyExtractor = func(cmd command.Command) string { return cmd.ID().String() }
	}

	return AsCommand(NewIdempotency(CommandAdapter, store, ttl, keyExtractor))
}

// EventIdempotency wires the Store into an event handler middleware chain.
// Pass nil for keyExtractor to use the event's minted ID (evt.ID().String()).
//
// For ordered event consumption (projections), checkpoint-based dedup
// (projectionhost) is structurally stronger than key-based dedup. Use this
// middleware when you don't own the checkpoint (webhooks, external sinks,
// cross-system delivery) or as defense-in-depth alongside checkpoints.
func EventIdempotency(
	store idempotency.Store,
	ttl time.Duration,
	keyExtractor func(event.Event) string,
) event.Middleware {
	if keyExtractor == nil {
		keyExtractor = func(evt event.Event) string { return evt.ID().String() }
	}

	return AsEvent(NewIdempotency(EventAdapter, store, ttl, keyExtractor))
}

// QueryIdempotency wires the Store into a query dispatcher middleware chain.
// Queries have no built-in identity, so a non-nil keyExtractor must be provided.
// Return "" from the keyExtractor to skip dedup for a specific query.
//
// Panics if keyExtractor is nil — this is a programming error, not a runtime
// condition. Use CommandIdempotency or EventIdempotency if you want a default
// key strategy.
func QueryIdempotency(
	store idempotency.Store,
	ttl time.Duration,
	keyExtractor func(query.Query) string,
) query.Middleware {
	if keyExtractor == nil {
		panic("middleware.QueryIdempotency: keyExtractor must not be nil " +
			"(queries have no built-in identity; provide a func(query.Query) string)")
	}

	return AsQuery(NewIdempotency(QueryAdapter, store, ttl, keyExtractor))
}
