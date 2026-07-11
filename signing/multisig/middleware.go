package multisig

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
)

// MultiSignMiddleware returns event.PublishMiddleware that signs every published event
// on behalf of the given actor, appending to any existing multi-signature entries.
//
// Chain multiple MultiSignMiddleware calls for each actor in the pipeline:
//
//	deviceSigner := multisig.NewMultiSigner("device", multisig.AlgorithmEd25519, edSigner)
//	serverSigner := multisig.NewMultiSigner("server", multisig.AlgorithmHMACSHA256, hmacSigner)
//
//	bus.UsePublish(multisig.MultiSignMiddleware(deviceSigner))
//	bus.UsePublish(multisig.MultiSignMiddleware(serverSigner))
func MultiSignMiddleware(signer *MultiSigner) event.PublishMiddleware {
	if signer == nil {
		return signing.RejectingPublishMiddleware(
			"signing.nil_signer",
			"MultiSignMiddleware called with nil signer",
		)
	}

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			signed := make([]event.Event, 0, len(events))

			for _, evt := range events {
				clone, err := signer.Sign(evt)
				if err != nil {
					return errorfamily.WrapInfrastructure(
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
		return signing.RejectingHandlerMiddleware(
			"signing.nil_verifier",
			"MultiVerifyMiddleware called with nil signer",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			_, handled, err := signing.ExtractOrPassThrough(
				ctx, evt, next, ExtractMultiSignature,
				"signing.corrupt_multi_sig", "corrupt multi-sig on event "+string(evt.Type()),
			)
			if handled {
				if err != nil {
					return errorfamily.WrapCorruption(
						err,
						"signing.multi_verify_extract",
						"multi-verify extract",
					)
				}

				return nil
			}

			verifyErr := signer.Verify(evt)
			if verifyErr != nil {
				return errorfamily.WrapInfrastructure(
					verifyErr,
					"signing.verify_multi_sig",
					"verify multi-sig for actor "+string(
						signer.Actor(),
					)+" on event "+string(
						evt.Type(),
					),
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
func MultiVerifyMiddlewareFor(actor Actor, verifier signing.Verifier) event.Middleware {
	if verifier == nil {
		return signing.RejectingHandlerMiddleware(
			"signing.nil_verifier",
			"MultiVerifyMiddlewareFor called with nil verifier",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			multiSig, handled, err := signing.ExtractOrPassThrough(
				ctx, evt, next, ExtractMultiSignature,
				"signing.corrupt_multi_sig", "corrupt multi-sig on event "+string(evt.Type()),
			)
			if handled {
				if err != nil {
					return errorfamily.WrapCorruption(
						err,
						"signing.multi_verify_for_extract",
						"multi-verify-for extract",
					)
				}

				return nil
			}

			entry := multiSig.Get(actor)
			if entry == nil {
				return errorfamily.Newf(
					errorfamily.Rejection,
					"signing.missing_actor_signature",
					"no signature from actor %s on event %s",
					actor,
					evt.Type(),
				)
			}

			verifyErr := verifier.Verify(evt, entry.Sig)
			if verifyErr != nil {
				return errorfamily.WrapInfrastructure(
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
func RequireMultiSigMiddleware(verifiers map[Actor]signing.Verifier) event.Middleware {
	if len(verifiers) == 0 {
		return func(_ event.Handler) event.Handler {
			return func(_ context.Context, _ event.Event) error {
				return errorfamily.NewRejection(
					"signing.empty_verifiers",
					"RequireMultiSigMiddleware called with empty verifiers map",
				)
			}
		}
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if evt == nil {
				return errorfamily.WrapRejection(
					signing.ErrNilSignature,
					"signing.nil_event_multi_sig",
					"nil event",
				)
			}

			multiSig, err := ExtractMultiSignature(evt)
			if err != nil {
				return errorfamily.WrapRejection(
					signing.ErrNilSignature,
					"signing.no_multi_sig",
					"event "+string(evt.Type())+" has no multi-signature",
				)
			}

			for actor := range verifiers {
				if !multiSig.HasActor(actor) {
					return errorfamily.Newf(
						errorfamily.Rejection,
						"signing.missing_actor_signature",
						"event %s missing signature from actor %s",
						evt.Type(),
						actor,
					)
				}
			}

			verifyErr := VerifyAll(evt, verifiers)
			if verifyErr != nil {
				return errorfamily.WrapInfrastructure(
					verifyErr,
					"signing.require_multi_sig",
					"require multi-sig",
				)
			}

			return next(ctx, evt)
		}
	}
}
