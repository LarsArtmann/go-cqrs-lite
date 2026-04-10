package cattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
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

// Add adds a command, event, or query message to a service using the provided add function.
func Add(
	tb testing.TB,
	r *catalog.Registry,
	svcID string,
	msg catalog.Message,
	add func(string, catalog.Message),
) *catalog.Registry {
	tb.Helper()

	add(svcID, msg)

	return r
}

// AddCommand adds a command message to a service.
func AddCommand(tb testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	return Add(tb, r, svcID, msg, r.AddCommand)
}

// AddEvent adds an event message to a service.
func AddEvent(tb testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	return Add(tb, r, svcID, msg, r.AddEvent)
}

// AddQuery adds a query message to a service.
func AddQuery(tb testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	return Add(tb, r, svcID, msg, r.AddQuery)
}

// AddCommandSimple creates and adds a command message with minimal parameters.
func AddCommandSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	return AddCommand(tb, r, svcID, catalog.Message{
		Kind:    catalog.CommandMessage,
		ID:      id,
		Name:    name,
		Version: version,
		Summary: summary,
	})
}

// AddEventSimple creates and adds an event message with minimal parameters.
func AddEventSimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version string,
	direction catalog.Direction,
) *catalog.Registry {
	tb.Helper()

	return AddEvent(tb, r, svcID, catalog.Message{
		Kind:      catalog.EventMessage,
		ID:        id,
		Name:      name,
		Version:   version,
		Direction: direction,
	})
}

// AddQuerySimple creates and adds a query message with minimal parameters.
func AddQuerySimple(
	tb testing.TB,
	r *catalog.Registry,
	svcID, id, name, version, summary string,
) *catalog.Registry {
	tb.Helper()

	return AddQuery(tb, r, svcID, catalog.Message{
		Kind:    catalog.QueryMessage,
		ID:      id,
		Name:    name,
		Version: version,
		Summary: summary,
	})
}

// Build builds the catalog from the registry.
func Build(tb testing.TB, r *catalog.Registry) *catalog.Catalog {
	tb.Helper()

	return r.Build()
}

// MustExport exports the catalog using the given exporter and fails on error.
func MustExport(
	tb testing.TB,
	exp interface{ Export(*catalog.Catalog) error },
	cat *catalog.Catalog,
) {
	tb.Helper()

	err := exp.Export(cat)
	if err != nil {
		tb.Fatalf("export catalog: %v", err)
	}
}

// FileContains asserts that the file at path contains the given substring.
func FileContains(tb testing.TB, path, substring string) {
	tb.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read file %s: %v", path, err)
	}

	if !strings.Contains(string(data), substring) {
		tb.Errorf("file %s does not contain %q, content:\n%s", path, substring, string(data))
	}
}

// MustReadFile reads a file and returns its contents, failing on error.
func MustReadFile(tb testing.TB, path string) string {
	tb.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read file %s: %v", path, err)
	}

	return string(data)
}

// Join joins path elements for use in assertions.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}
