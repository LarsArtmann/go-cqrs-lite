package signing

import (
	"context"
	"errors"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// ExtractOrPassThrough extracts a value from an event, passing through to the next handler
// if no signature is present (ErrNilSignature). Returns the extracted value, whether the
// event was already handled, and any error.
func ExtractOrPassThrough[T any](
	ctx context.Context,
	evt event.Event,
	next event.Handler,
	extract func(event.Event) (T, error),
	code, msg string,
) (T, bool, error) {
	var zero T

	result, err := extract(evt)
	if err != nil {
		if errors.Is(err, ErrNilSignature) {
			return zero, true, next(ctx, evt)
		}

		return zero, true, event.WrapInfrastructure(err, code, msg)
	}

	return result, false, nil
}

func RejectingPublishMiddleware(code, msg string) event.PublishMiddleware {
	return func(_ event.Publisher) event.Publisher {
		return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
			return event.NewRejection(code, msg)
		})
	}
}

func RejectingHandlerMiddleware(code, msg string) event.Middleware {
	return func(_ event.Handler) event.Handler {
		return func(_ context.Context, _ event.Event) error {
			return event.NewRejection(code, msg)
		}
	}
}

// SignMiddleware returns event.PublishMiddleware that signs every published event.
// The signer must not be nil. Errors during signing are returned as publish errors.
//
// Usage:
//
//	bus.UsePublish(signing.SignMiddleware(signer))
func SignMiddleware(signer Signer) event.PublishMiddleware {
	if signer == nil {
		return RejectingPublishMiddleware(
			"signing.nil_signer",
			"SignMiddleware called with nil signer",
		)
	}

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			signed := make([]event.Event, 0, len(events))

			for _, evt := range events {
				sig, err := signer.Sign(evt)
				if err != nil {
					return event.WrapInfrastructure(
						err,
						"signing.sign_event",
						"sign event "+string(evt.Type()),
					)
				}

				clone, err := AttachSignature(evt, sig)
				if err != nil {
					return event.WrapInfrastructure(
						err,
						"signing.attach_signature",
						"attach signature to event "+string(evt.Type()),
					)
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
		return RejectingHandlerMiddleware(
			"signing.nil_verifier",
			"VerifyMiddleware called with nil verifier",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			sig, handled, err := ExtractOrPassThrough(
				ctx, evt, next, ExtractSignature,
				"signing.corrupt_signature", "corrupt signature on event "+string(evt.Type()),
			)
			if handled {
				return err
			}

			err = verifier.Verify(evt, sig)
			if err != nil {
				return event.WrapInfrastructure(
					err,
					"signing.verify_event",
					"verify event "+string(evt.Type()),
				)
			}

			return next(ctx, evt)
		}
	}
}

// RequireSignatureMiddleware returns event.Middleware that rejects events
// without signatures. Use when all events in a stream must be signed.
func RequireSignatureMiddleware(verifier Verifier) event.Middleware {
	if verifier == nil {
		return RejectingHandlerMiddleware(
			"signing.nil_verifier",
			"RequireSignatureMiddleware called with nil verifier",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			if !HasSignature(evt) {
				return event.Newf(
					event.Rejection,
					"signing.missing_signature",
					"event %s is missing a signature",
					evt.Type(),
				)
			}

			sig, extractErr := ExtractSignature(evt)
			if extractErr != nil {
				return event.WrapInfrastructure(
					extractErr,
					"signing.corrupt_signature",
					"corrupt signature on event "+string(evt.Type()),
				)
			}

			verifyErr := verifier.Verify(evt, sig)
			if verifyErr != nil {
				return event.WrapInfrastructure(
					verifyErr,
					"signing.verify_event",
					"verify event "+string(evt.Type()),
				)
			}

			return next(ctx, evt)
		}
	}
}
