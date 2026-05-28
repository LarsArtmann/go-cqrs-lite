package signing

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// MultiSignMiddleware returns event.PublishMiddleware that signs every published event
// on behalf of the given actor, appending to any existing multi-signature entries.
//
// Chain multiple MultiSignMiddleware calls for each actor in the pipeline:
//
//	deviceSigner := signing.NewMultiSigner("device", signing.AlgorithmEd25519, edSigner)
//	serverSigner := signing.NewMultiSigner("server", signing.AlgorithmHMACSHA256, hmacSigner)
//
//	bus.UsePublish(signing.MultiSignMiddleware(deviceSigner))
//	bus.UsePublish(signing.MultiSignMiddleware(serverSigner))
func MultiSignMiddleware(signer *MultiSigner) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			signed := make([]event.Event, 0, len(events))

			for _, evt := range events {
				clone, err := signer.Sign(evt)
				if err != nil {
					return fmt.Errorf("multi-sign event %s as %s: %w", evt.Type(), signer.Actor(), err)
				}

				signed = append(signed, clone)
			}

			return next.Publish(ctx, signed...)
		})
	}
}

// MultiVerifyMiddleware returns event.Middleware that verifies the signature
// of a specific actor from the event's multi-sig collection before handling.
// Events without a multi-sig or without that actor's signature pass through
// (to support mixed streams). Use RequireMultiSigMiddleware to enforce presence.
func MultiVerifyMiddleware(signer *MultiSigner) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasMultiSignature(evt) {
				return next(ctx, evt)
			}

			verifyErr := signer.Verify(evt)
			if verifyErr != nil {
				return fmt.Errorf(
					"verify multi-sig for actor %s on event %s: %w",
					signer.Actor(),
					evt.Type(),
					verifyErr,
				)
			}

			return next(ctx, evt)
		}
	}
}

// RequireMultiSigMiddleware returns event.Middleware that rejects events
// missing a multi-signature collection. Each actor in the chain must have signed.
func RequireMultiSigMiddleware(actors ...string) event.Middleware {
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasMultiSignature(evt) {
				return fmt.Errorf(
					"%w: event %s is missing multi-signature collection",
					ErrNilSignature,
					evt.Type(),
				)
			}

			multiSig, extractErr := ExtractMultiSignature(evt)
			if extractErr != nil {
				return fmt.Errorf("extract multi-sig: %w", extractErr)
			}

			for _, actor := range actors {
				if !multiSig.HasActor(actor) {
					return fmt.Errorf(
						"%w: event %s missing signature from actor %s (got: %v)",
						ErrNilSignature,
						evt.Type(),
						actor,
						multiSig.Actors(),
					)
				}
			}

			return next(ctx, evt)
		}
	}
}
