package watermill

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// Metadata keys for event field mapping.
const (
	metaEventID       = "event_id"
	metaEventType     = "event_type"
	metaAggregateID   = "aggregate_id"
	metaAggregateType = "aggregate_type"
	metaVersion       = "version"
	metaSchemaVersion = "schema_version"
	metaOccurredAt    = "occurred_at"
	metaCorrelationID = "correlation_id"
	metaCausationID   = "causation_id"
	metaUserID        = "user_id"
	metaRequestID     = "request_id"
	metaSource        = "source"
	metaIPAddress     = "ip_address"
	metaUserAgent     = "user_agent"
	metaCustomPrefix  = "custom."
)

// eventToMessage maps a go-cqrs-lite event to a Watermill message.
// All event fields are preserved in message metadata; payload is stored as message payload.
func eventToMessage(evt event.Event) *message.Message {
	msg := message.NewMessage(evt.ID().String(), evt.Payload())
	md := msg.Metadata

	md.Set(metaEventID, evt.ID().String())
	md.Set(metaEventType, string(evt.Type()))
	md.Set(metaAggregateID, evt.AggregateID().String())
	md.Set(metaAggregateType, string(evt.AggregateType()))
	md.Set(metaVersion, strconv.Itoa(evt.Version().Int()))
	md.Set(metaSchemaVersion, strconv.Itoa(evt.SchemaVersion().Int()))
	md.Set(metaOccurredAt, evt.OccurredAt().Format(time.RFC3339Nano))

	m := evt.Metadata()
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
	if m.Source != "" {
		md.Set(metaSource, string(m.Source))
	}
	if m.IPAddress != "" {
		md.Set(metaIPAddress, string(m.IPAddress))
	}
	if m.UserAgent != "" {
		md.Set(metaUserAgent, string(m.UserAgent))
	}
	for k, v := range m.Custom {
		md.Set(metaCustomPrefix+string(k), v)
	}

	return msg
}

// messageToEvent reconstructs a go-cqrs-lite event from a Watermill message.
// The topic is used as the event type; all other fields come from metadata.
func messageToEvent(topic string, msg *message.Message) (event.Event, error) {
	md := msg.Metadata

	eventType := event.Type(topic)
	if v := md.Get(metaEventType); v != "" {
		eventType = event.Type(v)
	}

	aggregateID, err := id.ParseAggregateID(md.Get(metaAggregateID))
	if err != nil {
		return nil, event.WrapRejection(err,
			"watermill.parse_aggregate_id_failed", "parse aggregate_id")
	}

	aggregateType := event.AggregateType(md.Get(metaAggregateType))
	if aggregateType == "" {
		return nil, event.NewRejection("watermill.missing_metadata",
			"missing "+metaAggregateType+" metadata")
	}

	version, err := parseInt(md.Get(metaVersion), metaVersion)
	if err != nil {
		return nil, event.WrapRejection(err,
			"watermill.parse_version_failed",
			fmt.Sprintf("topic %s: parse %s", topic, metaVersion))
	}

	schemaVersion, err := parseSchemaVersion(md, topic)
	if err != nil {
		return nil, err
	}

	opts := []event.Option{event.WithSchemaVersion(event.SchemaVersion(schemaVersion))}

	if eventOpts, err := parseOptionalFields(md); err != nil {
		return nil, event.Wrapf(
			err,
			event.Rejection,
			"watermill.parse_optional",
			"topic %s: parse optional fields",
			topic,
		)
	} else {
		opts = append(opts, eventOpts...)
	}

	metadata, metaErr := buildMetadata(md)
	opts = append(opts, event.WithMetadata(metadata))

	evt, err := event.NewEvent(
		eventType, aggregateID, aggregateType, event.Version(version),
		msg.Payload, opts...,
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "watermill.create_event_failed", "create event")
	}

	if metaErr != nil {
		return evt, event.WrapCorruption(metaErr, "watermill.corrupt_metadata",
			"event created with corrupt metadata fields")
	}

	return evt, nil
}

func parseSchemaVersion(md message.Metadata, topic string) (int, error) {
	svStr := md.Get(metaSchemaVersion)
	if svStr == "" {
		return 1, nil
	}

	version, err := parseInt(svStr, metaSchemaVersion)
	if err != nil {
		return 0, event.WrapRejection(err,
			"watermill.parse_schema_version_failed",
			fmt.Sprintf("topic %s: parse %s", topic, metaSchemaVersion))
	}

	return version, nil
}

func parseOptionalFields(md message.Metadata) ([]event.Option, error) {
	var opts []event.Option

	if eventIDStr := md.Get(metaEventID); eventIDStr != "" {
		eventID, err := id.ParseEventID(eventIDStr)
		if err != nil {
			return nil, event.WrapRejection(
				err,
				"watermill.parse_event_id_failed",
				"parse event_id",
			)
		}
		opts = append(opts, event.WithEventID(eventID))
	}

	if occurredAtStr := md.Get(metaOccurredAt); occurredAtStr != "" {
		occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)
		if err != nil {
			return nil, event.WrapRejection(
				err,
				"watermill.parse_occurred_at_failed",
				"parse occurred_at",
			)
		}
		opts = append(opts, event.WithOccurredAt(occurredAt))
	}

	return opts, nil
}

func buildMetadata(md message.Metadata) (event.Metadata, error) {
	m := event.NewMetadata()
	var errs []error

	parseIDField(
		md,
		metaCorrelationID,
		id.ParseCorrelationID,
		func(v id.CorrelationID) { m.CorrelationID = v },
		&errs,
	)
	parseIDField(
		md,
		metaCausationID,
		id.ParseCausationID,
		func(v id.CausationID) { m.CausationID = v },
		&errs,
	)
	parseIDField(md, metaUserID, id.ParseUserID, func(v id.UserID) { m.UserID = v }, &errs)
	parseIDField(
		md,
		metaRequestID,
		id.ParseRequestID,
		func(v id.RequestID) { m.RequestID = v },
		&errs,
	)

	if v := md.Get(metaSource); v != "" {
		m.Source = event.Source(v)
	}
	if v := md.Get(metaIPAddress); v != "" {
		m.IPAddress = event.IPAddress(v)
	}
	if v := md.Get(metaUserAgent); v != "" {
		m.UserAgent = event.UserAgent(v)
	}

	for k, v := range md {
		if after, ok := strings.CutPrefix(k, metaCustomPrefix); ok {
			event.EnsureCustom(&m)
			m.Custom[event.MetadataKey(after)] = v
		}
	}

	return m, event.Compose(errs...)
}

func parseIDField[T any](
	md message.Metadata,
	key string,
	parse func(string) (T, error),
	set func(T),
	errs *[]error,
) {
	v := md.Get(key)
	if v == "" {
		return
	}

	parsed, err := parse(v)
	if err != nil {
		*errs = append(*errs, event.WrapRejection(err, "watermill.parse_id_field_failed", key))

		return
	}

	set(parsed)
}

func parseInt(s, field string) (int, error) {
	if s == "" {
		return 0, event.NewRejection("watermill.missing_metadata", "missing "+field+" metadata")
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, event.WrapRejection(err, "watermill.parse_failed", "parse "+field)
	}

	return v, nil
}
