package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// Validate reports the cross-resource catalog rules that
// [catalog.Catalog.Validate] cannot express as field-level uniqueness checks.
//
// The composed checks are:
//
//   - coeffect completeness, delegated verbatim to
//     [catalog.Catalog.ValidateCoeffects] so the two can never disagree;
//   - direction legality: commands and queries are received by a service,
//     events are sent or received, and no message leaves direction empty;
//   - channel references resolve, in both directions;
//   - data store references resolve.
//
// It is ADDITIVE by contract. Every check [catalog.Catalog.Validate] performs
// is untouched; run both. Nothing here re-implements a rule another function
// already owns.
func Validate(c *catalog.Catalog) []catalog.Violation {
	if c == nil {
		return nil
	}

	violations := c.ValidateCoeffects()
	violations = append(violations, validateDirections(c)...)
	violations = append(violations, validateChannelRefs(c)...)
	violations = append(violations, validateDataStoreRefs(c)...)

	return violations
}

func validateDirections(c *catalog.Catalog) []catalog.Violation {
	var violations []catalog.Violation

	for _, svc := range c.Services {
		for _, cmd := range svc.Commands {
			violations = append(
				violations,
				wantDirection(svc.ID, "command", cmd, catalog.Receives)...,
			)
		}

		for _, query := range svc.Queries {
			violations = append(
				violations,
				wantDirection(svc.ID, "query", query, catalog.Receives)...,
			)
		}

		for _, evt := range svc.Events {
			violations = append(violations, wantEventDirection(svc.ID, evt)...)
		}
	}

	return violations
}

func wantDirection(
	svcID catalog.ServiceID,
	kind string,
	msg catalog.Message,
	want catalog.Direction,
) []catalog.Violation {
	if msg.Direction == want {
		return nil
	}

	return []catalog.Violation{{
		Path: messagePath(svcID, kind, msg) + ".direction",
		Message: fmt.Sprintf(
			"a %s is %q by definition, got %q",
			kind,
			want,
			msg.Direction,
		),
	}}
}

func wantEventDirection(svcID catalog.ServiceID, evt catalog.Message) []catalog.Violation {
	if evt.Direction == catalog.Sends || evt.Direction == catalog.Receives {
		return nil
	}

	return []catalog.Violation{
		{
			Path: messagePath(svcID, "event", evt) + ".direction",
			Message: fmt.Sprintf(
				"an event must be %q or %q, got %q",
				catalog.Sends,
				catalog.Receives,
				evt.Direction,
			),
		},
	}
}

func validateChannelRefs(c *catalog.Catalog) []catalog.Violation {
	messages := indexMessages(c)
	channels := make(map[catalog.ChannelID]bool, len(c.Channels))

	for _, ch := range c.Channels {
		channels[ch.ID] = true
	}

	var violations []catalog.Violation

	for _, ch := range c.Channels {
		for _, msgID := range ch.Messages {
			if _, ok := messages[msgID]; ok {
				continue
			}

			violations = append(violations, catalog.Violation{
				Path:    fmt.Sprintf("channels[%s].messages", ch.ID),
				Message: fmt.Sprintf("channel references undeclared message %q", msgID),
			})
		}
	}

	for _, svc := range c.Services {
		for _, msg := range serviceMessages(svc) {
			violations = append(violations, danglingChannels(svc, msg, channels)...)
		}
	}

	return violations
}

func danglingChannels(
	svc catalog.Service,
	msg catalog.Message,
	channels map[catalog.ChannelID]bool,
) []catalog.Violation {
	var violations []catalog.Violation

	for _, chID := range msg.Channels {
		if channels[chID] {
			continue
		}

		violations = append(violations, catalog.Violation{
			Path:    messagePath(svc.ID, msg.Kind.String(), msg) + ".channels",
			Message: fmt.Sprintf("message references undeclared channel %q", chID),
		})
	}

	return violations
}

func validateDataStoreRefs(c *catalog.Catalog) []catalog.Violation {
	stores := make(map[catalog.DataStoreID]bool, len(c.DataStores))
	for _, store := range c.DataStores {
		stores[store.ID] = true
	}

	violations := make([]catalog.Violation, 0, len(c.Services))

	for _, svc := range c.Services {
		violations = append(violations, danglingStores(svc, stores)...)
	}

	return violations
}

func danglingStores(
	svc catalog.Service,
	stores map[catalog.DataStoreID]bool,
) []catalog.Violation {
	var violations []catalog.Violation

	check := func(field string, ids []catalog.DataStoreID) {
		for _, id := range ids {
			if stores[id] {
				continue
			}

			violations = append(violations, catalog.Violation{
				Path:    fmt.Sprintf("services[%s].%s", svc.ID, field),
				Message: fmt.Sprintf("service references undeclared data store %q", id),
			})
		}
	}

	check("writesTo", svc.WritesTo)
	check("readsFrom", svc.ReadsFrom)

	return violations
}

func messagePath(svcID catalog.ServiceID, kind string, msg catalog.Message) string {
	return fmt.Sprintf("services[%s].%s[%s]", svcID, kind, catalog.Key(msg))
}
