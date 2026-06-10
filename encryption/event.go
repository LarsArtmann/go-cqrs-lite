package encryption

import (
	"encoding/base64"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const MetadataKey event.MetadataKey = "event.encrypted"

func cloneEvent(
	evt event.Event,
	key event.MetadataKey,
	value string,
) (*event.ImmutableEvent, error) {
	return event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		event.PayloadReadOnly(evt),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
		event.WithCustom(key, value),
	)
}

func AttachEncryption(evt event.Event, ciphertext Ciphertext) (*event.ImmutableEvent, error) {
	if evt == nil {
		return nil, ErrNilEvent
	}

	encoded := base64.URLEncoding.EncodeToString(ciphertext.Bytes())

	clone, err := event.NewEvent(
		evt.Type(),
		evt.AggregateID(),
		evt.AggregateType(),
		evt.Version(),
		[]byte(ciphertext),
		event.WithEventID(evt.ID()),
		event.WithOccurredAt(evt.OccurredAt()),
		event.WithSchemaVersion(evt.SchemaVersion()),
		event.WithMetadata(evt.Metadata()),
		event.WithCustom(MetadataKey, encoded),
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

	md := evt.Metadata()
	if md.Custom == nil {
		return nil, ErrNilCiphertext
	}

	encoded, ok := md.Custom[MetadataKey]
	if !ok || encoded == "" {
		return nil, ErrNilCiphertext
	}

	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"encryption.decode_ciphertext",
			"decode ciphertext from base64",
		)
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
