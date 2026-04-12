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

// AssertLenEqual fails if the actual length doesn't match the expected length.
func AssertLenEqual[T any](tb testing.TB, name string, actual int, expected int, slice []T) {
	tb.Helper()

	if actual != expected {
		tb.Fatalf("%s = %d, want %d", name, actual, expected)
	}
}

// AssertFileExists fails if the file doesn't exist.
func AssertFileExists(tb testing.TB, path string) {
	tb.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		tb.Errorf("file does not exist: %s", path)
	}
}

// AssertFileContentContains fails if the file content doesn't contain the substring.
func AssertFileContentContains(tb testing.TB, path, substring string) {
	tb.Helper()

	content := MustReadFile(tb, path)
	if !strings.Contains(content, substring) {
		tb.Errorf("file %s does not contain %q, content:\n%s", path, substring, content)
	}
}

// AssertSliceLen fails if slice length doesn't match expected.
func AssertSliceLen[T any](tb testing.TB, name string, slice []T, expected int) {
	tb.Helper()

	if len(slice) != expected {
		tb.Fatalf("%s = %d, want %d", name, len(slice), expected)
	}
}

// AssertMapLen fails if map length doesn't match expected.
func AssertMapLen[K comparable, V any](tb testing.TB, name string, m map[K]V, expected int) {
	tb.Helper()

	if len(m) != expected {
		tb.Fatalf("%s = %d, want %d", name, len(m), expected)
	}
}

// AssertSchemaProperty fails if the schema doesn't contain the expected property.
func AssertSchemaProperty(tb testing.TB, schema *catalog.Schema, propName string) {
	tb.Helper()

	if schema == nil {
		tb.Fatal("schema should not be nil")
	}

	if _, ok := schema.Properties[propName]; !ok {
		tb.Errorf("schema missing %q property", propName)
	}
}

// AssertContentContains fails if the content doesn't contain all substrings.
func AssertContentContains(tb testing.TB, content, desc string, substrs ...string) {
	tb.Helper()

	for _, sub := range substrs {
		if !strings.Contains(content, sub) {
			tb.Errorf("%s missing %q", desc, sub)
		}
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

// ReadFileAndAssert reads a file and asserts it contains all substrings.
func ReadFileAndAssert(tb testing.TB, path, desc string, substrs ...string) string {
	tb.Helper()

	content := MustReadFile(tb, path)
	AssertContentContains(tb, content, desc, substrs...)

	return content
}

// AssertServiceFrontmatter asserts a service frontmatter contains expected fields.
func AssertServiceFrontmatter(tb testing.TB, content string, svcID, svcName string) {
	tb.Helper()

	AssertContentContains(tb, content, "service frontmatter",
		"id: "+svcID,
		"name: "+svcName,
		"# "+svcName,
	)
}

// AssertMessageFrontmatter asserts a message frontmatter contains expected fields.
func AssertMessageFrontmatter(tb testing.TB, content, msgID string, checkHeading bool) {
	tb.Helper()

	tb.Helper()

	AssertContentContains(tb, content, "message frontmatter",
		"id: "+msgID,
	)
	if checkHeading {
		AssertContentContains(tb, content, "message frontmatter heading",
			"# "+msgID,
		)
	}
}
