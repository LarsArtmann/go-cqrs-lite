package cattest

import (
	"encoding/json"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
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

	return AddEventSimple(
		tb,
		r,
		catalog.ServiceID(serviceID),
		catalog.MessageID(messageID),
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

// CreateItemSchema returns a reusable object schema with a required "name" string property.
func NewTestCreateOrderFlow(title string) catalog.Flow {
	return catalog.Flow{
		ID: "create-order", Name: "Create Order", Version: "1.0.0",
		Summary: "",
		Steps: []catalog.FlowStep{
			{
				ID:        "1",
				Title:     title,
				Message:   &catalog.FlowStepRef{ID: "CreateOrder", Version: ""},
				Summary:   "",
				Service:   nil,
				Channel:   nil,
				Actor:     nil,
				External:  nil,
				Custom:    nil,
				NextStep:  nil,
				NextSteps: nil,
			},
		},
		Badges: nil,
	}
}

func CreateItemSchema() *catalog.Schema {
	return &catalog.Schema{
		Type: "object",
		Properties: map[string]catalog.Property{
			"name": {Type: "string", Description: "Item name"},
		},
		Required: []string{"name"},
	}
}

func StringSchema(props ...string) *catalog.Schema {
	if len(props)%2 != 0 {
		panic("StringSchema: props must be key-value pairs")
	}

	//nolint:mnd // key-value pairs = half the input length
	m := make(map[string]catalog.Property, len(props)/2)
	for i := 0; i < len(props); i += 2 {
		m[props[i]] = catalog.Property{Type: catalog.TypeString}
	}

	return &catalog.Schema{Type: catalog.TypeObject, Properties: m}
}

func addServiceWithMessage(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
	kind catalog.MessageKind,
	addFn func(catalog.ServiceID, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	AddService(tb, r, serviceID, string(serviceID), version)

	return AddMessageSimple(tb, r, serviceID, messageID, name, version, summary, kind, addFn)
}

func AddServiceWithQuery(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	return addServiceWithMessage(
		tb,
		r,
		serviceID,
		messageID,
		name,
		version,
		summary,
		catalog.QueryMessage,
		r.AddQuery,
	)
}

func AddServiceWithCommand(
	tb testing.TB,
	r *catalog.Registry,
	serviceID catalog.ServiceID,
	messageID catalog.MessageID,
	name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	return addServiceWithMessage(
		tb,
		r,
		serviceID,
		messageID,
		name,
		version,
		summary,
		catalog.CommandMessage,
		r.AddCommand,
	)
}

func AddCommandWithExample(
	tb testing.TB,
	r *catalog.Registry,
	messageID, name, version, payload string,
) *catalog.Registry {
	tb.Helper()

	return AddCommandWithExamples(
		tb,
		r,
		catalog.ServiceID("svc"),
		catalog.MessageID(messageID),
		name,
		version,
		json.RawMessage(payload),
	)
}

func AddServiceWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.ServiceID,
	name, version, summary string,
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

func AddDataStore(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.DataStoreID,
	name, version, containerType string,
) *catalog.Registry {
	tb.Helper()

	r.AddDataStore(catalog.DataStore{
		ID:            id,
		Name:          name,
		Version:       version,
		ContainerType: containerType,
	})

	return r
}

func AddChannel(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.ChannelID,
	name, version, summary string,
	protocols []string,
) *catalog.Registry {
	tb.Helper()

	r.AddChannel(catalog.Channel{
		ID:        id,
		Name:      name,
		Version:   version,
		Summary:   summary,
		Protocols: protocols,
	})

	return r
}

func AddCreateOrderCommand(
	tb testing.TB,
	r *catalog.Registry,
	name string,
) {
	tb.Helper()

	r.AddCommand("order-svc", catalog.Message{
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: name,
		Version: "1.0.0", Summary: "Create a new order",
	})
}
