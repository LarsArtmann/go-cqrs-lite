package encryption

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2"
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
		return signing.RejectingPublishMiddleware(
			"encryption.nil_encrypter",
			"EncryptMiddleware called with nil encrypter",
		)
	}

	cfg := middlewareConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	alg := detectAlgorithm(encrypter)

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			encrypted := make([]event.Event, 0, len(events))

			for _, evt := range events {
				payload := event.PayloadReadOnly(evt)
				if len(payload) == 0 {
					encrypted = append(encrypted, evt)

					continue
				}

				ct, err := encrypter.Encrypt(payload)
				if err != nil {
					return event.WrapInfrastructure(
						err,
						"encryption.encrypt_event",
						"encrypt event "+string(evt.Type()),
					)
				}

				attachOpts := []AttachOption{}
				if !alg.IsZero() {
					attachOpts = append(attachOpts, func(c *attachConfig) { c.algorithm = alg })
				}

				if !cfg.keyID.IsZero() {
					attachOpts = append(attachOpts, WithKeyID(cfg.keyID))
				}

				clone, err := AttachEncryption(evt, ct, attachOpts...)
				if err != nil {
					return event.WrapInfrastructure(
						err,
						"encryption.attach_ciphertext",
						"attach ciphertext to event "+string(evt.Type()),
					)
				}

				encrypted = append(encrypted, clone)
			}

			return next.Publish(ctx, encrypted...)
		})
	}
}

func DecryptMiddleware(decrypter Decrypter) event.Middleware {
	if decrypter == nil {
		return signing.RejectingHandlerMiddleware(
			"encryption.nil_decrypter",
			"DecryptMiddleware called with nil decrypter",
		)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			ct, err := ExtractCiphertext(evt)
			if err != nil {
				return next(ctx, evt)
			}

			plaintext, err := decrypter.Decrypt(ct)
			if err != nil {
				return event.WrapInfrastructure(
					err,
					"encryption.decrypt_event",
					"decrypt event "+string(evt.Type()),
				)
			}

			md := evt.Metadata().Clone()
			delete(md.Custom, MetadataKey)
			delete(md.Custom, AlgorithmKey)
			delete(md.Custom, KeyIDKey)

			plainEvt, err := event.NewEvent(
				evt.Type(),
				evt.AggregateID(),
				evt.AggregateType(),
				evt.Version(),
				plaintext,
				event.WithEventID(evt.ID()),
				event.WithOccurredAt(evt.OccurredAt()),
				event.WithSchemaVersion(evt.SchemaVersion()),
				event.WithMetadata(md),
			)
			if err != nil {
				return event.WrapInfrastructure(
					err,
					"encryption.rebuild_event",
					"rebuild decrypted event "+string(evt.Type()),
				)
			}

			return next(ctx, plainEvt)
		}
	}
}

func detectAlgorithm(encrypter Encrypter) Algorithm {
	type algorithmer interface {
		Algorithm() Algorithm
	}

	if a, ok := encrypter.(algorithmer); ok {
		return a.Algorithm()
	}

	return ""
}
