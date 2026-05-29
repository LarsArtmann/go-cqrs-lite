package cattest

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func NewRegistry(tb testing.TB, title, version string) *catalog.Registry {
	tb.Helper()

	return catalog.NewRegistry(title, version)
}

func AddService(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.ServiceID,
	name, version string,
) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      id,
		Name:    name,
		Version: version,
	})

	return r
}

func AddDomain(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.DomainID,
	name, version, summary string,
	services []catalog.ServiceID,
) {
	tb.Helper()

	r.AddDomain(catalog.Domain{
		ID:       id,
		Name:     name,
		Version:  version,
		Summary:  summary,
		Services: services,
	})
}

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
		Name:    name,
		Version: version,
		Summary: summary,
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
		Name:      name,
		Version:   version,
		Direction: direction,
	}

	r.AddEvent(serviceID, msg)

	return r
}

func AddEvent(
	tb testing.TB,
	r *catalog.Registry,
	serviceID, messageID, name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	return AddEventSimple(tb, r, catalog.ServiceID(serviceID), catalog.MessageID(messageID), name, version, direction)
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
		Name:    name,
		Version: version,
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
		Name:      name,
		Version:   version,
		Summary:   summary,
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
	examples ...json.RawMessage,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       messageID,
		Name:     name,
		Version:  version,
		Examples: examples,
	}
	r.AddCommand(serviceID, msg)

	return r
}

func AddQuerySimple(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      messageID,
		Name:    name,
		Version: version,
		Summary: summary,
	}

	r.AddQuery(serviceID, msg)

	return r
}
