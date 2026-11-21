package bbolt

import (
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type serializableCommand struct {
	ID         id.CommandID     `json:"id"`
	Type       string           `json:"type"`
	StreamID   id.StreamID      `json:"stream_id"`
	StreamType string           `json:"stream_type"`
	ReceivedAt int64            `json:"received_at"`
	Payload    []byte           `json:"payload"`
	Metadata   command.Metadata `json:"metadata"`
}

func marshalCommand(cmd *command.PersistedCommand) ([]byte, error) {
	record := serializableCommand{
		ID:         cmd.ID(),
		Type:       string(cmd.Type()),
		StreamID:   cmd.StreamID(),
		StreamType: string(cmd.StreamType()),
		ReceivedAt: cmd.ReceivedAt().UnixNano(),
		Payload:    cmd.Payload(),
		Metadata:   cmd.Metadata(),
	}

	//art-dupl:accept mirror of event/query serialization — type-specific wire structs
	data, err := marshalCBOR(record)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.serialize_command",
			"marshal command")
	}

	return data, nil
}

func unmarshalCommand(data []byte) (*command.PersistedCommand, error) {
	var record serializableCommand

	if err := unmarshalCBOROrJSON(data, &record, "bbolt.unmarshal_command",
		"failed to unmarshal command"); err != nil {
		return nil, err
	}

	adoptLegacyStreamKeys(data, &record.StreamID, &record.StreamType)

	ref := id.StreamRef{Type: id.StreamType(record.StreamType), ID: record.StreamID}

	cmd, err := command.NewPersistedCommand(
		command.Type(record.Type),
		ref,
		record.Payload,
		command.WithReceivedAt(time.Unix(0, record.ReceivedAt).UTC()),
		command.WithPersistedCommandID(record.ID),
		command.WithCommandMetadata(record.Metadata),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.reconstruct_command",
			"reconstruct command")
	}

	return cmd, nil
}
