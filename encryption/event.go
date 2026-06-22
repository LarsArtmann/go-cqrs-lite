package encryption

import (
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

const MetadataKey event.MetadataKey = "event.encrypted"

type attachConfig struct {
	algorithm Algorithm
	keyID     KeyID
}

type AttachOption func(*attachConfig)

func WithKeyID(id KeyID) AttachOption {
	return func(c *attachConfig) { c.keyID = id }
}

func AttachEncryption(
	evt event.Event,
	ciphertext Ciphertext,
	opts ...AttachOption,
) (event.Event, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	cfg := attachConfig{} //nolint:exhaustruct // zero-valued fields are ready
	for _, o := range opts {
		o(&cfg)
	}

	encoded := base64.URLEncoding.EncodeToString(ciphertext.Bytes())

	eventOpts := []event.Option{
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
		event.WithCustom(MetadataKey, encoded),
	}

	if !cfg.algorithm.IsZero() {
		eventOpts = append(eventOpts, event.WithCustom(AlgorithmKey, cfg.algorithm.String()))
	}

	if !cfg.keyID.IsZero() {
		eventOpts = append(eventOpts, event.WithCustom(KeyIDKey, cfg.keyID.String()))
	}

	clone, err := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		[]byte(ciphertext),
		eventOpts...,
	)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.attach",
			"reconstruct event with ciphertext",
		)
	}

	return clone, nil
}

func ExtractCiphertext(evt event.Event) (Ciphertext, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	decoded, found, err := event.ExtractCustomBytes(evt, MetadataKey)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.extract",
			"extract ciphertext from event",
		)
	}

	if !found {
		return nil, ErrNilCiphertext
	}

	return Ciphertext(decoded), nil
}

func HasEncryption(evt event.Event) bool {
	_, err := ExtractCiphertext(evt)
	if err == nil {
		return true
	}

	return event.Classify(err) != event.Rejection
}
