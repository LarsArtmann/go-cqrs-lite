package cattest

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func AddMessageSimple(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
	kind catalog.MessageKind,
	addFn func(catalog.ServiceID, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    kind,
		ID:      messageID,
		Name:    catalog.Name(name),
		Version: catalog.Version(version),
		Summary: catalog.Summary(summary),
	}

	addFn(serviceID, msg)

	return r
}

func AddEventSimple(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        messageID,
		Name:      catalog.Name(name),
		Version:   catalog.Version(version),
		Direction: direction,
	}

	r.AddEvent(serviceID, msg)

	return r
}

func AddEvent(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	return AddEventSimple(
		tb,
		r,
		serviceID,
		messageID,
		name,
		version,
		direction,
	)
}

func AddCommandWithSchema(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version string,
	schema *catalog.Schema,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      messageID,
		Name:    catalog.Name(name),
		Version: catalog.Version(version),
		Schema:  schema,
	}
	r.AddCommand(serviceID, msg)

	return r
}

func AddEventWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        messageID,
		Name:      catalog.Name(name),
		Version:   catalog.Version(version),
		Summary:   catalog.Summary(summary),
		Direction: direction,
	}
	r.AddEvent(serviceID, msg)

	return r
}

func AddCommandWithExamples(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version string,
	examples ...jsontext.Value,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       messageID,
		Name:     catalog.Name(name),
		Version:  catalog.Version(version),
		Examples: examples,
	}
	r.AddCommand(serviceID, msg)

	return r
}

func AddCommandWithExample(
	tb testing.TB,
	r *catalog.Registry,
	messageID catalog.MessageID,
	name, version, payload string,
) *catalog.Registry {
	tb.Helper()

	return AddCommandWithExamples(
		tb,
		r,
		catalog.ServiceID("svc"),
		messageID,
		name,
		version,
		jsontext.Value(payload),
	)
}

func AddCreateOrderCommand(
	tb testing.TB,
	r *catalog.Registry,
	name string,
) {
	tb.Helper()

	r.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: catalog.Name(name),
		Version: "1.0.0", Summary: "Create a new order",
	})
}
