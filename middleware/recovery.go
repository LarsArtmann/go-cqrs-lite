package middleware

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func panicError(msgKind, typeName string, r any) error {
	return event.Wrapf(
		ErrPanicRecovered, event.Corruption,
		"middleware.panic_detail",
		"panic recovered in %s %s: %v",
		msgKind,
		typeName,
		r,
	)
}

func handleRecovery(cfg middlewareConfig, msgKind, typeName string, r any) error {
	err := panicError(msgKind, typeName, r)

	if cfg.logger != nil {
		cfg.logger.Error(
			"panic recovered",
			"kind", msgKind,
			"type", typeName,
			"panic", r,
		)
	}

	return fmt.Errorf("msgKind=%s, typeName=%s: %w", msgKind, typeName, err)
}

// NewRecovery returns a generic middleware that recovers from panics.
func NewRecovery[M any](adapter MessageAdapter[M], opts ...Option) Middleware[M] {
	cfg := applyOptions(opts)

	return func(next Handler[M]) Handler[M] {
		return func(ctx context.Context, msg M) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = handleRecovery(cfg, adapter.Kind, adapter.ExtractType(msg), r)
				}
			}()

			return next(ctx, msg)
		}
	}
}

// CommandRecovery returns a command middleware that recovers from panics.
func CommandRecovery(opts ...Option) command.Middleware {
	return AsCommand(NewRecovery(CommandAdapter, opts...))
}

// EventRecovery returns an event middleware that recovers from panics.
func EventRecovery(opts ...Option) event.Middleware {
	return AsEvent(NewRecovery(EventAdapter, opts...))
}

// QueryRecovery returns a query middleware that recovers from panics.
func QueryRecovery(opts ...Option) query.Middleware {
	return AsQuery(NewRecovery(QueryAdapter, opts...))
}
