package pebble

import (
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// serializableCommand is the CBOR (and legacy JSON) storage format for commands.
// fxamacker/cbor reads `json` struct tags by default, so no separate `cbor` tags needed.
type serializableCommand struct {
	ID            id.CommandID     `json:"id"`
	Type          string           `json:"type"`
	StreamID   id.StreamID   `json:"aggregate_id"`
	StreamType string           `json:"aggregate_type"`
	ReceivedAt    int64            `json:"received_at"`
	Payload       []byte           `json:"payload"`
	Metadata      command.Metadata `json:"metadata"`
}

func (s *CommandStore) serializeCommand(cmd *command.PersistedCommand) ([]byte, error) {
	serialized := serializableCommand{
		ID:            cmd.ID(),
		Type:          string(cmd.Type()),
		StreamID:   cmd.StreamID(),
		StreamType: string(cmd.StreamType()),
		ReceivedAt:    cmd.ReceivedAt().UnixNano(),
		Payload:       cmd.Payload(),
		Metadata:      cmd.Metadata(),
	}

	return marshalCBOROrErr(serialized, "pebble.serialize_command", "marshal command")
}

func (s *CommandStore) deserializeCommand(data []byte) (*command.PersistedCommand, error) {
	var serialized serializableCommand

	if err := unmarshalCBOROrJSON(data, &serialized, "pebble.unmarshal_command",
		"failed to unmarshal command"); err != nil {
		return nil, err
	}

	ref := command.NewAggregateRef(
		command.StreamType(serialized.StreamType),
		serialized.StreamID,
	)

	cmd, err := command.NewPersistedCommand(
		command.Type(serialized.Type),
		ref,
		serialized.Payload,
		command.WithPersistedCommandID(serialized.ID),
		command.WithReceivedAt(time.Unix(0, serialized.ReceivedAt).UTC()),
		command.WithCommandMetadata(serialized.Metadata),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.reconstruct_command",
			"failed to reconstruct command from stored fields")
	}

	return cmd, nil
}
