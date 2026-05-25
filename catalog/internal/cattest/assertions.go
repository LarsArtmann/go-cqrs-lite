package cattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

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
func AssertLenEqual[T any](tb testing.TB, name string, actual, expected int, _ []T) {
	tb.Helper()

	if actual != expected {
		tb.Fatalf("%s = %d, want %d", name, actual, expected)
	}
}

// AssertFileExists fails if the file doesn't exist.
func AssertFileExists(tb testing.TB, path string) {
	tb.Helper()

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
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

		return
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

// ReadFileAndAssert reads a file and asserts it contains all substrings.
func ReadFileAndAssert(tb testing.TB, path, desc string, substrs ...string) string {
	tb.Helper()

	content := MustReadFile(tb, path)
	AssertContentContains(tb, content, desc, substrs...)

	return content
}

// AssertServiceFrontmatter asserts a service frontmatter contains expected fields.
func AssertServiceFrontmatter(tb testing.TB, content string, serviceID catalog.ServiceID, svcName string) {
	tb.Helper()

	AssertContentContains(
		tb, content, "service frontmatter",
		"id: "+string(serviceID),
		"name: "+svcName,
		"# "+svcName,
	)
}

// AssertMessageFrontmatter asserts a message frontmatter contains expected fields.
func AssertMessageFrontmatter(tb testing.TB, content string, messageID catalog.MessageID, checkHeading bool) {
	tb.Helper()

	AssertContentContains(
		tb, content, "message frontmatter",
		"id: "+string(messageID),
	)

	if checkHeading {
		AssertContentContains(
			tb, content, "message frontmatter heading",
			"# "+string(messageID),
		)
	}
}
