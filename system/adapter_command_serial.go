package system

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// serializedCommand is the JSON envelope for persisting commands in SQL-based
// StreamLogBackends. The Memory engine stores pointers directly; SQL engines
// store this envelope as a TEXT value.
type serializedCommand struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	StreamID   string    `json:"stream_id"`
	StreamType string    `json:"stream_type"`
	ReceivedAt time.Time `json:"received_at"`
	Payload    []byte    `json:"payload"`
	Metadata   []byte    `json:"metadata"`
}

func (a *CommandAdapter) encodeCommand(cmd *command.PersistedCommand) string {
	// encodeCommand cannot propagate errors: AdapterCore.Encode is `func(T) string`
	// by design (ADR-0126 core constraint). On a failed metadata marshal the
	// envelope persists a nil Metadata field (decodes to zero-value metadata)
	// instead of partial JSON. Today's fields are all marshal-safe (typed
	// string IDs, map[K]string custom data); the guard keeps that guarantee
	// if richer values ever land.
	metaJSON, metaErr := json.Marshal(cmd.Metadata(), json.Deterministic(true))
	if metaErr != nil {
		metaJSON = nil
	}

	env := serializedCommand{
		ID:         cmd.ID().String(),
		Type:       string(cmd.Type()),
		StreamID:   cmd.StreamID().String(),
		StreamType: string(cmd.StreamType()),
		ReceivedAt: cmd.ReceivedAt(),
		Payload:    cmd.Payload(),
		Metadata:   metaJSON,
	}

	data, _ := json.Marshal(env)

	return string(data)
}

func (a *CommandAdapter) decodeCommand(s string) (*command.PersistedCommand, error) {
	var env serializedCommand
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, fmt.Errorf("command adapter: decode envelope: %w", err)
	}

	cmdID, err := id.ParseCommandID(env.ID)
	if err != nil {
		return nil, fmt.Errorf("command adapter: parse command ID: %w", err)
	}

	streamID, err := id.ParseStreamID(env.StreamID)
	if err != nil {
		return nil, fmt.Errorf("command adapter: parse stream ID: %w", err)
	}

	ref := command.NewStreamRef(command.StreamType(env.StreamType), streamID)

	var meta command.Metadata
	if len(env.Metadata) > 0 {
		if err := json.Unmarshal(env.Metadata, &meta); err != nil {
			return nil, fmt.Errorf("command adapter: decode metadata: %w", err)
		}
	}

	cmd, err := command.NewPersistedCommand(
		command.Type(env.Type), ref, env.Payload,
		command.WithPersistedCommandID(cmdID),
		command.WithReceivedAt(env.ReceivedAt),
		command.WithCommandMetadata(meta),
	)
	if err != nil {
		return nil, fmt.Errorf("command adapter: reconstruct command: %w", err)
	}

	return cmd, nil
}
