package encryption

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type middlewareConfig struct {
	keyID KeyID
}

type MiddlewareOption func(*middlewareConfig)

func WithMiddlewareKeyID(id KeyID) MiddlewareOption {
	return func(c *middlewareConfig) { c.keyID = id }
}

func EncryptMiddleware(encrypter Encrypter, opts ...MiddlewareOption) event.PublishMiddleware {
	if encrypter == nil {
		return event.RejectingPublishMiddleware(
			"encryption.nil_encrypter",
			"EncryptMiddleware called with nil encrypter",
		)
	}

	transform := EncryptSinkTransform(encrypter, opts...)

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			encrypted, err := transform(events)
			if err != nil {
				return err
			}

			return next.Publish(ctx, encrypted...)
		})
	}
}

func DecryptMiddleware(decrypter Decrypter) event.Middleware {
	if decrypter == nil {
		return event.RejectingHandlerMiddleware(
			"encryption.nil_decrypter",
			"DecryptMiddleware called with nil decrypter",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			decrypted, err := decryptEvent(evt, decrypter)
			if err != nil {
				return err
			}

			return next(ctx, decrypted)
		}
	}
}

func detectAlgorithm(encrypter Encrypter) Algorithm {
	if a, ok := encrypter.(Algorithmer); ok {
		return a.Algorithm()
	}

	return ""
}
