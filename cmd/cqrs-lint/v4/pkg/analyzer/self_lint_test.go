package analyzer

import "testing"

// TestIsExampleModulePath pins the example/consumer classification that keeps
// the V007/F030-class detectors running on the example apps (they share the
// library's module prefix but are semantically consumers).
func TestIsExampleModulePath(t *testing.T) {
	t.Parallel()

	examples := []string{
		"github.com/larsartmann/go-cqrs-lite/example/taskmanager",
		"github.com/larsartmann/go-cqrs-lite/example/getting-started",
		"github.com/larsartmann/go-cqrs-lite/example/metaengine-quickstart",
		"github.com/larsartmann/go-cqrs-lite/example/readme-quickstart",
	}

	for _, path := range examples {
		if !IsExampleModulePath(path) {
			t.Errorf("IsExampleModulePath(%q) = false, want true", path)
		}
	}

	consumers := []string{
		"example.com/taskmanager",
		"github.com/larsartmann/bank-sync",
		"github.com/larsartmann/go-cqrs-lite/examplex/taskmanager",
		"github.com/larsartmann/go-cqrs-lite/event/v4",
	}

	for _, path := range consumers {
		if IsExampleModulePath(path) {
			t.Errorf("IsExampleModulePath(%q) = true, want false", path)
		}
	}
}

// TestIsLibrarySelfLint_ExamplesAreConsumers pins the false-green fix: an
// AnalysisContext whose module is an example app must NOT classify as library
// self-lint, or the v5-removed-API detectors silently skip it.
func TestIsLibrarySelfLint_ExamplesAreConsumers(t *testing.T) {
	t.Parallel()

	exampleCtx := &AnalysisContext{
		ModulePath: "github.com/larsartmann/go-cqrs-lite/example/taskmanager",
	}
	if exampleCtx.IsLibrarySelfLint() {
		t.Error(
			"example module classified as self-lint — V007/F030 would silently skip it (the false-green class)",
		)
	}

	libraryCtx := &AnalysisContext{
		ModulePath: "github.com/larsartmann/go-cqrs-lite/metaengine/v4",
	}
	if !libraryCtx.IsLibrarySelfLint() {
		t.Error("library module classified as consumer — self-lint suppressions would stop working")
	}

	consumerCtx := &AnalysisContext{
		ModulePath: "example.com/standalone-consumer",
	}
	if consumerCtx.IsLibrarySelfLint() {
		t.Error("unrelated consumer module classified as self-lint")
	}
}
