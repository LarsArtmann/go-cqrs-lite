package encryption

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type middlewareConfig struct {
	keyID KeyID
}

type MiddlewareOption func(*middlewareConfig)

func WithMiddlewareKeyID(id KeyID) MiddlewareOption {
	return func(c *middlewareConfig) { c.keyID = id }
}

func rejectingPublishMiddleware(code, msg string) event.PublishMiddleware {
	return func(_ event.Publisher) event.Publisher {
		return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
			return errorfamily.NewRejection(code, msg)
		})
	}
}

func rejectingHandlerMiddleware(code, msg string) event.Middleware {
	return func(_ event.Handler) event.Handler {
		return func(_ context.Context, _ event.Event) error {
			return errorfamily.NewRejection(code, msg)
		}
	}
}

func EncryptMiddleware(encrypter Encrypter, opts ...MiddlewareOption) event.PublishMiddleware {
	if encrypter == nil {
		return rejectingPublishMiddleware(
			"encryption.nil_encrypter",
			"EncryptMiddleware called with nil encrypter",
		)
	}

	cfg := middlewareConfig{} //nolint:exhaustruct // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			encrypted := make([]event.Event, 0, len(events))

			for _, evt := range events {
				enc, err := encryptEvent(evt, encrypter, cfg.keyID)
				if err != nil {
					return err
				}

				encrypted = append(encrypted, enc)
			}

			return next.Publish(ctx, encrypted...)
		})
	}
}

func DecryptMiddleware(decrypter Decrypter) event.Middleware {
	if decrypter == nil {
		return rejectingHandlerMiddleware(
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
