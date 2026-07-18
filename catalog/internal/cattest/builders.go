package cattest

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
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
