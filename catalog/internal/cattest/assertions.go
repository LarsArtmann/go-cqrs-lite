package cattest

import (
	"os"
	"strings"
	"testing"
)

func MustReadFile(tb testing.TB, path string) string {
	tb.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read file %s: %v", path, err)
	}

	return string(data)
}

func AssertFileExists(tb testing.TB, path string) {
	tb.Helper()

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		tb.Errorf("file does not exist: %s", path)
	}
}

func AssertSliceLen[T any](tb testing.TB, name string, slice []T, expected int) {
	tb.Helper()

	if len(slice) != expected {
		tb.Fatalf("%s = %d, want %d", name, len(slice), expected)
	}
}

func AssertMapLen[K comparable, V any](tb testing.TB, name string, m map[K]V, expected int) {
	tb.Helper()

	if len(m) != expected {
		tb.Fatalf("%s = %d, want %d", name, len(m), expected)
	}
}

func AssertContentContains(tb testing.TB, content, desc string, substrs ...string) {
	tb.Helper()

	for _, sub := range substrs {
		if !strings.Contains(content, sub) {
			tb.Errorf("%s missing %q", desc, sub)
		}
	}
}

func ReadFileAndAssert(tb testing.TB, path, desc string, substrs ...string) string {
	tb.Helper()

	content := MustReadFile(tb, path)
	AssertContentContains(tb, content, desc, substrs...)

	return content
}
