package watermill

import (
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// Metadata keys for command field mapping. Tracing and custom keys are shared
// with the event protocol (same semantics); command-specific keys are new.
const (
	metaCommandID   = "command_id"
	metaCommandType = "command_type"
)

// metadataProvider is the optional interface a command implements to expose
// metadata for wire serialization. *command.BasicCommand satisfies this.
type metadataProvider interface {
	Metadata() command.Metadata
}

// CommandToMessage maps a go-cqrs-lite command to a Watermill message.
// All command fields are preserved in message metadata; message payload is
// empty (commands carry routing identity, not serialized domain data —
// consumers encode payloads via custom metadata, same as transport/grpc).
//
// The command's [command.Command.ID] is used as the Watermill message UUID
// for dedup and traceability. The ID is minted at [command.New] time and
// stable for the lifetime of the command object — retrying with the same
// command instance preserves the ID. For cross-retry idempotency, use
// [command.WithCommandID] with a deterministic key.
//
// It is the inverse of [MessageToCommand]. Exported so callers that publish
// commands directly to a Watermill topic can build messages without
// duplicating the field-mapping protocol.
func CommandToMessage(cmd command.Command) *message.Message {
	cmdID := cmd.ID()
	msg := message.NewMessage(cmdID.String(), nil)

	md := msg.Metadata
	md.Set(metaCommandID, cmdID.String())
	md.Set(metaCommandType, string(cmd.Type()))
	md.Set(metaAggregateID, cmd.StreamID().String())

	if mp, ok := cmd.(metadataProvider); ok {
		m := mp.Metadata()
		writeTracing(md, m.Tracing)
		writeCustomEntries(md, m.Custom)
	}

	return msg
}

// MessageToCommand reconstructs a go-cqrs-lite command from a Watermill message.
// The topic is used as the command type fallback; all other fields come from
// metadata. Exported so other packages can reuse the same protocol instead of
// duplicating decode logic.
func MessageToCommand(topic string, msg *message.Message) (*command.BasicCommand, error) {
	md := msg.Metadata

	cmdType := command.Type(topic)
	if v := md.Get(metaCommandType); v != "" {
		cmdType = command.Type(v)
	}

	if cmdType.IsZero() {
		return nil, errorfamily.NewRejection(
			"watermill.missing_metadata",
			"missing "+metaCommandType+" metadata and empty topic",
		)
	}

	streamID, err := id.ParseStreamID(md.Get(metaAggregateID))
	if err != nil {
		return nil, errorfamily.WrapRejection(err,
			"watermill.parse_aggregate_id_failed", "parse stream_id")
	}

	opts := parseCommandOptions(md)

	cmd, err := command.New(cmdType, streamID, opts...)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			"watermill.create_command_failed",
			"create command",
		)
	}

	return cmd, nil
}

// writeTracing writes the 4 shared tracing identifiers from metadata.Tracing
// into message metadata. Both event.Metadata and command.Metadata embed
// metadata.Tracing, so this is reused by both protocols.
func writeTracing(md message.Metadata, t metadata.Tracing) {
	if !t.CorrelationID.IsZero() {
		md.Set(metaCorrelationID, t.CorrelationID.String())
	}
	if !t.CausationID.IsZero() {
		md.Set(metaCausationID, t.CausationID.String())
	}
	if !t.UserID.IsZero() {
		md.Set(metaUserID, t.UserID.String())
	}
	if !t.RequestID.IsZero() {
		md.Set(metaRequestID, t.RequestID.String())
	}
}

// writeCustomEntries writes custom metadata entries with the custom. prefix.
// Works for any map whose key type is ~string.
func writeCustomEntries[K ~string](md message.Metadata, custom map[K]string) {
	for k, v := range custom {
		md.Set(metaCustomPrefix+string(k), v)
	}
}

func parseCommandOptions(md message.Metadata) []command.Option {
	var opts []command.Option

	if cmdIDStr := md.Get(metaCommandID); cmdIDStr != "" {
		if cmdID, err := id.ParseCommandID(cmdIDStr); err == nil {
			opts = append(opts, command.WithCommandID(cmdID))
		}
	}

	parseIDOption(
		md, metaCorrelationID, id.ParseCorrelationID,
		func(v id.CorrelationID) { opts = append(opts, command.WithCorrelationID(v)) },
	)
	parseIDOption(
		md, metaCausationID, id.ParseCausationID,
		func(v id.CausationID) { opts = append(opts, command.WithCausationID(v)) },
	)
	parseIDOption(
		md, metaUserID, id.ParseUserID,
		func(v id.UserID) { opts = append(opts, command.WithUserID(v)) },
	)
	parseIDOption(
		md, metaRequestID, id.ParseRequestID,
		func(v id.RequestID) { opts = append(opts, command.WithRequestID(v)) },
	)

	for k, v := range md {
		if after, ok := strings.CutPrefix(k, metaCustomPrefix); ok {
			opts = append(opts, command.WithCustomMetadata(after, v))
		}
	}

	return opts
}

func parseIDOption[T any](
	md message.Metadata,
	key string,
	parse func(string) (T, error),
	set func(T),
) {
	v := md.Get(key)
	if v == "" {
		return
	}

	parsed, err := parse(v)
	if err != nil {
		return
	}

	set(parsed)
}
