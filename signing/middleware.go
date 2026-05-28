package signing

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// SignMiddleware returns event.PublishMiddleware that signs every published event.
// The signer must not be nil. Errors during signing are returned as publish errors.
//
// Usage:
//
//	bus.UsePublish(signing.SignMiddleware(signer))
func SignMiddleware(signer Signer) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			signed := make([]event.Event, 0, len(events))

			for _, evt := range events {
				sig, err := signer.Sign(evt)
				if err != nil {
					return fmt.Errorf("sign event %s: %w", evt.Type(), err)
				}

				clone, err := AttachSignature(evt, sig)
				if err != nil {
					return fmt.Errorf("attach signature to event %s: %w", evt.Type(), err)
				}

				signed = append(signed, clone)
			}

			return next.Publish(ctx, signed...)
		})
	}
}

// VerifyMiddleware returns event.Middleware that verifies event signatures before handling.
// Events without signatures pass through (to support mixed streams).
// Events with invalid signatures are rejected with ErrInvalidSignature.
//
// Usage:
//
//	bus.Use(signing.VerifyMiddleware(signer))
func VerifyMiddleware(signer Signer) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasSignature(evt) {
				return next(ctx, evt)
			}

			sig, err := ExtractSignature(evt)
			if err != nil {
				return fmt.Errorf("extract signature for event %s: %w", evt.Type(), err)
			}

			err = signer.Verify(evt, sig)
			if err != nil {
				return fmt.Errorf("verify event %s: %w", evt.Type(), err)
			}

			return next(ctx, evt)
		}
	}
}

// RequireSignatureMiddleware returns event.Middleware that rejects events
// without signatures. Use when all events in a stream must be signed.
func RequireSignatureMiddleware(signer Signer) event.Middleware {
	verify := VerifyMiddleware(signer)

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasSignature(evt) {
				return fmt.Errorf(
					"%w: event %s is missing a signature",
					ErrNilSignature,
					evt.Type(),
				)
			}

			return verify(next)(ctx, evt)
		}
	}
}
