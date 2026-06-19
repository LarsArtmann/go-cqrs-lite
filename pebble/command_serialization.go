package pebble

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// serializableCommand is the CBOR (and legacy JSON) storage format for commands.
// fxamacker/cbor reads `json` struct tags by default, so no separate `cbor` tags needed.
type serializableCommand struct {
	ID            id.CommandID     `json:"id"`
	Type          string           `json:"type"`
	AggregateID   id.AggregateID   `json:"aggregate_id"`   //nolint:tagliatelle // on-disk format uses snake_case
	AggregateType string           `json:"aggregate_type"` //nolint:tagliatelle // on-disk/external format uses snake_case
	ReceivedAt    int64            `json:"received_at"`    //nolint:tagliatelle // on-disk/external format uses snake_case
	Payload       []byte           `json:"payload"`
	Metadata      command.Metadata `json:"metadata"`
}

func (s *CommandStore) serializeCommand(cmd *command.PersistedCommand) ([]byte, error) {
	serialized := serializableCommand{
		ID:            cmd.ID(),
		Type:          string(cmd.Type()),
		AggregateID:   cmd.AggregateID(),
		AggregateType: string(cmd.AggregateType()),
		ReceivedAt:    cmd.ReceivedAt().UnixNano(),
		Payload:       cmd.Payload(),
		Metadata:      cmd.Metadata(),
	}

	data, err := pebbleEncMode.Marshal(serialized)
	if err != nil {
		return nil, fmt.Errorf("pebble: marshal command: %w", err)
	}

	return data, nil
}

func (s *CommandStore) deserializeCommand(data []byte) (*command.PersistedCommand, error) {
	var serialized serializableCommand

	var err error

	if isCBOR(data) {
		err = pebbleDecMode.Unmarshal(data, &serialized)
	} else {
		err = json.Unmarshal(
			data,
			&serialized,
		) //nolint:nolintlint // legacy JSON fallback for backward compat
	}

	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_command",
			"failed to unmarshal command")
	}

	ref := command.NewAggregateRef(
		command.AggregateType(serialized.AggregateType),
		serialized.AggregateID,
	)

	cmd, err := command.NewPersistedCommand(
		command.Type(serialized.Type),
		ref,
		serialized.Payload,
		command.WithCommandID(serialized.ID),
		command.WithReceivedAt(time.Unix(0, serialized.ReceivedAt)),
		command.WithCommandMetadata(serialized.Metadata),
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.reconstruct_command",
			"failed to reconstruct command from stored fields")
	}

	return cmd, nil
}
