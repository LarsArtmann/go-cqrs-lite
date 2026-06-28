package watermill

import (
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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
// A fresh command ID is generated per call for Watermill message dedup and
// traceability. Commands do not carry pre-existing IDs on the Bus contract.
//
// It is the inverse of [MessageToCommand]. Exported so callers that publish
// commands directly to a Watermill topic can build messages without
// duplicating the field-mapping protocol.
func CommandToMessage(cmd command.Command) *message.Message {
	cmdID := id.NewCommandID()
	msg := message.NewMessage(cmdID.String(), nil)

	md := msg.Metadata
	md.Set(metaCommandID, cmdID.String())
	md.Set(metaCommandType, string(cmd.Type()))
	md.Set(metaAggregateID, cmd.AggregateID().String())

	if mp, ok := cmd.(metadataProvider); ok {
		m := mp.Metadata()
		writeTracingMetadata(md, m)
		writeCustomMetadata(md, m)
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
		return nil, event.NewRejection(
			"watermill.missing_metadata",
			"missing "+metaCommandType+" metadata and empty topic",
		)
	}

	aggregateID, err := id.ParseAggregateID(md.Get(metaAggregateID))
	if err != nil {
		return nil, event.WrapRejection(err,
			"watermill.parse_aggregate_id_failed", "parse aggregate_id")
	}

	opts := parseCommandOptions(md)

	cmd, err := command.New(cmdType, aggregateID, opts...)
	if err != nil {
		return nil, event.WrapCorruption(err, "watermill.create_command_failed", "create command")
	}

	return cmd, nil
}

func writeTracingMetadata(md message.Metadata, m command.Metadata) {
	if !m.CorrelationID.IsZero() {
		md.Set(metaCorrelationID, m.CorrelationID.String())
	}
	if !m.CausationID.IsZero() {
		md.Set(metaCausationID, m.CausationID.String())
	}
	if !m.UserID.IsZero() {
		md.Set(metaUserID, m.UserID.String())
	}
	if !m.RequestID.IsZero() {
		md.Set(metaRequestID, m.RequestID.String())
	}
}

func writeCustomMetadata(md message.Metadata, m command.Metadata) {
	for k, v := range m.Custom {
		md.Set(metaCustomPrefix+string(k), v)
	}
}

func parseCommandOptions(md message.Metadata) []command.Option {
	var opts []command.Option

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
