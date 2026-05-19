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

func AddService(tb testing.TB, r *catalog.Registry, id, name, version string) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      catalog.ServiceID(id),
		Name:    name,
		Version: version,
	})

	return r
}

func AddDomain(
	tb testing.TB,
	r *catalog.Registry,
	id, name, version, summary string,
	services []string,
) {
	tb.Helper()

	sids := make([]catalog.ServiceID, len(services))
	for i, s := range services {
		sids[i] = catalog.ServiceID(s)
	}

	r.AddDomain(catalog.Domain{
		ID:       catalog.DomainID(id),
		Name:     name,
		Version:  version,
		Summary:  summary,
		Services: sids,
	})
}

func AddServiceWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      catalog.ServiceID(id),
		Name:    name,
		Version: version,
		Summary: summary,
	})

	return r
}

func addMessage(
	tb testing.TB,
	r *catalog.Registry,
	svcID catalog.ServiceID,
	msg catalog.Message,
	fn func(catalog.ServiceID, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	fn(svcID, msg)

	return r
}

func AddMessage(
	tb testing.TB,
	r *catalog.Registry,
	svcID string,
	msg catalog.Message,
) *catalog.Registry {
	tb.Helper()

	sid := catalog.ServiceID(svcID)

	switch msg.Kind {
	case catalog.CommandMessage:
		return addMessage(tb, r, sid, msg, r.AddCommand)
	case catalog.EventMessage:
		return addMessage(tb, r, sid, msg, r.AddEvent)
	case catalog.QueryMessage:
		return addMessage(tb, r, sid, msg, r.AddQuery)
	default:
		tb.Fatalf("unknown message kind: %v", msg.Kind)

		return nil
	}
}

func AddMessageSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
	kind catalog.MessageKind,
	addFn func(catalog.ServiceID, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    kind,
		ID:      catalog.MessageID(id),
		Name:    name,
		Version: version,
		Summary: summary,
	}

	addFn(catalog.ServiceID(svcID), msg)

	return r
}

func AddEventSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        catalog.MessageID(id),
		Name:      name,
		Version:   version,
		Direction: direction,
	}

	r.AddEvent(catalog.ServiceID(svcID), msg)

	return r
}

func AddCommandWithSchema(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	schema *catalog.Schema,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      catalog.MessageID(id),
		Name:    name,
		Version: version,
		Schema:  schema,
	}
	r.AddCommand(catalog.ServiceID(svcID), msg)

	return r
}

func AddEventWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        catalog.MessageID(id),
		Name:      name,
		Version:   version,
		Summary:   summary,
		Direction: direction,
	}
	r.AddEvent(catalog.ServiceID(svcID), msg)

	return r
}

func AddCommandWithExamples(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	examples ...json.RawMessage,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       catalog.MessageID(id),
		Name:     name,
		Version:  version,
		Examples: examples,
	}
	r.AddCommand(catalog.ServiceID(svcID), msg)

	return r
}

func AddQuerySimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      catalog.MessageID(id),
		Name:    name,
		Version: version,
		Summary: summary,
	}

	r.AddQuery(catalog.ServiceID(svcID), msg)

	return r
}
