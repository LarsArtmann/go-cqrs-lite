package signing

import (
	"context"

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
	if signer == nil {
		panic("signing: MultiSignMiddleware called with nil signer")
	}

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			signed := make([]event.Event, 0, len(events))

			for _, evt := range events {
				clone, err := signer.Sign(evt)
				if err != nil {
					return event.WrapInfrastructure(
						err,
						"signing.multi_sign_event",
						"multi-sign event "+string(evt.Type())+" as "+string(signer.Actor()),
					)
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
	if signer == nil {
		panic("signing: MultiVerifyMiddleware called with nil signer")
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			_, handled, err := extractOrPassThrough(
				ctx, evt, next, ExtractMultiSignature,
				"signing.corrupt_multi_sig", "corrupt multi-sig on event "+string(evt.Type()),
			)
			if handled {
				return err
			}

			verifyErr := signer.Verify(evt)
			if verifyErr != nil {
				return event.WrapInfrastructure(
					verifyErr,
					"signing.verify_multi_sig",
					"verify multi-sig for actor "+string(signer.Actor())+" on event "+string(evt.Type()),
				)
			}

			return next(ctx, evt)
		}
	}
}

// MultiVerifyMiddlewareFor returns event.Middleware that verifies a specific
// actor's signature without requiring the caller to construct a *MultiSigner.
// This is a convenience wrapper for the common case where you already have
// a Verifier and just want to check one actor's signature.
func MultiVerifyMiddlewareFor(actor Actor, verifier Verifier) event.Middleware {
	if verifier == nil {
		panic("signing: MultiVerifyMiddlewareFor called with nil verifier")
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			multiSig, handled, err := extractOrPassThrough(
				ctx, evt, next, ExtractMultiSignature,
				"signing.corrupt_multi_sig", "corrupt multi-sig on event "+string(evt.Type()),
			)
			if handled {
				return err
			}

			entry := multiSig.Get(actor)
			if entry == nil {
				return event.Newf(
					event.Rejection,
					"signing.missing_actor_signature",
					"no signature from actor %s on event %s",
					actor,
					evt.Type(),
				)
			}

			verifyErr := verifier.Verify(evt, entry.Sig)
			if verifyErr != nil {
				return event.WrapInfrastructure(
					verifyErr,
					"signing.verify_multi_sig",
					"verify multi-sig for actor "+string(actor)+" on event "+string(evt.Type()),
				)
			}

			return next(ctx, evt)
		}
	}
}

// RequireMultiSigMiddleware returns event.Middleware that rejects events
// missing verified signatures from all actors in the provided verifier map.
// Cryptographically verifies every signature entry and ensures every actor
// in the verifier map has a corresponding valid signature.
func RequireMultiSigMiddleware(verifiers map[Actor]Verifier) event.Middleware {
	if len(verifiers) == 0 {
		panic("signing: RequireMultiSigMiddleware called with empty verifiers map")
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if evt == nil {
				return event.WrapRejection(
					ErrNilSignature,
					"signing.nil_event_multi_sig",
					"nil event",
				)
			}

			multiSig, err := ExtractMultiSignature(evt)
			if err != nil {
				return event.WrapRejection(
					ErrNilSignature,
					"signing.no_multi_sig",
					"event "+string(evt.Type())+" has no multi-signature",
				)
			}

			for actor := range verifiers {
				if !multiSig.HasActor(actor) {
					return event.Newf(
						event.Rejection,
						"signing.missing_actor_signature",
						"event %s missing signature from actor %s",
						evt.Type(),
						actor,
					)
				}
			}

			verifyErr := VerifyAll(evt, verifiers)
			if verifyErr != nil {
				return event.WrapInfrastructure(
					verifyErr,
					"signing.require_multi_sig",
					"require multi-sig",
				)
			}

			return next(ctx, evt)
		}
	}
}
