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
	metaJSON, _ := json.Marshal(cmd.Metadata())

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

// ─── encode/decode helpers ───

func (a *CommandAdapter) commandsToAny(cmds []*command.PersistedCommand) []any {
	if !a.serialize {
		result := make([]any, len(cmds))
		for i, cmd := range cmds {
			result[i] = cmd
		}

		return result
	}

	result := make([]any, len(cmds))
	for i, cmd := range cmds {
		result[i] = a.encodeCommand(cmd)
	}

	return result
}

func (a *CommandAdapter) anyToCommands(values []any) ([]*command.PersistedCommand, error) {
	result := make([]*command.PersistedCommand, 0, len(values))
	for _, val := range values {
		cmd, err := a.decodeCommandValue(val)
		if err != nil {
			return nil, err
		}

		result = append(result, cmd)
	}

	return result, nil
}

func (a *CommandAdapter) decodeCommandValue(val any) (*command.PersistedCommand, error) {
	// Direct pointer (Memory engine).
	if cmd, ok := val.(*command.PersistedCommand); ok {
		return cmd, nil
	}

	// Serialized string (SQLite/Pebble engine, raw string passthrough).
	if s, ok := val.(string); ok {
		return a.decodeCommand(s)
	}

	// Decoded JSON map (SQLite engine auto-decodes JSON strings on read).
	// Re-marshal to JSON and decode as a serializedCommand envelope.
	if m, ok := val.(map[string]any); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("command adapter: re-marshal decoded value: %w", err)
		}

		return a.decodeCommand(string(data))
	}

	return nil, fmt.Errorf("%w: %T", ErrUnsupportedValueType, val)
}
