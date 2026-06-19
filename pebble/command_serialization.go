package pebble

import (
	"encoding/json"
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
	sc := serializableCommand{
		ID:            cmd.ID(),
		Type:          string(cmd.Type()),
		AggregateID:   cmd.AggregateID(),
		AggregateType: string(cmd.AggregateType()),
		ReceivedAt:    cmd.ReceivedAt().UnixNano(),
		Payload:       cmd.Payload(),
		Metadata:      cmd.Metadata(),
	}

	return pebbleEncMode.Marshal(sc) //nolint:wrapcheck // storage serialization, not domain error
}

func (s *CommandStore) deserializeCommand(data []byte) (*command.PersistedCommand, error) {
	var sc serializableCommand

	var err error

	if isCBOR(data) {
		err = pebbleDecMode.Unmarshal(data, &sc)
	} else {
		err = json.Unmarshal(data, &sc) //nolint:nolintlint // legacy JSON fallback for backward compat
	}

	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_command",
			"failed to unmarshal command")
	}

	ref := command.NewAggregateRef(
		command.AggregateType(sc.AggregateType),
		sc.AggregateID,
	)

	cmd, err := command.NewPersistedCommand(
		command.Type(sc.Type),
		ref,
		sc.Payload,
		command.WithCommandID(sc.ID),
		command.WithReceivedAt(time.Unix(0, sc.ReceivedAt)),
		command.WithCommandMetadata(sc.Metadata),
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.reconstruct_command",
			"failed to reconstruct command from stored fields")
	}

	return cmd, nil
}
