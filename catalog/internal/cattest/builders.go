package cattest

import (
	"encoding/json/jsontext"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
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
		Name:    catalog.Name(name),
		Version: catalog.Version(version),
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
		Name:     catalog.Name(name),
		Version:  catalog.Version(version),
		Summary:  catalog.Summary(summary),
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

// NewTestCreateOrderFlow returns a flow with a single CreateOrder step.
func NewTestCreateOrderFlow(title string) catalog.Flow {
	return catalog.Flow{
		ID: "create-order", Name: "Create Order", Version: "1.0.0",
		Summary: "",
		Steps: []catalog.FlowStep{
			{
				ID:    "1",
				Title: catalog.Title(title),
				Message: &catalog.FlowStepRef{
					ID:      testCreateOrderMsgID,
					Version: "",
				},
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

func StringSchema(props ...string) (*catalog.Schema, error) {
	if len(props)%2 != 0 {
		return nil, fmt.Errorf(
			"cattest.StringSchema: props must be key-value pairs, got %d",
			len(props),
		)
	}

	//nolint:mnd // key-value pairs = half the input length
	m := make(map[string]catalog.Property, len(props)/2)
	for i := 0; i < len(props); i += 2 {
		m[props[i]] = catalog.Property{Type: catalog.TypeString}
	}

	return &catalog.Schema{Type: catalog.TypeObject, Properties: m}, nil
}

func addServiceWithMessage(
	tb testing.TB,
	registry *catalog.Registry,
	svcID catalog.ServiceID,
	msgID catalog.MessageID,
	msgName, msgVersion, msgSummary string,
	msgKind catalog.MessageKind,
	register func(catalog.ServiceID, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	AddService(tb, registry, svcID, string(svcID), msgVersion)

	return AddMessageSimple(
		tb,
		registry,
		svcID,
		msgID,
		msgName,
		msgVersion,
		msgSummary,
		msgKind,
		register,
	)
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
	svc catalog.ServiceID,
	msg catalog.MessageID,
	msgName, msgVersion, msgSummary string,
) *catalog.Registry {
	tb.Helper()

	return addServiceWithMessage(
		tb,
		r,
		svc,
		msg,
		msgName,
		msgVersion,
		msgSummary,
		catalog.CommandMessage,
		r.AddCommand,
	)
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

func AddServiceWithSummary(
	tb testing.TB,
	r *catalog.Registry,
	id catalog.ServiceID,
	name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	r.AddService(catalog.Service{
		ID:      id,
		Name:    catalog.Name(name),
		Version: catalog.Version(version),
		Summary: catalog.Summary(summary),
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
		Name:          catalog.Name(name),
		Version:       catalog.Version(version),
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

	protos := make([]catalog.Protocol, len(protocols))
	for i, p := range protocols {
		protos[i] = catalog.Protocol(p)
	}

	r.AddChannel(catalog.Channel{
		ID:        id,
		Name:      catalog.Name(name),
		Version:   catalog.Version(version),
		Summary:   catalog.Summary(summary),
		Protocols: protos,
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
		Kind: catalog.CommandMessage, ID: "CreateOrder", Name: catalog.Name(name),
		Version: "1.0.0", Summary: "Create a new order",
	})
}
