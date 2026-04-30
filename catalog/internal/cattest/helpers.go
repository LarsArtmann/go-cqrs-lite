package cattest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// NewRegistry creates a new test registry with the given title and version.
func NewRegistry(tb testing.TB, title, version string) *catalog.Registry {
	tb.Helper()

	return catalog.NewRegistry(title, version)
}

// AddService adds a service to the registry and returns the registry for chaining.
func AddService(tb testing.TB, r *catalog.Registry, id, name, version string) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      id,
		Name:    name,
		Version: version,
	})

	return r
}

// AddDomain adds a domain to the registry.
func AddDomain(
	tb testing.TB,
	r *catalog.Registry,
	id, name, version, summary string,
	services []string,
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

// addMessage is a generic helper that adds a message to a service.
func addMessage(
	tb testing.TB,
	r *catalog.Registry,
	svcID string,
	msg catalog.Message,
	fn func(string, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	fn(svcID, msg)

	return r
}

// AddMessage adds a message to a service by kind.
func AddMessage(
	tb testing.TB,
	r *catalog.Registry,
	svcID string,
	msg catalog.Message,
) *catalog.Registry {
	tb.Helper()

	switch msg.Kind {
	case catalog.CommandMessage:
		return addMessage(tb, r, svcID, msg, r.AddCommand)
	case catalog.EventMessage:
		return addMessage(tb, r, svcID, msg, r.AddEvent)
	case catalog.QueryMessage:
		return addMessage(tb, r, svcID, msg, r.AddQuery)
	default:
		tb.Fatalf("unknown message kind: %v", msg.Kind)

		return nil
	}
}

// AddMessageSimple creates a message with common fields and adds it via the provided function.
func AddMessageSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
	kind catalog.MessageKind,
	addFn func(string, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    kind,
		ID:      id,
		Name:    name,
		Version: version,
		Summary: summary,
	}

	addFn(svcID, msg)

	return r
}

// AddEventSimple creates and adds an event message with minimal parameters.
func AddEventSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        id,
		Name:      name,
		Version:   version,
		Direction: direction,
	}

	r.AddEvent(svcID, msg)

	return r
}

// Build builds the catalog from the registry.
func Build(tb testing.TB, r *catalog.Registry) *catalog.Catalog {
	tb.Helper()

	return r.Build()
}

// MustExport exports the catalog using the given exporter and fails on error.
func MustExport(
	tb testing.TB,
	exp interface {
		Export(cat *catalog.Catalog) error
	},
	cat *catalog.Catalog,
) {
	tb.Helper()

	err := exp.Export(cat)
	if err != nil {
		tb.Fatalf("export catalog: %v", err)
	}
}

// AddServiceWithSummary adds a service with summary.
func AddServiceWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      id,
		Name:    name,
		Version: version,
		Summary: summary,
	})

	return r
}

// AddCommandWithSchema creates and adds a command message with a schema.
func AddCommandWithSchema(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	schema *catalog.Schema,
) *catalog.Registry {
	tb.Helper()

	addCommandWithSchema(r, svcID, id, name, version, schema)

	return r
}

// AddEventWithSummary creates and adds an event message with summary and direction.
func AddEventWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	addEventWithSummary(r, svcID, id, name, version, summary, direction)

	return r
}

// AddCommandWithExamples creates and adds a command message with examples.
func AddCommandWithExamples(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	examples ...json.RawMessage,
) *catalog.Registry {
	tb.Helper()

	addCommandWithExamples(r, svcID, id, name, version, examples...)

	return r
}

func addCommandWithSchema(
	r *catalog.Registry,
	svcID, id, name, version string,
	schema *catalog.Schema,
) {
	msg := catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      id,
		Name:    name,
		Version: version,
		Schema:  schema,
	}
	r.AddCommand(svcID, msg)
}

func addEventWithSummary(
	r *catalog.Registry,
	svcID, id, name, version, summary string,
	direction catalog.Direction,
) {
	msg := catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        id,
		Name:      name,
		Version:   version,
		Summary:   summary,
		Direction: direction,
	}
	r.AddEvent(svcID, msg)
}

func addCommandWithExamples(
	r *catalog.Registry,
	svcID, id, name, version string,
	examples ...json.RawMessage,
) {
	msg := catalog.Message{
		Kind:     catalog.CommandMessage,
		ID:       id,
		Name:     name,
		Version:  version,
		Examples: examples,
	}
	r.AddCommand(svcID, msg)
}

// AddQuerySimple creates and adds a query message with minimal parameters.
func AddQuerySimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	msg := catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      id,
		Name:    name,
		Version: version,
		Summary: summary,
	}

	r.AddQuery(svcID, msg)

	return r
}

// NewCatalogCore creates an event catalog core with defaults for testing.
func NewCatalogCore(
	tb testing.TB,
	eventType string,
	meta event.CatalogMeta,
) (*event.CatalogCore, error) {
	tb.Helper()

	core, err := event.NewCatalogCore(
		event.Type(eventType),
		id.NewAggregateID(),
		"Order",
		1,
		nil,
		meta,
	)
	if err != nil {
		return nil, fmt.Errorf("new event catalog core: %w", err)
	}

	return core, nil
}

// BuildTestCatalog creates a standard test catalog with order service for golden tests.
func BuildTestCatalog() *catalog.Catalog {
	reg := catalog.NewRegistry("E-Commerce", "1.0.0")
	reg.AddService(catalog.Service{
		ID: "order-svc", Name: "Order Service", Version: "1.0.0", Summary: "Manages orders",
	})
	reg.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: "Create Order", Version: "1.0.0",
		Summary: "Create a new order",
		Schema: &catalog.Schema{Type: "object", Properties: map[string]catalog.Property{
			"orderId": {Type: "string"},
		}},
	})
	reg.AddEvent("order-svc", catalog.Message{
		Kind: catalog.EventMessage, ID: "OrderCreated", Name: "Order Created", Version: "1.0.0",
		Summary: "Order was created", Direction: catalog.Sends,
	})
	reg.AddQuery("order-svc", catalog.Message{
		Kind: catalog.QueryMessage, ID: "GetOrder", Name: "Get Order", Version: "1.0.0",
		Summary: "Get order by ID",
	})

	return reg.Build()
}
