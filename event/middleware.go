package event

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
)

// RejectingPublishMiddleware returns a PublishMiddleware whose publisher
// rejects every publish with a Rejection carrying the given code and message.
//
// It is the shared nil-guard for middleware constructors (e.g.
// signing.SignMiddleware, encryption.EncryptMiddleware): when a required
// dependency is nil, return this instead of panicking so the error surfaces
// at first use, at the call site, with a clear code.
func RejectingPublishMiddleware(code, msg string) PublishMiddleware {
	return func(_ Publisher) Publisher {
		return PublisherFunc(func(_ context.Context, _ ...Event) error {
			return errorfamily.NewRejection(code, msg)
		})
	}
}

// RejectingHandlerMiddleware returns a Middleware whose handler rejects
// every event with a Rejection carrying the given code and message.
//
// It is the handler-side counterpart to [RejectingPublishMiddleware].
func RejectingHandlerMiddleware(code, msg string) Middleware {
	return func(_ Handler) Handler {
		return func(_ context.Context, _ Event) error {
			return errorfamily.NewRejection(code, msg)
		}
	}
}
