package encryption

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func rejectingPublishMiddleware(code, msg string) event.PublishMiddleware {
	return func(_ event.Publisher) event.Publisher {
		return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
			return event.NewRejection(code, msg)
		})
	}
}

func rejectingHandlerMiddleware(code, msg string) event.Middleware {
	return func(_ event.Handler) event.Handler {
		return func(_ context.Context, _ event.Event) error {
			return event.NewRejection(code, msg)
		}
	}
}

func EncryptMiddleware(encrypter Encrypter) event.PublishMiddleware {
	if encrypter == nil {
		return rejectingPublishMiddleware(
			"encryption.nil_encrypter",
			"EncryptMiddleware called with nil encrypter",
		)
	}

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

				clone, err := AttachEncryption(evt, ct)
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
		return rejectingHandlerMiddleware(
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

			decrypted, err := cloneEvent(evt, MetadataKey, "")
			if err != nil {
				return event.WrapInfrastructure(
					err,
					"encryption.reconstruct_plaintext",
					"reconstruct plaintext event "+string(evt.Type()),
				)
			}

			plainEvt, err := event.NewEvent(
				decrypted.Type(),
				decrypted.AggregateID(),
				decrypted.AggregateType(),
				decrypted.Version(),
				plaintext,
				event.WithEventID(decrypted.ID()),
				event.WithOccurredAt(decrypted.OccurredAt()),
				event.WithSchemaVersion(decrypted.SchemaVersion()),
				event.WithMetadata(decrypted.Metadata()),
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
