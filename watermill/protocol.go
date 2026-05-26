package watermill

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

	if m := evt.Metadata(); m != nil {
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
		return nil, fmt.Errorf("parse aggregate_id: %w", err)
	}

	aggregateType := event.AggregateType(md.Get(metaAggregateType))
	if aggregateType == "" {
		return nil, fmt.Errorf("missing %s metadata", metaAggregateType)
	}

	version, err := parseInt(md.Get(metaVersion), metaVersion)
	if err != nil {
		return nil, err
	}

	schemaVersion := 1
	if svStr := md.Get(metaSchemaVersion); svStr != "" {
		sv, err := parseInt(svStr, metaSchemaVersion)
		if err != nil {
			return nil, err
		}
		schemaVersion = sv
	}

	opts := []event.Option{event.WithSchemaVersion(event.SchemaVersion(schemaVersion))}

	if eventIDStr := md.Get(metaEventID); eventIDStr != "" {
		eventID, err := id.ParseEventID(eventIDStr)
		if err != nil {
			return nil, fmt.Errorf("parse event_id: %w", err)
		}
		opts = append(opts, event.WithEventID(eventID))
	}

	if occurredAtStr := md.Get(metaOccurredAt); occurredAtStr != "" {
		occurredAt, err := time.Parse(time.RFC3339Nano, occurredAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse occurred_at: %w", err)
		}
		opts = append(opts, event.WithOccurredAt(occurredAt))
	}

	metadata := buildMetadata(md)
	if metadata != nil {
		opts = append(opts, event.WithMetadata(metadata))
	}

	evt, err := event.NewEvent(eventType, aggregateID, aggregateType, event.Version(version), msg.Payload, opts...)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	return evt, nil
}

func buildMetadata(md message.Metadata) *event.Metadata {
	var m *event.Metadata

	setIfPresent := func(key string, setter func(*event.Metadata)) {
		if v := md.Get(key); v != "" {
			if m == nil {
				m = event.NewMetadata()
			}
			setter(m)
		}
	}

	setIfPresent(metaCorrelationID, func(m *event.Metadata) {
		m.CorrelationID = id.MustParseCorrelationID(md.Get(metaCorrelationID))
	})
	setIfPresent(metaCausationID, func(m *event.Metadata) {
		m.CausationID = id.MustParseCausationID(md.Get(metaCausationID))
	})
	setIfPresent(metaUserID, func(m *event.Metadata) {
		m.UserID = id.MustParseUserID(md.Get(metaUserID))
	})
	setIfPresent(metaRequestID, func(m *event.Metadata) {
		m.RequestID = id.MustParseRequestID(md.Get(metaRequestID))
	})
	setIfPresent(metaSource, func(m *event.Metadata) { m.Source = event.Source(md.Get(metaSource)) })
	setIfPresent(metaIPAddress, func(m *event.Metadata) { m.IPAddress = event.IPAddress(md.Get(metaIPAddress)) })
	setIfPresent(metaUserAgent, func(m *event.Metadata) { m.UserAgent = event.UserAgent(md.Get(metaUserAgent)) })

	for k, v := range md {
		if strings.HasPrefix(k, metaCustomPrefix) {
			if m == nil {
				m = event.NewMetadata()
			}
			customKey := event.MetadataKey(strings.TrimPrefix(k, metaCustomPrefix))
			m.Custom[customKey] = v
		}
	}

	return m
}

func parseInt(s, field string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("missing %s metadata", field)
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", field, err)
	}

	return v, nil
}
