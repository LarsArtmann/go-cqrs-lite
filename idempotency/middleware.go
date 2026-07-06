package idempotency

import (
	"context"
	"errors"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
)

// KeyExtractor derives the idempotency key from a command. Return an empty
// string to skip the idempotency check for this command (pass-through).
type KeyExtractor func(cmd command.Command) string

// CommandIDKey is the default [KeyExtractor]: it returns the command's minted
// [id.CommandID] as a string. Because [command.New] mints a fresh ID per call,
// this provides dedup within a single command object's lifetime. For
// cross-retry dedup, use [command.WithCommandID] to set a deterministic key,
// or provide a custom [KeyExtractor] that reads a client-supplied header.
func CommandIDKey(cmd command.Command) string {
	return cmd.ID().String()
}

// CommandIdempotency returns a [command.Middleware] that rejects duplicate
// commands using the provided [Store]. On the first occurrence of a key, the
// middleware records it with the given TTL and passes the command to the next
// handler. On subsequent occurrences within the TTL, it returns [ErrDuplicate]
// without calling the next handler.
//
// The key is extracted from each command via keyExtractor. Pass nil to use
// [CommandIDKey] (the command's minted ID).
//
// If the store returns an error (not [ErrDuplicate]), the middleware wraps it
// as a Transient error and returns it — the command is NOT processed, because
// the dedup state is uncertain.
//
// Usage:
//
//	store := idempotency.NewMemoryStore(5 * time.Minute)
//	defer store.Close()
//	cmds.Use(idempotency.CommandIdempotency(store, 10*time.Minute, nil))
func CommandIdempotency(
	store Store,
	ttl time.Duration,
	keyExtractor KeyExtractor,
) command.Middleware {
	if keyExtractor == nil {
		keyExtractor = CommandIDKey
	}

	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			key := keyExtractor(cmd)
			if key == "" {
				return next(ctx, cmd)
			}

			if err := store.CheckAndRecord(ctx, key, ttl); err != nil {
				if isDuplicate(err) {
					return err
				}

				return errorfamily.Wrapf(
					err, errorfamily.Transient,
					"idempotency.store_error",
					"check-and-record failed for command %s", cmd.Type(),
				)
			}

			return next(ctx, cmd)
		}
	}
}

// isDuplicate returns true if err is or wraps [ErrDuplicate].
func isDuplicate(err error) bool {
	return errors.Is(err, ErrDuplicate)
}
