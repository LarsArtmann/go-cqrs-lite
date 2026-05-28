package signing

import (
	"context"
	"errors"
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
	if signer == nil {
		panic("signing: SignMiddleware called with nil signer")
	}

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
func VerifyMiddleware(verifier Verifier) event.Middleware {
	if verifier == nil {
		panic("signing: VerifyMiddleware called with nil verifier")
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			sig, err := ExtractSignature(evt)
			if err != nil {
				if errors.Is(err, ErrNilSignature) {
					return next(ctx, evt)
				}

				return fmt.Errorf("corrupt signature on event %s: %w", evt.Type(), err)
			}

			err = verifier.Verify(evt, sig)
			if err != nil {
				return fmt.Errorf("verify event %s: %w", evt.Type(), err)
			}

			return next(ctx, evt)
		}
	}
}

// RequireSignatureMiddleware returns event.Middleware that rejects events
// without signatures. Use when all events in a stream must be signed.
func RequireSignatureMiddleware(verifier Verifier) event.Middleware {
	if verifier == nil {
		panic("signing: RequireSignatureMiddleware called with nil verifier")
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasSignature(evt) {
				return fmt.Errorf(
					"%w: event %s is missing a signature",
					ErrNilSignature,
					evt.Type(),
				)
			}

			sig, extractErr := ExtractSignature(evt)
			if extractErr != nil {
				return fmt.Errorf("corrupt signature on event %s: %w", evt.Type(), extractErr)
			}

			verifyErr := verifier.Verify(evt, sig)
			if verifyErr != nil {
				return fmt.Errorf("verify event %s: %w", evt.Type(), verifyErr)
			}

			return next(ctx, evt)
		}
	}
}
