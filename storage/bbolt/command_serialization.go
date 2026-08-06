package bbolt

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type serializableCommand struct {
	ID         id.CommandID     `json:"id"`
	Type       string           `json:"type"`
	StreamID   id.StreamID      `json:"aggregate_id"`
	StreamType string           `json:"aggregate_type"`
	ReceivedAt int64            `json:"received_at"`
	Payload    []byte           `json:"payload"`
	Metadata   command.Metadata `json:"metadata"`
}

func marshalCommand(cmd *command.PersistedCommand) ([]byte, error) {
	sc := serializableCommand{
		ID:         cmd.ID(),
		Type:       string(cmd.Type()),
		StreamID:   cmd.StreamID(),
		StreamType: string(cmd.StreamType()),
		ReceivedAt: cmd.ReceivedAt().UnixNano(),
		Payload:    cmd.Payload(),
		Metadata:   cmd.Metadata(),
	}

	data, err := marshalCBOR(sc)
	if err != nil {
		return nil, fmt.Errorf("marshal command: %w", err)
	}

	return data, nil
}

func unmarshalCommand(data []byte) (*command.PersistedCommand, error) {
	var sc serializableCommand

	if err := unmarshalCBOROrJSON(data, &sc, "bbolt.unmarshal_command",
		"failed to unmarshal command"); err != nil {
		return nil, err
	}

	ref := id.StreamRef{Type: id.StreamType(sc.StreamType), ID: sc.StreamID}

	cmd, err := command.NewPersistedCommand(
		command.Type(sc.Type),
		ref,
		sc.Payload,
		command.WithReceivedAt(time.Unix(0, sc.ReceivedAt).UTC()),
		command.WithPersistedCommandID(sc.ID),
		command.WithCommandMetadata(sc.Metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("reconstruct command: %w", err)
	}

	return cmd, nil
}
