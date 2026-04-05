package cattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

// NewRegistry creates a new test registry with the given title and version.
func NewRegistry(t testing.TB, title, version string) *catalog.Registry {
	t.Helper()

	return catalog.NewRegistry(title, version)
}

// AddService adds a service to the registry and returns the registry for chaining.
func AddService(t testing.TB, r *catalog.Registry, id, name, version string) *catalog.Registry {
	t.Helper()

	r.AddService(catalog.Service{
		ID:      id,
		Name:    name,
		Version: version,
	})

	return r
}

// AddCommand adds a command message to a service.
func AddCommand(t testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	t.Helper()

	r.AddCommand(svcID, msg)

	return r
}

// AddEvent adds an event message to a service.
func AddEvent(t testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	t.Helper()

	r.AddEvent(svcID, msg)

	return r
}

// AddQuery adds a query message to a service.
func AddQuery(t testing.TB, r *catalog.Registry, svcID string, msg catalog.Message) *catalog.Registry {
	t.Helper()

	r.AddQuery(svcID, msg)

	return r
}

// Build builds the catalog from the registry.
func Build(t testing.TB, r *catalog.Registry) *catalog.Catalog {
	t.Helper()

	return r.Build()
}

// MustExport exports the catalog using the given exporter and fails on error.
func MustExport(t testing.TB, exp interface{ Export(*catalog.Catalog) error }, cat *catalog.Catalog) {
	t.Helper()

	if err := exp.Export(cat); err != nil {
		t.Fatalf("export catalog: %v", err)
	}
}

// FileContains asserts that the file at path contains the given substring.
func FileContains(t testing.TB, path, substring string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	if !strings.Contains(string(data), substring) {
		t.Errorf("file %s does not contain %q, content:\n%s", path, substring, string(data))
	}
}

// MustReadFile reads a file and returns its contents, failing on error.
func MustReadFile(t testing.TB, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	return string(data)
}

// Join joins path elements for use in assertions.
func Join(elem ...string) string {
	return filepath.Join(elem...)
}
